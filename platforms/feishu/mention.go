package feishu

import (
	"fmt"

	"github.com/kart-io/notifyhub/platforms/common"
)

// FeishuMention implements the Mention interface for Feishu platform
type FeishuMention struct {
	MentionType common.MentionType
	UserID      string
	UserName    string
	Strict      bool // Whether to enforce valid user
}

// Platform returns the platform name
func (fm FeishuMention) Platform() string {
	return "feishu"
}

// Type returns the mention type
func (fm FeishuMention) Type() common.MentionType {
	return fm.MentionType
}

// Target returns the mention target
func (fm FeishuMention) Target() string {
	switch fm.MentionType {
	case common.MentionTypeUser:
		return fm.UserID
	case common.MentionTypeAll:
		return "all"
	default:
		return ""
	}
}

// Render renders the mention according to the message format
func (fm FeishuMention) Render(format string) (string, error) {
	switch fm.MentionType {
	case common.MentionTypeUser:
		return fm.renderUserMention(format)
	case common.MentionTypeAll:
		return fm.renderAllMention(format)
	default:
		return "", fmt.Errorf("unsupported mention type: %s", fm.MentionType)
	}
}

// renderUserMention renders a user mention
func (fm FeishuMention) renderUserMention(format string) (string, error) {
	switch format {
	case "text":
		return fmt.Sprintf("@%s", fm.UserName), nil
	case "markdown":
		return fmt.Sprintf("<at user_id=\"%s\">%s</at>", fm.UserID, fm.UserName), nil
	case "card":
		// Card format uses structured data
		return fmt.Sprintf("{\"tag\":\"at\",\"user_id\":\"%s\",\"user_name\":\"%s\"}", fm.UserID, fm.UserName), nil
	default:
		return "", fmt.Errorf("unsupported format for user mention: %s", format)
	}
}

// renderAllMention renders an @all mention
func (fm FeishuMention) renderAllMention(format string) (string, error) {
	switch format {
	case "text":
		return "@所有人", nil
	case "markdown":
		return "<at user_id=\"all\">所有人</at>", nil
	case "card":
		return "{\"tag\":\"at\",\"user_id\":\"all\"}", nil
	default:
		return "", fmt.Errorf("unsupported format for all mention: %s", format)
	}
}
