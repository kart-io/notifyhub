// Package feishu provides Feishu platform integration for NotifyHub
// This file implements the core Platform interface for Feishu notifications
package feishu

import (
	"bytes"
	"context"
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

// FeishuPlatform implements the Platform interface for Feishu webhooks
// This is the core platform implementation that coordinates all Feishu functionality
type FeishuPlatform struct {
	config    *FeishuConfig
	client    *http.Client
	auth      *AuthHandler
	messenger *MessageBuilder
	logger    logger.Logger
}

// FeishuConfig holds the configuration for Feishu platform
type FeishuConfig struct {
	WebhookURL string        `json:"webhook_url"`
	Secret     string        `json:"secret,omitempty"`
	Keywords   []string      `json:"keywords,omitempty"`
	Timeout    time.Duration `json:"timeout"`
}

// NewFeishuPlatform creates a new Feishu platform with strong-typed configuration
func NewFeishuPlatform(feishuConfig *config.FeishuConfig, logger logger.Logger) (platform.Platform, error) {
	if feishuConfig.WebhookURL == "" {
		return nil, fmt.Errorf("feishu webhook URL is required")
	}

	// Convert to internal config structure
	internalConfig := &FeishuConfig{
		WebhookURL: feishuConfig.WebhookURL,
		Secret:     feishuConfig.Secret,
		Keywords:   feishuConfig.Keywords,
		Timeout:    feishuConfig.Timeout,
	}

	// Set default timeout if not specified
	if internalConfig.Timeout == 0 {
		internalConfig.Timeout = 30 * time.Second
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: internalConfig.Timeout,
	}

	// Create specialized components
	auth := NewAuthHandler(internalConfig.Secret, internalConfig.Keywords)
	messenger := NewMessageBuilder(internalConfig, logger)

	return &FeishuPlatform{
		config:    internalConfig,
		client:    client,
		auth:      auth,
		messenger: messenger,
		logger:    logger,
	}, nil
}

// Name returns the platform name
func (f *FeishuPlatform) Name() string {
	return "feishu"
}

// Send implements the Platform interface for sending messages
func (f *FeishuPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	// Filter targets for Feishu
	for i, t := range targets {
		if !f.isFeishuTarget(t) {
			results[i] = &platform.SendResult{
				Target:  t,
				Success: false,
				Error:   fmt.Errorf("not a feishu target"),
			}
			continue
		}

		// Send to this target
		err := f.sendSingleMessage(ctx, msg, t)
		if err != nil {
			results[i] = &platform.SendResult{
				Target:  t,
				Success: false,
				Error:   err,
			}
		} else {
			messageID := msg.ID
			if messageID == "" {
				messageID = fmt.Sprintf("feishu_%d", time.Now().UnixNano())
			}
			results[i] = &platform.SendResult{
				Target:    t,
				Success:   true,
				MessageID: messageID,
			}
		}
	}

	return results, nil
}

// sendSingleMessage sends a message to a single feishu target
func (f *FeishuPlatform) sendSingleMessage(ctx context.Context, msg *message.Message, target target.Target) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Build Feishu message using the message builder
	feishuMsg, err := f.messenger.BuildMessage(msg)
	if err != nil {
		f.logger.Error("Failed to build Feishu message", "error", err)
		return fmt.Errorf("failed to build Feishu message: %w", err)
	}

	// Apply keyword processing if needed (integrating auth with message builder)
	if err := f.auth.ProcessKeywordRequirement(feishuMsg, msg, f.messenger); err != nil {
		f.logger.Error("Failed to process keyword requirement", "error", err)
		return fmt.Errorf("failed to process keyword requirement: %w", err)
	}

	// Apply authentication (signature will be added during HTTP send)
	if err := f.auth.AddAuth(feishuMsg); err != nil {
		f.logger.Error("Failed to add authentication", "error", err)
		return fmt.Errorf("failed to add authentication: %w", err)
	}

	// Send using HTTP client
	if err := f.sendToWebhook(ctx, feishuMsg); err != nil {
		f.logger.Error("Failed to send to Feishu webhook", "error", err)
		return fmt.Errorf("failed to send to Feishu webhook: %w", err)
	}

	f.logger.Info("Feishu message sent successfully", "messageID", msg.ID, "target", target.Value)
	return nil
}

// ValidateTarget implements the Platform interface
func (f *FeishuPlatform) ValidateTarget(target target.Target) error {
	if target.Type != "feishu" && target.Type != "webhook" {
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}
	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}
	return nil
}

// IsHealthy implements the Platform interface
func (f *FeishuPlatform) IsHealthy(ctx context.Context) error {
	// Simple health check - verify webhook URL is configured
	if f.config.WebhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}
	return nil
}

// sendToWebhook sends a message to the Feishu webhook
func (f *FeishuPlatform) sendToWebhook(ctx context.Context, msg *FeishuMessage) error {
	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", f.config.WebhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close implements the Platform interface
func (f *FeishuPlatform) Close() error {
	f.logger.Info("Closing Feishu platform")
	// Close HTTP client if needed
	if f.client != nil {
		f.client.CloseIdleConnections()
	}
	return nil
}

// GetCapabilities implements the Platform interface
func (f *FeishuPlatform) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "feishu",
		SupportedTargetTypes: []string{"feishu", "webhook"},
		SupportedFormats:     []string{"text", "markdown", "card", "rich_text"},
		MaxMessageSize:       4000,
	}
}

// isFeishuTarget checks if a target is relevant for Feishu
func (f *FeishuPlatform) isFeishuTarget(target target.Target) bool {
	return target.Type == "feishu" || target.Type == "webhook"
}

// NewPlatform is the factory function for creating Feishu platforms
// This function will be called by the platform registry
func NewPlatform(cfg interface{}, log logger.Logger) (platform.Platform, error) {
	feishuConfig, ok := cfg.(*config.FeishuConfig)
	if !ok {
		return nil, fmt.Errorf("invalid feishu configuration type")
	}

	return NewFeishuPlatform(feishuConfig, log)
}
