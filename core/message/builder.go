package message

import (
	"math/rand"
	"time"

	"github.com/kart-io/notifyhub/core"
)

// CoreMessageBuilder provides a builder interface for core messages
type CoreMessageBuilder struct {
	message *core.Message
}

// NewBuilder creates a new core message builder
func NewBuilder() *CoreMessageBuilder {
	return &CoreMessageBuilder{
		message: core.NewMessage(),
	}
}

// Title sets the message title
func (b *CoreMessageBuilder) Title(title string) *CoreMessageBuilder {
	b.message.Title = title
	return b
}

// Body sets the message body
func (b *CoreMessageBuilder) Body(body string) *CoreMessageBuilder {
	b.message.Body = body
	return b
}

// Priority sets the message priority
func (b *CoreMessageBuilder) Priority(priority int) *CoreMessageBuilder {
	b.message.Priority = core.Priority(priority)
	return b
}

// Format sets the message format
func (b *CoreMessageBuilder) Format(format core.Format) *CoreMessageBuilder {
	b.message.Format = format
	return b
}

// Template sets the message template
func (b *CoreMessageBuilder) Template(template string) *CoreMessageBuilder {
	b.message.Template = template
	return b
}

// Variable sets a template variable
func (b *CoreMessageBuilder) Variable(key string, value interface{}) *CoreMessageBuilder {
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	b.message.Variables[key] = value
	return b
}

// Variables sets multiple template variables
func (b *CoreMessageBuilder) Variables(vars map[string]interface{}) *CoreMessageBuilder {
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	for k, v := range vars {
		b.message.Variables[k] = v
	}
	return b
}

// Metadata sets a metadata value
func (b *CoreMessageBuilder) Metadata(key, value string) *CoreMessageBuilder {
	if b.message.Metadata == nil {
		b.message.Metadata = make(map[string]string)
	}
	b.message.Metadata[key] = value
	return b
}

// Delay sets the message delay
func (b *CoreMessageBuilder) Delay(delay time.Duration) *CoreMessageBuilder {
	b.message.Delay = delay
	return b
}

// AddTarget adds a target to the message
func (b *CoreMessageBuilder) AddTarget(target core.Target) *CoreMessageBuilder {
	if b.message.Targets == nil {
		b.message.Targets = make([]core.Target, 0)
	}
	b.message.Targets = append(b.message.Targets, target)
	return b
}

// Build builds the final message
func (b *CoreMessageBuilder) Build() *core.Message {
	// Generate ID if not set
	if b.message.ID == "" {
		b.message.ID = generateMessageID()
	}
	return b.message
}

// GetMessage returns the current message state
func (b *CoreMessageBuilder) GetMessage() *core.Message {
	return b.message
}

// Validate validates the current message
func (b *CoreMessageBuilder) Validate() error {
	return b.message.Validate()
}

// === Convenience constructors ===

// Quick creates a simple message with title and body
func Quick(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body)
}

// Alert creates a high-priority alert message
func Alert(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Priority(int(core.PriorityHigh))
}

// Emergency creates a critical priority emergency message
func Emergency(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Priority(int(core.PriorityCritical))
}

// Notice creates a low-priority notice message
func Notice(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Priority(int(core.PriorityLow))
}

// Markdown creates a markdown-formatted message
func Markdown(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Format(core.FormatMarkdown)
}

// HTML creates an HTML-formatted message
func HTML(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Format(core.FormatHTML)
}

// Card creates a card-formatted message
func Card(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Format(core.FormatCard)
}

// === Helper functions ===

// generateMessageID generates a unique message ID
func generateMessageID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 12

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
