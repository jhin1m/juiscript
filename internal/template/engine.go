package template

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

// Embed all template files into the binary at compile time.
// This means the single binary contains everything it needs.
//
//go:embed templates/*
var templateFS embed.FS

// Engine manages embedded templates for config generation.
type Engine struct {
	templates *template.Template
}

// New parses all embedded templates and returns an Engine.
// Fails fast at startup if any template has syntax errors.
func New() (*Engine, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Engine{templates: tmpl}, nil
}

// Render executes a named template with the given data.
// Returns the rendered string, ready to write to a config file.
func (e *Engine) Render(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := e.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return buf.String(), nil
}

// Available returns a list of all loaded template names.
func (e *Engine) Available() []string {
	var names []string
	for _, t := range e.templates.Templates() {
		names = append(names, t.Name())
	}
	return names
}
