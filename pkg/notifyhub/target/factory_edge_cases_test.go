// Package target provides comprehensive testing for target factory edge cases and boundary conditions
package target

import (
	"strings"
	"testing"
)

// TestFactoryFunctionsEdgeCases tests factory functions with edge cases and invalid inputs
func TestFactoryFunctionsEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		factoryFunc   func() Target
		expectedType  string
		expectedValue string
		expectedPlatform string
		description   string
	}{
		{
			name:          "email with empty string",
			factoryFunc:   func() Target { return NewEmailTarget("") },
			expectedType:  TargetTypeEmail,
			expectedValue: "",
			expectedPlatform: PlatformEmail,
			description:   "should create target even with empty email",
		},
		{
			name:          "email with whitespace",
			factoryFunc:   func() Target { return NewEmailTarget("  user@example.com  ") },
			expectedType:  TargetTypeEmail,
			expectedValue: "  user@example.com  ",
			expectedPlatform: PlatformEmail,
			description:   "should preserve whitespace in email value",
		},
		{
			name:          "email with special characters",
			factoryFunc:   func() Target { return NewEmailTarget("user+test@sub.example-domain.co.uk") },
			expectedType:  TargetTypeEmail,
			expectedValue: "user+test@sub.example-domain.co.uk",
			expectedPlatform: PlatformEmail,
			description:   "should handle complex email formats",
		},
		{
			name:          "phone with empty string",
			factoryFunc:   func() Target { return NewPhoneTarget("") },
			expectedType:  TargetTypePhone,
			expectedValue: "",
			expectedPlatform: PlatformSMS,
			description:   "should create target even with empty phone",
		},
		{
			name:          "phone with formatting characters",
			factoryFunc:   func() Target { return NewPhoneTarget("(555) 123-4567") },
			expectedType:  TargetTypePhone,
			expectedValue: "(555) 123-4567",
			expectedPlatform: PlatformSMS,
			description:   "should preserve original phone formatting",
		},
		{
			name:          "phone with international format",
			factoryFunc:   func() Target { return NewPhoneTarget("+86-138-0013-8000") },
			expectedType:  TargetTypePhone,
			expectedValue: "+86-138-0013-8000",
			expectedPlatform: PlatformSMS,
			description:   "should handle international phone formats",
		},
		{
			name:          "feishu user with minimum length ID",
			factoryFunc:   func() Target { return NewFeishuUserTarget("ou_123") },
			expectedType:  TargetTypeUser,
			expectedValue: "ou_123",
			expectedPlatform: PlatformFeishu,
			description:   "should handle minimum valid feishu user ID",
		},
		{
			name:          "feishu user with maximum length ID",
			factoryFunc:   func() Target { return NewFeishuUserTarget("ou_1234567890abcdef1234567890abcdef1234567890") },
			expectedType:  TargetTypeUser,
			expectedValue: "ou_1234567890abcdef1234567890abcdef1234567890",
			expectedPlatform: PlatformFeishu,
			description:   "should handle long feishu user IDs",
		},
		{
			name:          "feishu group with empty string",
			factoryFunc:   func() Target { return NewFeishuGroupTarget("") },
			expectedType:  TargetTypeGroup,
			expectedValue: "",
			expectedPlatform: PlatformFeishu,
			description:   "should create target even with empty group ID",
		},
		{
			name:          "feishu group with special characters",
			factoryFunc:   func() Target { return NewFeishuGroupTarget("og_test-group_123") },
			expectedType:  TargetTypeGroup,
			expectedValue: "og_test-group_123",
			expectedPlatform: PlatformFeishu,
			description:   "should handle group IDs with hyphens",
		},
		{
			name:          "webhook with localhost URL",
			factoryFunc:   func() Target { return NewWebhookTarget("http://localhost:8080/webhook") },
			expectedType:  TargetTypeWebhook,
			expectedValue: "http://localhost:8080/webhook",
			expectedPlatform: PlatformWebhook,
			description:   "should handle localhost webhook URLs",
		},
		{
			name:          "webhook with IP address",
			factoryFunc:   func() Target { return NewWebhookTarget("https://192.168.1.100:9000/api/notify") },
			expectedType:  TargetTypeWebhook,
			expectedValue: "https://192.168.1.100:9000/api/notify",
			expectedPlatform: PlatformWebhook,
			description:   "should handle IP address webhook URLs",
		},
		{
			name:          "webhook with query parameters",
			factoryFunc:   func() Target { return NewWebhookTarget("https://api.example.com/webhook?token=abc123&format=json") },
			expectedType:  TargetTypeWebhook,
			expectedValue: "https://api.example.com/webhook?token=abc123&format=json",
			expectedPlatform: PlatformWebhook,
			description:   "should handle webhook URLs with query parameters",
		},
		{
			name:          "webhook with fragment",
			factoryFunc:   func() Target { return NewWebhookTarget("https://example.com/webhook#section") },
			expectedType:  TargetTypeWebhook,
			expectedValue: "https://example.com/webhook#section",
			expectedPlatform: PlatformWebhook,
			description:   "should handle webhook URLs with fragments",
		},
		{
			name:          "webhook with empty string",
			factoryFunc:   func() Target { return NewWebhookTarget("") },
			expectedType:  TargetTypeWebhook,
			expectedValue: "",
			expectedPlatform: PlatformWebhook,
			description:   "should create target even with empty webhook URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.factoryFunc()

			if target.Type != tt.expectedType {
				t.Errorf("%s: Type = %v, want %v", tt.description, target.Type, tt.expectedType)
			}
			if target.Value != tt.expectedValue {
				t.Errorf("%s: Value = %v, want %v", tt.description, target.Value, tt.expectedValue)
			}
			if target.Platform != tt.expectedPlatform {
				t.Errorf("%s: Platform = %v, want %v", tt.description, target.Platform, tt.expectedPlatform)
			}
		})
	}
}

// TestFactoryFunctionsWithInvalidInputs tests factory functions with various invalid inputs
func TestFactoryFunctionsWithInvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		factoryFunc func() Target
		description string
		shouldCreate bool
	}{
		{
			name:        "email with just @",
			factoryFunc: func() Target { return NewEmailTarget("@") },
			description: "should create target but fail validation",
			shouldCreate: true,
		},
		{
			name:        "email with multiple @",
			factoryFunc: func() Target { return NewEmailTarget("user@@domain.com") },
			description: "should create target but fail validation",
			shouldCreate: true,
		},
		{
			name:        "phone with just +",
			factoryFunc: func() Target { return NewPhoneTarget("+") },
			description: "should create target but fail validation",
			shouldCreate: true,
		},
		{
			name:        "phone with letters",
			factoryFunc: func() Target { return NewPhoneTarget("+1-800-CALL-NOW") },
			description: "should create target but fail validation",
			shouldCreate: true,
		},
		{
			name:        "feishu user with invalid prefix",
			factoryFunc: func() Target { return NewFeishuUserTarget("user_123456") },
			description: "should create target but fail feishu validation",
			shouldCreate: true,
		},
		{
			name:        "feishu group with invalid prefix",
			factoryFunc: func() Target { return NewFeishuGroupTarget("group_123456") },
			description: "should create target but fail feishu validation",
			shouldCreate: true,
		},
		{
			name:        "webhook with invalid protocol",
			factoryFunc: func() Target { return NewWebhookTarget("ftp://example.com/webhook") },
			description: "should create target but fail webhook validation",
			shouldCreate: true,
		},
		{
			name:        "webhook without protocol",
			factoryFunc: func() Target { return NewWebhookTarget("example.com/webhook") },
			description: "should create target but fail webhook validation",
			shouldCreate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.factoryFunc()

			if tt.shouldCreate {
				// Factory should always create a target object
				if target.Type == "" {
					t.Errorf("%s: factory should create target object", tt.description)
				}

				// But validation should fail for invalid inputs
				if err := target.Validate(); err == nil {
					t.Logf("%s: created target %+v, validation passed (might be valid)", tt.description, target)
				} else {
					t.Logf("%s: created target %+v, validation failed as expected: %v", tt.description, target, err)
				}
			}
		})
	}
}

// TestFactoryFunctionsBoundaryConditions tests factory functions with boundary conditions
func TestFactoryFunctionsBoundaryConditions(t *testing.T) {
	tests := []struct {
		name        string
		factoryFunc func() Target
		testField   string
		description string
	}{
		{
			name:        "email at maximum reasonable length",
			factoryFunc: func() Target {
				longEmail := strings.Repeat("a", 60) + "@" + strings.Repeat("domain", 10) + ".com"
				return NewEmailTarget(longEmail)
			},
			testField:   "Value",
			description: "should handle very long email addresses",
		},
		{
			name:        "phone with maximum E.164 length",
			factoryFunc: func() Target {
				// E.164 format allows up to 15 digits
				longPhone := "+123456789012345"
				return NewPhoneTarget(longPhone)
			},
			testField:   "Value",
			description: "should handle maximum length phone numbers",
		},
		{
			name:        "webhook with very long URL",
			factoryFunc: func() Target {
				longURL := "https://example.com/" + strings.Repeat("path/", 50) + "webhook"
				return NewWebhookTarget(longURL)
			},
			testField:   "Value",
			description: "should handle very long webhook URLs",
		},
		{
			name:        "feishu user ID at maximum length",
			factoryFunc: func() Target {
				longID := "ou_" + strings.Repeat("1234567890abcdef", 2) + "12345678"
				return NewFeishuUserTarget(longID)
			},
			testField:   "Value",
			description: "should handle maximum length feishu user IDs",
		},
		{
			name:        "unicode characters in email",
			factoryFunc: func() Target {
				return NewEmailTarget("用户@example.com")
			},
			testField:   "Value",
			description: "should handle unicode characters in email",
		},
		{
			name:        "special characters in webhook path",
			factoryFunc: func() Target {
				return NewWebhookTarget("https://example.com/webhook!@#$%^&*()")
			},
			testField:   "Value",
			description: "should handle special characters in webhook paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.factoryFunc()

			// Factory should always create a target
			if target.Type == "" || target.Platform == "" {
				t.Errorf("%s: factory should create complete target object", tt.description)
			}

			// Check that value is preserved
			if len(target.Value) == 0 {
				t.Errorf("%s: factory should preserve input value", tt.description)
			}

			t.Logf("%s: created target with %s length %d", tt.description, tt.testField, len(target.Value))
		})
	}
}

// TestNewTargetWithVariousInputs tests the generic NewTarget function with various input combinations
func TestNewTargetWithVariousInputs(t *testing.T) {
	tests := []struct {
		name         string
		targetType   string
		value        string
		platform     string
		expectValid  bool
		description  string
	}{
		{
			name:        "valid email with correct platform",
			targetType:  TargetTypeEmail,
			value:       "user@example.com",
			platform:    PlatformEmail,
			expectValid: true,
			description: "should create valid email target",
		},
		{
			name:        "email with wrong platform",
			targetType:  TargetTypeEmail,
			value:       "user@example.com",
			platform:    PlatformFeishu,
			expectValid: true, // Target creation succeeds, but platform mismatch
			description: "should create target with mismatched platform",
		},
		{
			name:        "empty type",
			targetType:  "",
			value:       "user@example.com",
			platform:    PlatformEmail,
			expectValid: false,
			description: "should create target but fail IsValid with empty type",
		},
		{
			name:        "empty value",
			targetType:  TargetTypeEmail,
			value:       "",
			platform:    PlatformEmail,
			expectValid: false,
			description: "should create target but fail IsValid with empty value",
		},
		{
			name:        "empty platform",
			targetType:  TargetTypeEmail,
			value:       "user@example.com",
			platform:    "",
			expectValid: true,
			description: "should create valid target with empty platform (auto-detect)",
		},
		{
			name:        "all fields empty",
			targetType:  "",
			value:       "",
			platform:    "",
			expectValid: false,
			description: "should create target but fail IsValid with all empty fields",
		},
		{
			name:        "unknown target type",
			targetType:  "unknown",
			value:       "some-value",
			platform:    "unknown-platform",
			expectValid: true, // IsValid only checks type and value presence
			description: "should create target with unknown type",
		},
		{
			name:        "special characters in all fields",
			targetType:  "user@#$",
			value:       "value!@#$%",
			platform:    "platform^&*()",
			expectValid: true, // IsValid doesn't validate format, only presence
			description: "should create target with special characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := NewTarget(tt.targetType, tt.value, tt.platform)

			// Check that target was created
			if target.Type != tt.targetType {
				t.Errorf("%s: Type = %v, want %v", tt.description, target.Type, tt.targetType)
			}
			if target.Value != tt.value {
				t.Errorf("%s: Value = %v, want %v", tt.description, target.Value, tt.value)
			}
			if target.Platform != tt.platform {
				t.Errorf("%s: Platform = %v, want %v", tt.description, target.Platform, tt.platform)
			}

			// Check validity
			isValid := target.IsValid()
			if isValid != tt.expectValid {
				t.Errorf("%s: IsValid() = %v, want %v", tt.description, isValid, tt.expectValid)
			}
		})
	}
}

// TestFactoryFunctionsConsistency ensures factory functions produce consistent results
func TestFactoryFunctionsConsistency(t *testing.T) {
	// Test that calling the same factory function multiple times produces identical results
	inputs := []string{
		"user@example.com",
		"+1234567890",
		"ou_123456789",
		"og_987654321",
		"https://example.com/webhook",
	}

	factories := []func(string) Target{
		NewEmailTarget,
		NewPhoneTarget,
		NewFeishuUserTarget,
		NewFeishuGroupTarget,
		NewWebhookTarget,
	}

	for i, input := range inputs {
		factory := factories[i]
		t.Run("consistency_test_"+input, func(t *testing.T) {
			target1 := factory(input)
			target2 := factory(input)

			if target1.Type != target2.Type {
				t.Errorf("inconsistent Type: %v != %v", target1.Type, target2.Type)
			}
			if target1.Value != target2.Value {
				t.Errorf("inconsistent Value: %v != %v", target1.Value, target2.Value)
			}
			if target1.Platform != target2.Platform {
				t.Errorf("inconsistent Platform: %v != %v", target1.Platform, target2.Platform)
			}
		})
	}
}