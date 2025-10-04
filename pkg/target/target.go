// Package target provides unified target structures and types for NotifyHub
package target

import (
	"github.com/kart/notifyhub/pkg/errors"
)

// Target represents a unified target structure
type Target struct {
	Type     string `json:"type"`     // "email", "user", "group", "channel"
	Value    string `json:"value"`    // specific address or ID
	Platform string `json:"platform"` // "feishu", "email", "webhook"
}

// Target type constants
const (
	TargetTypeEmail   = "email"
	TargetTypePhone   = "phone"
	TargetTypeUser    = "user"
	TargetTypeGroup   = "group"
	TargetTypeChannel = "channel"
	TargetTypeWebhook = "webhook"
)

// Platform constants
const (
	PlatformFeishu  = "feishu"
	PlatformEmail   = "email"
	PlatformWebhook = "webhook"
)

// New creates a new target
func New(targetType, value, platform string) Target {
	return Target{
		Type:     targetType,
		Value:    value,
		Platform: platform,
	}
}

// NewEmail creates an email target
func NewEmail(email string) Target {
	return Target{
		Type:     TargetTypeEmail,
		Value:    email,
		Platform: PlatformEmail,
	}
}

// NewFeishuUser creates a Feishu user target
func NewFeishuUser(userID string) Target {
	return Target{
		Type:     TargetTypeUser,
		Value:    userID,
		Platform: PlatformFeishu,
	}
}

// NewFeishuGroup creates a Feishu group target
func NewFeishuGroup(groupID string) Target {
	return Target{
		Type:     TargetTypeGroup,
		Value:    groupID,
		Platform: PlatformFeishu,
	}
}

// NewWebhook creates a webhook target
func NewWebhook(url string) Target {
	return Target{
		Type:     TargetTypeWebhook,
		Value:    url,
		Platform: PlatformWebhook,
	}
}

// Validate validates the target
func (t *Target) Validate() error {
	if t.Type == "" {
		return errors.New(errors.ErrEmptyTargetType, "target type cannot be empty")
	}
	if t.Value == "" {
		return errors.New(errors.ErrEmptyTargetValue, "target value cannot be empty")
	}
	if t.Platform == "" {
		return errors.New(errors.ErrEmptyPlatform, "target platform cannot be empty")
	}
	return nil
}

// IsEmail returns true if the target is an email
func (t *Target) IsEmail() bool {
	return t.Type == TargetTypeEmail
}

// IsUser returns true if the target is a user
func (t *Target) IsUser() bool {
	return t.Type == TargetTypeUser
}

// IsGroup returns true if the target is a group
func (t *Target) IsGroup() bool {
	return t.Type == TargetTypeGroup
}

// IsWebhook returns true if the target is a webhook
func (t *Target) IsWebhook() bool {
	return t.Type == TargetTypeWebhook
}

// String returns a string representation of the target
func (t *Target) String() string {
	return t.Platform + ":" + t.Type + ":" + t.Value
}
