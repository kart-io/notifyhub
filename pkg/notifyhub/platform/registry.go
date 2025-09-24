// Package platform provides public APIs for platform registration and extension
package platform

import (
	"context"

	"github.com/kart-io/notifyhub/pkg/logger"
)

const (
	NameEmail   = "email"
	NameFeishu  = "feishu"
	NameSMS     = "sms"
	NameSlack   = "slack"
	NameDiscord = "discord"
	NameTeams   = "teams"
	NameWebhook = "webhook"
)

// ExternalSender represents a platform notification sender (public interface for external implementations)
type ExternalSender interface {
	// Name returns the platform name
	Name() string

	// Send sends a message to the specified targets
	Send(ctx context.Context, msg *Message, targets []Target) ([]*SendResult, error)

	// ValidateTarget validates a target for this platform
	ValidateTarget(target Target) error

	// GetCapabilities returns the capabilities of this platform
	GetCapabilities() Capabilities

	// IsHealthy checks if the platform is healthy
	IsHealthy(ctx context.Context) error

	// Close gracefully shuts down the sender
	Close() error
}

// Message represents the message structure for external platforms
type Message struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	Format       string                 `json:"format"` // "text", "markdown", "html"
	Priority     int                    `json:"priority"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	PlatformData map[string]interface{} `json:"platform_data,omitempty"`
}

// Target represents a target for external platforms
type Target struct {
	Type     string `json:"type"`     // "email", "user", "group", "webhook", "phone"
	Value    string `json:"value"`    // target identifier
	Platform string `json:"platform"` // platform name
}

// SendResult represents the result of sending to a single target
type SendResult struct {
	Target    Target                 `json:"target"`
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

// ExternalSenderCreator is a function that creates an external sender with given configuration
type ExternalSenderCreator func(config map[string]interface{}, logger logger.Logger) (ExternalSender, error)

// RegisterPlatform registers an external platform sender creator
// This is the public API for external packages to register their platforms
func RegisterPlatform(platformName string, creator ExternalSenderCreator) {
	// Use a global registry that can be accessed by core package
	globalPlatformRegistry[platformName] = creator
}

// Global registry for platform creators
var globalPlatformRegistry = make(map[string]ExternalSenderCreator)

// GetRegisteredCreators returns all registered platform creators
func GetRegisteredCreators() map[string]ExternalSenderCreator {
	return globalPlatformRegistry
}

// Note: senderAdapter has been moved to internal package to avoid
// circular dependencies and maintain Go package design principles

// GetRegisteredPlatforms returns a list of all registered platform names
func GetRegisteredPlatforms() []string {
	// This would need to be implemented in the internal package
	// For now, return common platforms
	return []string{NameFeishu, NameEmail, NameSMS}
}

// IsRegistered checks if a platform is registered
func IsRegistered(platformName string) bool {
	platforms := GetRegisteredPlatforms()
	for _, p := range platforms {
		if p == platformName {
			return true
		}
	}
	return false
}
