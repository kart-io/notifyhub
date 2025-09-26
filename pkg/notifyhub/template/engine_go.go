// Package template provides Go template engine implementation
package template

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// GoEngine implements Engine interface using Go's text/template
type GoEngine struct {
	logger logger.Logger
}

// NewGoEngine creates a new Go template engine
func NewGoEngine(logger logger.Logger) *GoEngine {
	return &GoEngine{
		logger: logger,
	}
}

// Name returns the engine name
func (e *GoEngine) Name() string {
	return "go"
}

// Render renders template content with variables using Go templates
func (e *GoEngine) Render(ctx context.Context, content string, variables map[string]interface{}) (string, error) {
	// Create new template
	tmpl, err := template.New("template").Parse(content)
	if err != nil {
		e.logger.Error("Failed to parse Go template", "error", err)
		return "", fmt.Errorf("template parse error: %w", err)
	}

	// Render template with variables
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		e.logger.Error("Failed to execute Go template", "error", err)
		return "", fmt.Errorf("template execution error: %w", err)
	}

	result := buf.String()
	e.logger.Debug("Go template rendered", "length", len(result))

	return result, nil
}

// Validate validates template syntax using Go template parser
func (e *GoEngine) Validate(content string) error {
	_, err := template.New("validation").Parse(content)
	if err != nil {
		e.logger.Debug("Go template validation failed", "error", err)
		return fmt.Errorf("invalid Go template syntax: %w", err)
	}

	return nil
}

// GetCapabilities returns Go template engine capabilities
func (e *GoEngine) GetCapabilities() EngineCapabilities {
	return EngineCapabilities{
		Name: "Go text/template",
		SupportedFeatures: []string{
			"conditionals", // {{if .condition}}...{{end}}
			"loops",        // {{range .items}}...{{end}}
			"variables",    // {{.variable}}
			"functions",    // {{function .arg}}
			"pipelines",    // {{.value | function}}
			"comments",     // {{/* comment */}}
			"with_blocks",  // {{with .value}}...{{end}}
		},
		MaxTemplateSize:     1024 * 1024, // 1MB
		SupportsCompilation: true,
		SupportsPartials:    true, // {{template "partial" .}}
		SupportsFunctions:   true, // Custom functions via FuncMap
		PerformanceLevel:    "high",
	}
}

// ExtractVariables extracts variable references from Go template content
func (e *GoEngine) ExtractVariables(content string) ([]string, error) {
	// Regular expression to match Go template variables: {{.Variable}}
	// This is a simplified implementation - real implementation would parse AST
	varRegex := regexp.MustCompile(`\{\{\s*\.(\w+)\s*\}\}`)

	matches := varRegex.FindAllStringSubmatch(content, -1)
	variables := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			variable := match[1]
			if !seen[variable] {
				variables = append(variables, variable)
				seen[variable] = true
			}
		}
	}

	// Also look for range variables: {{range .Items}}
	rangeRegex := regexp.MustCompile(`\{\{\s*range\s+\.(\w+)\s*\}\}`)
	rangeMatches := rangeRegex.FindAllStringSubmatch(content, -1)

	for _, match := range rangeMatches {
		if len(match) > 1 {
			variable := match[1]
			if !seen[variable] {
				variables = append(variables, variable)
				seen[variable] = true
			}
		}
	}

	// Look for with variables: {{with .Value}}
	withRegex := regexp.MustCompile(`\{\{\s*with\s+\.(\w+)\s*\}\}`)
	withMatches := withRegex.FindAllStringSubmatch(content, -1)

	for _, match := range withMatches {
		if len(match) > 1 {
			variable := match[1]
			if !seen[variable] {
				variables = append(variables, variable)
				seen[variable] = true
			}
		}
	}

	e.logger.Debug("Extracted Go template variables", "count", len(variables), "variables", variables)

	return variables, nil
}

// CreateWithFunctions creates a Go template with custom functions
func (e *GoEngine) CreateWithFunctions(content string, funcMap template.FuncMap) (*template.Template, error) {
	tmpl := template.New("template")

	if funcMap != nil {
		tmpl = tmpl.Funcs(funcMap)
	}

	return tmpl.Parse(content)
}

// DefaultFunctions returns default template functions for Go templates
func (e *GoEngine) DefaultFunctions() template.FuncMap {
	return template.FuncMap{
		// String functions
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
		},
		"trim": strings.TrimSpace,

		// Utility functions
		"default": func(defaultValue interface{}, value interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},

		// Math functions
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},

		// Comparison functions
		"eq":  func(a, b interface{}) bool { return a == b },
		"ne":  func(a, b interface{}) bool { return a != b },
		"gt":  func(a, b int) bool { return a > b },
		"lt":  func(a, b int) bool { return a < b },
		"gte": func(a, b int) bool { return a >= b },
		"lte": func(a, b int) bool { return a <= b },

		// Array/slice functions
		"length": func(v interface{}) int {
			switch val := v.(type) {
			case []interface{}:
				return len(val)
			case string:
				return len(val)
			default:
				return 0
			}
		},

		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},
	}
}
