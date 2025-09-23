package platform

import (
	"fmt"
	"testing"
	"time"
)

func TestNewInternalMessage(t *testing.T) {
	id := "test-id"
	title := "Test Title"
	body := "Test Body"

	msg := NewInternalMessage(id, title, body)

	if msg.ID != id {
		t.Errorf("Expected ID %s, got %s", id, msg.ID)
	}
	if msg.Title != title {
		t.Errorf("Expected Title %s, got %s", title, msg.Title)
	}
	if msg.Body != body {
		t.Errorf("Expected Body %s, got %s", body, msg.Body)
	}
	if msg.Format != "text" {
		t.Errorf("Expected Format 'text', got %s", msg.Format)
	}
	if msg.Priority != 2 {
		t.Errorf("Expected Priority 2, got %d", msg.Priority)
	}
	if msg.Metadata == nil {
		t.Error("Expected Metadata to be initialized")
	}
	if msg.Variables == nil {
		t.Error("Expected Variables to be initialized")
	}
	if msg.PlatformData == nil {
		t.Error("Expected PlatformData to be initialized")
	}
}

func TestNewInternalTarget(t *testing.T) {
	targetType := "email"
	value := "test@example.com"
	platform := "email"

	target := NewInternalTarget(targetType, value, platform)

	if target.Type != targetType {
		t.Errorf("Expected Type %s, got %s", targetType, target.Type)
	}
	if target.Value != value {
		t.Errorf("Expected Value %s, got %s", value, target.Value)
	}
	if target.Platform != platform {
		t.Errorf("Expected Platform %s, got %s", platform, target.Platform)
	}
}

func TestNewSendResult(t *testing.T) {
	target := NewInternalTarget("email", "test@example.com", "email")
	success := true

	result := NewSendResult(target, success)

	if result.Target != target {
		t.Error("Expected Target to match")
	}
	if result.Success != success {
		t.Errorf("Expected Success %v, got %v", success, result.Success)
	}
	if result.SentAt.IsZero() {
		t.Error("Expected SentAt to be set")
	}
	if result.Metadata == nil {
		t.Error("Expected Metadata to be initialized")
	}
}

func TestSendResultSetDuration(t *testing.T) {
	target := NewInternalTarget("email", "test@example.com", "email")
	result := NewSendResult(target, true)
	duration := time.Second * 5

	result.SetDuration(duration)

	if result.Duration != duration {
		t.Errorf("Expected Duration %v, got %v", duration, result.Duration)
	}
}

func TestNewPlatformCapabilities(t *testing.T) {
	name := "test-platform"

	caps := NewPlatformCapabilities(name)

	if caps.Name != name {
		t.Errorf("Expected Name %s, got %s", name, caps.Name)
	}
	if caps.SupportedTargetTypes == nil {
		t.Error("Expected SupportedTargetTypes to be initialized")
	}
	if caps.SupportedFormats == nil {
		t.Error("Expected SupportedFormats to be initialized")
	}
	if caps.RequiredSettings == nil {
		t.Error("Expected RequiredSettings to be initialized")
	}
}

func TestNewPlatformError(t *testing.T) {
	platform := "test-platform"
	code := "TEST_ERROR"
	message := "Test error message"
	cause := &PlatformError{Platform: "other", Code: "OTHER", Message: "other"}

	err := NewPlatformError(platform, code, message, cause)

	if err.Platform != platform {
		t.Errorf("Expected Platform %s, got %s", platform, err.Platform)
	}
	if err.Code != code {
		t.Errorf("Expected Code %s, got %s", code, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected Message %s, got %s", message, err.Message)
	}
	if err.Cause != cause {
		t.Error("Expected Cause to match")
	}
}

func TestPlatformErrorError(t *testing.T) {
	message := "Test error message"
	err := NewPlatformError("test", "TEST", message, nil)

	if err.Error() != message {
		t.Errorf("Expected Error() to return %s, got %s", message, err.Error())
	}
}

func TestPlatformErrorUnwrap(t *testing.T) {
	cause := &PlatformError{Platform: "other", Code: "OTHER", Message: "other"}
	err := NewPlatformError("test", "TEST", "test message", cause)

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Error("Expected Unwrap() to return the cause")
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Network error is retryable",
			err:      NewPlatformError("test", ErrorCodeNetworkError, "network error", nil),
			expected: true,
		},
		{
			name:     "Timeout error is retryable",
			err:      NewPlatformError("test", ErrorCodeTimeout, "timeout", nil),
			expected: true,
		},
		{
			name:     "Rate limited error is retryable",
			err:      NewPlatformError("test", ErrorCodeRateLimited, "rate limited", nil),
			expected: true,
		},
		{
			name:     "Platform down error is retryable",
			err:      NewPlatformError("test", ErrorCodePlatformDown, "platform down", nil),
			expected: true,
		},
		{
			name:     "Invalid config error is not retryable",
			err:      NewPlatformError("test", ErrorCodeInvalidConfig, "invalid config", nil),
			expected: false,
		},
		{
			name:     "Non-platform error is not retryable",
			err:      &PlatformError{Platform: "test", Code: "OTHER", Message: "other"},
			expected: false,
		},
		{
			name:     "Regular error is not retryable",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected IsRetryableError(%v) = %v, got %v", tt.err, tt.expected, result)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "Platform error returns code",
			err:      NewPlatformError("test", ErrorCodeNetworkError, "network error", nil),
			expected: ErrorCodeNetworkError,
		},
		{
			name:     "Regular error returns empty string",
			err:      fmt.Errorf("regular error"),
			expected: "",
		},
		{
			name:     "Nil error returns empty string",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorCode(tt.err)
			if result != tt.expected {
				t.Errorf("Expected GetErrorCode(%v) = %s, got %s", tt.err, tt.expected, result)
			}
		})
	}
}
