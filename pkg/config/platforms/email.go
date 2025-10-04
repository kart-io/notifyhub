// Package platforms provides platform-specific configuration structures
package platforms

import (
	"fmt"
	"time"
)

// EmailConfig represents configuration for Email platform
type EmailConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	From     string `json:"from" yaml:"from"`

	// Security settings
	UseTLS    bool `json:"use_tls" yaml:"use_tls"`
	UseSSL    bool `json:"use_ssl" yaml:"use_ssl"`
	VerifySSL bool `json:"verify_ssl" yaml:"verify_ssl"`

	// Connection settings
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	Retries    int           `json:"retries" yaml:"retries"`
	MaxRetries int           `json:"max_retries" yaml:"max_retries"`
	RateLimit  int           `json:"rate_limit" yaml:"rate_limit"`
}

// Validate validates the Email configuration
func (c *EmailConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required for Email platform")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if c.From == "" {
		return fmt.Errorf("from address is required for Email platform")
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
