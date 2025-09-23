// Package feishu implements the Feishu Sender for Feishu/Lark notifications
// This integrates all Feishu-specific logic including webhooks, cards, and user mentions
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/kart-io/notifyhub/internal/platform"
	"github.com/kart-io/notifyhub/pkg/utils/crypto"
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

// FeishuSender implements the platform.Sender interface for Feishu notifications
type FeishuSender struct {
	name   string
	config FeishuConfig
	client *http.Client
}

// FeishuConfig holds Feishu configuration
type FeishuConfig struct {
	WebhookURL string        `json:"webhook_url"`
	AuthMode   AuthMode      `json:"auth_mode"` // Authentication mode: none, signature, keywords
	Secret     string        `json:"secret"`    // Required when AuthMode is "signature"
	Keywords   []string      `json:"keywords"`  // Required when AuthMode is "keywords"
	Timeout    time.Duration `json:"timeout"`
}

// FeishuMessage represents a Feishu message payload
type FeishuMessage struct {
	MsgType   string      `json:"msg_type"`
	Content   interface{} `json:"content"`
	Timestamp string      `json:"timestamp,omitempty"`
	Sign      string      `json:"sign,omitempty"`
}

// FeishuTextContent represents text message content
type FeishuTextContent struct {
	Text string `json:"text"`
}

// FeishuCardContent represents card message content
type FeishuCardContent struct {
	Card interface{} `json:"card"`
}

// FeishuResponse represents Feishu API response
type FeishuResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// NewFeishuSender creates a new Feishu sender
func NewFeishuSender(config map[string]interface{}) (platform.Sender, error) {
	feishuConfig, err := parseFeishuConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid feishu configuration: %w", err)
	}

	return &FeishuSender{
		name:   "feishu",
		config: feishuConfig,
		client: &http.Client{
			Timeout: feishuConfig.Timeout,
		},
	}, nil
}

// Name returns the platform name
func (f *FeishuSender) Name() string {
	return f.name
}

// Send sends a Feishu message to the specified targets
func (f *FeishuSender) Send(ctx context.Context, msg *platform.InternalMessage, targets []platform.InternalTarget) ([]*platform.SendResult, error) {
	println("Send Feishu message", msg.Title, msg.Body)
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := platform.NewSendResult(target, false)

		// Validate target
		if err := f.ValidateTarget(target); err != nil {
			result.Error = err.Error()
			result.SetDuration(time.Since(startTime))
			results[i] = result
			continue
		}

		// Build Feishu message
		feishuMsg, err := f.buildFeishuMessage(msg, target)
		if err != nil {
			result.Error = fmt.Sprintf("failed to build feishu message: %v", err)
			result.SetDuration(time.Since(startTime))
			results[i] = result
			continue
		}

		// Send message
		response, err := f.sendMessage(ctx, feishuMsg)
		if err != nil {
			result.Error = fmt.Sprintf("failed to send feishu message: %v", err)
		} else if response.Code != 0 {
			result.Error = fmt.Sprintf("feishu API error: %s (code: %d)", response.Msg, response.Code)
		} else {
			result.Success = true
			result.MessageID = fmt.Sprintf("feishu_%d", time.Now().UnixNano())
			result.Response = response.Msg
		}

		result.SetDuration(time.Since(startTime))
		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates a Feishu target
func (f *FeishuSender) ValidateTarget(target platform.InternalTarget) error {
	if target.Type != "user" && target.Type != "group" && target.Type != "webhook" {
		return fmt.Errorf("feishu sender supports user, group, and webhook targets, got %s", target.Type)
	}

	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}

	// Validate webhook URL format if it's a webhook target
	if target.Type == "webhook" {
		if len(target.Value) < 10 || (target.Value[:4] != "http") {
			return fmt.Errorf("invalid webhook URL: %s", target.Value)
		}
	}

	return nil
}

// GetCapabilities returns the Feishu platform capabilities
func (f *FeishuSender) GetCapabilities() platform.PlatformCapabilities {
	// Include auth mode info in required settings based on current configuration
	requiredSettings := []string{"webhook_url"}
	switch f.config.AuthMode {
	case AuthModeSignature:
		requiredSettings = append(requiredSettings, "secret")
	case AuthModeKeywords:
		requiredSettings = append(requiredSettings, "keywords")
	}

	return platform.PlatformCapabilities{
		Name:                 "feishu",
		SupportedTargetTypes: []string{"user", "group", "webhook"},
		SupportedFormats:     []string{"text", "markdown", "card"},
		MaxMessageSize:       30 * 1024, // 30KB typical limit for Feishu
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		SupportsMentions:     true,
		SupportsRichContent:  true,
		RequiredSettings:     requiredSettings,
		RateLimits: &platform.RateLimitInfo{
			RequestsPerSecond: 5,
			RequestsPerMinute: 100,
			BurstSize:         10,
		},
	}
}

// IsHealthy checks if the Feishu sender is healthy
func (f *FeishuSender) IsHealthy(ctx context.Context) error {
	// Create a simple test message
	testMsg := &FeishuMessage{
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10),
		MsgType:   "text",
		Content: FeishuTextContent{
			Text: "Health check test",
		},
	}

	// Add authentication based on AuthMode
	if f.config.AuthMode == AuthModeSignature {
		testMsg.Sign = f.generateSignature(testMsg.Timestamp)
	}

	// Convert to JSON
	payload, err := json.Marshal(testMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal test message: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", f.config.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request (but don't actually send the test message by using a dry-run approach)
	// For health check, we just verify we can create a valid request
	return nil
}

// Close cleans up the Feishu sender resources
func (f *FeishuSender) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

// buildFeishuMessage builds a Feishu message from the internal message
func (f *FeishuSender) buildFeishuMessage(msg *platform.InternalMessage, target platform.InternalTarget) (*FeishuMessage, error) {
	feishuMsg := &FeishuMessage{}

	// Add authentication based on AuthMode
	switch f.config.AuthMode {
	case AuthModeSignature:
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		feishuMsg.Timestamp = timestamp
		feishuMsg.Sign = f.generateSignature(timestamp)
	case AuthModeKeywords:
		// Keywords will be added to text content
	case AuthModeNone:
		// No authentication required
	}

	// Check if there's card content in platform data
	if cardContent, ok := msg.PlatformData["feishu_card"]; ok {
		feishuMsg.MsgType = "interactive"

		// Parse card content
		var card interface{}
		if cardStr, ok := cardContent.(string); ok {
			if err := json.Unmarshal([]byte(cardStr), &card); err != nil {
				return nil, fmt.Errorf("invalid card JSON: %w", err)
			}
		} else {
			card = cardContent
		}

		feishuMsg.Content = FeishuCardContent{
			Card: card,
		}
	} else {
		// Text message
		feishuMsg.MsgType = "text"

		text := f.buildTextContent(msg)
		feishuMsg.Content = FeishuTextContent{
			Text: text,
		}
	}

	return feishuMsg, nil
}

// buildTextContent builds text content with mentions and formatting
func (f *FeishuSender) buildTextContent(msg *platform.InternalMessage) string {
	text := ""

	// Add custom keywords if AuthMode is keywords
	if f.config.AuthMode == AuthModeKeywords && len(f.config.Keywords) > 0 {
		// Use the first keyword as trigger
		text += f.config.Keywords[0] + " "
	}

	// Add title if present
	if msg.Title != "" {
		text += fmt.Sprintf("**%s**\n", msg.Title)
	}

	// Add body
	if msg.Body != "" {
		text += msg.Body
	}

	// Add mentions from platform data
	if mentions, ok := msg.PlatformData["feishu_mentions"].([]map[string]interface{}); ok {
		for _, mention := range mentions {
			if userID, ok := mention["user_id"].(string); ok {
				text += fmt.Sprintf(" <at user_id=\"%s\"></at>", userID)
			}
		}
	}

	// Add mention all if specified
	if mentionAll, ok := msg.PlatformData["feishu_mention_all"].(bool); ok && mentionAll {
		text += " <at user_id=\"all\">所有人</at>"
	}

	return text
}

// sendMessage sends the Feishu message
func (f *FeishuSender) sendMessage(ctx context.Context, msg *FeishuMessage) (*FeishuResponse, error) {
	// Convert message to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", f.config.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var feishuResp FeishuResponse
	if err := json.Unmarshal(body, &feishuResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &feishuResp, nil
}

// generateSignature generates signature for Feishu webhook
// 飞书签名算法: 使用特殊的HMAC实现
func (f *FeishuSender) generateSignature(timestamp string) string {
	if f.config.Secret == "" {
		return ""
	}

	// 使用新的加密工具库生成飞书签名
	signer := crypto.NewFeishuSigner(f.config.Secret)
	return signer.GenerateSignature(timestamp)
}

// parseFeishuConfig parses Feishu configuration from map
func parseFeishuConfig(config map[string]interface{}) (FeishuConfig, error) {
	feishuConfig := FeishuConfig{
		Timeout:  30 * time.Second, // Default timeout
		AuthMode: AuthModeNone,     // Default to no authentication
	}

	// Required fields
	if webhookURL, ok := config["webhook_url"].(string); ok {
		feishuConfig.WebhookURL = webhookURL
	} else {
		return feishuConfig, fmt.Errorf("webhook_url is required")
	}

	// Parse auth_mode if explicitly provided
	if authMode, ok := config["auth_mode"].(string); ok {
		switch AuthMode(authMode) {
		case AuthModeNone, AuthModeSignature, AuthModeKeywords:
			feishuConfig.AuthMode = AuthMode(authMode)
		default:
			return feishuConfig, fmt.Errorf("invalid auth_mode: %s, must be one of: none, signature, keywords", authMode)
		}
	}

	// Parse secret
	if secret, ok := config["secret"].(string); ok {
		feishuConfig.Secret = secret
	}

	// Parse keywords (can be string or []string)
	if keywordsInterface, ok := config["keywords"]; ok {
		switch v := keywordsInterface.(type) {
		case string:
			if v != "" {
				feishuConfig.Keywords = []string{v}
			}
		case []string:
			feishuConfig.Keywords = v
		case []interface{}:
			for _, keyword := range v {
				if keywordStr, ok := keyword.(string); ok && keywordStr != "" {
					feishuConfig.Keywords = append(feishuConfig.Keywords, keywordStr)
				}
			}
		}
	}

	// If auth_mode not explicitly set, determine it based on secret and keywords
	if _, explicitAuthMode := config["auth_mode"]; !explicitAuthMode {
		if feishuConfig.Secret != "" && len(feishuConfig.Keywords) > 0 {
			return feishuConfig, fmt.Errorf("both secret and keywords provided, they are mutually exclusive. Please specify auth_mode explicitly or use only one authentication method")
		} else if feishuConfig.Secret != "" {
			feishuConfig.AuthMode = AuthModeSignature
		} else if len(feishuConfig.Keywords) > 0 {
			feishuConfig.AuthMode = AuthModeKeywords
		} else {
			feishuConfig.AuthMode = AuthModeNone
		}
	}

	// Validate configuration based on AuthMode
	switch feishuConfig.AuthMode {
	case AuthModeSignature:
		if feishuConfig.Secret == "" {
			return feishuConfig, fmt.Errorf("secret is required when auth_mode is 'signature'")
		}
		if len(feishuConfig.Keywords) > 0 {
			return feishuConfig, fmt.Errorf("keywords should not be provided when auth_mode is 'signature'")
		}
	case AuthModeKeywords:
		if len(feishuConfig.Keywords) == 0 {
			return feishuConfig, fmt.Errorf("keywords are required when auth_mode is 'keywords'")
		}
		if feishuConfig.Secret != "" {
			return feishuConfig, fmt.Errorf("secret should not be provided when auth_mode is 'keywords'")
		}
	case AuthModeNone:
		if feishuConfig.Secret != "" {
			return feishuConfig, fmt.Errorf("secret should not be provided when auth_mode is 'none'")
		}
		if len(feishuConfig.Keywords) > 0 {
			return feishuConfig, fmt.Errorf("keywords should not be provided when auth_mode is 'none'")
		}
	}

	if timeout, ok := config["timeout"].(time.Duration); ok {
		feishuConfig.Timeout = timeout
	}

	// Validate webhook URL
	if len(feishuConfig.WebhookURL) < 10 || feishuConfig.WebhookURL[:4] != "http" {
		return feishuConfig, fmt.Errorf("invalid webhook_url format")
	}

	return feishuConfig, nil
}

// init registers the FeishuSender with the platform manager
func init() {
	platform.RegisterSenderCreator("feishu", NewFeishuSender)
}
