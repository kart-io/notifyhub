package template

import (
	"bytes"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"net/http"
	"strings"
	textTemplate "text/template"
	"time"
	"unicode"

	"github.com/kart-io/notifyhub/notifiers"
)

// ================================
// 模板系统
// ================================

// Engine handles message template rendering
type Engine struct {
	textTemplates map[string]*textTemplate.Template
	htmlTemplates map[string]*htmlTemplate.Template
	funcMap       textTemplate.FuncMap
}

// NewEngine creates a new template engine
func NewEngine() *Engine {
	engine := &Engine{
		textTemplates: make(map[string]*textTemplate.Template),
		htmlTemplates: make(map[string]*htmlTemplate.Template),
		funcMap:       createFuncMap(),
	}

	// Load built-in templates
	engine.loadBuiltinTemplates()
	return engine
}

// title implements title case conversion to replace deprecated strings.Title
func title(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	result := make([]rune, len(runes))
	inWord := false

	for i, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				result[i] = unicode.ToUpper(r)
				inWord = true
			} else {
				result[i] = unicode.ToLower(r)
			}
		} else {
			result[i] = r
			inWord = false
		}
	}

	return string(result)
}

// createFuncMap creates template functions
func createFuncMap() textTemplate.FuncMap {
	return textTemplate.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": title,
		"trim":  strings.TrimSpace,
		"now":   time.Now,
		"formatTime": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"default": func(defaultValue, value interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"replace": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"repeat": strings.Repeat,
	}
}

// AddTextTemplate adds a text template
func (e *Engine) AddTextTemplate(name, templateText string) error {
	tmpl, err := textTemplate.New(name).Funcs(e.funcMap).Parse(templateText)
	if err != nil {
		return fmt.Errorf("parse text template %s: %v", name, err)
	}
	e.textTemplates[name] = tmpl
	return nil
}

// AddHTMLTemplate adds an HTML template
func (e *Engine) AddHTMLTemplate(name, templateText string) error {
	// Convert funcMap for HTML template
	htmlFuncMap := htmlTemplate.FuncMap{}
	for k, v := range e.funcMap {
		htmlFuncMap[k] = v
	}

	tmpl, err := htmlTemplate.New(name).Funcs(htmlFuncMap).Parse(templateText)
	if err != nil {
		return fmt.Errorf("parse HTML template %s: %v", name, err)
	}
	e.htmlTemplates[name] = tmpl
	return nil
}

// RenderMessage renders a message using templates
func (e *Engine) RenderMessage(message *notifiers.Message) (*notifiers.Message, error) {
	if message.Template == "" && !strings.Contains(message.Title+message.Body, "{{") {
		return message, nil // No template to render
	}

	// Create a copy of the message
	rendered := *message

	// Prepare template data
	data := e.createTemplateData(message)

	// Render title if it contains template syntax
	if strings.Contains(message.Title, "{{") {
		title, err := e.renderString(message.Title, data, message.Format)
		if err != nil {
			return nil, fmt.Errorf("render title: %v", err)
		}
		rendered.Title = title
	}

	// Render body
	if message.Template != "" {
		body, err := e.renderTemplate(message.Template, data, message.Format)
		if err != nil {
			return nil, fmt.Errorf("render template %s: %v", message.Template, err)
		}
		rendered.Body = body
	} else if strings.Contains(message.Body, "{{") {
		body, err := e.renderString(message.Body, data, message.Format)
		if err != nil {
			return nil, fmt.Errorf("render body: %v", err)
		}
		rendered.Body = body
	}

	return &rendered, nil
}

// renderTemplate renders a named template
func (e *Engine) renderTemplate(name string, data interface{}, format notifiers.MessageFormat) (string, error) {
	var buf bytes.Buffer

	switch format {
	case notifiers.FormatHTML:
		tmpl, exists := e.htmlTemplates[name]
		if !exists {
			return "", fmt.Errorf("HTML template %s not found", name)
		}
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}

	default: // FormatText, FormatMarkdown
		tmpl, exists := e.textTemplates[name]
		if !exists {
			return "", fmt.Errorf("text template %s not found", name)
		}
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

// renderString renders an inline template string
func (e *Engine) renderString(templateStr string, data interface{}, format notifiers.MessageFormat) (string, error) {
	var buf bytes.Buffer

	switch format {
	case notifiers.FormatHTML:
		// Convert funcMap for HTML template
		htmlFuncMap := htmlTemplate.FuncMap{}
		for k, v := range e.funcMap {
			htmlFuncMap[k] = v
		}

		tmpl, err := htmlTemplate.New("inline").Funcs(htmlFuncMap).Parse(templateStr)
		if err != nil {
			return "", err
		}
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}

	default: // FormatText, FormatMarkdown
		tmpl, err := textTemplate.New("inline").Funcs(e.funcMap).Parse(templateStr)
		if err != nil {
			return "", err
		}
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

// createTemplateData creates template data from message
func (e *Engine) createTemplateData(message *notifiers.Message) map[string]interface{} {
	data := map[string]interface{}{
		"Message":   message,
		"Title":     message.Title,
		"Body":      message.Body,
		"Priority":  message.Priority,
		"Format":    string(message.Format),
		"Variables": message.Variables,
		"Metadata":  message.Metadata,
		"CreatedAt": message.CreatedAt,
		"Now":       time.Now(),
	}

	// Flatten variables to top level for easier access
	for k, v := range message.Variables {
		data[k] = v
	}

	return data
}

// loadBuiltinTemplates loads built-in templates
func (e *Engine) loadBuiltinTemplates() {
	// Alert template
	alertText := `🚨 ALERT: {{.Title}}

{{.Body}}

{{if .Variables.server}}Server: {{.Variables.server}}{{end}}
{{if .Variables.environment}}Environment: {{.Variables.environment | upper}}{{end}}
{{if .Variables.error}}Error: {{.Variables.error}}{{end}}

Time: {{.CreatedAt.Format "2006-01-02 15:04:05"}}

---
This is an automated alert from NotifyHub`

	_ = e.AddTextTemplate("alert", alertText)

	// Notice template
	noticeText := `📢 {{.Title}}

{{.Body}}

{{range $key, $value := .Variables}}
{{$key}}: {{$value}}
{{end}}

Sent at {{.CreatedAt.Format "2006-01-02 15:04:05"}}`

	_ = e.AddTextTemplate("notice", noticeText)

	// Report template
	reportText := `📊 {{.Title | upper}}

{{.Body}}

{{if .Variables.metrics}}
Metrics:
{{range $name, $value := .Variables.metrics}}
- {{$name}}: {{$value}}
{{end}}
{{end}}

{{if .Variables.summary}}
Summary: {{.Variables.summary}}
{{end}}

Generated on {{.CreatedAt.Format "2006-01-02 15:04:05"}}`

	_ = e.AddTextTemplate("report", reportText)

	// HTML Alert template
	alertHTML := `<!DOCTYPE html>
<html>
<head>
    <style>
        .alert { color: #d32f2f; font-weight: bold; }
        .info { color: #1976d2; }
        .metadata { background-color: #f5f5f5; padding: 10px; margin: 10px 0; }
    </style>
</head>
<body>
    <h2 class="alert">🚨 ALERT: {{.Title}}</h2>
    <p>{{.Body}}</p>

    {{if .Variables}}
    <div class="metadata">
        <h3>Details:</h3>
        <ul>
        {{range $key, $value := .Variables}}
            <li><strong>{{$key}}:</strong> {{$value}}</li>
        {{end}}
        </ul>
    </div>
    {{end}}

    <p class="info">
        <small>Sent at {{.CreatedAt.Format "2006-01-02 15:04:05"}} by NotifyHub</small>
    </p>
</body>
</html>`

	_ = e.AddHTMLTemplate("alert", alertHTML)
}

// GetAvailableTemplates returns list of available templates
func (e *Engine) GetAvailableTemplates() []string {
	var templates []string

	for name := range e.textTemplates {
		templates = append(templates, name+" (text)")
	}

	for name := range e.htmlTemplates {
		templates = append(templates, name+" (html)")
	}

	return templates
}

// ValidateTemplate validates a template string
func (e *Engine) ValidateTemplate(templateStr string, format notifiers.MessageFormat) error {
	switch format {
	case notifiers.FormatHTML:
		htmlFuncMap := htmlTemplate.FuncMap{}
		for k, v := range e.funcMap {
			htmlFuncMap[k] = v
		}
		_, err := htmlTemplate.New("validation").Funcs(htmlFuncMap).Parse(templateStr)
		return err
	default:
		_, err := textTemplate.New("validation").Funcs(e.funcMap).Parse(templateStr)
		return err
	}
}

// LoadTemplateFromURL loads a template from a remote HTTP URL
func (e *Engine) LoadTemplateFromURL(name, url string, format notifiers.MessageFormat, headers map[string]string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %v", err)
	}

	// Add custom headers if provided
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch template: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %v", err)
	}

	// Add template based on format
	switch format {
	case notifiers.FormatHTML:
		return e.AddHTMLTemplate(name, string(content))
	default:
		return e.AddTextTemplate(name, string(content))
	}
}
