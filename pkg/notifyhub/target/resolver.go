// Package target provides target resolution functionality for NotifyHub
package target

import (
	"fmt"
	"regexp"
	"strings"
)

// AutoDetectTarget automatically detects target type and platform based on value
func AutoDetectTarget(value string) Target {
	targetType, platform := detectTargetTypeAndPlatform(value)
	return Target{
		Type:     targetType,
		Value:    value,
		Platform: platform,
	}
}

// detectTargetTypeAndPlatform automatically detects target type and platform based on value
func detectTargetTypeAndPlatform(value string) (string, string) {
	// Email detection
	if strings.Contains(value, "@") && strings.Contains(value, ".") {
		return TargetTypeEmail, PlatformEmail
	}

	// Phone number detection
	if strings.HasPrefix(value, "+") && len(value) > 5 {
		return TargetTypePhone, PlatformSMS
	}

	// Webhook URL detection
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return TargetTypeWebhook, PlatformWebhook
	}

	// Feishu user ID detection (starts with "ou_" or "oc_")
	if strings.HasPrefix(value, "ou_") || strings.HasPrefix(value, "oc_") {
		return TargetTypeUser, PlatformFeishu
	}

	// Feishu group ID detection (starts with "oc_" but different pattern)
	if strings.HasPrefix(value, "og_") {
		return TargetTypeGroup, PlatformFeishu
	}

	// Default to user on auto platform
	return TargetTypeUser, PlatformAuto
}

// Validate validates the target configuration
func (t Target) Validate() error {
	if t.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}

	switch t.Type {
	case TargetTypeEmail:
		return t.validateEmail()
	case TargetTypePhone:
		return t.validatePhone()
	case TargetTypeWebhook:
		return t.validateWebhook()
	case TargetTypeUser, TargetTypeGroup, TargetTypeChannel:
		return t.validateID()
	default:
		return fmt.Errorf("unsupported target type: %s", t.Type)
	}
}

// IsEmail returns true if the target is an email
func (t Target) IsEmail() bool {
	return t.Type == TargetTypeEmail
}

// IsPhone returns true if the target is a phone number
func (t Target) IsPhone() bool {
	return t.Type == TargetTypePhone
}

// IsWebhook returns true if the target is a webhook
func (t Target) IsWebhook() bool {
	return t.Type == TargetTypeWebhook
}

// IsFeishu returns true if the target is for Feishu platform
func (t Target) IsFeishu() bool {
	return t.Platform == PlatformFeishu
}

// validateEmail validates email format
func (t Target) validateEmail() error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(t.Value) {
		return fmt.Errorf("invalid email format: %s", t.Value)
	}
	return nil
}

// validatePhone validates phone number format
func (t Target) validatePhone() error {
	// Basic phone number validation (starts with + followed by digits)
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(t.Value) {
		return fmt.Errorf("invalid phone number format: %s (expected format: +1234567890)", t.Value)
	}
	return nil
}

// validateWebhook validates webhook URL format
func (t Target) validateWebhook() error {
	if !strings.HasPrefix(t.Value, "http://") && !strings.HasPrefix(t.Value, "https://") {
		return fmt.Errorf("invalid webhook URL: %s (must start with http:// or https://)", t.Value)
	}
	return nil
}

// validateID validates user/group/channel ID format
func (t Target) validateID() error {
	if len(t.Value) < 2 {
		return fmt.Errorf("invalid ID format: %s (too short)", t.Value)
	}
	return nil
}
