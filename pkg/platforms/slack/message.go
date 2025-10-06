// Package slack provides message building and formatting for Slack platform
package slack

import (
	"fmt"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/target"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// MessageBuilder handles building Slack-specific messages
type MessageBuilder struct {
	config *SlackConfig
	logger logger.Logger
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Text        string                 `json:"text,omitempty"`
	Channel     string                 `json:"channel,omitempty"`
	Username    string                 `json:"username,omitempty"`
	IconEmoji   string                 `json:"icon_emoji,omitempty"`
	IconURL     string                 `json:"icon_url,omitempty"`
	Blocks      []SlackBlock           `json:"blocks,omitempty"`
	Attachments []SlackAttachment      `json:"attachments,omitempty"`
	ThreadTS    string                 `json:"thread_ts,omitempty"`
	Markdown    bool                   `json:"mrkdwn,omitempty"`
	UnfurlLinks bool                   `json:"unfurl_links,omitempty"`
	UnfurlMedia bool                   `json:"unfurl_media,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SlackBlock represents a Slack block element
type SlackBlock struct {
	Type     string        `json:"type"`
	Text     *SlackText    `json:"text,omitempty"`
	BlockID  string        `json:"block_id,omitempty"`
	Elements []interface{} `json:"elements,omitempty"`
}

// SlackText represents text in Slack blocks
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackAttachment represents a Slack attachment
type SlackAttachment struct {
	Color      string                 `json:"color,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Text       string                 `json:"text,omitempty"`
	Footer     string                 `json:"footer,omitempty"`
	Timestamp  int64                  `json:"ts,omitempty"`
	Fields     []SlackField           `json:"fields,omitempty"`
	MarkdownIn []string               `json:"mrkdwn_in,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SlackField represents a field in Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackAPIResponse represents a response from Slack API
type SlackAPIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	TS    string `json:"ts,omitempty"`
}

// NewMessageBuilder creates a new Slack message builder
func NewMessageBuilder(config *SlackConfig, logger logger.Logger) *MessageBuilder {
	return &MessageBuilder{
		config: config,
		logger: logger,
	}
}

// BuildMessage converts a NotifyHub message to Slack format
func (b *MessageBuilder) BuildMessage(msg *message.Message, target target.Target) (*SlackMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	slackMsg := &SlackMessage{
		Channel:     b.config.Channel,
		Username:    b.config.Username,
		IconEmoji:   b.config.IconEmoji,
		IconURL:     b.config.IconURL,
		Markdown:    true,
		UnfurlLinks: true,
		UnfurlMedia: true,
		Metadata:    msg.Metadata,
	}

	// Override channel if target specifies one
	if target.Type == "slack" && target.Value != "" {
		slackMsg.Channel = target.Value
	}

	// Handle different message formats
	switch msg.Format {
	case message.FormatText:
		err := b.buildTextMessage(slackMsg, msg)
		if err != nil {
			return nil, err
		}
	case message.FormatMarkdown:
		err := b.buildMarkdownMessage(slackMsg, msg)
		if err != nil {
			return nil, err
		}
	case message.FormatHTML:
		// Convert HTML to Slack markdown (simplified)
		err := b.buildHTMLMessage(slackMsg, msg)
		if err != nil {
			return nil, err
		}
	default:
		// Default to text format
		err := b.buildTextMessage(slackMsg, msg)
		if err != nil {
			return nil, err
		}
	}

	// Add priority-based styling
	b.applyPriorityFormatting(slackMsg, msg)

	// Validate message size
	if err := b.validateMessage(slackMsg); err != nil {
		return nil, err
	}

	return slackMsg, nil
}

// buildTextMessage builds a simple text message
func (b *MessageBuilder) buildTextMessage(slackMsg *SlackMessage, msg *message.Message) error {
	content := ""

	// Add title if present
	if msg.Title != "" {
		content += fmt.Sprintf("*%s*\n", msg.Title)
	}

	// Add body
	content += msg.Body

	slackMsg.Text = content
	return nil
}

// buildMarkdownMessage builds a markdown-formatted message
func (b *MessageBuilder) buildMarkdownMessage(slackMsg *SlackMessage, msg *message.Message) error {
	content := ""

	// Add title if present
	if msg.Title != "" {
		content += fmt.Sprintf("*%s*\n\n", msg.Title)
	}

	// Convert common markdown to Slack format
	body := b.convertMarkdownToSlack(msg.Body)
	content += body

	slackMsg.Text = content
	return nil
}

// buildHTMLMessage builds a message from HTML (simplified conversion)
func (b *MessageBuilder) buildHTMLMessage(slackMsg *SlackMessage, msg *message.Message) error {
	// Simple HTML to Slack conversion
	content := b.convertHTMLToSlack(msg.Body)

	if msg.Title != "" {
		content = fmt.Sprintf("*%s*\n\n%s", msg.Title, content)
	}

	slackMsg.Text = content
	return nil
}

// convertMarkdownToSlack converts common markdown syntax to Slack format
func (b *MessageBuilder) convertMarkdownToSlack(content string) string {
	// Bold: **text** to *text*
	content = strings.ReplaceAll(content, "**", "*")

	// Italic: *text* to _text_
	// This is tricky because * is now bold, so we need to handle this carefully
	// For simplicity, we'll leave italics as they are in this basic implementation

	// Code: `code` stays the same
	// Code blocks: ```code``` to ```code```

	// Links: [text](url) to <url|text>
	// This is a simplified implementation

	return content
}

// convertHTMLToSlack converts basic HTML to Slack format
func (b *MessageBuilder) convertHTMLToSlack(content string) string {
	// Remove HTML tags and convert to Slack format
	// This is a very basic implementation

	// Bold
	content = strings.ReplaceAll(content, "<b>", "*")
	content = strings.ReplaceAll(content, "</b>", "*")
	content = strings.ReplaceAll(content, "<strong>", "*")
	content = strings.ReplaceAll(content, "</strong>", "*")

	// Italic
	content = strings.ReplaceAll(content, "<i>", "_")
	content = strings.ReplaceAll(content, "</i>", "_")
	content = strings.ReplaceAll(content, "<em>", "_")
	content = strings.ReplaceAll(content, "</em>", "_")

	// Code
	content = strings.ReplaceAll(content, "<code>", "`")
	content = strings.ReplaceAll(content, "</code>", "`")

	// Line breaks
	content = strings.ReplaceAll(content, "<br>", "\n")
	content = strings.ReplaceAll(content, "<br/>", "\n")
	content = strings.ReplaceAll(content, "<br />", "\n")

	// Paragraphs
	content = strings.ReplaceAll(content, "<p>", "")
	content = strings.ReplaceAll(content, "</p>", "\n")

	// Remove remaining HTML tags (basic)
	// This is not a complete HTML parser, just basic cleanup

	return content
}

// applyPriorityFormatting applies priority-based formatting to the message
func (b *MessageBuilder) applyPriorityFormatting(slackMsg *SlackMessage, msg *message.Message) {
	switch msg.Priority {
	case message.PriorityUrgent:
		// Add urgent styling with red color and warning emoji
		attachment := SlackAttachment{
			Color:      "danger",
			Title:      ":warning: URGENT",
			Text:       slackMsg.Text,
			Footer:     "NotifyHub - Urgent Priority",
			Timestamp:  time.Now().Unix(),
			MarkdownIn: []string{"text"},
		}
		slackMsg.Attachments = []SlackAttachment{attachment}
		slackMsg.Text = "" // Move text to attachment

	case message.PriorityHigh:
		// Add high priority styling with orange color
		attachment := SlackAttachment{
			Color:      "warning",
			Title:      ":exclamation: High Priority",
			Text:       slackMsg.Text,
			Footer:     "NotifyHub - High Priority",
			Timestamp:  time.Now().Unix(),
			MarkdownIn: []string{"text"},
		}
		slackMsg.Attachments = []SlackAttachment{attachment}
		slackMsg.Text = ""

	case message.PriorityLow:
		// Add low priority styling with gray color
		attachment := SlackAttachment{
			Color:      "#808080",
			Text:       slackMsg.Text,
			Footer:     "NotifyHub - Low Priority",
			Timestamp:  time.Now().Unix(),
			MarkdownIn: []string{"text"},
		}
		slackMsg.Attachments = []SlackAttachment{attachment}
		slackMsg.Text = ""

	default:
		// Normal priority - no special formatting
		if slackMsg.Text != "" {
			attachment := SlackAttachment{
				Color:      "good",
				Text:       slackMsg.Text,
				Footer:     "NotifyHub",
				Timestamp:  time.Now().Unix(),
				MarkdownIn: []string{"text"},
			}
			slackMsg.Attachments = []SlackAttachment{attachment}
			slackMsg.Text = ""
		}
	}
}

// validateMessage validates the Slack message
func (b *MessageBuilder) validateMessage(slackMsg *SlackMessage) error {
	// Check message size (Slack has limits)
	totalSize := len(slackMsg.Text)
	for _, attachment := range slackMsg.Attachments {
		totalSize += len(attachment.Text) + len(attachment.Title)
	}

	if totalSize > 4000 {
		return fmt.Errorf("message too large: %d characters (max 4000)", totalSize)
	}

	// Must have either text or attachments
	if slackMsg.Text == "" && len(slackMsg.Attachments) == 0 && len(slackMsg.Blocks) == 0 {
		return fmt.Errorf("message must have text, attachments, or blocks")
	}

	return nil
}
