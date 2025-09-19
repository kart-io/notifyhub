package email

import (
	"fmt"

	"github.com/kart-io/notifyhub/platforms/common"
)

// EmailMention implements the Mention interface for Email platform
// This provides email-specific mention functionality (e.g., reply-to, CC mentions)
type EmailMention struct {
	MentionType  common.MentionType
	EmailAddress string
	DisplayName  string
	ReplyTo      bool // Whether this is a reply-to mention
}

// Platform returns the platform name
func (em EmailMention) Platform() string {
	return "email"
}

// Type returns the mention type
func (em EmailMention) Type() common.MentionType {
	return em.MentionType
}

// Target returns the mention target
func (em EmailMention) Target() string {
	return em.EmailAddress
}

// Render renders the mention according to the message format
func (em EmailMention) Render(format string) (string, error) {
	switch format {
	case "text":
		return em.renderTextMention()
	case "html":
		return em.renderHTMLMention()
	default:
		return "", fmt.Errorf("unsupported format for email mention: %s", format)
	}
}

// renderTextMention renders a plain text mention
func (em EmailMention) renderTextMention() (string, error) {
	if em.DisplayName != "" {
		return fmt.Sprintf("%s <%s>", em.DisplayName, em.EmailAddress), nil
	}
	return em.EmailAddress, nil
}

// renderHTMLMention renders an HTML mention with mailto link
func (em EmailMention) renderHTMLMention() (string, error) {
	if em.DisplayName != "" {
		return fmt.Sprintf(`<a href="mailto:%s">%s</a>`, em.EmailAddress, em.DisplayName), nil
	}
	return fmt.Sprintf(`<a href="mailto:%s">%s</a>`, em.EmailAddress, em.EmailAddress), nil
}
