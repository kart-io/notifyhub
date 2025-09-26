// Package errors defines error codes and categories for NotifyHub
package errors

// Error Categories
const (
	// Configuration Errors (CON)
	ConfigurationCategory = "CON"

	// Platform Errors (PLT)
	PlatformCategory = "PLT"

	// Message Errors (MSG)
	MessageCategory = "MSG"

	// Template Errors (TPL)
	TemplateCategory = "TPL"

	// Queue Errors (QUE)
	QueueCategory = "QUE"

	// Network Errors (NET)
	NetworkCategory = "NET"

	// Validation Errors (VAL)
	ValidationCategory = "VAL"

	// System Errors (SYS)
	SystemCategory = "SYS"
)

// Configuration Error Codes
const (
	ErrInvalidConfig     Code = "CON001" // Invalid configuration
	ErrMissingConfig     Code = "CON002" // Missing required configuration
	ErrConfigValidation  Code = "CON003" // Configuration validation failed
	ErrUnsupportedConfig Code = "CON004" // Unsupported configuration option
	ErrConfigLoadFailed  Code = "CON005" // Failed to load configuration
)

// Platform Error Codes
const (
	ErrPlatformNotFound    Code = "PLT001" // Platform not found
	ErrPlatformUnavailable Code = "PLT002" // Platform unavailable
	ErrPlatformAuth        Code = "PLT003" // Platform authentication failed
	ErrPlatformRateLimit   Code = "PLT004" // Platform rate limit exceeded
	ErrPlatformTimeout     Code = "PLT005" // Platform timeout
	ErrPlatformInternal    Code = "PLT006" // Platform internal error
	ErrPlatformMaintenance Code = "PLT007" // Platform under maintenance
)

// Message Error Codes
const (
	ErrInvalidMessage    Code = "MSG001" // Invalid message format
	ErrMessageTooLarge   Code = "MSG002" // Message size exceeds limit
	ErrInvalidTarget     Code = "MSG003" // Invalid message target
	ErrMessageEncoding   Code = "MSG004" // Message encoding error
	ErrMessageSendFailed Code = "MSG005" // Failed to send message
	ErrMessageTimeout    Code = "MSG006" // Message sending timeout
)

// Template Error Codes
const (
	ErrTemplateNotFound   Code = "TPL001" // Template not found
	ErrTemplateInvalid    Code = "TPL002" // Invalid template syntax
	ErrTemplateRender     Code = "TPL003" // Template rendering failed
	ErrTemplateEngine     Code = "TPL004" // Template engine error
	ErrTemplateVariables  Code = "TPL005" // Missing or invalid template variables
	ErrTemplateCacheError Code = "TPL006" // Template cache error
)

// Queue Error Codes
const (
	ErrQueueFull          Code = "QUE001" // Queue is full
	ErrQueueEmpty         Code = "QUE002" // Queue is empty
	ErrQueueTimeout       Code = "QUE003" // Queue operation timeout
	ErrQueueConnection    Code = "QUE004" // Queue connection error
	ErrQueueSerialization Code = "QUE005" // Queue serialization error
	ErrQueueWorkerFailed  Code = "QUE006" // Queue worker failed
)

// Network Error Codes
const (
	ErrNetworkTimeout    Code = "NET001" // Network timeout
	ErrNetworkConnection Code = "NET002" // Network connection error
	ErrNetworkDNS        Code = "NET003" // DNS resolution error
	ErrNetworkSSL        Code = "NET004" // SSL/TLS error
	ErrNetworkProtocol   Code = "NET005" // Protocol error
)

// Validation Error Codes
const (
	ErrValidationFailed Code = "VAL001" // Validation failed
	ErrInvalidFormat    Code = "VAL002" // Invalid format
	ErrMissingRequired  Code = "VAL003" // Missing required field
	ErrValueOutOfRange  Code = "VAL004" // Value out of range
	ErrInvalidType      Code = "VAL005" // Invalid data type
)

// System Error Codes
const (
	ErrSystemUnavailable Code = "SYS001" // System unavailable
	ErrInternalError     Code = "SYS002" // Internal system error
	ErrResourceExhausted Code = "SYS003" // Resource exhausted
	ErrPermissionDenied  Code = "SYS004" // Permission denied
	ErrSystemTimeout     Code = "SYS005" // System operation timeout
	ErrSystemOverload    Code = "SYS006" // System overload
)

// ErrorInfo contains metadata about error codes
type ErrorInfo struct {
	Code        Code   `json:"code"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Retryable   bool   `json:"retryable"`
}

// GetErrorInfo returns metadata for a given error code
func GetErrorInfo(code Code) ErrorInfo {
	errorInfoMap := map[Code]ErrorInfo{
		// Configuration Errors
		ErrInvalidConfig:     {code, ConfigurationCategory, "ERROR", "Invalid configuration provided", false},
		ErrMissingConfig:     {code, ConfigurationCategory, "ERROR", "Required configuration missing", false},
		ErrConfigValidation:  {code, ConfigurationCategory, "ERROR", "Configuration validation failed", false},
		ErrUnsupportedConfig: {code, ConfigurationCategory, "ERROR", "Unsupported configuration option", false},
		ErrConfigLoadFailed:  {code, ConfigurationCategory, "ERROR", "Failed to load configuration", true},

		// Platform Errors
		ErrPlatformNotFound:    {code, PlatformCategory, "ERROR", "Platform not found or registered", false},
		ErrPlatformUnavailable: {code, PlatformCategory, "ERROR", "Platform currently unavailable", true},
		ErrPlatformAuth:        {code, PlatformCategory, "ERROR", "Platform authentication failed", false},
		ErrPlatformRateLimit:   {code, PlatformCategory, "WARN", "Platform rate limit exceeded", true},
		ErrPlatformTimeout:     {code, PlatformCategory, "WARN", "Platform operation timeout", true},
		ErrPlatformInternal:    {code, PlatformCategory, "ERROR", "Platform internal error", true},
		ErrPlatformMaintenance: {code, PlatformCategory, "WARN", "Platform under maintenance", true},

		// Message Errors
		ErrInvalidMessage:    {code, MessageCategory, "ERROR", "Invalid message format", false},
		ErrMessageTooLarge:   {code, MessageCategory, "ERROR", "Message size exceeds limit", false},
		ErrInvalidTarget:     {code, MessageCategory, "ERROR", "Invalid message target", false},
		ErrMessageEncoding:   {code, MessageCategory, "ERROR", "Message encoding error", false},
		ErrMessageSendFailed: {code, MessageCategory, "ERROR", "Failed to send message", true},
		ErrMessageTimeout:    {code, MessageCategory, "WARN", "Message sending timeout", true},

		// Template Errors
		ErrTemplateNotFound:   {code, TemplateCategory, "ERROR", "Template not found", false},
		ErrTemplateInvalid:    {code, TemplateCategory, "ERROR", "Invalid template syntax", false},
		ErrTemplateRender:     {code, TemplateCategory, "ERROR", "Template rendering failed", false},
		ErrTemplateEngine:     {code, TemplateCategory, "ERROR", "Template engine error", false},
		ErrTemplateVariables:  {code, TemplateCategory, "ERROR", "Missing or invalid template variables", false},
		ErrTemplateCacheError: {code, TemplateCategory, "WARN", "Template cache error", true},

		// Queue Errors
		ErrQueueFull:          {code, QueueCategory, "WARN", "Queue is full", true},
		ErrQueueEmpty:         {code, QueueCategory, "INFO", "Queue is empty", false},
		ErrQueueTimeout:       {code, QueueCategory, "WARN", "Queue operation timeout", true},
		ErrQueueConnection:    {code, QueueCategory, "ERROR", "Queue connection error", true},
		ErrQueueSerialization: {code, QueueCategory, "ERROR", "Queue serialization error", false},
		ErrQueueWorkerFailed:  {code, QueueCategory, "ERROR", "Queue worker failed", true},

		// Network Errors
		ErrNetworkTimeout:    {code, NetworkCategory, "WARN", "Network timeout", true},
		ErrNetworkConnection: {code, NetworkCategory, "ERROR", "Network connection error", true},
		ErrNetworkDNS:        {code, NetworkCategory, "ERROR", "DNS resolution error", true},
		ErrNetworkSSL:        {code, NetworkCategory, "ERROR", "SSL/TLS error", false},
		ErrNetworkProtocol:   {code, NetworkCategory, "ERROR", "Protocol error", false},

		// Validation Errors
		ErrValidationFailed: {code, ValidationCategory, "ERROR", "Validation failed", false},
		ErrInvalidFormat:    {code, ValidationCategory, "ERROR", "Invalid format", false},
		ErrMissingRequired:  {code, ValidationCategory, "ERROR", "Missing required field", false},
		ErrValueOutOfRange:  {code, ValidationCategory, "ERROR", "Value out of range", false},
		ErrInvalidType:      {code, ValidationCategory, "ERROR", "Invalid data type", false},

		// System Errors
		ErrSystemUnavailable: {code, SystemCategory, "CRITICAL", "System unavailable", true},
		ErrInternalError:     {code, SystemCategory, "ERROR", "Internal system error", false},
		ErrResourceExhausted: {code, SystemCategory, "CRITICAL", "Resource exhausted", true},
		ErrPermissionDenied:  {code, SystemCategory, "ERROR", "Permission denied", false},
		ErrSystemTimeout:     {code, SystemCategory, "WARN", "System operation timeout", true},
		ErrSystemOverload:    {code, SystemCategory, "CRITICAL", "System overload", true},
	}

	if info, exists := errorInfoMap[code]; exists {
		return info
	}

	// Return default info for unknown codes
	return ErrorInfo{
		Code:        code,
		Category:    "UNKNOWN",
		Severity:    "ERROR",
		Description: "Unknown error code",
		Retryable:   false,
	}
}

// IsRetryable checks if an error code is retryable
func IsRetryable(code Code) bool {
	info := GetErrorInfo(code)
	return info.Retryable
}

// GetCategory returns the category for an error code
func GetCategory(code Code) string {
	info := GetErrorInfo(code)
	return info.Category
}

// GetSeverity returns the severity level for an error code
func GetSeverity(code Code) string {
	info := GetErrorInfo(code)
	return info.Severity
}

// Factory functions for common error creation

// NewConfigError creates a configuration error
func NewConfigError(code Code, message string) *NotifyError {
	return New(code, message).WithContext("category", ConfigurationCategory)
}

// NewPlatformError creates a platform error
func NewPlatformError(code Code, platform string, message string) *NotifyError {
	return New(code, message).
		WithContext("category", PlatformCategory).
		WithContext("platform", platform)
}

// NewMessageError creates a message error
func NewMessageError(code Code, messageID string, message string) *NotifyError {
	return New(code, message).
		WithContext("category", MessageCategory).
		WithContext("message_id", messageID)
}

// NewTemplateError creates a template error
func NewTemplateError(code Code, templateName string, message string) *NotifyError {
	return New(code, message).
		WithContext("category", TemplateCategory).
		WithContext("template", templateName)
}

// NewQueueError creates a queue error
func NewQueueError(code Code, queueName string, message string) *NotifyError {
	return New(code, message).
		WithContext("category", QueueCategory).
		WithContext("queue", queueName)
}

// NewNetworkError creates a network error
func NewNetworkError(code Code, endpoint string, message string) *NotifyError {
	return New(code, message).
		WithContext("category", NetworkCategory).
		WithContext("endpoint", endpoint)
}

// NewValidationError creates a validation error
func NewValidationError(code Code, field string, message string) *NotifyError {
	return New(code, message).
		WithContext("category", ValidationCategory).
		WithContext("field", field)
}

// NewSystemError creates a system error
func NewSystemError(code Code, component string, message string) *NotifyError {
	return New(code, message).
		WithContext("category", SystemCategory).
		WithContext("component", component)
}
