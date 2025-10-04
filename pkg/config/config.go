// Package config provides the core configuration system for NotifyHub
package config

import (
	"fmt"
	"time"

	"github.com/kart/notifyhub/pkg/config/platforms"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// Type aliases for platform configurations
type FeishuConfig = platforms.FeishuConfig
type EmailConfig = platforms.EmailConfig
type WebhookConfig = platforms.WebhookConfig

// Config represents the unified configuration structure
type Config struct {
	// Core settings
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`

	// Platform configurations (strongly typed)
	Feishu  *FeishuConfig  `json:"feishu,omitempty"`
	Email   *EmailConfig   `json:"email,omitempty"`
	Webhook *WebhookConfig `json:"webhook,omitempty"`

	// Async configuration
	Async AsyncConfig `json:"async"`

	// Logger configuration
	Logger LoggerConfig `json:"logger"`

	// Instance-level settings
	LoggerInstance logger.Logger `json:"-"`
}

// AsyncConfig configures asynchronous processing
type AsyncConfig struct {
	Enabled bool `json:"enabled"`
	Workers int  `json:"workers"`
}

// LoggerConfig configures logging behavior
type LoggerConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// Option defines a functional option for configuration
type Option func(*Config) error

// New creates a new configuration with the given options
func New(opts ...Option) (*Config, error) {
	// Start with defaults
	cfg := &Config{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		Async: AsyncConfig{
			Enabled: false,
			Workers: 4,
		},
		Logger: LoggerConfig{
			Level:  "info",
			Format: "json",
		},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsAsyncEnabled returns true if async processing is enabled
func (c *Config) IsAsyncEnabled() bool {
	return c.Async.Enabled
}

// GetWorkers returns the number of async workers
func (c *Config) GetWorkers() int {
	if c.Async.Workers <= 0 {
		return 4 // default
	}
	return c.Async.Workers
}

// HasFeishu returns true if Feishu is configured
func (c *Config) HasFeishu() bool {
	return c.Feishu != nil
}

// HasEmail returns true if Email is configured
func (c *Config) HasEmail() bool {
	return c.Email != nil
}

// HasWebhook returns true if Webhook is configured
func (c *Config) HasWebhook() bool {
	return c.Webhook != nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate timeout
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	// Validate max retries
	if c.MaxRetries < 0 {
		c.MaxRetries = 3
	}

	// Validate async configuration
	if c.Async.Workers <= 0 {
		c.Async.Workers = 4
	}

	// Validate logger configuration
	if c.Logger.Level == "" {
		c.Logger.Level = "info"
	}
	if c.Logger.Format == "" {
		c.Logger.Format = "json"
	}

	// Validate platform configurations
	if c.Feishu != nil {
		if err := c.Feishu.Validate(); err != nil {
			return fmt.Errorf("feishu configuration validation failed: %w", err)
		}
	}

	if c.Email != nil {
		if err := c.Email.Validate(); err != nil {
			return fmt.Errorf("email configuration validation failed: %w", err)
		}
	}

	if c.Webhook != nil {
		if err := c.Webhook.Validate(); err != nil {
			return fmt.Errorf("webhook configuration validation failed: %w", err)
		}
	}

	// Ensure logger instance is set
	if c.LoggerInstance == nil {
		c.LoggerInstance = logger.New()
	}

	return nil
}
