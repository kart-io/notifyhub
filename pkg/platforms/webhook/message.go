// Package webhook provides webhook message building functionality for NotifyHub
package webhook

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/errors"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/target"
)

// MessageBuilder builds webhook messages
type MessageBuilder struct {
	config *Config
}

// NewMessageBuilder creates a new webhook message builder
func NewMessageBuilder(config *Config) *MessageBuilder {
	return &MessageBuilder{
		config: config,
	}
}

// Message represents a webhook message
type Message struct {
	// Standard webhook fields
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Event     string `json:"event,omitempty"`
	Source    string `json:"source,omitempty"`

	// Message content
	Title    string `json:"title,omitempty"`
	Body     string `json:"body,omitempty"`
	Format   string `json:"format,omitempty"`
	Priority int    `json:"priority,omitempty"`

	// Webhook specific
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Payload interface{}       `json:"payload,omitempty"`
	Targets []target.Target   `json:"targets,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PayloadFormat represents different payload formats
type PayloadFormat string

const (
	PayloadFormatJSON PayloadFormat = "json"
	PayloadFormatForm PayloadFormat = "form"
	PayloadFormatXML  PayloadFormat = "xml"
)

// BuildMessage builds a webhook message from a standard message
func (b *MessageBuilder) BuildMessage(msg *message.Message, targets []target.Target) (*Message, error) {
	if msg == nil {
		return nil, &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message cannot be nil",
		}
	}

	if len(targets) == 0 {
		return nil, &errors.NotifyError{
			Code:    errors.ErrInvalidTarget,
			Message: "at least one target is required",
		}
	}

	// Build webhook message
	webhookMsg := &Message{
		ID:        msg.ID,
		Timestamp: time.Now().Unix(),
		Event:     "notification",
		Source:    "notifyhub",
		Title:     msg.Title,
		Body:      msg.Body,
		Format:    string(msg.Format),
		Priority:  int(msg.Priority),
		Targets:   targets,
		Metadata:  msg.Metadata,
	}

	// Apply configuration
	if b.config.Method != "" {
		webhookMsg.Method = b.config.Method
	}

	if b.config.Headers != nil {
		webhookMsg.Headers = make(map[string]string)
		for k, v := range b.config.Headers {
			webhookMsg.Headers[k] = v
		}
	}

	// Build payload based on format
	payload, err := b.buildPayload(msg, targets)
	if err != nil {
		return nil, fmt.Errorf("failed to build payload: %w", err)
	}
	webhookMsg.Payload = payload

	// Apply field mapping if configured
	if b.config.FieldMapping != nil {
		if err := b.applyFieldMapping(webhookMsg); err != nil {
			return nil, fmt.Errorf("failed to apply field mapping: %w", err)
		}
	}

	// Validate the built message
	if err := b.ValidateMessage(webhookMsg); err != nil {
		return nil, fmt.Errorf("invalid webhook message: %w", err)
	}

	return webhookMsg, nil
}

// buildPayload builds the payload based on configured format
func (b *MessageBuilder) buildPayload(msg *message.Message, targets []target.Target) (interface{}, error) {
	if b.config.PayloadTemplate != "" {
		return b.buildFromTemplate(msg, targets)
	}

	switch PayloadFormat(b.config.PayloadFormat) {
	case PayloadFormatJSON:
		return b.buildJSONPayload(msg, targets)
	case PayloadFormatForm:
		return b.buildFormPayload(msg, targets)
	case PayloadFormatXML:
		return b.buildXMLPayload(msg, targets)
	default:
		return b.buildJSONPayload(msg, targets)
	}
}

// buildJSONPayload builds a JSON payload
func (b *MessageBuilder) buildJSONPayload(msg *message.Message, targets []target.Target) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"id":        msg.ID,
		"title":     msg.Title,
		"body":      msg.Body,
		"format":    string(msg.Format),
		"priority":  int(msg.Priority),
		"timestamp": time.Now().Unix(),
	}

	// Add targets
	if len(targets) > 0 {
		payload["targets"] = targets
	}

	// Add metadata
	if msg.Metadata != nil {
		payload["metadata"] = msg.Metadata
	}

	// Add template variables if present
	if msg.Variables != nil {
		payload["variables"] = msg.Variables
	}

	// Add webhook specific fields
	payload["event"] = "notification"
	payload["source"] = "notifyhub"

	return payload, nil
}

// buildFormPayload builds a form-encoded payload
func (b *MessageBuilder) buildFormPayload(msg *message.Message, targets []target.Target) (url.Values, error) {
	values := url.Values{}
	values.Set("id", msg.ID)
	values.Set("title", msg.Title)
	values.Set("body", msg.Body)
	values.Set("format", string(msg.Format))
	values.Set("priority", fmt.Sprintf("%d", int(msg.Priority)))
	values.Set("timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	values.Set("event", "notification")
	values.Set("source", "notifyhub")

	// Add targets as JSON string
	if len(targets) > 0 {
		targetsJSON, err := json.Marshal(targets)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal targets: %w", err)
		}
		values.Set("targets", string(targetsJSON))
	}

	// Add metadata as JSON string
	if msg.Metadata != nil {
		metadataJSON, err := json.Marshal(msg.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		values.Set("metadata", string(metadataJSON))
	}

	return values, nil
}

// buildXMLPayload builds an XML payload
func (b *MessageBuilder) buildXMLPayload(msg *message.Message, targets []target.Target) (string, error) {
	// Simple XML structure
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<notification>
	<id>%s</id>
	<title><![CDATA[%s]]></title>
	<body><![CDATA[%s]]></body>
	<format>%s</format>
	<priority>%d</priority>
	<timestamp>%d</timestamp>
	<event>notification</event>
	<source>notifyhub</source>
</notification>`,
		msg.ID, msg.Title, msg.Body, string(msg.Format), int(msg.Priority), time.Now().Unix())

	return xml, nil
}

// buildFromTemplate builds payload from custom template
func (b *MessageBuilder) buildFromTemplate(msg *message.Message, targets []target.Target) (interface{}, error) {
	// This would integrate with the template engine
	// For now, return a basic implementation
	template := b.config.PayloadTemplate

	// Replace common variables
	template = strings.ReplaceAll(template, "{{.ID}}", msg.ID)
	template = strings.ReplaceAll(template, "{{.Title}}", msg.Title)
	template = strings.ReplaceAll(template, "{{.Body}}", msg.Body)
	template = strings.ReplaceAll(template, "{{.Format}}", string(msg.Format))
	template = strings.ReplaceAll(template, "{{.Priority}}", fmt.Sprintf("%d", int(msg.Priority)))
	template = strings.ReplaceAll(template, "{{.Timestamp}}", fmt.Sprintf("%d", time.Now().Unix()))

	// Try to parse as JSON, otherwise return as string
	var result interface{}
	if err := json.Unmarshal([]byte(template), &result); err != nil {
		return template, nil
	}

	return result, nil
}

// applyFieldMapping applies field mapping to the webhook message
func (b *MessageBuilder) applyFieldMapping(webhookMsg *Message) error {
	if len(b.config.FieldMapping) == 0 {
		return nil
	}

	// This would apply field mapping transformations
	// For now, just validate that the mapping is valid
	for from, to := range b.config.FieldMapping {
		if from == "" || to == "" {
			return fmt.Errorf("invalid field mapping: empty field name")
		}
	}

	return nil
}

// ValidateMessage validates a webhook message
func (b *MessageBuilder) ValidateMessage(msg *Message) error {
	if msg == nil {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "webhook message cannot be nil",
		}
	}

	if msg.ID == "" {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "webhook message ID cannot be empty",
		}
	}

	if msg.Payload == nil {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "webhook message payload cannot be nil",
		}
	}

	// Validate payload size if configured
	if b.config != nil {
		payloadJSON, err := json.Marshal(msg.Payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload for validation: %w", err)
		}

		maxSize := 1024 * 1024 // 1MB default
		if len(payloadJSON) > maxSize {
			return &errors.NotifyError{
				Code:    errors.ErrMessageTooLarge,
				Message: fmt.Sprintf("payload size %d exceeds maximum %d", len(payloadJSON), maxSize),
			}
		}
	}

	return nil
}

// GetSupportedFormats returns supported payload formats
func (b *MessageBuilder) GetSupportedFormats() []string {
	return []string{
		string(PayloadFormatJSON),
		string(PayloadFormatForm),
		string(PayloadFormatXML),
	}
}

// Clone creates a copy of the message builder
func (b *MessageBuilder) Clone() *MessageBuilder {
	return &MessageBuilder{
		config: b.config.Clone(),
	}
}

// SetConfig updates the builder configuration
func (b *MessageBuilder) SetConfig(config *Config) {
	b.config = config
}

// GetConfig returns the current configuration
func (b *MessageBuilder) GetConfig() *Config {
	return b.config
}

// MessageMetrics represents webhook message metrics
type MessageMetrics struct {
	TotalMessages   int   `json:"total_messages"`
	SuccessfulSends int   `json:"successful_sends"`
	FailedSends     int   `json:"failed_sends"`
	AverageSize     int   `json:"average_size"`
	LastSentAt      int64 `json:"last_sent_at,omitempty"`
}

// GetMetrics returns message building metrics
func (b *MessageBuilder) GetMetrics() *MessageMetrics {
	// This would integrate with the metrics system
	return &MessageMetrics{
		TotalMessages:   0,
		SuccessfulSends: 0,
		FailedSends:     0,
		AverageSize:     0,
	}
}
