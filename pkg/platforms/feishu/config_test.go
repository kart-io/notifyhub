package feishu

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateConfig tests the ValidateConfig function with various scenarios
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.FeishuConfig
		wantError bool
		errorMsg  string
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
			errorMsg:  "feishu config cannot be nil",
		},
		{
			name: "empty webhook URL",
			config: &config.FeishuConfig{
				WebhookURL: "",
			},
			wantError: true,
			errorMsg:  "webhook_url is required for webhook authentication",
		},
		{
			name: "invalid webhook URL - no protocol",
			config: &config.FeishuConfig{
				WebhookURL: "invalid-url",
			},
			wantError: true,
			errorMsg:  "webhook_url must start with http:// or https://",
		},
		{
			name: "invalid webhook URL - wrong protocol",
			config: &config.FeishuConfig{
				WebhookURL: "ftp://example.com",
			},
			wantError: true,
			errorMsg:  "webhook_url must start with http:// or https://",
		},
		{
			name: "negative timeout",
			config: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Timeout:    -5 * time.Second,
			},
			wantError: true,
			errorMsg:  "timeout cannot be negative",
		},
		{
			name: "empty keyword in list",
			config: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Keywords:   []string{"valid", "", "also-valid"},
			},
			wantError: true,
			errorMsg:  "keyword at index 1 cannot be empty",
		},
		{
			name: "whitespace-only keyword",
			config: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Keywords:   []string{"valid", "   ", "also-valid"},
			},
			wantError: true,
			errorMsg:  "keyword at index 1 cannot be empty",
		},
		{
			name: "valid minimal config",
			config: &config.FeishuConfig{
				WebhookURL: "https://example.com",
			},
			wantError: false,
		},
		{
			name: "valid config with HTTPS",
			config: &config.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/webhook-token",
				Secret:     "test-secret",
				Timeout:    30 * time.Second,
				Keywords:   []string{"alert", "notification"},
			},
			wantError: false,
		},
		{
			name: "valid config with HTTP",
			config: &config.FeishuConfig{
				WebhookURL: "http://localhost:8080/webhook",
				Secret:     "test-secret",
				Timeout:    10 * time.Second,
			},
			wantError: false,
		},
		{
			name: "keywords with extra whitespace - should be trimmed",
			config: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Keywords:   []string{" alert ", "  notification  "},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				// Verify keywords are trimmed after validation
				if tt.config != nil && len(tt.config.Keywords) > 0 {
					for _, keyword := range tt.config.Keywords {
						assert.Equal(t, keyword, strings.TrimSpace(keyword), "Keywords should be trimmed")
					}
				}
			}
		})
	}
}

// TestSetDefaults tests the SetDefaults function
func TestSetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		input          *config.FeishuConfig
		expectedOutput *config.FeishuConfig
	}{
		{
			name: "set default timeout",
			input: &config.FeishuConfig{
				WebhookURL: "https://example.com",
			},
			expectedOutput: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				AuthType:   "webhook",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				RateLimit:  60,
				Keywords:   []string{},
			},
		},
		{
			name: "preserve existing timeout",
			input: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Timeout:    60 * time.Second,
			},
			expectedOutput: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				AuthType:   "webhook",
				Timeout:    60 * time.Second,
				MaxRetries: 3,
				RateLimit:  60,
				Keywords:   []string{},
			},
		},
		{
			name: "trim keywords and remove empty ones",
			input: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Keywords:   []string{" alert ", "", "  notification  ", "", " warning "},
			},
			expectedOutput: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				AuthType:   "webhook",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				RateLimit:  60,
				Keywords:   []string{"alert", "notification", "warning"},
			},
		},
		{
			name: "handle nil keywords",
			input: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Keywords:   nil,
			},
			expectedOutput: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				AuthType:   "webhook",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				RateLimit:  60,
				Keywords:   []string{},
			},
		},
		{
			name: "handle empty keywords slice",
			input: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Keywords:   []string{},
			},
			expectedOutput: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				AuthType:   "webhook",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				RateLimit:  60,
				Keywords:   []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDefaults(tt.input)
			assert.Equal(t, tt.expectedOutput, tt.input)
		})
	}
}

// TestNewConfigFromMap tests the NewConfigFromMap function for backward compatibility
func TestNewConfigFromMap(t *testing.T) {
	tests := []struct {
		name      string
		configMap map[string]interface{}
		expected  *config.FeishuConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "basic config from map",
			configMap: map[string]interface{}{
				"webhook_url": "https://example.com",
				"secret":      "test-secret",
			},
			expected: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Secret:     "test-secret",
				AuthType:   "webhook",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				RateLimit:  60,
				Keywords:   []string{},
			},
			wantError: false,
		},
		{
			name: "config with timeout duration",
			configMap: map[string]interface{}{
				"webhook_url": "https://example.com",
				"timeout":     45 * time.Second,
			},
			expected: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				AuthType:   "webhook",
				Timeout:    45 * time.Second,
				MaxRetries: 3,
				RateLimit:  60,
				Keywords:   []string{},
			},
			wantError: false,
		},
		{
			name: "config with timeout string",
			configMap: map[string]interface{}{
				"webhook_url": "https://example.com",
				"timeout":     "1m30s",
			},
			expected: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Timeout:    90 * time.Second,
				Keywords:   []string{},
			},
			wantError: false,
		},
		{
			name: "config with keywords as interface slice",
			configMap: map[string]interface{}{
				"webhook_url": "https://example.com",
				"keywords":    []interface{}{"alert", "notification", "warning"},
			},
			expected: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Timeout:    30 * time.Second,
				Keywords:   []string{"alert", "notification", "warning"},
			},
			wantError: false,
		},
		{
			name: "config with keywords as string slice",
			configMap: map[string]interface{}{
				"webhook_url": "https://example.com",
				"keywords":    []string{"alert", "notification"},
			},
			expected: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Timeout:    30 * time.Second,
				Keywords:   []string{"alert", "notification"},
			},
			wantError: false,
		},
		{
			name: "config with keywords as comma-separated string",
			configMap: map[string]interface{}{
				"webhook_url": "https://example.com",
				"keywords":    "alert, notification, warning",
			},
			expected: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Timeout:    30 * time.Second,
				Keywords:   []string{"alert", "notification", "warning"},
			},
			wantError: false,
		},
		{
			name: "config with empty keywords string",
			configMap: map[string]interface{}{
				"webhook_url": "https://example.com",
				"keywords":    "",
			},
			expected: &config.FeishuConfig{
				WebhookURL: "https://example.com",
				Timeout:    30 * time.Second,
				Keywords:   []string{},
			},
			wantError: false,
		},
		{
			name: "invalid config - missing webhook URL",
			configMap: map[string]interface{}{
				"secret": "test-secret",
			},
			expected:  nil,
			wantError: true,
			errorMsg:  "webhook URL is required",
		},
		{
			name: "invalid config - invalid webhook URL",
			configMap: map[string]interface{}{
				"webhook_url": "invalid-url",
			},
			expected:  nil,
			wantError: true,
			errorMsg:  "webhook_url must start with http:// or https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewConfigFromMap(tt.configMap)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestConfigSerialization tests JSON serialization and deserialization
func TestConfigSerialization(t *testing.T) {
	original := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test-token",
		Secret:     "test-secret-12345",
		AppID:      "cli_test_app_id",
		AppSecret:  "test_app_secret",
		AuthType:   "webhook",
		Timeout:    45 * time.Second,
		MaxRetries: 5,
		RateLimit:  60,
		SignVerify: true,
		Keywords:   []string{"alert", "notification", "warning"},
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// Test JSON deserialization
	var deserialized config.FeishuConfig
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.WebhookURL, deserialized.WebhookURL)
	assert.Equal(t, original.Secret, deserialized.Secret)
	assert.Equal(t, original.AppID, deserialized.AppID)
	assert.Equal(t, original.AppSecret, deserialized.AppSecret)
	assert.Equal(t, original.AuthType, deserialized.AuthType)
	assert.Equal(t, original.Timeout, deserialized.Timeout)
	assert.Equal(t, original.MaxRetries, deserialized.MaxRetries)
	assert.Equal(t, original.RateLimit, deserialized.RateLimit)
	assert.Equal(t, original.SignVerify, deserialized.SignVerify)
	assert.Equal(t, original.Keywords, deserialized.Keywords)
}

// TestEnvironmentVariableLoading tests loading configuration from environment variables
func TestEnvironmentVariableLoading(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"FEISHU_WEBHOOK_URL": os.Getenv("FEISHU_WEBHOOK_URL"),
		"FEISHU_SECRET":      os.Getenv("FEISHU_SECRET"),
		"FEISHU_TIMEOUT":     os.Getenv("FEISHU_TIMEOUT"),
		"FEISHU_KEYWORDS":    os.Getenv("FEISHU_KEYWORDS"),
	}

	// Clean up function to restore environment
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
		expected *config.FeishuConfig
	}{
		{
			name: "load from environment variables",
			envVars: map[string]string{
				"FEISHU_WEBHOOK_URL": "https://open.feishu.cn/open-apis/bot/v2/hook/env-token",
				"FEISHU_SECRET":      "env-secret-123",
				"FEISHU_TIMEOUT":     "60s",
				"FEISHU_KEYWORDS":    "env-alert,env-notification",
			},
			expected: &config.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/env-token",
				Secret:     "env-secret-123",
				Timeout:    60 * time.Second,
				Keywords:   []string{"env-alert", "env-notification"},
			},
		},
		{
			name: "partial environment variables with defaults",
			envVars: map[string]string{
				"FEISHU_WEBHOOK_URL": "https://example.com/webhook",
			},
			expected: &config.FeishuConfig{
				WebhookURL: "https://example.com/webhook",
				Timeout:    30 * time.Second,
				Keywords:   []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config := LoadFromEnvironment()

			// Validate and set defaults
			SetDefaults(config)
			err := ValidateConfig(config)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, config)
		})
	}
}

// TestConfigValidationWithRealScenarios tests configuration validation with real-world scenarios
func TestConfigValidationWithRealScenarios(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.FeishuConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "production-like config",
			config: &config.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/abc123def456",
				Secret:     "SECabcdef123456789",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				RateLimit:  20,
				Keywords:   []string{"[ALERT]", "[ERROR]", "[CRITICAL]"},
			},
			wantError: false,
		},
		{
			name: "development config",
			config: &config.FeishuConfig{
				WebhookURL: "http://localhost:8080/feishu/webhook",
				Timeout:    10 * time.Second,
				MaxRetries: 1,
			},
			wantError: false,
		},
		{
			name: "minimal valid config",
			config: &config.FeishuConfig{
				WebhookURL: "https://hooks.example.com/feishu",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply defaults first
			SetDefaults(tt.config)

			err := ValidateConfig(tt.config)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// Ensure defaults are applied
				assert.Greater(t, tt.config.Timeout, time.Duration(0))
			}
		})
	}
}

// BenchmarkValidateConfig benchmarks the validation performance
func BenchmarkValidateConfig(b *testing.B) {
	config := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/benchmark-token",
		Secret:     "benchmark-secret-123456789",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RateLimit:  60,
		Keywords:   []string{"benchmark", "test", "performance", "validation"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateConfig(config)
	}
}

// BenchmarkSetDefaults benchmarks the default setting performance
func BenchmarkSetDefaults(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := &config.FeishuConfig{
			WebhookURL: "https://example.com",
			Keywords:   []string{" test ", "", " benchmark ", ""},
		}
		SetDefaults(config)
	}
}