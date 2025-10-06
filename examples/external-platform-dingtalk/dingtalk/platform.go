// Package dingtalk provides DingTalk platform implementation for NotifyHub
// This demonstrates how to implement an external platform without modifying NotifyHub core code
package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/platform"
	"github.com/kart-io/notifyhub/pkg/target"
)

// Config holds DingTalk platform configuration
type Config struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret,omitempty"`
	Timeout    int    `json:"timeout"` // seconds
}

// Platform implements the Platform interface for DingTalk
type Platform struct {
	config     Config
	httpClient *http.Client
}

// Message represents DingTalk message format
type Message struct {
	MsgType  string    `json:"msgtype"`
	Text     *Text     `json:"text,omitempty"`
	Markdown *Markdown `json:"markdown,omitempty"`
	At       *At       `json:"at,omitempty"`
}

type Text struct {
	Content string `json:"content"`
}

type Markdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type At struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	AtUserIds []string `json:"atUserIds,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

// Response represents DingTalk API response
type Response struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// New creates a new DingTalk platform instance
func New(config interface{}) (platform.Platform, error) {
	cfg, ok := config.(Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type for DingTalk platform")
	}

	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required for DingTalk platform")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 // default 30 seconds
	}

	httpClient := &http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Second,
	}

	return &Platform{
		config:     cfg,
		httpClient: httpClient,
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "dingtalk"
}

// GetCapabilities returns platform capabilities
func (p *Platform) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "dingtalk",
		SupportedTargetTypes: []string{"dingtalk_webhook", "dingtalk_group"},
		SupportedFormats:     []string{"text", "markdown"},
		MaxMessageSize:       20000, // 20KB limit
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		RequiredSettings:     []string{"webhook_url"},
	}
}

// ValidateTarget validates if the target is supported
func (p *Platform) ValidateTarget(target target.Target) error {
	switch target.Type {
	case "dingtalk_webhook", "dingtalk_group":
		return nil
	default:
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}
}

// Send sends message to DingTalk
func (p *Platform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		result := &platform.SendResult{
			Target: target,
		}

		// Validate target
		if err := p.ValidateTarget(target); err != nil {
			result.Error = err
			results[i] = result
			continue
		}

		// Convert message to DingTalk format
		dingMsg, err := p.convertMessage(msg, target)
		if err != nil {
			result.Error = fmt.Errorf("failed to convert message: %w", err)
			results[i] = result
			continue
		}

		// Send to DingTalk
		messageID, response, err := p.sendToDingTalk(ctx, dingMsg)
		if err != nil {
			result.Error = err
			result.Response = response
		} else {
			result.Success = true
			result.MessageID = messageID
			result.Response = response
		}

		results[i] = result
	}

	return results, nil
}

// convertMessage converts NotifyHub message to DingTalk format
func (p *Platform) convertMessage(msg *message.Message, target target.Target) (*Message, error) {
	dingMsg := &Message{}

	// Handle @mentions from target - simplified for demo
	if at := p.parseAtMentions(target); at != nil {
		dingMsg.At = at
	}

	// Convert based on message format
	switch msg.Format {
	case message.FormatMarkdown:
		dingMsg.MsgType = "markdown"
		dingMsg.Markdown = &Markdown{
			Title: msg.Title,
			Text:  msg.Body,
		}
	case message.FormatText:
		fallthrough
	default:
		dingMsg.MsgType = "text"
		content := msg.Body
		if msg.Title != "" {
			content = fmt.Sprintf("%s\n\n%s", msg.Title, msg.Body)
		}
		dingMsg.Text = &Text{
			Content: content,
		}
	}

	return dingMsg, nil
}

// parseAtMentions parses @mentions from target - simplified for demo
func (p *Platform) parseAtMentions(target target.Target) *At {
	// In a real implementation, you would parse mentions from target configuration
	// For demo purposes, we'll return nil (no mentions)
	return nil
}

// sendToDingTalk sends the message to DingTalk webhook
func (p *Platform) sendToDingTalk(ctx context.Context, dingMsg *Message) (string, string, error) {
	// Marshal message to JSON
	msgBytes, err := json.Marshal(dingMsg)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.WebhookURL, bytes.NewBuffer(msgBytes))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var dingResp Response
	if err := json.NewDecoder(resp.Body).Decode(&dingResp); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors
	if dingResp.ErrCode != 0 {
		return "", dingResp.ErrMsg, fmt.Errorf("DingTalk API error: %s (code: %d)", dingResp.ErrMsg, dingResp.ErrCode)
	}

	// Generate message ID (DingTalk doesn't return one)
	messageID := fmt.Sprintf("dingtalk_%d", time.Now().Unix())

	return messageID, "success", nil
}

// IsHealthy checks if the platform is healthy
func (p *Platform) IsHealthy(ctx context.Context) error {
	// Simple health check - verify webhook URL is accessible
	req, err := http.NewRequestWithContext(ctx, "HEAD", p.config.WebhookURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook endpoint returned status: %d", resp.StatusCode)
	}

	return nil
}

// Close cleans up platform resources
func (p *Platform) Close() error {
	// Close HTTP client if needed
	if transport, ok := p.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

// CreateTarget creates a DingTalk target with the given webhook URL
func CreateTarget(webhookURL string) target.Target {
	return target.Target{
		Type:     "dingtalk_webhook",
		Value:    webhookURL,
		Platform: "dingtalk",
	}
}

// CreateTargetWithAt creates a DingTalk target with @mentions
// Note: In a real implementation, mentions would be handled via target metadata or configuration
func CreateTargetWithAt(webhookURL string, atAll bool, atMobiles []string, atUserIds []string) target.Target {
	// For this demo, we'll use the same structure as CreateTarget
	// In a real implementation, you would need to extend the target structure
	// or use a configuration mechanism to handle @mentions
	return target.Target{
		Type:     "dingtalk_webhook",
		Value:    webhookURL,
		Platform: "dingtalk",
	}
}
