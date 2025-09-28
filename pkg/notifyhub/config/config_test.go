// Package config tests provide comprehensive testing for the unified configuration system
package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestNewConfig tests the basic configuration creation with various option combinations
func TestNewConfig(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		wantError bool
		validate  func(*testing.T, *Config)
	}{
		{
			name: "empty config with defaults",
			opts: []Option{},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 30*time.Second, cfg.DefaultTimeout)
				assert.Equal(t, 3, cfg.MaxRetries)
				assert.False(t, cfg.EnableMetrics)
				assert.False(t, cfg.EnableTracing)
				assert.NotNil(t, cfg.Platforms)
				assert.Equal(t, "round_robin", cfg.RouterConfig.LoadBalancing)
				assert.Equal(t, "memory", cfg.AsyncConfig.QueueType)
			},
		},
		{
			name: "config with timeout option",
			opts: []Option{WithTimeout(60 * time.Second)},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 60*time.Second, cfg.DefaultTimeout)
			},
		},
		{
			name: "config with multiple options",
			opts: []Option{
				WithTimeout(45 * time.Second),
				WithMaxRetries(5),
				WithMetrics(true),
				WithTracing(true),
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 45*time.Second, cfg.DefaultTimeout)
				assert.Equal(t, 5, cfg.MaxRetries)
				assert.True(t, cfg.EnableMetrics)
				assert.True(t, cfg.EnableTracing)
			},
		},
		{
			name: "config with Feishu platform",
			opts: []Option{
				WithFeishu(FeishuConfig{
					WebhookURL: "https://open.feishu.cn/webhook/test",
					Secret:     "test-secret",
					AuthType:   "webhook",
					Timeout:    20 * time.Second,
				}),
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Feishu)
				assert.Equal(t, "https://open.feishu.cn/webhook/test", cfg.Feishu.WebhookURL)
				assert.Equal(t, "test-secret", cfg.Feishu.Secret)
				assert.Equal(t, 20*time.Second, cfg.Feishu.Timeout)
			},
		},
		{
			name: "config with Email platform",
			opts: []Option{
				WithEmail(EmailConfig{
					SMTPHost: "smtp.gmail.com",
					SMTPPort: 587,
					SMTPFrom: "test@example.com",
					SMTPTLS:  true,
					Timeout:  30 * time.Second,
				}),
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Email)
				assert.Equal(t, "smtp.gmail.com", cfg.Email.SMTPHost)
				assert.Equal(t, 587, cfg.Email.SMTPPort)
				assert.Equal(t, "test@example.com", cfg.Email.SMTPFrom)
				assert.True(t, cfg.Email.SMTPTLS)
			},
		},
		{
			name: "config with async enabled",
			opts: []Option{
				WithAsyncConfig(AsyncConfig{
					Enabled:        true,
					QueueType:      "redis",
					Workers:        8,
					QueueSize:      2000,
					BatchSize:      10,
					ProcessTimeout: 30 * time.Second,
				}),
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.AsyncConfig.Enabled)
				assert.Equal(t, "redis", cfg.AsyncConfig.QueueType)
				assert.Equal(t, 8, cfg.AsyncConfig.Workers)
				assert.Equal(t, 2000, cfg.AsyncConfig.QueueSize)
			},
		},
		{
			name:      "invalid timeout should fail validation",
			opts:      []Option{WithTimeout(-5 * time.Second)},
			wantError: true,
		},
		{
			name:      "invalid max retries should fail validation",
			opts:      []Option{WithMaxRetries(-1)},
			wantError: true,
		},
		{
			name: "invalid Feishu config should fail validation",
			opts: []Option{
				WithFeishu(FeishuConfig{
					WebhookURL: "invalid-url", // Invalid URL
				}),
			},
			wantError: true,
		},
		{
			name: "invalid Email config should fail validation",
			opts: []Option{
				WithEmail(EmailConfig{
					SMTPHost: "", // Required field empty
					SMTPPort: 587,
					SMTPFrom: "invalid-email", // Invalid email format
				}),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.opts...)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

// TestNewConfigWithoutValidation tests config creation skipping validation
func TestNewConfigWithoutValidation(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		validate func(*testing.T, *Config)
	}{
		{
			name: "invalid config structure should not fail when validation is skipped",
			opts: []Option{
				WithFeishu(FeishuConfig{
					WebhookURL: "invalid-url", // Invalid URL but validation skipped
					Secret:     "test-secret",
					AuthType:   "webhook", // Valid auth type
					Timeout:    30 * time.Second, // Valid timeout
				}),
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Feishu)
				assert.Equal(t, "invalid-url", cfg.Feishu.WebhookURL)
				assert.Equal(t, "test-secret", cfg.Feishu.Secret)
			},
		},
		{
			name: "partial invalid Feishu config should not fail",
			opts: []Option{
				WithFeishu(FeishuConfig{
					WebhookURL: "invalid-url", // Invalid URL but validation skipped
					Secret:     "test-secret",
					AuthType:   "webhook", // Valid auth type
					Timeout:    30 * time.Second, // Valid timeout
				}),
			},
			validate: func(t *testing.T, cfg *Config) {
				require.NotNil(t, cfg.Feishu)
				assert.Equal(t, "invalid-url", cfg.Feishu.WebhookURL)
				assert.Equal(t, "test-secret", cfg.Feishu.Secret)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfigWithoutValidation(tt.opts...)

			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

// TestConfigValidation tests the comprehensive validation system
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid minimal config",
			config: &Config{
				DefaultTimeout: 30 * time.Second,
				MaxRetries:     3,
				EnableMetrics:  false,
				EnableTracing:  false,
				Platforms:      make(map[string]map[string]interface{}),
				RetryPolicy: RetryPolicy{
					MaxRetries:      3,
					InitialInterval: 1 * time.Second,
					Multiplier:      2.0,
					MaxInterval:     30 * time.Second,
				},
				AsyncConfig: AsyncConfig{
					Enabled:        false,
					QueueType:      "memory",
					QueueSize:      1000,
					Workers:        4,
					BatchSize:      10,
					ProcessTimeout: 30 * time.Second,
				},
				RouterConfig: RouterConfig{
					LoadBalancing:       "round_robin",
					HealthCheckInterval: 30 * time.Second,
					MaxRetries:          3,
					RetryDelay:          1 * time.Second,
					EnableFailover:      true,
				},
			},
			wantError: false,
		},
		{
			name: "invalid timeout",
			config: &Config{
				DefaultTimeout: -10 * time.Second, // Invalid
				MaxRetries:     3,
			},
			wantError: true,
			errorMsg:  "DefaultTimeout",
		},
		{
			name: "invalid max retries",
			config: &Config{
				DefaultTimeout: 30 * time.Second,
				MaxRetries:     -1, // Invalid
			},
			wantError: true,
			errorMsg:  "MaxRetries",
		},
		{
			name: "invalid async config with workers but disabled",
			config: &Config{
				DefaultTimeout: 30 * time.Second,
				MaxRetries:     3,
				AsyncConfig: AsyncConfig{
					Enabled: true,
					Workers: 0, // Invalid when enabled
				},
			},
			wantError: true,
			errorMsg:  "async workers must be positive when async is enabled",
		},
		{
			name: "invalid router config with too many retries",
			config: &Config{
				DefaultTimeout: 30 * time.Second,
				MaxRetries:     3,
				RouterConfig: RouterConfig{
					MaxRetries: 15, // Too many retries
				},
			},
			wantError: true,
			errorMsg:  "router max_retries should not exceed 10",
		},
		{
			name: "invalid Feishu config",
			config: &Config{
				DefaultTimeout: 30 * time.Second,
				MaxRetries:     3,
				Feishu: &FeishuConfig{
					WebhookURL: "invalid-url", // Invalid URL
				},
			},
			wantError: true,
			errorMsg:  "feishu config",
		},
		{
			name: "invalid Email config",
			config: &Config{
				DefaultTimeout: 30 * time.Second,
				MaxRetries:     3,
				Email: &EmailConfig{
					SMTPHost: "", // Required field empty
					SMTPPort: 587,
					SMTPFrom: "invalid-email", // Invalid email
				},
			},
			wantError: true,
			errorMsg:  "email config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.NotEmpty(t, tt.config.ValidationErrors)
			} else {
				assert.NoError(t, err)
				assert.Empty(t, tt.config.ValidationErrors)
			}
		})
	}
}

// TestConfigSerialization tests JSON and YAML serialization roundtrips
func TestConfigSerialization(t *testing.T) {
	original := &Config{
		DefaultTimeout: 45 * time.Second,
		MaxRetries:     5,
		EnableMetrics:  true,
		EnableTracing:  true,
		Email: &EmailConfig{
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password123",
			SMTPFrom:     "test@example.com",
			SMTPTLS:      true,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
			RateLimit:    60,
		},
		Feishu: &FeishuConfig{
			WebhookURL: "https://open.feishu.cn/webhook/test",
			Secret:     "secret123",
			AuthType:   "webhook",
			Timeout:    20 * time.Second,
			MaxRetries: 2,
			RateLimit:  100,
			Keywords:   []string{"alert", "error"},
		},
		AsyncConfig: AsyncConfig{
			Enabled:        true,
			QueueType:      "redis",
			QueueSize:      2000,
			Workers:        8,
			BatchSize:      20,
			ProcessTimeout: 60 * time.Second,
		},
		RouterConfig: RouterConfig{
			LoadBalancing:       "weighted",
			HealthCheckInterval: 60 * time.Second,
			MaxRetries:          5,
			RetryDelay:          2 * time.Second,
			EnableFailover:      true,
		},
		RetryPolicy: RetryPolicy{
			MaxRetries:      5,
			InitialInterval: 2 * time.Second,
			Multiplier:      2.5,
			MaxInterval:     60 * time.Second,
		},
	}

	t.Run("JSON serialization roundtrip", func(t *testing.T) {
		// Serialize to JSON
		jsonData, err := json.Marshal(original)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Deserialize from JSON
		var deserialized Config
		err = json.Unmarshal(jsonData, &deserialized)
		require.NoError(t, err)

		// Compare key fields (excluding runtime fields like Logger)
		assert.Equal(t, original.DefaultTimeout, deserialized.DefaultTimeout)
		assert.Equal(t, original.MaxRetries, deserialized.MaxRetries)
		assert.Equal(t, original.EnableMetrics, deserialized.EnableMetrics)
		assert.Equal(t, original.EnableTracing, deserialized.EnableTracing)

		// Platform configs
		assert.Equal(t, original.Email, deserialized.Email)
		assert.Equal(t, original.Feishu, deserialized.Feishu)

		// Subsystem configs
		assert.Equal(t, original.AsyncConfig, deserialized.AsyncConfig)
		assert.Equal(t, original.RouterConfig, deserialized.RouterConfig)
		assert.Equal(t, original.RetryPolicy, deserialized.RetryPolicy)
	})

	t.Run("YAML serialization roundtrip", func(t *testing.T) {
		// Serialize to YAML
		yamlData, err := yaml.Marshal(original)
		require.NoError(t, err)
		assert.NotEmpty(t, yamlData)

		// Deserialize from YAML
		var deserialized Config
		err = yaml.Unmarshal(yamlData, &deserialized)
		require.NoError(t, err)

		// Compare key fields
		assert.Equal(t, original.DefaultTimeout, deserialized.DefaultTimeout)
		assert.Equal(t, original.MaxRetries, deserialized.MaxRetries)
		assert.Equal(t, original.Email, deserialized.Email)
		assert.Equal(t, original.Feishu, deserialized.Feishu)
	})
}

// TestGetPlatformConfig tests backward compatibility methods
func TestGetPlatformConfig(t *testing.T) {
	cfg := &Config{
		Email: &EmailConfig{
			SMTPHost: "smtp.gmail.com",
			SMTPPort: 587,
			SMTPFrom: "test@example.com",
			SMTPTLS:  true,
		},
		Feishu: &FeishuConfig{
			WebhookURL: "https://open.feishu.cn/webhook/test",
			Secret:     "secret123",
		},
		// Legacy platforms map
		Platforms: map[string]map[string]interface{}{
			"legacy": {
				"key1": "value1",
				"key2": 42,
			},
		},
	}

	tests := []struct {
		name         string
		platform     string
		expectNil    bool
		validateFunc func(*testing.T, map[string]interface{})
	}{
		{
			name:     "get email config as map",
			platform: "email",
			validateFunc: func(t *testing.T, config map[string]interface{}) {
				assert.Equal(t, "smtp.gmail.com", config["smtp_host"])
				assert.Equal(t, 587, config["smtp_port"])
				assert.Equal(t, "test@example.com", config["smtp_from"])
				assert.Equal(t, true, config["smtp_tls"])
			},
		},
		{
			name:     "get feishu config as map",
			platform: "feishu",
			validateFunc: func(t *testing.T, config map[string]interface{}) {
				assert.Equal(t, "https://open.feishu.cn/webhook/test", config["webhook_url"])
				assert.Equal(t, "secret123", config["secret"])
			},
		},
		{
			name:     "get legacy platform config",
			platform: "legacy",
			validateFunc: func(t *testing.T, config map[string]interface{}) {
				assert.Equal(t, "value1", config["key1"])
				assert.Equal(t, 42, config["key2"])
			},
		},
		{
			name:      "get non-existent platform",
			platform:  "nonexistent",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.GetPlatformConfig(tt.platform)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				if tt.validateFunc != nil {
					tt.validateFunc(t, result)
				}
			}
		})
	}
}

// TestConfigAccessors tests the convenience accessor methods
func TestConfigAccessors(t *testing.T) {
	cfg := &Config{
		AsyncConfig: AsyncConfig{
			Enabled: true,
			Workers: 8,
		},
	}

	assert.True(t, cfg.IsAsyncEnabled())
	assert.Equal(t, 8, cfg.Workers())

	cfg.AsyncConfig.Enabled = false
	assert.False(t, cfg.IsAsyncEnabled())
}

// TestConfigValidationMessages tests the validation error message formatting
func TestConfigValidationMessages(t *testing.T) {
	cfg := &Config{
		DefaultTimeout: -10 * time.Second, // Invalid
		MaxRetries:     -1,                // Invalid
		Email: &EmailConfig{
			SMTPHost: "",              // Required field empty
			SMTPPort: 70000,           // Out of range
			SMTPFrom: "invalid-email", // Invalid email format
		},
		AsyncConfig: AsyncConfig{
			Enabled: true,
			Workers: 0, // Invalid when enabled
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "configuration validation failed")
	assert.Contains(t, errorMsg, "errors")

	// Check that validation errors are stored
	assert.NotEmpty(t, cfg.ValidationErrors)
	assert.Greater(t, len(cfg.ValidationErrors), 0)
}

// TestLegacyPlatformMapConversion tests conversion of platform configs to maps
func TestLegacyPlatformMapConversion(t *testing.T) {
	cfg := &Config{
		SMS: &SMSConfig{
			Provider:  "twilio",
			APIKey:    "test-key",
			APISecret: "test-secret",
			From:      "+1234567890",
			Region:    "us-east-1",
			Timeout:   30 * time.Second,
			Templates: map[string]string{
				"verification": "Your code is {{code}}",
			},
		},
		Slack: &SlackConfig{
			WebhookURL:  "https://hooks.slack.com/webhook",
			Token:       "xoxb-token",
			Channel:     "#general",
			Username:    "NotifyBot",
			IconURL:     "https://example.com/icon.png",
			LinkNames:   true,
			UnfurlLinks: true,
		},
		Webhook: &WebhookConfig{
			URL:         "https://api.example.com/webhook",
			Method:      "POST",
			Headers:     map[string]string{"X-API-Key": "secret"},
			AuthType:    "bearer",
			AuthValue:   "token123",
			Timeout:     20 * time.Second,
			VerifySSL:   true,
			ContentType: "application/json",
		},
	}

	tests := []struct {
		name         string
		platform     string
		validateFunc func(*testing.T, map[string]interface{})
	}{
		{
			name:     "SMS config conversion",
			platform: "sms",
			validateFunc: func(t *testing.T, config map[string]interface{}) {
				assert.Equal(t, "twilio", config["provider"])
				assert.Equal(t, "test-key", config["api_key"])
				assert.Equal(t, "+1234567890", config["from"])
				assert.NotNil(t, config["templates"])
			},
		},
		{
			name:     "Slack config conversion",
			platform: "slack",
			validateFunc: func(t *testing.T, config map[string]interface{}) {
				assert.Equal(t, "https://hooks.slack.com/webhook", config["webhook_url"])
				assert.Equal(t, "xoxb-token", config["token"])
				assert.Equal(t, "#general", config["channel"])
				assert.Equal(t, true, config["link_names"])
			},
		},
		{
			name:     "Webhook config conversion",
			platform: "webhook",
			validateFunc: func(t *testing.T, config map[string]interface{}) {
				assert.Equal(t, "https://api.example.com/webhook", config["url"])
				assert.Equal(t, "POST", config["method"])
				assert.Equal(t, "bearer", config["auth_type"])
				assert.Equal(t, true, config["verify_ssl"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.GetPlatformConfig(tt.platform)
			assert.NotNil(t, result)
			tt.validateFunc(t, result)
		})
	}
}