package handler

import (
	"fmt"
	"strconv"
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
	Name     string        `json:"name"`
	Owner    string        `json:"owner"`
	ReadOnly BooleanString `json:"readOnly"`
}

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

// NewTemplate for github key title and secrets manager path.
func NewTemplate(team, repository, owner, template string) *Template {
	return &Template{
		Team:       team,
		Owner:      owner,
		Repository: repository,
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
