// Package email provides email template functionality for NotifyHub
package email

import (
	"bytes"
	"fmt"
	htmlTemplate "html/template"
	"os"
	"path/filepath"
	"strings"
	textTemplate "text/template"

	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// TemplateType represents the type of email template
type TemplateType string

const (
	TemplateTypeText     TemplateType = "text"
	TemplateTypeHTML     TemplateType = "html"
	TemplateTypeMarkdown TemplateType = "markdown"
)

// EmailTemplate represents an email template
type EmailTemplate struct {
	Name        string                 `json:"name"`
	Type        TemplateType           `json:"type"`
	Subject     string                 `json:"subject"`
	Content     string                 `json:"content"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IsDefault   bool                   `json:"is_default,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// TemplateData represents data to be used in template rendering
type TemplateData struct {
	// Message data
	Title    string `json:"title"`
	Body     string `json:"body"`
	Priority string `json:"priority"`
	ID       string `json:"id"`

	// Template variables
	Variables map[string]interface{} `json:"variables"`

	// System data
	Timestamp string `json:"timestamp"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`

	// Custom data
	Custom map[string]interface{} `json:"custom"`
}

// TemplateManager manages email templates
type TemplateManager struct {
	templates   map[string]*EmailTemplate
	templateDir string
	logger      logger.Logger
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(templateDir string, logger logger.Logger) *TemplateManager {
	return &TemplateManager{
		templates:   make(map[string]*EmailTemplate),
		templateDir: templateDir,
		logger:      logger,
	}
}

// LoadTemplatesFromDir loads templates from a directory
func (tm *TemplateManager) LoadTemplatesFromDir() error {
	if tm.templateDir == "" {
		tm.logger.Debug("模板目录未设置，跳过模板加载")
		return nil
	}

	files, err := os.ReadDir(tm.templateDir)
	if err != nil {
		tm.logger.Warn("无法读取模板目录", "dir", tm.templateDir, "error", err)
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext != ".html" && ext != ".txt" && ext != ".md" {
			continue
		}

		if err := tm.loadTemplateFromFile(filepath.Join(tm.templateDir, file.Name())); err != nil {
			tm.logger.Error("加载模板文件失败", "file", file.Name(), "error", err)
		}
	}

	tm.logger.Info("模板加载完成", "count", len(tm.templates), "dir", tm.templateDir)
	return nil
}

// loadTemplateFromFile loads a template from a file
func (tm *TemplateManager) loadTemplateFromFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取模板文件失败: %w", err)
	}

	fileName := filepath.Base(filePath)
	ext := strings.ToLower(filepath.Ext(fileName))
	name := strings.TrimSuffix(fileName, ext)

	var templateType TemplateType
	switch ext {
	case ".html":
		templateType = TemplateTypeHTML
	case ".txt":
		templateType = TemplateTypeText
	case ".md":
		templateType = TemplateTypeMarkdown
	default:
		templateType = TemplateTypeText
	}

	// Parse template content for metadata
	templateContent := string(content)
	subject, body := tm.parseTemplateContent(templateContent)

	template := &EmailTemplate{
		Name:    name,
		Type:    templateType,
		Subject: subject,
		Content: body,
	}

	tm.templates[name] = template
	tm.logger.Debug("模板加载成功", "name", name, "type", templateType)
	return nil
}

// parseTemplateContent parses template content to extract subject and body
func (tm *TemplateManager) parseTemplateContent(content string) (subject, body string) {
	lines := strings.Split(content, "\n")

	// Look for subject line (e.g., "Subject: {{.Title}}")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Subject:") {
			subject = strings.TrimSpace(strings.TrimPrefix(line, "Subject:"))
			// Remove the subject line and join the rest as body
			if i+1 < len(lines) {
				body = strings.Join(lines[i+1:], "\n")
			}
			return
		}
	}

	// If no subject line found, use the entire content as body
	body = content
	return
}

// AddTemplate adds a template
func (tm *TemplateManager) AddTemplate(template *EmailTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("模板名称不能为空")
	}

	tm.templates[template.Name] = template
	tm.logger.Info("添加模板", "name", template.Name, "type", template.Type)
	return nil
}

// GetTemplate gets a template by name
func (tm *TemplateManager) GetTemplate(name string) (*EmailTemplate, error) {
	template, exists := tm.templates[name]
	if !exists {
		return nil, fmt.Errorf("模板不存在: %s", name)
	}
	return template, nil
}

// RenderTemplate renders a template with data
func (tm *TemplateManager) RenderTemplate(templateName string, data *TemplateData) (*message.Message, error) {
	emailTemplate, err := tm.GetTemplate(templateName)
	if err != nil {
		return nil, err
	}

	// Render subject
	subject, err := tm.renderText(emailTemplate.Subject, data)
	if err != nil {
		return nil, fmt.Errorf("渲染邮件主题失败: %w", err)
	}

	// Render content
	body, err := tm.renderContent(emailTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("渲染邮件内容失败: %w", err)
	}

	// Create message
	msg := message.New()
	msg.Title = subject
	msg.Body = body

	// Set format based on template type
	switch emailTemplate.Type {
	case TemplateTypeHTML:
		msg.Format = message.FormatHTML
	case TemplateTypeMarkdown:
		msg.Format = message.FormatMarkdown
	default:
		msg.Format = message.FormatText
	}

	// Set priority if provided in data
	if data.Priority != "" {
		switch strings.ToLower(data.Priority) {
		case "urgent":
			msg.Priority = message.PriorityUrgent
		case "high":
			msg.Priority = message.PriorityHigh
		case "low":
			msg.Priority = message.PriorityLow
		default:
			msg.Priority = message.PriorityNormal
		}
	}

	// Add metadata
	if emailTemplate.Metadata != nil {
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]interface{})
		}
		for k, v := range emailTemplate.Metadata {
			msg.Metadata[k] = v
		}
	}

	tm.logger.Debug("模板渲染成功", "template", templateName, "subject", subject)
	return msg, nil
}

// renderContent renders template content based on type
func (tm *TemplateManager) renderContent(emailTemplate *EmailTemplate, data *TemplateData) (string, error) {
	switch emailTemplate.Type {
	case TemplateTypeHTML:
		return tm.renderHTML(emailTemplate.Content, data)
	case TemplateTypeMarkdown:
		return tm.renderText(emailTemplate.Content, data)
	default:
		return tm.renderText(emailTemplate.Content, data)
	}
}

// renderText renders text template
func (tm *TemplateManager) renderText(content string, data *TemplateData) (string, error) {
	tmpl, err := textTemplate.New("text").Parse(content)
	if err != nil {
		return "", fmt.Errorf("解析文本模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("执行文本模板失败: %w", err)
	}

	return buf.String(), nil
}

// renderHTML renders HTML template
func (tm *TemplateManager) renderHTML(content string, data *TemplateData) (string, error) {
	tmpl, err := htmlTemplate.New("html").Parse(content)
	if err != nil {
		return "", fmt.Errorf("解析HTML模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("执行HTML模板失败: %w", err)
	}

	return buf.String(), nil
}

// ListTemplates returns all available templates
func (tm *TemplateManager) ListTemplates() map[string]*EmailTemplate {
	result := make(map[string]*EmailTemplate)
	for k, v := range tm.templates {
		result[k] = v
	}
	return result
}

// GetDefaultTemplate returns the default template or creates a basic one
func (tm *TemplateManager) GetDefaultTemplate() *EmailTemplate {
	// Look for a template marked as default
	for _, template := range tm.templates {
		if template.IsDefault {
			return template
		}
	}

	// Return a basic default template
	return &EmailTemplate{
		Name:        "default",
		Type:        TemplateTypeText,
		Subject:     "{{.Title}}",
		Content:     "{{.Body}}",
		IsDefault:   true,
		Description: "默认邮件模板",
	}
}

// CreateBasicTemplates creates basic built-in templates
func (tm *TemplateManager) CreateBasicTemplates() {
	templates := []*EmailTemplate{
		{
			Name:    "notification",
			Type:    TemplateTypeHTML,
			Subject: "[通知] {{.Title}}",
			Content: `
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
        .header { background: #f8f9fa; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .content { line-height: 1.6; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="header">
        <h2>{{.Title}}</h2>
    </div>
    <div class="content">
        {{.Body}}
    </div>
    <div class="footer">
        <p>发送时间: {{.Timestamp}}</p>
        <p>发件人: {{.Sender}}</p>
        <p>此邮件由 NotifyHub 自动发送</p>
    </div>
</body>
</html>`,
			Description: "通知类邮件模板",
		},
		{
			Name:    "alert",
			Type:    TemplateTypeHTML,
			Subject: "🚨 [警报] {{.Title}}",
			Content: `
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
        .alert { background: #f8d7da; color: #721c24; padding: 15px; border-radius: 5px; border: 1px solid #f5c6cb; margin-bottom: 20px; }
        .content { line-height: 1.6; }
        .priority { font-weight: bold; color: #dc3545; }
    </style>
</head>
<body>
    <div class="alert">
        <h2>🚨 警报通知</h2>
        <p class="priority">优先级: {{.Priority}}</p>
    </div>
    <div class="content">
        <h3>{{.Title}}</h3>
        {{.Body}}
    </div>
    <div style="margin-top: 30px; color: #666; font-size: 12px;">
        <p>警报时间: {{.Timestamp}}</p>
        <p>请及时处理此警报</p>
    </div>
</body>
</html>`,
			Description: "警报类邮件模板",
		},
		{
			Name:    "plain",
			Type:    TemplateTypeText,
			Subject: "{{.Title}}",
			Content: `{{.Title}}

{{.Body}}

---
发送时间: {{.Timestamp}}
发件人: {{.Sender}}

此邮件由 NotifyHub 自动发送`,
			Description: "纯文本邮件模板",
			IsDefault:   true,
		},
		{
			Name:    "marketing",
			Type:    TemplateTypeHTML,
			Subject: "{{.Title}}",
			Content: `
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f4f4f4; }
        .container { max-width: 600px; margin: 0 auto; background: white; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; }
        .content { padding: 30px; }
        .button { display: inline-block; background: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; margin: 20px 0; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
        </div>
        <div class="content">
            {{.Body}}
            {{if .Variables.button_text}}
            <p><a href="{{.Variables.button_url}}" class="button">{{.Variables.button_text}}</a></p>
            {{end}}
        </div>
        <div class="footer">
            <p>{{.Sender}}</p>
            <p>发送时间: {{.Timestamp}}</p>
        </div>
    </div>
</body>
</html>`,
			Description: "营销类邮件模板",
		},
	}

	for _, template := range templates {
		if err := tm.AddTemplate(template); err != nil {
			tm.logger.Error("Failed to add template", "template_name", template.Name, "error", err)
		}
	}

	tm.logger.Info("创建基础模板完成", "count", len(templates))
}

// ValidateTemplate validates a template
func (tm *TemplateManager) ValidateTemplate(template *EmailTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("模板名称不能为空")
	}

	if template.Content == "" {
		return fmt.Errorf("模板内容不能为空")
	}

	// Try to parse the template
	testData := &TemplateData{
		Title:     "测试标题",
		Body:      "测试内容",
		Timestamp: "2023-01-01 12:00:00",
		Sender:    "test@example.com",
		Variables: make(map[string]interface{}),
	}

	// Test subject rendering
	if template.Subject != "" {
		_, err := tm.renderText(template.Subject, testData)
		if err != nil {
			return fmt.Errorf("模板主题验证失败: %w", err)
		}
	}

	// Test content rendering
	_, err := tm.renderContent(template, testData)
	if err != nil {
		return fmt.Errorf("模板内容验证失败: %w", err)
	}

	return nil
}

// RemoveTemplate removes a template
func (tm *TemplateManager) RemoveTemplate(name string) error {
	if _, exists := tm.templates[name]; !exists {
		return fmt.Errorf("模板不存在: %s", name)
	}

	delete(tm.templates, name)
	tm.logger.Info("删除模板", "name", name)
	return nil
}
