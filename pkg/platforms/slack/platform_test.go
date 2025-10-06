package slack

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/config"
	"github.com/kart-io/notifyhub/pkg/target"
	"github.com/kart-io/notifyhub/pkg/utils/logger"
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

func TestNewSlackPlatform(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.SlackConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config with webhook",
			config: &config.SlackConfig{
				WebhookURL: "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX",
			},
			wantError: false,
		},
		{
			name: "valid config with token",
			config: &config.SlackConfig{
				Token:   "xxxxx-xxxx-xxxx-xxxxxxxxxxxxxxxxxxxx",
				Channel: "#notifications",
			},
			wantError: false,
		},
		{
			name: "valid config with webhook and options",
			config: &config.SlackConfig{
				WebhookURL: "https://hooks.slack.com/services/TEST",
				Channel:    "#alerts",
				Username:   "CustomBot",
				IconEmoji:  ":rocket:",
			},
			wantError: false,
		},
		{
			name: "valid config with custom timeout",
			config: &config.SlackConfig{
				WebhookURL: "https://hooks.slack.com/services/TEST",
				Timeout:    60 * time.Second,
			},
			wantError: false,
		},
		{
			name:      "missing both webhook and token",
			config:    &config.SlackConfig{},
			wantError: true,
			errorMsg:  "webhook URL or token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewSlackPlatform(tt.config, &mockLogger{})

			if tt.wantError {
				if err == nil {
					t.Errorf("NewSlackPlatform() expected error, got nil")
				}
				if tt.errorMsg != "" && err != nil {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("NewSlackPlatform() error = %v, want error containing %v", err, tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("NewSlackPlatform() unexpected error = %v", err)
				}
				if p == nil {
					t.Error("NewSlackPlatform() returned nil platform")
				}

				// Verify defaults are set
				slack := p.(*SlackPlatform)
				if tt.config.Timeout == 0 && slack.config.Timeout != 30*time.Second {
					t.Errorf("Default timeout = %v, want 30s", slack.config.Timeout)
				}
				if tt.config.Channel == "" && slack.config.Channel != "#general" {
					t.Errorf("Default channel = %v, want #general", slack.config.Channel)
				}
				if tt.config.Username == "" && slack.config.Username != "NotifyHub" {
					t.Errorf("Default username = %v, want NotifyHub", slack.config.Username)
				}
				if tt.config.IconEmoji == "" && tt.config.IconURL == "" && slack.config.IconEmoji != ":bell:" {
					t.Errorf("Default icon = %v, want :bell:", slack.config.IconEmoji)
				}
			}
		})
	}
}

func TestSlackPlatform_Name(t *testing.T) {
	cfg := &config.SlackConfig{
		WebhookURL: "https://hooks.slack.com/services/TEST",
	}
	p, err := NewSlackPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewSlackPlatform() error = %v", err)
	}

	if got := p.Name(); got != "slack" {
		t.Errorf("Name() = %v, want 'slack'", got)
	}
}

func TestSlackPlatform_ValidateTarget(t *testing.T) {
	cfg := &config.SlackConfig{
		WebhookURL: "https://hooks.slack.com/services/TEST",
	}
	p, err := NewSlackPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewSlackPlatform() error = %v", err)
	}

	slack := p.(*SlackPlatform)

	tests := []struct {
		name      string
		target    target.Target
		wantError bool
	}{
		{
			name:      "valid slack target",
			target:    target.Target{Type: "slack", Value: "#channel"},
			wantError: false,
		},
		{
			name:      "valid webhook target",
			target:    target.Target{Type: "webhook", Value: "https://hooks.slack.com/services/TEST"},
			wantError: false,
		},
		{
			name:      "invalid target type",
			target:    target.Target{Type: "email", Value: "test@example.com"},
			wantError: true,
		},
		{
			name:      "empty value",
			target:    target.Target{Type: "slack", Value: ""},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := slack.ValidateTarget(tt.target)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTarget() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSlackPlatform_GetCapabilities(t *testing.T) {
	cfg := &config.SlackConfig{
		WebhookURL: "https://hooks.slack.com/services/TEST",
	}
	p, err := NewSlackPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewSlackPlatform() error = %v", err)
	}

	caps := p.GetCapabilities()

	if caps.Name != "slack" {
		t.Errorf("GetCapabilities() Name = %v, want 'slack'", caps.Name)
	}

	if len(caps.SupportedTargetTypes) == 0 {
		t.Error("GetCapabilities() SupportedTargetTypes is empty")
	}

	// Check that slack is in supported target types
	found := false
	for _, targetType := range caps.SupportedTargetTypes {
		if targetType == "slack" || targetType == "webhook" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetCapabilities() should support slack or webhook target type")
	}

	if len(caps.SupportedFormats) == 0 {
		t.Error("GetCapabilities() SupportedFormats is empty")
	}

	if caps.MaxMessageSize == 0 {
		t.Error("GetCapabilities() MaxMessageSize should be set")
	}
}

func TestSlackPlatform_Close(t *testing.T) {
	cfg := &config.SlackConfig{
		WebhookURL: "https://hooks.slack.com/services/TEST",
	}
	p, err := NewSlackPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewSlackPlatform() error = %v", err)
	}

	// Close should not panic
	err = p.Close()
	if err != nil {
		t.Errorf("Close() unexpected error = %v", err)
	}
}

func TestSlackConfig_Defaults(t *testing.T) {
	cfg := &config.SlackConfig{
		WebhookURL: "https://hooks.slack.com/services/TEST",
		// No other fields specified
	}
	p, err := NewSlackPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewSlackPlatform() error = %v", err)
	}

	slack := p.(*SlackPlatform)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"default timeout", slack.config.Timeout, 30 * time.Second},
		{"default channel", slack.config.Channel, "#general"},
		{"default username", slack.config.Username, "NotifyHub"},
		{"default icon", slack.config.IconEmoji, ":bell:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestSlackPlatform_isSlackTarget(t *testing.T) {
	cfg := &config.SlackConfig{
		WebhookURL: "https://hooks.slack.com/services/TEST",
	}
	p, err := NewSlackPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewSlackPlatform() error = %v", err)
	}

	slack := p.(*SlackPlatform)

	tests := []struct {
		name   string
		target target.Target
		want   bool
	}{
		{
			name:   "slack type",
			target: target.Target{Type: "slack", Value: "#test"},
			want:   true,
		},
		{
			name:   "webhook type",
			target: target.Target{Type: "webhook", Value: "https://hooks.slack.com/test"},
			want:   true,
		},
		{
			name:   "email type",
			target: target.Target{Type: "email", Value: "test@example.com"},
			want:   false,
		},
		{
			name:   "feishu type",
			target: target.Target{Type: "feishu", Value: "test"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slack.isSlackTarget(tt.target); got != tt.want {
				t.Errorf("isSlackTarget() = %v, want %v", got, tt.want)
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
