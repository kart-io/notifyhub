package feishu

import (
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
)

// ConfigOption represents a configuration option function
type ConfigOption func(*config.FeishuConfig) error

// WithWebhookURL sets the webhook URL option
func WithWebhookURL(url string) ConfigOption {
	return func(cfg *config.FeishuConfig) error {
		if url == "" {
			return fmt.Errorf("webhook URL cannot be empty")
		}
		cfg.WebhookURL = url
		cfg.AuthType = "webhook"
		return nil
	}
}

// WithSecret sets the webhook secret option
func WithSecret(secret string) ConfigOption {
	return func(cfg *config.FeishuConfig) error {
		cfg.Secret = secret
		return nil
	}
}

// WithApp sets the app ID and secret for app-based authentication
func WithApp(appID, appSecret string) ConfigOption {
	return func(cfg *config.FeishuConfig) error {
		if appID == "" {
			return fmt.Errorf("app ID cannot be empty")
		}
		if appSecret == "" {
			return fmt.Errorf("app secret cannot be empty")
		}
		cfg.AppID = appID
		cfg.AppSecret = appSecret
		cfg.AuthType = "app"
		return nil
	}
}

// WithKeywords sets the keywords option
func WithKeywords(keywords ...string) ConfigOption {
	return func(cfg *config.FeishuConfig) error {
		cfg.Keywords = keywords
		return nil
	}
}

// WithTimeout sets the timeout option
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(cfg *config.FeishuConfig) error {
		if timeout < 0 {
			return fmt.Errorf("timeout cannot be negative")
		}
		cfg.Timeout = timeout
		return nil
	}
}

// WithMaxRetries sets the max retries option
func WithMaxRetries(maxRetries int) ConfigOption {
	return func(cfg *config.FeishuConfig) error {
		if maxRetries < 0 {
			return fmt.Errorf("max retries cannot be negative")
		}
		cfg.MaxRetries = maxRetries
		return nil
	}
}

// WithRateLimit sets the rate limit option
func WithRateLimit(rateLimit int) ConfigOption {
	return func(cfg *config.FeishuConfig) error {
		if rateLimit < 0 {
			return fmt.Errorf("rate limit cannot be negative")
		}
		cfg.RateLimit = rateLimit
		return nil
	}
}

// WithSignVerify enables or disables signature verification
func WithSignVerify(enabled bool) ConfigOption {
	return func(cfg *config.FeishuConfig) error {
		cfg.SignVerify = enabled
		return nil
	}
}

// NewConfig creates a new FeishuConfig with options
func NewConfig(options ...ConfigOption) (*config.FeishuConfig, error) {
	cfg := &config.FeishuConfig{}

	// Apply options
	for _, option := range options {
		if err := option(cfg); err != nil {
			return nil, err
		}
	}

	// Set defaults and validate
	SetDefaults(cfg)
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}