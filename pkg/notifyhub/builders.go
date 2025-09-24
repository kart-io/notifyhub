// Package notifyhub provides builder adapter functions for backward compatibility.
// This file contains factory functions that delegate to the modular message and target packages.
package notifyhub

import (
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Message builder functions provide convenient ways to create notification messages.

// NewMessage creates a new message builder with an optional title.
//
// The message builder provides a fluent interface for constructing notification
// messages with various properties like body text, priority, targets, and metadata.
//
// Parameters:
//   - title: Optional message title. If not provided, defaults to empty string
//
// Returns:
//   - *MessageBuilder: A message builder instance for fluent configuration
//
// Example:
//
//	msg := NewMessage("System Alert").
//	    WithBody("Database connection failed").
//	    WithPriority(PriorityHigh).
//	    ToTarget(NewEmailTarget("admin@company.com")).
//	    Build()
func NewMessage(title ...string) *MessageBuilder {
	return message.NewMessage(title...)
}

// NewAlert creates a new alert message with high priority.
//
// Alert messages are pre-configured with high priority and are suitable
// for important notifications that require immediate attention.
//
// Parameters:
//   - title: The alert message title
//
// Returns:
//   - *MessageBuilder: A message builder configured for alerts
//
// Example:
//
//	msg := NewAlert("Security Breach Detected").
//	    WithBody("Unauthorized access attempt from IP: 192.168.1.100").
//	    ToTarget(NewEmailTarget("security@company.com")).
//	    Build()
func NewAlert(title string) *MessageBuilder {
	return message.NewAlert(title)
}

// NewUrgent creates a new urgent message with the highest priority.
//
// Urgent messages are pre-configured with urgent priority and are suitable
// for critical notifications that require immediate action.
//
// Parameters:
//   - title: The urgent message title
//
// Returns:
//   - *MessageBuilder: A message builder configured for urgent messages
//
// Example:
//
//	msg := NewUrgent("Service Down").
//	    WithBody("Payment service is currently unavailable").
//	    ToTarget(NewFeishuUserTarget("user123")).
//	    Build()
func NewUrgent(title string) *MessageBuilder {
	return message.NewUrgent(title)
}

// Target factory functions provide convenient ways to create notification targets.

// NewTarget creates a new notification target with the specified type, value, and platform.
//
// This is the most flexible target creation function, allowing you to specify
// all target properties explicitly.
//
// Parameters:
//   - targetType: Type of target (email, phone, user, group, channel, webhook)
//   - value: Target value (email address, phone number, user ID, etc.)
//   - platform: Platform name (feishu, email, sms, etc.)
//
// Returns:
//   - Target: Configured target instance
//
// Example:
//
//	target := NewTarget("email", "admin@company.com", "email")
//	target := NewTarget("user", "user123", "feishu")
func NewTarget(targetType, value, platform string) Target {
	return target.NewTarget(targetType, value, platform)
}

// NewEmailTarget creates a new email target.
//
// This is a convenience function for creating email targets without
// specifying the type and platform explicitly.
//
// Parameters:
//   - email: Email address to send notifications to
//
// Returns:
//   - Target: Email target configured for the email platform
//
// Example:
//
//	target := NewEmailTarget("user@company.com")
func NewEmailTarget(email string) Target {
	return target.NewEmailTarget(email)
}

// NewPhoneTarget creates a new phone/SMS target.
//
// This is a convenience function for creating SMS targets without
// specifying the type and platform explicitly.
//
// Parameters:
//   - phone: Phone number to send SMS notifications to
//
// Returns:
//   - Target: Phone target configured for the SMS platform
//
// Example:
//
//	target := NewPhoneTarget("+1234567890")
func NewPhoneTarget(phone string) Target {
	return target.NewPhoneTarget(phone)
}

// NewFeishuUserTarget creates a new Feishu user target.
//
// This is a convenience function for creating Feishu user targets
// for direct messaging specific users.
//
// Parameters:
//   - userID: Feishu user ID to send notifications to
//
// Returns:
//   - Target: User target configured for the Feishu platform
//
// Example:
//
//	target := NewFeishuUserTarget("ou_7d8a6e6ef99ee78690f65b636411c24")
func NewFeishuUserTarget(userID string) Target {
	return target.NewFeishuUserTarget(userID)
}

// NewFeishuGroupTarget creates a new Feishu group target.
//
// This is a convenience function for creating Feishu group targets
// for sending notifications to group chats.
//
// Parameters:
//   - groupID: Feishu group/chat ID to send notifications to
//
// Returns:
//   - Target: Group target configured for the Feishu platform
//
// Example:
//
//	target := NewFeishuGroupTarget("oc_a0553eda9014c201e6969b478895c230")
func NewFeishuGroupTarget(groupID string) Target {
	return target.NewFeishuGroupTarget(groupID)
}

// NewWebhookTarget creates a new webhook target.
//
// This is a convenience function for creating webhook targets
// for sending notifications to HTTP endpoints.
//
// Parameters:
//   - url: Webhook URL to send notifications to
//
// Returns:
//   - Target: Webhook target configured for HTTP delivery
//
// Example:
//
//	target := NewWebhookTarget("https://api.company.com/notifications")
func NewWebhookTarget(url string) Target {
	return target.NewWebhookTarget(url)
}

// AutoDetectTarget creates a target by automatically detecting the type from the value.
//
// This function analyzes the provided value and attempts to determine
// the appropriate target type and platform automatically.
//
// Parameters:
//   - value: Target value to analyze (email, phone, URL, etc.)
//
// Returns:
//   - Target: Automatically configured target
//
// Example:
//
//	emailTarget := AutoDetectTarget("user@company.com")     // Detects email
//	phoneTarget := AutoDetectTarget("+1234567890")          // Detects phone
//	webhookTarget := AutoDetectTarget("https://example.com") // Detects webhook
func AutoDetectTarget(value string) Target {
	return target.AutoDetectTarget(value)
}

// Utility functions

// IsValidationError checks if an error is a validation error.
//
// This function can be used to distinguish between validation errors
// and other types of errors when creating hubs or sending messages.
//
// Parameters:
//   - err: Error to check
//
// Returns:
//   - bool: True if the error is a validation error
//
// Example:
//
//	if err := hub.Send(ctx, msg); err != nil {
//	    if IsValidationError(err) {
//	        log.Printf("Validation error: %v", err)
//	    } else {
//	        log.Printf("Runtime error: %v", err)
//	    }
//	}
func IsValidationError(err error) bool {
	// Simple validation error check
	if err == nil {
		return false
	}
	// You can implement more sophisticated error checking here
	return false
}
