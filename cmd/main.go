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
	Region string `long:"region" env:"REGION" description:"AWS region to use for API calls."`
	Path   string `long:"secrets-manager-path" env:"SECRETS_MANAGER_PATH" default:"/concourse/{{.Team}}/{{.Account}}" description:"Path to use when writing to AWS Secrets manager."`
	Title  string `long:"github-title" env:"GITHUB_TITLE" default:"concourse-{{.Team}}-deploy-key" description:"Template for Github title."`
	Owner  string `long:"github-owner" env:"GITHUB_OWNER" description:"Organization or individual owner of the repositories."`
	Token  string `long:"github-token" env:"GITHUB_TOKEN" description:"Access token which grants access to create deploy keys for the org."`
}

// Validate the Command options.
func (c *Command) Validate() error {
	if c.Region == "" {
		return errors.New("missing required argument 'region'")
	}
	if c.Owner == "" {
		return errors.New("missing required argument 'github-owner'")
	}
	if c.Token == "" {
		return errors.New("missing required argument 'github-token'")
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

	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}

	f := handler.New(handler.NewManager(sess, command.Region, command.Owner, command.Token), command.Path, command.Title, logger)
	lambda.Start(f)
}
