package webhook

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

func TestNewWebhookPlatform(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.WebhookConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid minimal config",
			config: &config.WebhookConfig{
				URL: "https://example.com/webhook",
			},
			wantError: false,
		},
		{
			name: "valid config with all options",
			config: &config.WebhookConfig{
				URL:         "https://example.com/webhook",
				Method:      "POST",
				ContentType: "application/json",
				Headers: map[string]string{
					"Authorization": "Bearer token123",
					"X-Custom":      "value",
				},
				Timeout:   60 * time.Second,
				VerifySSL: true,
			},
			wantError: false,
		},
		{
			name: "valid config with custom method",
			config: &config.WebhookConfig{
				URL:    "https://example.com/webhook",
				Method: "PUT",
			},
			wantError: false,
		},
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
			errorMsg:  "configuration cannot be nil",
		},
		{
			name:      "empty URL",
			config:    &config.WebhookConfig{},
			wantError: true,
			errorMsg:  "url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewWebhookPlatform(tt.config, &mockLogger{})

			if tt.wantError {
				if err == nil {
					t.Errorf("NewWebhookPlatform() expected error, got nil")
				}
				if tt.errorMsg != "" && err != nil {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("NewWebhookPlatform() error = %v, want error containing %v", err, tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("NewWebhookPlatform() unexpected error = %v", err)
				}
				if p == nil {
					t.Error("NewWebhookPlatform() returned nil platform")
				}

				// Verify defaults
				webhook := p.(*WebhookPlatform)
				if tt.config.Method == "" && webhook.config.Method != "POST" {
					t.Errorf("Default method = %v, want POST", webhook.config.Method)
				}
				if tt.config.ContentType == "" && webhook.config.ContentType != "application/json" {
					t.Errorf("Default content type = %v, want application/json", webhook.config.ContentType)
				}
				if tt.config.Timeout == 0 && webhook.config.Timeout != 30*time.Second {
					t.Errorf("Default timeout = %v, want 30s", webhook.config.Timeout)
				}
			}
		})
	}
}

func TestWebhookPlatform_Name(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL: "https://example.com/webhook",
	}
	p, err := NewWebhookPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewWebhookPlatform() error = %v", err)
	}

	if got := p.Name(); got != "webhook" {
		t.Errorf("Name() = %v, want 'webhook'", got)
	}
}

func TestWebhookPlatform_ValidateTarget(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL: "https://example.com/webhook",
	}
	p, err := NewWebhookPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewWebhookPlatform() error = %v", err)
	}

	webhook := p.(*WebhookPlatform)

	tests := []struct {
		name      string
		target    target.Target
		wantError bool
	}{
		{
			name:      "valid webhook target",
			target:    target.Target{Type: "webhook", Value: "https://example.com/endpoint"},
			wantError: false,
		},
		{
			name:      "invalid target type",
			target:    target.Target{Type: "email", Value: "test@example.com"},
			wantError: true,
		},
		{
			name:      "empty value",
			target:    target.Target{Type: "webhook", Value: ""},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := webhook.ValidateTarget(tt.target)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTarget() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestWebhookPlatform_GetCapabilities(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL: "https://example.com/webhook",
	}
	p, err := NewWebhookPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewWebhookPlatform() error = %v", err)
	}

	caps := p.GetCapabilities()

	if caps.Name != "webhook" {
		t.Errorf("GetCapabilities() Name = %v, want 'webhook'", caps.Name)
	}

	if len(caps.SupportedTargetTypes) == 0 {
		t.Error("GetCapabilities() SupportedTargetTypes is empty")
	}

	// Check that webhook is in supported target types
	found := false
	for _, targetType := range caps.SupportedTargetTypes {
		if targetType == "webhook" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetCapabilities() should support webhook target type")
	}

	if len(caps.SupportedFormats) == 0 {
		t.Error("GetCapabilities() SupportedFormats is empty")
	}

	if caps.MaxMessageSize == 0 {
		t.Error("GetCapabilities() MaxMessageSize should be set")
	}
}

func TestWebhookPlatform_Close(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL: "https://example.com/webhook",
	}
	p, err := NewWebhookPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewWebhookPlatform() error = %v", err)
	}

	// Close should not panic
	err = p.Close()
	if err != nil {
		t.Errorf("Close() unexpected error = %v", err)
	}
}

func TestWebhookConfig_Defaults(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL: "https://example.com/webhook",
		// No other fields specified
	}
	p, err := NewWebhookPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewWebhookPlatform() error = %v", err)
	}

	webhook := p.(*WebhookPlatform)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"default method", webhook.config.Method, "POST"},
		{"default content type", webhook.config.ContentType, "application/json"},
		{"default timeout", webhook.config.Timeout, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestWebhookConfig_CustomValues(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL:         "https://example.com/webhook",
		Method:      "PUT",
		ContentType: "application/xml",
		Timeout:     90 * time.Second,
	}
	p, err := NewWebhookPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewWebhookPlatform() error = %v", err)
	}

	webhook := p.(*WebhookPlatform)

	if webhook.config.Method != "PUT" {
		t.Errorf("Custom method = %v, want PUT", webhook.config.Method)
	}
	if webhook.config.ContentType != "application/xml" {
		t.Errorf("Custom content type = %v, want application/xml", webhook.config.ContentType)
	}
	if webhook.config.Timeout != 90*time.Second {
		t.Errorf("Custom timeout = %v, want 90s", webhook.config.Timeout)
	}
}

func TestWebhookConfig_Headers(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer token123",
		"X-Custom":      "custom-value",
	}
	cfg := &config.WebhookConfig{
		URL:     "https://example.com/webhook",
		Headers: headers,
	}
	p, err := NewWebhookPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewWebhookPlatform() error = %v", err)
	}

	webhook := p.(*WebhookPlatform)

	if len(webhook.config.Headers) != 2 {
		t.Errorf("Headers count = %d, want 2", len(webhook.config.Headers))
	}
	if webhook.config.Headers["Authorization"] != "Bearer token123" {
		t.Error("Authorization header not set correctly")
	}
	if webhook.config.Headers["X-Custom"] != "custom-value" {
		t.Error("X-Custom header not set correctly")
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
