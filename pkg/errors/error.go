// Package errors provides error types for NotifyHub
package errors

import (
	"encoding/json"
	"fmt"
	"time"
)

// NotifyError represents a NotifyHub error with structured information
type NotifyError struct {
	// Core error information
	Code     ErrorCode              `json:"code"`
	Message  string                 `json:"message"`
	Platform string                 `json:"platform,omitempty"`
	Target   string                 `json:"target,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Context information
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`

	// Error hierarchy
	Cause   error  `json:"-"`                     // Original error (not serialized)
	Context string `json:"context,omitempty"`     // Additional context
	Stack   string `json:"stack_trace,omitempty"` // Stack trace for debugging

	// Retry information
	Retryable    bool           `json:"retryable"`
	RetryAfter   *time.Duration `json:"retry_after,omitempty"`
	AttemptCount int            `json:"attempt_count,omitempty"`
}

// Error implements the error interface
func (e *NotifyError) Error() string {
	if e.Platform != "" && e.Target != "" {
		return fmt.Sprintf("%s: %s (platform: %s, target: %s)", e.Code, e.Message, e.Platform, e.Target)
	}
	if e.Platform != "" {
		return fmt.Sprintf("%s: %s (platform: %s)", e.Code, e.Message, e.Platform)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// String returns a string representation of the error
func (e *NotifyError) String() string {
	return e.Error()
}

// Unwrap returns the underlying cause error
func (e *NotifyError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error
func (e *NotifyError) Is(target error) bool {
	if targetErr, ok := target.(*NotifyError); ok {
		return e.Code == targetErr.Code
	}
	return false
}

// MarshalJSON implements json.Marshaler
func (e *NotifyError) MarshalJSON() ([]byte, error) {
	type Alias NotifyError
	return json.Marshal(&struct {
		*Alias
		CauseMessage string `json:"cause_message,omitempty"`
	}{
		Alias:        (*Alias)(e),
		CauseMessage: e.getCauseMessage(),
	})
}

// getCauseMessage returns the cause error message if available
func (e *NotifyError) getCauseMessage() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return ""
}

// WithCause adds a cause error
func (e *NotifyError) WithCause(cause error) *NotifyError {
	e.Cause = cause
	return e
}

// WithContext adds context information
func (e *NotifyError) WithContext(context string) *NotifyError {
	e.Context = context
	return e
}

// WithMetadata adds metadata
func (e *NotifyError) WithMetadata(key string, value interface{}) *NotifyError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithPlatform sets the platform
func (e *NotifyError) WithPlatform(platform string) *NotifyError {
	e.Platform = platform
	return e
}

// WithTarget sets the target
func (e *NotifyError) WithTarget(target string) *NotifyError {
	e.Target = target
	return e
}

// WithRequestID sets the request ID
func (e *NotifyError) WithRequestID(requestID string) *NotifyError {
	e.RequestID = requestID
	return e
}

// WithUserID sets the user ID
func (e *NotifyError) WithUserID(userID string) *NotifyError {
	e.UserID = userID
	return e
}

// WithRetryAfter sets the retry delay
func (e *NotifyError) WithRetryAfter(delay time.Duration) *NotifyError {
	e.RetryAfter = &delay
	return e
}

// WithAttemptCount sets the attempt count
func (e *NotifyError) WithAttemptCount(count int) *NotifyError {
	e.AttemptCount = count
	return e
}

// IsRetryable returns whether the error is retryable
func (e *NotifyError) IsRetryable() bool {
	if e.Retryable {
		return true
	}
	return IsRetryable(e.Code)
}

// GetRetryDelay returns the recommended retry delay
func (e *NotifyError) GetRetryDelay() time.Duration {
	if e.RetryAfter != nil {
		return *e.RetryAfter
	}
	return 0
}

// GetSeverity returns the error severity based on code priority
func (e *NotifyError) GetSeverity() string {
	priority := GetPriority(e.Code)
	switch priority {
	case PriorityCritical:
		return "critical"
	case PriorityHigh:
		return "high"
	case PriorityNormal:
		return "normal"
	case PriorityLow:
		return "low"
	default:
		return "unknown"
	}
}

// ToMap converts the error to a map representation
func (e *NotifyError) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"code":      string(e.Code),
		"message":   e.Message,
		"timestamp": e.Timestamp,
		"retryable": e.IsRetryable(),
		"severity":  e.GetSeverity(),
	}

	if e.Platform != "" {
		result["platform"] = e.Platform
	}
	if e.Target != "" {
		result["target"] = e.Target
	}
	if e.RequestID != "" {
		result["request_id"] = e.RequestID
	}
	if e.UserID != "" {
		result["user_id"] = e.UserID
	}
	if e.Context != "" {
		result["context"] = e.Context
	}
	if e.Cause != nil {
		result["cause"] = e.Cause.Error()
	}
	if e.Metadata != nil {
		result["metadata"] = e.Metadata
	}
	if e.RetryAfter != nil {
		result["retry_after"] = e.RetryAfter.String()
	}
	if e.AttemptCount > 0 {
		result["attempt_count"] = e.AttemptCount
	}

	return result
}

// MultiError represents multiple errors that occurred
type MultiError struct {
	Errors []error `json:"errors"`
}

// Error implements the error interface
func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("multiple errors occurred (%d errors)", len(e.Errors))
}

// Add adds an error to the multi-error
func (e *MultiError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// IsEmpty returns true if no errors are present
func (e *MultiError) IsEmpty() bool {
	return len(e.Errors) == 0
}

// ErrorOrNil returns the multi-error if it contains errors, otherwise nil
func (e *MultiError) ErrorOrNil() error {
	if e.IsEmpty() {
		return nil
	}
	return e
}

// First returns the first error, or nil if none
func (e *MultiError) First() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}

// Count returns the number of errors
func (e *MultiError) Count() int {
	return len(e.Errors)
}

// Constructor functions

// New creates a new NotifyError
func New(code ErrorCode, message string) *NotifyError {
	return &NotifyError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Retryable: IsRetryable(code),
	}
}

// Newf creates a new NotifyError with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *NotifyError {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap wraps an existing error with a NotifyError
func Wrap(err error, code ErrorCode, message string) *NotifyError {
	return New(code, message).WithCause(err)
}

// Wrapf wraps an existing error with a NotifyError and formatted message
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *NotifyError {
	return Wrap(err, code, fmt.Sprintf(format, args...))
}

// NewMultiError creates a new MultiError
func NewMultiError() *MultiError {
	return &MultiError{
		Errors: make([]error, 0),
	}
}

// Convenience constructors for common error types

// NewConfigError creates a configuration error
func NewConfigError(message string) *NotifyError {
	return New(ErrInvalidConfig, message)
}

// NewPlatformError creates a platform error
func NewPlatformError(platform, message string) *NotifyError {
	return New(ErrPlatformError, message).WithPlatform(platform)
}

// NewTargetError creates a target error
func NewTargetError(target, message string) *NotifyError {
	return New(ErrInvalidTarget, message).WithTarget(target)
}

// NewNetworkError creates a network error
func NewNetworkError(message string) *NotifyError {
	return New(ErrConnectionFailed, message)
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(message string) *NotifyError {
	return New(ErrNetworkTimeout, message).WithRetryAfter(5 * time.Second)
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(retryAfter time.Duration) *NotifyError {
	return New(ErrRateLimitExceeded, "rate limit exceeded").WithRetryAfter(retryAfter)
}

// NewInternalError creates an internal error
func NewInternalError(message string) *NotifyError {
	return New(ErrInternal, message)
}

// Error classification functions

// IsConfigError checks if error is a configuration error
func IsConfigError(err error) bool {
	if notifyErr, ok := err.(*NotifyError); ok {
		return GetCategory(notifyErr.Code) == "configuration"
	}
	return false
}

// IsPlatformError checks if error is a platform error
func IsPlatformError(err error) bool {
	if notifyErr, ok := err.(*NotifyError); ok {
		return GetCategory(notifyErr.Code) == "platform"
	}
	return false
}

// IsNetworkError checks if error is a network error
func IsNetworkError(err error) bool {
	if notifyErr, ok := err.(*NotifyError); ok {
		return GetCategory(notifyErr.Code) == "network"
	}
	return false
}

// IsRetryableError checks if error is retryable
func IsRetryableError(err error) bool {
	if notifyErr, ok := err.(*NotifyError); ok {
		return notifyErr.IsRetryable()
	}
	return false
}

// IsTemporaryError checks if error is temporary
func IsTemporaryError(err error) bool {
	if notifyErr, ok := err.(*NotifyError); ok {
		return notifyErr.IsRetryable() && (notifyErr.Code == ErrPlatformUnavailable ||
			notifyErr.Code == ErrNetworkTimeout ||
			notifyErr.Code == ErrConnectionFailed ||
			notifyErr.Code == ErrRateLimitExceeded)
	}
	return false
}

// Error extraction functions

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	if notifyErr, ok := err.(*NotifyError); ok {
		return notifyErr.Code
	}
	return ErrInternal
}

// GetErrorMessage extracts the error message from an error
func GetErrorMessage(err error) string {
	if notifyErr, ok := err.(*NotifyError); ok {
		return notifyErr.Message
	}
	return err.Error()
}

// GetErrorPlatform extracts the platform from an error
func GetErrorPlatform(err error) string {
	if notifyErr, ok := err.(*NotifyError); ok {
		return notifyErr.Platform
	}
	return ""
}
