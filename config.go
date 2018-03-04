package main

import (
	"strings"
	"text/template"
)

// Repository represents the configuration of a repository.
type Repository struct {
	Name     string `json:"name"`
	ReadOnly bool   `json:"readOnly"`
}

// Team represents the configuration for a single CI/CD team.
type Team struct {
	Name         string       `json:"name"`
	KeyID        string       `json:"keyId"`
	Repositories []Repository `json:"repositories"`
}

// NewPath a new secret path...
func NewPath(team string, repository Repository, template string) *Path {
	return &Path{
		Team:       team,
		Repository: repository.Name,
		Template:   template,
	}
}

// Path represents the path used to write secrets into SSM.
type Path struct {
	Team       string
	Repository string
	Template   string
}

func (p *Path) String() (string, error) {
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
