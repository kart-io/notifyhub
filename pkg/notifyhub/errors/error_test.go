package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestNotifyError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotifyError
		expected string
	}{
		{
			name: "error with details",
			err: &NotifyError{
				Code:    ErrInvalidMessage,
				Message: "Invalid message format",
				Details: "Message body is empty",
			},
			expected: "[MSG001] Invalid message format: Message body is empty",
		},
		{
			name: "error without details",
			err: &NotifyError{
				Code:    ErrPlatformUnavailable,
				Message: "Platform unavailable",
			},
			expected: "[PLT002] Platform unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("NotifyError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNotifyError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := New(ErrInternalError, "wrapped error").WithCause(cause)

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("NotifyError.Unwrap() = %v, want %v", unwrapped, cause)
	}

	errWithoutCause := New(ErrInvalidConfig, "config error")
	if unwrapped := errWithoutCause.Unwrap(); unwrapped != nil {
		t.Errorf("NotifyError.Unwrap() = %v, want nil", unwrapped)
	}
}

func TestNotifyError_Is(t *testing.T) {
	err1 := New(ErrInvalidMessage, "message 1")
	err2 := New(ErrInvalidMessage, "message 2")
	err3 := New(ErrPlatformUnavailable, "different error")

	if !err1.Is(err2) {
		t.Error("Errors with same code should match")
	}

	if err1.Is(err3) {
		t.Error("Errors with different codes should not match")
	}

	stdErr := errors.New("standard error")
	if err1.Is(stdErr) {
		t.Error("NotifyError should not match standard errors")
	}
}

func TestNotifyError_WithContext(t *testing.T) {
	err := New(ErrPlatformAuth, "auth failed")
	err.WithContext("platform", "feishu").WithContext("user_id", "123")

	if err.Context["platform"] != "feishu" {
		t.Errorf("Expected platform context to be 'feishu', got %v", err.Context["platform"])
	}

	if err.Context["user_id"] != "123" {
		t.Errorf("Expected user_id context to be '123', got %v", err.Context["user_id"])
	}
}

func TestNotifyError_WithDetails(t *testing.T) {
	err := New(ErrMessageTooLarge, "message too large")
	err.WithDetails("Message size: 5MB, limit: 1MB")

	if err.Details != "Message size: 5MB, limit: 1MB" {
		t.Errorf("Expected details to be set, got %v", err.Details)
	}
}

func TestNotifyError_WithCause(t *testing.T) {
	cause := errors.New("network timeout")
	err := New(ErrNetworkTimeout, "request failed").WithCause(cause)

	if err.Cause != cause {
		t.Errorf("Expected cause to be set, got %v", err.Cause)
	}
}

func TestErrorBuilder(t *testing.T) {
	cause := errors.New("original error")
	stackTrace := []string{"func1", "func2", "func3"}

	err := NewErrorBuilder(ErrPlatformInternal, "platform error").
		WithDetails("Internal server error").
		WithContext("platform", "feishu").
		WithContext("endpoint", "/webhook").
		WithCause(cause).
		WithStackTrace(stackTrace).
		Build()

	if err.Code != ErrPlatformInternal {
		t.Errorf("Expected code %v, got %v", ErrPlatformInternal, err.Code)
	}

	if err.Message != "platform error" {
		t.Errorf("Expected message 'platform error', got %v", err.Message)
	}

	if err.Details != "Internal server error" {
		t.Errorf("Expected details 'Internal server error', got %v", err.Details)
	}

	if err.Context["platform"] != "feishu" {
		t.Errorf("Expected platform context 'feishu', got %v", err.Context["platform"])
	}

	if err.Cause != cause {
		t.Errorf("Expected cause to be set, got %v", err.Cause)
	}

	if len(err.StackTrace) != 3 {
		t.Errorf("Expected stack trace length 3, got %v", len(err.StackTrace))
	}

	// Verify timestamp is set
	if err.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestNew(t *testing.T) {
	err := New(ErrInvalidConfig, "configuration error")

	if err.Code != ErrInvalidConfig {
		t.Errorf("Expected code %v, got %v", ErrInvalidConfig, err.Code)
	}

	if err.Message != "configuration error" {
		t.Errorf("Expected message 'configuration error', got %v", err.Message)
	}

	if err.Context == nil {
		t.Error("Expected context to be initialized")
	}

	if err.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	err := Wrap(cause, ErrNetworkConnection, "connection failed")

	if err.Cause != cause {
		t.Errorf("Expected cause to be set, got %v", err.Cause)
	}

	if err.Code != ErrNetworkConnection {
		t.Errorf("Expected code %v, got %v", ErrNetworkConnection, err.Code)
	}

	if err.Message != "connection failed" {
		t.Errorf("Expected message 'connection failed', got %v", err.Message)
	}
}

func TestWrapf(t *testing.T) {
	cause := errors.New("timeout")
	err := Wrapf(cause, ErrNetworkTimeout, "request failed after %d seconds", 30)

	expectedMessage := "request failed after 30 seconds"
	if err.Message != expectedMessage {
		t.Errorf("Expected message '%s', got %v", expectedMessage, err.Message)
	}

	if err.Cause != cause {
		t.Errorf("Expected cause to be set, got %v", err.Cause)
	}
}

func TestErrorAggregator(t *testing.T) {
	aggregator := NewErrorAggregator()

	// Test empty aggregator
	if aggregator.HasErrors() {
		t.Error("New aggregator should not have errors")
	}

	if aggregator.Count() != 0 {
		t.Errorf("Expected count 0, got %d", aggregator.Count())
	}

	if err := aggregator.ToError(); err != nil {
		t.Errorf("Expected nil error for empty aggregator, got %v", err)
	}

	// Add single error
	err1 := New(ErrPlatformUnavailable, "platform 1 failed")
	aggregator.Add(err1)

	if !aggregator.HasErrors() {
		t.Error("Aggregator should have errors after adding one")
	}

	if aggregator.Count() != 1 {
		t.Errorf("Expected count 1, got %d", aggregator.Count())
	}

	// Single error should return the error itself
	if singleErr := aggregator.ToError(); singleErr != err1 {
		t.Errorf("Expected single error to be returned, got %v", singleErr)
	}

	// Add more errors
	err2 := New(ErrPlatformAuth, "platform 2 auth failed").WithContext("platform", "email")
	err3 := New(ErrNetworkTimeout, "network timeout")
	aggregator.Add(err2)
	aggregator.Add(err3)

	if aggregator.Count() != 3 {
		t.Errorf("Expected count 3, got %d", aggregator.Count())
	}

	// Test aggregated error
	aggregatedErr := aggregator.ToError()
	if aggregatedErr == nil {
		t.Fatal("Expected aggregated error, got nil")
	}

	notifyErr, ok := aggregatedErr.(*NotifyError)
	if !ok {
		t.Fatal("Expected NotifyError, got different type")
	}

	if notifyErr.Code != ErrInternalError {
		t.Errorf("Expected code %v, got %v", ErrInternalError, notifyErr.Code)
	}

	// Check context
	if count, exists := notifyErr.Context["error_count"]; !exists || count != 3 {
		t.Errorf("Expected error_count 3, got %v", count)
	}

	// Test adding nil error (should be ignored)
	aggregator.Add(nil)
	if aggregator.Count() != 3 {
		t.Errorf("Expected count to remain 3 after adding nil, got %d", aggregator.Count())
	}
}

func TestErrorFormatter(t *testing.T) {
	formatter := &ErrorFormatter{}

	// Create a complex error for testing
	cause := errors.New("network timeout")
	err := NewErrorBuilder(ErrPlatformUnavailable, "platform unavailable").
		WithDetails("Service is temporarily down").
		WithContext("platform", "feishu").
		WithContext("endpoint", "/webhook").
		WithContext("secret", "sensitive-data"). // This should be filtered in API format
		WithCause(cause).
		WithStackTrace([]string{"func1", "func2"}).
		Build()

	t.Run("FormatForLogging", func(t *testing.T) {
		result := formatter.FormatForLogging(err)

		// Check required fields
		if result["error_message"] != err.Error() {
			t.Errorf("Expected error_message %v, got %v", err.Error(), result["error_message"])
		}

		if result["error_code"] != string(ErrPlatformUnavailable) {
			t.Errorf("Expected error_code %v, got %v", string(ErrPlatformUnavailable), result["error_code"])
		}

		if result["category"] != PlatformCategory {
			t.Errorf("Expected category %v, got %v", PlatformCategory, result["category"])
		}

		if result["retryable"] != true {
			t.Errorf("Expected retryable true, got %v", result["retryable"])
		}

		// Check context is included
		context, exists := result["context"]
		if !exists {
			t.Error("Expected context to be included")
		}

		contextMap, ok := context.(map[string]interface{})
		if !ok {
			t.Error("Expected context to be a map")
		}

		if contextMap["platform"] != "feishu" {
			t.Errorf("Expected platform feishu, got %v", contextMap["platform"])
		}
	})

	t.Run("FormatForAPI", func(t *testing.T) {
		result := formatter.FormatForAPI(err)

		// Check safe fields
		if result["code"] != string(ErrPlatformUnavailable) {
			t.Errorf("Expected code %v, got %v", string(ErrPlatformUnavailable), result["code"])
		}

		if result["message"] != "platform unavailable" {
			t.Errorf("Expected message 'platform unavailable', got %v", result["message"])
		}

		// Check that only safe context fields are included
		context, exists := result["context"]
		if !exists {
			t.Error("Expected safe context to be included")
		}

		contextMap, ok := context.(map[string]interface{})
		if !ok {
			t.Error("Expected context to be a map")
		}

		if contextMap["platform"] != "feishu" {
			t.Errorf("Expected platform feishu, got %v", contextMap["platform"])
		}

		// Sensitive field should not be included
		if _, exists := contextMap["secret"]; exists {
			t.Error("Expected sensitive 'secret' field to be filtered out")
		}

		// Stack trace should not be included in API format
		if _, exists := result["stack_trace"]; exists {
			t.Error("Expected stack_trace to be excluded from API format")
		}
	})

	t.Run("FormatForDebug", func(t *testing.T) {
		result := formatter.FormatForDebug(err)

		// Debug format should include cause information
		if result["cause"] != cause.Error() {
			t.Errorf("Expected cause %v, got %v", cause.Error(), result["cause"])
		}

		if result["cause_type"] != fmt.Sprintf("%T", cause) {
			t.Errorf("Expected cause_type %v, got %v", fmt.Sprintf("%T", cause), result["cause_type"])
		}
	})

	t.Run("FormatStandardError", func(t *testing.T) {
		stdErr := errors.New("standard error")
		result := formatter.FormatForLogging(stdErr)

		// Should handle standard errors gracefully
		if result["error_message"] != stdErr.Error() {
			t.Errorf("Expected error_message %v, got %v", stdErr.Error(), result["error_message"])
		}

		if result["error_type"] != "*errors.errorString" {
			t.Errorf("Expected error_type *errors.errorString, got %v", result["error_type"])
		}

		// Should not include NotifyError-specific fields
		if _, exists := result["error_code"]; exists {
			t.Error("Standard error should not include error_code")
		}
	})
}

func TestErrorSerializer(t *testing.T) {
	serializer := NewErrorSerializer()

	err := New(ErrMessageTooLarge, "message too large").
		WithDetails("Size: 5MB, Limit: 1MB").
		WithContext("platform", "email").
		WithContext("size", 5242880)

	t.Run("SerializeForLogging", func(t *testing.T) {
		data, serErr := serializer.SerializeForLogging(err)
		if serErr != nil {
			t.Fatalf("Expected no serialization error, got %v", serErr)
		}

		// Verify it's valid JSON
		var result map[string]interface{}
		if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
			t.Fatalf("Expected valid JSON, got error: %v", jsonErr)
		}

		// Check key fields
		if result["error_code"] != string(ErrMessageTooLarge) {
			t.Errorf("Expected error_code %v, got %v", string(ErrMessageTooLarge), result["error_code"])
		}
	})

	t.Run("SerializeForAPI", func(t *testing.T) {
		data, serErr := serializer.SerializeForAPI(err)
		if serErr != nil {
			t.Fatalf("Expected no serialization error, got %v", serErr)
		}

		// Verify it's valid JSON
		var result map[string]interface{}
		if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
			t.Fatalf("Expected valid JSON, got error: %v", jsonErr)
		}

		// Check that it doesn't include sensitive information
		if _, exists := result["stack_trace"]; exists {
			t.Error("API serialization should not include stack_trace")
		}
	})

	t.Run("SerializeForDebug", func(t *testing.T) {
		cause := errors.New("underlying cause")
		errWithCause := err.WithCause(cause)

		data, serErr := serializer.SerializeForDebug(errWithCause)
		if serErr != nil {
			t.Fatalf("Expected no serialization error, got %v", serErr)
		}

		// Verify it's valid JSON
		var result map[string]interface{}
		if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
			t.Fatalf("Expected valid JSON, got error: %v", jsonErr)
		}

		// Debug should include cause information
		if result["cause"] != cause.Error() {
			t.Errorf("Expected cause %v, got %v", cause.Error(), result["cause"])
		}
	})
}

func TestErrorAggregator_Concurrent(t *testing.T) {
	aggregator := NewErrorAggregator()

	// Test concurrent access
	done := make(chan bool)
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			err := New(ErrPlatformUnavailable, fmt.Sprintf("error %d", id))
			aggregator.Add(err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	if aggregator.Count() != numGoroutines {
		t.Errorf("Expected %d errors, got %d", numGoroutines, aggregator.Count())
	}
}

func BenchmarkNotifyError_Creation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := New(ErrInvalidMessage, "test error").
			WithContext("platform", "feishu").
			WithContext("message_id", "12345").
			WithDetails("test details")
		_ = err
	}
}

func BenchmarkNotifyError_Error(b *testing.B) {
	err := New(ErrInvalidMessage, "test error").WithDetails("test details")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkErrorFormatter_FormatForLogging(b *testing.B) {
	formatter := &ErrorFormatter{}
	err := New(ErrPlatformUnavailable, "platform error").
		WithContext("platform", "feishu").
		WithContext("endpoint", "/webhook").
		WithDetails("Service temporarily unavailable")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatter.FormatForLogging(err)
	}
}

func BenchmarkErrorAggregator_Add(b *testing.B) {
	aggregator := NewErrorAggregator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := New(ErrPlatformUnavailable, "test error")
		aggregator.Add(err)
	}
}