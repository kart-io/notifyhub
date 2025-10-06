// Package platform provides unified platform interfaces and types for NotifyHub
package platform

import (
	"context"

	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/target"
)

// Platform represents the unified platform interface
type Platform interface {
	// Platform identification
	Name() string
	GetCapabilities() Capabilities

	// Message sending
	Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error)
	ValidateTarget(target target.Target) error

	// Lifecycle management
	IsHealthy(ctx context.Context) error
	Close() error
}

// Capabilities describes platform capabilities and limitations
type Capabilities struct {
	Name                 string   `json:"name"`
	SupportedTargetTypes []string `json:"supported_target_types"`
	SupportedFormats     []string `json:"supported_formats"`
	MaxMessageSize       int      `json:"max_message_size"`
	SupportsScheduling   bool     `json:"supports_scheduling"`
	SupportsAttachments  bool     `json:"supports_attachments"`
	RequiredSettings     []string `json:"required_settings"`
}

// SendResult represents the result of sending to a single target
type SendResult struct {
	Target    target.Target `json:"target"`
	Success   bool          `json:"success"`
	MessageID string        `json:"message_id,omitempty"`
	Response  string        `json:"response,omitempty"`
	Error     error         `json:"error,omitempty"`
}

// Factory represents a platform factory function
type Factory func(config interface{}) (Platform, error)

// Registry manages platform instances and factories
type Registry interface {
	// Register a platform factory
	RegisterFactory(name string, factory Factory) error

	// Set configuration for a platform
	SetConfig(name string, config interface{}) error

	// Get a platform instance
	GetPlatform(name string) (Platform, error)

	// List registered platforms
	ListPlatforms() []string

	// Health check for all platforms
	Health(ctx context.Context) map[string]error

	// Close all platforms
	Close() error
}
