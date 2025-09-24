// Package platform provides the interface and management for notification platform senders
// This defines the contract that all platform implementations must follow
package platform

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// Sender interface defines the contract that all platform notification senders must implement
// This is the core interface that bridges the dispatcher and actual platform implementations
type Sender interface {
	// Name returns the platform name (e.g., "feishu", "email", "sms")
	Name() string

	// Send sends a message to the specified targets and returns results
	Send(ctx context.Context, msg *InternalMessage, targets []InternalTarget) ([]*SendResult, error)

	// ValidateTarget validates a target for this platform
	ValidateTarget(target InternalTarget) error

	// GetCapabilities returns the capabilities of this platform
	GetCapabilities() PlatformCapabilities

	// IsHealthy checks if the platform is currently healthy and ready to send
	IsHealthy(ctx context.Context) error

	// Close gracefully shuts down the sender and cleans up resources
	Close() error
}

// InternalMessage represents a message in the internal format used by platform senders
// This is converted from the public API Message by the dispatcher
type InternalMessage struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Format      string                 `json:"format"` // "text", "markdown", "html"
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`

	// Platform-specific data extracted from public message
	PlatformData map[string]interface{} `json:"platform_data,omitempty"`
}

// InternalTarget represents a target in the internal format
type InternalTarget struct {
	Type     string `json:"type"`     // "email", "user", "group", "webhook", "phone"
	Value    string `json:"value"`    // target identifier
	Platform string `json:"platform"` // platform name
}

// SendResult represents the result of sending to a single target
type SendResult struct {
	Target     InternalTarget    `json:"target"`
	Success    bool              `json:"success"`
	MessageID  string            `json:"message_id,omitempty"`
	Error      string            `json:"error,omitempty"`
	StatusCode int               `json:"status_code,omitempty"`
	Response   string            `json:"response,omitempty"`
	SentAt     time.Time         `json:"sent_at"`
	Duration   time.Duration     `json:"duration"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// PlatformCapabilities describes what a platform can do
type PlatformCapabilities struct {
	Name                 string         `json:"name"`
	SupportedTargetTypes []string       `json:"supported_target_types"`
	SupportedFormats     []string       `json:"supported_formats"`
	MaxMessageSize       int            `json:"max_message_size"`
	SupportsScheduling   bool           `json:"supports_scheduling"`
	SupportsAttachments  bool           `json:"supports_attachments"`
	SupportsMentions     bool           `json:"supports_mentions"`
	SupportsRichContent  bool           `json:"supports_rich_content"`
	RateLimits           *RateLimitInfo `json:"rate_limits,omitempty"`
	RequiredSettings     []string       `json:"required_settings"`
}

// RateLimitInfo describes platform rate limiting information
type RateLimitInfo struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	RequestsPerMinute int           `json:"requests_per_minute"`
	RequestsPerHour   int           `json:"requests_per_hour"`
	BurstSize         int           `json:"burst_size"`
	WindowSize        time.Duration `json:"window_size"`
}

// SenderFactory creates platform senders based on configuration
type SenderFactory interface {
	// CreateSender creates a new sender for the specified platform
	CreateSender(platform string, config map[string]interface{}, logger logger.Logger) (Sender, error)

	// GetSupportedPlatforms returns a list of platforms this factory can create
	GetSupportedPlatforms() []string

	// ValidateConfig validates configuration for a platform
	ValidateConfig(platform string, config map[string]interface{}) error
}

// Manager manages multiple platform senders
type Manager interface {
	// RegisterSender registers a sender with the manager
	RegisterSender(sender Sender) error

	// GetSender retrieves a sender by platform name
	GetSender(platform string) (Sender, bool)

	// ListSenders returns all registered sender names
	ListSenders() []string

	// SendToAll sends a message to all targets across all relevant platforms
	SendToAll(ctx context.Context, msg *InternalMessage, targets []InternalTarget) ([]*SendResult, error)

	// HealthCheck checks the health of all registered senders
	HealthCheck(ctx context.Context) map[string]error

	// Close shuts down all senders
	Close() error
}

// MessageConverter converts between different message formats (placeholder)
type MessageConverter interface {
	// Placeholder - actual implementation depends on specific use case
	Convert(input interface{}) (interface{}, error)
	// ToInternal converts a message to internal format
	ToInternal(message interface{}, platform string) (*InternalMessage, error)
}

// Validator validates messages and targets
type Validator interface {
	// ValidateMessage validates a message for general requirements
	ValidateMessage(msg *InternalMessage) error

	// ValidateTarget validates a target format
	ValidateTarget(target InternalTarget) error

	// ValidateMessageForPlatform validates a message for a specific platform
	ValidateMessageForPlatform(msg *InternalMessage, platform string, capabilities PlatformCapabilities) error
}

// TargetResolver resolves target routing and platform assignment (placeholder)
type TargetResolver interface {
	// Placeholder - actual implementation depends on specific use case
	Resolve(input interface{}) (interface{}, error)
	// ResolveTargets determines which platform should handle each target
	ResolveTargets(targets interface{}) map[string][]InternalTarget
}

// HealthStatus represents the health status of a platform
type HealthStatus struct {
	Platform  string            `json:"platform"`
	Healthy   bool              `json:"healthy"`
	LastCheck time.Time         `json:"last_check"`
	Error     string            `json:"error,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Error types for platform operations

// PlatformError represents a platform-specific error
type PlatformError struct {
	Platform string
	Code     string
	Message  string
	Cause    error
}

func (e *PlatformError) Error() string {
	return e.Message
}

func (e *PlatformError) Unwrap() error {
	return e.Cause
}

// Common error codes
const (
	ErrorCodeInvalidConfig      = "INVALID_CONFIG"
	ErrorCodeUnsupportedTarget  = "UNSUPPORTED_TARGET"
	ErrorCodeUnsupportedFormat  = "UNSUPPORTED_FORMAT"
	ErrorCodeMessageTooLarge    = "MESSAGE_TOO_LARGE"
	ErrorCodeRateLimited        = "RATE_LIMITED"
	ErrorCodePlatformDown       = "PLATFORM_DOWN"
	ErrorCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrorCodeNetworkError       = "NETWORK_ERROR"
	ErrorCodeTimeout            = "TIMEOUT"
)

// Utility functions

// NewInternalMessage creates a new internal message
func NewInternalMessage(id, title, body string) *InternalMessage {
	return &InternalMessage{
		ID:           id,
		Title:        title,
		Body:         body,
		Format:       "text",
		Priority:     2, // Normal priority
		Metadata:     make(map[string]interface{}),
		Variables:    make(map[string]interface{}),
		PlatformData: make(map[string]interface{}),
	}
}

// NewInternalTarget creates a new internal target
func NewInternalTarget(targetType, value, platform string) InternalTarget {
	return InternalTarget{
		Type:     targetType,
		Value:    value,
		Platform: platform,
	}
}

// NewSendResult creates a new send result
func NewSendResult(target InternalTarget, success bool) *SendResult {
	return &SendResult{
		Target:   target,
		Success:  success,
		SentAt:   time.Now(),
		Metadata: make(map[string]string),
	}
}

// SetDuration sets the duration for the send result
func (r *SendResult) SetDuration(duration time.Duration) {
	r.Duration = duration
}

// NewPlatformCapabilities creates a new platform capabilities structure
func NewPlatformCapabilities(name string) PlatformCapabilities {
	return PlatformCapabilities{
		Name:                 name,
		SupportedTargetTypes: make([]string, 0),
		SupportedFormats:     make([]string, 0),
		RequiredSettings:     make([]string, 0),
	}
}

// NewPlatformError creates a new platform error
func NewPlatformError(platform, code, message string, cause error) *PlatformError {
	return &PlatformError{
		Platform: platform,
		Code:     code,
		Message:  message,
		Cause:    cause,
	}
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if platformErr, ok := err.(*PlatformError); ok {
		return platformErr.Code == ErrorCodeNetworkError ||
			platformErr.Code == ErrorCodeTimeout ||
			platformErr.Code == ErrorCodeRateLimited ||
			platformErr.Code == ErrorCodePlatformDown
	}
	return false
}

// GetErrorCode extracts the error code from a platform error
func GetErrorCode(err error) string {
	if platformErr, ok := err.(*PlatformError); ok {
		return platformErr.Code
	}
	return ""
}
