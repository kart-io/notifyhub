// Package common provides shared configuration and utilities for NotifyHub examples
package common

import (
	"time"

	"github.com/kart/notifyhub/pkg/config"
)

// ExampleConfig holds configuration for examples
type ExampleConfig struct {
	// Email settings
	Email EmailSettings `json:"email"`

	// Feishu settings
	Feishu FeishuSettings `json:"feishu"`

	// Webhook settings
	Webhook WebhookSettings `json:"webhook"`

	// Test settings
	Test TestSettings `json:"test"`
}

// EmailSettings for email examples
type EmailSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"`
	UseTLS   bool   `json:"use_tls"`
}

// FeishuSettings for feishu examples
type FeishuSettings struct {
	WebhookURL string   `json:"webhook_url"`
	Secret     string   `json:"secret"`
	Keywords   []string `json:"keywords"`
}

// WebhookSettings for webhook examples
type WebhookSettings struct {
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	AuthType string            `json:"auth_type"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Token    string            `json:"token"`
}

// TestSettings for testing
type TestSettings struct {
	DryRun  bool   `json:"dry_run"`
	Verbose bool   `json:"verbose"`
	Delay   string `json:"delay"`
}

// DefaultExampleConfig returns default configuration for examples
func DefaultExampleConfig() *ExampleConfig {
	return &ExampleConfig{
		Email: EmailSettings{
			Host:   "smtp.gmail.com",
			Port:   587,
			UseTLS: true,
			To:     "test@example.com",
		},
		Feishu: FeishuSettings{
			Keywords: []string{"测试"},
		},
		Webhook: WebhookSettings{
			Method: "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		Test: TestSettings{
			DryRun:  false,
			Verbose: true,
			Delay:   "1s",
		},
	}
}

// CreateEmailConfig creates NotifyHub email configuration
func (c *ExampleConfig) CreateEmailConfig() *config.Config {
	return &config.Config{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		Email: &config.EmailConfig{
			Host:     c.Email.Host,
			Port:     c.Email.Port,
			Username: c.Email.Username,
			Password: c.Email.Password,
			From:     c.Email.From,
			UseTLS:   c.Email.UseTLS,
			Timeout:  15 * time.Second,
		},
		Async: config.AsyncConfig{
			Enabled: false,
			Workers: 2,
		},
		Logger: config.LoggerConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// CreateFeishuConfig creates NotifyHub feishu configuration
func (c *ExampleConfig) CreateFeishuConfig() *config.Config {
	return &config.Config{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		Feishu: &config.FeishuConfig{
			WebhookURL: c.Feishu.WebhookURL,
			Secret:     c.Feishu.Secret,
			Keywords:   c.Feishu.Keywords,
			Timeout:    15 * time.Second,
		},
		Async: config.AsyncConfig{
			Enabled: false,
			Workers: 2,
		},
		Logger: config.LoggerConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// CreateWebhookConfig creates NotifyHub webhook configuration
func (c *ExampleConfig) CreateWebhookConfig() *config.Config {
	return &config.Config{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		Webhook: &config.WebhookConfig{
			URL:     c.Webhook.URL,
			Method:  c.Webhook.Method,
			Headers: c.Webhook.Headers,
			Timeout: 15 * time.Second,
		},
		Async: config.AsyncConfig{
			Enabled: false,
			Workers: 2,
		},
		Logger: config.LoggerConfig{
			Level:  "info",
			Format: "text",
		},
	}
}