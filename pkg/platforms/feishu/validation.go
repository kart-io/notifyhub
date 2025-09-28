// Package feishu provides message validation functionality for Feishu platform
// This file handles validation and security checking of Feishu messages
package feishu

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

// Message validation constants
const (
	// MaxTitleLength is the maximum title length in characters
	MaxTitleLength = 500

	// MaxBodyLength is the maximum body length in characters
	MaxBodyLength = 20000
)

var (
	// Potentially dangerous patterns that should be escaped or rejected
	dangerousPatterns = []*regexp.Regexp{
		regexp.MustCompile(`<script[^>]*>.*?</script>`), // Script tags
		regexp.MustCompile(`javascript:`),               // JavaScript URLs
		regexp.MustCompile(`on\w+\s*=`),                 // Event handlers
	}
)

// MessageValidator handles validation and security checking for Feishu messages
type MessageValidator struct {
	logger logger.Logger
}

// NewMessageValidator creates a new message validator
func NewMessageValidator(logger logger.Logger) *MessageValidator {
	return &MessageValidator{
		logger: logger,
	}
}

// ValidateMessage validates the message format and content for security and size limits
func (v *MessageValidator) ValidateMessage(msg *message.Message) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Validate message ID
	if strings.TrimSpace(msg.ID) == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	// Validate title length
	if utf8.RuneCountInString(msg.Title) > MaxTitleLength {
		return fmt.Errorf("title exceeds maximum length of %d characters", MaxTitleLength)
	}

	// Validate body length
	if utf8.RuneCountInString(msg.Body) > MaxBodyLength {
		return fmt.Errorf("body exceeds maximum length of %d characters", MaxBodyLength)
	}

	// Validate overall message size
	totalContent := msg.Title + msg.Body
	if len([]byte(totalContent)) > MaxMessageSize {
		return fmt.Errorf("message exceeds maximum size of %d bytes", MaxMessageSize)
	}

	// Security validation
	if err := v.validateSecurity(msg); err != nil {
		return fmt.Errorf("security validation failed: %w", err)
	}

	// Format-specific validation
	if err := v.validateFormat(msg); err != nil {
		return fmt.Errorf("format validation failed: %w", err)
	}

	return nil
}

// validateSecurity performs security checks on message content
func (v *MessageValidator) validateSecurity(msg *message.Message) error {
	content := msg.Title + " " + msg.Body

	// Check for dangerous patterns
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(content) {
			v.logger.Warn("Potentially dangerous content detected",
				"pattern", pattern.String(), "messageID", msg.ID)
			return fmt.Errorf("message contains potentially dangerous content")
		}
	}

	// Check for excessive special characters that might indicate injection attempts
	specialCharCount := strings.Count(content, "<") + strings.Count(content, ">") +
		strings.Count(content, "&") + strings.Count(content, "\"") +
		strings.Count(content, "'")

	if len(content) > 0 && float64(specialCharCount)/float64(len(content)) > 0.3 {
		v.logger.Warn("High ratio of special characters detected",
			"ratio", float64(specialCharCount)/float64(len(content)), "messageID", msg.ID)
	}

	return nil
}

// validateFormat validates message format consistency
func (v *MessageValidator) validateFormat(msg *message.Message) error {
	switch msg.Format {
	case message.FormatText:
		// Text format should not contain HTML tags
		if strings.Contains(msg.Title, "<") || strings.Contains(msg.Body, "<") {
			return fmt.Errorf("text format should not contain HTML tags")
		}
	case message.FormatHTML:
		// HTML format should be properly escaped
		if !v.isValidHTML(msg.Title) || !v.isValidHTML(msg.Body) {
			return fmt.Errorf("invalid HTML format detected")
		}
	case message.FormatMarkdown:
		// Markdown format basic validation
		if err := v.validateMarkdown(msg); err != nil {
			return err
		}
	}
	return nil
}

// isValidHTML checks if content is valid HTML (basic check)
func (v *MessageValidator) isValidHTML(content string) bool {
	if content == "" {
		return true
	}

	// Count opening and closing tags
	openTags := strings.Count(content, "<")
	closeTags := strings.Count(content, ">")

	// Basic validation: should have matching tag counts
	return openTags == closeTags
}

// validateMarkdown performs basic markdown validation
func (v *MessageValidator) validateMarkdown(msg *message.Message) error {
	content := msg.Title + " " + msg.Body

	// Check for unmatched markdown syntax
	if strings.Count(content, "```")%2 != 0 {
		return fmt.Errorf("unmatched code block syntax in markdown")
	}

	if strings.Count(content, "`")%2 != 0 {
		return fmt.Errorf("unmatched inline code syntax in markdown")
	}

	return nil
}

// SanitizeContent sanitizes content for safe processing
func (v *MessageValidator) SanitizeContent(content string) string {
	// HTML escape dangerous characters
	sanitized := html.EscapeString(content)

	// Remove or escape potentially dangerous patterns
	for _, pattern := range dangerousPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, "")
	}

	return sanitized
}

// EstimateMessageSize estimates the final message size in bytes
func (v *MessageValidator) EstimateMessageSize(msg *message.Message, msgType string) int {
	// Estimate JSON size (rough calculation)
	baseSize := len(`{"msg_type":"","content":{}}`)
	contentSize := 0

	switch msgType {
	case "text":
		contentSize = len([]byte(msg.Title+msg.Body)) + len(`"text":""`)
	case "post":
		// Rich text is larger due to structure
		contentSize = len([]byte(msg.Title+msg.Body)) * 3 // Rough multiplier for JSON structure
	case "interactive":
		// Card messages are typically larger
		contentSize = len([]byte(msg.Title+msg.Body)) * 4
	}

	return baseSize + contentSize
}
