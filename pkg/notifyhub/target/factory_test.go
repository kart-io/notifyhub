// Package target tests provide testing for target factory functionality
package target

import (
	"testing"
)

func TestNewEmailTarget(t *testing.T) {
	email := "user@example.com"
	target := NewEmailTarget(email)

	expected := Target{
		Type:     TargetTypeEmail,
		Value:    email,
		Platform: PlatformEmail,
	}

	if target.Type != expected.Type {
		t.Errorf("NewEmailTarget() Type = %v, want %v", target.Type, expected.Type)
	}
	if target.Value != expected.Value {
		t.Errorf("NewEmailTarget() Value = %v, want %v", target.Value, expected.Value)
	}
	if target.Platform != expected.Platform {
		t.Errorf("NewEmailTarget() Platform = %v, want %v", target.Platform, expected.Platform)
	}
}

func TestNewPhoneTarget(t *testing.T) {
	phone := "+1234567890"
	target := NewPhoneTarget(phone)

	expected := Target{
		Type:     TargetTypePhone,
		Value:    phone,
		Platform: PlatformSMS,
	}

	if target.Type != expected.Type {
		t.Errorf("NewPhoneTarget() Type = %v, want %v", target.Type, expected.Type)
	}
	if target.Value != expected.Value {
		t.Errorf("NewPhoneTarget() Value = %v, want %v", target.Value, expected.Value)
	}
	if target.Platform != expected.Platform {
		t.Errorf("NewPhoneTarget() Platform = %v, want %v", target.Platform, expected.Platform)
	}
}

func TestNewFeishuUserTarget(t *testing.T) {
	userID := "ou_123456789"
	target := NewFeishuUserTarget(userID)

	expected := Target{
		Type:     TargetTypeUser,
		Value:    userID,
		Platform: PlatformFeishu,
	}

	if target.Type != expected.Type {
		t.Errorf("NewFeishuUserTarget() Type = %v, want %v", target.Type, expected.Type)
	}
	if target.Value != expected.Value {
		t.Errorf("NewFeishuUserTarget() Value = %v, want %v", target.Value, expected.Value)
	}
	if target.Platform != expected.Platform {
		t.Errorf("NewFeishuUserTarget() Platform = %v, want %v", target.Platform, expected.Platform)
	}
}

func TestNewFeishuGroupTarget(t *testing.T) {
	groupID := "og_123456789"
	target := NewFeishuGroupTarget(groupID)

	expected := Target{
		Type:     TargetTypeGroup,
		Value:    groupID,
		Platform: PlatformFeishu,
	}

	if target.Type != expected.Type {
		t.Errorf("NewFeishuGroupTarget() Type = %v, want %v", target.Type, expected.Type)
	}
	if target.Value != expected.Value {
		t.Errorf("NewFeishuGroupTarget() Value = %v, want %v", target.Value, expected.Value)
	}
	if target.Platform != expected.Platform {
		t.Errorf("NewFeishuGroupTarget() Platform = %v, want %v", target.Platform, expected.Platform)
	}
}

func TestNewWebhookTarget(t *testing.T) {
	url := "https://example.com/webhook"
	target := NewWebhookTarget(url)

	expected := Target{
		Type:     TargetTypeWebhook,
		Value:    url,
		Platform: PlatformWebhook,
	}

	if target.Type != expected.Type {
		t.Errorf("NewWebhookTarget() Type = %v, want %v", target.Type, expected.Type)
	}
	if target.Value != expected.Value {
		t.Errorf("NewWebhookTarget() Value = %v, want %v", target.Value, expected.Value)
	}
	if target.Platform != expected.Platform {
		t.Errorf("NewWebhookTarget() Platform = %v, want %v", target.Platform, expected.Platform)
	}
}

func TestFactoryFunctions(t *testing.T) {
	tests := []struct {
		name     string
		factory  func() Target
		wantType string
		want     Target
	}{
		{
			name:     "email factory",
			factory:  func() Target { return NewEmailTarget("test@example.com") },
			wantType: "email",
			want:     Target{Type: "email", Value: "test@example.com", Platform: "email"},
		},
		{
			name:     "phone factory",
			factory:  func() Target { return NewPhoneTarget("+1234567890") },
			wantType: "phone",
			want:     Target{Type: "phone", Value: "+1234567890", Platform: "sms"},
		},
		{
			name:     "feishu user factory",
			factory:  func() Target { return NewFeishuUserTarget("ou_123") },
			wantType: "user",
			want:     Target{Type: "user", Value: "ou_123", Platform: "feishu"},
		},
		{
			name:     "feishu group factory",
			factory:  func() Target { return NewFeishuGroupTarget("og_123") },
			wantType: "group",
			want:     Target{Type: "group", Value: "og_123", Platform: "feishu"},
		},
		{
			name:     "webhook factory",
			factory:  func() Target { return NewWebhookTarget("https://example.com") },
			wantType: "webhook",
			want:     Target{Type: "webhook", Value: "https://example.com", Platform: "webhook"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.factory()

			if got.Type != tt.want.Type {
				t.Errorf("Factory Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Value != tt.want.Value {
				t.Errorf("Factory Value = %v, want %v", got.Value, tt.want.Value)
			}
			if got.Platform != tt.want.Platform {
				t.Errorf("Factory Platform = %v, want %v", got.Platform, tt.want.Platform)
			}

			// Verify target is valid
			if !got.IsValid() {
				t.Errorf("Factory should produce valid target")
			}

			// Verify helper methods work correctly
			switch tt.wantType {
			case "email":
				if !got.IsEmail() {
					t.Errorf("Email target should return true for IsEmail()")
				}
			case "phone":
				if !got.IsPhone() {
					t.Errorf("Phone target should return true for IsPhone()")
				}
			case "webhook":
				if !got.IsWebhook() {
					t.Errorf("Webhook target should return true for IsWebhook()")
				}
			case "user", "group":
				if got.Platform == PlatformFeishu && !got.IsFeishu() {
					t.Errorf("Feishu target should return true for IsFeishu()")
				}
			}
		})
	}
}
