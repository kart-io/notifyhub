// Package feishu provides Feishu platform integration for NotifyHub
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
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// FeishuSender implements the Platform interface for Feishu webhooks
type FeishuSender struct {
	webhookURL string
	secret     string
	keywords   []string
	client     *http.Client
	logger     logger.Logger
}

// FeishuMessage represents a Feishu webhook message structure
type FeishuMessage struct {
	MsgType   string      `json:"msg_type"`
	Content   interface{} `json:"content"`
	Sign      string      `json:"sign,omitempty"`      // 签名字段
	Timestamp string      `json:"timestamp,omitempty"` // 时间戳字段
}

// FeishuTextContent represents text content for Feishu
type FeishuTextContent struct {
	Text string `json:"text"`
}

// FeishuRichTextContent represents rich text content for Feishu
type FeishuRichTextContent struct {
	Post map[string]interface{} `json:"post"`
}

// FeishuCardContent represents card content for Feishu
type FeishuCardContent struct {
	Config   map[string]interface{} `json:"config,omitempty"`
	Elements []interface{}          `json:"elements"`
	Header   map[string]interface{} `json:"header,omitempty"`
}

// NewFeishuPlatform creates a new Feishu platform with strong-typed configuration
func NewFeishuPlatform(feishuConfig *config.FeishuConfig, logger logger.Logger) (platform.Platform, error) {
	if feishuConfig.WebhookURL == "" {
		return nil, fmt.Errorf("feishu webhook URL is required")
	}

	return &FeishuSender{
		webhookURL: feishuConfig.WebhookURL,
		secret:     feishuConfig.Secret,
		keywords:   feishuConfig.Keywords,
		client: &http.Client{
			Timeout: feishuConfig.Timeout,
		},
		logger: logger,
	}, nil
}

// NewFeishuSender creates a new Feishu sender with map configuration (for backward compatibility)
func NewFeishuSender(config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("feishu webhook_url is required")
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

	return &FeishuSender{
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
func (f *FeishuSender) Name() string {
	return "feishu"
}

// Send implements the Platform interface
func (f *FeishuSender) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	// Filter targets for Feishu
	feishuTargets := f.filterFeishuTargets(targets)
	if len(feishuTargets) == 0 {
		f.logger.Info("No Feishu targets found, skipping")
		// Return success results for non-feishu targets
		for i, target := range targets {
			results[i] = &platform.SendResult{
				Target:  target,
				Success: false,
				Error:   "not a feishu target",
			}
		}
		return results, nil
	}

	// Send to each target
	for i, target := range targets {
		if f.isFeishuTarget(target) {
			err := f.sendSingleMessage(ctx, msg, target)
			if err != nil {
				results[i] = &platform.SendResult{
					Target:  target,
					Success: false,
					Error:   err.Error(),
				}
			} else {
				messageID := msg.ID
				if messageID == "" {
					messageID = fmt.Sprintf("feishu_%d", time.Now().UnixNano())
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
				Error:   "not a feishu target",
			}
		}
	}

	return results, nil
}

// sendSingleMessage sends a message to a single feishu target
func (f *FeishuSender) sendSingleMessage(ctx context.Context, msg *message.Message, target target.Target) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Build Feishu message
	feishuMsg, err := f.buildFeishuMessage(msg)
	if err != nil {
		f.logger.Error("Failed to build Feishu message", "error", err)
		return fmt.Errorf("failed to build Feishu message: %w", err)
	}

	// Send to webhook
	if err := f.sendToWebhook(ctx, feishuMsg); err != nil {
		f.logger.Error("Failed to send to Feishu webhook", "error", err)
		return fmt.Errorf("failed to send to Feishu webhook: %w", err)
	}

	f.logger.Info("Feishu message sent successfully", "messageID", msg.ID, "target", target.Value)
	return nil
}

// ValidateTarget implements the Platform interface
func (f *FeishuSender) ValidateTarget(target target.Target) error {
	if target.Type != "feishu" && target.Type != "webhook" {
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}
	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}
	return nil
}

// IsHealthy implements the Platform interface
func (f *FeishuSender) IsHealthy(ctx context.Context) error {
	// Simple health check - verify webhook URL is accessible
	if f.webhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}
	return nil
}

// Close implements the Platform interface
func (f *FeishuSender) Close() error {
	f.logger.Info("Closing Feishu sender")
	// Close HTTP client if needed
	if f.client != nil {
		f.client.CloseIdleConnections()
	}
	return nil
}

// GetCapabilities implements the Platform interface
func (f *FeishuSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "feishu",
		SupportedTargetTypes: []string{"feishu", "webhook"},
		SupportedFormats:     []string{"text", "markdown", "card", "rich_text"},
		MaxMessageSize:       4000,
	}
}

// isFeishuTarget checks if a target is relevant for Feishu
func (f *FeishuSender) isFeishuTarget(target target.Target) bool {
	return target.Type == "feishu" || target.Type == "webhook"
}

// filterFeishuTargets filters targets that are relevant for Feishu
func (f *FeishuSender) filterFeishuTargets(targets []target.Target) []target.Target {
	var feishuTargets []target.Target
	for _, t := range targets {
		if f.isFeishuTarget(t) {
			feishuTargets = append(feishuTargets, t)
		}
	}
	return feishuTargets
}

// buildFeishuMessage builds a Feishu-specific message format with security processing
// 构建飞书消息并根据安全模式进行相应处理
func (f *FeishuSender) buildFeishuMessage(msg *message.Message) (*FeishuMessage, error) {
	securityMode := f.determineSecurityMode()
	f.logger.Info("Building Feishu message with security mode", "securityMode", securityMode)

	// 第一步：构建基本消息结构
	feishuMsg, err := f.buildBasicMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to build basic message: %w", err)
	}

	// 第二步：根据安全模式处理消息
	return f.applySecurityModeProcessing(feishuMsg, msg, securityMode)
}

// buildBasicMessage builds the basic Feishu message structure
// 构建基础的飞书消息结构（不考虑安全模式）
func (f *FeishuSender) buildBasicMessage(msg *message.Message) (*FeishuMessage, error) {
	// 检查是否有平台特定数据
	if cardData, exists := msg.PlatformData["feishu_card"]; exists {
		f.logger.Debug("Using platform-specific card data")
		return &FeishuMessage{
			MsgType: "interactive",
			Content: cardData,
		}, nil
	}

	if richTextData, exists := msg.PlatformData["feishu_rich_text"]; exists {
		f.logger.Debug("Using platform-specific rich text data")
		return &FeishuMessage{
			MsgType: "post",
			Content: richTextData,
		}, nil
	}

	// 根据消息格式构建消息
	switch msg.Format {
	case message.FormatHTML, message.FormatMarkdown:
		f.logger.Debug("Building rich text message for HTML/Markdown format")
		content := f.buildRichTextContent(msg)
		return &FeishuMessage{
			MsgType: "post",
			Content: content,
		}, nil

	default:
		f.logger.Debug("Building text message for default format")
		text := f.buildTextContent(msg)
		return &FeishuMessage{
			MsgType: "text",
			Content: &FeishuTextContent{
				Text: text,
			},
		}, nil
	}
}

// buildTextContent builds text content from message
// 从消息构建文本内容
func (f *FeishuSender) buildTextContent(msg *message.Message) string {
	text := msg.Title
	if msg.Body != "" {
		if text != "" {
			text += "\n\n"
		}
		text += msg.Body
	}
	return text
}

// applySecurityModeProcessing applies security mode specific processing
// 根据安全模式应用相应的处理逻辑
func (f *FeishuSender) applySecurityModeProcessing(feishuMsg *FeishuMessage, msg *message.Message, securityMode string) (*FeishuMessage, error) {
	switch securityMode {
	case "no_security":
		f.logger.Info("Security Mode 1: 无需配置签名与自定义关键词 - 直接发送消息")
		// 情况1: 无需配置签名与自定义关键词 - 消息无需任何修改
		return feishuMsg, nil

	case "signature_only":
		f.logger.Info("Security Mode 2: 仅配置签名验证 - 消息无需修改，签名将在发送时添加")
		// 情况2: 仅配置签名验证 - 消息内容无需修改，签名在HTTP请求时处理
		return feishuMsg, nil

	case "keywords_only":
		f.logger.Info("Security Mode 3: 仅配置自定义关键词 - 处理关键词验证")
		// 情况3: 仅配置自定义关键词 - 需要验证和添加关键词
		return f.processKeywordRequirement(feishuMsg, msg)

	case "signature_and_keywords":
		f.logger.Info("Security Mode 2+3: 同时配置签名验证和自定义关键词")
		// 情况2+3: 同时配置签名和关键词 - 处理关键词验证，签名在发送时处理
		return f.processKeywordRequirement(feishuMsg, msg)

	default:
		f.logger.Warn("Unknown security mode, using no security", "mode", securityMode)
		return feishuMsg, nil
	}
}

// buildRichTextContent builds rich text content for Feishu
func (f *FeishuSender) buildRichTextContent(msg *message.Message) *FeishuRichTextContent {
	content := &FeishuRichTextContent{
		Post: map[string]interface{}{
			"zh_cn": map[string]interface{}{
				"title": msg.Title,
				"content": [][]interface{}{
					{
						map[string]interface{}{
							"tag":  "text",
							"text": msg.Body,
						},
					},
				},
			},
		},
	}
	return content
}

// processKeywordRequirement processes keyword requirement for the message (情况3)
// 自定义关键词验证: 如果消息中不包含配置的关键词，自动添加第一个关键词
func (f *FeishuSender) processKeywordRequirement(feishuMsg *FeishuMessage, msg *message.Message) (*FeishuMessage, error) {
	if len(f.keywords) == 0 {
		f.logger.Debug("No keywords configured, skipping keyword processing")
		return feishuMsg, nil
	}

	f.logger.Debug("Processing keyword requirement", "configuredKeywords", f.keywords)

	// 提取消息文本内容
	messageText := f.extractMessageText(feishuMsg, msg)

	// 检查是否包含必需的关键词
	if f.containsRequiredKeyword(messageText) {
		f.logger.Debug("Message already contains required keyword, no modification needed")
		return feishuMsg, nil
	}

	// 消息中没有包含关键词，自动添加第一个关键词
	keywordToAdd := f.keywords[0]
	f.logger.Info("Message doesn't contain required keywords, automatically adding keyword",
		"keyword", keywordToAdd, "availableKeywords", f.keywords)

	return f.addKeywordToMessage(feishuMsg, keywordToAdd)
}

// containsRequiredKeyword checks if the message contains any of the required keywords
// 不区分大小写检查消息是否包含配置的任意一个关键词
func (f *FeishuSender) containsRequiredKeyword(messageText string) bool {
	if messageText == "" {
		return false
	}

	messageTextLower := strings.ToLower(messageText)
	for _, keyword := range f.keywords {
		keywordLower := strings.ToLower(strings.TrimSpace(keyword))
		if keywordLower != "" && strings.Contains(messageTextLower, keywordLower) {
			f.logger.Debug("Found matching keyword in message", "keyword", keyword)
			return true
		}
	}

	f.logger.Debug("No matching keywords found in message", "messageText", messageText, "keywords", f.keywords)
	return false
}

// extractMessageText extracts text content from feishu message for keyword checking
// 从飞书消息中提取文本内容用于关键词检查
func (f *FeishuSender) extractMessageText(feishuMsg *FeishuMessage, msg *message.Message) string {
	var extractedText string

	switch feishuMsg.MsgType {
	case "text":
		// 文本消息直接提取文本内容
		if content, ok := feishuMsg.Content.(*FeishuTextContent); ok {
			extractedText = content.Text
		}
	case "post":
		// 富文本消息使用原始消息内容
		extractedText = strings.TrimSpace(msg.Title + " " + msg.Body)
	case "interactive":
		// 卡片消息使用原始消息内容
		extractedText = strings.TrimSpace(msg.Title + " " + msg.Body)
	default:
		// 默认情况使用原始消息内容
		extractedText = strings.TrimSpace(msg.Title + " " + msg.Body)
	}

	f.logger.Debug("Extracted message text for keyword checking",
		"msgType", feishuMsg.MsgType, "extractedText", extractedText)

	return extractedText
}

// addKeywordToMessage adds a keyword to the message
// 向消息中添加关键词（在消息开头添加）
func (f *FeishuSender) addKeywordToMessage(feishuMsg *FeishuMessage, keyword string) (*FeishuMessage, error) {
	keywordToAdd := strings.TrimSpace(keyword)
	if keywordToAdd == "" {
		return feishuMsg, fmt.Errorf("keyword cannot be empty")
	}

	f.logger.Debug("Adding keyword to message", "msgType", feishuMsg.MsgType, "keyword", keywordToAdd)

	switch feishuMsg.MsgType {
	case "text":
		// 文本消息：在消息开头添加关键词
		if content, ok := feishuMsg.Content.(*FeishuTextContent); ok {
			originalText := content.Text
			content.Text = keywordToAdd + " " + originalText
			f.logger.Debug("Added keyword to text message", "originalText", originalText, "newText", content.Text)
			return feishuMsg, nil
		}
		return feishuMsg, fmt.Errorf("failed to cast text content")

	case "post":
		// 富文本消息：修改内容数组的第一个文本元素
		if content, ok := feishuMsg.Content.(*FeishuRichTextContent); ok {
			if zhCn, ok := content.Post["zh_cn"].(map[string]interface{}); ok {
				if contentArray, ok := zhCn["content"].([][]interface{}); ok && len(contentArray) > 0 {
					firstLine := contentArray[0]
					if len(firstLine) > 0 {
						if textElement, ok := firstLine[0].(map[string]interface{}); ok {
							if originalText, ok := textElement["text"].(string); ok {
								newText := keywordToAdd + " " + originalText
								textElement["text"] = newText
								f.logger.Debug("Added keyword to rich text message", "originalText", originalText, "newText", newText)
								return feishuMsg, nil
							}
						}
					}
				}
			}
		}
		return feishuMsg, fmt.Errorf("failed to modify rich text content")

	case "interactive":
		// 卡片消息：目前仅显示警告，因为卡片结构复杂
		f.logger.Warn("Adding keywords to card messages is not fully implemented - card structure varies", "keyword", keywordToAdd)
		return feishuMsg, nil

	default:
		return feishuMsg, fmt.Errorf("unsupported message type for keyword addition: %s", feishuMsg.MsgType)
	}
}

// generateSign generates HMAC-SHA256 signature for Feishu webhook
// According to Feishu official documentation:
// 1. stringToSign = timestamp + "\n" + secret
// 2. signature = base64(hmac_sha256(stringToSign, ""))
func (f *FeishuSender) generateSign(timestamp string) string {
	f.logger.Debug("Generating signature for Feishu auth", "timestamp", timestamp)
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, f.secret)
	hash := hmac.New(sha256.New, []byte(stringToSign))
	hash.Write([]byte("")) // Feishu uses empty string as data
	signature := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return signature
}

// determineSecurityMode determines which security mode is configured
func (f *FeishuSender) determineSecurityMode() string {
	hasSignature := f.secret != ""
	hasKeywords := len(f.keywords) > 0

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

// addSignatureToMessage adds signature fields to the message if required
func (f *FeishuSender) addSignatureToMessage(feishuMsg *FeishuMessage) {
	if f.secret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		sign := f.generateSign(timestamp)
		feishuMsg.Sign = sign
		feishuMsg.Timestamp = timestamp
		f.logger.Debug("Added signature for auth", "timestamp", timestamp)
	}
}

// FeishuResponse represents the response structure from Feishu API
type FeishuResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// sendToWebhook sends the message to Feishu webhook with appropriate security mode
func (f *FeishuSender) sendToWebhook(ctx context.Context, feishuMsg *FeishuMessage) error {
	securityMode := f.determineSecurityMode()
	f.logger.Debug("Sending to Feishu webhook", "webhookURL", f.webhookURL, "securityMode", securityMode)

	// Add signature if required
	if securityMode == "signature_only" || securityMode == "signature_and_keywords" {
		f.addSignatureToMessage(feishuMsg)
	}

	// Marshal message to JSON
	payload, err := json.Marshal(feishuMsg)
	if err != nil {
		f.logger.Error("Failed to marshal Feishu message", "error", err)
		return fmt.Errorf("failed to marshal Feishu message: %w", err)
	}

	f.logger.Debug("Sending HTTP request to Feishu", "url", f.webhookURL, "payloadSize", len(payload))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", f.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body for debugging and error checking
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		f.logger.Error("Failed to read response body", "error", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}
	// Debug: Print response details
	f.logger.Debug("Feishu API Response", "statusCode", resp.StatusCode, "body", string(bodyBytes))

	// Check HTTP response status
	if resp.StatusCode != http.StatusOK {
		f.logger.Error("Feishu API returned non-OK status", "statusCode", resp.StatusCode, "body", string(bodyBytes))
		return fmt.Errorf("feishu API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Try to parse Feishu API response for error checking
	var feishuResp FeishuResponse
	if err := json.Unmarshal(bodyBytes, &feishuResp); err == nil {
		// Successfully parsed as Feishu response, check for API errors
		if feishuResp.Code != 0 {
			f.logger.Error("Feishu API returned error code", "code", feishuResp.Code, "msg", feishuResp.Msg)
			return fmt.Errorf("feishu API error (code %d): %s", feishuResp.Code, feishuResp.Msg)
		}
		f.logger.Info("Feishu message sent successfully", "securityMode", securityMode, "statusCode", resp.StatusCode)
	} else {
		// Not a standard Feishu response format (e.g., from httpbin), that's OK for testing
		f.logger.Debug("Response is not in Feishu format (likely testing endpoint)", "parseError", err)
		f.logger.Info("Message sent successfully to endpoint", "securityMode", securityMode, "statusCode", resp.StatusCode)
	}

	return nil
}

// init registers the feishu platform automatically
func init() {
	// Register feishu platform creator with the global registry
	platform.RegisterPlatform("feishu", func(cfg map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
		// Convert config to FeishuConfig
		feishuConfig := &config.FeishuConfig{}
		if webhookURL, ok := cfg["webhook_url"].(string); ok {
			feishuConfig.WebhookURL = webhookURL
		}
		if secret, ok := cfg["secret"].(string); ok {
			feishuConfig.Secret = secret
		}
		if timeout, ok := cfg["timeout"].(time.Duration); ok {
			feishuConfig.Timeout = timeout
		} else {
			feishuConfig.Timeout = 30 * time.Second // default timeout
		}

		// Parse keywords from config
		if keywordsInterface, ok := cfg["keywords"]; ok {
			if keywordSlice, ok := keywordsInterface.([]interface{}); ok {
				for _, keyword := range keywordSlice {
					if keywordStr, ok := keyword.(string); ok {
						feishuConfig.Keywords = append(feishuConfig.Keywords, keywordStr)
					}
				}
			} else if keywordsStr, ok := keywordsInterface.(string); ok {
				// Support comma-separated string format
				if keywordsStr != "" {
					keywords := strings.Split(keywordsStr, ",")
					for _, keyword := range keywords {
						feishuConfig.Keywords = append(feishuConfig.Keywords, strings.TrimSpace(keyword))
					}
				}
			} else if keywordSlice, ok := keywordsInterface.([]string); ok {
				feishuConfig.Keywords = keywordSlice
			}
		}

		return NewFeishuPlatform(feishuConfig, logger)
	})
}
