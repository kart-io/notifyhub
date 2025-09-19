package api

import (
	"time"

	"github.com/kart-io/notifyhub/notifiers"
	"github.com/kart-io/notifyhub/platforms/registry"
)

// MessageBuilder is the new unified message builder
// This follows the proposal's requirement for platform-agnostic API
type MessageBuilder struct {
	message *notifiers.Message
}

// NewMessage creates a new unified message builder
func NewMessage() *MessageBuilder {
	return &MessageBuilder{
		message: &notifiers.Message{
			Format:    notifiers.FormatText,
			Variables: make(map[string]interface{}),
			Metadata:  make(map[string]string),
			Targets:   make([]notifiers.Target, 0),
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

// Priority sets the message priority
func (b *MessageBuilder) Priority(priority int) *MessageBuilder {
	b.message.Priority = priority
	return b
}

// ToEmail adds an email target
func (b *MessageBuilder) ToEmail(email string) *MessageBuilder {
	target := notifiers.Target{
		Type:     notifiers.TargetTypeEmail,
		Value:    email,
		Platform: "email",
	}
	b.message.Targets = append(b.message.Targets, target)
	return b
}

// Build returns the built message
func (b *MessageBuilder) Build() *notifiers.Message {
	return b.message
}

// Platform returns a platform-specific builder
// This is the new way to access platform-specific features
func (b *MessageBuilder) Platform(platformName string) (interface{}, error) {
	adapter, err := registry.GetPlatform(platformName)
	if err != nil {
		return nil, err
	}

	// Return the platform adapter for platform-specific operations
	return adapter, nil
}

// === Convenience methods for common platforms ===
// These are provided for backward compatibility but use the new architecture

// Feishu returns the Feishu platform adapter
func (b *MessageBuilder) Feishu() (interface{}, error) {
	return b.Platform("feishu")
}

// Email returns the Email platform adapter
func (b *MessageBuilder) Email() (interface{}, error) {
	return b.Platform("email")
}

// SMS returns the SMS platform adapter
func (b *MessageBuilder) SMS() (interface{}, error) {
	return b.Platform("sms")
}
