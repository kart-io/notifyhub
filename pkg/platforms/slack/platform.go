// Package slack provides Slack platform integration for NotifyHub
// This file implements the core Platform interface for Slack notifications
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/config"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/platform"
	"github.com/kart/notifyhub/pkg/target"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// SlackPlatform implements the Platform interface for Slack notifications
// This is the core platform implementation that coordinates all Slack functionality
type SlackPlatform struct {
	config    *SlackConfig
	client    *http.Client
	messenger *MessageBuilder
	logger    logger.Logger
}

// SlackConfig holds the configuration for Slack platform
type SlackConfig struct {
	WebhookURL string        `json:"webhook_url"`
	Token      string        `json:"token"`
	Channel    string        `json:"channel"`
	Username   string        `json:"username"`
	IconEmoji  string        `json:"icon_emoji"`
	IconURL    string        `json:"icon_url"`
	Timeout    time.Duration `json:"timeout"`
}

// NewSlackPlatform creates a new Slack platform with strong-typed configuration
func NewSlackPlatform(slackConfig *config.SlackConfig, logger logger.Logger) (platform.Platform, error) {
	if slackConfig.WebhookURL == "" && slackConfig.Token == "" {
		return nil, fmt.Errorf("slack webhook URL or token is required")
	}

	// Convert to internal config structure
	internalConfig := &SlackConfig{
		WebhookURL: slackConfig.WebhookURL,
		Token:      slackConfig.Token,
		Channel:    slackConfig.Channel,
		Username:   slackConfig.Username,
		IconEmoji:  slackConfig.IconEmoji,
		IconURL:    slackConfig.IconURL,
		Timeout:    slackConfig.Timeout,
	}

	// Set default timeout if not specified
	if internalConfig.Timeout == 0 {
		internalConfig.Timeout = 30 * time.Second
	}

	// Set default channel if not specified
	if internalConfig.Channel == "" {
		internalConfig.Channel = "#general"
	}

	// Set default username if not specified
	if internalConfig.Username == "" {
		internalConfig.Username = "NotifyHub"
	}

	// Set default icon if not specified
	if internalConfig.IconEmoji == "" && internalConfig.IconURL == "" {
		internalConfig.IconEmoji = ":bell:"
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: internalConfig.Timeout,
	}

	// Create specialized components
	messenger := NewMessageBuilder(internalConfig, logger)

	return &SlackPlatform{
		config:    internalConfig,
		client:    client,
		messenger: messenger,
		logger:    logger,
	}, nil
}

// Name returns the platform name
func (s *SlackPlatform) Name() string {
	return "slack"
}

// Send implements the Platform interface for sending messages
func (s *SlackPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	// Filter targets for Slack
	for i, t := range targets {
		if !s.isSlackTarget(t) {
			results[i] = &platform.SendResult{
				Target:  t,
				Success: false,
				Error:   fmt.Errorf("not a slack target"),
			}
			continue
		}

		// Send to this target
		err := s.sendSingleMessage(ctx, msg, t)
		if err != nil {
			results[i] = &platform.SendResult{
				Target:  t,
				Success: false,
				Error:   err,
			}
		} else {
			messageID := msg.ID
			if messageID == "" {
				messageID = fmt.Sprintf("slack_%d", time.Now().UnixNano())
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

// sendSingleMessage sends a message to a single slack target
func (s *SlackPlatform) sendSingleMessage(ctx context.Context, msg *message.Message, target target.Target) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Build Slack message using the message builder
	slackMsg, err := s.messenger.BuildMessage(msg, target)
	if err != nil {
		s.logger.Error("Failed to build Slack message", "error", err)
		return fmt.Errorf("failed to build Slack message: %w", err)
	}

	// Send using the appropriate method
	if s.config.Token != "" {
		// Use Slack API
		return s.sendToAPI(ctx, slackMsg, target)
	} else if s.config.WebhookURL != "" {
		// Use Slack webhook
		return s.sendToWebhook(ctx, slackMsg)
	}

	return fmt.Errorf("no valid sending method configured")
}

// ValidateTarget implements the Platform interface
func (s *SlackPlatform) ValidateTarget(target target.Target) error {
	if target.Type != "slack" && target.Type != "webhook" {
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}
	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}
	// Validate channel format
	if target.Type == "slack" {
		channel := target.Value
		if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "@") &&
			!strings.HasPrefix(channel, "C") && !strings.HasPrefix(channel, "D") {
			return fmt.Errorf("slack channel must start with # (public), @ (user), C (channel ID), or D (DM ID)")
		}
	}
	return nil
}

// IsHealthy implements the Platform interface
func (s *SlackPlatform) IsHealthy(ctx context.Context) error {
	// Simple health check - verify configuration
	if s.config.WebhookURL == "" && s.config.Token == "" {
		return fmt.Errorf("no webhook URL or token configured")
	}
	return nil
}

// sendToWebhook sends a message to the Slack webhook
func (s *SlackPlatform) sendToWebhook(ctx context.Context, msg *SlackMessage) error {
	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.WebhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
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

// sendToAPI sends a message to the Slack API
func (s *SlackPlatform) sendToAPI(ctx context.Context, msg *SlackMessage, target target.Target) error {
	// Override channel if target specifies one
	if target.Type == "slack" && target.Value != "" {
		msg.Channel = target.Value
	}

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request to Slack API
	req, err := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.Token)

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to check for API errors
	var apiResp SlackAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode API response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("slack API error: %s", apiResp.Error)
	}

	return nil
}

// Close implements the Platform interface
func (s *SlackPlatform) Close() error {
	s.logger.Info("Closing Slack platform")
	// Close HTTP client if needed
	if s.client != nil {
		s.client.CloseIdleConnections()
	}
	return nil
}

// GetCapabilities implements the Platform interface
func (s *SlackPlatform) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "slack",
		SupportedTargetTypes: []string{"slack", "webhook"},
		SupportedFormats:     []string{"text", "markdown", "blocks"},
		MaxMessageSize:       4000,
	}
}

// isSlackTarget checks if a target is relevant for Slack
func (s *SlackPlatform) isSlackTarget(target target.Target) bool {
	return target.Type == "slack" || target.Type == "webhook"
}

// NewPlatform is the factory function for creating Slack platforms
// This function will be called by the platform registry
func NewPlatform(cfg interface{}, log logger.Logger) (platform.Platform, error) {
	slackConfig, ok := cfg.(*config.SlackConfig)
	if !ok {
		return nil, fmt.Errorf("invalid slack configuration type")
	}

	return NewSlackPlatform(slackConfig, log)
}
