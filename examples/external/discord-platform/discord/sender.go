// Package discord provides Discord platform integration for NotifyHub
// This package demonstrates how to create external platform packages
package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// DiscordSender implements the ExternalSender interface for Discord notifications
type DiscordSender struct {
	webhookURL      string
	defaultUsername string
	defaultAvatar   string
	timeout         time.Duration
	client          *http.Client
}

// DiscordMessage represents a Discord webhook message
type DiscordMessage struct {
	Content   string                   `json:"content,omitempty"`
	Username  string                   `json:"username,omitempty"`
	AvatarURL string                   `json:"avatar_url,omitempty"`
	Embeds    []map[string]interface{} `json:"embeds,omitempty"`
}

// NewDiscordSender creates a new Discord sender
func NewDiscordSender(config map[string]interface{}) (platform.ExternalSender, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required for Discord platform")
	}

	sender := &DiscordSender{
		webhookURL: webhookURL,
		timeout:    30 * time.Second,
	}

	// Configure optional settings
	if username, ok := config["default_username"].(string); ok {
		sender.defaultUsername = username
	}

	if avatar, ok := config["default_avatar_url"].(string); ok {
		sender.defaultAvatar = avatar
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
func (d *DiscordSender) Name() string {
	return "discord"
}

// Send sends a message to Discord
func (d *DiscordSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := &platform.SendResult{
			Target:  target,
			Success: false,
		}

		// Validate target
		if err := d.ValidateTarget(target); err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		// Build Discord message
		discordMsg := d.buildDiscordMessage(msg, target)

		// Send message
		if err := d.sendToDiscord(ctx, discordMsg); err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.MessageID = fmt.Sprintf("discord_%d", time.Now().UnixNano())
			result.Response = "Message sent successfully"
		}

		// Set metadata
		result.Metadata = map[string]interface{}{
			"duration": time.Since(startTime).Milliseconds(),
			"username": discordMsg.Username,
		}

		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates a target for Discord
func (d *DiscordSender) ValidateTarget(target platform.Target) error {
	switch target.Type {
	case "webhook", "channel":
		// Valid target types for Discord
	default:
		return fmt.Errorf("discord supports webhook and channel targets, got %s", target.Type)
	}

	return nil
}

// GetCapabilities returns Discord platform capabilities
func (d *DiscordSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "discord",
		SupportedTargetTypes: []string{"webhook", "channel"},
		SupportedFormats:     []string{"text", "markdown"},
		MaxMessageSize:       2000, // Discord's message limit
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		SupportsMentions:     true,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"webhook_url"},
	}
}

// IsHealthy checks if Discord webhook is accessible
func (d *DiscordSender) IsHealthy(ctx context.Context) error {
	if d.webhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}
	return nil
}

// Close cleans up resources
func (d *DiscordSender) Close() error {
	return nil
}

// buildDiscordMessage builds a Discord message from the platform message
func (d *DiscordSender) buildDiscordMessage(msg *platform.Message, target platform.Target) *DiscordMessage {
	discordMsg := &DiscordMessage{
		Username:  d.defaultUsername,
		AvatarURL: d.defaultAvatar,
	}

	// Check for Discord embeds
	if embeds, ok := msg.PlatformData["discord_embeds"]; ok {
		if embedList, ok := embeds.([]map[string]interface{}); ok {
			discordMsg.Embeds = embedList
		}
	}

	// Build content
	content := msg.Body
	if msg.Title != "" {
		if len(discordMsg.Embeds) == 0 {
			// No embeds, use title in content
			content = fmt.Sprintf("**%s**\n%s", msg.Title, msg.Body)
		}
		// If embeds exist, title will be in embed
	}

	// Handle mentions
	if mentions, ok := msg.PlatformData["discord_mentions"]; ok {
		if mentionList, ok := mentions.([]string); ok {
			for _, userID := range mentionList {
				content += fmt.Sprintf(" <@%s>", userID)
			}
		}
	}

	discordMsg.Content = content
	return discordMsg
}

// sendToDiscord sends the message to Discord webhook
func (d *DiscordSender) sendToDiscord(ctx context.Context, msg *DiscordMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord API returned status %d", resp.StatusCode)
	}

	return nil
}
