// Package webhook provides Webhook platform integration for NotifyHub
// This package implements unified platform architecture for webhook notifications
package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/pkg/config"
	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/platform"
	"github.com/kart-io/notifyhub/pkg/target"
	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// WebhookPlatform implements the unified Platform interface for webhook notifications
type WebhookPlatform struct {
	config *config.WebhookConfig
	client *http.Client
	logger logger.Logger
}

// WebhookPayload represents the structure of webhook payload
type WebhookPayload struct {
	MessageID string                 `json:"message_id"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Format    string                 `json:"format"`
	Priority  int                    `json:"priority"`
	Targets   []target.Target        `json:"targets"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewWebhookPlatform creates a new Webhook platform with strong-typed configuration
func NewWebhookPlatform(webhookConfig *config.WebhookConfig, logger logger.Logger) (platform.Platform, error) {
	if webhookConfig == nil {
		return nil, fmt.Errorf("webhook configuration cannot be nil")
	}

	// Validate required fields
	if webhookConfig.URL == "" {
		return nil, fmt.Errorf("url is required for Webhook platform")
	}

	// Set default method if not specified
	if webhookConfig.Method == "" {
		webhookConfig.Method = "POST"
	}

	// Set default content type if not specified
	if webhookConfig.ContentType == "" {
		webhookConfig.ContentType = "application/json"
	}

	// Set default timeout if not specified
	if webhookConfig.Timeout == 0 {
		webhookConfig.Timeout = 30 * time.Second
	}

	// Create HTTP client with custom transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !webhookConfig.VerifySSL,
		},
		DisableKeepAlives:  false,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   webhookConfig.Timeout,
	}

	platform := &WebhookPlatform{
		config: webhookConfig,
		client: client,
		logger: logger,
	}

	return platform, nil
}

// Name returns the platform name
func (w *WebhookPlatform) Name() string {
	return "webhook"
}

// Send sends a message to Webhook endpoint
func (w *WebhookPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, tgt := range targets {
		result := &platform.SendResult{
			Target:  tgt,
			Success: false,
		}

		// Validate target
		if err := w.ValidateTarget(tgt); err != nil {
			result.Error = err
			results[i] = result
			continue
		}

		// Build webhook payload
		payload := w.buildWebhookPayload(msg, tgt)

		// Send webhook request
		response, err := w.sendWebhookRequest(ctx, payload)
		if err != nil {
			result.Error = err
		} else {
			result.Success = true
			result.MessageID = fmt.Sprintf("webhook_%d", time.Now().UnixNano())
			result.Response = string(response)
		}

		// Metadata not supported in SendResult structure

		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates a target for Webhook
func (w *WebhookPlatform) ValidateTarget(tgt target.Target) error {
	switch tgt.Type {
	case "webhook", "url":
		// Valid target types for Webhook
	default:
		return fmt.Errorf("webhook supports webhook and url targets, got %s", tgt.Type)
	}

	if tgt.Value == "" {
		return fmt.Errorf("webhook target value cannot be empty")
	}

	return nil
}

// GetCapabilities returns Webhook platform capabilities
func (w *WebhookPlatform) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "webhook",
		SupportedTargetTypes: []string{"webhook", "url"},
		SupportedFormats:     []string{"text", "html", "markdown", "json"},
		MaxMessageSize:       10 * 1024 * 1024, // 10MB
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		RequiredSettings:     []string{"url"},
	}
}

// IsHealthy checks if Webhook endpoint is accessible
func (w *WebhookPlatform) IsHealthy(ctx context.Context) error {
	if w.config.URL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	// Create a simple HEAD request for health check
	req, err := http.NewRequestWithContext(ctx, "HEAD", w.config.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Add authentication headers if configured
	w.addAuthHeaders(req)

	// Add custom headers
	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	// Set user agent
	req.Header.Set("User-Agent", "NotifyHub-Webhook/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook health check failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Consider 2xx and 405 (Method Not Allowed) as healthy
	// since some endpoints might not support HEAD but support POST
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode == 405 {
		return nil // Method not allowed is acceptable for health check
	}

	return fmt.Errorf("webhook endpoint returned status: %d", resp.StatusCode)
}

// Close cleans up resources
func (w *WebhookPlatform) Close() error {
	if w.client != nil {
		w.client.CloseIdleConnections()
	}
	return nil
}

// buildWebhookPayload builds a webhook payload from the message
func (w *WebhookPlatform) buildWebhookPayload(msg *message.Message, tgt target.Target) *WebhookPayload {
	payload := &WebhookPayload{
		MessageID: msg.ID,
		Title:     msg.Title,
		Body:      msg.Body,
		Format:    string(msg.Format),
		Priority:  int(msg.Priority),
		Targets:   []target.Target{tgt},
		Timestamp: time.Now().Unix(),
	}

	// Add message metadata if present
	if msg.Metadata != nil {
		payload.Metadata = msg.Metadata
	}

	return payload
}

// sendWebhookRequest sends the webhook HTTP request
func (w *WebhookPlatform) sendWebhookRequest(ctx context.Context, payload *WebhookPayload) ([]byte, error) {
	// Serialize payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, w.config.Method, w.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set content type
	req.Header.Set("Content-Type", w.config.ContentType)

	// Add authentication headers
	w.addAuthHeaders(req)

	// Add custom headers
	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	// Set user agent
	req.Header.Set("User-Agent", "NotifyHub-Webhook/1.0")

	// Log request details
	if w.logger != nil {
		w.logger.Debug("Sending webhook request",
			"url", w.config.URL,
			"method", w.config.Method,
			"content_type", w.config.ContentType,
			"payload_size", len(jsonData))
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return respBody, fmt.Errorf("webhook request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	if w.logger != nil {
		w.logger.Info("Webhook request successful",
			"url", w.config.URL,
			"status", resp.StatusCode,
			"response_size", len(respBody))
	}

	return respBody, nil
}

// addAuthHeaders adds authentication headers based on configuration
func (w *WebhookPlatform) addAuthHeaders(req *http.Request) {
	switch w.config.AuthType {
	case "basic":
		// Use username:password for basic auth
		if w.config.Username != "" && w.config.Password != "" {
			req.SetBasicAuth(w.config.Username, w.config.Password)
		}
	case "bearer":
		// Use token for bearer auth
		if w.config.Token != "" {
			req.Header.Set("Authorization", "Bearer "+w.config.Token)
		}
	case "custom":
		// Use token for custom auth
		if w.config.Token != "" {
			req.Header.Set("Authorization", w.config.Token)
		}
	}
}

// NewPlatform is the factory function for creating Webhook platforms
// This function will be called by the platform registry
func NewPlatform(cfg interface{}, log logger.Logger) (platform.Platform, error) {
	webhookConfig, ok := cfg.(*config.WebhookConfig)
	if !ok {
		return nil, fmt.Errorf("invalid webhook configuration type")
	}

	return NewWebhookPlatform(webhookConfig, log)
}

// NewConfigFromMap creates WebhookConfig from a configuration map for backward compatibility
func NewConfigFromMap(cfg map[string]interface{}) (*config.WebhookConfig, error) {
	webhookConfig := &config.WebhookConfig{}

	if url, ok := cfg["url"].(string); ok {
		webhookConfig.URL = url
	}
	if method, ok := cfg["method"].(string); ok {
		webhookConfig.Method = method
	}
	if contentType, ok := cfg["content_type"].(string); ok {
		webhookConfig.ContentType = contentType
	}
	if authType, ok := cfg["auth_type"].(string); ok {
		webhookConfig.AuthType = authType
	}
	if token, ok := cfg["token"].(string); ok {
		webhookConfig.Token = token
	}
	if verifySSL, ok := cfg["verify_ssl"].(bool); ok {
		webhookConfig.VerifySSL = verifySSL
	}
	if timeout, ok := cfg["timeout"].(time.Duration); ok {
		webhookConfig.Timeout = timeout
	}
	if headers, ok := cfg["headers"].(map[string]string); ok {
		webhookConfig.Headers = headers
	}
	if maxRetries, ok := cfg["max_retries"].(int); ok {
		webhookConfig.MaxRetries = maxRetries
	}
	if rateLimit, ok := cfg["rate_limit"].(int); ok {
		webhookConfig.RateLimit = rateLimit
	}

	// Set defaults
	if webhookConfig.Method == "" {
		webhookConfig.Method = "POST"
	}
	if webhookConfig.ContentType == "" {
		webhookConfig.ContentType = "application/json"
	}
	if webhookConfig.Timeout == 0 {
		webhookConfig.Timeout = 30 * time.Second
	}
	if webhookConfig.Headers == nil {
		webhookConfig.Headers = make(map[string]string)
	}

	return webhookConfig, nil
}
