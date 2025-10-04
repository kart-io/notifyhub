// Package template provides template engine interface for NotifyHub
package template

import (
	"context"
	"io"
)

// Engine defines the template engine interface
type Engine interface {
	// Render renders a template with the given data
	Render(ctx context.Context, templateName string, data interface{}) (string, error)

	// RenderToWriter renders a template to a writer
	RenderToWriter(ctx context.Context, w io.Writer, templateName string, data interface{}) error

	// Parse parses a template from string
	Parse(templateName, templateContent string) error

	// ParseFile parses a template from file
	ParseFile(templateName, filename string) error

	// Exists checks if a template exists
	Exists(templateName string) bool

	// List returns all available template names
	List() []string

	// Remove removes a template
	Remove(templateName string) error

	// Clear removes all templates
	Clear() error
}

// TemplateData represents template rendering data
type TemplateData struct {
	Variables map[string]interface{} `json:"variables"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// RenderOptions provides options for template rendering
type RenderOptions struct {
	EscapeHTML bool `json:"escape_html"`
	Strict     bool `json:"strict"`
}
