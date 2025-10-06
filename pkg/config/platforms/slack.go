// Package platforms provides platform-specific configuration structures
package platforms

import (
	"fmt"
	"strings"
	"time"
)

// SlackConfig represents configuration for Slack platform
type SlackConfig struct {
	// Core Slack settings
	WebhookURL string `json:"webhook_url" yaml:"webhook_url"`
	Token      string `json:"token" yaml:"token"`     // Bot token for Slack API
	Channel    string `json:"channel" yaml:"channel"` // Default channel

	// Connection settings
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	Retries    int           `json:"retries" yaml:"retries"`
	MaxRetries int           `json:"max_retries" yaml:"max_retries"`
	RateLimit  int           `json:"rate_limit" yaml:"rate_limit"`

	// Security settings
	VerifySSL bool `json:"verify_ssl" yaml:"verify_ssl"`

	// Slack-specific settings
	Username  string `json:"username" yaml:"username"`     // Bot username (for webhook)
	IconEmoji string `json:"icon_emoji" yaml:"icon_emoji"` // Bot icon emoji
	IconURL   string `json:"icon_url" yaml:"icon_url"`     // Bot icon URL
}

// Validate validates the Slack configuration
func (c *SlackConfig) Validate() error {
	if c.WebhookURL == "" && c.Token == "" {
		return fmt.Errorf("either webhook_url or token is required for Slack platform")
	}

	// Validate webhook URL format if provided
	if c.WebhookURL != "" {
		if !strings.HasPrefix(c.WebhookURL, "https://hooks.slack.com/") {
			return fmt.Errorf("webhook_url must be a valid Slack webhook URL")
		}
	}

	// Validate token format if provided
	if c.Token != "" {
		if !strings.HasPrefix(c.Token, "xoxb-") && !strings.HasPrefix(c.Token, "xoxp-") {
			return fmt.Errorf("token must be a valid Slack bot token (xoxb-) or user token (xoxp-)")
		}
	}

	// Validate channel format if provided
	if c.Channel != "" {
		if !strings.HasPrefix(c.Channel, "#") && !strings.HasPrefix(c.Channel, "@") && !strings.HasPrefix(c.Channel, "C") && !strings.HasPrefix(c.Channel, "D") {
			return fmt.Errorf("channel must start with # (public), @ (user), C (channel ID), or D (DM ID)")
		}
	}

	if c.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	if c.Retries < 0 {
		return fmt.Errorf("retries cannot be negative")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}

	if c.RateLimit < 0 {
		return fmt.Errorf("rate_limit cannot be negative")
	}

	return nil
}

// GetAuthMethod returns the preferred authentication method
func (c *SlackConfig) GetAuthMethod() string {
	if c.Token != "" {
		return "api"
	}
	if c.WebhookURL != "" {
		return "webhook"
	}
	return "none"
}

// IsWebhookMode returns true if using webhook mode
func (c *SlackConfig) IsWebhookMode() bool {
	return c.WebhookURL != "" && c.Token == ""
}

// IsAPIMode returns true if using API mode
func (c *SlackConfig) IsAPIMode() bool {
	return c.Token != ""
}
