// Package config provides functional options for unified configuration management
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// Core Configuration Options

// WithTimeout sets the default timeout for all operations
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *Config) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
		cfg.DefaultTimeout = timeout
		return nil
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(retries int) Option {
	return func(cfg *Config) error {
		if retries < 0 {
			return fmt.Errorf("max retries must be non-negative")
		}
		cfg.MaxRetries = retries
		return nil
	}
}

// WithMetrics enables metrics collection
func WithMetrics(enabled bool) Option {
	return func(cfg *Config) error {
		cfg.EnableMetrics = enabled
		return nil
	}
}

// WithTracing enables distributed tracing
func WithTracing(enabled bool) Option {
	return func(cfg *Config) error {
		cfg.EnableTracing = enabled
		return nil
	}
}

// WithLogger sets a custom logger instance
func WithLogger(logger logger.Logger) Option {
	return func(cfg *Config) error {
		cfg.Logger = logger
		return nil
	}
}

// Platform Configuration Options

// WithEmail configures the email platform with strong typing
func WithEmail(emailCfg EmailConfig) Option {
	return func(cfg *Config) error {
		cfg.Email = &emailCfg
		return nil
	}
}

// WithEmailBasic provides a simplified email configuration
func WithEmailBasic(host string, port int, from string) Option {
	return func(cfg *Config) error {
		cfg.Email = &EmailConfig{
			SMTPHost:   host,
			SMTPPort:   port,
			SMTPFrom:   from,
			SMTPTLS:    true, // Default to TLS
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			RateLimit:  60, // 60 emails per minute
		}
		return nil
	}
}

// WithGmailSMTP provides Gmail SMTP preset configuration
func WithGmailSMTP(username, password string) Option {
	return func(cfg *Config) error {
		cfg.Email = &EmailConfig{
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     587,
			SMTPUsername: username,
			SMTPPassword: password,
			SMTPFrom:     username,
			SMTPTLS:      true,
			SMTPSSL:      false,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
			RateLimit:    60,
		}
		return nil
	}
}

// WithFeishu configures the Feishu platform with strong typing
func WithFeishu(feishuCfg FeishuConfig) Option {
	return func(cfg *Config) error {
		cfg.Feishu = &feishuCfg
		return nil
	}
}

// WithFeishuWebhook provides a simplified Feishu webhook configuration
func WithFeishuWebhook(webhookURL string, secret ...string) Option {
	return func(cfg *Config) error {
		config := &FeishuConfig{
			WebhookURL: webhookURL,
			AuthType:   "webhook",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			RateLimit:  100, // 100 messages per minute
			SignVerify: false,
		}

		if len(secret) > 0 {
			config.Secret = secret[0]
			config.SignVerify = true
		}

		cfg.Feishu = config
		return nil
	}
}

// WithSMS configures the SMS platform with strong typing
func WithSMS(smsCfg SMSConfig) Option {
	return func(cfg *Config) error {
		cfg.SMS = &smsCfg
		return nil
	}
}

// WithSMSBasic provides a simplified SMS configuration
func WithSMSBasic(provider, apiKey, from string) Option {
	return func(cfg *Config) error {
		cfg.SMS = &SMSConfig{
			Provider:   provider,
			APIKey:     apiKey,
			From:       from,
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			RateLimit:  30, // 30 SMS per minute
			Templates:  make(map[string]string),
		}
		return nil
	}
}

// WithSlack configures the Slack platform with strong typing
func WithSlack(slackCfg SlackConfig) Option {
	return func(cfg *Config) error {
		cfg.Slack = &slackCfg
		return nil
	}
}

// WithSlackWebhook provides a simplified Slack webhook configuration
func WithSlackWebhook(webhookURL string, channel ...string) Option {
	return func(cfg *Config) error {
		config := &SlackConfig{
			WebhookURL:  webhookURL,
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			RateLimit:   100, // 100 messages per minute
			LinkNames:   true,
			UnfurlLinks: true,
			UnfurlMedia: true,
		}

		if len(channel) > 0 {
			config.Channel = channel[0]
		}

		cfg.Slack = config
		return nil
	}
}

// WithWebhook configures a generic webhook platform
func WithWebhook(webhookCfg WebhookConfig) Option {
	return func(cfg *Config) error {
		cfg.Webhook = &webhookCfg
		return nil
	}
}

// WithWebhookBasic provides a simplified webhook configuration
func WithWebhookBasic(url string) Option {
	return func(cfg *Config) error {
		cfg.Webhook = &WebhookConfig{
			URL:         url,
			Method:      "POST",
			Headers:     make(map[string]string),
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			RateLimit:   100, // 100 requests per minute
			VerifySSL:   true,
			ContentType: "application/json",
		}
		return nil
	}
}

// Async Configuration Options

// WithAsync enables asynchronous processing
func WithAsync(enabled bool) Option {
	return func(cfg *Config) error {
		cfg.AsyncConfig.Enabled = enabled
		return nil
	}
}

// WithAsyncConfig configures asynchronous processing with detailed options
func WithAsyncConfig(asyncCfg AsyncConfig) Option {
	return func(cfg *Config) error {
		cfg.AsyncConfig = asyncCfg
		return nil
	}
}

// WithAsyncBasic provides simplified async configuration
func WithAsyncBasic(queueType string, workers int) Option {
	return func(cfg *Config) error {
		cfg.AsyncConfig = AsyncConfig{
			Enabled:        true,
			QueueType:      queueType,
			QueueSize:      1000,
			Workers:        workers,
			BatchSize:      10,
			ProcessTimeout: 30 * time.Second,
		}
		return nil
	}
}

// Router Configuration Options

// WithRouter configures the smart routing system
func WithRouter(routerCfg RouterConfig) Option {
	return func(cfg *Config) error {
		cfg.RouterConfig = routerCfg
		return nil
	}
}

// WithRouterBasic provides simplified router configuration
func WithRouterBasic(loadBalancing string) Option {
	return func(cfg *Config) error {
		cfg.RouterConfig = RouterConfig{
			LoadBalancing:       loadBalancing,
			HealthCheckInterval: 30 * time.Second,
			MaxRetries:          3,
			RetryDelay:          time.Second,
			EnableFailover:      true,
		}
		return nil
	}
}

// Environment Variable Options

// WithEnvDefaults loads configuration from environment variables
func WithEnvDefaults() Option {
	return func(cfg *Config) error {
		// Core settings from environment
		if timeout := os.Getenv("NOTIFYHUB_TIMEOUT"); timeout != "" {
			if d, err := time.ParseDuration(timeout); err == nil {
				cfg.DefaultTimeout = d
			}
		}

		if metrics := os.Getenv("NOTIFYHUB_ENABLE_METRICS"); metrics == "true" {
			cfg.EnableMetrics = true
		}

		if tracing := os.Getenv("NOTIFYHUB_ENABLE_TRACING"); tracing == "true" {
			cfg.EnableTracing = true
		}

		// Email from environment
		if host := os.Getenv("NOTIFYHUB_EMAIL_HOST"); host != "" {
			if cfg.Email == nil {
				cfg.Email = &EmailConfig{}
			}
			cfg.Email.SMTPHost = host

			if port := os.Getenv("NOTIFYHUB_EMAIL_PORT"); port != "" {
				// Simple port parsing (production code should handle errors)
				switch port {
				case "587":
					cfg.Email.SMTPPort = 587
				case "465":
					cfg.Email.SMTPPort = 465
				case "25":
					cfg.Email.SMTPPort = 25
				}
			}

			if from := os.Getenv("NOTIFYHUB_EMAIL_FROM"); from != "" {
				cfg.Email.SMTPFrom = from
			}

			if username := os.Getenv("NOTIFYHUB_EMAIL_USERNAME"); username != "" {
				cfg.Email.SMTPUsername = username
			}

			if password := os.Getenv("NOTIFYHUB_EMAIL_PASSWORD"); password != "" {
				cfg.Email.SMTPPassword = password
			}
		}

		// Feishu from environment
		if webhookURL := os.Getenv("NOTIFYHUB_FEISHU_WEBHOOK"); webhookURL != "" {
			if cfg.Feishu == nil {
				cfg.Feishu = &FeishuConfig{}
			}
			cfg.Feishu.WebhookURL = webhookURL
			cfg.Feishu.AuthType = "webhook"

			if secret := os.Getenv("NOTIFYHUB_FEISHU_SECRET"); secret != "" {
				cfg.Feishu.Secret = secret
				cfg.Feishu.SignVerify = true
			}
		}

		// Async from environment
		if enabled := os.Getenv("NOTIFYHUB_ASYNC_ENABLED"); enabled == "true" {
			cfg.AsyncConfig.Enabled = true
		}

		if queueType := os.Getenv("NOTIFYHUB_QUEUE_TYPE"); queueType != "" {
			cfg.AsyncConfig.QueueType = queueType
		}

		return nil
	}
}

// Legacy Support Options (for backward compatibility)

// WithPlatform adds a platform using the legacy map-based approach
// This maintains backward compatibility while encouraging migration to strong types
func WithPlatform(name string, config map[string]interface{}) Option {
	return func(cfg *Config) error {
		if cfg.Platforms == nil {
			cfg.Platforms = make(map[string]map[string]interface{})
		}
		cfg.Platforms[name] = config
		return nil
	}
}

// Utility Options

// WithDefaults applies sensible default configurations
func WithDefaults() Option {
	return func(cfg *Config) error {
		// Apply multiple default configurations
		defaults := []Option{
			WithTimeout(30 * time.Second),
			WithMaxRetries(3),
			WithRouterBasic("round_robin"),
			WithAsyncBasic("memory", 4),
		}

		for _, opt := range defaults {
			if err := opt(cfg); err != nil {
				return err
			}
		}

		return nil
	}
}

// WithTestDefaults provides safe defaults for testing
func WithTestDefaults() Option {
	return func(cfg *Config) error {
		// Configure with test-safe defaults
		testDefaults := []Option{
			WithTimeout(5 * time.Second),
			WithMaxRetries(1),
			WithMetrics(false),
			WithTracing(false),
			WithAsyncBasic("memory", 2),
		}

		for _, opt := range testDefaults {
			if err := opt(cfg); err != nil {
				return err
			}
		}

		return nil
	}
}

// WithValidationSkip skips validation during configuration creation
// This is useful for testing scenarios where partial configuration is needed
func WithValidationSkip() Option {
	return func(cfg *Config) error {
		// This is a marker option that NewConfigWithValidation can check
		// The actual skip logic is handled in a separate constructor function
		return nil
	}
}

// Additional Configuration Options for NotifyHub Integration

// WithRetryPolicy configures the global retry policy
func WithRetryPolicy(maxRetries int, initialInterval int, multiplier float64) Option {
	return func(cfg *Config) error {
		cfg.RetryPolicy = RetryPolicy{
			MaxRetries:      maxRetries,
			InitialInterval: time.Duration(initialInterval) * time.Millisecond,
			Multiplier:      multiplier,
			MaxInterval:     30 * time.Second, // Default max interval
		}
		return nil
	}
}
