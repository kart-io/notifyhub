package errors

import (
	"fmt"
	"net/http"
	"strings"
)

// ErrorCode represents a standardized error code
type ErrorCode string

const (
	// Configuration errors
	CodeInvalidConfig   ErrorCode = "INVALID_CONFIG"
	CodeMissingConfig   ErrorCode = "MISSING_CONFIG"
	CodeInvalidPlatform ErrorCode = "INVALID_PLATFORM"

	// Message errors
	CodeEmptyMessage    ErrorCode = "EMPTY_MESSAGE"
	CodeInvalidFormat   ErrorCode = "INVALID_FORMAT"
	CodeInvalidPriority ErrorCode = "INVALID_PRIORITY"
	CodeTemplateError   ErrorCode = "TEMPLATE_ERROR"

	// Target errors
	CodeInvalidTarget     ErrorCode = "INVALID_TARGET"
	CodeEmptyTarget       ErrorCode = "EMPTY_TARGET"
	CodeUnsupportedTarget ErrorCode = "UNSUPPORTED_TARGET"

	// Network and transport errors
	CodeNetworkError ErrorCode = "NETWORK_ERROR"
	CodeTimeout      ErrorCode = "TIMEOUT"
	CodeRateLimited  ErrorCode = "RATE_LIMITED"
	CodeUnauthorized ErrorCode = "UNAUTHORIZED"
	CodeForbidden    ErrorCode = "FORBIDDEN"
	CodeNotFound     ErrorCode = "NOT_FOUND"
	CodeServerError  ErrorCode = "SERVER_ERROR"

	// Authentication errors
	CodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	CodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	CodePermissionDenied   ErrorCode = "PERMISSION_DENIED"

	// General errors
	CodeSendingFailed    ErrorCode = "SENDING_FAILED"
	CodeProcessingFailed ErrorCode = "PROCESSING_FAILED"
	CodeUnknownError     ErrorCode = "UNKNOWN_ERROR"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	CategoryConfig     ErrorCategory = "CONFIG"
	CategoryValidation ErrorCategory = "VALIDATION"
	CategoryNetwork    ErrorCategory = "NETWORK"
	CategoryAuth       ErrorCategory = "AUTH"
	CategoryRateLimit  ErrorCategory = "RATE_LIMIT"
	CategoryPlatform   ErrorCategory = "PLATFORM"
	CategoryTransport  ErrorCategory = "TRANSPORT"
	CategoryInternal   ErrorCategory = "INTERNAL"
)

// NotifyError represents a standardized error with category and code
type NotifyError struct {
	Code     ErrorCode     `json:"code"`
	Category ErrorCategory `json:"category"`
	Message  string        `json:"message"`
	Platform string        `json:"platform,omitempty"`
	Cause    error         `json:"-"`
}

// Error implements the error interface
func (e *NotifyError) Error() string {
	if e.Platform != "" {
		return fmt.Sprintf("[%s:%s] %s (platform: %s)", e.Category, e.Code, e.Message, e.Platform)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Category, e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *NotifyError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison
func (e *NotifyError) Is(target error) bool {
	if t, ok := target.(*NotifyError); ok {
		return e.Code == t.Code && e.Category == t.Category
	}
	return false
}

// IsRetryable returns true if the error indicates a retryable condition
func (e *NotifyError) IsRetryable() bool {
	switch e.Code {
	case CodeNetworkError, CodeTimeout, CodeRateLimited, CodeServerError:
		return true
	default:
		return false
	}
}

// IsTemporary returns true if the error is temporary
func (e *NotifyError) IsTemporary() bool {
	return e.IsRetryable()
}

// HTTPStatusCode returns the corresponding HTTP status code
func (e *NotifyError) HTTPStatusCode() int {
	switch e.Code {
	case CodeInvalidConfig, CodeInvalidFormat, CodeInvalidTarget, CodeEmptyTarget, CodeEmptyMessage:
		return http.StatusBadRequest
	case CodeUnauthorized, CodeInvalidCredentials:
		return http.StatusUnauthorized
	case CodeForbidden, CodePermissionDenied:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeTimeout:
		return http.StatusRequestTimeout
	case CodeServerError, CodeSendingFailed, CodeProcessingFailed:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// New creates a new NotifyError
func New(code ErrorCode, category ErrorCategory, message string) *NotifyError {
	return &NotifyError{
		Code:     code,
		Category: category,
		Message:  message,
	}
}

// NewWithPlatform creates a new NotifyError with platform information
func NewWithPlatform(code ErrorCode, category ErrorCategory, message, platform string) *NotifyError {
	return &NotifyError{
		Code:     code,
		Category: category,
		Message:  message,
		Platform: platform,
	}
}

// Wrap wraps an existing error with NotifyError
func Wrap(code ErrorCode, category ErrorCategory, message string, cause error) *NotifyError {
	return &NotifyError{
		Code:     code,
		Category: category,
		Message:  message,
		Cause:    cause,
	}
}

// WrapWithPlatform wraps an existing error with NotifyError and platform info
func WrapWithPlatform(code ErrorCode, category ErrorCategory, message, platform string, cause error) *NotifyError {
	return &NotifyError{
		Code:     code,
		Category: category,
		Message:  message,
		Platform: platform,
		Cause:    cause,
	}
}

// MapHTTPError maps HTTP status codes to NotifyError
func MapHTTPError(statusCode int, body string, platform string) *NotifyError {
	var code ErrorCode
	var category ErrorCategory
	var message string

	switch {
	case statusCode == 401:
		code = CodeUnauthorized
		category = CategoryAuth
		message = "Authentication required"
	case statusCode == 403:
		code = CodeForbidden
		category = CategoryAuth
		message = "Access forbidden"
	case statusCode == 404:
		code = CodeNotFound
		category = CategoryNetwork
		message = "Resource not found"
	case statusCode == 429:
		code = CodeRateLimited
		category = CategoryRateLimit
		message = "Rate limit exceeded"
	case statusCode >= 400 && statusCode < 500:
		code = CodeInvalidConfig
		category = CategoryValidation
		message = fmt.Sprintf("Client error: %d", statusCode)
	case statusCode >= 500:
		code = CodeServerError
		category = CategoryNetwork
		message = fmt.Sprintf("Server error: %d", statusCode)
	default:
		code = CodeNetworkError
		category = CategoryNetwork
		message = fmt.Sprintf("HTTP error: %d", statusCode)
	}

	// Include response body if available and not too long
	if body != "" && len(body) < 200 {
		message += fmt.Sprintf(" - %s", strings.TrimSpace(body))
	}

	return NewWithPlatform(code, category, message, platform)
}

// MapNetworkError maps network errors to NotifyError
func MapNetworkError(err error, platform string) *NotifyError {
	if err == nil {
		return nil
	}

	// Check for timeout errors
	if isTimeoutError(err) {
		return WrapWithPlatform(CodeTimeout, CategoryNetwork, "Request timeout", platform, err)
	}

	// Check for connection errors
	if isConnectionError(err) {
		return WrapWithPlatform(CodeNetworkError, CategoryNetwork, "Connection failed", platform, err)
	}

	// Default network error
	return WrapWithPlatform(CodeNetworkError, CategoryNetwork, "Network error", platform, err)
}

// MapSMTPError maps SMTP errors to NotifyError
func MapSMTPError(err error) *NotifyError {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Common SMTP error patterns
	switch {
	case strings.Contains(errStr, "authentication failed") || strings.Contains(errStr, "535"):
		return WrapWithPlatform(CodeInvalidCredentials, CategoryAuth, "SMTP authentication failed", "email", err)
	case strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "421"):
		return WrapWithPlatform(CodeRateLimited, CategoryRateLimit, "SMTP rate limit exceeded", "email", err)
	case strings.Contains(errStr, "timeout"):
		return WrapWithPlatform(CodeTimeout, CategoryNetwork, "SMTP timeout", "email", err)
	case strings.Contains(errStr, "550") || strings.Contains(errStr, "invalid recipient"):
		return WrapWithPlatform(CodeInvalidTarget, CategoryValidation, "Invalid email recipient", "email", err)
	default:
		return WrapWithPlatform(CodeSendingFailed, CategoryTransport, "SMTP send failed", "email", err)
	}
}

// Helper functions for error detection
func isTimeoutError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "i/o timeout")
}

func isConnectionError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection closed") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "network unreachable")
}
