package handler

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/google/go-github/github"
	"golang.org/x/crypto/ssh"
)

// RepoClient for testing purposes
//go:generate mockgen -destination=mocks/mock_repo_client.go -package=mocks github.com/telia-oss/concourse-github-lambda RepoClient
type RepoClient interface {
	ListKeys(ctx context.Context, owner string, repo string, opt *github.ListOptions) ([]*github.Key, *github.Response, error)
	CreateKey(ctx context.Context, owner string, repo string, key *github.Key) (*github.Key, *github.Response, error)
	DeleteKey(ctx context.Context, owner string, repo string, id int) (*github.Response, error)
}

// AppsClient for testing purposes
//go:generate mockgen -destination=mocks/mock_apps_client.go -package=mocks github.com/telia-oss/concourse-github-lambda AppsClient
type AppsClient interface {
	ListRepos(ctx context.Context, opt *github.ListOptions) ([]*github.Repository, *github.Response, error)
	CreateInstallationToken(ctx context.Context, id int64) (*github.InstallationToken, *github.Response, error)
}

// SecretsClient for testing purposes.
//go:generate mockgen -destination=mocks/mock_secrets_client.go -package=mocks github.com/telia-oss/concourse-github-lambda SecretsClient
type SecretsClient secretsmanageriface.SecretsManagerAPI

// EC2Client for testing purposes.
//go:generate mockgen -destination=mocks/mock_ec2_client.go -package=mocks github.com/telia-oss/concourse-github-lambda EC2Client
type EC2Client ec2iface.EC2API

// NewTestManager for testing purposes.
func NewTestManager(s SecretsClient, e EC2Client, tokenService, keyService *GithubApp) *Manager {
	return &Manager{secretsClient: s, ec2Client: e, tokenService: tokenService, keyService: keyService}
}

// Manager handles API calls to AWS.
type Manager struct {
	tokenService  *GithubApp
	keyService    *GithubApp
	secretsClient SecretsClient
	ec2Client     EC2Client
}

// NewManager creates a new manager for handling rotation of Github deploy keys and access tokens.
func NewManager(
	sess *session.Session,
	tokenServiceIntegrationID int,
	tokenServicePrivateKey string,
	keyServiceIntegrationID int,
	keyServicePrivateKey string,
) (*Manager, error) {
	tokenService, err := newGithubApp(tokenServiceIntegrationID, tokenServicePrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for token service: %s", err)
	}

	keyService, err := newGithubApp(keyServiceIntegrationID, keyServicePrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for key service: %s", err)
	}

	return &Manager{
		tokenService:  tokenService,
		keyService:    keyService,
		secretsClient: secretsmanager.New(sess),
		ec2Client:     ec2.New(sess),
	}, nil
}

// Create an access token for the organisation
func (m *Manager) createAccessToken(owner string) (string, error) {
	return m.tokenService.createInstallationToken(owner)
}

// List deploy keys for a repository
func (m *Manager) listKeys(repository Repository) ([]*github.Key, error) {
	client, err := m.keyService.getInstallationClient(repository.Owner)
	if err != nil {
		return nil, err
	}
	keys, _, err := client.Repos.ListKeys(context.TODO(), repository.Owner, repository.Name, nil)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// Create deploy key for a repository
func (m *Manager) createKey(repository Repository, title, publicKey string) error {
	client, err := m.keyService.getInstallationClient(repository.Owner)
	if err != nil {
		return err
	}
	input := &github.Key{
		ID:       nil,
		Key:      github.String(publicKey),
		URL:      nil,
		Title:    github.String(title),
		ReadOnly: github.Bool(bool(repository.ReadOnly)),
	}

	_, _, err = client.Repos.CreateKey(context.TODO(), repository.Owner, repository.Name, input)
	return err
}

// Delete a deploy key.
func (m *Manager) deleteKey(repository Repository, id int) error {
	client, err := m.keyService.getInstallationClient(repository.Owner)
	if err != nil {
		return err
	}
	_, err = client.Repos.DeleteKey(context.TODO(), repository.Owner, repository.Name, id)
	return err
}

// Write a secret to secrets manager.
func (m *Manager) describeSecret(name string) (string, error) {
	out, err := m.secretsClient.DescribeSecret(&secretsmanager.DescribeSecretInput{
		SecretId: aws.String(name),
	})
	if err != nil {
		return "", err
	}
	return aws.StringValue(out.Description), nil
}

// Write a secret to secrets manager.
func (m *Manager) writeSecret(name, secret string) error {
	var err error
	timestamp := time.Now().Format(time.RFC3339)

	_, err = m.secretsClient.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:        aws.String(name),
		Description: aws.String(fmt.Sprintf("Github credentials for Concourse. Last updated: %s", timestamp)),
	})
	if err != nil {
		e, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("failed to convert error: %s", err)
		}
		if e.Code() != secretsmanager.ErrCodeResourceExistsException {
			return err
		}
	}

	_, err = m.secretsClient.UpdateSecret(&secretsmanager.UpdateSecretInput{
		Description:  aws.String(fmt.Sprintf("Github credentials for Concourse. Last updated: %s", timestamp)),
		SecretId:     aws.String(name),
		SecretString: aws.String(secret),
	})
	return err
}

// Generate a key pair for the deploy key.
func (m *Manager) generateKeyPair(title string) (privateKey string, publicKey string, err error) {
	// Have EC2 Generate a new private key
	res, err := m.ec2Client.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: aws.String(title),
	})
	if err != nil {
		return "", "", err
	}

	// Remember to clean up temporary key when done
	defer func() {
		// TODO: Don't discard error, handle it somehow.
		m.ec2Client.DeleteKeyPair(&ec2.DeleteKeyPairInput{
			KeyName: aws.String(title),
		})
	}()
	privateKey = aws.StringValue(res.KeyMaterial)

	// Parse the private key
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return "", "", errors.New("failed to decode private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", "", err
	}

	public, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", err
	}
	publicKey = string(ssh.MarshalAuthorizedKey(public))

	return privateKey, publicKey, nil
}
