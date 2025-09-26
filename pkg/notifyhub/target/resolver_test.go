// Package target tests provide testing for target resolver functionality
package target

import (
	"strings"
	"testing"
)

func TestAutoDetectTarget(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		wantType     string
		wantPlatform string
	}{
		{
			name:         "detect email",
			value:        "user@example.com",
			wantType:     TargetTypeEmail,
			wantPlatform: PlatformEmail,
		},
		{
			name:         "detect complex email",
			value:        "test.user+tag@sub.example.co.uk",
			wantType:     TargetTypeEmail,
			wantPlatform: PlatformEmail,
		},
		{
			name:         "detect phone number",
			value:        "+1234567890",
			wantType:     TargetTypePhone,
			wantPlatform: PlatformSMS,
		},
		{
			name:         "detect international phone",
			value:        "+861234567890",
			wantType:     TargetTypePhone,
			wantPlatform: PlatformSMS,
		},
		{
			name:         "detect HTTPS webhook",
			value:        "https://example.com/webhook",
			wantType:     TargetTypeWebhook,
			wantPlatform: PlatformWebhook,
		},
		{
			name:         "detect HTTP webhook",
			value:        "http://localhost:8080/webhook",
			wantType:     TargetTypeWebhook,
			wantPlatform: PlatformWebhook,
		},
		{
			name:         "detect feishu user ID (ou_)",
			value:        "ou_123456789abcdef",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformFeishu,
		},
		{
			name:         "detect feishu user ID (oc_)",
			value:        "oc_123456789abcdef",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformFeishu,
		},
		{
			name:         "detect feishu group ID",
			value:        "og_123456789abcdef",
			wantType:     TargetTypeGroup,
			wantPlatform: PlatformFeishu,
		},
		{
			name:         "unknown format defaults to user/auto",
			value:        "some-unknown-value",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformAuto,
		},
		{
			name:         "username-like defaults to user/auto",
			value:        "username123",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformAuto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := AutoDetectTarget(tt.value)

			if target.Type != tt.wantType {
				t.Errorf("AutoDetectTarget(%v) Type = %v, want %v", tt.value, target.Type, tt.wantType)
			}
			if target.Platform != tt.wantPlatform {
				t.Errorf("AutoDetectTarget(%v) Platform = %v, want %v", tt.value, target.Platform, tt.wantPlatform)
			}
			if target.Value != tt.value {
				t.Errorf("AutoDetectTarget(%v) Value = %v, want %v", tt.value, target.Value, tt.value)
			}

			// Verify the target is valid (has required fields)
			if !target.IsValid() {
				t.Errorf("AutoDetectTarget(%v) should produce valid target", tt.value)
			}
		})
	}
}

func TestDetectTargetTypeAndPlatform(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		wantType     string
		wantPlatform string
	}{
		{
			name:         "email detection",
			value:        "test@domain.com",
			wantType:     TargetTypeEmail,
			wantPlatform: PlatformEmail,
		},
		{
			name:         "phone detection",
			value:        "+1234567890",
			wantType:     TargetTypePhone,
			wantPlatform: PlatformSMS,
		},
		{
			name:         "webhook detection",
			value:        "https://api.example.com/hook",
			wantType:     TargetTypeWebhook,
			wantPlatform: PlatformWebhook,
		},
		{
			name:         "feishu user detection",
			value:        "ou_abcd123",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformFeishu,
		},
		{
			name:         "feishu group detection",
			value:        "og_xyz789",
			wantType:     TargetTypeGroup,
			wantPlatform: PlatformFeishu,
		},
		{
			name:         "unknown format",
			value:        "random-text",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformAuto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotPlatform := detectTargetTypeAndPlatform(tt.value)

			if gotType != tt.wantType {
				t.Errorf("detectTargetTypeAndPlatform(%v) type = %v, want %v", tt.value, gotType, tt.wantType)
			}
			if gotPlatform != tt.wantPlatform {
				t.Errorf("detectTargetTypeAndPlatform(%v) platform = %v, want %v", tt.value, gotPlatform, tt.wantPlatform)
			}
		})
	}
}

func TestTargetValidationMethods(t *testing.T) {
	tests := []struct {
		name      string
		target    Target
		wantError bool
		errorMsg  string
	}{
		// Email validation tests
		{
			name:      "valid email",
			target:    Target{Type: "email", Value: "user@example.com"},
			wantError: false,
		},
		{
			name:      "invalid email - no @",
			target:    Target{Type: "email", Value: "userexample.com"},
			wantError: true,
			errorMsg:  "invalid email format",
		},
		{
			name:      "invalid email - no domain",
			target:    Target{Type: "email", Value: "user@"},
			wantError: true,
			errorMsg:  "invalid email format",
		},
		{
			name:      "invalid email - no local part",
			target:    Target{Type: "email", Value: "@example.com"},
			wantError: true,
			errorMsg:  "invalid email format",
		},

		// Phone validation tests
		{
			name:      "valid phone E164",
			target:    Target{Type: "phone", Value: "+1234567890"},
			wantError: false,
		},
		{
			name:      "valid international phone",
			target:    Target{Type: "phone", Value: "+861234567890123"},
			wantError: false,
		},
		{
			name:      "invalid phone - no +",
			target:    Target{Type: "phone", Value: "1234567890"},
			wantError: true,
			errorMsg:  "invalid phone number format",
		},
		{
			name:      "invalid phone - starts with +0",
			target:    Target{Type: "phone", Value: "+0123456789"},
			wantError: true,
			errorMsg:  "invalid phone number format",
		},
		{
			name:      "invalid phone - too long",
			target:    Target{Type: "phone", Value: "+123456789012345678"},
			wantError: true,
			errorMsg:  "invalid phone number format",
		},

		// Webhook validation tests
		{
			name:      "valid HTTPS webhook",
			target:    Target{Type: "webhook", Value: "https://example.com/webhook"},
			wantError: false,
		},
		{
			name:      "valid HTTP webhook",
			target:    Target{Type: "webhook", Value: "http://localhost:8080/webhook"},
			wantError: false,
		},
		{
			name:      "invalid webhook - no protocol",
			target:    Target{Type: "webhook", Value: "example.com/webhook"},
			wantError: true,
			errorMsg:  "invalid webhook URL",
		},
		{
			name:      "invalid webhook - wrong protocol",
			target:    Target{Type: "webhook", Value: "ftp://example.com/webhook"},
			wantError: true,
			errorMsg:  "invalid webhook URL",
		},

		// ID validation tests
		{
			name:      "valid user ID",
			target:    Target{Type: "user", Value: "ou_123456"},
			wantError: false,
		},
		{
			name:      "valid group ID",
			target:    Target{Type: "group", Value: "og_abcdef123"},
			wantError: false,
		},
		{
			name:      "valid channel ID",
			target:    Target{Type: "channel", Value: "channel_123"},
			wantError: false,
		},
		{
			name:      "invalid ID - too short",
			target:    Target{Type: "user", Value: "u"},
			wantError: true,
			errorMsg:  "invalid ID format",
		},

		// Generic validation tests
		{
			name:      "empty value",
			target:    Target{Type: "email", Value: ""},
			wantError: true,
			errorMsg:  "target value cannot be empty",
		},
		{
			name:      "unsupported type",
			target:    Target{Type: "unsupported", Value: "some-value"},
			wantError: true,
			errorMsg:  "unsupported target type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.target.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("Target.Validate() expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Target.Validate() error = %v, want error containing %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Target.Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestAutoDetectEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		description  string
		wantType     string
		wantPlatform string
	}{
		{
			name:         "email with subdomain",
			value:        "user@mail.sub.example.com",
			description:  "should detect emails with complex domains",
			wantType:     TargetTypeEmail,
			wantPlatform: PlatformEmail,
		},
		{
			name:         "short phone number",
			value:        "+12345",
			description:  "should detect short but valid E164 phone numbers",
			wantType:     TargetTypePhone,
			wantPlatform: PlatformSMS,
		},
		{
			name:         "webhook with query params",
			value:        "https://example.com/webhook?token=abc123",
			description:  "should detect webhooks with query parameters",
			wantType:     TargetTypeWebhook,
			wantPlatform: PlatformWebhook,
		},
		{
			name:         "feishu ID with underscores",
			value:        "ou_abc_def_123",
			description:  "should detect feishu IDs with multiple underscores",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformFeishu,
		},
		{
			name:         "empty string",
			value:        "",
			description:  "should handle empty string gracefully",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformAuto,
		},
		{
			name:         "just @ symbol",
			value:        "@",
			description:  "should not detect malformed email",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformAuto,
		},
		{
			name:         "just + symbol",
			value:        "+",
			description:  "should not detect malformed phone",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformAuto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := AutoDetectTarget(tt.value)

			if target.Type != tt.wantType {
				t.Errorf("AutoDetectTarget(%v) Type = %v, want %v (%s)", tt.value, target.Type, tt.wantType, tt.description)
			}
			if target.Platform != tt.wantPlatform {
				t.Errorf("AutoDetectTarget(%v) Platform = %v, want %v (%s)", tt.value, target.Platform, tt.wantPlatform, tt.description)
			}
		})
	}
}
