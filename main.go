package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/google/go-github/github"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"time"
)

// Command options
type Command struct {
	Region string `long:"region" env:"REGION" default:"eu-west-1" description:"AWS region to use for API calls."`
	Path   string `long:"path" env:"SSM_PATH" default:"/concourse/{{.Team}}/{{.Repository}}-deploy-key" description:"Path to use when writing to SSM."`
	Title  string `long:"title" env:"GITHUB_TITLE" default:"concourse-{{.Team}}-deploy-key" description:"Template for Github title."`
	Owner  string `long:"owner" env:"GITHUB_OWNER" description:"Organization or individual owner of the repositories."`
	Token  string `long:"token" env:"GITHUB_TOKEN" description:"Access token which grants access to create deploy keys for the org."`
}

// Validate the Command options.
func (c *Command) Validate() error {
	if c.Region == "" {
		return errors.New("missing REGION")
	}
	if c.Owner == "" {
		return errors.New("missing GITHUB_OWNER")
	}
	if c.Token == "" {
		return errors.New("missing GITHUB_TOKEN")
	}
	return nil
}

// Handler for Lambda
func Handler(team Team) error {
	var command Command

	// Parse flags and validate
	_, err := flags.Parse(&command)
	if err != nil {
		return errors.Wrap(err, "failed to parse flags")
	}
	if err := command.Validate(); err != nil {
		return errors.Wrap(err, "invalid command")
	}

	// New session and manager
	sess := session.Must(session.NewSession())
	manager := NewManager(sess, command.Region, command.Owner, command.Token)

	for _, repository := range team.Repositories {
		var oldKey *github.Key

		keyTitle, err := NewPath(team.Name, repository, command.Title).String()
		if err != nil {
			return errors.Wrap(err, "failed to parse github key title")
		}

		// Look for existing keys for the team
		keys, err := manager.ListKeys(repository)
		for _, key := range keys {
			if *key.Title == keyTitle {
				oldKey = key
			}
		}

		// Generate a new key pair
		private, public, err := manager.GenerateKeyPair()

		// Write the new public key to Github
		if _, err = manager.CreateKey(repository, keyTitle, string(public)); err != nil {
			return errors.Wrap(err, "failed to create key on github")
		}

		// Write the new private key to SSM
		secretPath, err := NewPath(team.Name, repository, command.Path).String()
		if err != nil {
			return errors.Wrap(err, "failed to parse ssm secret path")
		}
		if err = manager.WriteSecret(secretPath, string(private), team.KeyID); err != nil {
			return errors.Wrap(err, "failed to write private key to ssm")
		}

		// Sleep before deleting old key (in case someone has just fetched the old key)
		if oldKey != nil {
			time.Sleep(time.Second * 1)
			if err = manager.DeleteKey(repository, int(*oldKey.ID)); err != nil {
				return errors.Wrapf(err, "failed to delete old key: %s", *oldKey.ID)
			}
		}
	}
	return nil
}

func main() {
	lambda.Start(Handler)
}
