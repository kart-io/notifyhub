package client

import (
	"time"

	"github.com/kart-io/notifyhub/internal"
	"github.com/kart-io/notifyhub/notifiers"
)

// MessageBuilder provides fluent interface for building messages
type MessageBuilder struct {
	message *notifiers.Message
}

// NewMessage creates a new message builder
func NewMessage() *MessageBuilder {
	return &MessageBuilder{
		message: &notifiers.Message{
			Format:    notifiers.FormatText,
			Variables: make(map[string]interface{}),
			Metadata:  make(map[string]string),
			Priority:  3,
			CreatedAt: time.Now(),
		},
	}
}

// Title sets the message title
func (b *MessageBuilder) Title(title string) *MessageBuilder {
	b.message.Title = title
	return b
}

// Body sets the message body
func (b *MessageBuilder) Body(body string) *MessageBuilder {
	b.message.Body = body
	return b
}

// Format sets the message format
func (b *MessageBuilder) Format(format notifiers.MessageFormat) *MessageBuilder {
	b.message.Format = format
	return b
}

// Template sets the template name
func (b *MessageBuilder) Template(template string) *MessageBuilder {
	b.message.Template = template
	return b
}

// Variable adds a template variable
func (b *MessageBuilder) Variable(key string, value interface{}) *MessageBuilder {
	b.message.Variables[key] = value
	return b
}

// Variables adds multiple template variables
func (b *MessageBuilder) Variables(variables map[string]interface{}) *MessageBuilder {
	for k, v := range variables {
		b.message.Variables[k] = v
	}
	return b
}

// Metadata adds metadata
func (b *MessageBuilder) Metadata(key, value string) *MessageBuilder {
	b.message.Metadata[key] = value
	return b
}

// Priority sets the message priority
func (b *MessageBuilder) Priority(priority int) *MessageBuilder {
	b.message.Priority = priority
	return b
}

// Target adds a notification target
func (b *MessageBuilder) Target(target notifiers.Target) *MessageBuilder {
	b.message.Targets = append(b.message.Targets, target)
	return b
}

// Email adds an email target
func (b *MessageBuilder) Email(email string) *MessageBuilder {
	return b.Target(notifiers.Target{Type: notifiers.TargetTypeEmail, Value: email})
}

// FeishuGroup adds a Feishu group target
func (b *MessageBuilder) FeishuGroup(groupID string) *MessageBuilder {
	return b.Target(notifiers.Target{Type: notifiers.TargetTypeGroup, Value: groupID, Platform: "feishu"})
}

// FeishuUser adds a Feishu user target
func (b *MessageBuilder) FeishuUser(userID string) *MessageBuilder {
	return b.Target(notifiers.Target{Type: notifiers.TargetTypeUser, Value: userID, Platform: "feishu"})
}

// Build creates the final message
func (b *MessageBuilder) Build() *notifiers.Message {
	if b.message.ID == "" {
		b.message.ID = internal.GenerateID()
	}
	return b.message
}

// Convenience builders for different message types

// NewAlert creates a new alert message builder
func NewAlert(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Priority(4).
		Metadata("type", "alert").
		Format(notifiers.FormatText)
}

// NewNotice creates a new notice message builder
func NewNotice(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Priority(3).
		Metadata("type", "notice").
		Format(notifiers.FormatText)
}

// NewReport creates a new report message builder
func NewReport(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Priority(2).
		Metadata("type", "report").
		Format(notifiers.FormatText)
}

// NewMarkdown creates a new markdown message builder
func NewMarkdown(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Format(notifiers.FormatMarkdown)
}

// NewHTML creates a new HTML message builder
func NewHTML(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Format(notifiers.FormatHTML)
}