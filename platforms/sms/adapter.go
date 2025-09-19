package sms

import (
	"github.com/kart-io/notifyhub/platforms/common"
)

// SMSAdapter implements the PlatformAdapter interface for SMS
type SMSAdapter struct{}

// Name returns the platform name
func (a *SMSAdapter) Name() string {
	return "sms"
}

// Supports returns supported message formats for SMS
func (a *SMSAdapter) Supports() []string {
	return []string{
		"text", // SMS primarily supports text
	}
}

// CreateBuilder creates an SMS-specific builder
func (a *SMSAdapter) CreateBuilder(base *common.MessageBuilder) common.PlatformBuilder {
	return &SMSBuilder{
		base:           base,
		fromNumber:     "",
		groups:         make([]string, 0),
		deliveryReport: false,
	}
}

// CreateTransport creates an SMS transport
func (a *SMSAdapter) CreateTransport(config interface{}) (interface{}, error) {
	if cfg, ok := config.(*SMSConfig); ok {
		return NewSMSTransport(cfg), nil
	}
	return nil, ErrInvalidConfig
}

// Validate validates a message for SMS platform
func (a *SMSAdapter) Validate(msg interface{}) error {
	// Use type assertion for validation
	if data, ok := msg.(map[string]interface{}); ok {
		body, _ := data["body"].(string)
		if body == "" {
			return ErrEmptyMessage
		}
		if len(body) > 160 {
			return ErrMessageTooLong
		}
	}
	return nil
}

// NewSMSTransport creates a new SMS transport (placeholder)
func NewSMSTransport(config *SMSConfig) interface{} {
	// This would implement the actual SMS transport
	return &SMSTransport{config: config}
}

// SMSTransport placeholder for SMS transport implementation
type SMSTransport struct {
	config *SMSConfig
}
