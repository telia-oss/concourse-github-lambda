package handler

import (
	"strings"
	"text/template"
)

// Team represents the configuration for a single CI/CD team.
type Team struct {
	Name         string       `json:"name"`
	Repositories []Repository `json:"repositories"`
}

// Repository represents the configuration of a repository.
type Repository struct {
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	ReadOnly bool   `json:"readOnly"`
}

// NewTemplate for github key title and secrets manager path.
func NewTemplate(team, repository, owner, template string) *Template {
	return &Template{
		Team:  team,
		Owner: owner,
		// sanitise the secrets manager path as concourse treats dots as delimiters
		Repository: strings.ReplaceAll(repository, ".", "-"),
		Template:   template,
	}
}

// Template ...
type Template struct {
	Team       string
	Owner      string
	Repository string
	Template   string
}

func (p *Template) String() (string, error) {
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
