package feishu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

func TestNewAuthHandler(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		keywords []string
	}{
		{
			name:     "with secret and keywords",
			secret:   "test-secret",
			keywords: []string{"keyword1", "keyword2"},
		},
		{
			name:     "with secret only",
			secret:   "test-secret",
			keywords: nil,
		},
		{
			name:     "with keywords only",
			secret:   "",
			keywords: []string{"keyword1"},
		},
		{
			name:     "no security",
			secret:   "",
			keywords: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAuthHandler(tt.secret, tt.keywords)
			if got.secret != tt.secret {
				t.Errorf("NewAuthHandler() secret = %v, want %v", got.secret, tt.secret)
			}
			if len(got.keywords) != len(tt.keywords) {
				t.Errorf("NewAuthHandler() keywords length = %v, want %v", len(got.keywords), len(tt.keywords))
			}
			for i, keyword := range got.keywords {
				if i < len(tt.keywords) && keyword != tt.keywords[i] {
					t.Errorf("NewAuthHandler() keywords[%d] = %v, want %v", i, keyword, tt.keywords[i])
				}
			}
		})
	}
}

func TestAuthHandler_SecurityMode(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		keywords []string
		want     SecurityMode
	}{
		{
			name:     "signature and keywords",
			secret:   "test-secret",
			keywords: []string{"keyword1"},
			want:     SecurityModeSignatureKeywords,
		},
		{
			name:     "signature only",
			secret:   "test-secret",
			keywords: nil,
			want:     SecurityModeSignatureOnly,
		},
		{
			name:     "keywords only",
			secret:   "",
			keywords: []string{"keyword1"},
			want:     SecurityModeKeywordsOnly,
		},
		{
			name:     "no security",
			secret:   "",
			keywords: nil,
			want:     SecurityModeNone,
		},
		{
			name:     "empty keywords slice",
			secret:   "",
			keywords: []string{},
			want:     SecurityModeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(tt.secret, tt.keywords)
			got := handler.GetSecurityMode()
			if got != tt.want {
				t.Errorf("GetSecurityMode() = %v, want %v", got, tt.want)
			}
			if handler.mode != tt.want {
				t.Errorf("handler.mode = %v, want %v", handler.mode, tt.want)
			}
		})
	}
}

func TestAuthHandler_GenerateSign(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		timestamp string
	}{
		{
			name:      "basic signature generation",
			secret:    "test-secret",
			timestamp: "1234567890",
		},
		{
			name:      "different secret",
			secret:    "another-secret",
			timestamp: "1234567890",
		},
		{
			name:      "different timestamp",
			secret:    "test-secret",
			timestamp: "9876543210",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(tt.secret, nil)
			got := handler.generateSign(tt.timestamp)

			// Verify signature can be decoded as base64
			decoded, err := base64.StdEncoding.DecodeString(got)
			if err != nil {
				t.Errorf("generateSign() signature should be valid base64, got error: %v", err)
			}
			if len(decoded) != 32 {
				t.Errorf("generateSign() decoded signature should be 32 bytes (SHA256), got %d", len(decoded))
			}

			// Verify signature is deterministic
			got2 := handler.generateSign(tt.timestamp)
			if got != got2 {
				t.Errorf("generateSign() should be deterministic, got different signatures: %v vs %v", got, got2)
			}

			// Verify expected algorithm
			expected := generateExpectedSignature(tt.secret, tt.timestamp)
			if got != expected {
				t.Errorf("generateSign() = %v, want %v", got, expected)
			}
		})
	}
}

func TestAuthHandler_AddSignature(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		msgType string
	}{
		{
			name:    "text message with signature",
			secret:  "test-secret",
			msgType: "text",
		},
		{
			name:    "rich text message with signature",
			secret:  "test-secret",
			msgType: "post",
		},
		{
			name:    "no secret provided",
			secret:  "",
			msgType: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(tt.secret, nil)
			feishuMsg := &FeishuMessage{
				MsgType: tt.msgType,
				Content: &FeishuTextContent{Text: "test message"},
			}

			handler.addSignature(feishuMsg)

			if tt.secret != "" {
				if feishuMsg.Sign == "" {
					t.Error("signature should be added when secret is provided")
				}
				if feishuMsg.Timestamp == "" {
					t.Error("timestamp should be added when secret is provided")
				}

				// Verify the timestamp is a valid Unix timestamp
				timestamp, err := strconv.ParseInt(feishuMsg.Timestamp, 10, 64)
				if err != nil {
					t.Errorf("timestamp should be a valid integer, got error: %v", err)
				}
				if timestamp <= 0 {
					t.Errorf("timestamp should be positive, got %d", timestamp)
				}

				// Verify the signature is properly formatted (base64)
				_, err = base64.StdEncoding.DecodeString(feishuMsg.Sign)
				if err != nil {
					t.Errorf("signature should be valid base64, got error: %v", err)
				}
			} else {
				if feishuMsg.Sign != "" {
					t.Error("signature should not be added when no secret")
				}
				if feishuMsg.Timestamp != "" {
					t.Error("timestamp should not be added when no secret")
				}
			}
		})
	}
}

func TestAuthHandler_ContainsRequiredKeyword(t *testing.T) {
	tests := []struct {
		name        string
		keywords    []string
		messageText string
		want        bool
	}{
		{
			name:        "exact keyword match",
			keywords:    []string{"alert"},
			messageText: "This is an alert message",
			want:        true,
		},
		{
			name:        "case insensitive match",
			keywords:    []string{"Alert"},
			messageText: "this is an alert message",
			want:        true,
		},
		{
			name:        "multiple keywords - first match",
			keywords:    []string{"warning", "error"},
			messageText: "warning: something happened",
			want:        true,
		},
		{
			name:        "multiple keywords - second match",
			keywords:    []string{"warning", "error"},
			messageText: "an error occurred",
			want:        true,
		},
		{
			name:        "no match",
			keywords:    []string{"alert"},
			messageText: "normal message",
			want:        false,
		},
		{
			name:        "empty message",
			keywords:    []string{"alert"},
			messageText: "",
			want:        false,
		},
		{
			name:        "no keywords",
			keywords:    nil,
			messageText: "any message",
			want:        false,
		},
		{
			name:        "empty keywords slice",
			keywords:    []string{},
			messageText: "any message",
			want:        false,
		},
		{
			name:        "keyword with spaces",
			keywords:    []string{" alert "},
			messageText: "this is an alert message",
			want:        true,
		},
		{
			name:        "partial keyword match",
			keywords:    []string{"warn"},
			messageText: "warning message",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler("", tt.keywords)
			got := handler.ContainsRequiredKeyword(tt.messageText)
			if got != tt.want {
				t.Errorf("ContainsRequiredKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthHandler_GetFirstKeyword(t *testing.T) {
	tests := []struct {
		name     string
		keywords []string
		want     string
	}{
		{
			name:     "single keyword",
			keywords: []string{"alert"},
			want:     "alert",
		},
		{
			name:     "multiple keywords",
			keywords: []string{"warning", "error", "info"},
			want:     "warning",
		},
		{
			name:     "no keywords",
			keywords: nil,
			want:     "",
		},
		{
			name:     "empty keywords slice",
			keywords: []string{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler("", tt.keywords)
			got := handler.GetFirstKeyword()
			if got != tt.want {
				t.Errorf("GetFirstKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthHandler_AddAuth(t *testing.T) {
	tests := []struct {
		name           string
		secret         string
		keywords       []string
		feishuMsg      *FeishuMessage
		expectError    bool
		expectSign     bool
	}{
		{
			name:     "no security mode",
			secret:   "",
			keywords: nil,
			feishuMsg: &FeishuMessage{
				MsgType: "text",
				Content: &FeishuTextContent{Text: "test message"},
			},
			expectError: false,
			expectSign:  false,
		},
		{
			name:     "signature only mode",
			secret:   "test-secret",
			keywords: nil,
			feishuMsg: &FeishuMessage{
				MsgType: "text",
				Content: &FeishuTextContent{Text: "test message"},
			},
			expectError: false,
			expectSign:  true,
		},
		{
			name:     "keywords only mode",
			secret:   "",
			keywords: []string{"alert"},
			feishuMsg: &FeishuMessage{
				MsgType: "text",
				Content: &FeishuTextContent{Text: "test message"},
			},
			expectError: false,
			expectSign:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(tt.secret, tt.keywords)
			err := handler.AddAuth(tt.feishuMsg)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectSign {
				if tt.feishuMsg.Sign == "" {
					t.Error("signature should be present")
				}
				if tt.feishuMsg.Timestamp == "" {
					t.Error("timestamp should be present")
				}
			} else {
				if tt.feishuMsg.Sign != "" {
					t.Error("signature should not be present")
				}
				if tt.feishuMsg.Timestamp != "" {
					t.Error("timestamp should not be present")
				}
			}
		})
	}
}

func TestAuthHandler_ProcessKeywordRequirement(t *testing.T) {
	testLogger := logger.Discard // Use discard logger to reduce test output
	config := &FeishuConfig{}
	builder := NewMessageBuilder(config, testLogger)

	tests := []struct {
		name           string
		keywords       []string
		initialText    string
		expectedPrefix string
		expectError    bool
	}{
		{
			name:           "no keywords configured",
			keywords:       nil,
			initialText:    "test message",
			expectedPrefix: "",
			expectError:    false,
		},
		{
			name:           "keyword already present",
			keywords:       []string{"alert"},
			initialText:    "alert: test message",
			expectedPrefix: "",
			expectError:    false,
		},
		{
			name:           "keyword needs to be added",
			keywords:       []string{"warning"},
			initialText:    "test message",
			expectedPrefix: "warning ",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler("", tt.keywords)

			msg := &message.Message{
				Title: "Test",
				Body:  tt.initialText,
			}

			feishuMsg := &FeishuMessage{
				MsgType: "text",
				Content: &FeishuTextContent{Text: tt.initialText},
			}

			err := handler.ProcessKeywordRequirement(feishuMsg, msg, builder)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectedPrefix != "" {
				content, ok := feishuMsg.Content.(*FeishuTextContent)
				if !ok {
					t.Fatal("content should be text content")
				}
				if !strings.HasPrefix(content.Text, tt.expectedPrefix) {
					t.Errorf("text should have expected prefix. Got: %s, Expected prefix: %s",
						content.Text, tt.expectedPrefix)
				}
			}
		})
	}
}

// Helper function to generate expected signature for testing
func generateExpectedSignature(secret, timestamp string) string {
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, secret)
	hash := hmac.New(sha256.New, []byte(stringToSign))
	hash.Write([]byte("")) // Feishu uses empty string as data
	signature := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return signature
}

// Benchmark tests for authentication operations
func BenchmarkAuthHandler_GenerateSign(b *testing.B) {
	handler := NewAuthHandler("test-secret", nil)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.generateSign(timestamp)
	}
}

func BenchmarkAuthHandler_ContainsRequiredKeyword(b *testing.B) {
	handler := NewAuthHandler("", []string{"alert", "warning", "error"})
	messageText := "This is an alert message with some additional content to make it realistic"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ContainsRequiredKeyword(messageText)
	}
}

func BenchmarkAuthHandler_AddAuth(b *testing.B) {
	handler := NewAuthHandler("test-secret", []string{"alert"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		feishuMsg := &FeishuMessage{
			MsgType: "text",
			Content: &FeishuTextContent{Text: "test message"},
		}
		handler.AddAuth(feishuMsg)
	}
}

// Integration test to verify signature generation matches Feishu's expected format
func TestAuthHandler_SignatureIntegration(t *testing.T) {
	// This test verifies the signature generation follows Feishu's documented algorithm
	secret := "test-webhook-secret"
	timestamp := "1234567890"

	handler := NewAuthHandler(secret, nil)
	signature := handler.generateSign(timestamp)

	// Verify signature can be decoded as base64
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		t.Errorf("signature should be valid base64, got error: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("decoded signature should be 32 bytes (SHA256), got %d", len(decoded))
	}

	// Verify signature is deterministic
	signature2 := handler.generateSign(timestamp)
	if signature != signature2 {
		t.Errorf("signature should be deterministic, got %v vs %v", signature, signature2)
	}

	// Verify different timestamps produce different signatures
	signature3 := handler.generateSign("9876543210")
	if signature == signature3 {
		t.Error("different timestamps should produce different signatures")
	}
}

// Security test to ensure proper error handling and no information leakage
func TestAuthHandler_SecurityEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		secret      string
		keywords    []string
		messageText string
		description string
	}{
		{
			name:        "very long secret",
			secret:      generateLongString(1000),
			keywords:    nil,
			messageText: "test",
			description: "should handle very long secrets without issues",
		},
		{
			name:        "very long keyword",
			secret:      "",
			keywords:    []string{generateLongString(500)},
			messageText: "test message",
			description: "should handle very long keywords",
		},
		{
			name:        "many keywords",
			secret:      "",
			keywords:    generateManyKeywords(100),
			messageText: "test message",
			description: "should handle many keywords efficiently",
		},
		{
			name:        "special characters in keyword",
			secret:      "",
			keywords:    []string{"alert🚨", "warning⚠️"},
			messageText: "test alert🚨 message",
			description: "should handle unicode characters in keywords",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(tt.secret, tt.keywords)

			// These operations should complete without panic or excessive delay
			securityMode := handler.determineSecurityMode()
			if securityMode == "" {
				t.Errorf("%s: security mode should not be empty", tt.description)
			}

			if tt.secret != "" {
				timestamp := strconv.FormatInt(time.Now().Unix(), 10)
				signature := handler.generateSign(timestamp)
				if signature == "" {
					t.Error("signature should be generated for any valid secret")
				}
			}

			if len(tt.keywords) > 0 {
				hasKeyword := handler.ContainsRequiredKeyword(tt.messageText)
				// Should complete without error, result depends on content
				_ = hasKeyword
			}
		})
	}
}

// Test new enhanced functionality
func TestAuthHandler_VerifySignature(t *testing.T) {
	secret := "test-secret"
	handler := NewAuthHandler(secret, []string{})

	tests := []struct {
		name        string
		timestamp   string
		signature   string
		expectError bool
		errorCode   string
	}{
		{
			name:        "valid signature",
			timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
			signature:   "", // Will be generated
			expectError: false,
		},
		{
			name:        "empty timestamp",
			timestamp:   "",
			signature:   "some-signature",
			expectError: true,
			errorCode:   "EMPTY_TIMESTAMP",
		},
		{
			name:        "invalid timestamp format",
			timestamp:   "not-a-number",
			signature:   "some-signature",
			expectError: true,
			errorCode:   "INVALID_TIMESTAMP_FORMAT",
		},
		{
			name:        "timestamp expired",
			timestamp:   strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10),
			signature:   "some-signature",
			expectError: true,
			errorCode:   "TIMESTAMP_EXPIRED",
		},
		{
			name:        "timestamp too future",
			timestamp:   strconv.FormatInt(time.Now().Add(2*time.Minute).Unix(), 10),
			signature:   "some-signature",
			expectError: true,
			errorCode:   "TIMESTAMP_TOO_FUTURE",
		},
		{
			name:        "invalid signature",
			timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
			signature:   "invalid-signature",
			expectError: true,
			errorCode:   "SIGNATURE_VERIFICATION_FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature := tt.signature
			if tt.name == "valid signature" {
				signature = handler.generateSign(tt.timestamp)
			}

			err := handler.VerifySignature(tt.timestamp, signature)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if authErr, ok := err.(*AuthError); ok {
					if authErr.Code != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, authErr.Code)
					}
				} else {
					t.Errorf("expected AuthError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestAuthHandler_VerifySignatureNoSecret(t *testing.T) {
	handler := NewAuthHandler("", []string{})

	err := handler.VerifySignature("123456789", "some-signature")
	if err == nil {
		t.Error("expected error when no secret configured")
		return
	}

	if authErr, ok := err.(*AuthError); ok {
		if authErr.Code != "NO_SECRET_CONFIGURED" {
			t.Errorf("expected error code NO_SECRET_CONFIGURED, got %s", authErr.Code)
		}
	} else {
		t.Errorf("expected AuthError, got %T", err)
	}
}

func TestAuthHandler_ValidateKeywordRequirement(t *testing.T) {
	handler := NewAuthHandler("", []string{"alert", "notification"})

	tests := []struct {
		name        string
		messageText string
		expectError bool
		errorCode   string
	}{
		{
			name:        "valid message with keyword",
			messageText: "This is an alert message",
			expectError: false,
		},
		{
			name:        "empty message text",
			messageText: "",
			expectError: true,
			errorCode:   "EMPTY_MESSAGE_TEXT",
		},
		{
			name:        "missing required keywords",
			messageText: "This is a regular message",
			expectError: true,
			errorCode:   "KEYWORD_REQUIREMENT_NOT_MET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.ValidateKeywordRequirement(tt.messageText)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if authErr, ok := err.(*AuthError); ok {
					if authErr.Code != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, authErr.Code)
					}
				} else {
					t.Errorf("expected AuthError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestAuthHandler_GetDiagnosticInfo(t *testing.T) {
	secret := "test-secret"
	keywords := []string{"alert", "notification"}
	handler := NewAuthHandler(secret, keywords)

	info := handler.GetDiagnosticInfo()

	if info["security_mode"] != SecurityModeSignatureKeywords {
		t.Errorf("expected security_mode %v, got %v", SecurityModeSignatureKeywords, info["security_mode"])
	}

	if info["has_secret"] != true {
		t.Error("expected has_secret to be true")
	}

	if info["secret_length"] != len(secret) {
		t.Errorf("expected secret_length %d, got %v", len(secret), info["secret_length"])
	}

	if info["keywords_count"] != len(keywords) {
		t.Errorf("expected keywords_count %d, got %v", len(keywords), info["keywords_count"])
	}

	supportedModes, ok := info["supported_modes"].([]SecurityMode)
	if !ok {
		t.Error("expected supported_modes to be []SecurityMode")
	} else if len(supportedModes) != 4 {
		t.Errorf("expected 4 supported modes, got %d", len(supportedModes))
	}
}

func TestAuthError(t *testing.T) {
	handler := NewAuthHandler("secret", []string{"alert"})

	err := handler.newAuthError("TEST_CODE", "Test message", map[string]interface{}{
		"custom_field": "custom_value",
	})

	if err.Code != "TEST_CODE" {
		t.Errorf("expected code TEST_CODE, got %s", err.Code)
	}

	if err.Message != "Test message" {
		t.Errorf("expected message 'Test message', got %s", err.Message)
	}

	if err.Details["custom_field"] != "custom_value" {
		t.Error("expected custom_field to be preserved")
	}

	if err.Details["mode"] != SecurityModeSignatureKeywords {
		t.Error("expected mode to be automatically added")
	}

	if err.Details["has_secret"] != true {
		t.Error("expected has_secret to be automatically added")
	}

	errorString := err.Error()
	expectedString := "[TEST_CODE] Test message"
	if errorString != expectedString {
		t.Errorf("expected error string %s, got %s", expectedString, errorString)
	}
}

func TestNewAuthHandlerWithTimeout(t *testing.T) {
	secret := "test-secret"
	keywords := []string{"alert"}
	timeout := 10 * time.Minute

	handler := NewAuthHandlerWithTimeout(secret, keywords, timeout)

	if handler.timeoutWindow != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, handler.timeoutWindow)
	}

	if handler.mode != SecurityModeSignatureKeywords {
		t.Errorf("expected mode %v, got %v", SecurityModeSignatureKeywords, handler.mode)
	}
}

// Helper functions for security tests
func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = byte('a' + (i % 26))
	}
	return string(result)
}

func generateManyKeywords(count int) []string {
	keywords := make([]string, count)
	for i := 0; i < count; i++ {
		keywords[i] = fmt.Sprintf("keyword%d", i)
	}
	return keywords
}