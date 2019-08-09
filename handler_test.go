package handler_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/golang/mock/gomock"
	"github.com/google/go-github/github"
	logrus "github.com/sirupsen/logrus/hooks/test"
	"github.com/telia-oss/concourse-github-lambda"
	"github.com/telia-oss/concourse-github-lambda/mocks"
)

func TestHandler(t *testing.T) {
	owner := "telia-oss"

	team := handler.Team{
		Name: "test-team",
		Repositories: []handler.Repository{
			{
				Name:     "test-repository",
				Owner:    owner,
				ReadOnly: true,
			},
		},
	}

	tests := []struct {
		description       string
		tokenPath         string
		keyPath           string
		keyTitle          string
		team              handler.Team
		existingKey       *github.Key
		secretLastUpdated string
		shouldRotate      bool
	}{

		{
			description: "creates new keys and writes them to github and secrets manager",
			tokenPath:   "/concourse/{{.Team}}/{{.Owner}}",
			keyPath:     "/concourse/{{.Team}}/{{.Repository}}",
			keyTitle:    "concourse-{{.Team}}-deploy-key",
			team:        team,
			existingKey: &github.Key{
				ID:       github.Int64(1),
				Title:    github.String("concourse-test-team-deploy-key"),
				ReadOnly: github.Bool(true),
			},
			secretLastUpdated: time.Now().AddDate(0, 0, -10).UTC().Format(time.RFC3339),
			shouldRotate:      true,
		},
		{
			description: "does not rotate keys if they have recently been updated",
			tokenPath:   "/concourse/{{.Team}}/{{.Owner}}",
			keyPath:     "/concourse/{{.Team}}/{{.Repository}}",
			keyTitle:    "concourse-{{.Team}}-deploy-key",
			team:        team,
			existingKey: &github.Key{
				ID:       github.Int64(1),
				Title:    github.String("concourse-test-team-deploy-key"),
				ReadOnly: github.Bool(true),
			},
			secretLastUpdated: time.Now().UTC().Format(time.RFC3339),
		},
		{
			description: "rotates recently updated keys if the desired permissions have changed",
			tokenPath:   "/concourse/{{.Team}}/{{.Owner}}",
			keyPath:     "/concourse/{{.Team}}/{{.Repository}}",
			keyTitle:    "concourse-{{.Team}}-deploy-key",
			team:        team,
			existingKey: &github.Key{
				ID:       github.Int64(1),
				Title:    github.String("concourse-test-team-deploy-key"),
				ReadOnly: github.Bool(false),
			},
			secretLastUpdated: time.Now().UTC().Format(time.RFC3339),
			shouldRotate:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			newTokenExpiration := time.Now().Add(1 * time.Hour)
			newToken := &github.InstallationToken{Token: github.String("token"), ExpiresAt: &newTokenExpiration}

			apps := mocks.NewMockAppsClient(ctrl)
			apps.EXPECT().CreateInstallationToken(gomock.Any(), gomock.Any()).MinTimes(1).Return(newToken, nil, nil)

			repos := mocks.NewMockRepoClient(ctrl)
			repos.EXPECT().ListKeys(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]*github.Key{tc.existingKey}, nil, nil)
			if tc.shouldRotate {
				repos.EXPECT().CreateKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil, nil)
				repos.EXPECT().DeleteKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
			}

			secrets := mocks.NewMockSecretsClient(ctrl)
			description := &secretsmanager.DescribeSecretOutput{
				Description: aws.String(fmt.Sprintf("Github credentials for Concourse. Last updated: %s", tc.secretLastUpdated)),
			}
			if *tc.existingKey.ReadOnly == bool(tc.team.Repositories[0].ReadOnly) {
				secrets.EXPECT().DescribeSecret(gomock.Any()).MinTimes(1).Return(description, nil)
			}
			secrets.EXPECT().CreateSecret(gomock.Any()).MinTimes(1).Return(nil, nil)
			secrets.EXPECT().UpdateSecret(gomock.Any()).MinTimes(1).Return(nil, nil)

			// TODO: If we want to test teams with multiple repos we'll need to create installations/clients in a loop.
			services := &handler.GithubApp{
				App:           apps,
				Installations: map[string]int64{tc.team.Repositories[0].Owner: 1},
				Clients: map[string]*handler.GithubClient{
					tc.team.Repositories[0].Owner: {
						Apps:       apps,
						Repos:      repos,
						Expiration: time.Now().Add(1 * time.Hour),
					},
				},
			}
			manager := handler.NewTestManager(secrets, services, services)
			logger, hook := logrus.NewNullLogger()
			handle := handler.New(manager, tc.tokenPath, tc.keyPath, tc.keyTitle, logger)

			if err := handle(tc.team); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			// Look for warning, error, fatal and panic level logs
			for _, e := range hook.AllEntries() {
				if e.Level <= 3 {
					t.Errorf("unexpected log severity: '%s': %s", e.Level.String(), e.Message)
				}
			}
		})
	}
}
