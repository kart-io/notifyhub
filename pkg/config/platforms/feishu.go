// Package platforms provides platform-specific configuration structures
package platforms

import (
	"fmt"
	"time"
)

// FeishuConfig represents configuration for Feishu platform
type FeishuConfig struct {
	WebhookURL string   `json:"webhook_url" yaml:"webhook_url"`
	Secret     string   `json:"secret" yaml:"secret"`
	Keywords   []string `json:"keywords" yaml:"keywords"`

	// Connection settings
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	Retries    int           `json:"retries" yaml:"retries"`
	MaxRetries int           `json:"max_retries" yaml:"max_retries"`
	RateLimit  int           `json:"rate_limit" yaml:"rate_limit"`

	// Security settings
	VerifySSL bool `json:"verify_ssl" yaml:"verify_ssl"`
}

// Validate validates the Feishu configuration
func (c *FeishuConfig) Validate() error {
	if c.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required for Feishu platform")
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
