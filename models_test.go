package handler_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/telia-oss/concourse-github-lambda"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		description string
		input       string
		expected    handler.Team
	}{
		{
			description: "Unmarshal works as intended",
			input: strings.TrimSpace(`
{
    "name": "team",
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
`),
			expected: handler.Team{
				Name: "team",
				Repositories: []handler.Repository{
					{
						Name:     "repo1",
						ReadOnly: handler.BooleanString(true),
					},
					{
						Name:     "repo2",
						ReadOnly: handler.BooleanString(false),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			var output handler.Team
			err := json.Unmarshal([]byte(tc.input), &output)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if got, want := output, tc.expected; !reflect.DeepEqual(got, want) {
				t.Errorf("\ngot:\n%v\nwant:\n%v\n", got, want)
			}
		})
	}
}

func TestTemplate(t *testing.T) {
	tests := []struct {
		description string
		template    string
		team        string
		repository  string
		expected    string
		shouldError bool
	}{
		{
			description: "template works as intended",
			template:    "/concourse/{{.Team}}/{{.Repository}}",
			team:        "TEAM",
			repository:  "REPOSITORY",
			shouldError: false,
		},
		{
			description: "fails if the template expects more parameters",
			template:    "/concourse/{{.Team}}/{{.Repository}}/{{.Something}}",
			team:        "TEAM",
			repository:  "REPOSITORY",
			shouldError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			got, err := handler.NewTemplate(tc.team, tc.repository, tc.template).String()

			if tc.shouldError && err == nil {
				t.Fatal("expected an error to occur")
			}

			if !tc.shouldError && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if want := tc.expected; got != want {
				t.Errorf("\ngot:\n%v\nwant:\n%v\n", got, want)
			}
		})
	}
}
