package email

import (
	"github.com/kart-io/notifyhub/platforms/common"
	"github.com/kart-io/notifyhub/transport/email"
)

// EmailAdapter implements the PlatformAdapter interface for Email
type EmailAdapter struct{}

// Name returns the platform name
func (a *EmailAdapter) Name() string {
	return "email"
}

// Supports returns supported message formats for Email
func (a *EmailAdapter) Supports() []string {
	return []string{
		"text",
		"html",
		"markdown",
	}
}

// CreateBuilder creates an Email-specific builder
func (a *EmailAdapter) CreateBuilder(base *common.MessageBuilder) common.PlatformBuilder {
	return &EmailBuilder{
		base:        base,
		ccList:      make([]string, 0),
		bccList:     make([]string, 0),
		attachments: make([]Attachment, 0),
	}
}

// CreateTransport creates an Email transport
func (a *EmailAdapter) CreateTransport(config interface{}) (interface{}, error) {
	if cfg, ok := config.(*email.Config); ok {
		return email.NewTransport(cfg), nil
	}
	return nil, ErrInvalidConfig
}

// Validate validates a message for Email platform
func (a *EmailAdapter) Validate(msg interface{}) error {
	// Use type assertion for validation
	if data, ok := msg.(map[string]interface{}); ok {
		title, _ := data["title"].(string)
		body, _ := data["body"].(string)
		if title == "" && body == "" {
			return ErrEmptyMessage
		}
	}
	return nil
}
