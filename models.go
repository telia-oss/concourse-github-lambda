package handler

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

// BooleanString because terraform has their own booleans.
type BooleanString bool

// UnmarshalJSON for terraform booleans..
func (b *BooleanString) UnmarshalJSON(data []byte) error {
	d, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("failed to unquote: %s", err)
	}
	v, err := strconv.ParseBool(d)
	if err != nil {
		return fmt.Errorf("failed to parse bool: %s", err)
	}
	*b = BooleanString(v)
	return nil
}

// Repository represents the configuration of a repository.
type Repository struct {
	Name     string        `json:"name"`
	ReadOnly BooleanString `json:"readOnly"`
}

// Team represents the configuration for a single CI/CD team.
type Team struct {
	Name         string       `json:"name"`
	Repositories []Repository `json:"repositories"`
}

// NewTemplate for github key title and secrets manager path.
func NewTemplate(team, repository, template string) *Template {
	return &Template{
		Team:       team,
		Repository: repository,
		Template:   template,
	}
}

// Template ...
type Template struct {
	Team       string
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
