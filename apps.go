package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// GithubClient ...
type GithubClient struct {
	Repos RepoClient
	Apps  AppsClient
}

// GithubApp ...
type GithubApp struct {
	App           AppsClient
	Installations map[string]int64
	Clients       map[string]*GithubClient
}

func newGithubApp(integrationID int, privateKey string) (*GithubApp, error) {
	tr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, integrationID, []byte(privateKey))
	if err != nil {
		return nil, err
	}
	client := github.NewClient(&http.Client{Transport: tr})

	// List installations (TODO: Paginate results.)
	installations, _, err := client.Apps.ListInstallations(context.TODO(), &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list installations: %s", err)
	}

	installs := make(map[string]int64, len(installations))
	for _, i := range installations {
		owner := i.GetAccount().GetLogin()
		if owner == "" {
			return nil, fmt.Errorf("failed to get owner for installation: %d", i.GetID())
		}
		installs[strings.ToLower(owner)] = i.GetID()
	}

	return &GithubApp{
		App:           client.Apps,
		Installations: installs,
		Clients:       make(map[string]*GithubClient, len(installations)),
	}, nil
}

func (a *GithubApp) createInstallationToken(owner string) (token string, err error) {
	owner = strings.ToLower(owner)
	id, ok := a.Installations[owner]
	if !ok {
		return token, fmt.Errorf("the deploy key app is not installed for user or org: '%s'", owner)
	}
	installationToken, _, err := a.App.CreateInstallationToken(context.TODO(), id)
	if err != nil {
		return token, fmt.Errorf("failed to create token: %s", err)
	}
	token = installationToken.GetToken()
	return token, nil
}

func (a *GithubApp) getInstallationClient(owner string) (client *GithubClient, err error) {
	owner = strings.ToLower(owner)
	if _, ok := a.Clients[owner]; !ok {
		token, err := a.createInstallationToken(owner)
		if err != nil {
			return nil, fmt.Errorf("failed to get installation token: %s", err)
		}
		oauth := oauth2.NewClient(context.TODO(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		))
		client := github.NewClient(oauth)
		a.Clients[owner] = &GithubClient{
			Repos: client.Repositories,
			Apps:  client.Apps,
		}
	}
	client, _ = a.Clients[owner]
	return client, nil
}
