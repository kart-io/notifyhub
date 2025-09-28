// Package errors provides unified error handling for NotifyHub
package errors

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Code represents an error code for categorization
type Code string

// NotifyError represents a unified error with code, message, and context
type NotifyError struct {
	Code       Code                   `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	Platform   string                 `json:"platform,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	StackTrace []string               `json:"stack_trace,omitempty"`
	Cause      error                  `json:"-"` // Original error cause
}

// Error implements the error interface
func (e *NotifyError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *NotifyError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error type or code
func (e *NotifyError) Is(target error) bool {
	if notifyErr, ok := target.(*NotifyError); ok {
		return e.Code == notifyErr.Code
	}
	return false
}

// WithContext adds context information to the error
func (e *NotifyError) WithContext(key string, value interface{}) *NotifyError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithDetails adds details to the error
func (e *NotifyError) WithDetails(details string) *NotifyError {
	e.Details = details
	return e
}

// WithCause sets the underlying cause error
func (e *NotifyError) WithCause(cause error) *NotifyError {
	e.Cause = cause
	return e
}

// ErrorBuilder helps construct NotifyError instances
type ErrorBuilder struct {
	code       Code
	message    string
	details    string
	context    map[string]interface{}
	cause      error
	stackTrace []string
}

// NewErrorBuilder creates a new error builder
func NewErrorBuilder(code Code, message string) *ErrorBuilder {
	return &ErrorBuilder{
		code:    code,
		message: message,
		context: make(map[string]interface{}),
	}
}

// WithDetails adds details to the error being built
func (b *ErrorBuilder) WithDetails(details string) *ErrorBuilder {
	b.details = details
	return b
}

// WithContext adds context information
func (b *ErrorBuilder) WithContext(key string, value interface{}) *ErrorBuilder {
	if b.context == nil {
		b.context = make(map[string]interface{})
	}
	b.context[key] = value
	return b
}

// WithCause sets the underlying cause error
func (b *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	b.cause = cause
	return b
}

// WithStackTrace adds stack trace information
func (b *ErrorBuilder) WithStackTrace(trace []string) *ErrorBuilder {
	b.stackTrace = trace
	return b
}

// Build creates the NotifyError
func (b *ErrorBuilder) Build() *NotifyError {
	return &NotifyError{
		Code:       b.code,
		Message:    b.message,
		Details:    b.details,
		Context:    b.context,
		Timestamp:  time.Now(),
		StackTrace: b.stackTrace,
		Cause:      b.cause,
	}
}

// New creates a new NotifyError
func New(code Code, message string) *NotifyError {
	return &NotifyError{
		Code:      code,
		Message:   message,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// Wrap wraps an existing error with a NotifyError
func Wrap(cause error, code Code, message string) *NotifyError {
	return &NotifyError{
		Code:      code,
		Message:   message,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
		Cause:     cause,
	}
}

// Wrapf wraps an existing error with a formatted message
func Wrapf(cause error, code Code, format string, args ...interface{}) *NotifyError {
	return &NotifyError{
		Code:      code,
		Message:   fmt.Sprintf(format, args...),
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
		Cause:     cause,
	}
}

// ErrorAggregator collects and aggregates errors from multiple operations
type ErrorAggregator struct {
	errors []error
	mu     sync.Mutex
}

// NewErrorAggregator creates a new error aggregator
func NewErrorAggregator() *ErrorAggregator {
	return &ErrorAggregator{
		errors: make([]error, 0),
	}
}

// Add adds an error to the aggregator
func (a *ErrorAggregator) Add(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		a.errors = append(a.errors, err)
	}
}

// HasErrors returns true if there are any aggregated errors
func (a *ErrorAggregator) HasErrors() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.errors) > 0
}

// Count returns the number of aggregated errors
func (a *ErrorAggregator) Count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.errors)
}

// Errors returns all aggregated errors
func (a *ErrorAggregator) Errors() []error {
	a.mu.Lock()
	defer a.mu.Unlock()
	errors := make([]error, len(a.errors))
	copy(errors, a.errors)
	return errors
}

// ToError converts the aggregated errors into a single NotifyError
func (a *ErrorAggregator) ToError() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.errors) == 0 {
		return nil
	}

	if len(a.errors) == 1 {
		return a.errors[0]
	}

	// Create aggregated error with context about all failures
	errorDetails := make([]string, len(a.errors))
	errorCodes := make(map[Code]int)
	platforms := make(map[string]int)

	for i, err := range a.errors {
		errorDetails[i] = err.Error()

		// Collect error code statistics
		if notifyErr, ok := err.(*NotifyError); ok {
			errorCodes[notifyErr.Code]++
			if platform, exists := notifyErr.Context["platform"]; exists {
				if platformStr, ok := platform.(string); ok {
					platforms[platformStr]++
				}
			}
		}
	}

	// Create aggregated error
	notifyErr := New(ErrInternalError, fmt.Sprintf("multiple errors occurred (%d failures)", len(a.errors)))
	notifyErr.WithContext("error_count", len(a.errors))
	notifyErr.WithContext("error_details", errorDetails)
	notifyErr.WithContext("error_codes", errorCodes)
	notifyErr.WithContext("affected_platforms", platforms)

	return notifyErr
}

// ErrorFormatter provides different formatting options for errors
type ErrorFormatter struct{}

// FormatForLogging formats an error for logging with structured information
func (f *ErrorFormatter) FormatForLogging(err error) map[string]interface{} {
	result := map[string]interface{}{
		"error_message": err.Error(),
		"error_type":    fmt.Sprintf("%T", err),
	}

	if notifyErr, ok := err.(*NotifyError); ok {
		result["error_code"] = string(notifyErr.Code)
		result["timestamp"] = notifyErr.Timestamp

		if notifyErr.Details != "" {
			result["error_details"] = notifyErr.Details
		}

		if notifyErr.Context != nil && len(notifyErr.Context) > 0 {
			result["context"] = notifyErr.Context
		}

		if notifyErr.StackTrace != nil && len(notifyErr.StackTrace) > 0 {
			result["stack_trace"] = notifyErr.StackTrace
		}

		// Add error classification
		info := GetErrorInfo(notifyErr.Code)
		result["category"] = info.Category
		result["severity"] = info.Severity
		result["retryable"] = info.Retryable
	}

	return result
}

// FormatForAPI formats an error for API responses (without sensitive info)
func (f *ErrorFormatter) FormatForAPI(err error) map[string]interface{} {
	result := map[string]interface{}{
		"message": "An error occurred",
	}

	if notifyErr, ok := err.(*NotifyError); ok {
		result["code"] = string(notifyErr.Code)
		result["message"] = notifyErr.Message

		if notifyErr.Details != "" {
			result["details"] = notifyErr.Details
		}

		// Only include safe context fields for API responses
		if notifyErr.Context != nil {
			safeContext := make(map[string]interface{})
			// Include only non-sensitive context fields
			for key, value := range notifyErr.Context {
				switch key {
				case "platform", "category", "target_type", "message_id":
					safeContext[key] = value
				}
			}
			if len(safeContext) > 0 {
				result["context"] = safeContext
			}
		}

		// Add error classification
		info := GetErrorInfo(notifyErr.Code)
		result["category"] = info.Category
		result["retryable"] = info.Retryable
	}

	return result
}

// FormatForDebug formats an error for debugging with full information
func (f *ErrorFormatter) FormatForDebug(err error) map[string]interface{} {
	result := f.FormatForLogging(err)

	// Add additional debug information
	if notifyErr, ok := err.(*NotifyError); ok {
		if notifyErr.Cause != nil {
			result["cause"] = notifyErr.Cause.Error()
			result["cause_type"] = fmt.Sprintf("%T", notifyErr.Cause)
		}
	}

	return result
}

// ErrorSerializer provides JSON serialization for errors
type ErrorSerializer struct {
	formatter *ErrorFormatter
}

// NewErrorSerializer creates a new error serializer
func NewErrorSerializer() *ErrorSerializer {
	return &ErrorSerializer{
		formatter: &ErrorFormatter{},
	}
}

// SerializeForLogging serializes an error for logging
func (s *ErrorSerializer) SerializeForLogging(err error) ([]byte, error) {
	data := s.formatter.FormatForLogging(err)
	return json.Marshal(data)
}

// SerializeForAPI serializes an error for API responses
func (s *ErrorSerializer) SerializeForAPI(err error) ([]byte, error) {
	data := s.formatter.FormatForAPI(err)
	return json.Marshal(data)
}

// SerializeForDebug serializes an error for debugging
func (s *ErrorSerializer) SerializeForDebug(err error) ([]byte, error) {
	data := s.formatter.FormatForDebug(err)
	return json.Marshal(data)
}
