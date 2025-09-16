package client

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCategory represents different categories of errors
type ErrorCategory string

const (
	// ErrorCategoryConfig indicates configuration-related errors
	ErrorCategoryConfig ErrorCategory = "CONFIG"

	// ErrorCategoryNetwork indicates network-related errors
	ErrorCategoryNetwork ErrorCategory = "NETWORK"

	// ErrorCategoryValidation indicates validation errors
	ErrorCategoryValidation ErrorCategory = "VALIDATION"

	// ErrorCategoryPlatform indicates platform-specific errors
	ErrorCategoryPlatform ErrorCategory = "PLATFORM"

	// ErrorCategoryQueue indicates queue-related errors
	ErrorCategoryQueue ErrorCategory = "QUEUE"

	// ErrorCategoryTemplate indicates template-related errors
	ErrorCategoryTemplate ErrorCategory = "TEMPLATE"

	// ErrorCategoryTimeout indicates timeout errors
	ErrorCategoryTimeout ErrorCategory = "TIMEOUT"

	// ErrorCategoryRateLimit indicates rate limiting errors
	ErrorCategoryRateLimit ErrorCategory = "RATE_LIMIT"

	// ErrorCategoryAuth indicates authentication/authorization errors
	ErrorCategoryAuth ErrorCategory = "AUTH"

	// ErrorCategoryInternal indicates internal system errors
	ErrorCategoryInternal ErrorCategory = "INTERNAL"
)

// NotifyHubError represents an enhanced error with category and retry information
type NotifyHubError struct {
	Category    ErrorCategory `json:"category"`
	Code        string        `json:"code"`
	Message     string        `json:"message"`
	Details     string        `json:"details,omitempty"`
	Retryable   bool          `json:"retryable"`
	Suggestions []string      `json:"suggestions,omitempty"`
	Cause       error         `json:"-"`
	Platform    string        `json:"platform,omitempty"`
}

// Error implements the error interface
func (e *NotifyHubError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s:%s] %s - %s", e.Category, e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Category, e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *NotifyHubError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns whether the error suggests retrying
func (e *NotifyHubError) IsRetryable() bool {
	return e.Retryable
}

// GetSuggestions returns actionable suggestions for fixing the error
func (e *NotifyHubError) GetSuggestions() []string {
	return e.Suggestions
}

// String returns a detailed string representation
func (e *NotifyHubError) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Category: %s", e.Category))
	parts = append(parts, fmt.Sprintf("Code: %s", e.Code))
	parts = append(parts, fmt.Sprintf("Message: %s", e.Message))

	if e.Details != "" {
		parts = append(parts, fmt.Sprintf("Details: %s", e.Details))
	}

	if e.Platform != "" {
		parts = append(parts, fmt.Sprintf("Platform: %s", e.Platform))
	}

	parts = append(parts, fmt.Sprintf("Retryable: %t", e.Retryable))

	if len(e.Suggestions) > 0 {
		parts = append(parts, fmt.Sprintf("Suggestions: %s", strings.Join(e.Suggestions, "; ")))
	}

	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("Caused by: %s", e.Cause.Error()))
	}

	return strings.Join(parts, "\n")
}

// Error constructors for different categories

// NewConfigError creates a configuration error
func NewConfigError(code, message string, suggestions ...string) *NotifyHubError {
	return &NotifyHubError{
		Category:    ErrorCategoryConfig,
		Code:        code,
		Message:     message,
		Retryable:   false,
		Suggestions: suggestions,
	}
}

// NewNetworkError creates a network error
func NewNetworkError(code, message string, cause error, retryable bool) *NotifyHubError {
	suggestions := []string{}
	if retryable {
		suggestions = append(suggestions, "Check network connectivity and retry")
	}

	return &NotifyHubError{
		Category:    ErrorCategoryNetwork,
		Code:        code,
		Message:     message,
		Retryable:   retryable,
		Cause:       cause,
		Suggestions: suggestions,
	}
}

// NewValidationError creates a validation error
func NewValidationError(code, message, details string, suggestions ...string) *NotifyHubError {
	return &NotifyHubError{
		Category:    ErrorCategoryValidation,
		Code:        code,
		Message:     message,
		Details:     details,
		Retryable:   false,
		Suggestions: suggestions,
	}
}

// NewPlatformError creates a platform-specific error
func NewPlatformError(platform, code, message string, retryable bool, cause error) *NotifyHubError {
	suggestions := []string{}
	switch platform {
	case "feishu":
		suggestions = append(suggestions, "Check Feishu webhook URL and secret", "Verify group permissions")
	case "email":
		suggestions = append(suggestions, "Check SMTP configuration", "Verify email addresses")
	}

	return &NotifyHubError{
		Category:    ErrorCategoryPlatform,
		Platform:    platform,
		Code:        code,
		Message:     message,
		Retryable:   retryable,
		Cause:       cause,
		Suggestions: suggestions,
	}
}

// NewQueueError creates a queue-related error
func NewQueueError(code, message string, retryable bool, cause error) *NotifyHubError {
	suggestions := []string{"Check queue configuration and capacity"}
	if retryable {
		suggestions = append(suggestions, "Retry after reducing load")
	}

	return &NotifyHubError{
		Category:    ErrorCategoryQueue,
		Code:        code,
		Message:     message,
		Retryable:   retryable,
		Cause:       cause,
		Suggestions: suggestions,
	}
}

// NewTemplateError creates a template-related error
func NewTemplateError(code, message, details string) *NotifyHubError {
	return &NotifyHubError{
		Category:    ErrorCategoryTemplate,
		Code:        code,
		Message:     message,
		Details:     details,
		Retryable:   false,
		Suggestions: []string{"Check template syntax and variable names"},
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(code, message string, platform string) *NotifyHubError {
	suggestions := []string{"Increase timeout duration", "Check network latency"}
	if platform != "" {
		suggestions = append(suggestions, fmt.Sprintf("Check %s service status", platform))
	}

	return &NotifyHubError{
		Category:    ErrorCategoryTimeout,
		Platform:    platform,
		Code:        code,
		Message:     message,
		Retryable:   true,
		Suggestions: suggestions,
	}
}

// NewRateLimitError creates a rate limiting error
func NewRateLimitError(platform, message string) *NotifyHubError {
	return &NotifyHubError{
		Category:    ErrorCategoryRateLimit,
		Platform:    platform,
		Code:        "RATE_LIMITED",
		Message:     message,
		Retryable:   true,
		Suggestions: []string{"Wait before retrying", "Reduce sending frequency", "Consider batching messages"},
	}
}

// NewAuthError creates an authentication error
func NewAuthError(platform, code, message string) *NotifyHubError {
	suggestions := []string{"Check credentials and permissions"}
	if platform == "feishu" {
		suggestions = append(suggestions, "Verify webhook URL and secret")
	} else if platform == "email" {
		suggestions = append(suggestions, "Check SMTP username and password")
	}

	return &NotifyHubError{
		Category:    ErrorCategoryAuth,
		Platform:    platform,
		Code:        code,
		Message:     message,
		Retryable:   false,
		Suggestions: suggestions,
	}
}

// NewInternalError creates an internal system error
func NewInternalError(code, message string, cause error) *NotifyHubError {
	return &NotifyHubError{
		Category:    ErrorCategoryInternal,
		Code:        code,
		Message:     message,
		Retryable:   false,
		Cause:       cause,
		Suggestions: []string{"Contact system administrator", "Check logs for details"},
	}
}

// Helper functions for error type checking

// IsConfigError checks if an error is a configuration error
func IsConfigError(err error) bool {
	var nhErr *NotifyHubError
	return errors.As(err, &nhErr) && nhErr.Category == ErrorCategoryConfig
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	var nhErr *NotifyHubError
	return errors.As(err, &nhErr) && nhErr.Category == ErrorCategoryNetwork
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	var nhErr *NotifyHubError
	return errors.As(err, &nhErr) && nhErr.Category == ErrorCategoryValidation
}

// IsRetryableError checks if an error suggests retrying
func IsRetryableError(err error) bool {
	var nhErr *NotifyHubError
	if errors.As(err, &nhErr) {
		return nhErr.Retryable
	}
	return false
}

// GetErrorSuggestions extracts suggestions from an error
func GetErrorSuggestions(err error) []string {
	var nhErr *NotifyHubError
	if errors.As(err, &nhErr) {
		return nhErr.Suggestions
	}
	return nil
}

// WrapError wraps an existing error with NotifyHub error information
func WrapError(err error, category ErrorCategory, code, message string) *NotifyHubError {
	return &NotifyHubError{
		Category:  category,
		Code:      code,
		Message:   message,
		Retryable: false,
		Cause:     err,
	}
}

// ErrorCollector collects multiple errors and provides summary information
type ErrorCollector struct {
	errors []error
	categories map[ErrorCategory]int
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors:     make([]error, 0),
		categories: make(map[ErrorCategory]int),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	if err == nil {
		return
	}

	ec.errors = append(ec.errors, err)

	var nhErr *NotifyHubError
	if errors.As(err, &nhErr) {
		ec.categories[nhErr.Category]++
	} else {
		ec.categories[ErrorCategoryInternal]++
	}
}

// HasErrors returns whether any errors were collected
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Count returns the number of errors collected
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// Categories returns error count by category
func (ec *ErrorCollector) Categories() map[ErrorCategory]int {
	return ec.categories
}

// Summary returns a summary of collected errors
func (ec *ErrorCollector) Summary() string {
	if len(ec.errors) == 0 {
		return "No errors"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("Total errors: %d", len(ec.errors)))

	for category, count := range ec.categories {
		parts = append(parts, fmt.Sprintf("%s: %d", category, count))
	}

	return strings.Join(parts, ", ")
}

// FirstError returns the first error or nil if no errors
func (ec *ErrorCollector) FirstError() error {
	if len(ec.errors) == 0 {
		return nil
	}
	return ec.errors[0]
}

// LastError returns the last error or nil if no errors
func (ec *ErrorCollector) LastError() error {
	if len(ec.errors) == 0 {
		return nil
	}
	return ec.errors[len(ec.errors)-1]
}