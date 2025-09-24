// Package webhook provides a generic webhook platform for NotifyHub
// This demonstrates creating flexible external platforms
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// WebhookSender implements the ExternalSender interface for generic webhooks
type WebhookSender struct {
	webhookURL      string
	method          string
	contentType     string
	headers         map[string]string
	payloadTemplate map[string]interface{}
	timeout         time.Duration
	client          *http.Client
}

// NewWebhookSender creates a new webhook sender
func NewWebhookSender(config map[string]interface{}) (platform.ExternalSender, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required for Webhook platform")
	}

	sender := &WebhookSender{
		webhookURL:  webhookURL,
		method:      "POST",
		contentType: "application/json",
		headers:     make(map[string]string),
		timeout:     30 * time.Second,
	}

	// Configure optional settings
	if method, ok := config["method"].(string); ok {
		sender.method = method
	}

	if contentType, ok := config["content_type"].(string); ok {
		sender.contentType = contentType
	}

	if headers, ok := config["headers"].(map[string]string); ok {
		sender.headers = headers
	}

	if template, ok := config["payload_template"].(map[string]interface{}); ok {
		sender.payloadTemplate = template
	}

	if timeout, ok := config["timeout"].(time.Duration); ok {
		sender.timeout = timeout
	}

	// Create HTTP client
	sender.client = &http.Client{
		Timeout: sender.timeout,
	}

	return sender, nil
}

// Name returns the platform name
func (w *WebhookSender) Name() string {
	return "webhook"
}

// Send sends a message to the webhook
func (w *WebhookSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := &platform.SendResult{
			Target:  target,
			Success: false,
		}

		// Validate target
		if err := w.ValidateTarget(target); err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		// Build webhook payload
		payload, err := w.buildPayload(msg, target)
		if err != nil {
			result.Error = fmt.Sprintf("failed to build payload: %v", err)
			results[i] = result
			continue
		}

		// Send webhook
		if err := w.sendWebhook(ctx, payload); err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.MessageID = fmt.Sprintf("webhook_%d", time.Now().UnixNano())
			result.Response = "Webhook sent successfully"
		}

		// Set metadata
		result.Metadata = map[string]interface{}{
			"duration":     time.Since(startTime).Milliseconds(),
			"method":       w.method,
			"content_type": w.contentType,
			"url":          w.webhookURL,
		}

		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates a target for webhook
func (w *WebhookSender) ValidateTarget(target platform.Target) error {
	switch target.Type {
	case "webhook", "url":
		// Valid target types for webhook
	default:
		return fmt.Errorf("webhook supports webhook and url targets, got %s", target.Type)
	}

	return nil
}

// GetCapabilities returns webhook platform capabilities
func (w *WebhookSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "webhook",
		SupportedTargetTypes: []string{"webhook", "url"},
		SupportedFormats:     []string{"text", "json"},
		MaxMessageSize:       10 * 1024 * 1024, // 10MB
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		SupportsMentions:     false,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"webhook_url"},
	}
}

// IsHealthy checks if webhook endpoint is accessible
func (w *WebhookSender) IsHealthy(ctx context.Context) error {
	if w.webhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	// Parse URL to validate
	_, err := url.Parse(w.webhookURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}

	return nil
}

// Close cleans up resources
func (w *WebhookSender) Close() error {
	return nil
}

// buildPayload builds the webhook payload
func (w *WebhookSender) buildPayload(msg *platform.Message, target platform.Target) ([]byte, error) {
	var payload interface{}

	// Check for custom webhook payload
	if customPayload, ok := msg.PlatformData["webhook_payload"]; ok {
		payload = customPayload
	} else if w.payloadTemplate != nil {
		// Use template with variable substitution
		payload = w.substituteVariables(w.payloadTemplate, msg)
	} else {
		// Default payload structure
		payload = map[string]interface{}{
			"title":     msg.Title,
			"message":   msg.Body,
			"priority":  msg.Priority,
			"format":    msg.Format,
			"metadata":  msg.Metadata,
			"variables": msg.Variables,
			"timestamp": time.Now().Unix(),
		}
	}

	// Serialize based on content type
	switch w.contentType {
	case "application/json":
		return json.Marshal(payload)
	case "application/x-www-form-urlencoded":
		return w.buildFormData(payload)
	case "text/plain":
		if str, ok := payload.(string); ok {
			return []byte(str), nil
		}
		return []byte(fmt.Sprintf("%v", payload)), nil
	default:
		return json.Marshal(payload)
	}
}

// substituteVariables replaces template variables
func (w *WebhookSender) substituteVariables(template map[string]interface{}, msg *platform.Message) map[string]interface{} {
	result := make(map[string]interface{})

	variables := map[string]interface{}{
		"title":    msg.Title,
		"message":  msg.Body,
		"body":     msg.Body,
		"format":   msg.Format,
		"priority": msg.Priority,
	}

	// Add message variables
	for k, v := range msg.Variables {
		variables[k] = v
	}

	// Add metadata
	for k, v := range msg.Metadata {
		variables[k] = v
	}

	// Substitute variables in template
	for key, value := range template {
		if str, ok := value.(string); ok {
			result[key] = w.replaceVariables(str, variables)
		} else {
			result[key] = value
		}
	}

	return result
}

// replaceVariables replaces {{variable}} patterns in strings
func (w *WebhookSender) replaceVariables(text string, variables map[string]interface{}) string {
	result := text
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// buildFormData converts payload to form data
func (w *WebhookSender) buildFormData(payload interface{}) ([]byte, error) {
	values := url.Values{}

	if payloadMap, ok := payload.(map[string]interface{}); ok {
		for key, value := range payloadMap {
			values.Set(key, fmt.Sprintf("%v", value))
		}
	} else {
		values.Set("data", fmt.Sprintf("%v", payload))
	}

	return []byte(values.Encode()), nil
}

// sendWebhook sends the webhook request
func (w *WebhookSender) sendWebhook(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, w.method, w.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set content type
	req.Header.Set("Content-Type", w.contentType)

	// Set custom headers
	for key, value := range w.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
