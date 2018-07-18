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

// Manager handles API calls to AWS.
type Manager struct {
	githubClients map[string]GithubClient
	secretsClient SecretsClient
	ec2Client     EC2Client
}

// NewManager creates a new manager from a session, region, Github integration ID and private key.
func NewManager(sess *session.Session, region string, integrationID int, privateKey string) (*Manager, error) {
	tr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, integrationID, []byte(privateKey))
	if err != nil {
		return nil, err
	}
	app := github.NewClient(&http.Client{Transport: tr})

	// List installations and make sure we (only) have 1 (private app)
	installations, _, err := app.Apps.ListInstallations(context.TODO(), &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	// TODO: Refactor
	clients := make(map[string]GithubClient, len(installations))
	for _, i := range installations {
		token, _, err := app.Apps.CreateInstallationToken(context.TODO(), i.GetID())
		if err != nil {
			return nil, fmt.Errorf("failed to create installation token: %s", err)
		}
		oauth := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token.GetToken()},
		))
		client := github.NewClient(oauth)
		owner := i.GetAccount().GetLogin()
		if owner == "" {
			return nil, errors.New("failed to get owner for github app")
		}
		clients[owner] = GithubClient{
			Repos: client.Repositories,
			Apps:  client.Apps,
		}
	}

	config := &aws.Config{Region: aws.String(region)}
	return &Manager{
		githubClients: clients,
		secretsClient: secretsmanager.New(sess, config),
		ec2Client:     ec2.New(sess, config),
	}, nil
}

// NewTestManager ...
func NewTestManager(owner string, r RepoClient, a AppsClient, s SecretsClient, e EC2Client) *Manager {
	gh := GithubClient{Repos: r, Apps: a}
	return &Manager{secretsClient: s, ec2Client: e, githubClients: map[string]GithubClient{owner: gh}}
}

// GetInstallationClient returns the Github client for a particular installation.
func (m *Manager) GetInstallationClient(owner string) (*GithubClient, error) {
	client, ok := m.githubClients[owner]
	if !ok {
		return nil, fmt.Errorf("the github app is not installed for user or org: '%s'", owner)
	}
	return &client, nil
}

// ListKeys for a repository.
func (m *Manager) ListKeys(repository Repository) ([]*github.Key, error) {
	client, err := m.GetInstallationClient(repository.Owner)
	if err != nil {
		return nil, err
	}
	keys, _, err := client.Repos.ListKeys(context.TODO(), repository.Owner, repository.Name, nil)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// CreateKey for a repository.
func (m *Manager) CreateKey(repository Repository, title, publicKey string) error {
	client, err := m.GetInstallationClient(repository.Owner)
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

// DeleteKey for a repository.
func (m *Manager) DeleteKey(repository Repository, id int) error {
	client, err := m.GetInstallationClient(repository.Owner)
	if err != nil {
		return err
	}
	_, err = client.Repos.DeleteKey(context.TODO(), repository.Owner, repository.Name, id)
	return err
}

// WriteSecret to secrets manager.
func (m *Manager) WriteSecret(name, secret string) error {
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
		Description:  aws.String(fmt.Sprintf("Github deploy key for Concourse. Last updated: %s", timestamp)),
		SecretId:     aws.String(name),
		SecretString: aws.String(secret),
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
