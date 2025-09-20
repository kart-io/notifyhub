package message

import (
	"math/rand"
	"time"

	"github.com/kart-io/notifyhub/notifiers"
)

// CoreMessageBuilder implements the common.MessageBuilder interface
type CoreMessageBuilder struct {
	message *notifiers.Message
}

// NewBuilder creates a new core message builder
func NewBuilder() *CoreMessageBuilder {
	return &CoreMessageBuilder{
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
func (b *CoreMessageBuilder) Title(title string) *CoreMessageBuilder {
	b.message.Title = title
	return b
}

// Body sets the message body
func (b *CoreMessageBuilder) Body(body string) *CoreMessageBuilder {
	b.message.Body = body
	return b
}

// Priority sets the message priority (1=low, 5=urgent)
func (b *CoreMessageBuilder) Priority(priority int) *CoreMessageBuilder {
	b.message.Priority = priority
	return b
}

// Format sets the message format
func (b *CoreMessageBuilder) Format(format notifiers.MessageFormat) *CoreMessageBuilder {
	b.message.Format = format
	return b
}

// AddTarget adds a target to the message
func (b *CoreMessageBuilder) AddTarget(target notifiers.Target) *CoreMessageBuilder {
	if b.message.Targets == nil {
		b.message.Targets = make([]notifiers.Target, 0)
	}
	b.message.Targets = append(b.message.Targets, target)
	return b
}

// Build builds the final message
func (b *CoreMessageBuilder) Build() *notifiers.Message {
	// Generate ID if not set
	if b.message.ID == "" {
		b.message.ID = generateMessageID()
	}
	return b.message
}

// GetMessage returns the current message state
func (b *CoreMessageBuilder) GetMessage() *notifiers.Message {
	return b.message
}

// === Additional methods for convenience ===

// Variable sets a template variable
func (b *CoreMessageBuilder) Variable(key string, value interface{}) *CoreMessageBuilder {
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	b.message.Variables[key] = value
	return b
}

// Variables sets multiple template variables
func (b *CoreMessageBuilder) Variables(variables map[string]interface{}) *CoreMessageBuilder {
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	for key, value := range variables {
		b.message.Variables[key] = value
	}
	return b
}

// Metadata sets a metadata key-value pair
func (b *CoreMessageBuilder) Metadata(key, value string) *CoreMessageBuilder {
	if b.message.Metadata == nil {
		b.message.Metadata = make(map[string]string)
	}
	b.message.Metadata[key] = value
	return b
}

// Template sets the template name
func (b *CoreMessageBuilder) Template(template string) *CoreMessageBuilder {
	b.message.Template = template
	return b
}

// Delay sets the delay for sending the message
func (b *CoreMessageBuilder) Delay(delay time.Duration) *CoreMessageBuilder {
	b.message.Delay = delay
	return b
}

// ID sets the message ID
func (b *CoreMessageBuilder) ID(id string) *CoreMessageBuilder {
	b.message.ID = id
	return b
}

// CorePlatformBuilder defines the interface for core platform builders
type CorePlatformBuilder interface {
	Platform() string
	Builder() *CoreMessageBuilder
}

// === Platform-specific builders ===

// Feishu returns a Feishu-specific builder
func (b *CoreMessageBuilder) Feishu() CorePlatformBuilder {
	// Platform-specific builders are not available in core
	// Use the client package for platform-specific features
	return &noOpPlatformBuilder{base: b}
}

// Email returns an Email-specific builder
func (b *CoreMessageBuilder) Email() CorePlatformBuilder {
	// Platform-specific builders are not available in core
	// Use the client package for platform-specific features
	return &noOpPlatformBuilder{base: b}
}

// SMS returns an SMS-specific builder
func (b *CoreMessageBuilder) SMS() CorePlatformBuilder {
	// Platform-specific builders are not available in core
	// Use the client package for platform-specific features
	return &noOpPlatformBuilder{base: b}
}

// Platform returns a platform-specific builder by name
func (b *CoreMessageBuilder) Platform(platform string) CorePlatformBuilder {
	// Platform-specific builders are not available in core
	// Use the client package for platform-specific features
	return &noOpPlatformBuilder{base: b}
}

// === Convenience methods for quick message creation ===

// Quick creates a quick message with title and body
func Quick(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body)
}

// Alert creates an alert message (high priority)
func Alert(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Priority(4)
}

// Emergency creates an emergency message (urgent priority)
func Emergency(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Priority(5)
}

// Notice creates a notice message (normal priority)
func Notice(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Priority(3)
}

// Markdown creates a markdown message
func Markdown(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Format(notifiers.FormatMarkdown)
}

// HTML creates an HTML message
func HTML(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Format(notifiers.FormatHTML)
}

// Card creates a card message
func Card(title, body string) *CoreMessageBuilder {
	return NewBuilder().Title(title).Body(body).Format(notifiers.FormatCard)
}

// === Helper methods ===

// Clone creates a copy of the current builder
func (b *CoreMessageBuilder) Clone() *CoreMessageBuilder {
	newBuilder := &CoreMessageBuilder{
		message: &notifiers.Message{
			ID:        b.message.ID,
			Title:     b.message.Title,
			Body:      b.message.Body,
			Format:    b.message.Format,
			Template:  b.message.Template,
			Priority:  b.message.Priority,
			Delay:     b.message.Delay,
			CardData:  b.message.CardData,
			CreatedAt: b.message.CreatedAt,
		},
	}

	// Deep copy variables
	if b.message.Variables != nil {
		newBuilder.message.Variables = make(map[string]interface{})
		for k, v := range b.message.Variables {
			newBuilder.message.Variables[k] = v
		}
	}

	// Deep copy metadata
	if b.message.Metadata != nil {
		newBuilder.message.Metadata = make(map[string]string)
		for k, v := range b.message.Metadata {
			newBuilder.message.Metadata[k] = v
		}
	}

	// Deep copy targets
	if b.message.Targets != nil {
		newBuilder.message.Targets = make([]notifiers.Target, len(b.message.Targets))
		copy(newBuilder.message.Targets, b.message.Targets)
	}

	// Deep copy mentions
	if b.message.AtMentions != nil {
		newBuilder.message.AtMentions = make([]notifiers.AtMention, len(b.message.AtMentions))
		copy(newBuilder.message.AtMentions, b.message.AtMentions)
	}

	return newBuilder
}

// Validate validates the current message
func (b *CoreMessageBuilder) Validate() error {
	if b.message.Title == "" && b.message.Body == "" {
		return &notifiers.ValidationError{
			Field:   "message",
			Message: "either title or body must be provided",
		}
	}

	if b.message.Priority < 1 || b.message.Priority > 5 {
		return &notifiers.ValidationError{
			Field:   "priority",
			Message: "priority must be between 1 and 5",
		}
	}

	return nil
}

// noOpPlatformBuilder is a no-operation platform builder for unregistered platforms
type noOpPlatformBuilder struct {
	base *CoreMessageBuilder
}

func (n *noOpPlatformBuilder) Platform() string {
	return "unknown"
}

func (n *noOpPlatformBuilder) Builder() *CoreMessageBuilder {
	return n.base
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return "msg_" + generateRandomString(16)
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range result {
		result[i] = charset[rng.Intn(len(charset))]
	}
	return string(result)
}
