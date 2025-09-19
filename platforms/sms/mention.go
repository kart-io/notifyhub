package sms

import (
	"fmt"

	"github.com/kart-io/notifyhub/platforms/common"
)

// SMSMention implements the Mention interface for SMS platform
// This provides SMS-specific mention functionality (e.g., phone number references)
type SMSMention struct {
	MentionType common.MentionType
	PhoneNumber string
	ContactName string
	IsEmergency bool // Whether this is an emergency contact mention
}

// Platform returns the platform name
func (sm SMSMention) Platform() string {
	return "sms"
}

// Type returns the mention type
func (sm SMSMention) Type() common.MentionType {
	return sm.MentionType
}

// Target returns the mention target
func (sm SMSMention) Target() string {
	return sm.PhoneNumber
}

// Render renders the mention according to the message format
func (sm SMSMention) Render(format string) (string, error) {
	switch format {
	case "text":
		return sm.renderTextMention()
	case "compact":
		return sm.renderCompactMention()
	default:
		return "", fmt.Errorf("unsupported format for SMS mention: %s", format)
	}
}

// renderTextMention renders a standard text mention
func (sm SMSMention) renderTextMention() (string, error) {
	if sm.ContactName != "" {
		if sm.IsEmergency {
			return fmt.Sprintf("EMERGENCY: %s (%s)", sm.ContactName, sm.PhoneNumber), nil
		}
		return fmt.Sprintf("%s (%s)", sm.ContactName, sm.PhoneNumber), nil
	}
	if sm.IsEmergency {
		return fmt.Sprintf("EMERGENCY: %s", sm.PhoneNumber), nil
	}
	return sm.PhoneNumber, nil
}

// renderCompactMention renders a compact mention for SMS length limits
func (sm SMSMention) renderCompactMention() (string, error) {
	if sm.ContactName != "" && len(sm.ContactName) <= 10 {
		return sm.ContactName, nil
	}
	// Return last 4 digits for privacy
	if len(sm.PhoneNumber) >= 4 {
		return "***" + sm.PhoneNumber[len(sm.PhoneNumber)-4:], nil
	}
	return sm.PhoneNumber, nil
}
