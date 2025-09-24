// Package slack provides Slack platform integration for NotifyHub
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// SlackSender implements the ExternalSender interface for Slack
type SlackSender struct {
	webhookURL string
	timeout    time.Duration
	client     *http.Client
}

// SlackMessage represents a Slack webhook message
type SlackMessage struct {
	Text        string       `json:"text,omitempty"`
	Blocks      []SlackBlock `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
}

// SlackBlock represents a Slack block element
type SlackBlock struct {
	Type string     `json:"type"`
	Text *SlackText `json:"text,omitempty"`
}

// SlackText represents text content in Slack blocks
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Attachment represents a Slack attachment
type Attachment struct {
	Color  string  `json:"color,omitempty"`
	Title  string  `json:"title,omitempty"`
	Text   string  `json:"text,omitempty"`
	Fields []Field `json:"fields,omitempty"`
}

// Field represents a field in Slack attachments
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackSender creates a new Slack sender
func NewSlackSender(config map[string]interface{}) (platform.ExternalSender, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required for Slack platform")
	}

	timeout := 30 * time.Second
	if t, ok := config["timeout"].(time.Duration); ok {
		timeout = t
	}

	return &SlackSender{
		webhookURL: webhookURL,
		timeout:    timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Name returns the platform name
func (s *SlackSender) Name() string {
	return "slack"
}

// Send sends a message to Slack
func (s *SlackSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := &platform.SendResult{
			Target:  target,
			Success: false,
		}

		// Validate target
		if err := s.ValidateTarget(target); err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		// Build Slack message
		slackMsg := s.buildSlackMessage(msg, target)

		// Send message
		if err := s.sendToSlack(ctx, slackMsg); err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.MessageID = fmt.Sprintf("slack_%d", time.Now().UnixNano())
			result.Response = "Message sent successfully"
		}

		// Set metadata
		result.Metadata = map[string]interface{}{
			"duration": time.Since(startTime).Milliseconds(),
		}

		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates a target for Slack
func (s *SlackSender) ValidateTarget(target platform.Target) error {
	if target.Type != "channel" && target.Type != "user" && target.Type != "webhook" {
		return fmt.Errorf("slack supports channel, user, and webhook targets, got %s", target.Type)
	}

	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}

	return nil
}

// GetCapabilities returns Slack platform capabilities
func (s *SlackSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "slack",
		SupportedTargetTypes: []string{"channel", "user", "webhook"},
		SupportedFormats:     []string{"text", "markdown"},
		MaxMessageSize:       40000, // Slack's message limit
		SupportsScheduling:   false,
		SupportsAttachments:  true,
		SupportsMentions:     true,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"webhook_url"},
	}
}

// IsHealthy checks if Slack webhook is accessible
func (s *SlackSender) IsHealthy(ctx context.Context) error {
	// Simple health check - just verify webhook URL format
	if s.webhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	// You could also send a test message or ping the webhook
	return nil
}

// Close cleans up resources
func (s *SlackSender) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

// buildSlackMessage builds a Slack message from the platform message
func (s *SlackSender) buildSlackMessage(msg *platform.Message, target platform.Target) *SlackMessage {
	slackMsg := &SlackMessage{}

	// Set channel if specified
	if target.Type == "channel" {
		slackMsg.Channel = target.Value
	}

	// Check for rich content in platform data
	if blocks, ok := msg.PlatformData["slack_blocks"]; ok {
		if slackBlocks, ok := blocks.([]SlackBlock); ok {
			slackMsg.Blocks = slackBlocks
		}
	} else if attachments, ok := msg.PlatformData["slack_attachments"]; ok {
		if slackAttachments, ok := attachments.([]Attachment); ok {
			slackMsg.Attachments = slackAttachments
		}
	} else {
		// Simple text message
		text := msg.Body
		if msg.Title != "" {
			text = fmt.Sprintf("*%s*\n%s", msg.Title, msg.Body)
		}
		slackMsg.Text = text
	}

	// Set additional options from platform data
	if username, ok := msg.PlatformData["slack_username"].(string); ok {
		slackMsg.Username = username
	}
	if iconEmoji, ok := msg.PlatformData["slack_icon_emoji"].(string); ok {
		slackMsg.IconEmoji = iconEmoji
	}

	return slackMsg
}

// sendToSlack sends the message to Slack webhook
func (s *SlackSender) sendToSlack(ctx context.Context, msg *SlackMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	return nil
}
