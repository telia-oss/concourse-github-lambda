package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/jessevdk/go-flags"
)

// Command options
type Command struct {
	TeamName      string `long:"team-name" env:"TEAM_NAME" description:"Name of the team."`
	Path          string `long:"path" env:"SSM_PATH" default:"/concourse/{{.Team}}/{{.Repository}}-deploy-key" description:"Path to use when writing to SSM."`
	Title         string `long:"title" env:"GITHUB_TITLE" default:"concourse-{{.Team}}-deploy-key" description:"Template for Github title."`
	Region        string `long:"region" env:"REGION" default:"eu-west-1" description:"AWS region to use for API calls."`
	KMSKeyID      string `long:"kms-key-id" env:"KMS_KEY_ID" description:"KMS Key ID for encrypting the secret."`
	PrivateKey    string `long:"private-key" env:"GITHUB_PRIVATE_KEY" description:"Private key for the Github App."`
	IntegrationID int    `long:"integration-id" env:"GITHUB_INTEGRATION_ID" description:"Integration ID for the Github App."`
}

// Validate the Command options.
func (c *Command) Validate() error {
	if c.TeamName == "" {
		return errors.New("missing TEAM_NAME")
	}
	if c.Region == "" {
		return errors.New("missing REGION")
	}
	if c.PrivateKey == "" {
		return errors.New("missing GITHUB_PRIVATE_KEY")
	}
	if c.IntegrationID == 0 {
		return errors.New("missing GITHUB_INTEGRATION_ID")
	}
	return nil
}

// Handler for Lambda
func Handler(interface{}) error {
	var command Command
	_, err := flags.Parse(&command)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %s", err)
	}
	if err := command.Validate(); err != nil {
		return fmt.Errorf("invalid command: %s", err)
	}

	manager, err := NewManager(command.Region, command.IntegrationID, command.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to create new manager: %s", err)
	}
	repos, err := manager.ListRepositories()
	if err != nil {
		return fmt.Errorf("failed to list repositories: %s", err)
	}

	for _, repo := range repos {
		var oldKey *github.Key

		keyTitle, err := NewPath(command.TeamName, repo.GetName(), command.Title).String()
		if err != nil {
			log.Printf("failed to parse github key title: %s", err)
			continue
		}

		// Look for existing keys for the team
		keys, err := manager.ListKeys(repo.GetName())
		for _, key := range keys {
			if *key.Title == keyTitle {
				oldKey = key
			}
		}

		// Generate a new key pair
		private, public, err := manager.GenerateKeyPair(keyTitle)
		if err != nil {
			log.Printf("failed to generate key pair: %s", err)
			continue
		}

		// Write the new public key to Github
		if _, err = manager.CreateKey(repo.GetName(), keyTitle, public); err != nil {
			log.Printf("failed to create key on github: %s", err)
			continue
		}

		// Write the new private key to SSM
		secretPath, err := NewPath(command.TeamName, repo.GetName(), command.Path).String()
		if err != nil {
			log.Printf("failed to parse ssm secret path: %s", err)
			continue
		}
		if err = manager.WriteSecret(secretPath, private, command.KMSKeyID); err != nil {
			log.Printf("failed to write private key to ssm: %s", err)
			continue
		}

		// Sleep before deleting old key (in case someone has just fetched the old key)
		if oldKey != nil {
			time.Sleep(time.Second * 1)
			if err = manager.DeleteKey(repo.GetName(), oldKey.GetID()); err != nil {
				log.Printf("failed to delete old key: %d: %s", *oldKey.ID, err)
				continue
			}
		}
	}
	return nil
}

func main() {
	// lambda.Start(Handler)
	Handler(nil)
}

// NewPath a new secret path...
func NewPath(team string, repository string, template string) *Path {
	return &Path{
		Team:       team,
		Repository: repository,
		Template:   template,
	}
}

// Path represents the path used to write secrets into SSM.
type Path struct {
	Team       string
	Repository string
	Template   string
}

func (p *Path) String() (string, error) {
	t, err := template.New("path").Option("missingkey=error").Parse(p.Template)
	if err != nil {
		return "", err
	}
	var s strings.Builder
	if err = t.Execute(&s, p); err != nil {
		return "", err
	}
	return s.String(), nil
}
