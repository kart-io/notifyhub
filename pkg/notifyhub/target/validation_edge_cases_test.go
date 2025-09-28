// Package target provides comprehensive validation testing for edge cases and boundary conditions
package target

import (
	"fmt"
	"strings"
	"testing"
)

// TestTargetValidationBoundaryConditions tests validation with boundary conditions
func TestTargetValidationBoundaryConditions(t *testing.T) {
	tests := []struct {
		name      string
		target    Target
		wantError bool
		errorType string
		description string
	}{
		// Email boundary conditions
		{
			name:        "email with minimum valid length",
			target:      Target{Type: "email", Value: "a@b.co"},
			wantError:   false,
			description: "should accept minimal valid email",
		},
		{
			name:        "email with very long local part",
			target:      Target{Type: "email", Value: strings.Repeat("a", 64) + "@example.com"},
			wantError:   false,
			description: "should accept long local part within limits",
		},
		{
			name:        "email with very long domain",
			target:      Target{Type: "email", Value: "user@" + strings.Repeat("sub.", 20) + "example.com"},
			wantError:   false,
			description: "should accept long domain within limits",
		},
		{
			name:        "email with consecutive dots in domain",
			target:      Target{Type: "email", Value: "user@example..com"},
			wantError:   true,
			errorType:   "email",
			description: "should reject email with consecutive dots in domain",
		},
		{
			name:        "email starting with dot",
			target:      Target{Type: "email", Value: ".user@example.com"},
			wantError:   true,
			errorType:   "email",
			description: "should reject email starting with dot",
		},
		{
			name:        "email ending with dot before @",
			target:      Target{Type: "email", Value: "user.@example.com"},
			wantError:   true,
			errorType:   "email",
			description: "should reject email ending with dot before @",
		},
		{
			name:        "email with unicode characters",
			target:      Target{Type: "email", Value: "用户@example.com"},
			wantError:   true,
			errorType:   "email",
			description: "should reject email with unicode characters (current implementation)",
		},

		// Phone boundary conditions
		{
			name:        "phone with minimum valid length",
			target:      Target{Type: "phone", Value: "+12"},
			wantError:   false,
			description: "should accept minimal valid E.164 phone",
		},
		{
			name:        "phone with maximum valid length",
			target:      Target{Type: "phone", Value: "+123456789012345"},
			wantError:   false,
			description: "should accept maximum valid E.164 phone",
		},
		{
			name:        "phone exceeding maximum length",
			target:      Target{Type: "phone", Value: "+1234567890123456"},
			wantError:   true,
			errorType:   "phone",
			description: "should reject phone exceeding maximum E.164 length",
		},
		{
			name:        "phone with country code 0",
			target:      Target{Type: "phone", Value: "+0123456789"},
			wantError:   true,
			errorType:   "phone",
			description: "should reject phone with invalid country code 0",
		},
		{
			name:        "phone with spaces",
			target:      Target{Type: "phone", Value: "+1 234 567 890"},
			wantError:   true,
			errorType:   "phone",
			description: "should reject phone with spaces in E.164 format",
		},
		{
			name:        "phone with extension",
			target:      Target{Type: "phone", Value: "+1234567890ext123"},
			wantError:   true,
			errorType:   "phone",
			description: "should reject phone with extension in E.164 format",
		},

		// Webhook boundary conditions
		{
			name:        "webhook with minimum valid URL",
			target:      Target{Type: "webhook", Value: "http://a.co"},
			wantError:   false,
			description: "should accept minimal valid webhook URL",
		},
		{
			name:        "webhook with very long URL",
			target:      Target{Type: "webhook", Value: "https://example.com/" + strings.Repeat("path/", 100)},
			wantError:   false,
			description: "should accept very long webhook URL",
		},
		{
			name:        "webhook with port number",
			target:      Target{Type: "webhook", Value: "https://example.com:8080/webhook"},
			wantError:   false,
			description: "should accept webhook URL with port",
		},
		{
			name:        "webhook with IP address",
			target:      Target{Type: "webhook", Value: "https://192.168.1.1/webhook"},
			wantError:   false,
			description: "should accept webhook URL with IP address",
		},
		{
			name:        "webhook with localhost",
			target:      Target{Type: "webhook", Value: "http://localhost:3000/webhook"},
			wantError:   false,
			description: "should accept localhost webhook URL",
		},
		{
			name:        "webhook with file protocol",
			target:      Target{Type: "webhook", Value: "file:///path/to/file"},
			wantError:   true,
			errorType:   "webhook",
			description: "should reject webhook URL with file protocol",
		},
		{
			name:        "webhook without domain",
			target:      Target{Type: "webhook", Value: "https://"},
			wantError:   true,
			errorType:   "webhook",
			description: "should reject webhook URL without domain",
		},

		// ID boundary conditions
		{
			name:        "user ID with minimum length",
			target:      Target{Type: "user", Value: "ab"},
			wantError:   false,
			description: "should accept minimal user ID",
		},
		{
			name:        "user ID with single character",
			target:      Target{Type: "user", Value: "a"},
			wantError:   true,
			errorType:   "ID",
			description: "should reject single character user ID",
		},
		{
			name:        "user ID with very long value",
			target:      Target{Type: "user", Value: strings.Repeat("a", 500)},
			wantError:   false,
			description: "should accept very long user ID",
		},
		{
			name:        "group ID with special prefix characters",
			target:      Target{Type: "group", Value: "@group123"},
			wantError:   true,
			errorType:   "ID",
			description: "should reject group ID starting with @",
		},
		{
			name:        "channel ID with phone prefix",
			target:      Target{Type: "channel", Value: "+channel123"},
			wantError:   true,
			errorType:   "ID",
			description: "should reject channel ID starting with +",
		},

		// Edge case combinations
		{
			name:        "empty value with valid type",
			target:      Target{Type: "email", Value: ""},
			wantError:   true,
			errorType:   "empty",
			description: "should reject empty value regardless of type",
		},
		{
			name:        "whitespace only value",
			target:      Target{Type: "user", Value: "   "},
			wantError:   false,
			description: "should accept whitespace-only value (current implementation)",
		},
		{
			name:        "value with null character",
			target:      Target{Type: "user", Value: "user\x00id"},
			wantError:   false,
			description: "should handle null character in value",
		},
		{
			name:        "value with newline characters",
			target:      Target{Type: "user", Value: "user\nid"},
			wantError:   false,
			description: "should handle newline characters in value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.target.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("%s: expected error but got nil", tt.description)
					return
				}

				// Check error type if specified
				if tt.errorType != "" {
					errorStr := strings.ToLower(err.Error())
					expectedType := strings.ToLower(tt.errorType)
					if !strings.Contains(errorStr, expectedType) {
						t.Errorf("%s: expected error containing '%s', got '%s'", tt.description, expectedType, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
				}
			}
		})
	}
}

// TestTargetValidationWithMalformedInputs tests validation with specifically malformed inputs
func TestTargetValidationWithMalformedInputs(t *testing.T) {
	tests := []struct {
		name    string
		target  Target
		description string
	}{
		{
			name:    "email with multiple @ symbols",
			target:  Target{Type: "email", Value: "user@@domain@com"},
			description: "should reject email with multiple @ symbols",
		},
		{
			name:    "email without domain extension",
			target:  Target{Type: "email", Value: "user@domain"},
			description: "should reject email without domain extension",
		},
		{
			name:    "email with domain starting with dash",
			target:  Target{Type: "email", Value: "user@-example.com"},
			description: "should reject email with domain starting with dash",
		},
		{
			name:    "phone with alphabetic characters",
			target:  Target{Type: "phone", Value: "+1ABCDEFGHIJ"},
			description: "should reject phone with alphabetic characters",
		},
		{
			name:    "phone with special characters",
			target:  Target{Type: "phone", Value: "+1(234)567-890"},
			description: "should reject phone with formatting in E.164",
		},
		{
			name:    "webhook with malformed URL",
			target:  Target{Type: "webhook", Value: "ht tp://example.com"},
			description: "should reject webhook with space in protocol",
		},
		{
			name:    "webhook with missing colon",
			target:  Target{Type: "webhook", Value: "https//example.com"},
			description: "should reject webhook with missing colon after protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.target.Validate()
			if err == nil {
				t.Errorf("%s: expected validation to fail but it passed", tt.description)
			} else {
				t.Logf("%s: validation correctly failed with: %v", tt.description, err)
			}
		})
	}
}

// TestTargetValidationPerformance tests validation performance with various input sizes
func TestTargetValidationPerformance(t *testing.T) {
	// Test validation performance with increasingly large inputs
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("email_validation_size_%d", size), func(t *testing.T) {
			longEmail := strings.Repeat("a", size) + "@example.com"
			target := Target{Type: "email", Value: longEmail}

			// Should not panic or take excessive time
			err := target.Validate()
			if err != nil {
				t.Logf("Email validation with size %d failed as expected: %v", size, err)
			} else {
				t.Logf("Email validation with size %d passed", size)
			}
		})

		t.Run(fmt.Sprintf("user_id_validation_size_%d", size), func(t *testing.T) {
			longID := strings.Repeat("a", size)
			target := Target{Type: "user", Value: longID}

			// Should not panic or take excessive time
			err := target.Validate()
			if err != nil {
				t.Logf("User ID validation with size %d failed: %v", size, err)
			} else {
				t.Logf("User ID validation with size %d passed", size)
			}
		})
	}
}

// TestTargetStringRepresentationEdgeCases tests String() method with edge cases
func TestTargetStringRepresentationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		expected string
		description string
	}{
		{
			name:     "target with empty fields",
			target:   Target{Type: "", Value: "", Platform: ""},
			expected: ":@",
			description: "should handle empty fields gracefully",
		},
		{
			name:     "target with special characters",
			target:   Target{Type: "user@#$", Value: "value!@#", Platform: "platform%^&"},
			expected: "user@#$:value!@#@platform%^&",
			description: "should preserve special characters in string representation",
		},
		{
			name:     "target with unicode characters",
			target:   Target{Type: "用户", Value: "测试@例子.com", Platform: "平台"},
			expected: "用户:测试@例子.com@平台",
			description: "should handle unicode characters correctly",
		},
		{
			name:     "target with whitespace",
			target:   Target{Type: " email ", Value: " user@example.com ", Platform: " email "},
			expected: " email : user@example.com @ email ",
			description: "should preserve whitespace in string representation",
		},
		{
			name:     "target with newlines",
			target:   Target{Type: "user\n", Value: "value\nwith\nnewlines", Platform: "platform\n"},
			expected: "user\n:value\nwith\nnewlines@platform\n",
			description: "should handle newline characters in string representation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.target.String()
			if result != tt.expected {
				t.Errorf("%s: String() = %q, want %q", tt.description, result, tt.expected)
			}
		})
	}
}

// TestTargetHelperMethodsEdgeCases tests helper methods with edge cases
func TestTargetHelperMethodsEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		checks map[string]bool // method name -> expected result
		description string
	}{
		{
			name:   "email target with wrong platform",
			target: Target{Type: TargetTypeEmail, Value: "user@example.com", Platform: PlatformFeishu},
			checks: map[string]bool{
				"IsEmail":   true,  // Based on type, not platform
				"IsPhone":   false,
				"IsWebhook": false,
				"IsFeishu":  true,  // Based on platform
			},
			description: "should check type and platform independently",
		},
		{
			name:   "phone target with email platform",
			target: Target{Type: TargetTypePhone, Value: "+1234567890", Platform: PlatformEmail},
			checks: map[string]bool{
				"IsEmail":   false,
				"IsPhone":   true,  // Based on type
				"IsWebhook": false,
				"IsFeishu":  false, // Based on platform
			},
			description: "should handle type/platform mismatch",
		},
		{
			name:   "target with empty type",
			target: Target{Type: "", Value: "some-value", Platform: PlatformFeishu},
			checks: map[string]bool{
				"IsEmail":   false,
				"IsPhone":   false,
				"IsWebhook": false,
				"IsFeishu":  true,  // Platform check still works
			},
			description: "should handle empty type gracefully",
		},
		{
			name:   "target with empty platform",
			target: Target{Type: TargetTypeUser, Value: "user123", Platform: ""},
			checks: map[string]bool{
				"IsEmail":   false,
				"IsPhone":   false,
				"IsWebhook": false,
				"IsFeishu":  false, // Empty platform != PlatformFeishu
			},
			description: "should handle empty platform gracefully",
		},
		{
			name:   "target with unknown type and platform",
			target: Target{Type: "unknown", Value: "value", Platform: "unknown"},
			checks: map[string]bool{
				"IsEmail":   false,
				"IsPhone":   false,
				"IsWebhook": false,
				"IsFeishu":  false,
			},
			description: "should handle unknown type and platform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if expected, ok := tt.checks["IsEmail"]; ok {
				if result := tt.target.IsEmail(); result != expected {
					t.Errorf("%s: IsEmail() = %v, want %v", tt.description, result, expected)
				}
			}
			if expected, ok := tt.checks["IsPhone"]; ok {
				if result := tt.target.IsPhone(); result != expected {
					t.Errorf("%s: IsPhone() = %v, want %v", tt.description, result, expected)
				}
			}
			if expected, ok := tt.checks["IsWebhook"]; ok {
				if result := tt.target.IsWebhook(); result != expected {
					t.Errorf("%s: IsWebhook() = %v, want %v", tt.description, result, expected)
				}
			}
			if expected, ok := tt.checks["IsFeishu"]; ok {
				if result := tt.target.IsFeishu(); result != expected {
					t.Errorf("%s: IsFeishu() = %v, want %v", tt.description, result, expected)
				}
			}
		})
	}
}

// TestTargetIsValidEdgeCases tests IsValid method with various edge cases
func TestTargetIsValidEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		expected bool
		description string
	}{
		{
			name:     "valid target all fields present",
			target:   Target{Type: "email", Value: "user@example.com", Platform: "email"},
			expected: true,
			description: "should be valid with all fields present",
		},
		{
			name:     "valid target without platform",
			target:   Target{Type: "email", Value: "user@example.com", Platform: ""},
			expected: true,
			description: "should be valid without platform (platform optional)",
		},
		{
			name:     "invalid target empty type",
			target:   Target{Type: "", Value: "user@example.com", Platform: "email"},
			expected: false,
			description: "should be invalid with empty type",
		},
		{
			name:     "invalid target empty value",
			target:   Target{Type: "email", Value: "", Platform: "email"},
			expected: false,
			description: "should be invalid with empty value",
		},
		{
			name:     "invalid target all empty",
			target:   Target{Type: "", Value: "", Platform: ""},
			expected: false,
			description: "should be invalid with all fields empty",
		},
		{
			name:     "target with whitespace type",
			target:   Target{Type: "   ", Value: "user@example.com", Platform: "email"},
			expected: true,
			description: "should consider whitespace-only type as valid (current implementation)",
		},
		{
			name:     "target with whitespace value",
			target:   Target{Type: "email", Value: "   ", Platform: "email"},
			expected: true,
			description: "should consider whitespace-only value as valid (current implementation)",
		},
		{
			name:     "target with special characters",
			target:   Target{Type: "user!@#", Value: "value$%^", Platform: "platform&*()"},
			expected: true,
			description: "should be valid with special characters (IsValid doesn't validate format)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.target.IsValid()
			if result != tt.expected {
				t.Errorf("%s: IsValid() = %v, want %v", tt.description, result, tt.expected)
			}
		})
	}
}