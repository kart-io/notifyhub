package errors

// Standard error definitions that replace scattered error definitions across packages

// Configuration errors
var (
	ErrInvalidConfig   = New(CodeInvalidConfig, CategoryConfig, "invalid configuration")
	ErrMissingConfig   = New(CodeMissingConfig, CategoryConfig, "missing required configuration")
	ErrInvalidPlatform = New(CodeInvalidPlatform, CategoryConfig, "invalid or unsupported platform")
)

// Message validation errors
var (
	ErrEmptyMessage    = New(CodeEmptyMessage, CategoryValidation, "message must have either title or body")
	ErrInvalidFormat   = New(CodeInvalidFormat, CategoryValidation, "invalid message format")
	ErrInvalidPriority = New(CodeInvalidPriority, CategoryValidation, "priority must be between 1 and 5")
	ErrTemplateError   = New(CodeTemplateError, CategoryValidation, "template rendering failed")
)

// Target validation errors
var (
	ErrInvalidTarget     = New(CodeInvalidTarget, CategoryValidation, "invalid target type or format")
	ErrEmptyTarget       = New(CodeEmptyTarget, CategoryValidation, "target value cannot be empty")
	ErrUnsupportedTarget = New(CodeUnsupportedTarget, CategoryValidation, "unsupported target type for platform")
)

// Network and transport errors
var (
	ErrNetworkError = New(CodeNetworkError, CategoryNetwork, "network communication failed")
	ErrTimeout      = New(CodeTimeout, CategoryNetwork, "request timeout")
	ErrRateLimited  = New(CodeRateLimited, CategoryRateLimit, "rate limit exceeded")
	ErrServerError  = New(CodeServerError, CategoryNetwork, "server error")
)

// Authentication errors
var (
	ErrInvalidCredentials = New(CodeInvalidCredentials, CategoryAuth, "invalid credentials")
	ErrUnauthorized       = New(CodeUnauthorized, CategoryAuth, "authentication required")
	ErrForbidden          = New(CodeForbidden, CategoryAuth, "access forbidden")
	ErrPermissionDenied   = New(CodePermissionDenied, CategoryAuth, "permission denied")
)

// General sending errors
var (
	ErrSendingFailed    = New(CodeSendingFailed, CategoryTransport, "message sending failed")
	ErrProcessingFailed = New(CodeProcessingFailed, CategoryInternal, "message processing failed")
)

// Legacy error compatibility - these will map to new standard errors
// This ensures backward compatibility while migration happens
var (
	// From core/message/errors.go
	ErrMissingTemplate = ErrTemplateError

	// From core/sending/errors.go
	ErrInvalidTargetType   = ErrInvalidTarget
	ErrEmptyTargetValue    = ErrEmptyTarget
	ErrEmptyPlatform       = ErrInvalidPlatform
	ErrUnsupportedPlatform = ErrInvalidPlatform
)

// Platform-specific error helpers

// NewFeishuError creates a Feishu-specific error
func NewFeishuError(code ErrorCode, message string) *NotifyError {
	return NewWithPlatform(code, CategoryPlatform, message, "feishu")
}

// NewEmailError creates an Email-specific error
func NewEmailError(code ErrorCode, message string) *NotifyError {
	return NewWithPlatform(code, CategoryPlatform, message, "email")
}

// NewSMSError creates an SMS-specific error
func NewSMSError(code ErrorCode, message string) *NotifyError {
	return NewWithPlatform(code, CategoryPlatform, message, "sms")
}

// Helper functions for common error scenarios

// IsConfigurationError checks if error is configuration-related
func IsConfigurationError(err error) bool {
	if nerr, ok := err.(*NotifyError); ok {
		return nerr.Category == CategoryConfig
	}
	return false
}

// IsValidationError checks if error is validation-related
func IsValidationError(err error) bool {
	if nerr, ok := err.(*NotifyError); ok {
		return nerr.Category == CategoryValidation
	}
	return false
}

// IsNetworkError checks if error is network-related
func IsNetworkError(err error) bool {
	if nerr, ok := err.(*NotifyError); ok {
		return nerr.Category == CategoryNetwork
	}
	return false
}

// IsAuthError checks if error is authentication-related
func IsAuthError(err error) bool {
	if nerr, ok := err.(*NotifyError); ok {
		return nerr.Category == CategoryAuth
	}
	return false
}

// IsRateLimitError checks if error is rate limit-related
func IsRateLimitError(err error) bool {
	if nerr, ok := err.(*NotifyError); ok {
		return nerr.Category == CategoryRateLimit || nerr.Code == CodeRateLimited
	}
	return false
}

// IsRetryableError checks if error is retryable
func IsRetryableError(err error) bool {
	if nerr, ok := err.(*NotifyError); ok {
		return nerr.IsRetryable()
	}
	return false
}

// IsTemporaryError checks if error is temporary
func IsTemporaryError(err error) bool {
	if nerr, ok := err.(*NotifyError); ok {
		return nerr.IsTemporary()
	}
	return false
}
