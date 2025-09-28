// Package target provides advanced testing for target resolver functionality
package target

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestTargetResolver_StandardizationEdgeCases tests standardization with edge cases
func TestTargetResolver_StandardizationEdgeCases(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name         string
		targetType   string
		input        string
		expected     string
		description  string
	}{
		// Email standardization edge cases
		{
			name:        "gmail with multiple plus aliases",
			targetType:  TargetTypeEmail,
			input:       "user+tag1+tag2+tag3@gmail.com",
			expected:    "user@gmail.com",
			description: "should remove everything after first + in gmail",
		},
		{
			name:        "gmail with dots and plus",
			targetType:  TargetTypeEmail,
			input:       "u.s.e.r+tag@gmail.com",
			expected:    "user@gmail.com",
			description: "should remove dots and plus alias in gmail",
		},
		{
			name:        "gmail with uppercase and formatting",
			targetType:  TargetTypeEmail,
			input:       "USER.NAME+TAG@GMAIL.COM",
			expected:    "username@gmail.com",
			description: "should normalize case and remove formatting",
		},
		{
			name:        "non-gmail with plus",
			targetType:  TargetTypeEmail,
			input:       "user+tag@example.com",
			expected:    "user+tag@example.com",
			description: "should preserve plus alias for non-gmail addresses",
		},
		{
			name:        "email with mixed case domain",
			targetType:  TargetTypeEmail,
			input:       "User@EXAMPLE.COM",
			expected:    "user@example.com",
			description: "should normalize all email parts to lowercase",
		},
		{
			name:        "email with whitespace",
			targetType:  TargetTypeEmail,
			input:       "  user@example.com  ",
			expected:    "user@example.com",
			description: "should trim whitespace from email",
		},

		// Phone standardization edge cases
		{
			name:        "phone with multiple formatting styles",
			targetType:  TargetTypePhone,
			input:       "(555) 123-4567",
			expected:    "+15551234567",
			description: "should standardize US phone with parentheses and dashes",
		},
		{
			name:        "phone with dots as separators",
			targetType:  TargetTypePhone,
			input:       "555.123.4567",
			expected:    "+15551234567",
			description: "should standardize US phone with dots",
		},
		{
			name:        "phone with mixed separators",
			targetType:  TargetTypePhone,
			input:       "1 (555) 123-4567",
			expected:    "+15551234567",
			description: "should handle mixed separator styles",
		},
		{
			name:        "international phone without country code",
			targetType:  TargetTypePhone,
			input:       "13800138000",
			expected:    "+13800138000",
			description: "should add US country code for 11-digit numbers starting with 1",
		},
		{
			name:        "already formatted E.164",
			targetType:  TargetTypePhone,
			input:       "+8613800138000",
			expected:    "+8613800138000",
			description: "should preserve already correct E.164 format",
		},
		{
			name:        "phone with extension notation",
			targetType:  TargetTypePhone,
			input:       "555-123-4567 ext 123",
			expected:    "555-123-4567 ext 123",
			description: "should preserve original if cannot standardize",
		},
		{
			name:        "phone with non-numeric characters",
			targetType:  TargetTypePhone,
			input:       "1-800-CALL-NOW",
			expected:    "+1800CALLNOW",
			description: "should attempt standardization even with letters",
		},

		// URL standardization edge cases
		{
			name:        "URL with uppercase scheme",
			targetType:  TargetTypeWebhook,
			input:       "HTTPS://EXAMPLE.COM/webhook",
			expected:    "https://example.com/webhook",
			description: "should normalize scheme and host to lowercase",
		},
		{
			name:        "URL without scheme",
			targetType:  TargetTypeWebhook,
			input:       "api.example.com/webhook",
			expected:    "https://api.example.com/webhook",
			description: "should add default https scheme",
		},
		{
			name:        "URL with query parameters",
			targetType:  TargetTypeWebhook,
			input:       "https://example.com/webhook?token=abc&format=json",
			expected:    "https://example.com/webhook?token=abc&format=json",
			description: "should preserve query parameters",
		},
		{
			name:        "URL with fragment",
			targetType:  TargetTypeWebhook,
			input:       "https://example.com/webhook#section",
			expected:    "https://example.com/webhook#section",
			description: "should preserve URL fragment",
		},
		{
			name:        "URL with port",
			targetType:  TargetTypeWebhook,
			input:       "https://example.com:8080/webhook",
			expected:    "https://example.com:8080/webhook",
			description: "should preserve port number",
		},
		{
			name:        "malformed URL",
			targetType:  TargetTypeWebhook,
			input:       "ht tp://example.com",
			expected:    "ht tp://example.com",
			description: "should return original if URL parsing fails",
		},

		// Default standardization
		{
			name:        "user ID with surrounding whitespace",
			targetType:  TargetTypeUser,
			input:       "  ou_123456789  ",
			expected:    "ou_123456789",
			description: "should trim whitespace for non-specific types",
		},
		{
			name:        "group ID with tabs and newlines",
			targetType:  TargetTypeGroup,
			input:       "\t\nog_123456789\n\t",
			expected:    "og_123456789",
			description: "should trim all whitespace characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.StandardizeTarget(tt.targetType, tt.input)
			if result != tt.expected {
				t.Errorf("%s: expected %q, got %q", tt.description, tt.expected, result)
			}
		})
	}
}

// TestTargetResolver_ResolveBatchAdvanced tests advanced batch resolution scenarios
func TestTargetResolver_ResolveBatchAdvanced(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name           string
		input          []string
		expectedCount  int
		expectedErrors int
		checkDuplicates bool
		description    string
	}{
		{
			name: "complex deduplication scenario",
			input: []string{
				"user@example.com",
				"User@Example.Com",           // Should be deduplicated (case difference)
				"user+tag@example.com",       // Different value, should not be deduplicated
				"test.user@gmail.com",
				"testuser@gmail.com",         // Should be deduplicated (gmail normalization)
				"Test.User+alias@Gmail.Com",  // Should be deduplicated (gmail normalization)
				"+1234567890",
				"(123) 456-7890",             // Should be deduplicated (phone normalization)
				"123-456-7890",               // Should be deduplicated (phone normalization)
			},
			expectedCount:   4, // After deduplication
			expectedErrors:  0,
			checkDuplicates: true,
			description:     "should properly deduplicate normalized targets",
		},
		{
			name: "mixed valid and invalid targets",
			input: []string{
				"valid@example.com",
				"invalid-email",
				"+1234567890",
				"invalid-phone",
				"https://example.com/webhook",
				"invalid-url",
				"ou_validfeishuuser",
				"invalid_feishu",
				"",        // Empty
				"   ",     // Whitespace only
			},
			expectedCount:   4, // Only valid ones
			expectedErrors:  4, // Invalid ones generate errors
			checkDuplicates: false,
			description:     "should handle mix of valid and invalid targets",
		},
		{
			name: "large batch with many duplicates",
			input: func() []string {
				inputs := make([]string, 0, 1000)
				// Add same targets multiple times
				for i := 0; i < 100; i++ {
					inputs = append(inputs,
						"user@example.com",
						"User@Example.Com",
						"+1234567890",
						"(123) 456-7890",
						"https://example.com/webhook",
						"ou_feishuuser123",
					)
				}
				return inputs
			}(),
			expectedCount:   4, // Should deduplicate to unique targets
			expectedErrors:  0,
			checkDuplicates: true,
			description:     "should efficiently handle large batches with many duplicates",
		},
		{
			name: "international phone numbers",
			input: []string{
				"+1234567890",                // US
				"+441234567890",              // UK
				"+8613800138000",             // China
				"+33123456789",               // France
				"+49123456789",               // Germany
				"+81123456789",               // Japan
				"13800138000",                // China without +
			},
			expectedCount:   6, // 6 unique after normalization
			expectedErrors:  0,
			checkDuplicates: true,
			description:     "should handle international phone numbers correctly",
		},
		{
			name: "special character handling",
			input: []string{
				"user+tag@example.com",
				"user-name@example.com",
				"user_name@example.com",
				"user.name@example.com",
				"https://example.com/webhook?param=value&other=test",
				"https://example.com/webhook?other=test&param=value", // Different order
				"ou_user-id_123",
				"og_group-id_456",
			},
			expectedCount:   8, // All should be unique
			expectedErrors:  0,
			checkDuplicates: false,
			description:     "should handle special characters in target values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, errors := resolver.ResolveBatch(tt.input)

			if len(targets) != tt.expectedCount {
				t.Errorf("%s: expected %d targets, got %d", tt.description, tt.expectedCount, len(targets))
			}

			if len(errors) != tt.expectedErrors {
				t.Errorf("%s: expected %d errors, got %d", tt.description, tt.expectedErrors, len(errors))
			}

			if tt.checkDuplicates {
				// Verify no duplicates in result
				seen := make(map[string]bool)
				for _, target := range targets {
					key := target.String()
					if seen[key] {
						t.Errorf("%s: found duplicate target: %s", tt.description, key)
					}
					seen[key] = true
				}
			}

			// Verify all returned targets are valid
			for i, target := range targets {
				if err := target.Validate(); err != nil {
					t.Errorf("%s: target %d failed validation: %v", tt.description, i, err)
				}
			}
		})
	}
}

// TestTargetResolver_AutoDetectPerformance tests auto-detection performance
func TestTargetResolver_AutoDetectPerformance(t *testing.T) {
	resolver := NewTargetResolver()

	// Test with various input patterns
	testInputs := []string{
		"user@example.com",
		"+1234567890",
		"https://api.example.com/webhook",
		"ou_1234567890abcdef1234567890abcdef",
		"og_1234567890abcdef1234567890abcdef",
		"(555) 123-4567",
		"random-text-value",
	}

	// Performance test - should handle large volumes quickly
	start := time.Now()
	for i := 0; i < 10000; i++ {
		input := testInputs[i%len(testInputs)]
		target := resolver.AutoDetectTarget(input)
		if !target.IsValid() {
			t.Errorf("AutoDetectTarget should produce valid targets, got invalid for: %s", input)
		}
	}
	duration := time.Since(start)

	// Should complete in reasonable time (less than 1 second for 10k operations)
	if duration > time.Second {
		t.Errorf("AutoDetectTarget performance test took too long: %v", duration)
	}

	t.Logf("AutoDetectTarget processed 10,000 operations in %v", duration)
}

// TestTargetResolver_ConcurrentSafety tests resolver thread safety
func TestTargetResolver_ConcurrentSafety(t *testing.T) {
	resolver := NewTargetResolver()

	// Test concurrent auto-detection
	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 100

	// Collect results to verify correctness
	resultChan := make(chan Target, numGoroutines*operationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			testValue := fmt.Sprintf("user%d@example.com", id)
			for j := 0; j < operationsPerGoroutine; j++ {
				target := resolver.AutoDetectTarget(testValue)
				resultChan <- target
			}
		}(i)
	}

	wg.Wait()
	close(resultChan)

	// Verify all results are consistent
	resultCount := 0
	for target := range resultChan {
		resultCount++
		if target.Type != TargetTypeEmail {
			t.Errorf("concurrent operation produced incorrect type: %s", target.Type)
		}
		if target.Platform != PlatformEmail {
			t.Errorf("concurrent operation produced incorrect platform: %s", target.Platform)
		}
	}

	expectedResults := numGoroutines * operationsPerGoroutine
	if resultCount != expectedResults {
		t.Errorf("expected %d results, got %d", expectedResults, resultCount)
	}
}

// TestTargetResolver_RegexPatterns tests the regex patterns used for detection
func TestTargetResolver_RegexPatterns(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name     string
		pattern  *regexp.Regexp
		valid    []string
		invalid  []string
	}{
		{
			name:    "email regex",
			pattern: resolver.emailRegex,
			valid: []string{
				"user@example.com",
				"test.user+tag@sub.example.co.uk",
				"user123@domain123.org",
				"a@b.co",
			},
			invalid: []string{
				"userexample.com",
				"user@",
				"@example.com",
				"user@@example.com",
				"user@example",
				"",
			},
		},
		{
			name:    "phone regex",
			pattern: resolver.phoneRegex,
			valid: []string{
				"+1234567890",
				"+123456789012345",
				"+86123456789",
			},
			invalid: []string{
				"1234567890",
				"+0123456789",
				"+123456789012345678",
				"+(123) 456-7890",
				"",
			},
		},
		{
			name:    "URL regex",
			pattern: resolver.urlRegex,
			valid: []string{
				"https://example.com",
				"http://localhost:8080/path",
				"https://api.example.com/webhook?param=value",
			},
			invalid: []string{
				"ftp://example.com",
				"example.com",
				"https://",
				"ht tp://example.com",
				"",
			},
		},
		{
			name:    "feishu user regex",
			pattern: resolver.feishuUserRegex,
			valid: []string{
				"ou_123456",
				"oc_123456789012345678901234567890123456",
				"ou_abc123def456",
			},
			invalid: []string{
				"ou_123",
				"ou_",
				"user_123456",
				"ou_123456789012345678901234567890123456789",
				"",
			},
		},
		{
			name:    "feishu group regex",
			pattern: resolver.feishuGroupRegex,
			valid: []string{
				"og_123456",
				"og_123456789012345678901234567890123456",
				"og_abc123def456",
			},
			invalid: []string{
				"og_123",
				"og_",
				"group_123456",
				"ou_123456",
				"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test valid patterns
			for _, validInput := range tt.valid {
				if !tt.pattern.MatchString(validInput) {
					t.Errorf("pattern should match valid input %q", validInput)
				}
			}

			// Test invalid patterns
			for _, invalidInput := range tt.invalid {
				if tt.pattern.MatchString(invalidInput) {
					t.Errorf("pattern should not match invalid input %q", invalidInput)
				}
			}
		})
	}
}

// TestTargetResolver_PlatformCompatibilityAdvanced tests advanced platform compatibility scenarios
func TestTargetResolver_PlatformCompatibilityAdvanced(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name              string
		target            Target
		requestedPlatform string
		expectError       bool
		description       string
	}{
		{
			name: "auto target platform with specific request",
			target: Target{
				Type:     TargetTypeUser,
				Value:    "user123",
				Platform: PlatformAuto,
			},
			requestedPlatform: PlatformFeishu,
			expectError:       false,
			description:       "auto platform should be compatible with any request",
		},
		{
			name: "specific platform with auto request",
			target: Target{
				Type:     TargetTypeEmail,
				Value:    "user@example.com",
				Platform: PlatformEmail,
			},
			requestedPlatform: PlatformAuto,
			expectError:       false,
			description:       "any platform should be compatible with auto request",
		},
		{
			name: "incompatible platforms",
			target: Target{
				Type:     TargetTypePhone,
				Value:    "+1234567890",
				Platform: PlatformSMS,
			},
			requestedPlatform: PlatformEmail,
			expectError:       true,
			description:       "incompatible platforms should fail validation",
		},
		{
			name: "same platform compatibility",
			target: Target{
				Type:     TargetTypeWebhook,
				Value:    "https://example.com/webhook",
				Platform: PlatformWebhook,
			},
			requestedPlatform: PlatformWebhook,
			expectError:       false,
			description:       "same platforms should be compatible",
		},
		{
			name: "empty platforms",
			target: Target{
				Type:     TargetTypeUser,
				Value:    "user123",
				Platform: "",
			},
			requestedPlatform: "",
			expectError:       false,
			description:       "empty platforms should be compatible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolver.ValidatePlatformCompatibility(tt.target, tt.requestedPlatform)

			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}
		})
	}
}

// TestTargetResolver_ReachabilityHintsAdvanced tests advanced reachability hint scenarios
func TestTargetResolver_ReachabilityHintsAdvanced(t *testing.T) {
	resolver := NewTargetResolver()

	tests := []struct {
		name     string
		target   Target
		expected string
		description string
	}{
		{
			name: "corporate email domain",
			target: Target{
				Type:  TargetTypeEmail,
				Value: "user@company.internal",
			},
			expected:    "medium_reliability",
			description: "corporate domains should have medium reliability",
		},
		{
			name: "development webhook",
			target: Target{
				Type:  TargetTypeWebhook,
				Value: "http://dev.example.com/webhook",
			},
			expected:    "medium_reliability",
			description: "HTTP webhooks should have medium reliability",
		},
		{
			name: "production webhook",
			target: Target{
				Type:  TargetTypeWebhook,
				Value: "https://api.production.com/webhook",
			},
			expected:    "high_reliability",
			description: "HTTPS webhooks should have high reliability",
		},
		{
			name: "test environment webhook",
			target: Target{
				Type:  TargetTypeWebhook,
				Value: "https://127.0.0.1:8080/webhook",
			},
			expected:    "test_environment",
			description: "localhost IP should be detected as test environment",
		},
		{
			name: "international phone number",
			target: Target{
				Type:  TargetTypePhone,
				Value: "+44123456789",
			},
			expected:    "medium_reliability",
			description: "non-US/China phone should have medium reliability",
		},
		{
			name: "feishu group target",
			target: Target{
				Type:     TargetTypeGroup,
				Platform: PlatformFeishu,
			},
			expected:    "high_reliability",
			description: "feishu groups should have high reliability",
		},
		{
			name: "unknown target type",
			target: Target{
				Type: "unknown",
			},
			expected:    "unknown",
			description: "unknown types should return unknown reliability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.GetTargetReachabilityHint(tt.target)
			if result != tt.expected {
				t.Errorf("%s: expected %s, got %s", tt.description, tt.expected, result)
			}
		})
	}
}

// TestTargetResolver_ErrorHandling tests resolver error handling scenarios
func TestTargetResolver_ErrorHandling(t *testing.T) {
	resolver := NewTargetResolver()

	// Test batch resolution with problematic inputs
	problematicInputs := []string{
		strings.Repeat("a", 10000) + "@example.com", // Very long email
		"user@" + strings.Repeat("domain.", 100) + "com", // Very long domain
		"+1" + strings.Repeat("2", 20), // Very long phone
		"https://" + strings.Repeat("subdomain.", 50) + "example.com/webhook", // Very long URL
	}

	targets, errors := resolver.ResolveBatch(problematicInputs)

	// Should handle all inputs without panicking
	t.Logf("Processed %d problematic inputs, got %d targets and %d errors",
		len(problematicInputs), len(targets), len(errors))

	// Verify no nil targets
	for i, target := range targets {
		if target.Type == "" || target.Value == "" {
			t.Errorf("target %d has empty fields: %+v", i, target)
		}
	}
}

// BenchmarkTargetResolver_Operations benchmarks various resolver operations
func BenchmarkTargetResolver_Operations(b *testing.B) {
	resolver := NewTargetResolver()

	b.Run("AutoDetectTarget", func(b *testing.B) {
		inputs := []string{
			"user@example.com",
			"+1234567890",
			"https://api.example.com/webhook",
			"ou_1234567890abcdef",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			input := inputs[i%len(inputs)]
			_ = resolver.AutoDetectTarget(input)
		}
	})

	b.Run("StandardizeTarget", func(b *testing.B) {
		inputs := []struct {
			targetType string
			value      string
		}{
			{TargetTypeEmail, "User+Tag@Gmail.Com"},
			{TargetTypePhone, "(555) 123-4567"},
			{TargetTypeWebhook, "HTTP://API.EXAMPLE.COM/webhook"},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			input := inputs[i%len(inputs)]
			_ = resolver.StandardizeTarget(input.targetType, input.value)
		}
	})

	b.Run("ResolveBatch", func(b *testing.B) {
		batch := []string{
			"user1@example.com",
			"user2@example.com",
			"+1234567890",
			"+1234567891",
			"https://api.example.com/hook1",
			"https://api.example.com/hook2",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = resolver.ResolveBatch(batch)
		}
	})
}