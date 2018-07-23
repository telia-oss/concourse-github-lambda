package handler

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
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
}

// GithubClient ...
type GithubClient struct {
	Repos RepoClient
	Apps  AppsClient
}

// SecretsClient for testing purposes.
//go:generate mockgen -destination=mocks/mock_secrets_client.go -package=mocks github.com/telia-oss/concourse-github-lambda SecretsClient
type SecretsClient secretsmanageriface.SecretsManagerAPI

// EC2Client for testing purposes.
//go:generate mockgen -destination=mocks/mock_ec2_client.go -package=mocks github.com/telia-oss/concourse-github-lambda EC2Client
type EC2Client ec2iface.EC2API

// NewTestManager for testing purposes.
func NewTestManager(g map[string]GithubClient, s SecretsClient, e EC2Client) *Manager {
	return &Manager{secretsClient: s, ec2Client: e, deployKeyClients: g}
}

// Manager handles API calls to AWS.
type Manager struct {
	accessTokens     map[string]string
	deployKeyClients map[string]GithubClient
	secretsClient    SecretsClient
	ec2Client        EC2Client
}

// NewManager creates a new manager for handling rotation of Github deploy keys and access tokens.
func NewManager(
	sess *session.Session,
	region string,
	tokenServiceIntegrationID int,
	tokenServicePrivateKey string,
	keyServiceIntegrationID int,
	keyServicePrivateKey string,
) (*Manager, error) {
	accessTokens, err := createInstallationTokens(tokenServiceIntegrationID, tokenServicePrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate installation tokens for access token app: %s", err)
	}

	deployKeyTokens, err := createInstallationTokens(keyServiceIntegrationID, keyServicePrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate installation tokens for deploy key app: %s", err)
	}

	deployKeyClients := make(map[string]GithubClient, len(deployKeyTokens))
	for owner, token := range deployKeyTokens {
		oauth := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		))
		client := github.NewClient(oauth)
		deployKeyClients[owner] = GithubClient{
			Repos: client.Repositories,
			Apps:  client.Apps,
		}
	}

	config := &aws.Config{Region: aws.String(region)}
	return &Manager{
		accessTokens:     accessTokens,
		deployKeyClients: deployKeyClients,
		secretsClient:    secretsmanager.New(sess, config),
		ec2Client:        ec2.New(sess, config),
	}, nil
}

func createInstallationTokens(integrationID int, privateKey string) (map[string]string, error) {
	tr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, integrationID, []byte(privateKey))
	if err != nil {
		return nil, err
	}
	client := github.NewClient(&http.Client{Transport: tr})

	// List installations (TODO: Paginate results.)
	installations, _, err := client.Apps.ListInstallations(context.TODO(), &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list installations: %s", err)
	}

	tokens := make(map[string]string, len(installations))

	for _, i := range installations {
		owner := i.GetAccount().GetLogin()
		if owner == "" {
			return nil, fmt.Errorf("failed to get owner for installation: %d", i.GetID())
		}
		token, _, err := client.Apps.CreateInstallationToken(context.TODO(), i.GetID())
		if err != nil {
			return nil, fmt.Errorf("failed to create token for installation: %s", err)
		}
		tokens[owner] = token.GetToken()
	}

	return tokens, nil
}

// Get an access token for an installation of the access token Github App.
func (m *Manager) getAccessToken(owner string) (string, error) {
	token, ok := m.accessTokens[owner]
	if !ok {
		return "", fmt.Errorf("the access token app is not installed for user or org: '%s'", owner)
	}
	return token, nil
}

// Get a Github client for the deploy key Github App.
func (m *Manager) getInstallationClient(owner string) (*GithubClient, error) {
	client, ok := m.deployKeyClients[owner]
	if !ok {
		return nil, fmt.Errorf("the deploy key app is not installed for user or org: '%s'", owner)
	}
	return &client, nil
}

// List deploy keys for a repository
func (m *Manager) listKeys(repository Repository) ([]*github.Key, error) {
	client, err := m.getInstallationClient(repository.Owner)
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
	client, err := m.getInstallationClient(repository.Owner)
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
	client, err := m.getInstallationClient(repository.Owner)
	if err != nil {
		return err
	}
	_, err = client.Repos.DeleteKey(context.TODO(), repository.Owner, repository.Name, id)
	return err
}

// Write a secret to secrets manager.
func (m *Manager) writeSecret(name, secret string) error {
	var err error

	_, err = m.secretsClient.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:        aws.String(name),
		Description: aws.String("Github deploy key for Concourse."),
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

	timestamp := time.Now().Format(time.RFC3339)

	_, err = m.secretsClient.UpdateSecret(&secretsmanager.UpdateSecretInput{
		Description:  aws.String(fmt.Sprintf("Github credential for Concourse. Last updated: %s", timestamp)),
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
