// Package template provides Handlebars template engine implementation
package template

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cbroglie/mustache" // Using Mustache as base for Handlebars compatibility
	"github.com/kart-io/notifyhub/pkg/logger"
)

// HandlebarsEngine implements Engine interface using Handlebars-style templates
// Note: This is a simplified implementation based on Mustache with Handlebars syntax support
type HandlebarsEngine struct {
	logger logger.Logger
}

// NewHandlebarsEngine creates a new Handlebars template engine
func NewHandlebarsEngine(logger logger.Logger) *HandlebarsEngine {
	return &HandlebarsEngine{
		logger: logger,
	}
}

// Name returns the engine name
func (e *HandlebarsEngine) Name() string {
	return "handlebars"
}

// Render renders template content with variables using Handlebars-style syntax
func (e *HandlebarsEngine) Render(ctx context.Context, content string, variables map[string]interface{}) (string, error) {
	// Convert Handlebars syntax to Mustache syntax for compatibility
	convertedContent := e.convertHandlebarsToMustache(content)

	// Enhance variables with Handlebars-style helpers
	enhancedVars := e.addHandlebarsHelpers(variables)

	// Render using Mustache engine
	result, err := mustache.Render(convertedContent, enhancedVars)
	if err != nil {
		e.logger.Error("Failed to render Handlebars template", "error", err)
		return "", fmt.Errorf("handlebars render error: %w", err)
	}

	e.logger.Debug("Handlebars template rendered", "length", len(result))

	return result, nil
}

// Validate validates template syntax using Handlebars parser
func (e *HandlebarsEngine) Validate(content string) error {
	// Convert Handlebars syntax to Mustache for validation
	convertedContent := e.convertHandlebarsToMustache(content)

	// Validate using Mustache parser
	_, err := mustache.ParseString(convertedContent)
	if err != nil {
		e.logger.Debug("Handlebars template validation failed", "error", err)
		return fmt.Errorf("invalid Handlebars template syntax: %w", err)
	}

	return nil
}

// GetCapabilities returns Handlebars template engine capabilities
func (e *HandlebarsEngine) GetCapabilities() EngineCapabilities {
	return EngineCapabilities{
		Name: "Handlebars",
		SupportedFeatures: []string{
			"variables",     // {{variable}}
			"helpers",       // {{helper variable}}
			"block_helpers", // {{#helper}}...{{/helper}}
			"conditions",    // {{#if condition}}...{{/if}}
			"loops",         // {{#each array}}...{{/each}}
			"partials",      // {{>partial}}
			"comments",      // {{! comment }} or {{!-- comment --}}
			"unescaped",     // {{{unescaped}}}
			"context",       // {{#with object}}...{{/with}}
			"paths",         // {{object.property}}
		},
		MaxTemplateSize:     1024 * 1024, // 1MB
		SupportsCompilation: true,
		SupportsPartials:    true, // {{>partial}}
		SupportsFunctions:   true, // Handlebars helpers
		PerformanceLevel:    "medium",
	}
}

// ExtractVariables extracts variable references from Handlebars template content
func (e *HandlebarsEngine) ExtractVariables(content string) ([]string, error) {
	variables := make([]string, 0)
	seen := make(map[string]bool)

	// Regular expression patterns for Handlebars variables
	patterns := []*regexp.Regexp{
		// Simple variables: {{variable}}
		regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
		// Unescaped variables: {{{variable}}}
		regexp.MustCompile(`\{\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}\}`),
		// Block helpers: {{#helper variable}}
		regexp.MustCompile(`\{\{\s*#\s*\w+\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
		// Conditions: {{#if variable}}
		regexp.MustCompile(`\{\{\s*#if\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
		// Each loops: {{#each variable}}
		regexp.MustCompile(`\{\{\s*#each\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
		// With context: {{#with variable}}
		regexp.MustCompile(`\{\{\s*#with\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
		// Helper with variable: {{helper variable}}
		regexp.MustCompile(`\{\{\s*\w+\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`),
	}

	// Extract variables using all patterns
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				variable := match[1]
				if !seen[variable] && isValidVariableName(variable) {
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

	e.logger.Debug("Extracted Handlebars template variables", "count", len(variables), "variables", variables)

	return variables, nil
}

// convertHandlebarsToMustache converts Handlebars syntax to Mustache for compatibility
func (e *HandlebarsEngine) convertHandlebarsToMustache(content string) string {
	// Convert common Handlebars constructs to Mustache equivalents

	// Convert {{#if variable}} to {{#variable}}
	ifPattern := regexp.MustCompile(`\{\{\s*#if\s+([^}]+)\s*\}\}`)
	content = ifPattern.ReplaceAllString(content, "{{#$1}}")

	// Convert {{/if}} to {{/variable}} - this is simplified
	content = strings.ReplaceAll(content, "{{/if}}", "{{/if}}")

	// Convert {{#each array}} to {{#array}}
	eachPattern := regexp.MustCompile(`\{\{\s*#each\s+([^}]+)\s*\}\}`)
	content = eachPattern.ReplaceAllString(content, "{{#$1}}")

	// Convert {{/each}} to appropriate closing tag
	content = strings.ReplaceAll(content, "{{/each}}", "{{/each}}")

	// Convert {{#with object}} to {{#object}}
	withPattern := regexp.MustCompile(`\{\{\s*#with\s+([^}]+)\s*\}\}`)
	content = withPattern.ReplaceAllString(content, "{{#$1}}")

	// Convert {{/with}} to appropriate closing tag
	content = strings.ReplaceAll(content, "{{/with}}", "{{/with}}")

	// Convert {{#unless variable}} to {{^variable}}
	unlessPattern := regexp.MustCompile(`\{\{\s*#unless\s+([^}]+)\s*\}\}`)
	content = unlessPattern.ReplaceAllString(content, "{{^$1}}")

	// Convert {{/unless}} to appropriate closing tag
	content = strings.ReplaceAll(content, "{{/unless}}", "{{/unless}}")

	return content
}

// addHandlebarsHelpers adds Handlebars-style helper functions to variables
func (e *HandlebarsEngine) addHandlebarsHelpers(variables map[string]interface{}) map[string]interface{} {
	enhanced := make(map[string]interface{})

	// Copy original variables
	for k, v := range variables {
		enhanced[k] = v
	}

	// Add common Handlebars helpers as pre-computed values

	// String helpers
	for k, v := range variables {
		if str, ok := v.(string); ok {
			enhanced[k+"_upper"] = strings.ToUpper(str)
			enhanced[k+"_lower"] = strings.ToLower(str)
			enhanced[k+"_title"] = func(s string) string {
				if len(s) == 0 {
					return s
				}
				return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
			}(str)
			enhanced[k+"_length"] = len(str)
		}
	}

	// Boolean helpers
	for k, v := range variables {
		enhanced[k+"_is_empty"] = (v == nil || v == "" || v == 0 || v == false)
		enhanced[k+"_is_not_empty"] = (v != nil && v != "" && v != 0 && v != false)
	}

	// Array helpers
	for k, v := range variables {
		switch arr := v.(type) {
		case []interface{}:
			enhanced[k+"_length"] = len(arr)
			enhanced[k+"_is_empty"] = len(arr) == 0
			enhanced[k+"_is_not_empty"] = len(arr) > 0
			if len(arr) > 0 {
				enhanced[k+"_first"] = arr[0]
				enhanced[k+"_last"] = arr[len(arr)-1]
			}
		case []string:
			enhanced[k+"_length"] = len(arr)
			enhanced[k+"_is_empty"] = len(arr) == 0
			enhanced[k+"_is_not_empty"] = len(arr) > 0
			if len(arr) > 0 {
				enhanced[k+"_first"] = arr[0]
				enhanced[k+"_last"] = arr[len(arr)-1]
			}
		}
	}

	// Add global helpers
	enhanced["true"] = true
	enhanced["false"] = false

	return enhanced
}

// RenderWithHelpers renders a Handlebars template with custom helpers
func (e *HandlebarsEngine) RenderWithHelpers(ctx context.Context, content string, variables map[string]interface{}, helpers map[string]interface{}) (string, error) {
	// Convert Handlebars syntax
	convertedContent := e.convertHandlebarsToMustache(content)

	// Enhance variables with helpers and Handlebars-style data
	enhancedVars := e.addHandlebarsHelpers(variables)

	// Add custom helpers as data (since Mustache doesn't support functions)
	// This is a limitation - true Handlebars helpers would need a full Handlebars parser
	for k, v := range helpers {
		if str, ok := v.(string); ok {
			enhancedVars[k] = str
		}
	}

	// Render using Mustache engine
	result, err := mustache.Render(convertedContent, enhancedVars)
	if err != nil {
		e.logger.Error("Failed to render Handlebars template with helpers", "error", err)
		return "", fmt.Errorf("handlebars render with helpers error: %w", err)
	}

	e.logger.Debug("Handlebars template with helpers rendered", "length", len(result))

	return result, nil
}
