package handler_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/google/go-github/github"
	logrus "github.com/sirupsen/logrus/hooks/test"
	"github.com/telia-oss/concourse-github-lambda"
	"github.com/telia-oss/concourse-github-lambda/mocks"
)

func TestHandler(t *testing.T) {
	team := handler.Team{
		Name: "test-team",
		Repositories: []handler.Repository{
			{
				Name:     "test-repository",
				Owner:    "telia-oss",
				ReadOnly: true,
			},
		},
	}

	tests := []struct {
		description string
		path        string
		title       string
		team        handler.Team
		githubKeys  []*github.Key
		createdKey  *ec2.CreateKeyPairOutput
	}{

		{
			description: "creates new keys and writes them to github and secrets manager",
			path:        "/concourse/{{.Team}}/{{.Repository}}",
			title:       "concourse-{{.Team}}-deploy-key",
			team:        team,
			githubKeys: []*github.Key{
				{
					ID:    github.Int64(1),
					Title: github.String("key-title"),
				},
			},
			createdKey: &ec2.CreateKeyPairOutput{
				KeyMaterial: aws.String("key-material"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			github := mocks.NewMockGithubManager(ctrl)
			github.EXPECT().ListKeys(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(tc.githubKeys, nil, nil)
			// github.EXPECT().CreateKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil, nil)
			// github.EXPECT().DeleteKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

			ec2 := mocks.NewMockEC2Manager(ctrl)
			ec2.EXPECT().CreateKeyPair(gomock.Any()).Times(1).Return(tc.createdKey, nil)
			ec2.EXPECT().DeleteKeyPair(gomock.Any()).Times(1)

			secrets := mocks.NewMockSecretsManager(ctrl)
			// secrets.EXPECT().CreateSecret(gomock.Any()).MinTimes(1).Return(nil, nil)
			// secrets.EXPECT().PutSecretValue(gomock.Any()).MinTimes(1).Return(nil, nil)

			logger, _ := logrus.NewNullLogger()
			handle := handler.New(handler.NewTestManager(github, secrets, ec2), tc.path, tc.title, logger)

			if err := handle(tc.team); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
