package main_test

import (
	"encoding/json"
	pkg "github.com/TeliaSoneraNorge/concourse-github-lambda"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestConfig(t *testing.T) {
	input := strings.TrimSpace(`
{
    "name": "team",
    "keyId": "key",
    "repositories": [
        {
            "name": "repo1",
            "readOnly": "true"
        },
        {
            "name": "repo2",
            "readOnly": "0"
        }
    ]
}
`)

	t.Run("Unmarshal works as intended", func(t *testing.T) {
		expected := pkg.Team{
			Name:  "team",
			KeyID: "key",
			Repositories: []pkg.Repository{
				{
					Name:     "repo1",
					ReadOnly: pkg.BooleanString(true),
				},
				{
					Name:     "repo2",
					ReadOnly: pkg.BooleanString(false),
				},
			},
		}

		var actual pkg.Team
		err := json.Unmarshal([]byte(input), &actual)

		assert.Nil(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestSecretPath(t *testing.T) {
	var (
		team       = "TEAM"
		repository = pkg.Repository{
			Name:     "REPOSITORY",
			ReadOnly: true,
		}
	)

	t.Run("Secret template works as intended", func(t *testing.T) {
		template := "/concourse/{{.Team}}/{{.Repository}}"
		expected := "/concourse/TEAM/REPOSITORY"
		actual, err := pkg.NewPath(team, repository, template).String()
		assert.Nil(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("Fails if template expects additional parameters", func(t *testing.T) {
		template := "/concourse/{{.Team}}/{{.Repository}}/{{.Something}}"
		actual, err := pkg.NewPath(team, repository, template).String()
		assert.NotNil(t, err)
		assert.Equal(t, "", actual)
	})
}
