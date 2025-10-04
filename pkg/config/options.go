// Functional options for NotifyHub configuration
package config

import (
	"time"

	"github.com/kart/notifyhub/pkg/utils/logger"
)

// WithFeishu configures Feishu platform
func WithFeishu(config FeishuConfig) Option {
	return func(c *Config) error {
		c.Feishu = &config
		return nil
	}
}

// WithEmail configures Email platform
func WithEmail(config EmailConfig) Option {
	return func(c *Config) error {
		c.Email = &config
		return nil
	}
}

// WithWebhook configures Webhook platform
func WithWebhook(config WebhookConfig) Option {
	return func(c *Config) error {
		c.Webhook = &config
		return nil
	}
}

// WithAsync enables async processing with specified workers
func WithAsync(workers int) Option {
	return func(c *Config) error {
		c.Async.Enabled = true
		c.Async.Workers = workers
		return nil
	}
}

// WithLogger sets the logger instance
func WithLogger(logger logger.Logger) Option {
	return func(c *Config) error {
		c.LoggerInstance = logger
		return nil
	}
}

// WithTimeout sets the default timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		c.Timeout = timeout
		return nil
	}
}

// WithMaxRetries sets the maximum retry attempts
func WithMaxRetries(retries int) Option {
	return func(c *Config) error {
		c.MaxRetries = retries
		return nil
	}
}

// WithDefaults applies sensible defaults
func WithDefaults() Option {
	return func(c *Config) error {
		c.Timeout = 30 * time.Second
		c.MaxRetries = 3
		c.Logger.Level = "info"
		c.Logger.Format = "json"
		return nil
	}
}

// WithTestDefaults applies test-friendly defaults
func WithTestDefaults() Option {
	return func(c *Config) error {
		c.Timeout = 5 * time.Second
		c.MaxRetries = 1
		c.Logger.Level = "debug"
		c.Logger.Format = "text"
		return nil
	}
}

// Helper functions for quick configuration creation

// NewFeishuConfig creates a new Feishu configuration
func NewFeishuConfig(webhookURL, secret string) FeishuConfig {
	return FeishuConfig{
		WebhookURL: webhookURL,
		Secret:     secret,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
}

// NewEmailConfig creates a new Email configuration
func NewEmailConfig(host string, port int, from string) EmailConfig {
	return EmailConfig{
		Host:       host,
		Port:       port,
		From:       from,
		UseTLS:     true,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
}

// NewWebhookConfig creates a new Webhook configuration
func NewWebhookConfig(url string) WebhookConfig {
	return WebhookConfig{
		URL:         url,
		Method:      "POST",
		ContentType: "application/json",
		VerifySSL:   true,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		Headers:     make(map[string]string),
	}
}

// WithQuickFeishu is a convenience method for quick Feishu setup
func WithQuickFeishu(webhookURL, secret string) Option {
	return WithFeishu(NewFeishuConfig(webhookURL, secret))
}

// WithQuickEmail is a convenience method for quick Email setup
func WithQuickEmail(host string, port int, from string) Option {
	return WithEmail(NewEmailConfig(host, port, from))
}

// WithQuickWebhook is a convenience method for quick Webhook setup
func WithQuickWebhook(url string) Option {
	return WithWebhook(NewWebhookConfig(url))
}

// Production configuration preset
func WithProductionDefaults() Option {
	return func(c *Config) error {
		c.Timeout = 60 * time.Second
		c.MaxRetries = 5
		c.Async.Enabled = true
		c.Async.Workers = 8
		c.Logger.Level = "info"
		c.Logger.Format = "json"
		return nil
	}
}
