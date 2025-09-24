// Package errors provides error definitions for NotifyHub
package errors

import "fmt"

// NotifyError represents a NotifyHub-specific error
type NotifyError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Category   string `json:"category"`
	Retryable  bool   `json:"retryable"`
	Temporary  bool   `json:"temporary"`
	underlying error
}

// Error implements the error interface
func (e *NotifyError) Error() string {
	if e.underlying != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.underlying)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *NotifyError) Unwrap() error {
	return e.underlying
}
