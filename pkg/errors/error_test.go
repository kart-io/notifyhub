package errors

import (
	"errors"
	"testing"
	"time"
)

func TestNotifyError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotifyError
		expected string
	}{
		{
			name: "basic error",
			err: &NotifyError{
				Code:    ErrInvalidConfig,
				Message: "invalid configuration",
			},
			expected: "INVALID_CONFIG: invalid configuration",
		},
		{
			name: "error with platform",
			err: &NotifyError{
				Code:     ErrPlatformUnavailable,
				Message:  "platform unavailable",
				Platform: "email",
			},
			expected: "PLATFORM_UNAVAILABLE: platform unavailable (platform: email)",
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

func TestNotifyError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotifyError
		expected bool
	}{
		{
			name: "retryable network error",
			err: &NotifyError{
				Code:      ErrNetworkTimeout,
				Retryable: true,
			},
			expected: true,
		},
		{
			name: "non-retryable auth error",
			err: &NotifyError{
				Code:      ErrInvalidCredentials,
				Retryable: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsRetryable(); got != tt.expected {
				t.Errorf("NotifyError.IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	code := ErrInvalidMessage
	msg := "test message"

	err := New(code, msg)

	if err.Code != code {
		t.Errorf("New() code = %v, want %v", err.Code, code)
	}
	if err.Message != msg {
		t.Errorf("New() message = %v, want %v", err.Message, msg)
	}
	if err.Timestamp.IsZero() {
		t.Error("New() timestamp should not be zero")
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	code := ErrInternal
	msg := "wrapper message"

	err := Wrap(originalErr, code, msg)

	if err.Code != code {
		t.Errorf("Wrap() code = %v, want %v", err.Code, code)
	}
	if err.Message != msg {
		t.Errorf("Wrap() message = %v, want %v", err.Message, msg)
	}
	if err.Cause != originalErr {
		t.Errorf("Wrap() cause = %v, want %v", err.Cause, originalErr)
	}
}

func TestIsConfigError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "config error",
			err:      New(ErrInvalidConfig, "test"),
			expected: true,
		},
		{
			name:     "non-config error",
			err:      New(ErrNetworkTimeout, "test"),
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("test"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConfigError(tt.err); got != tt.expected {
				t.Errorf("IsConfigError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsPlatformError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "platform error",
			err:      New(ErrPlatformUnavailable, "test"),
			expected: true,
		},
		{
			name:     "non-platform error",
			err:      New(ErrInvalidConfig, "test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPlatformError(tt.err); got != tt.expected {
				t.Errorf("IsPlatformError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "network error",
			err:      New(ErrNetworkTimeout, "test"),
			expected: true,
		},
		{
			name:     "connection error",
			err:      New(ErrConnectionFailed, "test"),
			expected: true,
		},
		{
			name:     "non-network error",
			err:      New(ErrInvalidConfig, "test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNetworkError(tt.err); got != tt.expected {
				t.Errorf("IsNetworkError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "retryable error",
			err: &NotifyError{
				Code:      ErrNetworkTimeout,
				Retryable: true,
			},
			expected: true,
		},
		{
			name: "non-retryable error",
			err: &NotifyError{
				Code:      ErrInvalidCredentials,
				Retryable: false,
			},
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.expected {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{
			name:     "notify error",
			err:      New(ErrInvalidConfig, "test"),
			expected: ErrInvalidConfig,
		},
		{
			name:     "wrapped notify error",
			err:      Wrap(errors.New("base"), ErrNetworkTimeout, "wrapped"),
			expected: ErrNetworkTimeout,
		},
		{
			name:     "standard error",
			err:      errors.New("test"),
			expected: ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorCode(tt.err); got != tt.expected {
				t.Errorf("GetErrorCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetCategory(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected string
	}{
		{
			name:     "config category",
			code:     ErrInvalidConfig,
			expected: "configuration",
		},
		{
			name:     "platform category",
			code:     ErrPlatformUnavailable,
			expected: "platform",
		},
		{
			name:     "network category",
			code:     ErrNetworkTimeout,
			expected: "network",
		},
		{
			name:     "message category",
			code:     ErrInvalidMessage,
			expected: "message",
		},
		{
			name:     "unknown category",
			code:     "UNKNOWN_CODE",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCategory(tt.code); got != tt.expected {
				t.Errorf("GetCategory() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNotifyError_String(t *testing.T) {
	err := &NotifyError{
		Code:    ErrInvalidConfig,
		Message: "test error",
	}
	result := err.String()
	expected := "INVALID_CONFIG: test error"
	if result != expected {
		t.Errorf("String() = %v, want %v", result, expected)
	}
}

func TestNotifyError_Unwrap(t *testing.T) {
	originalErr := errors.New("original")
	err := Wrap(originalErr, ErrInternal, "wrapped")

	unwrapped := err.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestNotifyError_Is(t *testing.T) {
	err1 := New(ErrInvalidConfig, "test1")
	err2 := New(ErrInvalidConfig, "test2")
	err3 := New(ErrNetworkTimeout, "test3")

	if !err1.Is(err2) {
		t.Error("Is() should return true for same error code")
	}
	if err1.Is(err3) {
		t.Error("Is() should return false for different error code")
	}
}

func TestNewf(t *testing.T) {
	code := ErrInvalidMessage
	err := Newf(code, "test %s %d", "message", 123)

	expected := "test message 123"
	if err.Message != expected {
		t.Errorf("Newf() message = %v, want %v", err.Message, expected)
	}
	if err.Code != code {
		t.Errorf("Newf() code = %v, want %v", err.Code, code)
	}
}

func TestWrapf(t *testing.T) {
	originalErr := errors.New("original")
	err := Wrapf(originalErr, ErrInternal, "wrapped %s", "error")

	if err.Message != "wrapped error" {
		t.Errorf("Wrapf() message = %v, want 'wrapped error'", err.Message)
	}
	if err.Cause != originalErr {
		t.Errorf("Wrapf() cause = %v, want %v", err.Cause, originalErr)
	}
}

func TestNotifyError_WithContext(t *testing.T) {
	err := New(ErrInvalidMessage, "test")
	result := err.WithContext("error context information")

	if result.Context != "error context information" {
		t.Error("WithContext() should set context")
	}
	if result != err {
		t.Error("WithContext() should return same error for chaining")
	}
}

func TestNotifyError_WithMetadata(t *testing.T) {
	err := New(ErrInvalidMessage, "test")
	result := err.WithMetadata("key1", "value1")
	result = result.WithMetadata("key2", 123)

	if len(result.Metadata) != 2 {
		t.Error("WithMetadata() should add metadata")
	}
	if result.Metadata["key1"] != "value1" {
		t.Error("WithMetadata() should set key1")
	}
	if result != err {
		t.Error("WithMetadata() should return same error for chaining")
	}
}

func TestNotifyError_WithPlatform(t *testing.T) {
	err := New(ErrPlatformUnavailable, "test")
	result := err.WithPlatform("email")

	if result.Platform != "email" {
		t.Errorf("WithPlatform() platform = %v, want 'email'", result.Platform)
	}
	if result != err {
		t.Error("WithPlatform() should return same error for chaining")
	}
}

func TestNotifyError_WithTarget(t *testing.T) {
	err := New(ErrInvalidTarget, "test")
	result := err.WithTarget("user@example.com")

	if result.Target != "user@example.com" {
		t.Errorf("WithTarget() target = %v, want 'user@example.com'", result.Target)
	}
	if result != err {
		t.Error("WithTarget() should return same error for chaining")
	}
}

func TestNotifyError_WithRequestID(t *testing.T) {
	err := New(ErrInternal, "test")
	result := err.WithRequestID("req-123")

	if result.RequestID != "req-123" {
		t.Errorf("WithRequestID() requestID = %v, want 'req-123'", result.RequestID)
	}
	if result != err {
		t.Error("WithRequestID() should return same error for chaining")
	}
}

func TestNotifyError_WithUserID(t *testing.T) {
	err := New(ErrInternal, "test")
	result := err.WithUserID("user-456")

	if result.UserID != "user-456" {
		t.Errorf("WithUserID() userID = %v, want 'user-456'", result.UserID)
	}
	if result != err {
		t.Error("WithUserID() should return same error for chaining")
	}
}

func TestNotifyError_GetRetryDelay(t *testing.T) {
	err := New(ErrRateLimitExceeded, "test")
	delay := time.Duration(30) * time.Second
	err.RetryAfter = &delay

	retryDelay := err.GetRetryDelay()
	expected := 30 * time.Second
	if retryDelay != expected {
		t.Errorf("GetRetryDelay() = %v, want %v", retryDelay, expected)
	}
}

func TestNewMultiError(t *testing.T) {
	multi := NewMultiError()

	if multi == nil {
		t.Fatal("NewMultiError() should not return nil")
	}
	if len(multi.Errors) != 0 {
		t.Error("NewMultiError() should have empty errors")
	}
}

func TestMultiError_Add(t *testing.T) {
	multi := NewMultiError()
	err1 := New(ErrInvalidConfig, "test1")
	err2 := New(ErrNetworkTimeout, "test2")

	multi.Add(err1)
	multi.Add(err2)

	if len(multi.Errors) != 2 {
		t.Errorf("Add() count = %d, want 2", len(multi.Errors))
	}
}

func TestMultiError_IsEmpty(t *testing.T) {
	multi := NewMultiError()

	if !multi.IsEmpty() {
		t.Error("IsEmpty() should return true for empty multi-error")
	}

	multi.Add(New(ErrInternal, "test"))
	if multi.IsEmpty() {
		t.Error("IsEmpty() should return false after adding error")
	}
}

func TestMultiError_ErrorOrNil(t *testing.T) {
	multi := NewMultiError()

	if multi.ErrorOrNil() != nil {
		t.Error("ErrorOrNil() should return nil for empty multi-error")
	}

	multi.Add(New(ErrInternal, "test"))
	if multi.ErrorOrNil() == nil {
		t.Error("ErrorOrNil() should not return nil after adding error")
	}
}

func TestMultiError_First(t *testing.T) {
	multi := NewMultiError()
	err1 := New(ErrInvalidConfig, "first")
	err2 := New(ErrNetworkTimeout, "second")

	multi.Add(err1)
	multi.Add(err2)

	first := multi.First()
	if first != err1 {
		t.Error("First() should return first error")
	}
}

func TestMultiError_Count(t *testing.T) {
	multi := NewMultiError()

	if multi.Count() != 0 {
		t.Error("Count() should return 0 for empty multi-error")
	}

	multi.Add(New(ErrInternal, "test1"))
	multi.Add(New(ErrInternal, "test2"))

	if multi.Count() != 2 {
		t.Errorf("Count() = %d, want 2", multi.Count())
	}
}

func TestNewConfigError(t *testing.T) {
	err := NewConfigError("invalid config")

	if err.Code != ErrInvalidConfig {
		t.Errorf("NewConfigError() code = %v, want %v", err.Code, ErrInvalidConfig)
	}
	if err.Message != "invalid config" {
		t.Errorf("NewConfigError() message = %v, want 'invalid config'", err.Message)
	}
}

func TestNewPlatformError(t *testing.T) {
	err := NewPlatformError("email", "send failed")

	if err.Code != ErrPlatformError {
		t.Errorf("NewPlatformError() code = %v, want %v", err.Code, ErrPlatformError)
	}
	if err.Platform != "email" {
		t.Errorf("NewPlatformError() platform = %v, want 'email'", err.Platform)
	}
	if err.Message != "send failed" {
		t.Errorf("NewPlatformError() message = %v, want 'send failed'", err.Message)
	}
}

func TestNewTargetError(t *testing.T) {
	err := NewTargetError("user@example.com", "invalid target")

	if err.Code != ErrInvalidTarget {
		t.Errorf("NewTargetError() code = %v, want %v", err.Code, ErrInvalidTarget)
	}
	if err.Target != "user@example.com" {
		t.Errorf("NewTargetError() target = %v, want 'user@example.com'", err.Target)
	}
}

func TestNewNetworkError(t *testing.T) {
	err := NewNetworkError("connection failed")

	if err.Code != ErrConnectionFailed {
		t.Errorf("NewNetworkError() code = %v, want %v", err.Code, ErrConnectionFailed)
	}
	if !err.Retryable {
		t.Error("NewNetworkError() should be retryable")
	}
}

func TestNewTimeoutError(t *testing.T) {
	err := NewTimeoutError("operation timed out")

	if err.Code != ErrNetworkTimeout {
		t.Errorf("NewTimeoutError() code = %v, want %v", err.Code, ErrNetworkTimeout)
	}
	if !err.Retryable {
		t.Error("NewTimeoutError() should be retryable")
	}
}

func TestNewRateLimitError(t *testing.T) {
	retryDuration := 60 * time.Second
	err := NewRateLimitError(retryDuration)

	if err.Code != ErrRateLimitExceeded {
		t.Errorf("NewRateLimitError() code = %v, want %v", err.Code, ErrRateLimitExceeded)
	}
	if err.RetryAfter == nil || *err.RetryAfter != retryDuration {
		t.Errorf("NewRateLimitError() retryAfter = %v, want %v", err.RetryAfter, retryDuration)
	}
	if !err.Retryable {
		t.Error("NewRateLimitError() should be retryable")
	}
}

func TestNewInternalError(t *testing.T) {
	err := NewInternalError("internal failure")

	if err.Code != ErrInternal {
		t.Errorf("NewInternalError() code = %v, want %v", err.Code, ErrInternal)
	}
	if err.Message != "internal failure" {
		t.Errorf("NewInternalError() message = %v, want 'internal failure'", err.Message)
	}
}

func TestIsTemporaryError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "network timeout (temporary)",
			err:      New(ErrNetworkTimeout, "test"),
			expected: true,
		},
		{
			name:     "rate limit (temporary)",
			err:      New(ErrRateLimitExceeded, "test"),
			expected: true,
		},
		{
			name:     "invalid config (not temporary)",
			err:      New(ErrInvalidConfig, "test"),
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTemporaryError(tt.err); got != tt.expected {
				t.Errorf("IsTemporaryError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "notify error",
			err:      New(ErrInvalidConfig, "config error"),
			expected: "config error",
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: "standard error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorMessage(tt.err); got != tt.expected {
				t.Errorf("GetErrorMessage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorPlatform(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "error with platform",
			err:      NewPlatformError("email", "failed"),
			expected: "email",
		},
		{
			name:     "error without platform",
			err:      New(ErrInvalidConfig, "test"),
			expected: "",
		},
		{
			name:     "standard error",
			err:      errors.New("test"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorPlatform(tt.err); got != tt.expected {
				t.Errorf("GetErrorPlatform() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetPriority(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected int
	}{
		{
			name:     "high priority",
			code:     ErrMissingPlatform,
			expected: 3, // PriorityHigh
		},
		{
			name:     "normal priority",
			code:     ErrNetworkTimeout,
			expected: 2, // PriorityNormal
		},
		{
			name:     "low priority",
			code:     ErrThrottled,
			expected: 1, // PriorityLow
		},
		{
			name:     "unknown code defaults to normal priority",
			code:     "UNKNOWN",
			expected: 2, // PriorityNormal (default for unknown codes)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPriority(tt.code); got != tt.expected {
				t.Errorf("GetPriority() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetAllErrorCodes(t *testing.T) {
	codes := GetAllErrorCodes()

	if len(codes) == 0 {
		t.Error("GetAllErrorCodes() should return non-empty slice")
	}

	// Check that some expected codes are present
	found := false
	for _, code := range codes {
		if code == ErrInvalidConfig {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetAllErrorCodes() should include ErrInvalidConfig")
	}
}

func TestGetErrorCodesByCategory(t *testing.T) {
	codes := GetErrorCodesByCategory("network")

	if len(codes) == 0 {
		t.Error("GetErrorCodesByCategory('network') should return non-empty slice")
	}

	// Check that network codes are present
	found := false
	for _, code := range codes {
		if code == ErrNetworkTimeout {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetErrorCodesByCategory('network') should include ErrNetworkTimeout")
	}
}
