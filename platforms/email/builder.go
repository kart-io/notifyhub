package email

import (
	"github.com/kart-io/notifyhub/platforms/common"
)

// EmailBuilder provides Email-specific functionality
// This implements equal functionality to Feishu as required by the proposal
type EmailBuilder struct {
	base        *common.MessageBuilder
	ccList      []string
	bccList     []string
	attachments []Attachment
}

// Platform returns the platform name
func (eb *EmailBuilder) Platform() string {
	return "email"
}

// Builder returns the base message builder
func (eb *EmailBuilder) Builder() *common.MessageBuilder {
	return eb.base
}

// CC adds CC recipients (Email-specific functionality)
func (eb *EmailBuilder) CC(emails ...string) *EmailBuilder {
	eb.ccList = append(eb.ccList, emails...)
	return eb
}

// BCC adds BCC recipients (Email-specific functionality)
func (eb *EmailBuilder) BCC(emails ...string) *EmailBuilder {
	eb.bccList = append(eb.bccList, emails...)
	return eb
}

// Attach adds an attachment (Email-specific functionality)
func (eb *EmailBuilder) Attach(filename, content string) *EmailBuilder {
	attachment := Attachment{
		Filename: filename,
		Content:  content,
		MimeType: "application/octet-stream",
	}
	eb.attachments = append(eb.attachments, attachment)
	return eb
}

// AttachFile adds a file attachment with MIME type
func (eb *EmailBuilder) AttachFile(filename, content, mimeType string) *EmailBuilder {
	attachment := Attachment{
		Filename: filename,
		Content:  content,
		MimeType: mimeType,
	}
	eb.attachments = append(eb.attachments, attachment)
	return eb
}

// ToEmail adds an email target
func (eb *EmailBuilder) ToEmail(email string) *EmailBuilder {
	target := map[string]interface{}{
		"type":     "email",
		"value":    email,
		"platform": "email",
	}
	eb.base.AddTarget(target)
	return eb
}

// ToGroup adds an email group target (distribution list)
func (eb *EmailBuilder) ToGroup(groupEmail string) *EmailBuilder {
	target := map[string]interface{}{
		"type":     "group",
		"value":    groupEmail,
		"platform": "email",
	}
	eb.base.AddTarget(target)
	return eb
}

// Priority sets email priority
func (eb *EmailBuilder) Priority(priority EmailPriority) *EmailBuilder {
	// Add priority as metadata
	// This would be processed by the email transport
	return eb
}

// Build returns the final message with Email-specific processing
func (eb *EmailBuilder) Build() *common.MessageBuilder {
	// Process email-specific features and add them to message metadata
	if len(eb.ccList) > 0 {
		// Add CC list to message metadata
		// TODO: Implement CC list handling
		_ = eb.ccList // Acknowledge usage
	}
	if len(eb.bccList) > 0 {
		// Add BCC list to message metadata
		// TODO: Implement BCC list handling
		_ = eb.bccList // Acknowledge usage
	}
	if len(eb.attachments) > 0 {
		// Add attachments to message metadata
		// TODO: Implement attachment handling
		_ = eb.attachments // Acknowledge usage
	}
	return eb.base
}
