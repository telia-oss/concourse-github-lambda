package main_test

import (
	"encoding/json"
	pkg "github.com/itsdalmo/concourse-github-credentials"
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
        "repo1",
        "repo2"
    ]
}
`)

	t.Run("Unmarshal works as intended", func(t *testing.T) {
		expected := pkg.Team{
			Name:  "team",
			KeyID: "key",
			Repositories: []string{
				"repo1",
				"repo2",
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
		repository = "REPOSITORY"
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
