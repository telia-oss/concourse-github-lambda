package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// GithubClient ...
type GithubClient struct {
	Expiration time.Time
	Repos      RepoClient
	Apps       AppsClient
}

func (c *GithubClient) isExpired() bool {
	return c.Expiration.Before(time.Now().Add(1 * time.Minute))
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

func (a *GithubApp) createInstallationToken(owner string) (token string, expiration time.Time, err error) {
	owner = strings.ToLower(owner)
	id, ok := a.Installations[owner]
	if !ok {
		return token, expiration, fmt.Errorf("the deploy key app is not installed for user or org: '%s'", owner)
	}
	installationToken, _, err := a.App.CreateInstallationToken(context.TODO(), id)
	if err != nil {
		return token, expiration, fmt.Errorf("failed to create token: %s", err)
	}
	token, expiration = installationToken.GetToken(), installationToken.GetExpiresAt()
	return token, expiration, nil
}

func (a *GithubApp) getInstallationClient(owner string) (client *GithubClient, err error) {
	owner = strings.ToLower(owner)
	if c, ok := a.Clients[owner]; !ok || c.isExpired() {
		token, expiration, err := a.createInstallationToken(owner)
		if err != nil {
			return nil, fmt.Errorf("failed to get installation token: %s", err)
		}
		oauth := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		))
		client := github.NewClient(oauth)
		a.Clients[owner] = &GithubClient{
			Repos:      client.Repositories,
			Apps:       client.Apps,
			Expiration: expiration,
		}
	}
	client, _ = a.Clients[owner]
	return client, nil
}
