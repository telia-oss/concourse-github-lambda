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

const keyMaterial = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAm9RgNyONxqSQHGhMk05iRxHSZ1PKxAPBioDjvzBwNyEVifGmiSmqcoeXgLQU\nQzFSTVgffLkPjndHvMrq+Shq0eSwsulSvyR5B+cL+ob7XPMkzO+2vmNAVfcBbG7jJ7kqdwP0KH3g\nZZ1+dgRfxSR/ziWRf0iiILc7mfPPrQ2W8MAfokS4kmw5OpvYlug05gje76CZtMR+/Ium7En+Ul/j\n8TuoahQno9LkxiXl8huEBM3VO6wQ7IAHvQHhoXb6w4pBFybgA3p3ZftSsY2LZHLAmXNxwzSMNACC\n+Q/Z1XejGNamjebSI3fgWghg6aAlvD6qjyx7AUEr+dbsHeHTaZzoIQIDAQABAoIBAQCCxeUFAQJf\nHQWPwXvZ92MEj5FKg4hbnWdT67y1W1og+dPQkwqWe2/+c4oSSY3jocWXAQhTrB7BCZsbdhNhi6ix\ngsFDNAnsPRiRKDXmRlc2dxqAHf/3oOWB/yujqx9Y280mWhwRyymBPX2+XwdcM7hJ8T88WWEuIXeU\nSIcVjJ0KZnFFmlQ0lm4bLR6nxccJROGhmYlhzxZCi+OroLjCA0usOhOPMiOxs71BQxSb4PyKiL0V\n1pgpat5UdG2pGZXoiYxmU5YWRv/IoOvvBjaE7vACJJEBiIv7T4yX1n6TrRvtHhI4fVkguGHkdf0C\nEbu55AUe17ga2aAfHfGBf48aznEdAoGBANbTyVlsEEgJkme5kElgmmCkkqTQy/HAApKDuX/WsecF\nFS4A3zw5mcde7NsW8dXcc+2EwZtE99+Wl1PhR8vSomV+K5tkNLUb+PFtEIDtIsIaczxzCuDyMDcY\nPyQ/VrUC5arE2M9sr5do/AqsxzlCZLEL7Uaqt2j+YR9TAvPLQ3NfAoGBALmx8jzkZAm5KRV2T6ng\nctm8XbWI5D5EiTyp+C74JOpNL8F+xeSpa/GQ3vKvTwu0NlOwn9FkePOKu+Nf9T9E1yvW3ppY3Iuf\nSJLlPEO3oyiewISskr6ueAf17tPXOtD3HR3+idbp4heNUsOOWeP5Rey+5F6dB3Nk2ZjUrXdp5NR/\nAoGBAJHKUM7642G//TefWygxAxOrHEn12TJLGHPOKUl0rm8Vp/X8aYM5o/8FkMBupdh5L8N1YN66\nw21diX1HWa4dWFCAe5+NNafjP+K4HYchZ4FK6gGQIUXflpENR2yV/4YAXVSzGmBKZi/e841bDCjz\nwdnVOkXG/YmneMoFT++bdj8JAoGBAJ+zfVyHI84E82Nk4/B6euvthz434+v1b32/xBVJDh5/kYG8\n8J7OYmpXqJZY1QeAznQ9Y8Vmvmrdtuc+wKHQJ6mpWrqtj8d4jqbfBWxLw8OMfI/eBzp8u/hEt0hz\nQz8yN1VzcsJlVS/iN/q9M2vQFyYbqjYAoMbKRiWdSy524PkrAoGAEOp+uT0mUy9c6T8Pk3I+ASZb\njCh03+/v87AFdInVNETZNJuR6IaoRW44+n9+3ClrbWFz+PJYisNHrsTqtMxKDDIjIaTohxjhNQGP\nsm53ZjEVsGPT+9NI8QZvbHVMB5lGFqD1riihTBlZms3YjKmPv6Z7svnh8w1R5tDhZ001Yjw=\n-----END RSA PRIVATE KEY-----"

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
		description string
		tokenPath   string
		keyPath     string
		keyTitle    string
		team        handler.Team
		githubKeys  []*github.Key
		createdKey  *ec2.CreateKeyPairOutput
	}{

		{
			description: "creates new keys and writes them to github and secrets manager",
			tokenPath:   "/concourse/{{.Team}}/{{.Owner}}",
			keyPath:     "/concourse/{{.Team}}/{{.Repository}}",
			keyTitle:    "concourse-{{.Team}}-deploy-key",
			team:        team,
			githubKeys: []*github.Key{
				{
					ID:    github.Int64(1),
					Title: github.String("concourse-test-team-deploy-key"),
				},
			},
			createdKey: &ec2.CreateKeyPairOutput{
				KeyMaterial: aws.String(keyMaterial),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repos := mocks.NewMockRepoClient(ctrl)
			repos.EXPECT().ListKeys(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(tc.githubKeys, nil, nil)
			repos.EXPECT().CreateKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil, nil)
			repos.EXPECT().DeleteKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

			apps := mocks.NewMockAppsClient(ctrl)
			apps.EXPECT().CreateInstallationToken(gomock.Any(), gomock.Any()).Times(1).Return(&github.InstallationToken{Token: github.String("token")}, nil, nil)

			ec2 := mocks.NewMockEC2Client(ctrl)
			ec2.EXPECT().CreateKeyPair(gomock.Any()).Times(1).Return(tc.createdKey, nil)
			ec2.EXPECT().DeleteKeyPair(gomock.Any()).Times(1)

			secrets := mocks.NewMockSecretsClient(ctrl)
			secrets.EXPECT().CreateSecret(gomock.Any()).MinTimes(1).Return(nil, nil)
			secrets.EXPECT().UpdateSecret(gomock.Any()).MinTimes(1).Return(nil, nil)

			// TODO: If we want to test teams with multiple repos we'll need to create installations/clients in a loop.
			services := &handler.GithubApp{
				App:           apps,
				Installations: map[string]int64{tc.team.Repositories[0].Owner: 1},
				Clients:       map[string]*handler.GithubClient{tc.team.Repositories[0].Owner: {Apps: apps, Repos: repos}},
			}
			logger, _ := logrus.NewNullLogger()
			handle := handler.New(handler.NewTestManager(secrets, ec2, services, services), tc.tokenPath, tc.keyPath, tc.keyTitle, logger)

			if err := handle(tc.team); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
