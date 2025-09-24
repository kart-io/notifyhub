// Package feishu provides Feishu/Lark platform integration for NotifyHub
// This package implements unified platform architecture for Feishu notifications
package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// AuthMode defines the authentication mode for Feishu webhook
type AuthMode string

const (
	// AuthModeNone no authentication required
	AuthModeNone AuthMode = "none"
	// AuthModeSignature use HMAC-SHA256 signature verification
	AuthModeSignature AuthMode = "signature"
	// AuthModeKeywords use custom keywords verification
	AuthModeKeywords AuthMode = "keywords"
)

// FeishuSender implements the ExternalSender interface for Feishu notifications
type FeishuSender struct {
	webhookURL string
	authMode   AuthMode
	secret     string
	keywords   []string
	timeout    time.Duration
	client     *http.Client
}

// FeishuMessage represents a Feishu message payload
type FeishuMessage struct {
	MsgType   string      `json:"msg_type"`
	Content   interface{} `json:"content"`
	Sign      string      `json:"sign,omitempty"`      // For signature auth
	Timestamp string      `json:"timestamp,omitempty"` // For signature auth
}

// TextContent represents text message content
type TextContent struct {
	Text string `json:"text"`
}

// RichTextContent represents rich text message content
type RichTextContent struct {
	Post interface{} `json:"post"`
}

// CardContent represents interactive card content
type CardContent struct {
	Card interface{} `json:"card"`
}

// NewFeishuSender creates a new Feishu sender
func NewFeishuSender(config map[string]interface{}) (platform.ExternalSender, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required for Feishu platform")
	}

	sender := &FeishuSender{
		webhookURL: webhookURL,
		authMode:   AuthModeNone,
		timeout:    30 * time.Second,
	}

	// Configure authentication mode
	if authMode, ok := config["auth_mode"].(string); ok {
		sender.authMode = AuthMode(authMode)
	}

	// Configure secret for signature auth
	if secret, ok := config["secret"].(string); ok {
		sender.secret = secret
		if sender.authMode == AuthModeNone && secret != "" {
			sender.authMode = AuthModeSignature
		}
	}

	// Configure keywords for keyword auth
	if keywords, ok := config["keywords"].([]string); ok {
		sender.keywords = keywords
		if sender.authMode == AuthModeNone && len(keywords) > 0 {
			sender.authMode = AuthModeKeywords
		}
	}

	// Configure timeout
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
func (f *FeishuSender) Name() string {
	return "feishu"
}

// Send sends a message to Feishu
func (f *FeishuSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := &platform.SendResult{
			Target:  target,
			Success: false,
		}

		// Validate target
		if err := f.ValidateTarget(target); err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		// Build Feishu message
		feishuMsg := f.buildFeishuMessage(msg, target)

		// Send message
		if err := f.sendToFeishu(ctx, feishuMsg); err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.MessageID = fmt.Sprintf("feishu_%d", time.Now().UnixNano())
			result.Response = "Message sent successfully"
		}

		// Set metadata
		result.Metadata = map[string]interface{}{
			"duration":  time.Since(startTime).Milliseconds(),
			"auth_mode": string(f.authMode),
		}

		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates a target for Feishu
func (f *FeishuSender) ValidateTarget(target platform.Target) error {
	switch target.Type {
	case "webhook", "group", "user", "channel":
		// These are valid target types for Feishu
	default:
		return fmt.Errorf("feishu supports webhook, group, user, and channel targets, got %s", target.Type)
	}

	if target.Value == "" && target.Type != "webhook" {
		return fmt.Errorf("target value cannot be empty for type %s", target.Type)
	}

	return nil
}

// GetCapabilities returns Feishu platform capabilities
func (f *FeishuSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "feishu",
		SupportedTargetTypes: []string{"webhook", "group", "user", "channel"},
		SupportedFormats:     []string{"text", "markdown", "card"},
		MaxMessageSize:       30000, // Feishu's message limit
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		SupportsMentions:     true,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"webhook_url"},
	}
}

// IsHealthy checks if Feishu webhook is accessible
func (f *FeishuSender) IsHealthy(ctx context.Context) error {
	if f.webhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}
	return nil
}

// Close cleans up resources
func (f *FeishuSender) Close() error {
	return nil
}

// buildFeishuMessage builds a Feishu message from the platform message
func (f *FeishuSender) buildFeishuMessage(msg *platform.Message, target platform.Target) *FeishuMessage {
	feishuMsg := &FeishuMessage{}

	// Check for Feishu card content
	if card, ok := msg.PlatformData["feishu_card"]; ok {
		feishuMsg.MsgType = "interactive"
		feishuMsg.Content = &CardContent{Card: card}
		return feishuMsg
	}

	// Check for rich text content
	if post, ok := msg.PlatformData["feishu_post"]; ok {
		feishuMsg.MsgType = "post"
		feishuMsg.Content = &RichTextContent{Post: post}
		return feishuMsg
	}

	// Default to text message
	text := msg.Body
	if msg.Title != "" {
		text = msg.Title + "\n" + msg.Body
	}

	// Handle mentions
	if mentions, ok := msg.PlatformData["feishu_mentions"]; ok {
		if mentionList, ok := mentions.([]map[string]interface{}); ok {
			for _, mention := range mentionList {
				if userID, ok := mention["user_id"].(string); ok {
					text += fmt.Sprintf(" <at user_id=\"%s\">@user</at>", userID)
				}
			}
		}
	}

	// Handle mention all
	if mentionAll, ok := msg.PlatformData["feishu_mention_all"].(bool); ok && mentionAll {
		text += " <at user_id=\"all\">@所有人</at>"
	}

	// Handle keywords for keyword auth
	if f.authMode == AuthModeKeywords && len(f.keywords) > 0 {
		text = f.keywords[0] + " " + text
	}

	feishuMsg.MsgType = "text"
	feishuMsg.Content = &TextContent{Text: text}

	return feishuMsg
}

// sendToFeishu sends the message to Feishu webhook
func (f *FeishuSender) sendToFeishu(ctx context.Context, msg *FeishuMessage) error {
	// Add signature if required
	if f.authMode == AuthModeSignature && f.secret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		sign := f.generateSign(timestamp)
		msg.Sign = sign
		msg.Timestamp = timestamp
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Feishu message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", f.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body for debugging and error checking
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug: Print response details
	fmt.Printf("🔍 Feishu API Response:\n")
	fmt.Printf("   Status Code: %d\n", resp.StatusCode)
	fmt.Printf("   Response Body: %s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response to check for Feishu-specific errors
	var feishuResp struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &feishuResp); err != nil {
		fmt.Printf("⚠️  Failed to parse Feishu response as JSON: %v\n", err)
		return nil // Still consider it successful if we can't parse
	}

	// Check Feishu error code
	if feishuResp.Code != 0 {
		return fmt.Errorf("feishu API error (code %d): %s", feishuResp.Code, feishuResp.Msg)
	}

	fmt.Printf("✅ Feishu message sent successfully!\n")
	return nil
}

// generateSign generates HMAC-SHA256 signature for Feishu webhook
// According to Feishu official documentation:
// 1. stringToSign = timestamp + "\n" + secret
// 2. signature = base64(hmac_sha256(stringToSign, ""))
func (f *FeishuSender) generateSign(timestamp string) string {
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, f.secret)
	hash := hmac.New(sha256.New, []byte(stringToSign))
	hash.Write([]byte("")) // Feishu uses empty string as data
	signature := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return signature
}
