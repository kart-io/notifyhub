package template

import (
	"strings"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/notifiers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.textTemplates)
	assert.NotNil(t, engine.htmlTemplates)
	assert.NotNil(t, engine.funcMap)
}

func TestEngineBuiltinTemplates(t *testing.T) {
	engine := NewEngine()

	// Test that built-in templates are loaded
	templates := engine.GetAvailableTemplates()
	assert.True(t, len(templates) > 0)

	// Check for specific built-in templates
	foundAlert := false
	foundNotice := false
	foundReport := false

	for _, tmpl := range templates {
		if strings.Contains(tmpl, "alert") {
			foundAlert = true
		}
		if strings.Contains(tmpl, "notice") {
			foundNotice = true
		}
		if strings.Contains(tmpl, "report") {
			foundReport = true
		}
	}

	assert.True(t, foundAlert, "Should have alert template")
	assert.True(t, foundNotice, "Should have notice template")
	assert.True(t, foundReport, "Should have report template")
}

func TestAddTextTemplate(t *testing.T) {
	engine := NewEngine()

	templateText := `Hello {{.Name}}, your score is {{.Score}}`
	err := engine.AddTextTemplate("greeting", templateText)
	assert.NoError(t, err)

	// Test invalid template
	invalidTemplate := `Hello {{.Name`
	err = engine.AddTextTemplate("invalid", invalidTemplate)
	assert.Error(t, err)
}

func TestAddHTMLTemplate(t *testing.T) {
	engine := NewEngine()

	templateHTML := `<h1>Hello {{.Name}}</h1><p>Score: {{.Score}}</p>`
	err := engine.AddHTMLTemplate("greeting_html", templateHTML)
	assert.NoError(t, err)

	// Test invalid template
	invalidTemplate := `<h1>Hello {{.Name</h1>`
	err = engine.AddHTMLTemplate("invalid_html", invalidTemplate)
	assert.Error(t, err)
}

func TestRenderMessageNoTemplate(t *testing.T) {
	engine := NewEngine()

	// Message without template syntax should pass through unchanged
	message := &notifiers.Message{
		Title: "Simple Title",
		Body:  "Simple body without templates",
	}

	rendered, err := engine.RenderMessage(message)
	assert.NoError(t, err)
	assert.Equal(t, message.Title, rendered.Title)
	assert.Equal(t, message.Body, rendered.Body)
}

func TestRenderMessageWithInlineTemplate(t *testing.T) {
	engine := NewEngine()

	message := &notifiers.Message{
		Title: "Hello {{.Name}}",
		Body:  "Your score is {{.Score}}",
		Variables: map[string]interface{}{
			"Name":  "John",
			"Score": 95,
		},
	}

	rendered, err := engine.RenderMessage(message)
	assert.NoError(t, err)
	assert.Equal(t, "Hello John", rendered.Title)
	assert.Equal(t, "Your score is 95", rendered.Body)
}

func TestRenderMessageWithNamedTemplate(t *testing.T) {
	engine := NewEngine()

	// Add a custom template
	templateText := `Hello {{.Name}}! Your score is {{.Score}}. {{if gt .Score 90}}Excellent work!{{else}}Keep trying!{{end}}`
	err := engine.AddTextTemplate("score_report", templateText)
	require.NoError(t, err)

	message := &notifiers.Message{
		Template: "score_report",
		Variables: map[string]interface{}{
			"Name":  "Alice",
			"Score": 95,
		},
	}

	rendered, err := engine.RenderMessage(message)
	assert.NoError(t, err)
	assert.Contains(t, rendered.Body, "Hello Alice")
	assert.Contains(t, rendered.Body, "score is 95")
	assert.Contains(t, rendered.Body, "Excellent work")
}

func TestRenderMessageWithHTMLTemplate(t *testing.T) {
	engine := NewEngine()

	// Add HTML template
	templateHTML := `<h1>Hello {{.Name}}</h1><p>Score: <strong>{{.Score}}</strong></p>`
	err := engine.AddHTMLTemplate("html_report", templateHTML)
	require.NoError(t, err)

	message := &notifiers.Message{
		Template: "html_report",
		Format:   notifiers.FormatHTML,
		Variables: map[string]interface{}{
			"Name":  "Bob",
			"Score": 88,
		},
	}

	rendered, err := engine.RenderMessage(message)
	assert.NoError(t, err)
	assert.Contains(t, rendered.Body, "<h1>Hello Bob</h1>")
	assert.Contains(t, rendered.Body, "<strong>88</strong>")
}

func TestRenderMessageWithBuiltinFunctions(t *testing.T) {
	engine := NewEngine()

	message := &notifiers.Message{
		Title: "{{.Name | upper}}",
		Body:  "Message: {{.Message | lower}} - Time: {{formatTime .Now \"2006-01-02\"}}",
		Variables: map[string]interface{}{
			"Name":    "john doe",
			"Message": "HELLO WORLD",
			"Now":     time.Date(2023, 5, 15, 10, 30, 0, 0, time.UTC),
		},
	}

	rendered, err := engine.RenderMessage(message)
	assert.NoError(t, err)
	assert.Equal(t, "JOHN DOE", rendered.Title)
	assert.Contains(t, rendered.Body, "hello world")
	assert.Contains(t, rendered.Body, "2023-05-15")
}

func TestRenderMessageWithDefaultFunction(t *testing.T) {
	engine := NewEngine()

	message := &notifiers.Message{
		Body: "Name: {{.Name | default \"Anonymous\"}} - Email: {{.Email | default \"no-email@example.com\"}}",
		Variables: map[string]interface{}{
			"Name": "John",
			// Email is intentionally missing
		},
	}

	rendered, err := engine.RenderMessage(message)
	assert.NoError(t, err)
	assert.Contains(t, rendered.Body, "Name: John")
	assert.Contains(t, rendered.Body, "Email: no-email@example.com")
}

func TestRenderMessageErrors(t *testing.T) {
	engine := NewEngine()

	// Test missing template
	message := &notifiers.Message{
		Template: "nonexistent",
	}

	_, err := engine.RenderMessage(message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test invalid inline template
	message = &notifiers.Message{
		Body: "Hello {{.Name",
	}

	_, err = engine.RenderMessage(message)
	assert.Error(t, err)
}

func TestCreateTemplateData(t *testing.T) {
	engine := NewEngine()

	now := time.Now()
	message := &notifiers.Message{
		Title:     "Test Title",
		Body:      "Test Body",
		Format:    notifiers.FormatText,
		CreatedAt: now,
		Variables: map[string]interface{}{
			"user":  "john",
			"count": 42,
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	data := engine.createTemplateData(message)

	assert.Equal(t, message, data["Message"])
	assert.Equal(t, "Test Title", data["Title"])
	assert.Equal(t, "Test Body", data["Body"])
	assert.Equal(t, "text", data["Format"])
	assert.Equal(t, now, data["CreatedAt"])
	assert.Equal(t, message.Variables, data["Variables"])
	assert.Equal(t, message.Metadata, data["Metadata"])

	// Variables should be flattened to top level
	assert.Equal(t, "john", data["user"])
	assert.Equal(t, 42, data["count"])

	// Should have "Now" field
	assert.IsType(t, time.Time{}, data["Now"])
}

func TestValidateTemplate(t *testing.T) {
	engine := NewEngine()

	// Valid text template
	err := engine.ValidateTemplate("Hello {{.Name}}", notifiers.FormatText)
	assert.NoError(t, err)

	// Valid HTML template
	err = engine.ValidateTemplate("<h1>{{.Title}}</h1>", notifiers.FormatHTML)
	assert.NoError(t, err)

	// Invalid template syntax
	err = engine.ValidateTemplate("Hello {{.Name", notifiers.FormatText)
	assert.Error(t, err)

	// Invalid HTML template syntax
	err = engine.ValidateTemplate("<h1>{{.Title</h1>", notifiers.FormatHTML)
	assert.Error(t, err)
}

func TestGetAvailableTemplates(t *testing.T) {
	engine := NewEngine()

	// Add some custom templates
	err := engine.AddTextTemplate("custom_text", "Text template")
	require.NoError(t, err)

	err = engine.AddHTMLTemplate("custom_html", "<p>HTML template</p>")
	require.NoError(t, err)

	templates := engine.GetAvailableTemplates()

	// Should contain built-in and custom templates
	assert.True(t, len(templates) >= 2)

	foundCustomText := false
	foundCustomHTML := false

	for _, tmpl := range templates {
		if strings.Contains(tmpl, "custom_text") {
			foundCustomText = true
		}
		if strings.Contains(tmpl, "custom_html") {
			foundCustomHTML = true
		}
	}

	assert.True(t, foundCustomText, "Should list custom text template")
	assert.True(t, foundCustomHTML, "Should list custom HTML template")
}

func TestBuiltinAlertTemplate(t *testing.T) {
	engine := NewEngine()

	message := &notifiers.Message{
		Template: "alert",
		Title:    "System Error",
		Body:     "Database connection failed",
		Variables: map[string]interface{}{
			"server":      "web-01",
			"environment": "production",
			"error":       "connection timeout",
		},
		CreatedAt: time.Date(2023, 5, 15, 14, 30, 0, 0, time.UTC),
	}

	rendered, err := engine.RenderMessage(message)
	assert.NoError(t, err)

	assert.Contains(t, rendered.Body, "🚨 ALERT: System Error")
	assert.Contains(t, rendered.Body, "Database connection failed")
	assert.Contains(t, rendered.Body, "Server: web-01")
	assert.Contains(t, rendered.Body, "Environment: PRODUCTION") // Should be uppercased
	assert.Contains(t, rendered.Body, "Error: connection timeout")
	assert.Contains(t, rendered.Body, "2023-05-15 14:30:00")
	assert.Contains(t, rendered.Body, "automated alert from NotifyHub")
}

func TestBuiltinNoticeTemplate(t *testing.T) {
	engine := NewEngine()

	message := &notifiers.Message{
		Template: "notice",
		Title:    "Deployment Complete",
		Body:     "Application v2.1.0 has been deployed successfully",
		Variables: map[string]interface{}{
			"version":     "v2.1.0",
			"environment": "staging",
			"deployer":    "alice",
		},
		CreatedAt: time.Date(2023, 5, 15, 16, 45, 0, 0, time.UTC),
	}

	rendered, err := engine.RenderMessage(message)
	assert.NoError(t, err)

	assert.Contains(t, rendered.Body, "📢 Deployment Complete")
	assert.Contains(t, rendered.Body, "deployed successfully")
	assert.Contains(t, rendered.Body, "version: v2.1.0")
	assert.Contains(t, rendered.Body, "environment: staging")
	assert.Contains(t, rendered.Body, "deployer: alice")
	assert.Contains(t, rendered.Body, "2023-05-15 16:45:00")
}

