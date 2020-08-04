package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	environment "github.com/telia-oss/aws-env"
	handler "github.com/telia-oss/concourse-github-lambda"
)

// Command options
type Command struct {
	TokenPath                 string `long:"token-path" env:"SECRETS_MANAGER_TOKEN_PATH" default:"/concourse/{{.Team}}/{{.Owner}}-access-token" description:"Path to use when writing access tokens to AWS Secrets manager."`
	KeyPath                   string `long:"key-path" env:"SECRETS_MANAGER_KEY_PATH" default:"/concourse/{{.Team}}/{{.Repository}}-deploy-key" description:"Path to use when writing private keys to AWS Secrets manager."`
	KeyTitle                  string `long:"key-title" env:"GITHUB_KEY_TITLE" default:"concourse-{{.Team}}-deploy-key" description:"Title to use when adding deploy keys to Github."`
	TokenServiceIntegrationID int64  `long:"token-service-integration-id" env:"GITHUB_TOKEN_SERVICE_INTEGRATION_ID" description:"Integration ID for the access token Github App." required:"true"`
	TokenServicePrivateKey    string `long:"token-service-private-key" env:"GITHUB_TOKEN_SERVICE_PRIVATE_KEY" description:"Private key for the access token Github App." required:"true"`
	KeyServiceIntegrationID   int64  `long:"key-service-integration-id" env:"GITHUB_KEY_SERVICE_INTEGRATION_ID" description:"Integration ID for the deploy key Github App." required:"true"`
	KeyServicePrivateKey      string `long:"key-service-private-key" env:"GITHUB_KEY_SERVICE_PRIVATE_KEY" description:"Private key for the deploy key Github App." required:"true"`
}

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
}

func main() {
	// New AWS Session with the default providers
	sess, err := session.NewSession()
	if err != nil {
		logger.Fatalf("failed to create a new session: %s", err)
	}

	// Exchange secrets in environment variables with their values.
	env, err := environment.New(sess)
	if err != nil {
		logger.Fatalf("failed to initialize aws-env: %s", err)
	}
	if err := env.Populate(); err != nil {
		logger.Fatalf("failed to populate environment: %s", err)
	}

	// Parse environment variables
	var command Command
	_, err = flags.Parse(&command)
	if err != nil {
		logger.Fatalf("failed to parse flag: %s", err)
	}

	// Create new manager
	manager, err := handler.NewManager(
		sess,
		command.TokenServiceIntegrationID,
		command.TokenServicePrivateKey,
		command.KeyServiceIntegrationID,
		command.KeyServicePrivateKey,
	)
	if err != nil {
		logger.Fatalf("failed to create new manager: %s", err)
	}

	// Run
	f := handler.New(manager, command.TokenPath, command.KeyPath, command.KeyTitle, logger)
	lambda.Start(f)
}
