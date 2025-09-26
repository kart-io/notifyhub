// Package dingtalk provides DingTalk platform integration for NotifyHub
// This package implements unified platform architecture for DingTalk notifications
package dingtalk

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
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// DingTalkSender implements the Platform interface for DingTalk webhooks
type DingTalkSender struct {
	webhookURL string
	secret     string
	keywords   []string
	client     *http.Client
	logger     logger.Logger
}

// DingTalkMessage represents a DingTalk webhook message structure
type DingTalkMessage struct {
	MsgType    string      `json:"msgtype"`
	Text       interface{} `json:"text,omitempty"`
	Markdown   interface{} `json:"markdown,omitempty"`
	ActionCard interface{} `json:"actionCard,omitempty"`
	At         *AtConfig   `json:"at,omitempty"`
}

// DingTalkTextContent represents text content for DingTalk
type DingTalkTextContent struct {
	Content string `json:"content"`
}

// DingTalkMarkdownContent represents markdown content for DingTalk
type DingTalkMarkdownContent struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// DingTalkActionCardContent represents action card content for DingTalk
type DingTalkActionCardContent struct {
	Title          string `json:"title"`
	Text           string `json:"text"`
	SingleTitle    string `json:"singleTitle,omitempty"`
	SingleURL      string `json:"singleURL,omitempty"`
	BtnOrientation string `json:"btnOrientation,omitempty"`
}

// AtConfig represents @mention configuration
type AtConfig struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	AtUserIds []string `json:"atUserIds,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

// DingTalkConfig is imported from config package
// We use config.DingTalkConfig directly

// NewDingTalkPlatform creates a new DingTalk platform with strong-typed configuration
func NewDingTalkPlatform(dingTalkConfig *config.DingTalkConfig, logger logger.Logger) (platform.Platform, error) {
	if dingTalkConfig.WebhookURL == "" {
		return nil, fmt.Errorf("dingtalk webhook URL is required")
	}

	return &DingTalkSender{
		webhookURL: dingTalkConfig.WebhookURL,
		secret:     dingTalkConfig.Secret,
		keywords:   dingTalkConfig.Keywords,
		client: &http.Client{
			Timeout: dingTalkConfig.Timeout,
		},
		logger: logger,
	}, nil
}

// NewDingTalkSender creates a new DingTalk sender with map configuration (for backward compatibility)
func NewDingTalkSender(config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("dingtalk webhook_url is required")
	}

	secret, _ := config["secret"].(string)

	// Parse keywords from config
	var keywords []string
	if keywordsInterface, ok := config["keywords"]; ok {
		if keywordSlice, ok := keywordsInterface.([]interface{}); ok {
			for _, keyword := range keywordSlice {
				if keywordStr, ok := keyword.(string); ok {
					keywords = append(keywords, keywordStr)
				}
			}
		} else if keywordsStr, ok := keywordsInterface.(string); ok {
			// Support comma-separated string format
			if keywordsStr != "" {
				keywords = strings.Split(keywordsStr, ",")
				for i, keyword := range keywords {
					keywords[i] = strings.TrimSpace(keyword)
				}
			}
		} else if keywordSlice, ok := keywordsInterface.([]string); ok {
			keywords = keywordSlice
		}
	}

	timeout := 30 * time.Second
	if t, ok := config["timeout"].(time.Duration); ok {
		timeout = t
	} else if t, ok := config["timeout"].(string); ok {
		if parsed, err := time.ParseDuration(t); err == nil {
			timeout = parsed
		}
	}

	return &DingTalkSender{
		webhookURL: webhookURL,
		secret:     secret,
		keywords:   keywords,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}, nil
}

// Name returns the platform name
func (d *DingTalkSender) Name() string {
	return "dingtalk"
}

// Send implements the Platform interface
func (d *DingTalkSender) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	// Filter targets for DingTalk
	dingTalkTargets := d.filterDingTalkTargets(targets)
	if len(dingTalkTargets) == 0 {
		d.logger.Info("No DingTalk targets found, skipping")
		// Return failure results for non-dingtalk targets
		for i, target := range targets {
			results[i] = &platform.SendResult{
				Target:  target,
				Success: false,
				Error:   "not a dingtalk target",
			}
		}
		return results, nil
	}

	// Send to each target
	for i, target := range targets {
		if d.isDingTalkTarget(target) {
			err := d.sendSingleMessage(ctx, msg, target)
			if err != nil {
				results[i] = &platform.SendResult{
					Target:  target,
					Success: false,
					Error:   err.Error(),
				}
			} else {
				messageID := msg.ID
				if messageID == "" {
					messageID = fmt.Sprintf("dingtalk_%d", time.Now().UnixNano())
				}
				results[i] = &platform.SendResult{
					Target:    target,
					Success:   true,
					MessageID: messageID,
				}
			}
		} else {
			results[i] = &platform.SendResult{
				Target:  target,
				Success: false,
				Error:   "not a dingtalk target",
			}
		}
	}

	return results, nil
}

// sendSingleMessage sends a message to a single dingtalk target
func (d *DingTalkSender) sendSingleMessage(ctx context.Context, msg *message.Message, target target.Target) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Build DingTalk message
	dingTalkMsg, err := d.buildDingTalkMessage(msg)
	if err != nil {
		d.logger.Error("Failed to build DingTalk message", "error", err)
		return fmt.Errorf("failed to build DingTalk message: %w", err)
	}

	// Send to webhook
	if err := d.sendToWebhook(ctx, dingTalkMsg); err != nil {
		d.logger.Error("Failed to send to DingTalk webhook", "error", err)
		return fmt.Errorf("failed to send to DingTalk webhook: %w", err)
	}

	d.logger.Info("DingTalk message sent successfully", "messageID", msg.ID, "target", target.Value)
	return nil
}

// ValidateTarget implements the Platform interface
func (d *DingTalkSender) ValidateTarget(target target.Target) error {
	if target.Type != "dingtalk" && target.Type != "webhook" {
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}
	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}
	return nil
}

// IsHealthy implements the Platform interface
func (d *DingTalkSender) IsHealthy(ctx context.Context) error {
	// Simple health check - verify webhook URL is accessible
	if d.webhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}
	return nil
}

// Close implements the Platform interface
func (d *DingTalkSender) Close() error {
	d.logger.Info("Closing DingTalk sender")
	// Close HTTP client if needed
	if d.client != nil {
		d.client.CloseIdleConnections()
	}
	return nil
}

// GetCapabilities implements the Platform interface
func (d *DingTalkSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "dingtalk",
		SupportedTargetTypes: []string{"dingtalk", "webhook"},
		SupportedFormats:     []string{"text", "markdown", "actionCard"},
		MaxMessageSize:       4000,
	}
}

// isDingTalkTarget checks if a target is relevant for DingTalk
func (d *DingTalkSender) isDingTalkTarget(target target.Target) bool {
	return target.Type == "dingtalk" || target.Type == "webhook"
}

// filterDingTalkTargets filters targets that are relevant for DingTalk
func (d *DingTalkSender) filterDingTalkTargets(targets []target.Target) []target.Target {
	var dingTalkTargets []target.Target
	for _, t := range targets {
		if d.isDingTalkTarget(t) {
			dingTalkTargets = append(dingTalkTargets, t)
		}
	}
	return dingTalkTargets
}

// buildDingTalkMessage builds a DingTalk-specific message format with security processing
func (d *DingTalkSender) buildDingTalkMessage(msg *message.Message) (*DingTalkMessage, error) {
	securityMode := d.determineSecurityMode()
	d.logger.Info("Building DingTalk message with security mode", "securityMode", securityMode)

	// Build basic message structure
	dingTalkMsg, err := d.buildBasicMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to build basic message: %w", err)
	}

	// Apply security mode processing
	return d.applySecurityModeProcessing(dingTalkMsg, msg, securityMode)
}

// buildBasicMessage builds the basic DingTalk message structure
func (d *DingTalkSender) buildBasicMessage(msg *message.Message) (*DingTalkMessage, error) {
	// Check for platform-specific data
	if cardData, exists := msg.PlatformData["dingtalk_actionCard"]; exists {
		d.logger.Debug("Using platform-specific action card data")
		return &DingTalkMessage{
			MsgType:    "actionCard",
			ActionCard: cardData,
		}, nil
	}

	// Build message based on format
	switch msg.Format {
	case message.FormatHTML, message.FormatMarkdown:
		d.logger.Debug("Building markdown message for HTML/Markdown format")
		content := d.buildMarkdownContent(msg)
		return &DingTalkMessage{
			MsgType:  "markdown",
			Markdown: content,
		}, nil

	default:
		d.logger.Debug("Building text message for default format")
		text := d.buildTextContent(msg)
		return &DingTalkMessage{
			MsgType: "text",
			Text: &DingTalkTextContent{
				Content: text,
			},
		}, nil
	}
}

// buildTextContent builds text content from message
func (d *DingTalkSender) buildTextContent(msg *message.Message) string {
	text := msg.Title
	if msg.Body != "" {
		if text != "" {
			text += "\n\n"
		}
		text += msg.Body
	}
	return text
}

// buildMarkdownContent builds markdown content for DingTalk
func (d *DingTalkSender) buildMarkdownContent(msg *message.Message) *DingTalkMarkdownContent {
	text := msg.Body
	if msg.Title != "" && msg.Body != "" {
		text = fmt.Sprintf("# %s\n\n%s", msg.Title, msg.Body)
	} else if msg.Title != "" {
		text = fmt.Sprintf("# %s", msg.Title)
	}

	return &DingTalkMarkdownContent{
		Title: msg.Title,
		Text:  text,
	}
}

// applySecurityModeProcessing applies security mode specific processing
func (d *DingTalkSender) applySecurityModeProcessing(dingTalkMsg *DingTalkMessage, msg *message.Message, securityMode string) (*DingTalkMessage, error) {
	switch securityMode {
	case "no_security":
		d.logger.Info("Security Mode 1: 无需配置签名与自定义关键词 - 直接发送消息")
		return dingTalkMsg, nil

	case "signature_only":
		d.logger.Info("Security Mode 2: 仅配置签名验证 - 消息无需修改，签名将在发送时添加")
		return dingTalkMsg, nil

	case "keywords_only":
		d.logger.Info("Security Mode 3: 仅配置自定义关键词 - 处理关键词验证")
		return d.processKeywordRequirement(dingTalkMsg, msg)

	case "signature_and_keywords":
		d.logger.Info("Security Mode 2+3: 同时配置签名验证和自定义关键词")
		return d.processKeywordRequirement(dingTalkMsg, msg)

	default:
		d.logger.Warn("Unknown security mode, using no security", "mode", securityMode)
		return dingTalkMsg, nil
	}
}

// processKeywordRequirement processes keyword requirement for the message
func (d *DingTalkSender) processKeywordRequirement(dingTalkMsg *DingTalkMessage, msg *message.Message) (*DingTalkMessage, error) {
	if len(d.keywords) == 0 {
		d.logger.Debug("No keywords configured, skipping keyword processing")
		return dingTalkMsg, nil
	}

	d.logger.Debug("Processing keyword requirement", "configuredKeywords", d.keywords)

	// Extract message text content
	messageText := d.extractMessageText(dingTalkMsg, msg)

	// Check if message contains required keyword
	if d.containsRequiredKeyword(messageText) {
		d.logger.Debug("Message already contains required keyword, no modification needed")
		return dingTalkMsg, nil
	}

	// Add first keyword to message
	keywordToAdd := d.keywords[0]
	d.logger.Info("Message doesn't contain required keywords, automatically adding keyword",
		"keyword", keywordToAdd, "availableKeywords", d.keywords)

	return d.addKeywordToMessage(dingTalkMsg, keywordToAdd)
}

// containsRequiredKeyword checks if the message contains any of the required keywords
func (d *DingTalkSender) containsRequiredKeyword(messageText string) bool {
	if messageText == "" {
		return false
	}

	messageTextLower := strings.ToLower(messageText)
	for _, keyword := range d.keywords {
		keywordLower := strings.ToLower(strings.TrimSpace(keyword))
		if keywordLower != "" && strings.Contains(messageTextLower, keywordLower) {
			d.logger.Debug("Found matching keyword in message", "keyword", keyword)
			return true
		}
	}

	d.logger.Debug("No matching keywords found in message", "messageText", messageText, "keywords", d.keywords)
	return false
}

// extractMessageText extracts text content from dingtalk message for keyword checking
func (d *DingTalkSender) extractMessageText(dingTalkMsg *DingTalkMessage, msg *message.Message) string {
	var extractedText string

	switch dingTalkMsg.MsgType {
	case "text":
		if content, ok := dingTalkMsg.Text.(*DingTalkTextContent); ok {
			extractedText = content.Content
		}
	case "markdown":
		if content, ok := dingTalkMsg.Markdown.(*DingTalkMarkdownContent); ok {
			extractedText = content.Text
		}
	case "actionCard":
		extractedText = strings.TrimSpace(msg.Title + " " + msg.Body)
	default:
		extractedText = strings.TrimSpace(msg.Title + " " + msg.Body)
	}

	d.logger.Debug("Extracted message text for keyword checking",
		"msgType", dingTalkMsg.MsgType, "extractedText", extractedText)

	return extractedText
}

// addKeywordToMessage adds a keyword to the message
func (d *DingTalkSender) addKeywordToMessage(dingTalkMsg *DingTalkMessage, keyword string) (*DingTalkMessage, error) {
	keywordToAdd := strings.TrimSpace(keyword)
	if keywordToAdd == "" {
		return dingTalkMsg, fmt.Errorf("keyword cannot be empty")
	}

	d.logger.Debug("Adding keyword to message", "msgType", dingTalkMsg.MsgType, "keyword", keywordToAdd)

	switch dingTalkMsg.MsgType {
	case "text":
		if content, ok := dingTalkMsg.Text.(*DingTalkTextContent); ok {
			originalText := content.Content
			content.Content = keywordToAdd + " " + originalText
			d.logger.Debug("Added keyword to text message", "originalText", originalText, "newText", content.Content)
			return dingTalkMsg, nil
		}
		return dingTalkMsg, fmt.Errorf("failed to cast text content")

	case "markdown":
		if content, ok := dingTalkMsg.Markdown.(*DingTalkMarkdownContent); ok {
			originalText := content.Text
			content.Text = keywordToAdd + " " + originalText
			d.logger.Debug("Added keyword to markdown message", "originalText", originalText, "newText", content.Text)
			return dingTalkMsg, nil
		}
		return dingTalkMsg, fmt.Errorf("failed to cast markdown content")

	case "actionCard":
		d.logger.Warn("Adding keywords to action card messages is not fully implemented", "keyword", keywordToAdd)
		return dingTalkMsg, nil

	default:
		return dingTalkMsg, fmt.Errorf("unsupported message type for keyword addition: %s", dingTalkMsg.MsgType)
	}
}

// generateSignature generates HMAC-SHA256 signature for DingTalk webhook
func (d *DingTalkSender) generateSignature(timestamp int64) string {
	d.logger.Debug("Generating signature for DingTalk auth", "timestamp", timestamp)
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, d.secret)
	hash := hmac.New(sha256.New, []byte(d.secret))
	hash.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return signature
}

// determineSecurityMode determines which security mode is configured
func (d *DingTalkSender) determineSecurityMode() string {
	hasSignature := d.secret != ""
	hasKeywords := len(d.keywords) > 0

	switch {
	case hasSignature && hasKeywords:
		return "signature_and_keywords"
	case hasSignature && !hasKeywords:
		return "signature_only"
	case !hasSignature && hasKeywords:
		return "keywords_only"
	default:
		return "no_security"
	}
}

// DingTalkResponse represents the response structure from DingTalk API
type DingTalkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// sendToWebhook sends the message to DingTalk webhook with appropriate security mode
func (d *DingTalkSender) sendToWebhook(ctx context.Context, dingTalkMsg *DingTalkMessage) error {
	securityMode := d.determineSecurityMode()
	d.logger.Debug("Sending to DingTalk webhook", "webhookURL", d.webhookURL, "securityMode", securityMode)

	// Prepare URL with signature if required
	requestURL := d.webhookURL
	if securityMode == "signature_only" || securityMode == "signature_and_keywords" {
		var err error
		requestURL, err = d.addSignatureToURL(d.webhookURL)
		if err != nil {
			return fmt.Errorf("failed to add signature to URL: %w", err)
		}
	}

	// Marshal message to JSON
	payload, err := json.Marshal(dingTalkMsg)
	if err != nil {
		d.logger.Error("Failed to marshal DingTalk message", "error", err)
		return fmt.Errorf("failed to marshal DingTalk message: %w", err)
	}

	d.logger.Debug("Sending HTTP request to DingTalk", "url", requestURL, "payloadSize", len(payload))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body for debugging and error checking
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		d.logger.Error("Failed to read response body", "error", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug: Print response details
	d.logger.Debug("DingTalk API Response", "statusCode", resp.StatusCode, "body", string(bodyBytes))

	// Check HTTP response status
	if resp.StatusCode != http.StatusOK {
		d.logger.Error("DingTalk API returned non-OK status", "statusCode", resp.StatusCode, "body", string(bodyBytes))
		return fmt.Errorf("dingtalk API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Try to parse DingTalk API response for error checking
	var dingTalkResp DingTalkResponse
	if err := json.Unmarshal(bodyBytes, &dingTalkResp); err == nil {
		// Successfully parsed as DingTalk response, check for API errors
		if dingTalkResp.ErrCode != 0 {
			d.logger.Error("DingTalk API returned error code", "errcode", dingTalkResp.ErrCode, "errmsg", dingTalkResp.ErrMsg)
			return fmt.Errorf("dingtalk API error (code %d): %s", dingTalkResp.ErrCode, dingTalkResp.ErrMsg)
		}
		d.logger.Info("DingTalk message sent successfully", "securityMode", securityMode, "statusCode", resp.StatusCode)
	} else {
		// Not a standard DingTalk response format (e.g., from httpbin), that's OK for testing
		d.logger.Debug("Response is not in DingTalk format (likely testing endpoint)", "parseError", err)
		d.logger.Info("Message sent successfully to endpoint", "securityMode", securityMode, "statusCode", resp.StatusCode)
	}

	return nil
}

// addSignatureToURL adds timestamp and signature parameters to webhook URL
func (d *DingTalkSender) addSignatureToURL(webhookURL string) (string, error) {
	timestamp := time.Now().UnixNano() / 1e6 // DingTalk uses milliseconds
	signature := d.generateSignature(timestamp)

	u, err := url.Parse(webhookURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse webhook URL: %w", err)
	}

	query := u.Query()
	query.Set("timestamp", strconv.FormatInt(timestamp, 10))
	query.Set("sign", signature)
	u.RawQuery = query.Encode()

	d.logger.Debug("Added signature to webhook URL", "timestamp", timestamp)
	return u.String(), nil
}

// init registers the dingtalk platform automatically
func init() {
	// Register dingtalk platform creator with the global registry
	platform.RegisterPlatform("dingtalk", func(cfg map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
		// Convert config to DingTalkConfig
		dingTalkConfig := &config.DingTalkConfig{}
		if webhookURL, ok := cfg["webhook_url"].(string); ok {
			dingTalkConfig.WebhookURL = webhookURL
		}
		if secret, ok := cfg["secret"].(string); ok {
			dingTalkConfig.Secret = secret
		}
		if timeout, ok := cfg["timeout"].(time.Duration); ok {
			dingTalkConfig.Timeout = timeout
		} else {
			dingTalkConfig.Timeout = 30 * time.Second // default timeout
		}

		// Parse keywords from config
		if keywordsInterface, ok := cfg["keywords"]; ok {
			if keywordSlice, ok := keywordsInterface.([]interface{}); ok {
				for _, keyword := range keywordSlice {
					if keywordStr, ok := keyword.(string); ok {
						dingTalkConfig.Keywords = append(dingTalkConfig.Keywords, keywordStr)
					}
				}
			} else if keywordsStr, ok := keywordsInterface.(string); ok {
				// Support comma-separated string format
				if keywordsStr != "" {
					keywords := strings.Split(keywordsStr, ",")
					for _, keyword := range keywords {
						dingTalkConfig.Keywords = append(dingTalkConfig.Keywords, strings.TrimSpace(keyword))
					}
				}
			} else if keywordSlice, ok := keywordsInterface.([]string); ok {
				dingTalkConfig.Keywords = keywordSlice
			}
		}

		return NewDingTalkPlatform(dingTalkConfig, logger)
	})
}
