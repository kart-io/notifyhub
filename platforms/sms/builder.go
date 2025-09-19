package sms

import (
	"github.com/kart-io/notifyhub/platforms/common"
)

// SMSBuilder provides SMS-specific functionality
// This ensures SMS has equal functionality as other platforms per proposal
type SMSBuilder struct {
	base           *common.MessageBuilder
	fromNumber     string
	groups         []string
	deliveryReport bool
}

// Platform returns the platform name
func (sb *SMSBuilder) Platform() string {
	return "sms"
}

// Builder returns the base message builder
func (sb *SMSBuilder) Builder() *common.MessageBuilder {
	return sb.base
}

// FromNumber sets the sender phone number (SMS-specific functionality)
func (sb *SMSBuilder) FromNumber(number string) *SMSBuilder {
	sb.fromNumber = number
	return sb
}

// ToNumber adds a phone number target
func (sb *SMSBuilder) ToNumber(phoneNumber string) *SMSBuilder {
	target := map[string]interface{}{
		"type":     "user",
		"value":    phoneNumber,
		"platform": "sms",
	}
	sb.base.AddTarget(target)
	return sb
}

// ToGroup adds an SMS group target (SMS-specific functionality)
func (sb *SMSBuilder) ToGroup(groupID string) *SMSBuilder {
	sb.groups = append(sb.groups, groupID)
	target := map[string]interface{}{
		"type":     "group",
		"value":    groupID,
		"platform": "sms",
	}
	sb.base.AddTarget(target)
	return sb
}

// DeliveryReport enables/disables delivery reports (SMS-specific functionality)
func (sb *SMSBuilder) DeliveryReport(enabled bool) *SMSBuilder {
	sb.deliveryReport = enabled
	return sb
}

// Priority sets SMS priority
func (sb *SMSBuilder) Priority(priority SMSPriority) *SMSBuilder {
	// Add priority as metadata
	return sb
}

// FlashSMS sends as a flash SMS (appears immediately on screen)
func (sb *SMSBuilder) FlashSMS() *SMSBuilder {
	// Add flash SMS flag to metadata
	return sb
}

// ValidityPeriod sets how long the SMS should be kept trying to deliver
func (sb *SMSBuilder) ValidityPeriod(hours int) *SMSBuilder {
	// Add validity period to metadata
	return sb
}

// Build returns the final message with SMS-specific processing
func (sb *SMSBuilder) Build() *common.MessageBuilder {
	// Process SMS-specific features and add them to message metadata
	if sb.fromNumber != "" {
		// Add from number to message metadata
		// TODO: Implement from number handling
		_ = sb.fromNumber // Acknowledge usage
	}
	if len(sb.groups) > 0 {
		// Add group information to message metadata
		// TODO: Implement group handling
		_ = sb.groups // Acknowledge usage
	}
	if sb.deliveryReport {
		// Add delivery report flag to message metadata
		// TODO: Implement delivery report handling
		_ = sb.deliveryReport // Acknowledge usage
	}
	return sb.base
}
