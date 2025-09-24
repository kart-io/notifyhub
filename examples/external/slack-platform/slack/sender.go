// Package slack provides a Slack platform for NotifyHub
// This demonstrates creating external platforms for popular services
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// SlackSender implements the ExternalSender interface for Slack webhooks
type SlackSender struct {
	webhookURL string
	channel    string
	username   string
	iconEmoji  string
	timeout    time.Duration
	client     *http.Client
}

// NewSlackSender creates a new Slack sender
func NewSlackSender(config map[string]interface{}) (platform.ExternalSender, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required for Slack platform")
	}

	sender := &SlackSender{
		webhookURL: webhookURL,
		channel:    "#general",
		username:   "NotifyHub Bot",
		iconEmoji:  ":robot_face:",
		timeout:    30 * time.Second,
	}

	// Configure optional settings
	if channel, ok := config["channel"].(string); ok && channel != "" {
		sender.channel = channel
	}

	if username, ok := config["username"].(string); ok && username != "" {
		sender.username = username
	}

	if iconEmoji, ok := config["icon_emoji"].(string); ok && iconEmoji != "" {
		sender.iconEmoji = iconEmoji
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

		// Build Slack payload
		payload, err := s.buildSlackPayload(msg, target)
		if err != nil {
			result.Error = fmt.Sprintf("failed to build payload: %v", err)
			results[i] = result
			continue
		}

		// Send to Slack
		if err := s.sendToSlack(ctx, payload); err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.MessageID = fmt.Sprintf("slack_%d", time.Now().UnixNano())
			result.Response = "Message sent to Slack"
		}

		// Set metadata
		result.Metadata = map[string]interface{}{
			"duration":    time.Since(startTime).Milliseconds(),
			"channel":     s.getChannelForTarget(target),
			"webhook_url": s.webhookURL,
		}

		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates a target for Slack
func (s *SlackSender) ValidateTarget(target platform.Target) error {
	switch target.Type {
	case "webhook", "channel":
		// Valid target types for Slack
	default:
		return fmt.Errorf("slack supports webhook and channel targets, got %s", target.Type)
	}

	return nil
}

// GetCapabilities returns Slack platform capabilities
func (s *SlackSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "slack",
		SupportedTargetTypes: []string{"webhook", "channel"},
		SupportedFormats:     []string{"text", "markdown"},
		MaxMessageSize:       40000, // Slack limit
		SupportsScheduling:   false,
		SupportsAttachments:  true,
		SupportsMentions:     true,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"webhook_url"},
	}
}

// IsHealthy checks if Slack webhook is accessible
func (s *SlackSender) IsHealthy(ctx context.Context) error {
	if s.webhookURL == "" {
		return fmt.Errorf("slack webhook URL is not configured")
	}

	// Simple health check - send a test request (commented out to avoid spam)
	// testPayload := SlackPayload{
	//     Text: "Health check",
	//     Channel: s.channel,
	// }
	//
	// return s.sendToSlack(ctx, testPayload)

	return nil
}

// Close cleans up resources
func (s *SlackSender) Close() error {
	return nil
}

// SlackPayload represents a Slack webhook payload
type SlackPayload struct {
	Text        string                   `json:"text,omitempty"`
	Channel     string                   `json:"channel,omitempty"`
	Username    string                   `json:"username,omitempty"`
	IconEmoji   string                   `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment        `json:"attachments,omitempty"`
	Blocks      []map[string]interface{} `json:"blocks,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
	MarkdownIn []string     `json:"mrkdwn_in,omitempty"`
}

// SlackField represents a field in Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// buildSlackPayload builds the Slack webhook payload
func (s *SlackSender) buildSlackPayload(msg *platform.Message, target platform.Target) (SlackPayload, error) {
	payload := SlackPayload{
		Channel:   s.getChannelForTarget(target),
		Username:  s.username,
		IconEmoji: s.iconEmoji,
	}

	// Check for custom Slack data
	if slackData, ok := msg.PlatformData["slack_payload"]; ok {
		// Use custom payload if provided
		if customPayload, ok := slackData.(SlackPayload); ok {
			return customPayload, nil
		}
		if customPayload, ok := slackData.(map[string]interface{}); ok {
			payloadBytes, _ := json.Marshal(customPayload)
			_ = json.Unmarshal(payloadBytes, &payload)
			return payload, nil
		}
	}

	// Check for Slack blocks
	if blocks, ok := msg.PlatformData["slack_blocks"].([]map[string]interface{}); ok {
		payload.Blocks = blocks
		payload.Text = msg.Title // Fallback text for notifications
		return payload, nil
	}

	// Check for rich attachments
	if attachments, ok := msg.PlatformData["slack_attachments"].([]SlackAttachment); ok {
		payload.Attachments = attachments
		payload.Text = msg.Title
		return payload, nil
	}

	// Build standard message
	if msg.Format == "markdown" || msg.Format == "" {
		// Use attachment for rich formatting
		attachment := SlackAttachment{
			Color:      s.getColorForPriority(msg.Priority),
			Title:      msg.Title,
			Text:       msg.Body,
			Footer:     "NotifyHub",
			Timestamp:  time.Now().Unix(),
			MarkdownIn: []string{"text", "title"},
		}

		// Add metadata as fields
		if len(msg.Metadata) > 0 {
			for key, value := range msg.Metadata {
				attachment.Fields = append(attachment.Fields, SlackField{
					Title: capitalizeFirst(key),
					Value: fmt.Sprintf("%v", value),
					Short: true,
				})
			}
		}

		// Add variables as fields
		if len(msg.Variables) > 0 {
			for key, value := range msg.Variables {
				attachment.Fields = append(attachment.Fields, SlackField{
					Title: capitalizeFirst(key),
					Value: fmt.Sprintf("%v", value),
					Short: true,
				})
			}
		}

		payload.Attachments = []SlackAttachment{attachment}
		payload.Text = msg.Title
	} else {
		// Simple text message
		text := msg.Title
		if msg.Body != "" {
			text = fmt.Sprintf("%s\n%s", msg.Title, msg.Body)
		}
		payload.Text = text
	}

	return payload, nil
}

// getChannelForTarget determines the target channel
func (s *SlackSender) getChannelForTarget(target platform.Target) string {
	if target.Type == "channel" && target.Value != "" {
		return target.Value
	}
	return s.channel
}

// getColorForPriority returns color based on message priority
func (s *SlackSender) getColorForPriority(priority int) string {
	switch {
	case priority >= 8: // Urgent
		return "danger"
	case priority >= 5: // Alert
		return "warning"
	case priority >= 3: // Normal
		return "good"
	default: // Low
		return "#CCCCCC"
	}
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

// sendToSlack sends the payload to Slack webhook
func (s *SlackSender) sendToSlack(ctx context.Context, payload SlackPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(payloadBytes))
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
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}
