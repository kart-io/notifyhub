// Package config tests for functional options and option combination scenarios
package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicOptions tests individual option functions
func TestBasicOptions(t *testing.T) {
	tests := []struct {
		name      string
		option    Option
		wantError bool
		validate  func(*testing.T, *Config)
	}{
		{
			name:   "WithTimeout valid",
			option: WithTimeout(60 * time.Second),
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 60*time.Second, cfg.DefaultTimeout)
			},
		},
		{
			name:      "WithTimeout invalid - negative",
			option:    WithTimeout(-5 * time.Second),
			wantError: true,
		},
		{
			name:      "WithTimeout invalid - zero",
			option:    WithTimeout(0),
			wantError: true,
		},
		{
			name:   "WithMaxRetries valid",
			option: WithMaxRetries(5),
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 5, cfg.MaxRetries)
			},
		},
		{
			name:   "WithMaxRetries zero (valid)",
			option: WithMaxRetries(0),
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 0, cfg.MaxRetries)
			},
		},
		{
			name:      "WithMaxRetries invalid - negative",
			option:    WithMaxRetries(-1),
			wantError: true,
		},
		{
			name:   "WithMetrics enabled",
			option: WithMetrics(true),
			validate: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.EnableMetrics)
			},
		},
		{
			name:   "WithMetrics disabled",
			option: WithMetrics(false),
			validate: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.EnableMetrics)
			},
		},
		{
			name:   "WithTracing enabled",
			option: WithTracing(true),
			validate: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.EnableTracing)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DefaultTimeout: 30 * time.Second,
				MaxRetries:     3,
				Platforms:      make(map[string]map[string]interface{}),
			}

			err := tt.option(cfg)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

// TestPlatformOptions tests platform-specific configuration options
func TestPlatformOptions(t *testing.T) {
	t.Run("WithEmail full config", func(t *testing.T) {
		emailCfg := EmailConfig{
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password123",
			SMTPFrom:     "test@example.com",
			SMTPTLS:      true,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
			RateLimit:    60,
		}

		cfg := &Config{Platforms: make(map[string]map[string]interface{})}
		err := WithEmail(emailCfg)(cfg)

		assert.NoError(t, err)
		require.NotNil(t, cfg.Email)
		assert.Equal(t, emailCfg, *cfg.Email)
	})

	t.Run("WithEmailBasic", func(t *testing.T) {
		cfg := &Config{Platforms: make(map[string]map[string]interface{})}
		err := WithEmailBasic("smtp.example.com", 587, "from@example.com")(cfg)

		assert.NoError(t, err)
		require.NotNil(t, cfg.Email)
		assert.Equal(t, "smtp.example.com", cfg.Email.SMTPHost)
		assert.Equal(t, 587, cfg.Email.SMTPPort)
		assert.Equal(t, "from@example.com", cfg.Email.SMTPFrom)
		assert.True(t, cfg.Email.SMTPTLS)
		assert.Equal(t, 30*time.Second, cfg.Email.Timeout)
	})

	t.Run("WithGmailSMTP", func(t *testing.T) {
		cfg := &Config{Platforms: make(map[string]map[string]interface{})}
		err := WithGmailSMTP("user@gmail.com", "app-password")(cfg)

		assert.NoError(t, err)
		require.NotNil(t, cfg.Email)
		assert.Equal(t, "smtp.gmail.com", cfg.Email.SMTPHost)
		assert.Equal(t, 587, cfg.Email.SMTPPort)
		assert.Equal(t, "user@gmail.com", cfg.Email.SMTPUsername)
		assert.Equal(t, "app-password", cfg.Email.SMTPPassword)
		assert.Equal(t, "user@gmail.com", cfg.Email.SMTPFrom)
		assert.True(t, cfg.Email.SMTPTLS)
		assert.False(t, cfg.Email.SMTPSSL)
	})

	t.Run("WithFeishu full config", func(t *testing.T) {
		feishuCfg := FeishuConfig{
			WebhookURL: "https://open.feishu.cn/webhook/test",
			Secret:     "secret123",
			AuthType:   "webhook",
			Timeout:    20 * time.Second,
			MaxRetries: 2,
			RateLimit:  100,
			Keywords:   []string{"alert", "error"},
		}

		cfg := &Config{Platforms: make(map[string]map[string]interface{})}
		err := WithFeishu(feishuCfg)(cfg)

		assert.NoError(t, err)
		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, feishuCfg, *cfg.Feishu)
	})

	t.Run("WithFeishuWebhook without secret", func(t *testing.T) {
		cfg := &Config{Platforms: make(map[string]map[string]interface{})}
		err := WithFeishuWebhook("https://open.feishu.cn/webhook/test")(cfg)

		assert.NoError(t, err)
		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, "https://open.feishu.cn/webhook/test", cfg.Feishu.WebhookURL)
		assert.Equal(t, "webhook", cfg.Feishu.AuthType)
		assert.Empty(t, cfg.Feishu.Secret)
		assert.False(t, cfg.Feishu.SignVerify)
	})

	t.Run("WithFeishuWebhook with secret", func(t *testing.T) {
		cfg := &Config{Platforms: make(map[string]map[string]interface{})}
		err := WithFeishuWebhook("https://open.feishu.cn/webhook/test", "secret123")(cfg)

		assert.NoError(t, err)
		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, "secret123", cfg.Feishu.Secret)
		assert.True(t, cfg.Feishu.SignVerify)
	})

	t.Run("WithSlackWebhook without channel", func(t *testing.T) {
		cfg := &Config{Platforms: make(map[string]map[string]interface{})}
		err := WithSlackWebhook("https://hooks.slack.com/webhook")(cfg)

		assert.NoError(t, err)
		require.NotNil(t, cfg.Slack)
		assert.Equal(t, "https://hooks.slack.com/webhook", cfg.Slack.WebhookURL)
		assert.Empty(t, cfg.Slack.Channel)
		assert.True(t, cfg.Slack.LinkNames)
	})

	t.Run("WithSlackWebhook with channel", func(t *testing.T) {
		cfg := &Config{Platforms: make(map[string]map[string]interface{})}
		err := WithSlackWebhook("https://hooks.slack.com/webhook", "#alerts")(cfg)

		assert.NoError(t, err)
		require.NotNil(t, cfg.Slack)
		assert.Equal(t, "#alerts", cfg.Slack.Channel)
	})

	t.Run("WithWebhookBasic", func(t *testing.T) {
		cfg := &Config{Platforms: make(map[string]map[string]interface{})}
		err := WithWebhookBasic("https://api.example.com/webhook")(cfg)

		assert.NoError(t, err)
		require.NotNil(t, cfg.Webhook)
		assert.Equal(t, "https://api.example.com/webhook", cfg.Webhook.URL)
		assert.Equal(t, "POST", cfg.Webhook.Method)
		assert.True(t, cfg.Webhook.VerifySSL)
		assert.Equal(t, "application/json", cfg.Webhook.ContentType)
	})
}

// TestAsyncOptions tests asynchronous processing configuration options
func TestAsyncOptions(t *testing.T) {
	t.Run("WithAsync", func(t *testing.T) {
		cfg := &Config{
			AsyncConfig: AsyncConfig{Enabled: false},
		}
		err := WithAsync(true)(cfg)

		assert.NoError(t, err)
		assert.True(t, cfg.AsyncConfig.Enabled)
	})

	t.Run("WithAsyncConfig", func(t *testing.T) {
		asyncCfg := AsyncConfig{
			Enabled:        true,
			QueueType:      "redis",
			QueueSize:      2000,
			Workers:        8,
			BatchSize:      20,
			ProcessTimeout: 60 * time.Second,
		}

		cfg := &Config{}
		err := WithAsyncConfig(asyncCfg)(cfg)

		assert.NoError(t, err)
		assert.Equal(t, asyncCfg, cfg.AsyncConfig)
	})

	t.Run("WithAsyncBasic", func(t *testing.T) {
		cfg := &Config{}
		err := WithAsyncBasic("redis", 6)(cfg)

		assert.NoError(t, err)
		assert.True(t, cfg.AsyncConfig.Enabled)
		assert.Equal(t, "redis", cfg.AsyncConfig.QueueType)
		assert.Equal(t, 6, cfg.AsyncConfig.Workers)
		assert.Equal(t, 1000, cfg.AsyncConfig.QueueSize)
		assert.Equal(t, 10, cfg.AsyncConfig.BatchSize)
	})
}

// TestRouterOptions tests router configuration options
func TestRouterOptions(t *testing.T) {
	t.Run("WithRouter", func(t *testing.T) {
		routerCfg := RouterConfig{
			LoadBalancing:       "weighted",
			HealthCheckInterval: 60 * time.Second,
			MaxRetries:          5,
			RetryDelay:          2 * time.Second,
			EnableFailover:      false,
		}

		cfg := &Config{}
		err := WithRouter(routerCfg)(cfg)

		assert.NoError(t, err)
		assert.Equal(t, routerCfg, cfg.RouterConfig)
	})

	t.Run("WithRouterBasic", func(t *testing.T) {
		cfg := &Config{}
		err := WithRouterBasic("random")(cfg)

		assert.NoError(t, err)
		assert.Equal(t, "random", cfg.RouterConfig.LoadBalancing)
		assert.Equal(t, 30*time.Second, cfg.RouterConfig.HealthCheckInterval)
		assert.Equal(t, 3, cfg.RouterConfig.MaxRetries)
		assert.True(t, cfg.RouterConfig.EnableFailover)
	})
}

// TestOptionCombinations tests various combinations of options
func TestOptionCombinations(t *testing.T) {
	t.Run("multiple platform configurations", func(t *testing.T) {
		opts := []Option{
			WithFeishuWebhook("https://feishu.webhook", "secret1"),
			WithEmailBasic("smtp.gmail.com", 587, "test@example.com"),
			WithSlackWebhook("https://slack.webhook", "#general"),
			WithWebhookBasic("https://api.webhook"),
		}

		cfg, err := NewConfigWithoutValidation(opts...)
		assert.NoError(t, err)

		assert.NotNil(t, cfg.Feishu)
		assert.NotNil(t, cfg.Email)
		assert.NotNil(t, cfg.Slack)
		assert.NotNil(t, cfg.Webhook)
	})

	t.Run("conflicting timeout settings", func(t *testing.T) {
		// Later options should override earlier ones
		opts := []Option{
			WithTimeout(30 * time.Second),
			WithTimeout(60 * time.Second), // This should win
			WithMaxRetries(3),
			WithMaxRetries(5), // This should win
		}

		cfg, err := NewConfigWithoutValidation(opts...)
		assert.NoError(t, err)

		assert.Equal(t, 60*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 5, cfg.MaxRetries)
	})

	t.Run("complex configuration scenario", func(t *testing.T) {
		opts := []Option{
			WithTimeout(45 * time.Second),
			WithMaxRetries(4),
			WithMetrics(true),
			WithTracing(true),
			WithFeishuWebhook("https://feishu.webhook", "secret"),
			WithEmailBasic("smtp.example.com", 587, "from@example.com"),
			WithAsyncBasic("redis", 8),
			WithRouterBasic("weighted"),
			WithRetryPolicy(5, 1000, 2.0),
		}

		cfg, err := NewConfigWithoutValidation(opts...)
		assert.NoError(t, err)

		// Verify core settings
		assert.Equal(t, 45*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 4, cfg.MaxRetries)
		assert.True(t, cfg.EnableMetrics)
		assert.True(t, cfg.EnableTracing)

		// Verify platforms
		assert.NotNil(t, cfg.Feishu)
		assert.NotNil(t, cfg.Email)

		// Verify subsystems
		assert.True(t, cfg.AsyncConfig.Enabled)
		assert.Equal(t, "redis", cfg.AsyncConfig.QueueType)
		assert.Equal(t, 8, cfg.AsyncConfig.Workers)
		assert.Equal(t, "weighted", cfg.RouterConfig.LoadBalancing)
		assert.Equal(t, 5, cfg.RetryPolicy.MaxRetries)
	})

	t.Run("option order independence", func(t *testing.T) {
		// Different order should produce same result
		opts1 := []Option{
			WithTimeout(30 * time.Second),
			WithFeishuWebhook("https://feishu.webhook"),
			WithAsyncBasic("memory", 4),
		}

		opts2 := []Option{
			WithAsyncBasic("memory", 4),
			WithFeishuWebhook("https://feishu.webhook"),
			WithTimeout(30 * time.Second),
		}

		cfg1, err1 := NewConfigWithoutValidation(opts1...)
		cfg2, err2 := NewConfigWithoutValidation(opts2...)

		assert.NoError(t, err1)
		assert.NoError(t, err2)

		assert.Equal(t, cfg1.DefaultTimeout, cfg2.DefaultTimeout)
		assert.Equal(t, cfg1.Feishu.WebhookURL, cfg2.Feishu.WebhookURL)
		assert.Equal(t, cfg1.AsyncConfig.Workers, cfg2.AsyncConfig.Workers)
	})
}

// TestEnvironmentOptions tests environment variable loading
func TestEnvironmentOptions(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"NOTIFYHUB_TIMEOUT":         os.Getenv("NOTIFYHUB_TIMEOUT"),
		"NOTIFYHUB_ENABLE_METRICS":  os.Getenv("NOTIFYHUB_ENABLE_METRICS"),
		"NOTIFYHUB_ENABLE_TRACING":  os.Getenv("NOTIFYHUB_ENABLE_TRACING"),
		"NOTIFYHUB_EMAIL_HOST":      os.Getenv("NOTIFYHUB_EMAIL_HOST"),
		"NOTIFYHUB_EMAIL_PORT":      os.Getenv("NOTIFYHUB_EMAIL_PORT"),
		"NOTIFYHUB_EMAIL_FROM":      os.Getenv("NOTIFYHUB_EMAIL_FROM"),
		"NOTIFYHUB_EMAIL_USERNAME":  os.Getenv("NOTIFYHUB_EMAIL_USERNAME"),
		"NOTIFYHUB_EMAIL_PASSWORD":  os.Getenv("NOTIFYHUB_EMAIL_PASSWORD"),
		"NOTIFYHUB_FEISHU_WEBHOOK":  os.Getenv("NOTIFYHUB_FEISHU_WEBHOOK"),
		"NOTIFYHUB_FEISHU_SECRET":   os.Getenv("NOTIFYHUB_FEISHU_SECRET"),
		"NOTIFYHUB_ASYNC_ENABLED":   os.Getenv("NOTIFYHUB_ASYNC_ENABLED"),
		"NOTIFYHUB_QUEUE_TYPE":      os.Getenv("NOTIFYHUB_QUEUE_TYPE"),
	}

	// Clean up function
	cleanup := func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}
	defer cleanup()

	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(*testing.T, *Config)
	}{
		{
			name: "full environment configuration",
			envVars: map[string]string{
				"NOTIFYHUB_TIMEOUT":         "60s",
				"NOTIFYHUB_ENABLE_METRICS":  "true",
				"NOTIFYHUB_ENABLE_TRACING":  "true",
				"NOTIFYHUB_EMAIL_HOST":      "smtp.env.com",
				"NOTIFYHUB_EMAIL_PORT":      "587",
				"NOTIFYHUB_EMAIL_FROM":      "env@example.com",
				"NOTIFYHUB_EMAIL_USERNAME":  "envuser",
				"NOTIFYHUB_EMAIL_PASSWORD":  "envpass",
				"NOTIFYHUB_FEISHU_WEBHOOK":  "https://env.feishu.webhook",
				"NOTIFYHUB_FEISHU_SECRET":   "envsecret",
				"NOTIFYHUB_ASYNC_ENABLED":   "true",
				"NOTIFYHUB_QUEUE_TYPE":      "redis",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 60*time.Second, cfg.DefaultTimeout)
				assert.True(t, cfg.EnableMetrics)
				assert.True(t, cfg.EnableTracing)

				require.NotNil(t, cfg.Email)
				assert.Equal(t, "smtp.env.com", cfg.Email.SMTPHost)
				assert.Equal(t, 587, cfg.Email.SMTPPort)
				assert.Equal(t, "env@example.com", cfg.Email.SMTPFrom)
				assert.Equal(t, "envuser", cfg.Email.SMTPUsername)
				assert.Equal(t, "envpass", cfg.Email.SMTPPassword)

				require.NotNil(t, cfg.Feishu)
				assert.Equal(t, "https://env.feishu.webhook", cfg.Feishu.WebhookURL)
				assert.Equal(t, "envsecret", cfg.Feishu.Secret)
				assert.True(t, cfg.Feishu.SignVerify)

				assert.True(t, cfg.AsyncConfig.Enabled)
				assert.Equal(t, "redis", cfg.AsyncConfig.QueueType)
			},
		},
		{
			name: "partial environment configuration",
			envVars: map[string]string{
				"NOTIFYHUB_FEISHU_WEBHOOK": "https://partial.feishu.webhook",
				"NOTIFYHUB_ASYNC_ENABLED":  "true",
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Feishu)
				assert.Equal(t, "https://partial.feishu.webhook", cfg.Feishu.WebhookURL)
				assert.Empty(t, cfg.Feishu.Secret)
				assert.False(t, cfg.Feishu.SignVerify)

				assert.True(t, cfg.AsyncConfig.Enabled)
			},
		},
		{
			name: "invalid port number in environment",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST": "smtp.example.com",
				"NOTIFYHUB_EMAIL_PORT": "invalid-port",
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Email)
				assert.Equal(t, "smtp.example.com", cfg.Email.SMTPHost)
				assert.Equal(t, 0, cfg.Email.SMTPPort) // Should remain default/zero
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment first
			for key := range originalEnv {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg := &Config{
				Platforms: make(map[string]map[string]interface{}),
			}
			err := WithEnvDefaults()(cfg)

			assert.NoError(t, err)
			tt.validate(t, cfg)
		})
	}
}

// TestUtilityOptions tests utility configuration functions
func TestUtilityOptions(t *testing.T) {
	t.Run("WithDefaults", func(t *testing.T) {
		cfg := &Config{
			Platforms: make(map[string]map[string]interface{}),
		}
		err := WithDefaults()(cfg)

		assert.NoError(t, err)
		assert.Equal(t, 30*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 3, cfg.MaxRetries)
		assert.Equal(t, "round_robin", cfg.RouterConfig.LoadBalancing)
		assert.True(t, cfg.AsyncConfig.Enabled)
		assert.Equal(t, "memory", cfg.AsyncConfig.QueueType)
		assert.Equal(t, 4, cfg.AsyncConfig.Workers)
	})

	t.Run("WithTestDefaults", func(t *testing.T) {
		cfg := &Config{
			Platforms: make(map[string]map[string]interface{}),
		}
		err := WithTestDefaults()(cfg)

		assert.NoError(t, err)
		assert.Equal(t, 5*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 1, cfg.MaxRetries)
		assert.False(t, cfg.EnableMetrics)
		assert.False(t, cfg.EnableTracing)
		assert.True(t, cfg.AsyncConfig.Enabled)
		assert.Equal(t, 2, cfg.AsyncConfig.Workers)
	})

	t.Run("WithValidationSkip", func(t *testing.T) {
		cfg := &Config{}
		err := WithValidationSkip()(cfg)

		// This is just a marker option, should not fail
		assert.NoError(t, err)
	})

	t.Run("WithRetryPolicy", func(t *testing.T) {
		cfg := &Config{}
		err := WithRetryPolicy(5, 500, 1.5)(cfg)

		assert.NoError(t, err)
		assert.Equal(t, 5, cfg.RetryPolicy.MaxRetries)
		assert.Equal(t, 500*time.Millisecond, cfg.RetryPolicy.InitialInterval)
		assert.Equal(t, 1.5, cfg.RetryPolicy.Multiplier)
		assert.Equal(t, 30*time.Second, cfg.RetryPolicy.MaxInterval)
	})
}

// TestLegacyPlatformOption tests backward compatibility support
func TestLegacyPlatformOption(t *testing.T) {
	t.Run("WithPlatform legacy support", func(t *testing.T) {
		cfg := &Config{}
		platformConfig := map[string]interface{}{
			"endpoint": "https://api.example.com",
			"api_key":  "secret123",
			"timeout":  "30s",
		}

		err := WithPlatform("custom", platformConfig)(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, cfg.Platforms)
		assert.Equal(t, platformConfig, cfg.Platforms["custom"])
	})

	t.Run("WithPlatform creates Platforms map if nil", func(t *testing.T) {
		cfg := &Config{Platforms: nil}
		platformConfig := map[string]interface{}{
			"key": "value",
		}

		err := WithPlatform("test", platformConfig)(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, cfg.Platforms)
		assert.Equal(t, platformConfig, cfg.Platforms["test"])
	})
}

// TestOptionErrorHandling tests error scenarios in option functions
func TestOptionErrorHandling(t *testing.T) {
	t.Run("invalid timeout in WithTimeout", func(t *testing.T) {
		cfg := &Config{}
		err := WithTimeout(-10 * time.Second)(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout must be positive")
	})

	t.Run("invalid retries in WithMaxRetries", func(t *testing.T) {
		cfg := &Config{}
		err := WithMaxRetries(-5)(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max retries must be non-negative")
	})

	t.Run("option combination with errors", func(t *testing.T) {
		opts := []Option{
			WithTimeout(30 * time.Second),   // Valid
			WithMaxRetries(-1),              // Invalid
			WithMetrics(true),               // Valid
		}

		cfg, err := NewConfig(opts...)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})
}

// TestComplexOptionScenarios tests real-world option combination scenarios
func TestComplexOptionScenarios(t *testing.T) {
	t.Run("production-like configuration", func(t *testing.T) {
		opts := []Option{
			WithTimeout(60 * time.Second),
			WithMaxRetries(5),
			WithMetrics(true),
			WithTracing(true),
			WithFeishuWebhook("https://open.feishu.cn/webhook/prod", "prod-secret"),
			WithEmailBasic("smtp.prod.com", 587, "alerts@prod.com"),
			WithSlackWebhook("https://hooks.slack.com/prod", "#alerts"),
			WithAsyncBasic("redis", 16),
			WithRouterBasic("weighted"),
			WithRetryPolicy(5, 1000, 2.0),
		}

		cfg, err := NewConfigWithoutValidation(opts...)
		assert.NoError(t, err)

		// Verify comprehensive configuration
		assert.Equal(t, 60*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 5, cfg.MaxRetries)
		assert.True(t, cfg.EnableMetrics)
		assert.True(t, cfg.EnableTracing)

		assert.NotNil(t, cfg.Feishu)
		assert.NotNil(t, cfg.Email)
		assert.NotNil(t, cfg.Slack)

		assert.True(t, cfg.AsyncConfig.Enabled)
		assert.Equal(t, 16, cfg.AsyncConfig.Workers)
		assert.Equal(t, "weighted", cfg.RouterConfig.LoadBalancing)
	})

	t.Run("development configuration", func(t *testing.T) {
		opts := []Option{
			WithTestDefaults(),
			WithFeishuWebhook("http://localhost:8080/feishu"),
			WithWebhookBasic("http://localhost:8080/webhook"),
		}

		cfg, err := NewConfig(opts...)
		assert.NoError(t, err)

		assert.Equal(t, 5*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 1, cfg.MaxRetries)
		assert.False(t, cfg.EnableMetrics)
		assert.NotNil(t, cfg.Feishu)
		assert.NotNil(t, cfg.Webhook)
	})

	t.Run("minimal configuration", func(t *testing.T) {
		opts := []Option{
			WithFeishuWebhook("https://minimal.webhook"),
		}

		cfg, err := NewConfig(opts...)
		assert.NoError(t, err)

		// Should have defaults applied
		assert.Equal(t, 30*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 3, cfg.MaxRetries)
		assert.NotNil(t, cfg.Feishu)
	})
}