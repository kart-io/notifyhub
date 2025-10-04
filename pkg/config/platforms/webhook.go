// Package platforms provides platform-specific configuration structures
package platforms

import (
	"fmt"
	"strings"
	"time"
)

// WebhookConfig represents configuration for Webhook platform
type WebhookConfig struct {
	URL         string            `json:"url" yaml:"url"`
	Method      string            `json:"method" yaml:"method"`
	Headers     map[string]string `json:"headers" yaml:"headers"`
	ContentType string            `json:"content_type" yaml:"content_type"`

	// Authentication
	AuthType string `json:"auth_type" yaml:"auth_type"` // "none", "basic", "bearer", "custom"
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Token    string `json:"token" yaml:"token"`

	// Security settings
	VerifySSL bool `json:"verify_ssl" yaml:"verify_ssl"`

	// Connection settings
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	Retries    int           `json:"retries" yaml:"retries"`
	MaxRetries int           `json:"max_retries" yaml:"max_retries"`
	RateLimit  int           `json:"rate_limit" yaml:"rate_limit"`
}

// Validate validates the Webhook configuration
func (c *WebhookConfig) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required for Webhook platform")
	}

	if c.Method == "" {
		c.Method = "POST" // Default to POST
	} else {
		method := strings.ToUpper(c.Method)
		validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
		isValid := false
		for _, valid := range validMethods {
			if method == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid HTTP method: %s", c.Method)
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
