package feishu

import (
	"testing"
	"time"

	"github.com/kart/notifyhub/pkg/config"
	"github.com/kart/notifyhub/pkg/target"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// mockLogger implements logger.Logger interface for testing
type mockLogger struct{}

func (m *mockLogger) LogMode(level logger.LogLevel) logger.Logger     { return m }
func (m *mockLogger) Debug(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Info(msg string, keysAndValues ...interface{})   {}
func (m *mockLogger) Warn(msg string, keysAndValues ...interface{})   {}
func (m *mockLogger) Error(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Fatal(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) With(keysAndValues ...interface{}) logger.Logger { return m }

func TestNewFeishuPlatform(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.FeishuConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: &config.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test-webhook",
			},
			wantError: false,
		},
		{
			name: "valid config with secret",
			config: &config.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
				Secret:     "test-secret",
			},
			wantError: false,
		},
		{
			name: "valid config with keywords",
			config: &config.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
				Keywords:   []string{"alert", "notification"},
			},
			wantError: false,
		},
		{
			name: "valid config with timeout",
			config: &config.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
				Timeout:    60 * time.Second,
			},
			wantError: false,
		},
		{
			name:      "empty webhook URL",
			config:    &config.FeishuConfig{},
			wantError: true,
			errorMsg:  "webhook URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewFeishuPlatform(tt.config, &mockLogger{})

			if tt.wantError {
				if err == nil {
					t.Errorf("NewFeishuPlatform() expected error, got nil")
				}
				if tt.errorMsg != "" && err != nil {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("NewFeishuPlatform() error = %v, want error containing %v", err, tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("NewFeishuPlatform() unexpected error = %v", err)
				}
				if p == nil {
					t.Error("NewFeishuPlatform() returned nil platform")
				}

				// Verify default timeout is set
				feishu := p.(*FeishuPlatform)
				if tt.config.Timeout == 0 && feishu.config.Timeout != 30*time.Second {
					t.Errorf("Default timeout = %v, want 30s", feishu.config.Timeout)
				}
			}
		})
	}
}

func TestFeishuPlatform_Name(t *testing.T) {
	cfg := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	}
	p, err := NewFeishuPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewFeishuPlatform() error = %v", err)
	}

	if got := p.Name(); got != "feishu" {
		t.Errorf("Name() = %v, want 'feishu'", got)
	}
}

func TestFeishuPlatform_ValidateTarget(t *testing.T) {
	cfg := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	}
	p, err := NewFeishuPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewFeishuPlatform() error = %v", err)
	}

	feishu := p.(*FeishuPlatform)

	tests := []struct {
		name      string
		target    target.Target
		wantError bool
	}{
		{
			name:      "valid feishu target",
			target:    target.Target{Type: "feishu", Value: "webhook-url"},
			wantError: false,
		},
		{
			name:      "valid webhook target",
			target:    target.Target{Type: "webhook", Value: "https://example.com/hook"},
			wantError: false,
		},
		{
			name:      "invalid target type",
			target:    target.Target{Type: "email", Value: "test@example.com"},
			wantError: true,
		},
		{
			name:      "empty value",
			target:    target.Target{Type: "feishu", Value: ""},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := feishu.ValidateTarget(tt.target)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTarget() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestFeishuPlatform_GetCapabilities(t *testing.T) {
	cfg := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	}
	p, err := NewFeishuPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewFeishuPlatform() error = %v", err)
	}

	caps := p.GetCapabilities()

	if caps.Name != "feishu" {
		t.Errorf("GetCapabilities() Name = %v, want 'feishu'", caps.Name)
	}

	if len(caps.SupportedTargetTypes) == 0 {
		t.Error("GetCapabilities() SupportedTargetTypes is empty")
	}

	// Check that feishu is in supported target types
	found := false
	for _, targetType := range caps.SupportedTargetTypes {
		if targetType == "feishu" || targetType == "webhook" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetCapabilities() should support feishu or webhook target type")
	}

	if len(caps.SupportedFormats) == 0 {
		t.Error("GetCapabilities() SupportedFormats is empty")
	}

	if caps.MaxMessageSize == 0 {
		t.Error("GetCapabilities() MaxMessageSize should be set")
	}
}

func TestFeishuPlatform_Close(t *testing.T) {
	cfg := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	}
	p, err := NewFeishuPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewFeishuPlatform() error = %v", err)
	}

	// Close should not panic
	err = p.Close()
	if err != nil {
		t.Errorf("Close() unexpected error = %v", err)
	}
}

func TestFeishuConfig_DefaultTimeout(t *testing.T) {
	cfg := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		// No timeout specified
	}
	p, err := NewFeishuPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewFeishuPlatform() error = %v", err)
	}

	feishu := p.(*FeishuPlatform)
	if feishu.config.Timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want 30s", feishu.config.Timeout)
	}
}

func TestFeishuConfig_CustomTimeout(t *testing.T) {
	customTimeout := 60 * time.Second
	cfg := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		Timeout:    customTimeout,
	}
	p, err := NewFeishuPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewFeishuPlatform() error = %v", err)
	}

	feishu := p.(*FeishuPlatform)
	if feishu.config.Timeout != customTimeout {
		t.Errorf("Custom timeout = %v, want %v", feishu.config.Timeout, customTimeout)
	}
}

func TestFeishuPlatform_isFeishuTarget(t *testing.T) {
	cfg := &config.FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	}
	p, err := NewFeishuPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewFeishuPlatform() error = %v", err)
	}

	feishu := p.(*FeishuPlatform)

	tests := []struct {
		name   string
		target target.Target
		want   bool
	}{
		{
			name:   "feishu type",
			target: target.Target{Type: "feishu", Value: "test"},
			want:   true,
		},
		{
			name:   "webhook type",
			target: target.Target{Type: "webhook", Value: "test"},
			want:   true,
		},
		{
			name:   "email type",
			target: target.Target{Type: "email", Value: "test@example.com"},
			want:   false,
		},
		{
			name:   "slack type",
			target: target.Target{Type: "slack", Value: "test"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := feishu.isFeishuTarget(tt.target); got != tt.want {
				t.Errorf("isFeishuTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
