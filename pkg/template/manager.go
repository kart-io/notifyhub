// Package template provides template management functionality
package template

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"github.com/kart/notifyhub/pkg/utils/logger"
)

// Manager manages templates with caching
type Manager struct {
	engine Engine
	cache  Cache
	logger logger.Logger
	config ManagerConfig
}

// ManagerConfig configures the template manager
type ManagerConfig struct {
	EnableCache bool          `json:"enable_cache"`
	CacheTTL    time.Duration `json:"cache_ttl"`
	EscapeHTML  bool          `json:"escape_html"`
	Strict      bool          `json:"strict"`
}

// NewManager creates a new template manager
func NewManager(config ManagerConfig, logger logger.Logger) *Manager {
	var cache Cache
	if config.EnableCache {
		cache = NewMemoryCache()
	}

	return &Manager{
		engine: NewTextEngine(),
		cache:  cache,
		logger: logger,
		config: config,
	}
}

// Render renders a template with data
func (m *Manager) Render(ctx context.Context, templateName string, data interface{}) (string, error) {
	// Check cache first
	if m.cache != nil {
		if cached, exists := m.cache.Get(templateName); exists {
			if tmpl, ok := cached.(*template.Template); ok {
				return m.executeTemplate(tmpl, data)
			}
		}
	}

	// Render using engine
	result, err := m.engine.Render(ctx, templateName, data)
	if err != nil {
		m.logger.Error("Failed to render template", "template", templateName, "error", err)
		return "", err
	}

	return result, nil
}

// RenderToWriter renders a template to a writer
func (m *Manager) RenderToWriter(ctx context.Context, w io.Writer, templateName string, data interface{}) error {
	return m.engine.RenderToWriter(ctx, w, templateName, data)
}

// RegisterTemplate registers a template
func (m *Manager) RegisterTemplate(name, content string) error {
	err := m.engine.Parse(name, content)
	if err != nil {
		m.logger.Error("Failed to register template", "name", name, "error", err)
		return err
	}

	m.logger.Debug("Template registered", "name", name)
	return nil
}

// RegisterTemplateFile registers a template from file
func (m *Manager) RegisterTemplateFile(name, filename string) error {
	err := m.engine.ParseFile(name, filename)
	if err != nil {
		m.logger.Error("Failed to register template file", "name", name, "file", filename, "error", err)
		return err
	}

	m.logger.Debug("Template file registered", "name", name, "file", filename)
	return nil
}

// ListTemplates returns all available templates
func (m *Manager) ListTemplates() []string {
	return m.engine.List()
}

// TemplateExists checks if a template exists
func (m *Manager) TemplateExists(name string) bool {
	return m.engine.Exists(name)
}

// RemoveTemplate removes a template
func (m *Manager) RemoveTemplate(name string) error {
	err := m.engine.Remove(name)
	if err != nil {
		m.logger.Error("Failed to remove template", "name", name, "error", err)
		return err
	}

	// Remove from cache if exists
	if m.cache != nil {
		m.cache.Delete(name)
	}

	m.logger.Debug("Template removed", "name", name)
	return nil
}

// ClearCache clears the template cache
func (m *Manager) ClearCache() {
	if m.cache != nil {
		m.cache.Clear()
		m.logger.Debug("Template cache cleared")
	}
}

// GetCacheStats returns cache statistics
func (m *Manager) GetCacheStats() map[string]interface{} {
	if m.cache == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	return map[string]interface{}{
		"enabled": true,
		"size":    m.cache.Size(),
		"ttl":     m.config.CacheTTL,
	}
}

// executeTemplate executes a parsed template
func (m *Manager) executeTemplate(tmpl *template.Template, data interface{}) (string, error) {
	// Implementation would depend on the specific template engine
	return "", fmt.Errorf("template execution not implemented")
}

// TextEngine is a simple text template engine
type TextEngine struct {
	templates map[string]*template.Template
}

// NewTextEngine creates a new text template engine
func NewTextEngine() *TextEngine {
	return &TextEngine{
		templates: make(map[string]*template.Template),
	}
}

// Render renders a template
func (e *TextEngine) Render(ctx context.Context, templateName string, data interface{}) (string, error) {
	tmpl, exists := e.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var result strings.Builder
	// Execute template
	err := tmpl.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return result.String(), nil
}

// RenderToWriter renders to writer
func (e *TextEngine) RenderToWriter(ctx context.Context, w io.Writer, templateName string, data interface{}) error {
	tmpl, exists := e.templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	return tmpl.Execute(w, data)
}

// Parse parses a template
func (e *TextEngine) Parse(templateName, templateContent string) error {
	tmpl, err := template.New(templateName).Parse(templateContent)
	if err != nil {
		return err
	}
	e.templates[templateName] = tmpl
	return nil
}

// ParseFile parses from file
func (e *TextEngine) ParseFile(templateName, filename string) error {
	tmpl, err := template.ParseFiles(filename)
	if err != nil {
		return err
	}
	e.templates[templateName] = tmpl
	return nil
}

// Exists checks if template exists
func (e *TextEngine) Exists(templateName string) bool {
	_, exists := e.templates[templateName]
	return exists
}

// List returns all template names
func (e *TextEngine) List() []string {
	names := make([]string, 0, len(e.templates))
	for name := range e.templates {
		names = append(names, name)
	}
	return names
}

// Remove removes a template
func (e *TextEngine) Remove(templateName string) error {
	delete(e.templates, templateName)
	return nil
}

// Clear removes all templates
func (e *TextEngine) Clear() error {
	e.templates = make(map[string]*template.Template)
	return nil
}
