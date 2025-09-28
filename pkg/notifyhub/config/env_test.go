// Package config tests for environment variable loading and validation
package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnvironmentVariableLoading tests comprehensive environment variable support
func TestEnvironmentVariableLoading(t *testing.T) {
	// Save original environment to restore later
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

	// Cleanup function to restore environment
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
			name: "core configuration from environment",
			envVars: map[string]string{
				"NOTIFYHUB_TIMEOUT":        "45s",
				"NOTIFYHUB_ENABLE_METRICS": "true",
				"NOTIFYHUB_ENABLE_TRACING": "true",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 45*time.Second, cfg.DefaultTimeout)
				assert.True(t, cfg.EnableMetrics)
				assert.True(t, cfg.EnableTracing)
			},
		},
		{
			name: "email configuration from environment",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST":     "smtp.env.test.com",
				"NOTIFYHUB_EMAIL_PORT":     "587",
				"NOTIFYHUB_EMAIL_FROM":     "env-test@example.com",
				"NOTIFYHUB_EMAIL_USERNAME": "env-user",
				"NOTIFYHUB_EMAIL_PASSWORD": "env-password",
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Email)
				assert.Equal(t, "smtp.env.test.com", cfg.Email.SMTPHost)
				assert.Equal(t, 587, cfg.Email.SMTPPort)
				assert.Equal(t, "env-test@example.com", cfg.Email.SMTPFrom)
				assert.Equal(t, "env-user", cfg.Email.SMTPUsername)
				assert.Equal(t, "env-password", cfg.Email.SMTPPassword)
			},
		},
		{
			name: "feishu configuration from environment",
			envVars: map[string]string{
				"NOTIFYHUB_FEISHU_WEBHOOK": "https://env.feishu.cn/webhook/test-token",
				"NOTIFYHUB_FEISHU_SECRET":  "env-secret-123",
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Feishu)
				assert.Equal(t, "https://env.feishu.cn/webhook/test-token", cfg.Feishu.WebhookURL)
				assert.Equal(t, "env-secret-123", cfg.Feishu.Secret)
				assert.Equal(t, "webhook", cfg.Feishu.AuthType)
				assert.True(t, cfg.Feishu.SignVerify)
			},
		},
		{
			name: "async configuration from environment",
			envVars: map[string]string{
				"NOTIFYHUB_ASYNC_ENABLED": "true",
				"NOTIFYHUB_QUEUE_TYPE":    "redis",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.AsyncConfig.Enabled)
				assert.Equal(t, "redis", cfg.AsyncConfig.QueueType)
			},
		},
		{
			name: "partial email configuration",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST": "smtp.partial.com",
				"NOTIFYHUB_EMAIL_FROM": "partial@example.com",
				// Missing port, username, password
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Email)
				assert.Equal(t, "smtp.partial.com", cfg.Email.SMTPHost)
				assert.Equal(t, "partial@example.com", cfg.Email.SMTPFrom)
				assert.Equal(t, 0, cfg.Email.SMTPPort) // Should remain 0 when not set
				assert.Empty(t, cfg.Email.SMTPUsername)
				assert.Empty(t, cfg.Email.SMTPPassword)
			},
		},
		{
			name: "feishu without secret",
			envVars: map[string]string{
				"NOTIFYHUB_FEISHU_WEBHOOK": "https://open.feishu.cn/webhook/no-secret",
				// No secret provided
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Feishu)
				assert.Equal(t, "https://open.feishu.cn/webhook/no-secret", cfg.Feishu.WebhookURL)
				assert.Empty(t, cfg.Feishu.Secret)
				assert.Equal(t, "webhook", cfg.Feishu.AuthType)
				assert.False(t, cfg.Feishu.SignVerify)
			},
		},
		{
			name: "invalid timeout duration",
			envVars: map[string]string{
				"NOTIFYHUB_TIMEOUT": "invalid-duration",
			},
			validate: func(t *testing.T, cfg *Config) {
				// Should fallback to default when parsing fails
				assert.Equal(t, 30*time.Second, cfg.DefaultTimeout)
			},
		},
		{
			name: "async disabled explicitly",
			envVars: map[string]string{
				"NOTIFYHUB_ASYNC_ENABLED": "false",
				"NOTIFYHUB_QUEUE_TYPE":    "redis", // Should still be set
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.AsyncConfig.Enabled)
				assert.Equal(t, "redis", cfg.AsyncConfig.QueueType)
			},
		},
		{
			name: "invalid port numbers",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST": "smtp.test.com",
				"NOTIFYHUB_EMAIL_PORT": "invalid-port",
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Email)
				assert.Equal(t, "smtp.test.com", cfg.Email.SMTPHost)
				assert.Equal(t, 0, cfg.Email.SMTPPort) // Should remain 0 when parsing fails
			},
		},
		{
			name: "standard email ports",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST": "smtp.test.com",
				"NOTIFYHUB_EMAIL_PORT": "465", // SSL port
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Email)
				assert.Equal(t, 465, cfg.Email.SMTPPort)
			},
		},
		{
			name: "submission port 25",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST": "smtp.test.com",
				"NOTIFYHUB_EMAIL_PORT": "25",
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Email)
				assert.Equal(t, 25, cfg.Email.SMTPPort)
			},
		},
		{
			name: "empty environment values",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST":     "",
				"NOTIFYHUB_FEISHU_WEBHOOK": "",
				"NOTIFYHUB_ASYNC_ENABLED":  "",
			},
			validate: func(t *testing.T, cfg *Config) {
				// Empty values should not create platform configs
				assert.Nil(t, cfg.Email)
				assert.Nil(t, cfg.Feishu)
				// Async should remain at default state
				assert.False(t, cfg.AsyncConfig.Enabled)
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

			// Create config and apply environment defaults
			cfg, err := NewConfigWithoutValidation(WithEnvDefaults())
			require.NoError(t, err)

			tt.validate(t, cfg)
		})
	}
}

// TestEnvironmentVariablePrecedence tests precedence order
func TestEnvironmentVariablePrecedence(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"NOTIFYHUB_TIMEOUT":        os.Getenv("NOTIFYHUB_TIMEOUT"),
		"NOTIFYHUB_EMAIL_HOST":     os.Getenv("NOTIFYHUB_EMAIL_HOST"),
		"NOTIFYHUB_FEISHU_WEBHOOK": os.Getenv("NOTIFYHUB_FEISHU_WEBHOOK"),
	}

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

	t.Run("environment overrides defaults", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		// Set environment variables
		os.Setenv("NOTIFYHUB_TIMEOUT", "60s")
		os.Setenv("NOTIFYHUB_EMAIL_HOST", "smtp.env.com")

		// Apply defaults first, then environment
		cfg, err := NewConfigWithoutValidation(
			WithDefaults(),       // Sets default timeout to 30s
			WithEnvDefaults(),    // Should override with 60s from env
		)
		require.NoError(t, err)

		assert.Equal(t, 60*time.Second, cfg.DefaultTimeout) // Environment wins
		require.NotNil(t, cfg.Email)
		assert.Equal(t, "smtp.env.com", cfg.Email.SMTPHost) // From environment
	})

	t.Run("explicit options override environment", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		// Set environment variables
		os.Setenv("NOTIFYHUB_TIMEOUT", "60s")
		os.Setenv("NOTIFYHUB_FEISHU_WEBHOOK", "https://env.webhook")

		// Apply environment first, then explicit options
		cfg, err := NewConfigWithoutValidation(
			WithEnvDefaults(),                                            // Sets from env
			WithTimeout(90*time.Second),                                  // Should override env
			WithFeishuWebhook("https://explicit.webhook", "explicit"),   // Should override env
		)
		require.NoError(t, err)

		assert.Equal(t, 90*time.Second, cfg.DefaultTimeout) // Explicit option wins
		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, "https://explicit.webhook", cfg.Feishu.WebhookURL) // Explicit option wins
		assert.Equal(t, "explicit", cfg.Feishu.Secret)
	})

	t.Run("precedence order: explicit > env > defaults", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		// Set environment variables
		os.Setenv("NOTIFYHUB_TIMEOUT", "45s")

		cfg, err := NewConfigWithoutValidation(
			WithDefaults(),                   // Default: 30s
			WithEnvDefaults(),                // Env: 45s
			WithTimeout(75*time.Second),      // Explicit: 75s
		)
		require.NoError(t, err)

		assert.Equal(t, 75*time.Second, cfg.DefaultTimeout) // Explicit wins
	})
}

// TestEnvironmentVariableTypeConversion tests type conversion and error handling
func TestEnvironmentVariableTypeConversion(t *testing.T) {
	originalEnv := map[string]string{
		"NOTIFYHUB_TIMEOUT":        os.Getenv("NOTIFYHUB_TIMEOUT"),
		"NOTIFYHUB_ENABLE_METRICS": os.Getenv("NOTIFYHUB_ENABLE_METRICS"),
		"NOTIFYHUB_EMAIL_PORT":     os.Getenv("NOTIFYHUB_EMAIL_PORT"),
	}

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
			name: "valid duration parsing",
			envVars: map[string]string{
				"NOTIFYHUB_TIMEOUT": "2m30s",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 150*time.Second, cfg.DefaultTimeout)
			},
		},
		{
			name: "invalid duration parsing - falls back to default",
			envVars: map[string]string{
				"NOTIFYHUB_TIMEOUT": "invalid-duration",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 30*time.Second, cfg.DefaultTimeout) // Default
			},
		},
		{
			name: "boolean parsing - true",
			envVars: map[string]string{
				"NOTIFYHUB_ENABLE_METRICS": "true",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.EnableMetrics)
			},
		},
		{
			name: "boolean parsing - false",
			envVars: map[string]string{
				"NOTIFYHUB_ENABLE_METRICS": "false",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.EnableMetrics)
			},
		},
		{
			name: "boolean parsing - 1 as true",
			envVars: map[string]string{
				"NOTIFYHUB_ENABLE_METRICS": "1",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.EnableMetrics) // Only "true" is recognized
			},
		},
		{
			name: "integer parsing - valid port",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST": "smtp.test.com",
				"NOTIFYHUB_EMAIL_PORT": "2525",
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Email)
				assert.Equal(t, 2525, cfg.Email.SMTPPort)
			},
		},
		{
			name: "integer parsing - standard ports",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST": "smtp.test.com",
				"NOTIFYHUB_EMAIL_PORT": "587",
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Email)
				assert.Equal(t, 587, cfg.Email.SMTPPort)
			},
		},
		{
			name: "complex duration formats",
			envVars: map[string]string{
				"NOTIFYHUB_TIMEOUT": "1h30m45s",
			},
			validate: func(t *testing.T, cfg *Config) {
				expected := 1*time.Hour + 30*time.Minute + 45*time.Second
				assert.Equal(t, expected, cfg.DefaultTimeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			for key := range originalEnv {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := NewConfigWithoutValidation(WithEnvDefaults())
			require.NoError(t, err)

			tt.validate(t, cfg)
		})
	}
}

// TestEnvironmentVariableValidation tests validation with environment variables
func TestEnvironmentVariableValidation(t *testing.T) {
	originalEnv := map[string]string{
		"NOTIFYHUB_FEISHU_WEBHOOK": os.Getenv("NOTIFYHUB_FEISHU_WEBHOOK"),
		"NOTIFYHUB_EMAIL_HOST":     os.Getenv("NOTIFYHUB_EMAIL_HOST"),
		"NOTIFYHUB_EMAIL_FROM":     os.Getenv("NOTIFYHUB_EMAIL_FROM"),
	}

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
		name      string
		envVars   map[string]string
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid environment configuration",
			envVars: map[string]string{
				"NOTIFYHUB_FEISHU_WEBHOOK": "https://open.feishu.cn/webhook/valid",
				"NOTIFYHUB_EMAIL_HOST":     "smtp.valid.com",
				"NOTIFYHUB_EMAIL_FROM":     "valid@example.com",
			},
			wantError: false,
		},
		{
			name: "invalid feishu webhook URL",
			envVars: map[string]string{
				"NOTIFYHUB_FEISHU_WEBHOOK": "invalid-url",
			},
			wantError: true,
			errorMsg:  "feishu config",
		},
		{
			name: "invalid email configuration",
			envVars: map[string]string{
				"NOTIFYHUB_EMAIL_HOST": "smtp.test.com",
				"NOTIFYHUB_EMAIL_FROM": "invalid-email", // Missing @
			},
			wantError: true,
			errorMsg:  "email config",
		},
		{
			name: "mixed valid and invalid",
			envVars: map[string]string{
				"NOTIFYHUB_FEISHU_WEBHOOK": "https://valid.webhook",
				"NOTIFYHUB_EMAIL_HOST":     "", // Required field empty
				"NOTIFYHUB_EMAIL_FROM":     "valid@example.com",
			},
			// Email config won't be created if host is empty, so no validation error
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			for key := range originalEnv {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Create config with validation enabled
			cfg, err := NewConfig(WithEnvDefaults())

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

// TestEnvironmentVariableEdgeCases tests edge cases and special scenarios
func TestEnvironmentVariableEdgeCases(t *testing.T) {
	originalEnv := map[string]string{
		"NOTIFYHUB_EMAIL_HOST":     os.Getenv("NOTIFYHUB_EMAIL_HOST"),
		"NOTIFYHUB_EMAIL_PORT":     os.Getenv("NOTIFYHUB_EMAIL_PORT"),
		"NOTIFYHUB_FEISHU_WEBHOOK": os.Getenv("NOTIFYHUB_FEISHU_WEBHOOK"),
		"NOTIFYHUB_FEISHU_SECRET":  os.Getenv("NOTIFYHUB_FEISHU_SECRET"),
	}

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

	t.Run("whitespace in environment variables", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		os.Setenv("NOTIFYHUB_EMAIL_HOST", "  smtp.test.com  ")
		os.Setenv("NOTIFYHUB_FEISHU_SECRET", "  secret-with-spaces  ")

		cfg, err := NewConfigWithoutValidation(WithEnvDefaults())
		require.NoError(t, err)

		// Environment values should be used as-is (no automatic trimming)
		require.NotNil(t, cfg.Email)
		assert.Equal(t, "  smtp.test.com  ", cfg.Email.SMTPHost)
		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, "  secret-with-spaces  ", cfg.Feishu.Secret)
	})

	t.Run("very long environment values", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		longWebhook := "https://very-long-webhook-url.example.com/" + string(make([]byte, 500))
		os.Setenv("NOTIFYHUB_FEISHU_WEBHOOK", longWebhook)

		cfg, err := NewConfigWithoutValidation(WithEnvDefaults())
		require.NoError(t, err)

		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, longWebhook, cfg.Feishu.WebhookURL)
	})

	t.Run("special characters in environment values", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		specialSecret := "secret!@#$%^&*()_+-={}[]|\\:;\"'<>,.?/"
		os.Setenv("NOTIFYHUB_FEISHU_WEBHOOK", "https://example.com")
		os.Setenv("NOTIFYHUB_FEISHU_SECRET", specialSecret)

		cfg, err := NewConfigWithoutValidation(WithEnvDefaults())
		require.NoError(t, err)

		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, specialSecret, cfg.Feishu.Secret)
	})

	t.Run("unicode in environment values", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		unicodeSecret := "密钥-αβγ-🔑"
		os.Setenv("NOTIFYHUB_FEISHU_WEBHOOK", "https://example.com")
		os.Setenv("NOTIFYHUB_FEISHU_SECRET", unicodeSecret)

		cfg, err := NewConfigWithoutValidation(WithEnvDefaults())
		require.NoError(t, err)

		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, unicodeSecret, cfg.Feishu.Secret)
	})

	t.Run("case sensitivity", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		// Set lowercase version (should not be recognized)
		os.Setenv("notifyhub_email_host", "smtp.lowercase.com")
		// Set correct uppercase version
		os.Setenv("NOTIFYHUB_EMAIL_HOST", "smtp.uppercase.com")

		cfg, err := NewConfigWithoutValidation(WithEnvDefaults())
		require.NoError(t, err)

		require.NotNil(t, cfg.Email)
		assert.Equal(t, "smtp.uppercase.com", cfg.Email.SMTPHost) // Only uppercase is recognized
	})
}

