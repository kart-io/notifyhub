// Package validation provides validation error types and utilities
package validation

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   interface{} `json:"value,omitempty"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "no validation errors"
	}

	if len(ve.Errors) == 1 {
		return ve.Errors[0].Error()
	}

	var messages []string
	for _, err := range ve.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("multiple validation errors: %s", strings.Join(messages, "; "))
}

// Add adds a validation error
func (ve *ValidationErrors) Add(field, rule, message string, value interface{}) {
	ve.Errors = append(ve.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

// Count returns the number of validation errors
func (ve *ValidationErrors) Count() int {
	return len(ve.Errors)
}

// Fields returns a list of fields that have errors
func (ve *ValidationErrors) Fields() []string {
	var fields []string
	for _, err := range ve.Errors {
		fields = append(fields, err.Field)
	}
	return fields
}

// FieldErrors returns errors for a specific field
func (ve *ValidationErrors) FieldErrors(field string) []ValidationError {
	var fieldErrors []ValidationError
	for _, err := range ve.Errors {
		if err.Field == field {
			fieldErrors = append(fieldErrors, err)
		}
	}
	return fieldErrors
}

// HasFieldError returns true if a specific field has errors
func (ve *ValidationErrors) HasFieldError(field string) bool {
	for _, err := range ve.Errors {
		if err.Field == field {
			return true
		}
	}
	return false
}

// Clear removes all validation errors
func (ve *ValidationErrors) Clear() {
	ve.Errors = nil
}

// ClearField removes validation errors for a specific field
func (ve *ValidationErrors) ClearField(field string) {
	var filteredErrors []ValidationError
	for _, err := range ve.Errors {
		if err.Field != field {
			filteredErrors = append(filteredErrors, err)
		}
	}
	ve.Errors = filteredErrors
}

// ToMap converts validation errors to a map of field -> error message
func (ve *ValidationErrors) ToMap() map[string]string {
	errorMap := make(map[string]string)
	for _, err := range ve.Errors {
		errorMap[err.Field] = err.Message
	}
	return errorMap
}

// ToDetailedMap converts validation errors to a detailed map with rule info
func (ve *ValidationErrors) ToDetailedMap() map[string]map[string]interface{} {
	errorMap := make(map[string]map[string]interface{})
	for _, err := range ve.Errors {
		errorMap[err.Field] = map[string]interface{}{
			"rule":    err.Rule,
			"message": err.Message,
			"value":   err.Value,
		}
	}
	return errorMap
}

// Merge merges another ValidationErrors into this one
func (ve *ValidationErrors) Merge(other *ValidationErrors) {
	if other != nil {
		ve.Errors = append(ve.Errors, other.Errors...)
	}
}

// Pre-defined error types

// ErrRequired represents a required field error
var ErrRequired = ValidationError{
	Rule:    "required",
	Message: "field is required",
}

// ErrInvalidFormat represents an invalid format error
var ErrInvalidFormat = ValidationError{
	Rule:    "format",
	Message: "invalid format",
}

// ErrOutOfRange represents an out of range error
var ErrOutOfRange = ValidationError{
	Rule:    "range",
	Message: "value out of range",
}

// ErrTooShort represents a too short error
var ErrTooShort = ValidationError{
	Rule:    "min_length",
	Message: "value too short",
}

// ErrTooLong represents a too long error
var ErrTooLong = ValidationError{
	Rule:    "max_length",
	Message: "value too long",
}

// ErrInvalidEmail represents an invalid email error
var ErrInvalidEmail = ValidationError{
	Rule:    "email",
	Message: "invalid email address",
}

// ErrInvalidURL represents an invalid URL error
var ErrInvalidURL = ValidationError{
	Rule:    "url",
	Message: "invalid URL",
}

// ErrInvalidPhone represents an invalid phone error
var ErrInvalidPhone = ValidationError{
	Rule:    "phone",
	Message: "invalid phone number",
}

// Error factory functions

// NewValidationError creates a new validation error
func NewValidationError(field, rule, message string, value interface{}) ValidationError {
	return ValidationError{
		Field:   field,
		Rule:    rule,
		Message: message,
		Value:   value,
	}
}

// NewValidationErrors creates a new ValidationErrors
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]ValidationError, 0),
	}
}

// NewValidationErrorsFromMap creates ValidationErrors from a map
func NewValidationErrorsFromMap(errorMap map[string]string) *ValidationErrors {
	ve := NewValidationErrors()
	for field, message := range errorMap {
		ve.Add(field, "unknown", message, nil)
	}
	return ve
}

// Error builders for common validation scenarios

// RequiredFieldError creates a required field error
func RequiredFieldError(field string) ValidationError {
	return ValidationError{
		Field:   field,
		Rule:    "required",
		Message: "field is required",
	}
}

// InvalidFormatError creates an invalid format error
func InvalidFormatError(field string, expectedFormat string, value interface{}) ValidationError {
	message := "invalid format"
	if expectedFormat != "" {
		message = fmt.Sprintf("invalid format, expected %s", expectedFormat)
	}
	return ValidationError{
		Field:   field,
		Rule:    "format",
		Message: message,
		Value:   value,
	}
}

// OutOfRangeError creates an out of range error
func OutOfRangeError(field string, min, max interface{}, value interface{}) ValidationError {
	message := fmt.Sprintf("value must be between %v and %v", min, max)
	return ValidationError{
		Field:   field,
		Rule:    "range",
		Message: message,
		Value:   value,
	}
}

// StringLengthError creates a string length error
func StringLengthError(field string, minLen, maxLen int, actualLen int) ValidationError {
	var message string
	if minLen > 0 && maxLen > 0 {
		message = fmt.Sprintf("length must be between %d and %d characters (got %d)", minLen, maxLen, actualLen)
	} else if minLen > 0 {
		message = fmt.Sprintf("must be at least %d characters (got %d)", minLen, actualLen)
	} else if maxLen > 0 {
		message = fmt.Sprintf("must be at most %d characters (got %d)", maxLen, actualLen)
	}

	return ValidationError{
		Field:   field,
		Rule:    "length",
		Message: message,
		Value:   actualLen,
	}
}

// InvalidChoiceError creates an invalid choice error
func InvalidChoiceError(field string, validChoices []interface{}, value interface{}) ValidationError {
	message := fmt.Sprintf("invalid choice, must be one of: %v", validChoices)
	return ValidationError{
		Field:   field,
		Rule:    "choice",
		Message: message,
		Value:   value,
	}
}

// Custom error types for NotifyHub

// PlatformConfigError represents a platform configuration error
type PlatformConfigError struct {
	Platform string `json:"platform"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}

// Error implements the error interface
func (e PlatformConfigError) Error() string {
	return fmt.Sprintf("platform '%s' config error in field '%s': %s", e.Platform, e.Field, e.Message)
}

// MessageValidationError represents a message validation error
type MessageValidationError struct {
	MessageID string            `json:"message_id"`
	Errors    *ValidationErrors `json:"errors"`
}

// Error implements the error interface
func (e MessageValidationError) Error() string {
	if e.Errors == nil || !e.Errors.HasErrors() {
		return fmt.Sprintf("message '%s' validation error", e.MessageID)
	}
	return fmt.Sprintf("message '%s' validation error: %s", e.MessageID, e.Errors.Error())
}

// TargetValidationError represents a target validation error
type TargetValidationError struct {
	TargetType string `json:"target_type"`
	TargetValue string `json:"target_value"`
	Message    string `json:"message"`
}

// Error implements the error interface
func (e TargetValidationError) Error() string {
	return fmt.Sprintf("target validation error for %s '%s': %s", e.TargetType, e.TargetValue, e.Message)
}

// Error aggregation utilities

// ErrorAggregator collects and aggregates validation errors
type ErrorAggregator struct {
	errors *ValidationErrors
}

// NewErrorAggregator creates a new error aggregator
func NewErrorAggregator() *ErrorAggregator {
	return &ErrorAggregator{
		errors: NewValidationErrors(),
	}
}

// AddError adds a validation error
func (ea *ErrorAggregator) AddError(field, rule, message string, value interface{}) {
	ea.errors.Add(field, rule, message, value)
}

// AddFieldError adds a field-specific error
func (ea *ErrorAggregator) AddFieldError(field, message string) {
	ea.errors.Add(field, "validation", message, nil)
}

// AddRequiredError adds a required field error
func (ea *ErrorAggregator) AddRequiredError(field string) {
	ea.errors.Add(field, "required", "field is required", nil)
}

// AddFormatError adds a format validation error
func (ea *ErrorAggregator) AddFormatError(field, expectedFormat string, value interface{}) {
	message := "invalid format"
	if expectedFormat != "" {
		message = fmt.Sprintf("invalid format, expected %s", expectedFormat)
	}
	ea.errors.Add(field, "format", message, value)
}

// AddRangeError adds a range validation error
func (ea *ErrorAggregator) AddRangeError(field string, min, max interface{}, value interface{}) {
	message := fmt.Sprintf("value must be between %v and %v", min, max)
	ea.errors.Add(field, "range", message, value)
}

// HasErrors returns true if there are any errors
func (ea *ErrorAggregator) HasErrors() bool {
	return ea.errors.HasErrors()
}

// GetErrors returns the collected errors
func (ea *ErrorAggregator) GetErrors() *ValidationErrors {
	return ea.errors
}

// Clear clears all collected errors
func (ea *ErrorAggregator) Clear() {
	ea.errors.Clear()
}

// Count returns the number of errors
func (ea *ErrorAggregator) Count() int {
	return ea.errors.Count()
}

// Error context for better error reporting

// ErrorContext provides context for validation errors
type ErrorContext struct {
	Path     string                 `json:"path"`
	Value    interface{}            `json:"value"`
	Context  string                 `json:"context"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ContextualValidationError represents a validation error with context
type ContextualValidationError struct {
	ValidationError
	Context *ErrorContext `json:"context,omitempty"`
}

// Error implements the error interface
func (e ContextualValidationError) Error() string {
	baseError := e.ValidationError.Error()
	if e.Context != nil && e.Context.Path != "" {
		return fmt.Sprintf("%s at path '%s'", baseError, e.Context.Path)
	}
	return baseError
}

// NewContextualError creates a new contextual validation error
func NewContextualError(field, rule, message string, value interface{}, context *ErrorContext) ContextualValidationError {
	return ContextualValidationError{
		ValidationError: ValidationError{
			Field:   field,
			Rule:    rule,
			Message: message,
			Value:   value,
		},
		Context: context,
	}
}

// Error formatting utilities

// FormatValidationErrors formats validation errors for user display
func FormatValidationErrors(errors *ValidationErrors, format string) string {
	if !errors.HasErrors() {
		return ""
	}

	switch format {
	case "simple":
		return formatSimple(errors)
	case "detailed":
		return formatDetailed(errors)
	case "json":
		return formatJSON(errors)
	default:
		return formatSimple(errors)
	}
}

// formatSimple formats errors in a simple format
func formatSimple(errors *ValidationErrors) string {
	var messages []string
	for _, err := range errors.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "; ")
}

// formatDetailed formats errors with detailed information
func formatDetailed(errors *ValidationErrors) string {
	var messages []string
	for _, err := range errors.Errors {
		message := fmt.Sprintf("Field '%s' failed '%s' validation: %s", err.Field, err.Rule, err.Message)
		if err.Value != nil {
			message += fmt.Sprintf(" (value: %v)", err.Value)
		}
		messages = append(messages, message)
	}
	return strings.Join(messages, "\n")
}

// formatJSON formats errors as JSON (simplified)
func formatJSON(errors *ValidationErrors) string {
	// This is a simplified JSON format
	// In a real implementation, you might use encoding/json
	return fmt.Sprintf("{\"errors\": %d, \"details\": \"%s\"}", errors.Count(), formatSimple(errors))
}