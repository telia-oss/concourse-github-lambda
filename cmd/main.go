package main

import (
	"errors"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/telia-oss/aws-env"
	"github.com/telia-oss/concourse-github-lambda"
)

// Command options
type Command struct {
	Region        string `long:"region" env:"REGION" description:"AWS region to use for API calls."`
	Path          string `long:"secrets-manager-path" env:"SECRETS_MANAGER_PATH" default:"/concourse/{{.Team}}/{{.Repository}}-deploy-key" description:"Path to use when writing to AWS Secrets manager."`
	Title         string `long:"github-title" env:"GITHUB_TITLE" default:"concourse-{{.Team}}-deploy-key" description:"Template for Github title."`
	IntegrationID int    `long:"github-integration-id" env:"GITHUB_INTEGRATION_ID" description:"Integration ID for the Github App."`
	PrivateKey    string `long:"github-private-key" env:"GITHUB_PRIVATE_KEY" description:"Private key for the Github App."`
}

// Validate the Command options.
func (c *Command) Validate() error {
	if c.Region == "" {
		return errors.New("missing required argument 'region'")
	}
	if c.PrivateKey == "" {
		return errors.New("missing required argument 'github-private-key'")
	}
	if c.IntegrationID == 0 {
		return errors.New("missing required argument 'github-integration-id'")
	}
	return nil
}

func main() {
	// Set up a logger
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}

	// New AWS Session with the default providers
	sess, err := session.NewSession()
	if err != nil {
		logger.Fatalf("failed to create a new session: %s", err)
	}

	// Exchange secrets in environment variables with their values.
	env, err := awsenv.New(sess, logger)
	if err != nil {
		logger.Fatalf("failed to initialize awsenv: %s", err)
	}
	if err := env.Replace(); err != nil {
		logger.Fatalf("failed to replace environment variables: %s", err)
	}

	// Parse environment variables
	var command Command
	_, err = flags.Parse(&command)
	if err != nil {
		logger.Fatalf("failed to parse flag: %s", err)
	}
	if err := command.Validate(); err != nil {
		logger.Fatalf("invalid command: %s", err)
	}

	// Create new manager
	manager, err := handler.NewManager(sess, command.Region, command.IntegrationID, command.PrivateKey)
	if err != nil {
		logger.Fatalf("failed to create new manager: %s", err)
	}

	// Run
	f := handler.New(manager, command.Path, command.Title, logger)
	lambda.Start(f)
}
