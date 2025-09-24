// Package target provides target management functionality for NotifyHub
package target

import (
	"fmt"
)

// Target represents a notification target (email, user, group, etc.)
// This structure encapsulates target information for different platforms
type Target struct {
	Type     string `json:"type"`     // "email", "user", "group", "channel", "webhook", "phone"
	Value    string `json:"value"`    // email address, user ID, phone number, etc.
	Platform string `json:"platform"` // "feishu", "email", "sms", "webhook"
}

// TargetType constants for type safety
const (
	TargetTypeEmail   = "email"
	TargetTypePhone   = "phone"
	TargetTypeUser    = "user"
	TargetTypeGroup   = "group"
	TargetTypeChannel = "channel"
	TargetTypeWebhook = "webhook"
)

// Platform constants for type safety
const (
	PlatformFeishu  = "feishu"
	PlatformEmail   = "email"
	PlatformSMS     = "sms"
	PlatformWebhook = "webhook"
	PlatformAuto    = "auto" // Auto-detect platform
)

// NewTarget creates a new target with the given type, value, and platform
func NewTarget(targetType, value, platform string) Target {
	return Target{
		Type:     targetType,
		Value:    value,
		Platform: platform,
	}
}

// String returns a string representation of the target
func (t Target) String() string {
	return fmt.Sprintf("%s:%s@%s", t.Type, t.Value, t.Platform)
}

// IsValid checks if the target has all required fields
func (t Target) IsValid() bool {
	return t.Type != "" && t.Value != ""
}
