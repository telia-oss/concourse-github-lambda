package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/google/go-github/github"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

// RepositoriesService interface
type RepositoriesService interface {
	ListKeys(ctx context.Context, owner string, repo string, opt *github.ListOptions) ([]*github.Key, *github.Response, error)
	CreateKey(ctx context.Context, owner string, repo string, key *github.Key) (*github.Key, *github.Response, error)
	DeleteKey(ctx context.Context, owner string, repo string, id int) (*github.Response, error)
}

// Manager handles API calls to AWS.
type Manager struct {
	repoClient RepositoriesService
	ssmClient  ssmiface.SSMAPI
	ec2Client  ec2iface.EC2API
	region     string
	owner      string
	ctx        context.Context
}

// NewManager creates a new manager from a session, region and Github access token.
func NewManager(sess *session.Session, region, owner, token string) *Manager {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	config := &aws.Config{Region: aws.String(region)}
	return &Manager{
		repoClient: github.NewClient(tc).Repositories,
		ssmClient:  ssm.New(sess, config),
		ec2Client:  ec2.New(sess, config),
		region:     region,
		owner:      owner,
		ctx:        context.Background(),
	}
}

// ListKeys for a repository.
func (m *Manager) ListKeys(repository string) ([]*github.Key, error) {
	keys, _, err := m.repoClient.ListKeys(m.ctx, m.owner, repository, nil)
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
		ReadOnly: github.Bool(true),
	}

	key, _, err := m.repoClient.CreateKey(m.ctx, m.owner, repository, input)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// DeleteKey for a repository.
func (m *Manager) DeleteKey(repository string, id int) error {
	_, err := m.repoClient.DeleteKey(m.ctx, m.owner, repository, id)
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
