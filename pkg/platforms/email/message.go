// Package email provides email message building functionality for NotifyHub
package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/target"
)

// Message represents an email message
type Message struct {
	// Headers
	From    string            `json:"from"`
	To      []string          `json:"to"`
	CC      []string          `json:"cc,omitempty"`
	BCC     []string          `json:"bcc,omitempty"`
	ReplyTo string            `json:"reply_to,omitempty"`
	Subject string            `json:"subject"`
	Headers map[string]string `json:"headers,omitempty"`

	// Content
	TextBody string `json:"text_body,omitempty"`
	HTMLBody string `json:"html_body,omitempty"`

	// Attachments
	Attachments []Attachment `json:"attachments,omitempty"`

	// Metadata
	MessageID  string                 `json:"message_id,omitempty"`
	References string                 `json:"references,omitempty"`
	InReplyTo  string                 `json:"in_reply_to,omitempty"`
	Priority   message.Priority       `json:"priority"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`

	// Tracking
	TrackOpens  bool `json:"track_opens,omitempty"`
	TrackClicks bool `json:"track_clicks,omitempty"`

	// Delivery options
	DeliveryReceipt bool      `json:"delivery_receipt,omitempty"`
	ReadReceipt     bool      `json:"read_receipt,omitempty"`
	Date            time.Time `json:"date"`
}

// Attachment represents an email attachment
type Attachment struct {
	Name        string            `json:"name"`
	ContentType string            `json:"content_type"`
	Content     []byte            `json:"content"`
	Inline      bool              `json:"inline,omitempty"`
	ContentID   string            `json:"content_id,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// MessageBuilder builds email messages from NotifyHub messages
type MessageBuilder struct {
	config *Config
}

// NewMessageBuilder creates a new email message builder
func NewMessageBuilder(config *Config) *MessageBuilder {
	return &MessageBuilder{
		config: config,
	}
}

// BuildMessage builds an email message from a NotifyHub message
func (b *MessageBuilder) BuildMessage(msg *message.Message, targets []target.Target) (*Message, error) {
	emailMsg := &Message{
		From:     b.config.GetFromAddress(),
		Subject:  msg.Title,
		Priority: msg.Priority,
		Date:     time.Now(),
		Headers:  make(map[string]string),
		Metadata: make(map[string]interface{}),
	}

	// Set message ID
	if msg.ID != "" {
		emailMsg.MessageID = fmt.Sprintf("<%s@%s>", msg.ID, b.extractDomain(b.config.From))
	}

	// Extract email addresses from targets
	if err := b.setRecipients(emailMsg, targets); err != nil {
		return nil, err
	}

	// Set content based on message format
	if err := b.setContent(emailMsg, msg); err != nil {
		return nil, err
	}

	// Set headers
	b.setHeaders(emailMsg, msg)

	// Set tracking options
	b.setTrackingOptions(emailMsg)

	// Process platform-specific data
	if err := b.processPlatformData(emailMsg, msg); err != nil {
		return nil, err
	}

	return emailMsg, nil
}

// setRecipients extracts email addresses from targets
func (b *MessageBuilder) setRecipients(emailMsg *Message, targets []target.Target) error {
	for _, target := range targets {
		switch target.Type {
		case "email":
			emailMsg.To = append(emailMsg.To, target.Value)
		case "cc":
			emailMsg.CC = append(emailMsg.CC, target.Value)
		case "bcc":
			emailMsg.BCC = append(emailMsg.BCC, target.Value)
		default:
			// Try to detect if it's an email address
			if strings.Contains(target.Value, "@") {
				emailMsg.To = append(emailMsg.To, target.Value)
			}
		}
	}

	if len(emailMsg.To) == 0 && len(emailMsg.CC) == 0 && len(emailMsg.BCC) == 0 {
		return fmt.Errorf("no valid email recipients found")
	}

	return nil
}

// setContent sets email content based on message format
func (b *MessageBuilder) setContent(emailMsg *Message, msg *message.Message) error {
	switch msg.Format {
	case message.FormatHTML:
		emailMsg.HTMLBody = msg.Body
		// Generate text version from HTML
		emailMsg.TextBody = b.htmlToText(msg.Body)

	case message.FormatMarkdown:
		// Convert markdown to HTML
		htmlBody, err := b.markdownToHTML(msg.Body)
		if err != nil {
			return fmt.Errorf("failed to convert markdown to HTML: %w", err)
		}
		emailMsg.HTMLBody = htmlBody
		emailMsg.TextBody = msg.Body // Keep original markdown as text

	default: // FormatText
		emailMsg.TextBody = msg.Body
		// Create simple HTML version
		emailMsg.HTMLBody = b.textToHTML(msg.Body)
	}

	return nil
}

// setHeaders sets email headers
func (b *MessageBuilder) setHeaders(emailMsg *Message, msg *message.Message) {
	// Set default headers
	emailMsg.Headers["X-Mailer"] = "NotifyHub"
	emailMsg.Headers["X-Priority"] = b.priorityToHeader(msg.Priority)

	// Set reply-to if configured
	if b.config.ReplyTo != "" {
		emailMsg.ReplyTo = b.config.ReplyTo
	}

	// Copy configured headers
	for k, v := range b.config.Headers {
		emailMsg.Headers[k] = v
	}

	// Set metadata as headers (with prefix)
	for k, v := range msg.Metadata {
		if str, ok := v.(string); ok {
			emailMsg.Headers[fmt.Sprintf("X-NotifyHub-%s", k)] = str
		}
	}
}

// setTrackingOptions sets tracking options
func (b *MessageBuilder) setTrackingOptions(emailMsg *Message) {
	emailMsg.TrackOpens = b.config.TrackOpens
	emailMsg.TrackClicks = b.config.TrackClicks
}

// processPlatformData processes platform-specific data
func (b *MessageBuilder) processPlatformData(emailMsg *Message, msg *message.Message) error {
	platformData, exists := msg.PlatformData["email"]
	if !exists {
		return nil
	}

	data, ok := platformData.(map[string]interface{})
	if !ok {
		return nil
	}

	// Process attachments
	if attachments, exists := data["attachments"]; exists {
		if err := b.processAttachments(emailMsg, attachments); err != nil {
			return err
		}
	}

	// Process custom headers
	if headers, exists := data["headers"]; exists {
		if headerMap, ok := headers.(map[string]interface{}); ok {
			for k, v := range headerMap {
				if str, ok := v.(string); ok {
					emailMsg.Headers[k] = str
				}
			}
		}
	}

	// Process delivery options
	if deliveryReceipt, exists := data["delivery_receipt"]; exists {
		if receipt, ok := deliveryReceipt.(bool); ok {
			emailMsg.DeliveryReceipt = receipt
		}
	}

	if readReceipt, exists := data["read_receipt"]; exists {
		if receipt, ok := readReceipt.(bool); ok {
			emailMsg.ReadReceipt = receipt
		}
	}

	return nil
}

// extractAttachmentString safely extracts a string from attachment data
func extractAttachmentString(data map[string]interface{}, key string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// extractAttachmentBool safely extracts a bool from attachment data
func extractAttachmentBool(data map[string]interface{}, key string) bool {
	if value, exists := data[key]; exists {
		if boolVal, ok := value.(bool); ok {
			return boolVal
		}
	}
	return false
}

// extractAttachmentContent safely extracts content from attachment data
func extractAttachmentContent(data map[string]interface{}, key string) ([]byte, error) {
	value, exists := data[key]
	if !exists {
		return nil, nil
	}

	// Try []byte directly
	if bytes, ok := value.([]byte); ok {
		return bytes, nil
	}

	// Try base64 encoded string
	if str, ok := value.(string); ok {
		decoded, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return nil, fmt.Errorf("failed to decode attachment content: %w", err)
		}
		return decoded, nil
	}

	return nil, nil
}

// detectContentType detects content type from filename if not provided
func detectContentType(contentType, filename string) string {
	if contentType != "" {
		return contentType
	}

	if filename != "" {
		detected := mime.TypeByExtension(filepath.Ext(filename))
		if detected != "" {
			return detected
		}
	}

	return "application/octet-stream"
}

// processAttachments processes email attachments
func (b *MessageBuilder) processAttachments(emailMsg *Message, attachments interface{}) error {
	attachmentList, ok := attachments.([]interface{})
	if !ok {
		return nil
	}

	for _, attachment := range attachmentList {
		attachmentData, ok := attachment.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract attachment content
		content, err := extractAttachmentContent(attachmentData, "content")
		if err != nil {
			return err
		}

		// Build attachment struct
		att := Attachment{
			Name:        extractAttachmentString(attachmentData, "name"),
			ContentType: extractAttachmentString(attachmentData, "content_type"),
			Content:     content,
			Inline:      extractAttachmentBool(attachmentData, "inline"),
			ContentID:   extractAttachmentString(attachmentData, "content_id"),
			Headers:     make(map[string]string),
		}

		// Detect content type if not provided
		att.ContentType = detectContentType(att.ContentType, att.Name)

		emailMsg.Attachments = append(emailMsg.Attachments, att)
	}

	return nil
}

// Helper methods

// extractDomain extracts domain from email address
func (b *MessageBuilder) extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return "localhost"
}

// priorityToHeader converts message priority to email header value
func (b *MessageBuilder) priorityToHeader(priority message.Priority) string {
	switch priority {
	case message.PriorityUrgent:
		return "1"
	case message.PriorityHigh:
		return "2"
	case message.PriorityNormal:
		return "3"
	case message.PriorityLow:
		return "4"
	default:
		return "3"
	}
}

// htmlToText converts HTML to plain text (simplified)
func (b *MessageBuilder) htmlToText(html string) string {
	// Simple HTML to text conversion
	text := html
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = strings.ReplaceAll(text, "<p>", "")
	text = strings.ReplaceAll(text, "</p>", "\n\n")
	text = strings.ReplaceAll(text, "<div>", "")
	text = strings.ReplaceAll(text, "</div>", "\n")

	// Remove HTML tags (basic)
	for strings.Contains(text, "<") && strings.Contains(text, ">") {
		start := strings.Index(text, "<")
		end := strings.Index(text[start:], ">")
		if end == -1 {
			break
		}
		text = text[:start] + text[start+end+1:]
	}

	return strings.TrimSpace(text)
}

// textToHTML converts plain text to HTML
func (b *MessageBuilder) textToHTML(text string) string {
	html := template.HTMLEscapeString(text)
	html = strings.ReplaceAll(html, "\n", "<br>")
	return fmt.Sprintf("<html><body>%s</body></html>", html)
}

// markdownToHTML converts markdown to HTML (basic implementation)
func (b *MessageBuilder) markdownToHTML(markdown string) (string, error) {
	// Basic markdown to HTML conversion
	html := template.HTMLEscapeString(markdown)

	// Bold
	html = strings.ReplaceAll(html, "**", "<strong>")
	html = strings.ReplaceAll(html, "__", "<strong>")

	// Italic
	html = strings.ReplaceAll(html, "*", "<em>")
	html = strings.ReplaceAll(html, "_", "<em>")

	// Line breaks
	html = strings.ReplaceAll(html, "\n", "<br>")

	return fmt.Sprintf("<html><body>%s</body></html>", html), nil
}

// ToRFC2822 converts the email message to RFC2822 format
func (m *Message) ToRFC2822() ([]byte, error) {
	var buf bytes.Buffer

	// Write headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", m.From))

	if len(m.To) > 0 {
		buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(m.To, ", ")))
	}

	if len(m.CC) > 0 {
		buf.WriteString(fmt.Sprintf("CC: %s\r\n", strings.Join(m.CC, ", ")))
	}

	if m.ReplyTo != "" {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", m.ReplyTo))
	}

	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", m.Subject))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", m.Date.Format(time.RFC1123Z)))

	if m.MessageID != "" {
		buf.WriteString(fmt.Sprintf("Message-ID: %s\r\n", m.MessageID))
	}

	// Write custom headers
	for k, v := range m.Headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	// MIME headers for multipart message
	boundary := fmt.Sprintf("boundary_%d", time.Now().Unix())
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(m.Attachments) > 0 {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	} else {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	}

	buf.WriteString("\r\n")

	// Write body parts
	if m.TextBody != "" {
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(m.TextBody)
		buf.WriteString("\r\n\r\n")
	}

	if m.HTMLBody != "" {
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(m.HTMLBody)
		buf.WriteString("\r\n\r\n")
	}

	// Write attachments
	for _, attachment := range m.Attachments {
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", attachment.ContentType))
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")

		if attachment.Name != "" {
			disposition := "attachment"
			if attachment.Inline {
				disposition = "inline"
			}
			buf.WriteString(fmt.Sprintf("Content-Disposition: %s; filename=\"%s\"\r\n", disposition, attachment.Name))
		}

		if attachment.ContentID != "" {
			buf.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", attachment.ContentID))
		}

		// Write custom attachment headers
		for k, v := range attachment.Headers {
			buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}

		buf.WriteString("\r\n")

		// Encode attachment content as base64
		encoded := base64.StdEncoding.EncodeToString(attachment.Content)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			buf.WriteString(encoded[i:end])
			buf.WriteString("\r\n")
		}
		buf.WriteString("\r\n")
	}

	// End boundary
	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return buf.Bytes(), nil
}

// Validate validates the email message
func (m *Message) Validate() error {
	if m.From == "" {
		return fmt.Errorf("from address is required")
	}

	if len(m.To) == 0 && len(m.CC) == 0 && len(m.BCC) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if m.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if m.TextBody == "" && m.HTMLBody == "" {
		return fmt.Errorf("message body is required")
	}

	return nil
}

// GetAllRecipients returns all recipients (To, CC, BCC)
func (m *Message) GetAllRecipients() []string {
	var recipients []string
	recipients = append(recipients, m.To...)
	recipients = append(recipients, m.CC...)
	recipients = append(recipients, m.BCC...)
	return recipients
}

// HasAttachments returns true if the message has attachments
func (m *Message) HasAttachments() bool {
	return len(m.Attachments) > 0
}

// GetSize returns the approximate size of the message in bytes
func (m *Message) GetSize() int {
	size := len(m.Subject) + len(m.TextBody) + len(m.HTMLBody)
	for _, attachment := range m.Attachments {
		size += len(attachment.Content)
	}
	return size
}
