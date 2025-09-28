package feishu

import (
	"strconv"
	"testing"
	"time"
)

// Test the enhanced auth functionality to validate Task 0.3 completion
func TestAuthHandlerEnhancements(t *testing.T) {
	t.Run("security mode detection", func(t *testing.T) {
		tests := []struct {
			name     string
			secret   string
			keywords []string
			expected SecurityMode
		}{
			{"no security", "", []string{}, SecurityModeNone},
			{"signature only", "secret", []string{}, SecurityModeSignatureOnly},
			{"keywords only", "", []string{"alert"}, SecurityModeKeywordsOnly},
			{"both", "secret", []string{"alert"}, SecurityModeSignatureKeywords},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := NewAuthHandler(tt.secret, tt.keywords)
				if handler.GetSecurityMode() != tt.expected {
					t.Errorf("expected mode %v, got %v", tt.expected, handler.GetSecurityMode())
				}
			})
		}
	})

	t.Run("signature verification", func(t *testing.T) {
		handler := NewAuthHandler("test-secret", []string{})
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		// Generate valid signature
		validSignature := handler.generateSign(timestamp)

		// Valid signature should pass
		if err := handler.VerifySignature(timestamp, validSignature); err != nil {
			t.Errorf("valid signature should pass: %v", err)
		}

		// Invalid signature should fail
		if err := handler.VerifySignature(timestamp, "invalid"); err == nil {
			t.Error("invalid signature should fail")
		}

		// Empty timestamp should fail
		if err := handler.VerifySignature("", validSignature); err == nil {
			t.Error("empty timestamp should fail")
		}

		// Old timestamp should fail
		oldTimestamp := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
		if err := handler.VerifySignature(oldTimestamp, "signature"); err == nil {
			t.Error("old timestamp should fail")
		}
	})

	t.Run("keyword validation", func(t *testing.T) {
		handler := NewAuthHandler("", []string{"alert", "warning"})

		// Valid message should pass
		if err := handler.ValidateKeywordRequirement("This is an alert message"); err != nil {
			t.Errorf("valid message should pass: %v", err)
		}

		// Invalid message should fail
		if err := handler.ValidateKeywordRequirement("normal message"); err == nil {
			t.Error("message without keywords should fail")
		}

		// Empty message should fail
		if err := handler.ValidateKeywordRequirement(""); err == nil {
			t.Error("empty message should fail")
		}
	})

	t.Run("auth error diagnostics", func(t *testing.T) {
		handler := NewAuthHandler("secret", []string{"alert"})

		err := handler.newAuthError("TEST_CODE", "Test message", map[string]interface{}{
			"custom": "value",
		})

		// Verify error structure
		if err.Code != "TEST_CODE" {
			t.Errorf("expected code TEST_CODE, got %s", err.Code)
		}

		if err.Message != "Test message" {
			t.Errorf("expected message 'Test message', got %s", err.Message)
		}

		// Verify diagnostic details are auto-added
		if err.Details["mode"] != SecurityModeSignatureKeywords {
			t.Error("mode should be auto-added to error details")
		}

		if err.Details["has_secret"] != true {
			t.Error("has_secret should be auto-added to error details")
		}

		if err.Details["keywords_count"] != 1 {
			t.Error("keywords_count should be auto-added to error details")
		}

		// Verify custom details are preserved
		if err.Details["custom"] != "value" {
			t.Error("custom details should be preserved")
		}
	})

	t.Run("diagnostic info", func(t *testing.T) {
		handler := NewAuthHandler("secret", []string{"alert", "warning"})

		info := handler.GetDiagnosticInfo()

		if info["security_mode"] != SecurityModeSignatureKeywords {
			t.Error("diagnostic info should include security mode")
		}

		if info["has_secret"] != true {
			t.Error("diagnostic info should include secret status")
		}

		if info["keywords_count"] != 2 {
			t.Error("diagnostic info should include keywords count")
		}

		if info["secret_length"] != 6 {
			t.Error("diagnostic info should include secret length")
		}

		supportedModes := info["supported_modes"].([]SecurityMode)
		if len(supportedModes) != 4 {
			t.Error("diagnostic info should include all supported modes")
		}
	})

	t.Run("timeout configuration", func(t *testing.T) {
		customTimeout := 15 * time.Minute
		handler := NewAuthHandlerWithTimeout("secret", []string{}, customTimeout)

		if handler.timeoutWindow != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, handler.timeoutWindow)
		}

		// Test timestamp validation with custom timeout
		oldTimestamp := strconv.FormatInt(time.Now().Add(-20*time.Minute).Unix(), 10)
		if err := handler.VerifySignature(oldTimestamp, "sig"); err == nil {
			t.Error("timestamp older than custom timeout should fail")
		}
	})

	t.Run("error message format", func(t *testing.T) {
		handler := NewAuthHandler("", []string{})

		err := handler.newAuthError("TEST_CODE", "Test message", nil)
		expectedStr := "[TEST_CODE] Test message"

		if err.Error() != expectedStr {
			t.Errorf("expected error string %s, got %s", expectedStr, err.Error())
		}
	})
}

// Test security mode constants
func TestSecurityModeConstants(t *testing.T) {
	expectedModes := map[SecurityMode]string{
		SecurityModeNone:              "no_security",
		SecurityModeSignatureOnly:     "signature_only",
		SecurityModeKeywordsOnly:      "keywords_only",
		SecurityModeSignatureKeywords: "signature_and_keywords",
	}

	for mode, expectedStr := range expectedModes {
		if string(mode) != expectedStr {
			t.Errorf("expected mode %s to equal %s", string(mode), expectedStr)
		}
	}
}

// Test that signature generation is deterministic and secure
func TestSignatureGeneration(t *testing.T) {
	handler := NewAuthHandler("test-secret", []string{})
	timestamp := "1234567890"

	// Same inputs should produce same signature
	sig1 := handler.generateSign(timestamp)
	sig2 := handler.generateSign(timestamp)

	if sig1 != sig2 {
		t.Error("signature generation should be deterministic")
	}

	// Different timestamps should produce different signatures
	sig3 := handler.generateSign("9876543210")
	if sig1 == sig3 {
		t.Error("different timestamps should produce different signatures")
	}

	// Signature should be non-empty base64
	if len(sig1) == 0 {
		t.Error("signature should not be empty")
	}
}

// Test keyword functionality edge cases
func TestKeywordFunctionality(t *testing.T) {
	t.Run("case insensitive matching", func(t *testing.T) {
		handler := NewAuthHandler("", []string{"ALERT", "Warning"})

		if !handler.ContainsRequiredKeyword("this is an alert message") {
			t.Error("keyword matching should be case insensitive")
		}

		if !handler.ContainsRequiredKeyword("warning: system issue") {
			t.Error("keyword matching should be case insensitive")
		}
	})

	t.Run("keyword trimming", func(t *testing.T) {
		handler := NewAuthHandler("", []string{" alert ", "  warning  "})

		if !handler.ContainsRequiredKeyword("alert system") {
			t.Error("keywords should be trimmed during matching")
		}
	})

	t.Run("first keyword selection", func(t *testing.T) {
		handler := NewAuthHandler("", []string{"first", "second", "third"})

		if handler.GetFirstKeyword() != "first" {
			t.Error("should return first keyword")
		}
	})

	t.Run("no keywords configured", func(t *testing.T) {
		handler := NewAuthHandler("", []string{})

		if handler.GetFirstKeyword() != "" {
			t.Error("should return empty string when no keywords")
		}

		if handler.ContainsRequiredKeyword("any message") {
			t.Error("should return false when no keywords configured")
		}
	})
}