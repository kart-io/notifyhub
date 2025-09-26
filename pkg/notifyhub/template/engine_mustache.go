// Package template provides Mustache template engine implementation
package template

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cbroglie/mustache"
	"github.com/kart-io/notifyhub/pkg/logger"
)

// MustacheEngine implements Engine interface using Mustache templates
type MustacheEngine struct {
	logger logger.Logger
}

// NewMustacheEngine creates a new Mustache template engine
func NewMustacheEngine(logger logger.Logger) *MustacheEngine {
	return &MustacheEngine{
		logger: logger,
	}
}

// Name returns the engine name
func (e *MustacheEngine) Name() string {
	return "mustache"
}

// Render renders template content with variables using Mustache
func (e *MustacheEngine) Render(ctx context.Context, content string, variables map[string]interface{}) (string, error) {
	// Render Mustache template
	result, err := mustache.Render(content, variables)
	if err != nil {
		e.logger.Error("Failed to render Mustache template", "error", err)
		return "", fmt.Errorf("mustache render error: %w", err)
	}

	e.logger.Debug("Mustache template rendered", "length", len(result))

	return result, nil
}

// Validate validates template syntax using Mustache parser
func (e *MustacheEngine) Validate(content string) error {
	// Parse the template to validate syntax
	_, err := mustache.ParseString(content)
	if err != nil {
		e.logger.Debug("Mustache template validation failed", "error", err)
		return fmt.Errorf("invalid Mustache template syntax: %w", err)
	}

	return nil
}

// GetCapabilities returns Mustache template engine capabilities
func (e *MustacheEngine) GetCapabilities() EngineCapabilities {
	return EngineCapabilities{
		Name: "Mustache",
		SupportedFeatures: []string{
			"variables", // {{variable}}
			"sections",  // {{#section}}...{{/section}}
			"inverted",  // {{^inverted}}...{{/inverted}}
			"partials",  // {{>partial}}
			"comments",  // {{! comment }}
			"unescaped", // {{{unescaped}}} or {{&unescaped}}
			"lambdas",   // Function-based sections
		},
		MaxTemplateSize:     1024 * 1024, // 1MB
		SupportsCompilation: true,
		SupportsPartials:    true,  // {{>partial}}
		SupportsFunctions:   false, // Mustache is logic-less
		PerformanceLevel:    "medium",
	}
}

// ExtractVariables extracts variable references from Mustache template content
func (e *MustacheEngine) ExtractVariables(content string) ([]string, error) {
	variables := make([]string, 0)
	seen := make(map[string]bool)

	// Regular expression patterns for Mustache variables
	patterns := []*regexp.Regexp{
		// Simple variables: {{variable}}
		regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
		// Unescaped variables: {{{variable}}}
		regexp.MustCompile(`\{\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}\}`),
		// Unescaped variables: {{&variable}}
		regexp.MustCompile(`\{\{\s*&\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
		// Section start: {{#variable}}
		regexp.MustCompile(`\{\{\s*#\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
		// Inverted section: {{^variable}}
		regexp.MustCompile(`\{\{\s*\^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
	}

	// Extract variables using all patterns
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				variable := match[1]
				if !seen[variable] {
					variables = append(variables, variable)
					seen[variable] = true
				}
			}
		}
	}

	// Extract nested object properties: {{object.property}}
	nestedPattern := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_.]*)\s*\}\}`)
	nestedMatches := nestedPattern.FindAllStringSubmatch(content, -1)

	for _, match := range nestedMatches {
		if len(match) > 1 {
			fullPath := match[1]
			// Extract root variable from nested path
			if parts := strings.Split(fullPath, "."); len(parts) > 0 {
				rootVar := parts[0]
				if !seen[rootVar] && isValidVariableName(rootVar) {
					variables = append(variables, rootVar)
					seen[rootVar] = true
				}
			}
		}
	}

	e.logger.Debug("Extracted Mustache template variables", "count", len(variables), "variables", variables)

	return variables, nil
}

// isValidVariableName checks if a string is a valid variable name
func isValidVariableName(name string) bool {
	// Must start with letter or underscore, followed by letters, numbers, or underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}

// RenderWithPartials renders a Mustache template with partial support
func (e *MustacheEngine) RenderWithPartials(ctx context.Context, content string, variables map[string]interface{}, partials map[string]string) (string, error) {
	// Create a provider for partials
	provider := &mapPartialProvider{partials: partials}

	// Parse template with partial provider
	template, err := mustache.ParseStringPartials(content, provider)
	if err != nil {
		e.logger.Error("Failed to parse Mustache template with partials", "error", err)
		return "", fmt.Errorf("mustache template parse error: %w", err)
	}

	// Render template
	result, err := template.Render(variables)
	if err != nil {
		return "", fmt.Errorf("mustache template render error: %w", err)
	}

	e.logger.Debug("Mustache template with partials rendered", "length", len(result))

	return result, nil
}

// mapPartialProvider implements mustache.PartialProvider interface
type mapPartialProvider struct {
	partials map[string]string
}

// Get implements mustache.PartialProvider
func (p *mapPartialProvider) Get(name string) (string, error) {
	if partial, exists := p.partials[name]; exists {
		return partial, nil
	}
	return "", fmt.Errorf("partial not found: %s", name)
}

// CreateAdvancedContext creates a context with helper functions for Mustache
func (e *MustacheEngine) CreateAdvancedContext(variables map[string]interface{}) map[string]interface{} {
	// Since Mustache is logic-less, we can only provide data transformations
	// through pre-processed variables, not through template functions

	context := make(map[string]interface{})

	// Copy original variables
	for k, v := range variables {
		context[k] = v
	}

	// Add some utility data transformations
	if str, ok := variables["text"].(string); ok {
		context["text_upper"] = strings.ToUpper(str)
		context["text_lower"] = strings.ToLower(str)
		context["text_title"] = func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
		}(str)
	}

	// Add boolean helpers
	for k, v := range variables {
		if v == nil || v == "" || v == 0 || v == false {
			context[k+"_empty"] = true
		} else {
			context[k+"_not_empty"] = true
		}
	}

	return context
}
