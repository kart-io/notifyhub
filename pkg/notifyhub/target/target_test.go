// Package target tests provide comprehensive testing for target management functionality
package target

import (
	"strings"
	"testing"
)

func TestNewTarget(t *testing.T) {
	tests := []struct {
		name         string
		targetType   string
		value        string
		platform     string
		wantType     string
		wantValue    string
		wantPlatform string
	}{
		{
			name:         "basic email target",
			targetType:   TargetTypeEmail,
			value:        "user@example.com",
			platform:     PlatformEmail,
			wantType:     "email",
			wantValue:    "user@example.com",
			wantPlatform: "email",
		},
		{
			name:         "phone target",
			targetType:   TargetTypePhone,
			value:        "+1234567890",
			platform:     PlatformSMS,
			wantType:     "phone",
			wantValue:    "+1234567890",
			wantPlatform: "sms",
		},
		{
			name:         "feishu user target",
			targetType:   TargetTypeUser,
			value:        "ou_123456",
			platform:     PlatformFeishu,
			wantType:     "user",
			wantValue:    "ou_123456",
			wantPlatform: "feishu",
		},
		{
			name:         "webhook target",
			targetType:   TargetTypeWebhook,
			value:        "https://example.com/webhook",
			platform:     PlatformWebhook,
			wantType:     "webhook",
			wantValue:    "https://example.com/webhook",
			wantPlatform: "webhook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := NewTarget(tt.targetType, tt.value, tt.platform)

			if target.Type != tt.wantType {
				t.Errorf("NewTarget() Type = %v, want %v", target.Type, tt.wantType)
			}
			if target.Value != tt.wantValue {
				t.Errorf("NewTarget() Value = %v, want %v", target.Value, tt.wantValue)
			}
			if target.Platform != tt.wantPlatform {
				t.Errorf("NewTarget() Platform = %v, want %v", target.Platform, tt.wantPlatform)
			}
		})
	}
}

func TestTargetString(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   string
	}{
		{
			name:   "email target string",
			target: Target{Type: "email", Value: "user@example.com", Platform: "email"},
			want:   "email:user@example.com@email",
		},
		{
			name:   "phone target string",
			target: Target{Type: "phone", Value: "+1234567890", Platform: "sms"},
			want:   "phone:+1234567890@sms",
		},
		{
			name:   "feishu group target string",
			target: Target{Type: "group", Value: "og_123456", Platform: "feishu"},
			want:   "group:og_123456@feishu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.String(); got != tt.want {
				t.Errorf("Target.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTargetIsValid(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   bool
	}{
		{
			name:   "valid email target",
			target: Target{Type: "email", Value: "user@example.com", Platform: "email"},
			want:   true,
		},
		{
			name:   "valid phone target",
			target: Target{Type: "phone", Value: "+1234567890", Platform: "sms"},
			want:   true,
		},
		{
			name:   "empty type - invalid",
			target: Target{Type: "", Value: "user@example.com", Platform: "email"},
			want:   false,
		},
		{
			name:   "empty value - invalid",
			target: Target{Type: "email", Value: "", Platform: "email"},
			want:   false,
		},
		{
			name:   "empty platform - valid (platform can be auto-detected)",
			target: Target{Type: "email", Value: "user@example.com", Platform: ""},
			want:   true,
		},
		{
			name:   "all empty - invalid",
			target: Target{Type: "", Value: "", Platform: ""},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.IsValid(); got != tt.want {
				t.Errorf("Target.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTargetHelperMethods(t *testing.T) {
	emailTarget := Target{Type: "email", Value: "user@example.com", Platform: "email"}
	phoneTarget := Target{Type: "phone", Value: "+1234567890", Platform: "sms"}
	webhookTarget := Target{Type: "webhook", Value: "https://example.com/webhook", Platform: "webhook"}
	feishuTarget := Target{Type: "user", Value: "ou_123456", Platform: "feishu"}

	// Test IsEmail
	if !emailTarget.IsEmail() {
		t.Error("emailTarget.IsEmail() should be true")
	}
	if phoneTarget.IsEmail() {
		t.Error("phoneTarget.IsEmail() should be false")
	}

	// Test IsPhone
	if !phoneTarget.IsPhone() {
		t.Error("phoneTarget.IsPhone() should be true")
	}
	if emailTarget.IsPhone() {
		t.Error("emailTarget.IsPhone() should be false")
	}

	// Test IsWebhook
	if !webhookTarget.IsWebhook() {
		t.Error("webhookTarget.IsWebhook() should be true")
	}
	if emailTarget.IsWebhook() {
		t.Error("emailTarget.IsWebhook() should be false")
	}

	// Test IsFeishu
	if !feishuTarget.IsFeishu() {
		t.Error("feishuTarget.IsFeishu() should be true")
	}
	if emailTarget.IsFeishu() {
		t.Error("emailTarget.IsFeishu() should be false")
	}
}

func TestTargetValidate(t *testing.T) {
	tests := []struct {
		name      string
		target    Target
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid email",
			target:    Target{Type: "email", Value: "user@example.com", Platform: "email"},
			wantError: false,
		},
		{
			name:      "invalid email format",
			target:    Target{Type: "email", Value: "invalid-email", Platform: "email"},
			wantError: true,
			errorMsg:  "invalid email format",
		},
		{
			name:      "valid phone E164 format",
			target:    Target{Type: "phone", Value: "+1234567890", Platform: "sms"},
			wantError: false,
		},
		{
			name:      "invalid phone format",
			target:    Target{Type: "phone", Value: "123-456-7890", Platform: "sms"},
			wantError: true,
			errorMsg:  "invalid phone number format",
		},
		{
			name:      "valid webhook URL",
			target:    Target{Type: "webhook", Value: "https://example.com/webhook", Platform: "webhook"},
			wantError: false,
		},
		{
			name:      "invalid webhook URL",
			target:    Target{Type: "webhook", Value: "not-a-url", Platform: "webhook"},
			wantError: true,
			errorMsg:  "invalid webhook URL",
		},
		{
			name:      "valid user ID",
			target:    Target{Type: "user", Value: "ou_123456", Platform: "feishu"},
			wantError: false,
		},
		{
			name:      "invalid user ID (too short)",
			target:    Target{Type: "user", Value: "u", Platform: "feishu"},
			wantError: true,
			errorMsg:  "invalid ID format",
		},
		{
			name:      "empty value",
			target:    Target{Type: "email", Value: "", Platform: "email"},
			wantError: true,
			errorMsg:  "target value cannot be empty",
		},
		{
			name:      "unsupported type",
			target:    Target{Type: "invalid", Value: "some-value", Platform: "some-platform"},
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
