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
