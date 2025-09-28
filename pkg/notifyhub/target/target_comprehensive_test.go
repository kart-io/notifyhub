// Package target provides comprehensive testing for task 3.3 requirements
package target

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// TestTask3_3_TargetFactoryComprehensive tests all factory function scenarios as required by task 3.3
func TestTask3_3_TargetFactoryComprehensive(t *testing.T) {
	t.Run("AllFactoryFunctions", func(t *testing.T) {
		// Test all factory functions with valid inputs
		factories := []struct {
			name   string
			fn     func() Target
			expect Target
		}{
			{
				name: "NewEmailTarget",
				fn:   func() Target { return NewEmailTarget("user@example.com") },
				expect: Target{Type: TargetTypeEmail, Value: "user@example.com", Platform: PlatformEmail},
			},
			{
				name: "NewPhoneTarget",
				fn:   func() Target { return NewPhoneTarget("+1234567890") },
				expect: Target{Type: TargetTypePhone, Value: "+1234567890", Platform: PlatformSMS},
			},
			{
				name: "NewFeishuUserTarget",
				fn:   func() Target { return NewFeishuUserTarget("ou_123456") },
				expect: Target{Type: TargetTypeUser, Value: "ou_123456", Platform: PlatformFeishu},
			},
			{
				name: "NewFeishuGroupTarget",
				fn:   func() Target { return NewFeishuGroupTarget("og_123456") },
				expect: Target{Type: TargetTypeGroup, Value: "og_123456", Platform: PlatformFeishu},
			},
			{
				name: "NewWebhookTarget",
				fn:   func() Target { return NewWebhookTarget("https://example.com/webhook") },
				expect: Target{Type: TargetTypeWebhook, Value: "https://example.com/webhook", Platform: PlatformWebhook},
			},
		}

		for _, test := range factories {
			t.Run(test.name, func(t *testing.T) {
				result := test.fn()
				if result.Type != test.expect.Type {
					t.Errorf("Type: got %v, want %v", result.Type, test.expect.Type)
				}
				if result.Value != test.expect.Value {
					t.Errorf("Value: got %v, want %v", result.Value, test.expect.Value)
				}
				if result.Platform != test.expect.Platform {
					t.Errorf("Platform: got %v, want %v", result.Platform, test.expect.Platform)
				}
				if !result.IsValid() {
					t.Error("Factory should produce valid target")
				}
			})
		}
	})

	t.Run("EdgeCaseInputs", func(t *testing.T) {
		// Test edge case inputs for factory functions
		edgeCases := []struct {
			name     string
			fn       func() Target
			checkFn  func(Target) bool
		}{
			{
				name: "EmptyEmail",
				fn:   func() Target { return NewEmailTarget("") },
				checkFn: func(t Target) bool { return t.Type == TargetTypeEmail && !t.IsValid() },
			},
			{
				name: "WhitespacePhone",
				fn:   func() Target { return NewPhoneTarget("  ") },
				checkFn: func(t Target) bool { return t.Type == TargetTypePhone && t.Value == "  " },
			},
			{
				name: "LongFeishuID",
				fn:   func() Target { return NewFeishuUserTarget(strings.Repeat("a", 100)) },
				checkFn: func(t Target) bool { return t.Type == TargetTypeUser && len(t.Value) == 100 },
			},
		}

		for _, test := range edgeCases {
			t.Run(test.name, func(t *testing.T) {
				result := test.fn()
				if !test.checkFn(result) {
					t.Errorf("Edge case check failed for %s", test.name)
				}
			})
		}
	})
}

// TestTask3_3_TargetValidationComprehensive tests validation scenarios as required by task 3.3
func TestTask3_3_TargetValidationComprehensive(t *testing.T) {
	t.Run("ValidTargetValidation", func(t *testing.T) {
		validTargets := []Target{
			{Type: TargetTypeEmail, Value: "user@example.com"},
			{Type: TargetTypePhone, Value: "+1234567890"},
			{Type: TargetTypeWebhook, Value: "https://example.com/webhook"},
			{Type: TargetTypeUser, Value: "ou_123456"},
			{Type: TargetTypeGroup, Value: "og_123456"},
			{Type: TargetTypeChannel, Value: "channel_123"},
		}

		for i, target := range validTargets {
			t.Run(target.Type, func(t *testing.T) {
				if err := target.Validate(); err != nil {
					t.Errorf("Target %d should be valid but got error: %v", i, err)
				}
			})
		}
	})

	t.Run("InvalidTargetDetection", func(t *testing.T) {
		invalidTargets := []struct {
			target   Target
			errorKey string
		}{
			{Target{Type: TargetTypeEmail, Value: "invalid-email"}, "email"},
			{Target{Type: TargetTypePhone, Value: "invalid-phone"}, "phone"},
			{Target{Type: TargetTypeWebhook, Value: "invalid-url"}, "webhook"},
			{Target{Type: TargetTypeUser, Value: "a"}, "short"},
			{Target{Type: "", Value: "value"}, "unsupported"},
			{Target{Type: TargetTypeEmail, Value: ""}, "empty"},
		}

		for _, test := range invalidTargets {
			t.Run(test.target.Type+"_"+test.errorKey, func(t *testing.T) {
				err := test.target.Validate()
				if err == nil {
					t.Error("Expected validation error but got nil")
				} else if !strings.Contains(strings.ToLower(err.Error()), test.errorKey) {
					t.Errorf("Expected error containing '%s', got: %v", test.errorKey, err)
				}
			})
		}
	})

	t.Run("BoundaryConditions", func(t *testing.T) {
		// Test boundary conditions for validation
		boundaries := []struct {
			name    string
			target  Target
			isValid bool
		}{
			{"MinValidEmail", Target{Type: TargetTypeEmail, Value: "a@b.co"}, true},
			{"MinValidPhone", Target{Type: TargetTypePhone, Value: "+12"}, true},
			{"MinValidWebhook", Target{Type: TargetTypeWebhook, Value: "http://a.co"}, true},
			{"MinValidUserID", Target{Type: TargetTypeUser, Value: "ab"}, true},
			{"TooShortUserID", Target{Type: TargetTypeUser, Value: "a"}, false},
		}

		for _, test := range boundaries {
			t.Run(test.name, func(t *testing.T) {
				err := test.target.Validate()
				isValid := err == nil
				if isValid != test.isValid {
					t.Errorf("Expected valid=%v but got valid=%v (error: %v)", test.isValid, isValid, err)
				}
			})
		}
	})
}

// TestTask3_3_ResolverAutoDetection tests auto-detection functionality as required by task 3.3
func TestTask3_3_ResolverAutoDetection(t *testing.T) {
	resolver := NewTargetResolver()

	t.Run("AutoTypeDetection", func(t *testing.T) {
		detectionTests := []struct {
			input        string
			expectedType string
			expectedPlatform string
		}{
			{"user@example.com", TargetTypeEmail, PlatformEmail},
			{"+1234567890", TargetTypePhone, PlatformSMS},
			{"(555) 123-4567", TargetTypePhone, PlatformSMS},
			{"https://example.com/webhook", TargetTypeWebhook, PlatformWebhook},
			{"ou_1234567890abcdef", TargetTypeUser, PlatformFeishu},
			{"og_1234567890abcdef", TargetTypeGroup, PlatformFeishu},
			{"unknown-format", TargetTypeUser, PlatformAuto},
		}

		for _, test := range detectionTests {
			t.Run(test.input, func(t *testing.T) {
				target := resolver.AutoDetectTarget(test.input)
				if target.Type != test.expectedType {
					t.Errorf("Type: got %v, want %v", target.Type, test.expectedType)
				}
				if target.Platform != test.expectedPlatform {
					t.Errorf("Platform: got %v, want %v", target.Platform, test.expectedPlatform)
				}
			})
		}
	})

	t.Run("BatchResolution", func(t *testing.T) {
		inputs := []string{
			"user@example.com",
			"user@example.com", // Duplicate
			"+1234567890",
			"https://example.com/webhook",
			"", // Empty - should be skipped
		}

		targets, errors := resolver.ResolveBatch(inputs)

		// Should deduplicate and skip empty
		if len(targets) != 3 {
			t.Errorf("Expected 3 unique targets, got %d", len(targets))
		}

		if len(errors) != 0 {
			t.Errorf("Expected no errors for valid inputs, got %d: %v", len(errors), errors)
		}

		// Verify deduplication worked
		seen := make(map[string]bool)
		for _, target := range targets {
			key := target.String()
			if seen[key] {
				t.Errorf("Found duplicate target: %s", key)
			}
			seen[key] = true
		}
	})

	t.Run("Standardization", func(t *testing.T) {
		standardizationTests := []struct {
			targetType string
			input      string
			expectDiff bool // Whether standardization should change the value
		}{
			{TargetTypeEmail, "User@Example.Com", true},
			{TargetTypePhone, "(555) 123-4567", true},
			{TargetTypeWebhook, "HTTP://API.EXAMPLE.COM/webhook", true},
			{TargetTypeUser, "  user_id  ", true},
		}

		for _, test := range standardizationTests {
			t.Run(test.targetType+"_"+test.input, func(t *testing.T) {
				result := resolver.StandardizeTarget(test.targetType, test.input)
				if test.expectDiff && result == test.input {
					t.Errorf("Expected standardization to change '%s' but it didn't", test.input)
				}
				if !test.expectDiff && result != test.input {
					t.Errorf("Expected no change to '%s' but got '%s'", test.input, result)
				}
			})
		}
	})
}

// TestTask3_3_ErrorHandling tests error handling scenarios as required by task 3.3
func TestTask3_3_ErrorHandling(t *testing.T) {
	resolver := NewTargetResolver()

	t.Run("InvalidTargetErrors", func(t *testing.T) {
		invalidInputs := []string{
			"@invalid-email",
			"+invalid-phone",
			"not-a-url",
		}

		targets, errors := resolver.ResolveBatch(invalidInputs)

		// Should have fewer targets than inputs due to validation failures
		if len(targets) >= len(invalidInputs) {
			t.Errorf("Expected fewer valid targets than inputs, got %d targets from %d inputs", len(targets), len(invalidInputs))
		}

		// Should have some errors for invalid inputs
		if len(errors) == 0 {
			t.Error("Expected validation errors for invalid inputs")
		}

		// Each error should contain meaningful information
		for i, err := range errors {
			if err == nil {
				t.Errorf("Error %d is nil", i)
			} else if len(err.Error()) < 5 {
				t.Errorf("Error %d too short: %v", i, err)
			}
		}
	})

	t.Run("PlatformCompatibility", func(t *testing.T) {
		compatibilityTests := []struct {
			target            Target
			requestedPlatform string
			shouldError       bool
		}{
			{Target{Type: TargetTypeEmail, Platform: PlatformEmail}, PlatformEmail, false},
			{Target{Type: TargetTypeEmail, Platform: PlatformAuto}, PlatformEmail, false},
			{Target{Type: TargetTypeEmail, Platform: PlatformEmail}, PlatformAuto, false},
			{Target{Type: TargetTypeEmail, Platform: PlatformEmail}, PlatformFeishu, true},
		}

		for i, test := range compatibilityTests {
			t.Run("compatibility_"+string(rune(i+'A')), func(t *testing.T) {
				err := resolver.ValidatePlatformCompatibility(test.target, test.requestedPlatform)
				hasError := err != nil
				if hasError != test.shouldError {
					t.Errorf("Expected error=%v but got error=%v (err: %v)", test.shouldError, hasError, err)
				}
			})
		}
	})
}

// TestTask3_3_PerformanceValidation tests performance requirements as per task 3.3
func TestTask3_3_PerformanceValidation(t *testing.T) {
	resolver := NewTargetResolver()

	t.Run("AutoDetectPerformance", func(t *testing.T) {
		// Test that auto-detection is reasonably fast
		start := time.Now()
		iterations := 10000

		for i := 0; i < iterations; i++ {
			_ = resolver.AutoDetectTarget("user@example.com")
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		// Should be faster than 1ms per operation
		if avgTime > time.Millisecond {
			t.Errorf("AutoDetectTarget too slow: %v per operation", avgTime)
		}

		t.Logf("AutoDetectTarget performance: %d operations in %v (avg: %v)", iterations, duration, avgTime)
	})

	t.Run("ConcurrentSafety", func(t *testing.T) {
		// Test that resolver is thread-safe
		var wg sync.WaitGroup
		numGoroutines := 10
		operationsPerGoroutine := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					target := resolver.AutoDetectTarget("user@example.com")
					if target.Type != TargetTypeEmail {
						t.Errorf("Goroutine %d: incorrect detection result", id)
					}
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("BatchPerformance", func(t *testing.T) {
		// Test batch resolution performance
		batch := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			batch[i] = "user" + string(rune(i%26+'a')) + "@example.com"
		}

		start := time.Now()
		targets, errors := resolver.ResolveBatch(batch)
		duration := time.Since(start)

		// Should complete in reasonable time
		if duration > 100*time.Millisecond {
			t.Errorf("Batch resolution too slow: %v for 1000 items", duration)
		}

		// Should produce valid results
		if len(targets) == 0 {
			t.Error("Batch resolution produced no targets")
		}
		if len(errors) > 0 {
			t.Errorf("Batch resolution produced unexpected errors: %v", errors)
		}

		t.Logf("Batch performance: 1000 items resolved in %v", duration)
	})
}

// TestTask3_3_ComprehensiveCoverage ensures all code paths are tested as required by task 3.3
func TestTask3_3_ComprehensiveCoverage(t *testing.T) {
	t.Run("AllConstants", func(t *testing.T) {
		// Test that all constants are accessible and correct
		constants := map[string]string{
			"TargetTypeEmail":   TargetTypeEmail,
			"TargetTypePhone":   TargetTypePhone,
			"TargetTypeUser":    TargetTypeUser,
			"TargetTypeGroup":   TargetTypeGroup,
			"TargetTypeChannel": TargetTypeChannel,
			"TargetTypeWebhook": TargetTypeWebhook,
			"PlatformFeishu":    PlatformFeishu,
			"PlatformEmail":     PlatformEmail,
			"PlatformSMS":       PlatformSMS,
			"PlatformWebhook":   PlatformWebhook,
			"PlatformAuto":      PlatformAuto,
		}

		for name, value := range constants {
			if value == "" {
				t.Errorf("Constant %s is empty", name)
			}
			if len(value) < 2 {
				t.Errorf("Constant %s too short: %s", name, value)
			}
		}
	})

	t.Run("AllHelperMethods", func(t *testing.T) {
		// Test all helper methods
		target := Target{Type: TargetTypeEmail, Value: "user@example.com", Platform: PlatformEmail}

		helpers := map[string]bool{
			"IsEmail":   target.IsEmail(),
			"IsPhone":   target.IsPhone(),
			"IsWebhook": target.IsWebhook(),
			"IsFeishu":  target.IsFeishu(),
			"IsValid":   target.IsValid(),
		}

		// Should have exactly one true result for type checks
		trueCount := 0
		for method, result := range helpers {
			if method == "IsValid" {
				continue // Skip IsValid in this count
			}
			if result {
				trueCount++
			}
		}

		if trueCount != 1 {
			t.Errorf("Expected exactly one type helper to be true, got %d: %v", trueCount, helpers)
		}

		// IsValid should be true for this target
		if !helpers["IsValid"] {
			t.Error("IsValid should be true for valid target")
		}

		// String method should work
		str := target.String()
		if len(str) == 0 {
			t.Error("String() method returned empty string")
		}
		if !strings.Contains(str, target.Type) {
			t.Error("String() method should contain target type")
		}
	})

	t.Run("DefaultResolverFunctions", func(t *testing.T) {
		// Test that default resolver functions work
		target := AutoDetectTarget("user@example.com")
		if target.Type != TargetTypeEmail {
			t.Errorf("AutoDetectTarget default function failed: got %s", target.Type)
		}

		targets, errors := ResolveBatch([]string{"user@example.com", "+1234567890"})
		if len(targets) != 2 {
			t.Errorf("ResolveBatch default function failed: got %d targets", len(targets))
		}
		if len(errors) != 0 {
			t.Errorf("ResolveBatch default function had errors: %v", errors)
		}
	})
}