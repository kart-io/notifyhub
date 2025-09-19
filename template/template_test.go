package template

import (
	"strings"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/core/message"
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

	htmlTemplate := `<h1>Hello {{.Name}}</h1><p>Score: {{.Score}}</p>`
	err := engine.AddHTMLTemplate("greeting-html", htmlTemplate)
	assert.NoError(t, err)

	// Test invalid HTML template
	invalidTemplate := `<h1>Hello {{.Name</h1>`
	err = engine.AddHTMLTemplate("invalid-html", invalidTemplate)
	assert.Error(t, err)
}

func TestRenderMessageText(t *testing.T) {
	engine := NewEngine()

	// Add custom template
	templateText := `Alert: {{.Title}}
Body: {{.Body}}
Priority: {{.Priority}}`
	err := engine.AddTextTemplate("custom-alert", templateText)
	require.NoError(t, err)

	// Create a message using notifiers.Message
	msg := &notifiers.Message{
		Title:    "System Alert",
		Body:     "Database connection failed",
		Format:   notifiers.FormatText,
		Priority: message.PriorityHigh,
		Template: "custom-alert",
	}

	// Render using template
	rendered, err := engine.RenderMessage(msg)
	assert.NoError(t, err)
	assert.Contains(t, rendered.Title, "Alert: System Alert")
	assert.Contains(t, rendered.Body, "Body: Database connection failed")
	assert.Contains(t, rendered.Body, "Priority: 4") // High priority = 4
}

func TestRenderMessageHTML(t *testing.T) {
	engine := NewEngine()

	// Add custom HTML template
	htmlTemplate := `<div class="alert">
<h2>{{.Title}}</h2>
<p>{{.Body}}</p>
<span class="priority">Priority: {{.Priority}}</span>
</div>`
	err := engine.AddHTMLTemplate("custom-alert-html", htmlTemplate)
	require.NoError(t, err)

	// Create a message using notifiers.Message
	msg := &notifiers.Message{
		Title:    "System Alert",
		Body:     "Database connection failed",
		Format:   notifiers.FormatHTML,
		Priority: message.PriorityHigh,
		Template: "custom-alert-html",
	}

	// Render as HTML
	rendered, err := engine.RenderMessage(msg)
	assert.NoError(t, err)
	assert.Contains(t, rendered.Body, "<h2>System Alert</h2>")
	assert.Contains(t, rendered.Body, "<p>Database connection failed</p>")
}

func TestRenderWithVariables(t *testing.T) {
	engine := NewEngine()

	// Template with variables
	templateText := `Server: {{.server}}
Status: {{.status}}
Response Time: {{.responseTime}}ms`
	err := engine.AddTextTemplate("server-status", templateText)
	require.NoError(t, err)

	// Create message with variables using notifiers.Message
	msg := &notifiers.Message{
		Title:    "Server Status",
		Body:     "Server health check",
		Format:   notifiers.FormatText,
		Template: "server-status",
		Variables: map[string]interface{}{
			"server":       "api-01.example.com",
			"status":       "healthy",
			"responseTime": "45",
		},
	}

	// Render with variables
	rendered, err := engine.RenderMessage(msg)
	assert.NoError(t, err)
	assert.Contains(t, rendered.Body, "Server: api-01.example.com")
	assert.Contains(t, rendered.Body, "Status: healthy")
	assert.Contains(t, rendered.Body, "Response Time: 45ms")
}

func TestTemplateHelpers(t *testing.T) {
	engine := NewEngine()

	// Template using helper functions
	templateText := `Time: {{formatTime .Time}}
Uppercase: {{upper .Name}}
Lowercase: {{lower .Name}}
Title Case: {{title .description}}`
	err := engine.AddTextTemplate("helpers-test", templateText)
	require.NoError(t, err)

	// Create message with data for helpers
	msg := &notifiers.Message{
		Title:    "Helper Test",
		Body:     "Testing template helpers",
		Format:   notifiers.FormatText,
		Template: "helpers-test",
		Variables: map[string]interface{}{
			"Time":        time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			"Name":        "John Doe",
			"description": "this is a test description",
		},
	}

	// Render
	rendered, err := engine.RenderMessage(msg)
	assert.NoError(t, err)
	assert.Contains(t, rendered.Body, "Time: 2024-01-15 14:30:00")
	assert.Contains(t, rendered.Body, "Uppercase: JOHN DOE")
	assert.Contains(t, rendered.Body, "Lowercase: john doe")
	assert.Contains(t, rendered.Body, "Title Case: This Is A Test Description")
}

func TestTemplateNotFound(t *testing.T) {
	engine := NewEngine()

	// Create message with non-existent template
	msg := message.NewMessage()
	msg.SetTitle("Test").
		SetBody("Body").
		SetTemplate("non-existent")

	// Convert to notifiers.Message for template engine
	notifierMsg := &notifiers.Message{
		ID:        msg.ID,
		Title:     msg.Title,
		Body:      msg.Body,
		Format:    notifiers.MessageFormat(msg.Format),
		Template:  msg.Template,
		Variables: msg.Variables,
		Metadata:  msg.Metadata,
		Priority:  msg.Priority,
		CreatedAt: msg.CreatedAt,
	}

	// Should return error
	_, err := engine.RenderMessage(notifierMsg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRenderMessageWithoutTemplate(t *testing.T) {
	engine := NewEngine()

	// Message without template should use default rendering
	msg := message.NewMessage()
	msg.SetTitle("Simple Message").
		SetBody("This is the body")

	// Convert to notifiers.Message for template engine
	notifierMsg := &notifiers.Message{
		ID:        msg.ID,
		Title:     msg.Title,
		Body:      msg.Body,
		Format:    notifiers.MessageFormat(msg.Format),
		Template:  msg.Template,
		Variables: msg.Variables,
		Metadata:  msg.Metadata,
		Priority:  msg.Priority,
		CreatedAt: msg.CreatedAt,
	}

	// Should use default rendering (just return body for simplicity)
	rendered, err := engine.RenderMessage(notifierMsg)
	assert.NoError(t, err)
	assert.Contains(t, rendered, "This is the body")
}

func TestMarkdownFormat(t *testing.T) {
	engine := NewEngine()

	// Add markdown template
	mdTemplate := `# {{.Title}}

{{.Body}}

**Priority:** {{.Priority}}
**Time:** {{formatTime .CreatedAt}}`
	err := engine.AddTextTemplate("markdown-alert", mdTemplate)
	require.NoError(t, err)

	// Create message
	msg := message.NewMessage()
	msg.SetTitle("Alert").
		SetBody("System issue detected").
		SetTemplate("markdown-alert").
		SetPriority(message.PriorityHigh)

	// Convert to notifiers.Message for template engine
	notifierMsg := &notifiers.Message{
		ID:        msg.ID,
		Title:     msg.Title,
		Body:      msg.Body,
		Format:    notifiers.FormatMarkdown,
		Template:  msg.Template,
		Variables: msg.Variables,
		Metadata:  msg.Metadata,
		Priority:  msg.Priority,
		CreatedAt: msg.CreatedAt,
	}

	// Render as markdown
	rendered, err := engine.RenderMessage(notifierMsg)
	assert.NoError(t, err)
	assert.Contains(t, rendered, "# Alert")
	assert.Contains(t, rendered, "**Priority:** 3")
}
