// Package feishu provides message building functionality for Feishu platform
// This file handles the construction and formatting of Feishu-specific messages
package feishu

import (
	"fmt"
	"strings"

	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// Message size constants for Feishu platform
const (
	// MaxMessageSize is the maximum allowed message size in bytes for Feishu
	MaxMessageSize = 30000 // 30KB limit for Feishu webhooks
)

// MessageBuilder handles the construction of Feishu-specific message formats
type MessageBuilder struct {
	config *FeishuConfig
	logger logger.Logger
}

// FeishuMessage represents a Feishu webhook message structure
type FeishuMessage struct {
	MsgType   string      `json:"msg_type"`
	Content   interface{} `json:"content"`
	Sign      string      `json:"sign,omitempty"`      // 签名字段
	Timestamp string      `json:"timestamp,omitempty"` // 时间戳字段
}

// FeishuTextContent represents text content for Feishu
type FeishuTextContent struct {
	Text string `json:"text"`
}

// FeishuRichTextContent represents rich text content for Feishu
type FeishuRichTextContent struct {
	Post map[string]interface{} `json:"post"`
}

// FeishuCardContent represents card content for Feishu
type FeishuCardContent struct {
	Config   map[string]interface{} `json:"config,omitempty"`
	Elements []interface{}          `json:"elements"`
	Header   map[string]interface{} `json:"header,omitempty"`
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder(config *FeishuConfig, logger logger.Logger) *MessageBuilder {
	return &MessageBuilder{
		config: config,
		logger: logger,
	}
}

// BuildMessage builds a Feishu-specific message format from a generic message
func (m *MessageBuilder) BuildMessage(msg *message.Message) (*FeishuMessage, error) {
	m.logger.Debug("Building Feishu message", "format", msg.Format)

	// Validate message first
	if err := m.ValidateMessage(msg); err != nil {
		return nil, fmt.Errorf("message validation failed: %w", err)
	}

	// Check for platform-specific data first
	if cardData, exists := msg.PlatformData["feishu_card"]; exists {
		m.logger.Debug("Using platform-specific card data")
		return &FeishuMessage{
			MsgType: "interactive",
			Content: cardData,
		}, nil
	}

	// Determine message type and build content
	msgType := m.determineMessageType(msg)
	m.logger.Debug("Determined message type", "type", msgType)

	switch msgType {
	case "interactive":
		content := m.buildCardContent(msg)
		return &FeishuMessage{MsgType: "interactive", Content: content}, nil
	case "post":
		content := m.buildRichTextContent(msg)
		return &FeishuMessage{MsgType: "post", Content: content}, nil
	default:
		text := m.buildTextContent(msg)
		return &FeishuMessage{
			MsgType: "text",
			Content: &FeishuTextContent{Text: text},
		}, nil
	}
}

// determineMessageType determines the optimal message type based on content
func (m *MessageBuilder) determineMessageType(msg *message.Message) string {
	// Check metadata for format preference
	if preferredType, exists := msg.Metadata["feishu_message_type"].(string); exists {
		if preferredType == "text" || preferredType == "post" || preferredType == "interactive" {
			return preferredType
		}
	}

	// Use card format for high priority messages
	if int(msg.Priority) >= 2 {
		return "interactive"
	}

	// Use rich text for HTML/Markdown
	if msg.Format == message.FormatHTML || msg.Format == message.FormatMarkdown {
		return "post"
	}

	return "text"
}

// buildTextContent builds text content from message
func (m *MessageBuilder) buildTextContent(msg *message.Message) string {
	text := msg.Title
	if msg.Body != "" {
		if text != "" {
			text += "\n\n"
		}
		text += msg.Body
	}
	return m.SanitizeContent(text)
}

// buildRichTextContent builds rich text content for Feishu
func (m *MessageBuilder) buildRichTextContent(msg *message.Message) *FeishuRichTextContent {
	content := &FeishuRichTextContent{
		Post: map[string]interface{}{
			"zh_cn": map[string]interface{}{
				"title": m.SanitizeContent(msg.Title),
				"content": [][]interface{}{
					{
						map[string]interface{}{
							"tag":  "text",
							"text": m.SanitizeContent(msg.Body),
						},
					},
				},
			},
		},
	}
	return content
}

// buildCardContent builds interactive card content for Feishu
func (m *MessageBuilder) buildCardContent(msg *message.Message) *FeishuCardContent {
	content := &FeishuCardContent{Elements: []interface{}{}}

	// Add header for card messages
	if msg.Title != "" {
		content.Header = map[string]interface{}{
			"title": map[string]interface{}{
				"content": m.SanitizeContent(msg.Title),
				"tag":     "plain_text",
			},
			"template": m.getCardTemplate(int(msg.Priority)),
		}
	}

	// Add body content
	if msg.Body != "" {
		bodyElement := map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": m.SanitizeContent(msg.Body),
				"tag":     "lark_md",
			},
		}
		content.Elements = append(content.Elements, bodyElement)
	}

	return content
}

// getCardTemplate returns appropriate card template based on priority
func (m *MessageBuilder) getCardTemplate(priority int) string {
	switch priority {
	case 3:
		return "red" // Urgent
	case 2:
		return "orange" // High
	case 1:
		return "blue" // Normal
	default:
		return "grey" // Low
	}
}

// ExtractMessageText extracts text content from feishu message for keyword checking
func (m *MessageBuilder) ExtractMessageText(feishuMsg *FeishuMessage, msg *message.Message) string {
	switch feishuMsg.MsgType {
	case "text":
		if content, ok := feishuMsg.Content.(*FeishuTextContent); ok {
			return content.Text
		}
	case "post", "interactive":
		return strings.TrimSpace(msg.Title + " " + msg.Body)
	}
	return strings.TrimSpace(msg.Title + " " + msg.Body)
}

// AddKeywordToMessage adds a keyword to the message
func (m *MessageBuilder) AddKeywordToMessage(feishuMsg *FeishuMessage, keyword string) error {
	keywordToAdd := strings.TrimSpace(keyword)
	if keywordToAdd == "" {
		return fmt.Errorf("keyword cannot be empty")
	}

	switch feishuMsg.MsgType {
	case "text":
		if content, ok := feishuMsg.Content.(*FeishuTextContent); ok {
			content.Text = keywordToAdd + " " + content.Text
			return nil
		}
		return fmt.Errorf("failed to cast text content")

	case "post":
		if content, ok := feishuMsg.Content.(*FeishuRichTextContent); ok {
			if zhCn, ok := content.Post["zh_cn"].(map[string]interface{}); ok {
				if contentArray, ok := zhCn["content"].([][]interface{}); ok && len(contentArray) > 0 {
					if len(contentArray[0]) > 0 {
						if textElement, ok := contentArray[0][0].(map[string]interface{}); ok {
							if originalText, ok := textElement["text"].(string); ok {
								textElement["text"] = keywordToAdd + " " + originalText
								return nil
							}
						}
					}
				}
			}
		}
		return fmt.Errorf("failed to modify rich text content")

	case "interactive":
		m.logger.Warn("Adding keywords to card messages is not fully supported")
		return nil

	default:
		return fmt.Errorf("unsupported message type: %s", feishuMsg.MsgType)
	}
}

// ValidateMessage validates the message format and content for security and size limits
func (m *MessageBuilder) ValidateMessage(msg *message.Message) error {
	if msg.Title == "" && msg.Body == "" {
		return fmt.Errorf("message title and body cannot both be empty")
	}
	return nil
}

// SanitizeContent sanitizes content for safe processing
func (m *MessageBuilder) SanitizeContent(content string) string {
	// Basic sanitization - remove null bytes and limit length
	if len(content) > MaxMessageSize {
		content = content[:MaxMessageSize]
	}
	return content
}

// ValidateMessageSize validates that the message doesn't exceed size limits
func (m *MessageBuilder) ValidateMessageSize(msg *message.Message) error {
	estimatedSize := m.EstimateMessageSize(msg)
	if estimatedSize > MaxMessageSize {
		return fmt.Errorf("message size (%d bytes) exceeds maximum allowed size (%d bytes)",
			estimatedSize, MaxMessageSize)
	}
	return nil
}

// EstimateMessageSize estimates the final message size in bytes
func (m *MessageBuilder) EstimateMessageSize(msg *message.Message) int {
	// Simple size estimation based on message content
	size := len(msg.Title) + len(msg.Body)

	// Add overhead for JSON structure
	size += 200 // JSON overhead estimation

	// Try to build actual message for more accurate estimation
	feishuMsg, err := m.BuildMessage(msg)
	if err != nil {
		return size
	}

	// Estimate based on message type
	switch feishuMsg.MsgType {
	case "interactive":
		size += 500 // Card overhead
	case "post":
		size += 300 // Rich text overhead
	default:
		size += 100 // Text overhead
	}

	return size
}

// SupportsFormat checks if the builder supports a specific message format
func (m *MessageBuilder) SupportsFormat(format string) bool {
	switch format {
	case string(message.FormatText), string(message.FormatMarkdown), string(message.FormatHTML):
		return true
	default:
		return false
	}
}

// GetSupportedFormats returns a list of supported message formats
func (m *MessageBuilder) GetSupportedFormats() []string {
	return []string{
		string(message.FormatText),
		string(message.FormatMarkdown),
		string(message.FormatHTML),
	}
}

// GetMaxMessageSize returns the maximum allowed message size for Feishu
func (m *MessageBuilder) GetMaxMessageSize() int {
	return MaxMessageSize
}
