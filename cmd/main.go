package main

import (
	"errors"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/telia-oss/concourse-github-lambda"
)

// Command options
type Command struct {
	Region        string `long:"region" env:"REGION" description:"AWS region to use for API calls."`
	Path          string `long:"secrets-manager-path" env:"SECRETS_MANAGER_PATH" default:"/concourse/{{.Team}}/{{.Repository}}-deploy-key" description:"Path to use when writing to AWS Secrets manager."`
	Title         string `long:"github-title" env:"GITHUB_TITLE" default:"concourse-{{.Team}}-deploy-key" description:"Template for Github title."`
	PrivateKey    string `long:"github-private-key" env:"GITHUB_PRIVATE_KEY" description:"Private key for the Github App."`
	IntegrationID int    `long:"github-integration-id" env:"GITHUB_INTEGRATION_ID" description:"Integration ID for the Github App."`
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
	var command Command

	_, err := flags.Parse(&command)
	if err != nil {
		panic(fmt.Errorf("failed to parse flag %s", err))
	}
	if err := command.Validate(); err != nil {
		panic(fmt.Errorf("invalid command: %s", err))
	}
	sess, err := session.NewSession()
	if err != nil {
		panic(fmt.Errorf("failed to create new session: %s", err))
	}

	manager, err := handler.NewManager(sess, command.Region, command.IntegrationID, command.PrivateKey)
	if err != nil {
		panic(fmt.Errorf("failed to create new manager: %s", err))
	}

	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}

	f := handler.New(manager, command.Path, command.Title, logger)
	lambda.Start(f)
}
