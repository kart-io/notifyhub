// Package target provides target resolution functionality for NotifyHub
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
		expectedValue string // after standardization
	}{
		{
			name:         "detect email",
			value:        "user@example.com",
			wantType:     TargetTypeEmail,
			wantPlatform: PlatformEmail,
			expectedValue: "user@example.com",
		},
		{
			name:         "detect complex email",
			value:        "test.user+tag@sub.example.co.uk",
			wantType:     TargetTypeEmail,
			wantPlatform: PlatformEmail,
			expectedValue: "test.user+tag@sub.example.co.uk",
		},
		{
			name:         "gmail standardization",
			value:        "Test.User+Tag@Gmail.Com",
			wantType:     TargetTypeEmail,
			wantPlatform: PlatformEmail,
			expectedValue: "testuser@gmail.com", // Should be standardized
		},
		{
			name:         "detect phone number",
			value:        "+1234567890",
			wantType:     TargetTypePhone,
			wantPlatform: PlatformSMS,
			expectedValue: "+1234567890",
		},
		{
			name:         "detect formatted phone",
			value:        "(555) 123-4567",
			wantType:     TargetTypePhone,
			wantPlatform: PlatformSMS,
			expectedValue: "+15551234567", // Should be standardized
		},
		{
			name:         "detect international phone",
			value:        "+861234567890",
			wantType:     TargetTypePhone,
			wantPlatform: PlatformSMS,
			expectedValue: "+861234567890",
		},
		{
			name:         "detect HTTPS webhook",
			value:        "https://example.com/webhook",
			wantType:     TargetTypeWebhook,
			wantPlatform: PlatformWebhook,
			expectedValue: "https://example.com/webhook",
		},
		{
			name:         "detect HTTP webhook",
			value:        "http://localhost:8080/webhook",
			wantType:     TargetTypeWebhook,
			wantPlatform: PlatformWebhook,
			expectedValue: "http://localhost:8080/webhook",
		},
		{
			name:         "detect feishu user ID (ou_)",
			value:        "ou_1234567890abcdef1234567890abcdef",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformFeishu,
			expectedValue: "ou_1234567890abcdef1234567890abcdef",
		},
		{
			name:         "detect feishu user ID (oc_)",
			value:        "oc_1234567890abcdef1234567890abcdef",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformFeishu,
			expectedValue: "oc_1234567890abcdef1234567890abcdef",
		},
		{
			name:         "detect feishu group ID",
			value:        "og_1234567890abcdef1234567890abcdef",
			wantType:     TargetTypeGroup,
			wantPlatform: PlatformFeishu,
			expectedValue: "og_1234567890abcdef1234567890abcdef",
		},
		{
			name:         "unknown format defaults to user/auto",
			value:        "some-unknown-value",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformAuto,
			expectedValue: "some-unknown-value",
		},
		{
			name:         "username-like defaults to user/auto",
			value:        "username123",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformAuto,
			expectedValue: "username123",
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
			if target.Value != tt.expectedValue {
				t.Errorf("AutoDetectTarget(%v) Value = %v, want %v", tt.value, target.Value, tt.expectedValue)
			}

			// Verify the target is valid (has required fields)
			if !target.IsValid() {
				t.Errorf("AutoDetectTarget(%v) should produce valid target", tt.value)
			}
		})
	}
}

func TestTargetResolver_NewResolver(t *testing.T) {
	resolver := NewTargetResolver()
	if resolver == nil {
		t.Error("NewTargetResolver() should return a non-nil resolver")
	}

	// Test that regex patterns are compiled
	if resolver.emailRegex == nil {
		t.Error("emailRegex should be compiled")
	}
	if resolver.phoneRegex == nil {
		t.Error("phoneRegex should be compiled")
	}
	if resolver.urlRegex == nil {
		t.Error("urlRegex should be compiled")
	}
}

func TestTargetResolver_StandardizeTarget(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name         string
		targetType   string
		input        string
		expected     string
	}{
		// Email standardization
		{
			name:       "email lowercase",
			targetType: TargetTypeEmail,
			input:      "User@Example.Com",
			expected:   "user@example.com",
		},
		{
			name:       "gmail normalization",
			targetType: TargetTypeEmail,
			input:      "test.user+tag@gmail.com",
			expected:   "testuser@gmail.com",
		},
		{
			name:       "gmail with dots",
			targetType: TargetTypeEmail,
			input:      "u.s.e.r@gmail.com",
			expected:   "user@gmail.com",
		},

		// Phone standardization
		{
			name:       "US phone with formatting",
			targetType: TargetTypePhone,
			input:      "(555) 123-4567",
			expected:   "+15551234567",
		},
		{
			name:       "phone with spaces and dashes",
			targetType: TargetTypePhone,
			input:      "555 123-4567",
			expected:   "+15551234567",
		},
		{
			name:       "already E.164",
			targetType: TargetTypePhone,
			input:      "+1234567890",
			expected:   "+1234567890",
		},
		{
			name:       "10 digit US number",
			targetType: TargetTypePhone,
			input:      "1234567890",
			expected:   "+11234567890",
		},
		{
			name:       "11 digit US with country code",
			targetType: TargetTypePhone,
			input:      "11234567890",
			expected:   "+11234567890",
		},

		// URL standardization
		{
			name:       "URL scheme normalization",
			targetType: TargetTypeWebhook,
			input:      "HTTP://API.EXAMPLE.COM/webhook",
			expected:   "http://api.example.com/webhook",
		},
		{
			name:       "add default scheme",
			targetType: TargetTypeWebhook,
			input:      "api.example.com/webhook",
			expected:   "https://api.example.com/webhook",
		},

		// Default standardization
		{
			name:       "trim whitespace",
			targetType: TargetTypeUser,
			input:      "  user_id_123  ",
			expected:   "user_id_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.StandardizeTarget(tt.targetType, tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTargetResolver_ResolveBatch(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name          string
		input         []string
		expectedCount int
		expectErrors  bool
	}{
		{
			name: "mixed valid targets",
			input: []string{
				"user@example.com",
				"+1234567890",
				"https://api.example.com/hook",
				"ou_1234567890abcdef1234567890abcdef",
			},
			expectedCount: 4,
			expectErrors:  false,
		},
		{
			name: "with duplicates",
			input: []string{
				"user@example.com",
				"User@Example.Com", // Should be deduplicated
				"+1234567890",
				"(123) 456-7890", // Different format but same number after standardization
			},
			expectedCount: 3, // Should be deduplicated to 3 unique targets
			expectErrors:  false,
		},
		{
			name: "with empty and invalid values",
			input: []string{
				"user@example.com",
				"", // Empty should be skipped
				"@invalid", // Invalid email that should fail validation
				"+1234567890",
				"   ", // Whitespace should be skipped
			},
			expectedCount: 2, // Only valid email and phone
			expectErrors:  true, // Should have errors for invalid values
		},
		{
			name:          "empty input",
			input:         []string{},
			expectedCount: 0,
			expectErrors:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, errors := resolver.ResolveBatch(tt.input)

			if len(targets) != tt.expectedCount {
				t.Errorf("expected %d targets, got %d", tt.expectedCount, len(targets))
			}

			if tt.expectErrors && len(errors) == 0 {
				t.Error("expected errors but got none")
			}
			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("expected no errors but got %d: %v", len(errors), errors)
			}

			// Verify no duplicates in result
			seen := make(map[string]bool)
			for _, target := range targets {
				key := target.String()
				if seen[key] {
					t.Errorf("found duplicate target: %s", key)
				}
				seen[key] = true
			}
		})
	}
}

func TestTargetResolver_ValidatePlatformCompatibility(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name           string
		target         Target
		requestedPlatform string
		expectError    bool
	}{
		{
			name: "compatible platforms",
			target: Target{
				Type:     TargetTypeEmail,
				Value:    "user@example.com",
				Platform: PlatformEmail,
			},
			requestedPlatform: PlatformEmail,
			expectError:       false,
		},
		{
			name: "auto platform target",
			target: Target{
				Type:     TargetTypeUser,
				Value:    "user123",
				Platform: PlatformAuto,
			},
			requestedPlatform: PlatformFeishu,
			expectError:       false,
		},
		{
			name: "auto requested platform",
			target: Target{
				Type:     TargetTypeEmail,
				Value:    "user@example.com",
				Platform: PlatformEmail,
			},
			requestedPlatform: PlatformAuto,
			expectError:       false,
		},
		{
			name: "incompatible platforms",
			target: Target{
				Type:     TargetTypeEmail,
				Value:    "user@example.com",
				Platform: PlatformEmail,
			},
			requestedPlatform: PlatformFeishu,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolver.ValidatePlatformCompatibility(tt.target, tt.requestedPlatform)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestTargetResolver_GetTargetReachabilityHint(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name     string
		target   Target
		expected string
	}{
		{
			name: "gmail email",
			target: Target{
				Type:  TargetTypeEmail,
				Value: "user@gmail.com",
			},
			expected: "high_reliability",
		},
		{
			name: "outlook email",
			target: Target{
				Type:  TargetTypeEmail,
				Value: "user@outlook.com",
			},
			expected: "high_reliability",
		},
		{
			name: "custom domain email",
			target: Target{
				Type:  TargetTypeEmail,
				Value: "user@company.com",
			},
			expected: "medium_reliability",
		},
		{
			name: "US phone number",
			target: Target{
				Type:  TargetTypePhone,
				Value: "+1234567890",
			},
			expected: "high_reliability",
		},
		{
			name: "China phone number",
			target: Target{
				Type:  TargetTypePhone,
				Value: "+861380013800",
			},
			expected: "high_reliability",
		},
		{
			name: "other country phone",
			target: Target{
				Type:  TargetTypePhone,
				Value: "+441234567890",
			},
			expected: "medium_reliability",
		},
		{
			name: "HTTPS webhook",
			target: Target{
				Type:  TargetTypeWebhook,
				Value: "https://api.example.com/hook",
			},
			expected: "high_reliability",
		},
		{
			name: "localhost webhook",
			target: Target{
				Type:  TargetTypeWebhook,
				Value: "http://localhost:8080/hook",
			},
			expected: "test_environment",
		},
		{
			name: "127.0.0.1 webhook",
			target: Target{
				Type:  TargetTypeWebhook,
				Value: "http://127.0.0.1:8080/hook",
			},
			expected: "test_environment",
		},
		{
			name: "Feishu user",
			target: Target{
				Type:     TargetTypeUser,
				Platform: PlatformFeishu,
				Value:    "ou_123456",
			},
			expected: "high_reliability",
		},
		{
			name: "unknown target type",
			target: Target{
				Type:  "unknown",
				Value: "test",
			},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.GetTargetReachabilityHint(tt.target)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTargetResolver_isNationalPhoneNumber(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "US format with parentheses",
			input:    "(555) 123-4567",
			expected: true,
		},
		{
			name:     "US format with dashes",
			input:    "555-123-4567",
			expected: true,
		},
		{
			name:     "US format no formatting",
			input:    "5551234567",
			expected: true,
		},
		{
			name:     "China mobile format",
			input:    "138-0013-8000",
			expected: true,
		},
		{
			name:     "China mobile no formatting",
			input:    "13800138000",
			expected: true,
		},
		{
			name:     "too short",
			input:    "12345",
			expected: false,
		},
		{
			name:     "too long",
			input:    "1234567890123456",
			expected: false,
		},
		{
			name:     "contains letters",
			input:    "555-CALL-NOW",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.isNationalPhoneNumber(tt.input)
			if result != tt.expected {
				t.Errorf("expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestDetectTargetTypeAndPlatform(t *testing.T) {
	resolver := NewTargetResolver()
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
			name:         "national phone detection",
			value:        "(555) 123-4567",
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
			value:        "ou_1234567890abcdef1234567890abcdef",
			wantType:     TargetTypeUser,
			wantPlatform: PlatformFeishu,
		},
		{
			name:         "feishu group detection",
			value:        "og_1234567890abcdef1234567890abcdef",
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
			gotType, gotPlatform := resolver.detectTargetTypeAndPlatform(tt.value)

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

func TestDefaultResolver(t *testing.T) {
	// Test that the default resolver functions work
	target := AutoDetectTarget("user@example.com")
	if target.Type != TargetTypeEmail {
		t.Errorf("expected email type, got %s", target.Type)
	}

	targets, errors := ResolveBatch([]string{"user@example.com", "+1234567890"})
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d", len(errors))
	}
}

// Benchmark tests for performance validation
func BenchmarkTargetResolver_AutoDetectTarget(b *testing.B) {
	resolver := NewTargetResolver()
	testValues := []string{
		"user@example.com",
		"+1234567890",
		"https://api.example.com/hook",
		"ou_1234567890abcdef1234567890abcdef",
		"(555) 123-4567",
		"user.name+tag@gmail.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := testValues[i%len(testValues)]
		_ = resolver.AutoDetectTarget(value)
	}
}

func BenchmarkTargetResolver_ResolveBatch(b *testing.B) {
	resolver := NewTargetResolver()
	testBatch := []string{
		"user1@example.com",
		"user2@example.com",
		"+1234567890",
		"+1234567891",
		"https://api.example.com/hook1",
		"https://api.example.com/hook2",
		"ou_1234567890abcdef1234567890abcdef",
		"og_1234567890abcdef1234567890abcdef",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.ResolveBatch(testBatch)
	}
}

func TestTargetResolver_ConcurrentAccess(t *testing.T) {
	resolver := NewTargetResolver()

	// Test concurrent access to ensure thread safety
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				target := resolver.AutoDetectTarget("user@example.com")
				if target.Type != TargetTypeEmail {
					t.Errorf("goroutine %d: expected email type, got %s", id, target.Type)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
