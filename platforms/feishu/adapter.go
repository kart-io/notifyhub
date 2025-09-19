package feishu

import (
	"github.com/kart-io/notifyhub/platforms/common"
	"github.com/kart-io/notifyhub/transport/feishu"
)

// FeishuAdapter implements the PlatformAdapter interface for Feishu
type FeishuAdapter struct{}

// Name returns the platform name
func (a *FeishuAdapter) Name() string {
	return "feishu"
}

// Supports returns supported message formats for Feishu
func (a *FeishuAdapter) Supports() []string {
	return []string{
		"text",
		"markdown",
		"card",
	}
}

// CreateBuilder creates a Feishu-specific builder
func (a *FeishuAdapter) CreateBuilder(base *common.MessageBuilder) common.PlatformBuilder {
	return &FeishuBuilder{
		base:     base,
		mentions: make([]FeishuMention, 0),
	}
}

// CreateTransport creates a Feishu transport
func (a *FeishuAdapter) CreateTransport(config interface{}) (interface{}, error) {
	if cfg, ok := config.(*feishu.Config); ok {
		return feishu.NewTransport(cfg), nil
	}
	return nil, ErrInvalidConfig
}

// Validate validates a message for Feishu platform
func (a *FeishuAdapter) Validate(msg interface{}) error {
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
