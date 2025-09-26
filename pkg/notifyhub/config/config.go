// Package config provides unified configuration management for NotifyHub
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/kart-io/notifyhub/pkg/logger"
)

// Config represents the unified configuration for NotifyHub
// This replaces the type-unsafe map[string]interface{} approach identified in architecture analysis
type Config struct {
	// Core settings
	DefaultTimeout time.Duration `json:"default_timeout" yaml:"default_timeout" validate:"min=1s"`
	MaxRetries     int           `json:"max_retries" yaml:"max_retries" validate:"min=0,max=10"`
	EnableMetrics  bool          `json:"enable_metrics" yaml:"enable_metrics"`
	EnableTracing  bool          `json:"enable_tracing" yaml:"enable_tracing"`

	// Platform configurations (strongly typed)
	Email   *EmailConfig   `json:"email,omitempty" yaml:"email,omitempty"`
	Feishu  *FeishuConfig  `json:"feishu,omitempty" yaml:"feishu,omitempty"`
	SMS     *SMSConfig     `json:"sms,omitempty" yaml:"sms,omitempty"`
	Slack   *SlackConfig   `json:"slack,omitempty" yaml:"slack,omitempty"`
	Webhook *WebhookConfig `json:"webhook,omitempty" yaml:"webhook,omitempty"`

	// Legacy platform support (for backward compatibility)
	Platforms map[string]map[string]interface{} `json:"platforms,omitempty" yaml:"platforms,omitempty"`

	// Subsystem configurations
	RetryPolicy  RetryPolicy  `json:"retry_policy" yaml:"retry_policy"`
	AsyncConfig  AsyncConfig  `json:"async_config" yaml:"async_config"`
	RouterConfig RouterConfig `json:"router_config" yaml:"router_config"`

	// Runtime fields
	Logger           logger.Logger `json:"-" yaml:"-"` // Logger instance (excluded from serialization)
	ValidationErrors []error       `json:"-" yaml:"-"` // Collect validation errors during configuration
}

// AsyncConfig represents asynchronous processing configuration
type AsyncConfig struct {
	Enabled        bool          `json:"enabled" yaml:"enabled"`
	QueueType      string        `json:"queue_type" yaml:"queue_type" validate:"oneof=memory redis"`
	QueueSize      int           `json:"queue_size" yaml:"queue_size" validate:"min=1,max=100000"`
	Workers        int           `json:"workers" yaml:"workers" validate:"min=1,max=100"`
	BatchSize      int           `json:"batch_size" yaml:"batch_size" validate:"min=1,max=1000"`
	ProcessTimeout time.Duration `json:"process_timeout" yaml:"process_timeout" validate:"min=1s"`
}

// RouterConfig represents intelligent routing configuration
type RouterConfig struct {
	LoadBalancing       string        `json:"load_balancing" yaml:"load_balancing" validate:"oneof=round_robin random weighted"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval" validate:"min=5s"`
	MaxRetries          int           `json:"max_retries" yaml:"max_retries" validate:"min=1,max=10"`
	RetryDelay          time.Duration `json:"retry_delay" yaml:"retry_delay" validate:"min=100ms"`
	EnableFailover      bool          `json:"enable_failover" yaml:"enable_failover"`
}

// RetryPolicy defines retry behavior for failed operations
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries" yaml:"max_retries" validate:"min=1,max=10"`
	InitialInterval time.Duration `json:"initial_interval" yaml:"initial_interval" validate:"min=100ms"`
	Multiplier      float64       `json:"multiplier" yaml:"multiplier" validate:"min=1.1,max=10"`
	MaxInterval     time.Duration `json:"max_interval" yaml:"max_interval" validate:"min=1s"`
}

// Option represents a configuration option function
type Option func(*Config) error

// NewConfig creates a new configuration with the given options
func NewConfig(opts ...Option) (*Config, error) {
	cfg := &Config{
		// Core defaults
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
		EnableMetrics:  false,
		EnableTracing:  false,

		// Initialize legacy platforms map for backward compatibility
		Platforms: make(map[string]map[string]interface{}),

		// Default retry policy
		RetryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			Multiplier:      2.0,
			MaxInterval:     30 * time.Second,
		},

		// Default async config
		AsyncConfig: AsyncConfig{
			Enabled:        false,
			QueueType:      "memory",
			QueueSize:      1000,
			Workers:        4,
			BatchSize:      10,
			ProcessTimeout: 30 * time.Second,
		},

		// Default router config
		RouterConfig: RouterConfig{
			LoadBalancing:       "round_robin",
			HealthCheckInterval: 30 * time.Second,
			MaxRetries:          3,
			RetryDelay:          1 * time.Second,
			EnableFailover:      true,
		},
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Automatically validate the configuration after applying all options
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// NewConfigWithoutValidation creates a new configuration with the given options but skips validation
// This is useful for testing scenarios where partial configuration is acceptable
func NewConfigWithoutValidation(opts ...Option) (*Config, error) {
	cfg := &Config{
		// Core defaults
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
		EnableMetrics:  false,
		EnableTracing:  false,

		// Initialize legacy platforms map for backward compatibility
		Platforms: make(map[string]map[string]interface{}),

		// Default retry policy
		RetryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			Multiplier:      2.0,
			MaxInterval:     30 * time.Second,
		},

		// Default async config
		AsyncConfig: AsyncConfig{
			Enabled:        false,
			QueueType:      "memory",
			QueueSize:      1000,
			Workers:        4,
			BatchSize:      10,
			ProcessTimeout: 30 * time.Second,
		},

		// Default router config
		RouterConfig: RouterConfig{
			LoadBalancing:       "round_robin",
			HealthCheckInterval: 30 * time.Second,
			MaxRetries:          3,
			RetryDelay:          1 * time.Second,
			EnableFailover:      true,
		},
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Skip validation for testing purposes
	return cfg, nil
}

// IsAsyncEnabled returns whether async processing is enabled
func (c *Config) IsAsyncEnabled() bool {
	return c.AsyncConfig.Enabled
}

// Workers returns the number of workers for async processing
func (c *Config) Workers() int {
	return c.AsyncConfig.Workers
}

// Platform Configuration Types (Strong-typed replacements for map[string]interface{})

// EmailConfig represents email platform configuration
type EmailConfig struct {
	SMTPHost     string        `json:"smtp_host" yaml:"smtp_host" validate:"required,hostname"`
	SMTPPort     int           `json:"smtp_port" yaml:"smtp_port" validate:"required,min=1,max=65535"`
	SMTPUsername string        `json:"smtp_username" yaml:"smtp_username"`
	SMTPPassword string        `json:"smtp_password" yaml:"smtp_password"`
	SMTPFrom     string        `json:"smtp_from" yaml:"smtp_from" validate:"required,email"`
	SMTPTLS      bool          `json:"smtp_tls" yaml:"smtp_tls"`
	SMTPSSL      bool          `json:"smtp_ssl" yaml:"smtp_ssl"`
	Timeout      time.Duration `json:"timeout" yaml:"timeout" validate:"min=1s"`
	MaxRetries   int           `json:"max_retries" yaml:"max_retries" validate:"min=0,max=10"`
	RateLimit    int           `json:"rate_limit" yaml:"rate_limit" validate:"min=0"` // emails per minute
}

// FeishuConfig represents Feishu platform configuration
type FeishuConfig struct {
	WebhookURL string        `json:"webhook_url" yaml:"webhook_url" validate:"required,url"`
	Secret     string        `json:"secret" yaml:"secret"`
	AppID      string        `json:"app_id" yaml:"app_id"`
	AppSecret  string        `json:"app_secret" yaml:"app_secret"`
	AuthType   string        `json:"auth_type" yaml:"auth_type" validate:"oneof=webhook app"`
	Timeout    time.Duration `json:"timeout" yaml:"timeout" validate:"min=1s"`
	MaxRetries int           `json:"max_retries" yaml:"max_retries" validate:"min=0,max=10"`
	RateLimit  int           `json:"rate_limit" yaml:"rate_limit" validate:"min=0"` // messages per minute
	SignVerify bool          `json:"sign_verify" yaml:"sign_verify"`
	Keywords   []string      `json:"keywords" yaml:"keywords"` // 自定义关键词列表
}

// SMSConfig represents SMS platform configuration
type SMSConfig struct {
	Provider   string            `json:"provider" yaml:"provider" validate:"required,oneof=twilio aliyun tencent"`
	APIKey     string            `json:"api_key" yaml:"api_key" validate:"required"`
	APISecret  string            `json:"api_secret" yaml:"api_secret"`
	From       string            `json:"from" yaml:"from" validate:"required"`
	Region     string            `json:"region" yaml:"region"`
	Timeout    time.Duration     `json:"timeout" yaml:"timeout" validate:"min=1s"`
	MaxRetries int               `json:"max_retries" yaml:"max_retries" validate:"min=0,max=10"`
	RateLimit  int               `json:"rate_limit" yaml:"rate_limit" validate:"min=0"` // SMS per minute
	Templates  map[string]string `json:"templates" yaml:"templates"`                    // template ID mappings
}

// SlackConfig represents Slack platform configuration
type SlackConfig struct {
	WebhookURL  string        `json:"webhook_url" yaml:"webhook_url" validate:"required,url"`
	Token       string        `json:"token" yaml:"token"`
	Channel     string        `json:"channel" yaml:"channel"`
	Username    string        `json:"username" yaml:"username"`
	IconURL     string        `json:"icon_url" yaml:"icon_url" validate:"omitempty,url"`
	IconEmoji   string        `json:"icon_emoji" yaml:"icon_emoji"`
	LinkNames   bool          `json:"link_names" yaml:"link_names"`
	UnfurlLinks bool          `json:"unfurl_links" yaml:"unfurl_links"`
	UnfurlMedia bool          `json:"unfurl_media" yaml:"unfurl_media"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout" validate:"min=1s"`
	MaxRetries  int           `json:"max_retries" yaml:"max_retries" validate:"min=0,max=10"`
	RateLimit   int           `json:"rate_limit" yaml:"rate_limit" validate:"min=0"` // messages per minute
}

// WebhookConfig represents generic webhook platform configuration
type WebhookConfig struct {
	URL         string            `json:"url" yaml:"url" validate:"required,url"`
	Method      string            `json:"method" yaml:"method" validate:"oneof=POST PUT PATCH"`
	Headers     map[string]string `json:"headers" yaml:"headers"`
	AuthType    string            `json:"auth_type" yaml:"auth_type" validate:"oneof= basic bearer custom"`
	AuthValue   string            `json:"auth_value" yaml:"auth_value"`
	Timeout     time.Duration     `json:"timeout" yaml:"timeout" validate:"min=1s"`
	MaxRetries  int               `json:"max_retries" yaml:"max_retries" validate:"min=0,max=10"`
	RateLimit   int               `json:"rate_limit" yaml:"rate_limit" validate:"min=0"` // requests per minute
	VerifySSL   bool              `json:"verify_ssl" yaml:"verify_ssl"`
	ContentType string            `json:"content_type" yaml:"content_type"`
}

// GetPlatformConfig returns configuration for a specific platform (legacy support)
// This method provides backward compatibility while encouraging migration to strong types
func (c *Config) GetPlatformConfig(platformName string) map[string]interface{} {
	// First check if we have a strong-typed config and convert it
	switch platformName {
	case "email":
		if c.Email != nil {
			return c.emailToMap()
		}
	case "feishu":
		if c.Feishu != nil {
			return c.feishuToMap()
		}
	case "sms":
		if c.SMS != nil {
			return c.smsToMap()
		}
	case "slack":
		if c.Slack != nil {
			return c.slackToMap()
		}
	case "webhook":
		if c.Webhook != nil {
			return c.webhookToMap()
		}
	}

	// Fallback to legacy platforms map
	return c.Platforms[platformName]
}

// Helper methods to convert strong-typed configs to maps (for backward compatibility)
func (c *Config) emailToMap() map[string]interface{} {
	return map[string]interface{}{
		"smtp_host":     c.Email.SMTPHost,
		"smtp_port":     c.Email.SMTPPort,
		"smtp_username": c.Email.SMTPUsername,
		"smtp_password": c.Email.SMTPPassword,
		"smtp_from":     c.Email.SMTPFrom,
		"smtp_tls":      c.Email.SMTPTLS,
		"smtp_ssl":      c.Email.SMTPSSL,
		"timeout":       c.Email.Timeout,
		"max_retries":   c.Email.MaxRetries,
		"rate_limit":    c.Email.RateLimit,
	}
}

func (c *Config) feishuToMap() map[string]interface{} {
	return map[string]interface{}{
		"webhook_url": c.Feishu.WebhookURL,
		"secret":      c.Feishu.Secret,
		"app_id":      c.Feishu.AppID,
		"app_secret":  c.Feishu.AppSecret,
		"auth_type":   c.Feishu.AuthType,
		"timeout":     c.Feishu.Timeout,
		"max_retries": c.Feishu.MaxRetries,
		"rate_limit":  c.Feishu.RateLimit,
		"sign_verify": c.Feishu.SignVerify,
	}
}

func (c *Config) smsToMap() map[string]interface{} {
	return map[string]interface{}{
		"provider":    c.SMS.Provider,
		"api_key":     c.SMS.APIKey,
		"api_secret":  c.SMS.APISecret,
		"from":        c.SMS.From,
		"region":      c.SMS.Region,
		"timeout":     c.SMS.Timeout,
		"max_retries": c.SMS.MaxRetries,
		"rate_limit":  c.SMS.RateLimit,
		"templates":   c.SMS.Templates,
	}
}

func (c *Config) slackToMap() map[string]interface{} {
	return map[string]interface{}{
		"webhook_url":  c.Slack.WebhookURL,
		"token":        c.Slack.Token,
		"channel":      c.Slack.Channel,
		"username":     c.Slack.Username,
		"icon_url":     c.Slack.IconURL,
		"icon_emoji":   c.Slack.IconEmoji,
		"link_names":   c.Slack.LinkNames,
		"unfurl_links": c.Slack.UnfurlLinks,
		"unfurl_media": c.Slack.UnfurlMedia,
		"timeout":      c.Slack.Timeout,
		"max_retries":  c.Slack.MaxRetries,
		"rate_limit":   c.Slack.RateLimit,
	}
}

func (c *Config) webhookToMap() map[string]interface{} {
	return map[string]interface{}{
		"url":          c.Webhook.URL,
		"method":       c.Webhook.Method,
		"headers":      c.Webhook.Headers,
		"auth_type":    c.Webhook.AuthType,
		"auth_value":   c.Webhook.AuthValue,
		"timeout":      c.Webhook.Timeout,
		"max_retries":  c.Webhook.MaxRetries,
		"rate_limit":   c.Webhook.RateLimit,
		"verify_ssl":   c.Webhook.VerifySSL,
		"content_type": c.Webhook.ContentType,
	}
}

// Package-level validator instance
var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Validate validates the entire configuration using struct tags and custom logic
func (c *Config) Validate() error {
	var errors []error

	// Validate main config struct using tags
	if err := validate.Struct(c); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			for _, err := range validationErrors {
				errors = append(errors, fmt.Errorf("field %s: %s", err.Field(), getValidationMessage(err)))
			}
		} else {
			errors = append(errors, fmt.Errorf("validation error: %w", err))
		}
	}

	// Validate platform-specific configurations
	if c.Email != nil {
		if err := validate.Struct(c.Email); err != nil {
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				for _, err := range validationErrors {
					errors = append(errors, fmt.Errorf("email config field %s: %s", err.Field(), getValidationMessage(err)))
				}
			}
		}
	}

	if c.Feishu != nil {
		if err := validate.Struct(c.Feishu); err != nil {
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				for _, err := range validationErrors {
					errors = append(errors, fmt.Errorf("feishu config field %s: %s", err.Field(), getValidationMessage(err)))
				}
			}
		}
	}

	if c.SMS != nil {
		if err := validate.Struct(c.SMS); err != nil {
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				for _, err := range validationErrors {
					errors = append(errors, fmt.Errorf("sms config field %s: %s", err.Field(), getValidationMessage(err)))
				}
			}
		}
	}

	if c.Slack != nil {
		if err := validate.Struct(c.Slack); err != nil {
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				for _, err := range validationErrors {
					errors = append(errors, fmt.Errorf("slack config field %s: %s", err.Field(), getValidationMessage(err)))
				}
			}
		}
	}

	if c.Webhook != nil {
		if err := validate.Struct(c.Webhook); err != nil {
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				for _, err := range validationErrors {
					errors = append(errors, fmt.Errorf("webhook config field %s: %s", err.Field(), getValidationMessage(err)))
				}
			}
		}
	}

	// Additional custom validation logic
	if c.AsyncConfig.Enabled && c.AsyncConfig.Workers <= 0 {
		errors = append(errors, fmt.Errorf("async workers must be positive when async is enabled"))
	}

	if c.RouterConfig.MaxRetries > 10 {
		errors = append(errors, fmt.Errorf("router max_retries should not exceed 10 for performance reasons"))
	}

	// Store validation errors for later inspection
	c.ValidationErrors = errors

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed with %d errors: %s", len(errors),
			strings.Join(getErrorMessages(errors), "; "))
	}

	return nil
}

// getValidationMessage converts validator errors to human-readable messages
func getValidationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	case "hostname":
		return "must be a valid hostname"
	case "min":
		return fmt.Sprintf("must be at least %s", err.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", err.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", err.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", err.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", err.Param())
	default:
		return fmt.Sprintf("failed validation: %s", err.Tag())
	}
}

// getErrorMessages extracts messages from error slice
func getErrorMessages(errors []error) []string {
	messages := make([]string, len(errors))
	for i, err := range errors {
		messages[i] = err.Error()
	}
	return messages
}
