package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/go-github/github"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// Command options
type Command struct {
	Region string `long:"region" env:"REGION" default:"eu-west-1" description:"AWS region to use for API calls."`
	Path   string `long:"path" env:"SSM_PATH" default:"/concourse/{{.Team}}/{{.Repository}}" description:"Path to use when writing to SSM."`
}

// Validate the Command options.
func (c *Command) Validate() error {
	if c.Region == "" {
		return errors.New("missing CONFIG_REGION")
	}
	return nil
}

func Handler() error {
	var command Command

	// Parse flags and validate
	_, err := flags.Parse(&command)
	if err != nil {
		return errors.Wrap(err, "failed to parse flags")
	}
	if err := command.Validate(); err != nil {
		return errors.Wrap(err, "invalid command")
	}

	// Oauth http client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ""},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Github client
	client := github.NewClient(tc)

	// List keys
	keys, _, err := client.Repositories.ListKeys(ctx, "itsdalmo", "", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(keys)

	// Create key
	_, public, err := generateKeyPair()
	if err != nil {
		panic(err)
	}

	key, _, err := client.Repositories.CreateKey(
		ctx, "itsdalmo", "Privat", &github.Key{
			ID:       nil,
			Key:      github.String(string(public)),
			URL:      nil,
			Title:    github.String("concourse-deploy-key"),
			ReadOnly: github.Bool(true),
		},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(key)
	return nil
}

func main() {
	lambda.Start(Handler)
}
