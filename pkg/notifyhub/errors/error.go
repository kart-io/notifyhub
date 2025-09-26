// Package errors provides unified error handling for NotifyHub
package errors

import (
	"fmt"
	"time"
)

// Code represents an error code for categorization
type Code string

// NotifyError represents a unified error with code, message, and context
type NotifyError struct {
	Code       Code                   `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
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
