package handler

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/google/go-github/github"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

// GithubManager for testing purposes
//go:generate mockgen -destination=mocks/mock_github_manager.go -package=mocks github.com/telia-oss/concourse-github-lambda GithubManager
type GithubManager interface {
	ListKeys(ctx context.Context, owner string, repo string, opt *github.ListOptions) ([]*github.Key, *github.Response, error)
	CreateKey(ctx context.Context, owner string, repo string, key *github.Key) (*github.Key, *github.Response, error)
	DeleteKey(ctx context.Context, owner string, repo string, id int) (*github.Response, error)
}

// SecretsManager for testing purposes.
//go:generate mockgen -destination=mocks/mock_secrets_manager.go -package=mocks github.com/telia-oss/concourse-github-lambda SecretsManager
type SecretsManager secretsmanageriface.SecretsManagerAPI

// EC2Manager for testing purposes.
//go:generate mockgen -destination=mocks/mock_ec2_manager.go -package=mocks github.com/telia-oss/concourse-github-lambda EC2Manager
type EC2Manager ec2iface.EC2API

// Manager handles API calls to AWS.
type Manager struct {
	githubClient  GithubManager
	secretsClient SecretsManager
	ec2Client     EC2Manager
}

// NewManager creates a new manager from a session, region and Github access token.
func NewManager(sess *session.Session, region, token string) *Manager {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	config := &aws.Config{Region: aws.String(region)}
	return &Manager{
		githubClient:  github.NewClient(tc).Repositories,
		secretsClient: secretsmanager.New(sess, config),
		ec2Client:     ec2.New(sess, config),
	}
}

// NewTestManager ...
func NewTestManager(g GithubManager, s SecretsManager, e EC2Manager) *Manager {
	return &Manager{githubClient: g, secretsClient: s, ec2Client: e}
}

// ListKeys for a repository.
func (m *Manager) ListKeys(repository Repository) ([]*github.Key, error) {
	keys, _, err := m.githubClient.ListKeys(context.TODO(), repository.Owner, repository.Name, nil)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// CreateKey for a repository.
func (m *Manager) CreateKey(repository Repository, title, publicKey string) error {
	input := &github.Key{
		ID:       nil,
		Key:      github.String(publicKey),
		URL:      nil,
		Title:    github.String(title),
		ReadOnly: github.Bool(bool(repository.ReadOnly)),
	}

	_, _, err := m.githubClient.CreateKey(context.TODO(), repository.Owner, repository.Name, input)
	return err
}

// DeleteKey for a repository.
func (m *Manager) DeleteKey(repository Repository, id int) error {
	_, err := m.githubClient.DeleteKey(context.TODO(), repository.Owner, repository.Name, id)
	return err
}

// WriteSecret to secrets manager.
func (m *Manager) WriteSecret(name, secret string) error {
	var err error

	_, err = m.secretsClient.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:        aws.String(name),
		Description: aws.String("Lambda created secret for Concourse."),
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

	_, err = m.secretsClient.PutSecretValue(&secretsmanager.PutSecretValueInput{
		SecretId:      aws.String(name),
		SecretString:  aws.String(secret),
		VersionStages: []*string{aws.String("AWSCURRENT")},
	})
	return err
}

// GenerateKeyPair to use as deploy key.
func (m *Manager) GenerateKeyPair(title string) (privateKey string, publicKey string, err error) {
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
