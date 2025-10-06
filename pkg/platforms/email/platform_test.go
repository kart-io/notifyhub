package email

import (
	"context"
	"errors"
	"testing"

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

func TestNewEmailPlatform(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.EmailConfig
		wantError bool
		errorMsg  string
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
			errorMsg:  "email configuration cannot be nil",
		},
		{
			name: "missing host",
			config: &config.EmailConfig{
				Port: 587,
				From: "sender@example.com",
			},
			wantError: true,
			errorMsg:  "host is required",
		},
		{
			name: "missing port",
			config: &config.EmailConfig{
				Host: "smtp.example.com",
				From: "sender@example.com",
			},
			wantError: true,
			errorMsg:  "port is required",
		},
		{
			name: "missing from",
			config: &config.EmailConfig{
				Host: "smtp.example.com",
				Port: 587,
			},
			wantError: true,
			errorMsg:  "from address is required",
		},
		{
			name: "valid config",
			config: &config.EmailConfig{
				Host:     "smtp.example.com",
				Port:     587,
				From:     "sender@example.com",
				Username: "user",
				Password: "pass",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewEmailPlatform(tt.config, &mockLogger{})

			if tt.wantError {
				if err == nil {
					t.Errorf("NewEmailPlatform() expected error, got nil")
				}
				if tt.errorMsg != "" && err != nil {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("NewEmailPlatform() error = %v, want error containing %v", err, tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("NewEmailPlatform() unexpected error = %v", err)
				}
				if p == nil {
					t.Error("NewEmailPlatform() returned nil platform")
				}
			}
		})
	}
}

func TestEmailPlatform_Name(t *testing.T) {
	cfg := &config.EmailConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "sender@example.com",
	}
	p, err := NewEmailPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewEmailPlatform() error = %v", err)
	}

	if got := p.Name(); got != "email" {
		t.Errorf("Name() = %v, want 'email'", got)
	}
}

func TestEmailPlatform_ValidateTarget(t *testing.T) {
	cfg := &config.EmailConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "sender@example.com",
	}
	p, err := NewEmailPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewEmailPlatform() error = %v", err)
	}

	emailPlatform := p.(*EmailPlatform)

	tests := []struct {
		name      string
		target    target.Target
		wantError bool
	}{
		{
			name:      "valid email",
			target:    target.NewEmail("user@example.com"),
			wantError: false,
		},
		{
			name:      "wrong target type",
			target:    target.Target{Type: "phone", Value: "123456"},
			wantError: true,
		},
		{
			name:      "empty value",
			target:    target.Target{Type: "email", Value: ""},
			wantError: true,
		},
		{
			name:      "invalid email format",
			target:    target.Target{Type: "email", Value: "invalid-email"},
			wantError: true,
		},
		{
			name:      "invalid email missing @",
			target:    target.Target{Type: "email", Value: "invalid.com"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := emailPlatform.ValidateTarget(tt.target)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTarget() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestEmailPlatform_GetCapabilities(t *testing.T) {
	cfg := &config.EmailConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "sender@example.com",
	}
	p, err := NewEmailPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewEmailPlatform() error = %v", err)
	}

	caps := p.GetCapabilities()

	if caps.Name != "email" {
		t.Errorf("GetCapabilities() Name = %v, want 'email'", caps.Name)
	}

	if len(caps.SupportedTargetTypes) == 0 {
		t.Error("GetCapabilities() SupportedTargetTypes is empty")
	}

	if caps.SupportedTargetTypes[0] != "email" {
		t.Errorf("GetCapabilities() SupportedTargetTypes[0] = %v, want 'email'", caps.SupportedTargetTypes[0])
	}

	if len(caps.SupportedFormats) < 2 {
		t.Errorf("GetCapabilities() SupportedFormats = %v, want at least 2 formats", caps.SupportedFormats)
	}

	if caps.MaxMessageSize == 0 {
		t.Error("GetCapabilities() MaxMessageSize should be set")
	}

	if !caps.SupportsAttachments {
		t.Error("GetCapabilities() SupportsAttachments should be true")
	}

	if len(caps.RequiredSettings) == 0 {
		t.Error("GetCapabilities() RequiredSettings is empty")
	}
}

func TestEmailPlatform_Send_NilMessage(t *testing.T) {
	cfg := &config.EmailConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "sender@example.com",
	}
	p, err := NewEmailPlatform(cfg, &mockLogger{})
	if err != nil {
		t.Fatalf("NewEmailPlatform() error = %v", err)
	}

	ctx := context.Background()
	targets := []target.Target{target.NewEmail("user@example.com")}

	results, err := p.Send(ctx, nil, targets)

	if err == nil {
		t.Error("Send() with nil message should return error")
	}
	if results != nil {
		t.Error("Send() with nil message should return nil results")
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "valid simple email",
			email: "user@example.com",
			want:  true,
		},
		{
			name:  "valid email with subdomain",
			email: "user@mail.example.com",
			want:  true,
		},
		{
			name:  "valid email with plus",
			email: "user+tag@example.com",
			want:  true,
		},
		{
			name:  "valid email with dots",
			email: "first.last@example.com",
			want:  true,
		},
		{
			name:  "invalid - no @",
			email: "userexample.com",
			want:  false,
		},
		{
			name:  "invalid - empty",
			email: "",
			want:  false,
		},
		{
			name:  "invalid - no domain",
			email: "user@",
			want:  false,
		},
		{
			name:  "invalid - no local part",
			email: "@example.com",
			want:  false,
		},
		{
			name:  "invalid - double @",
			email: "user@@example.com",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidEmail(tt.email); got != tt.want {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestConvertToInternalConfig(t *testing.T) {
	cfg := &config.EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		From:     "sender@example.com",
		Username: "user",
		Password: "pass",
		UseTLS:   true,
	}

	internal := convertToInternalConfig(cfg)

	if internal.SMTPHost != cfg.Host {
		t.Errorf("SMTPHost = %v, want %v", internal.SMTPHost, cfg.Host)
	}
	if internal.SMTPPort != cfg.Port {
		t.Errorf("SMTPPort = %v, want %v", internal.SMTPPort, cfg.Port)
	}
	if internal.From != cfg.From {
		t.Errorf("From = %v, want %v", internal.From, cfg.From)
	}
	if internal.Username != cfg.Username {
		t.Errorf("Username = %v, want %v", internal.Username, cfg.Username)
	}
	if internal.Password != cfg.Password {
		t.Errorf("Password = %v, want %v", internal.Password, cfg.Password)
	}
	if internal.UseTLS != cfg.UseTLS {
		t.Errorf("UseTLS = %v, want %v", internal.UseTLS, cfg.UseTLS)
	}
}

func TestGenerateShortID(t *testing.T) {
	// Test that generateShortID produces unique IDs
	ids := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		id := generateShortID()

		if id == "" {
			t.Error("generateShortID() returned empty string")
		}

		if len(id) != 8 {
			t.Errorf("generateShortID() length = %v, want 8", len(id))
		}

		if ids[id] {
			t.Errorf("generateShortID() generated duplicate ID: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != iterations {
		t.Errorf("Expected %d unique IDs, got %d", iterations, len(ids))
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{"a less than b", 5, 10, 5},
		{"b less than a", 10, 5, 5},
		{"equal", 7, 7, 7},
		{"zero and positive", 0, 5, 0},
		{"negative and positive", -3, 5, -3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := min(tt.a, tt.b); got != tt.want {
				t.Errorf("min(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestFormatErrorForUser(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "standard error",
			err:  errors.New("connection failed"),
			want: "connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatErrorForUser(tt.err)
			if !contains(got, tt.want) {
				t.Errorf("FormatErrorForUser() = %v, want to contain %v", got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
