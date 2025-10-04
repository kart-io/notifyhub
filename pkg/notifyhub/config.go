// Package notifyhub provides unified configuration management
package notifyhub

import (
	"fmt"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/utils/logger"
)

// Config represents the unified configuration for NotifyHub
// This replaces the complex map-based configuration with strong typing
type Config struct {
	// Core settings
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	MaxRetries int           `json:"max_retries" yaml:"max_retries"`

	// Platform configurations (strongly typed)
	Feishu  *FeishuConfig  `json:"feishu,omitempty" yaml:"feishu,omitempty"`
	Email   *EmailConfig   `json:"email,omitempty" yaml:"email,omitempty"`
	Webhook *WebhookConfig `json:"webhook,omitempty" yaml:"webhook,omitempty"`

	// Async configuration
	Async AsyncConfig `json:"async" yaml:"async"`

	// Logger configuration
	Logger LoggerConfig `json:"logger" yaml:"logger"`

	// Runtime logger instance (not serialized)
	LoggerInstance logger.Logger `json:"-" yaml:"-"`
}

// Option represents a functional configuration option
type Option func(*Config) error

// FeishuConfig represents Feishu platform configuration
type FeishuConfig struct {
	// Webhook configuration
	WebhookURL string   `json:"webhook_url" yaml:"webhook_url" validate:"required,url"`
	Secret     string   `json:"secret" yaml:"secret"`
	Keywords   []string `json:"keywords" yaml:"keywords"`

	// App-based authentication (alternative to webhook)
	AppID     string `json:"app_id" yaml:"app_id"`
	AppSecret string `json:"app_secret" yaml:"app_secret"`
	AuthType  string `json:"auth_type" yaml:"auth_type"` // "webhook" or "app"

	// Network and retry configuration
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	MaxRetries int           `json:"max_retries" yaml:"max_retries" validate:"min=0"`
	RateLimit  int           `json:"rate_limit" yaml:"rate_limit" validate:"min=0"` // requests per minute

	// Security configuration
	SignVerify bool `json:"sign_verify" yaml:"sign_verify"` // Enable signature verification
}

// EmailConfig represents email platform configuration
type EmailConfig struct {
	Host     string        `json:"host" yaml:"host"`
	Port     int           `json:"port" yaml:"port"`
	Username string        `json:"username" yaml:"username"`
	Password string        `json:"password" yaml:"password"`
	From     string        `json:"from" yaml:"from"`
	UseTLS   bool          `json:"use_tls" yaml:"use_tls"`
	Timeout  time.Duration `json:"timeout" yaml:"timeout"`
}

// WebhookConfig represents webhook platform configuration
type WebhookConfig struct {
	URL     string            `json:"url" yaml:"url"`
	Method  string            `json:"method" yaml:"method"`
	Headers map[string]string `json:"headers" yaml:"headers"`
	Timeout time.Duration     `json:"timeout" yaml:"timeout"`
}

// AsyncConfig represents asynchronous processing configuration
type AsyncConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
	Workers int  `json:"workers" yaml:"workers"`
}

// LoggerConfig represents logger configuration
type LoggerConfig struct {
	Level  string `json:"level" yaml:"level"`
	Format string `json:"format" yaml:"format"`
}

// Core configuration options

// WithFeishu configures Feishu platform
func WithFeishu(config FeishuConfig) Option {
	return func(c *Config) error {
		// Apply defaults first
		applyFeishuDefaults(&config)

		// Validate configuration
		if err := validateFeishuConfig(&config); err != nil {
			return fmt.Errorf("feishu config validation failed: %w", err)
		}

		c.Feishu = &config
		return nil
	}
}

// validateFeishuConfig validates Feishu configuration
func validateFeishuConfig(config *FeishuConfig) error {
	// AuthType validation
	if config.AuthType == "" {
		config.AuthType = "webhook" // Default to webhook
	}

	switch config.AuthType {
	case "webhook":
		if config.WebhookURL == "" {
			return fmt.Errorf("webhook_url is required for webhook auth type")
		}
		// Basic URL validation
		if !strings.HasPrefix(config.WebhookURL, "http://") && !strings.HasPrefix(config.WebhookURL, "https://") {
			return fmt.Errorf("webhook_url must start with http:// or https://")
		}
	case "app":
		if config.AppID == "" {
			return fmt.Errorf("app_id is required for app auth type")
		}
		if config.AppSecret == "" {
			return fmt.Errorf("app_secret is required for app auth type")
		}
	default:
		return fmt.Errorf("auth_type must be 'webhook' or 'app', got: %s", config.AuthType)
	}

	// Validate timeout
	if config.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	// Validate max retries
	if config.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}

	// Validate rate limit
	if config.RateLimit < 0 {
		return fmt.Errorf("rate_limit cannot be negative")
	}

	// Validate keywords
	for i, keyword := range config.Keywords {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			return fmt.Errorf("keyword at index %d cannot be empty", i)
		}
		config.Keywords[i] = trimmed
	}

	return nil
}

// applyFeishuDefaults applies default values to Feishu configuration
func applyFeishuDefaults(config *FeishuConfig) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RateLimit == 0 {
		config.RateLimit = 60 // 60 requests per minute default
	}
	if config.AuthType == "" {
		config.AuthType = "webhook"
	}

	// Clean up keywords
	cleanKeywords := make([]string, 0, len(config.Keywords))
	for _, keyword := range config.Keywords {
		trimmed := strings.TrimSpace(keyword)
		if trimmed != "" {
			cleanKeywords = append(cleanKeywords, trimmed)
		}
	}
	config.Keywords = cleanKeywords
}

// WithEmail configures email platform
func WithEmail(config EmailConfig) Option {
	return func(c *Config) error {
		if config.Host == "" {
			return fmt.Errorf("email host is required")
		}
		if config.Port <= 0 {
			return fmt.Errorf("email port must be positive")
		}
		if config.From == "" {
			return fmt.Errorf("email from address is required")
		}
		c.Email = &config
		return nil
	}
}

// WithWebhook configures webhook platform
func WithWebhook(config WebhookConfig) Option {
	return func(c *Config) error {
		if config.URL == "" {
			return fmt.Errorf("webhook URL is required")
		}
		if config.Method == "" {
			config.Method = "POST" // Default method
		}
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		c.Webhook = &config
		return nil
	}
}

// WithAsync configures asynchronous processing
func WithAsync(workers int) Option {
	return func(c *Config) error {
		if workers <= 0 {
			return fmt.Errorf("workers must be positive")
		}
		c.Async.Enabled = true
		c.Async.Workers = workers
		return nil
	}
}

// WithTimeout sets the global timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
		c.Timeout = timeout
		return nil
	}
}

// WithMaxRetries sets the maximum retry count
func WithMaxRetries(retries int) Option {
	return func(c *Config) error {
		if retries < 0 {
			return fmt.Errorf("max retries must be non-negative")
		}
		c.MaxRetries = retries
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

// WithLoggerConfig configures logging settings
func WithLoggerConfig(level, format string) Option {
	return func(c *Config) error {
		c.Logger.Level = level
		c.Logger.Format = format
		return nil
	}
}

// Convenience configuration functions

// WithFeishuWebhook creates a simplified Feishu webhook configuration
func WithFeishuWebhook(webhookURL, secret string) Option {
	return WithFeishu(FeishuConfig{
		WebhookURL: webhookURL,
		Secret:     secret,
		AuthType:   "webhook",
	})
}

// WithFeishuApp creates a Feishu app-based configuration
func WithFeishuApp(appID, appSecret string) Option {
	return WithFeishu(FeishuConfig{
		AppID:     appID,
		AppSecret: appSecret,
		AuthType:  "app",
	})
}

// WithFeishuAdvanced creates an advanced Feishu configuration with all options
func WithFeishuAdvanced(webhookURL, secret string, keywords []string, timeout time.Duration, maxRetries, rateLimit int, signVerify bool) Option {
	return WithFeishu(FeishuConfig{
		WebhookURL: webhookURL,
		Secret:     secret,
		Keywords:   keywords,
		AuthType:   "webhook",
		Timeout:    timeout,
		MaxRetries: maxRetries,
		RateLimit:  rateLimit,
		SignVerify: signVerify,
	})
}

// WithEmailSMTP creates a simplified email SMTP configuration
func WithEmailSMTP(host string, port int, username, password, from string) Option {
	return WithEmail(EmailConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		UseTLS:   true,
		Timeout:  30 * time.Second,
	})
}

// WithWebhookBasic creates a simplified webhook configuration
func WithWebhookBasic(url string) Option {
	return WithWebhook(WebhookConfig{
		URL:     url,
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Timeout: 30 * time.Second,
	})
}

// Default configurations

// WithDefaults applies sensible default configurations
func WithDefaults() Option {
	return func(c *Config) error {
		c.Timeout = 30 * time.Second
		c.MaxRetries = 3
		c.Async.Enabled = false
		c.Async.Workers = 4
		c.Logger.Level = "info"
		c.Logger.Format = "json"
		return nil
	}
}

// WithTestDefaults applies test-safe default configurations
func WithTestDefaults() Option {
	return func(c *Config) error {
		c.Timeout = 5 * time.Second
		c.MaxRetries = 1
		c.Async.Enabled = true
		c.Async.Workers = 2
		c.Logger.Level = "debug"
		c.Logger.Format = "text"
		return nil
	}
}

// Configuration validation

// Validate validates the configuration using basic validation
func (c *Config) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative")
	}

	// Validate platform configurations
	if c.Feishu != nil {
		if err := validateFeishuConfig(c.Feishu); err != nil {
			return fmt.Errorf("feishu config validation failed: %w", err)
		}
	}

	if c.Email != nil {
		if c.Email.Host == "" {
			return fmt.Errorf("email host is required")
		}
		if c.Email.Port <= 0 {
			return fmt.Errorf("email port must be positive")
		}
		if c.Email.From == "" {
			return fmt.Errorf("email from address is required")
		}
	}

	if c.Webhook != nil {
		if c.Webhook.URL == "" {
			return fmt.Errorf("webhook URL is required")
		}
	}

	// Validate async configuration
	if c.Async.Enabled && c.Async.Workers <= 0 {
		return fmt.Errorf("async workers must be positive when async is enabled")
	}

	return nil
}

// ValidateComprehensive performs comprehensive validation with detailed results
func (c *Config) ValidateComprehensive() error {
	// Validate core config
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", c.Timeout)
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative, got %d", c.MaxRetries)
	}

	// Validate platform configurations with basic checks
	if c.Feishu != nil {
		if c.Feishu.WebhookURL == "" {
			return fmt.Errorf("feishu webhook URL is required")
		}
	}

	if c.Email != nil {
		if c.Email.Host == "" {
			return fmt.Errorf("email host is required")
		}
	}

	if c.Webhook != nil {
		if c.Webhook.URL == "" {
			return fmt.Errorf("webhook URL is required")
		}
	}

	// Validate async configuration
	if c.Async.Enabled {
		if c.Async.Workers <= 0 {
			return fmt.Errorf("async workers must be positive when async is enabled, got %d", c.Async.Workers)
		}
	}

	return nil
}

// GetConfiguredPlatformCount returns the number of configured platforms
func (c *Config) GetConfiguredPlatformCount() int {
	count := 0
	if c.Feishu != nil {
		count++
	}
	if c.Email != nil {
		count++
	}
	if c.Webhook != nil {
		count++
	}
	return count
}

// GetValidationSummary returns a summary of the configuration
func (c *Config) GetValidationSummary() map[string]interface{} {
	summary := make(map[string]interface{})

	summary["platforms_configured"] = c.GetConfiguredPlatformCount()
	summary["async_enabled"] = c.Async.Enabled
	summary["timeout"] = c.Timeout.String()
	summary["max_retries"] = c.MaxRetries

	platforms := make([]string, 0)
	if c.Feishu != nil {
		platforms = append(platforms, "feishu")
	}
	if c.Email != nil {
		platforms = append(platforms, "email")
	}
	if c.Webhook != nil {
		platforms = append(platforms, "webhook")
	}
	summary["configured_platforms"] = platforms

	return summary
}

// IsAsyncEnabled returns whether async processing is enabled
func (c *Config) IsAsyncEnabled() bool {
	return c.Async.Enabled
}

// GetWorkers returns the number of async workers
func (c *Config) GetWorkers() int {
	return c.Async.Workers
}

// HasFeishu returns whether Feishu is configured
func (c *Config) HasFeishu() bool {
	return c.Feishu != nil
}

// HasEmail returns whether email is configured
func (c *Config) HasEmail() bool {
	return c.Email != nil
}

// HasWebhook returns whether webhook is configured
func (c *Config) HasWebhook() bool {
	return c.Webhook != nil
}
