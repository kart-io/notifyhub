// Package platform defines the unified platform interface for NotifyHub
package platform

import (
	"context"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Platform represents a unified notification platform interface
// This interface consolidates the previous internal.Sender and platform.ExternalSender interfaces
// to eliminate the dual-layer architecture identified in the analysis
type Platform interface {
	// Name returns the unique name of the platform
	Name() string

	// Send sends a message to the specified targets
	Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error)

	// ValidateTarget validates if a target is supported by this platform
	ValidateTarget(target target.Target) error

	// GetCapabilities returns the capabilities of this platform
	GetCapabilities() Capabilities

	// IsHealthy performs a health check for this platform
	IsHealthy(ctx context.Context) error

	// Close gracefully shuts down the platform and releases resources
	Close() error
}

// SendResult represents the result of sending to a single target
type SendResult struct {
	Target    target.Target          `json:"target"`
	Success   bool                   `json:"success"`
	MessageID string                 `json:"message_id,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Response  string                 `json:"response,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Capabilities describes what a platform can do
type Capabilities struct {
	Name                 string   `json:"name"`
	SupportedTargetTypes []string `json:"supported_target_types"`
	SupportedFormats     []string `json:"supported_formats"`
	MaxMessageSize       int      `json:"max_message_size"`
	SupportsScheduling   bool     `json:"supports_scheduling"`
	SupportsAttachments  bool     `json:"supports_attachments"`
	SupportsMentions     bool     `json:"supports_mentions"`
	SupportsRichContent  bool     `json:"supports_rich_content"`
	RequiredSettings     []string `json:"required_settings"`
}
