package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

// Manager handles API calls to AWS.
type Manager struct {
	githubClient *github.Client
	githubOwner  string
	ssmClient    ssmiface.SSMAPI
	ec2Client    ec2iface.EC2API
}

// NewManager creates a new manager for handling requests to AWS and Github.
func NewManager(region string, integrationID int, privateKey string) (*Manager, error) {
	// Create a client to interact with the Github App endpoints
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
	if len(installations) != 1 {
		switch len(installations) {
		case 0:
			return nil, errors.New("application has zero installations")
		default:
			return nil, errors.New("application has multiple installations")
		}
	}

	// Get an installation token and create a new Github Client.
	token, _, err := app.Apps.CreateInstallationToken(context.TODO(), *installations[0].ID)
	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token.GetToken()},
	))

	// Create AWS session and config
	awsSession := session.Must(session.NewSession())
	awsConfig := &aws.Config{Region: aws.String(region)}

	return &Manager{
		githubClient: github.NewClient(tc),
		githubOwner:  *installations[0].Account.Login,
		ssmClient:    ssm.New(awsSession, awsConfig),
		ec2Client:    ec2.New(awsSession, awsConfig),
	}, nil
}

// ListRepositories for an installation.
func (m *Manager) ListRepositories() ([]*github.Repository, error) {
	// TODO: Paginate the response
	repos, _, err := m.githubClient.Apps.ListRepos(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	return repos, nil
}

// ListKeys for a repository.
func (m *Manager) ListKeys(repository string) ([]*github.Key, error) {
	// TODO: Paginate the response
	keys, _, err := m.githubClient.Repositories.ListKeys(context.TODO(), m.githubOwner, repository, nil)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// CreateKey for a repository.
func (m *Manager) CreateKey(repository, title, publicKey string) (*github.Key, error) {
	input := &github.Key{
		ID:       nil,
		Key:      github.String(publicKey),
		URL:      nil,
		Title:    github.String(title),
		ReadOnly: github.Bool(false),
	}

	key, _, err := m.githubClient.Repositories.CreateKey(context.TODO(), m.githubOwner, repository, input)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// DeleteKey for a repository.
func (m *Manager) DeleteKey(repository string, id int64) error {
	_, err := m.githubClient.Repositories.DeleteKey(context.TODO(), m.githubOwner, repository, int(id))
	return err
}

// WriteSecret to SSM.
func (m *Manager) WriteSecret(name, value, key string) error {
	input := &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		KeyId:     aws.String(key),
		Type:      aws.String("SecureString"),
		Overwrite: aws.Bool(true),
	}
	_, err := m.ssmClient.PutParameter(input)
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
		m.ec2Client.DeleteKeyPair(&ec2.DeleteKeyPairInput{
			KeyName: aws.String(title),
		})
	}()
	privateKey = aws.StringValue(res.KeyMaterial)

	// Parse the private key
	block, _ := pem.Decode([]byte(privateKey))
	if err != nil {
		return "", "", err
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
