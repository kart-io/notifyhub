// Package wechatwork provides WeChat Work (企业微信) platform integration for NotifyHub
// This is an external platform implementation that demonstrates how to create custom platforms
package wechatwork

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
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// WeChatWorkSender implements the Platform interface for WeChat Work webhooks
type WeChatWorkSender struct {
	webhookURL string
	secret     string
	keywords   []string
	client     *http.Client
	logger     logger.Logger
}

// WeChatWorkMessage represents a WeChat Work webhook message structure
type WeChatWorkMessage struct {
	MsgType  string      `json:"msgtype"`
	Text     interface{} `json:"text,omitempty"`
	Markdown interface{} `json:"markdown,omitempty"`
	Image    interface{} `json:"image,omitempty"`
	News     interface{} `json:"news,omitempty"`
	File     interface{} `json:"file,omitempty"`
}

// WeChatWorkTextContent represents text content for WeChat Work
type WeChatWorkTextContent struct {
	Content             string   `json:"content"`
	MentionedList       []string `json:"mentioned_list,omitempty"`
	MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
}

// WeChatWorkMarkdownContent represents markdown content for WeChat Work
type WeChatWorkMarkdownContent struct {
	Content string `json:"content"`
}

// WeChatWorkImageContent represents image content for WeChat Work
type WeChatWorkImageContent struct {
	Base64 string `json:"base64"`
	MD5    string `json:"md5"`
}

// WeChatWorkNewsContent represents news content for WeChat Work
type WeChatWorkNewsContent struct {
	Articles []WeChatWorkArticle `json:"articles"`
}

// WeChatWorkArticle represents a news article
type WeChatWorkArticle struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	PicURL      string `json:"picurl"`
}

// WeChatWorkConfig represents WeChat Work-specific configuration
type WeChatWorkConfig struct {
	WebhookURL string
	Secret     string
	Keywords   []string
	Timeout    time.Duration
}

// NewWeChatWorkSender creates a new WeChat Work sender with configuration
func NewWeChatWorkSender(config *WeChatWorkConfig, logger logger.Logger) (*WeChatWorkSender, error) {
	if config.WebhookURL == "" {
		return nil, fmt.Errorf("wechatwork webhook URL is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &WeChatWorkSender{
		webhookURL: config.WebhookURL,
		secret:     config.Secret,
		keywords:   config.Keywords,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}, nil
}

// NewWeChatWorkSenderFromMap creates a new WeChat Work sender with map configuration
func NewWeChatWorkSenderFromMap(config map[string]interface{}, logger logger.Logger) (*WeChatWorkSender, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("wechatwork webhook_url is required")
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

	weChatWorkConfig := &WeChatWorkConfig{
		WebhookURL: webhookURL,
		Secret:     secret,
		Keywords:   keywords,
		Timeout:    timeout,
	}

	return NewWeChatWorkSender(weChatWorkConfig, logger)
}

// Name returns the platform name
func (w *WeChatWorkSender) Name() string {
	return "wechatwork"
}

// Send implements the Platform interface
func (w *WeChatWorkSender) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	// Filter targets for WeChat Work
	weChatWorkTargets := w.filterWeChatWorkTargets(targets)
	if len(weChatWorkTargets) == 0 {
		w.logger.Info("No WeChat Work targets found, skipping")
		// Return failure results for non-wechatwork targets
		for i, target := range targets {
			results[i] = &platform.SendResult{
				Target:  target,
				Success: false,
				Error:   "not a wechatwork target",
			}
		}
		return results, nil
	}

	// Send to each target
	for i, target := range targets {
		if w.isWeChatWorkTarget(target) {
			err := w.sendSingleMessage(ctx, msg, target)
			if err != nil {
				results[i] = &platform.SendResult{
					Target:  target,
					Success: false,
					Error:   err.Error(),
				}
			} else {
				messageID := msg.ID
				if messageID == "" {
					messageID = fmt.Sprintf("wechatwork_%d", time.Now().UnixNano())
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
				Error:   "not a wechatwork target",
			}
		}
	}

	return results, nil
}

// sendSingleMessage sends a message to a single WeChat Work target
func (w *WeChatWorkSender) sendSingleMessage(ctx context.Context, msg *message.Message, target target.Target) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Build WeChat Work message
	weChatWorkMsg, err := w.buildWeChatWorkMessage(msg)
	if err != nil {
		w.logger.Error("Failed to build WeChat Work message", "error", err)
		return fmt.Errorf("failed to build WeChat Work message: %w", err)
	}

	// Send to webhook
	if err := w.sendToWebhook(ctx, weChatWorkMsg); err != nil {
		w.logger.Error("Failed to send to WeChat Work webhook", "error", err)
		return fmt.Errorf("failed to send to WeChat Work webhook: %w", err)
	}

	w.logger.Info("WeChat Work message sent successfully", "messageID", msg.ID, "target", target.Value)
	return nil
}

// ValidateTarget implements the Platform interface
func (w *WeChatWorkSender) ValidateTarget(target target.Target) error {
	if target.Type != "wechatwork" && target.Type != "webhook" {
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}
	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}
	return nil
}

// IsHealthy implements the Platform interface
func (w *WeChatWorkSender) IsHealthy(ctx context.Context) error {
	// Simple health check - verify webhook URL is accessible
	if w.webhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}
	return nil
}

// Close implements the Platform interface
func (w *WeChatWorkSender) Close() error {
	w.logger.Info("Closing WeChat Work sender")
	// Close HTTP client if needed
	if w.client != nil {
		w.client.CloseIdleConnections()
	}
	return nil
}

// GetCapabilities implements the Platform interface
func (w *WeChatWorkSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "wechatwork",
		SupportedTargetTypes: []string{"wechatwork", "webhook"},
		SupportedFormats:     []string{"text", "markdown", "image", "news"},
		MaxMessageSize:       4096, // WeChat Work limit
	}
}

// isWeChatWorkTarget checks if a target is relevant for WeChat Work
func (w *WeChatWorkSender) isWeChatWorkTarget(target target.Target) bool {
	return target.Type == "wechatwork" || target.Type == "webhook"
}

// filterWeChatWorkTargets filters targets that are relevant for WeChat Work
func (w *WeChatWorkSender) filterWeChatWorkTargets(targets []target.Target) []target.Target {
	var weChatWorkTargets []target.Target
	for _, t := range targets {
		if w.isWeChatWorkTarget(t) {
			weChatWorkTargets = append(weChatWorkTargets, t)
		}
	}
	return weChatWorkTargets
}

// buildWeChatWorkMessage builds a WeChat Work-specific message format with security processing
func (w *WeChatWorkSender) buildWeChatWorkMessage(msg *message.Message) (*WeChatWorkMessage, error) {
	securityMode := w.determineSecurityMode()
	w.logger.Info("Building WeChat Work message with security mode", "securityMode", securityMode)

	// Build basic message structure
	weChatWorkMsg, err := w.buildBasicMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to build basic message: %w", err)
	}

	// Apply security mode processing
	return w.applySecurityModeProcessing(weChatWorkMsg, msg, securityMode)
}

// buildBasicMessage builds the basic WeChat Work message structure
func (w *WeChatWorkSender) buildBasicMessage(msg *message.Message) (*WeChatWorkMessage, error) {
	// Check for platform-specific data
	if newsData, exists := msg.PlatformData["wechatwork_news"]; exists {
		w.logger.Debug("Using platform-specific news data")
		return &WeChatWorkMessage{
			MsgType: "news",
			News:    newsData,
		}, nil
	}

	if imageData, exists := msg.PlatformData["wechatwork_image"]; exists {
		w.logger.Debug("Using platform-specific image data")
		return &WeChatWorkMessage{
			MsgType: "image",
			Image:   imageData,
		}, nil
	}

	// Build message based on format
	switch msg.Format {
	case message.FormatHTML, message.FormatMarkdown:
		w.logger.Debug("Building markdown message for HTML/Markdown format")
		content := w.buildMarkdownContent(msg)
		return &WeChatWorkMessage{
			MsgType:  "markdown",
			Markdown: content,
		}, nil

	default:
		w.logger.Debug("Building text message for default format")
		text := w.buildTextContent(msg)
		return &WeChatWorkMessage{
			MsgType: "text",
			Text: &WeChatWorkTextContent{
				Content: text,
			},
		}, nil
	}
}

// buildTextContent builds text content from message
func (w *WeChatWorkSender) buildTextContent(msg *message.Message) string {
	text := msg.Title
	if msg.Body != "" {
		if text != "" {
			text += "\n\n"
		}
		text += msg.Body
	}
	return text
}

// buildMarkdownContent builds markdown content for WeChat Work
func (w *WeChatWorkSender) buildMarkdownContent(msg *message.Message) *WeChatWorkMarkdownContent {
	content := msg.Body
	if msg.Title != "" && msg.Body != "" {
		content = fmt.Sprintf("# %s\n\n%s", msg.Title, msg.Body)
	} else if msg.Title != "" {
		content = fmt.Sprintf("# %s", msg.Title)
	}

	return &WeChatWorkMarkdownContent{
		Content: content,
	}
}

// applySecurityModeProcessing applies security mode specific processing
func (w *WeChatWorkSender) applySecurityModeProcessing(weChatWorkMsg *WeChatWorkMessage, msg *message.Message, securityMode string) (*WeChatWorkMessage, error) {
	switch securityMode {
	case "no_security":
		w.logger.Info("Security Mode 1: 无需配置签名与自定义关键词 - 直接发送消息")
		return weChatWorkMsg, nil

	case "signature_only":
		w.logger.Info("Security Mode 2: 仅配置签名验证 - 消息无需修改，签名将在发送时添加")
		return weChatWorkMsg, nil

	case "keywords_only":
		w.logger.Info("Security Mode 3: 仅配置自定义关键词 - 处理关键词验证")
		return w.processKeywordRequirement(weChatWorkMsg, msg)

	case "signature_and_keywords":
		w.logger.Info("Security Mode 2+3: 同时配置签名验证和自定义关键词")
		return w.processKeywordRequirement(weChatWorkMsg, msg)

	default:
		w.logger.Warn("Unknown security mode, using no security", "mode", securityMode)
		return weChatWorkMsg, nil
	}
}

// processKeywordRequirement processes keyword requirement for the message
func (w *WeChatWorkSender) processKeywordRequirement(weChatWorkMsg *WeChatWorkMessage, msg *message.Message) (*WeChatWorkMessage, error) {
	if len(w.keywords) == 0 {
		w.logger.Debug("No keywords configured, skipping keyword processing")
		return weChatWorkMsg, nil
	}

	w.logger.Debug("Processing keyword requirement", "configuredKeywords", w.keywords)

	// Extract message text content
	messageText := w.extractMessageText(weChatWorkMsg, msg)

	// Check if message contains required keyword
	if w.containsRequiredKeyword(messageText) {
		w.logger.Debug("Message already contains required keyword, no modification needed")
		return weChatWorkMsg, nil
	}

	// Add first keyword to message
	keywordToAdd := w.keywords[0]
	w.logger.Info("Message doesn't contain required keywords, automatically adding keyword",
		"keyword", keywordToAdd, "availableKeywords", w.keywords)

	return w.addKeywordToMessage(weChatWorkMsg, keywordToAdd)
}

// containsRequiredKeyword checks if the message contains any of the required keywords
func (w *WeChatWorkSender) containsRequiredKeyword(messageText string) bool {
	if messageText == "" {
		return false
	}

	messageTextLower := strings.ToLower(messageText)
	for _, keyword := range w.keywords {
		keywordLower := strings.ToLower(strings.TrimSpace(keyword))
		if keywordLower != "" && strings.Contains(messageTextLower, keywordLower) {
			w.logger.Debug("Found matching keyword in message", "keyword", keyword)
			return true
		}
	}

	w.logger.Debug("No matching keywords found in message", "messageText", messageText, "keywords", w.keywords)
	return false
}

// extractMessageText extracts text content from WeChat Work message for keyword checking
func (w *WeChatWorkSender) extractMessageText(weChatWorkMsg *WeChatWorkMessage, msg *message.Message) string {
	var extractedText string

	switch weChatWorkMsg.MsgType {
	case "text":
		if content, ok := weChatWorkMsg.Text.(*WeChatWorkTextContent); ok {
			extractedText = content.Content
		}
	case "markdown":
		if content, ok := weChatWorkMsg.Markdown.(*WeChatWorkMarkdownContent); ok {
			extractedText = content.Content
		}
	case "news", "image":
		extractedText = strings.TrimSpace(msg.Title + " " + msg.Body)
	default:
		extractedText = strings.TrimSpace(msg.Title + " " + msg.Body)
	}

	w.logger.Debug("Extracted message text for keyword checking",
		"msgType", weChatWorkMsg.MsgType, "extractedText", extractedText)

	return extractedText
}

// addKeywordToMessage adds a keyword to the message
func (w *WeChatWorkSender) addKeywordToMessage(weChatWorkMsg *WeChatWorkMessage, keyword string) (*WeChatWorkMessage, error) {
	keywordToAdd := strings.TrimSpace(keyword)
	if keywordToAdd == "" {
		return weChatWorkMsg, fmt.Errorf("keyword cannot be empty")
	}

	w.logger.Debug("Adding keyword to message", "msgType", weChatWorkMsg.MsgType, "keyword", keywordToAdd)

	switch weChatWorkMsg.MsgType {
	case "text":
		if content, ok := weChatWorkMsg.Text.(*WeChatWorkTextContent); ok {
			originalText := content.Content
			content.Content = keywordToAdd + " " + originalText
			w.logger.Debug("Added keyword to text message", "originalText", originalText, "newText", content.Content)
			return weChatWorkMsg, nil
		}
		return weChatWorkMsg, fmt.Errorf("failed to cast text content")

	case "markdown":
		if content, ok := weChatWorkMsg.Markdown.(*WeChatWorkMarkdownContent); ok {
			originalText := content.Content
			content.Content = keywordToAdd + " " + originalText
			w.logger.Debug("Added keyword to markdown message", "originalText", originalText, "newText", content.Content)
			return weChatWorkMsg, nil
		}
		return weChatWorkMsg, fmt.Errorf("failed to cast markdown content")

	case "news", "image":
		w.logger.Warn("Adding keywords to news/image messages is not fully implemented", "keyword", keywordToAdd)
		return weChatWorkMsg, nil

	default:
		return weChatWorkMsg, fmt.Errorf("unsupported message type for keyword addition: %s", weChatWorkMsg.MsgType)
	}
}

// generateSignature generates base64 encoded signature for WeChat Work webhook
func (w *WeChatWorkSender) generateSignature(timestamp int64) string {
	w.logger.Debug("Generating signature for WeChat Work auth", "timestamp", timestamp)
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, w.secret)
	hash := hmac.New(sha256.New, []byte(stringToSign))
	hash.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return signature
}

// determineSecurityMode determines which security mode is configured
func (w *WeChatWorkSender) determineSecurityMode() string {
	hasSignature := w.secret != ""
	hasKeywords := len(w.keywords) > 0

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

// WeChatWorkResponse represents the response structure from WeChat Work API
type WeChatWorkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// sendToWebhook sends the message to WeChat Work webhook with appropriate security mode
func (w *WeChatWorkSender) sendToWebhook(ctx context.Context, weChatWorkMsg *WeChatWorkMessage) error {
	securityMode := w.determineSecurityMode()
	w.logger.Debug("Sending to WeChat Work webhook", "webhookURL", w.webhookURL, "securityMode", securityMode)

	// Add signature to message if required
	if securityMode == "signature_only" || securityMode == "signature_and_keywords" {
		timestamp := time.Now().Unix()
		signature := w.generateSignature(timestamp)

		// WeChat Work requires signature in URL parameters
		webhookURL := fmt.Sprintf("%s&timestamp=%d&sign=%s", w.webhookURL, timestamp, signature)
		w.webhookURL = webhookURL
		w.logger.Debug("Added signature to webhook URL", "timestamp", timestamp)
	}

	// Marshal message to JSON
	payload, err := json.Marshal(weChatWorkMsg)
	if err != nil {
		w.logger.Error("Failed to marshal WeChat Work message", "error", err)
		return fmt.Errorf("failed to marshal WeChat Work message: %w", err)
	}

	w.logger.Debug("Sending HTTP request to WeChat Work", "url", w.webhookURL, "payloadSize", len(payload))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", w.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body for debugging and error checking
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.Error("Failed to read response body", "error", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug: Print response details
	w.logger.Debug("WeChat Work API Response", "statusCode", resp.StatusCode, "body", string(bodyBytes))

	// Check HTTP response status
	if resp.StatusCode != http.StatusOK {
		w.logger.Error("WeChat Work API returned non-OK status", "statusCode", resp.StatusCode, "body", string(bodyBytes))
		return fmt.Errorf("wechatwork API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Try to parse WeChat Work API response for error checking
	var weChatWorkResp WeChatWorkResponse
	if err := json.Unmarshal(bodyBytes, &weChatWorkResp); err == nil {
		// Successfully parsed as WeChat Work response, check for API errors
		if weChatWorkResp.ErrCode != 0 {
			w.logger.Error("WeChat Work API returned error code", "errcode", weChatWorkResp.ErrCode, "errmsg", weChatWorkResp.ErrMsg)
			return fmt.Errorf("wechatwork API error (code %d): %s", weChatWorkResp.ErrCode, weChatWorkResp.ErrMsg)
		}
		w.logger.Info("WeChat Work message sent successfully", "securityMode", securityMode, "statusCode", resp.StatusCode)
	} else {
		// Not a standard WeChat Work response format (e.g., from httpbin), that's OK for testing
		w.logger.Debug("Response is not in WeChat Work format (likely testing endpoint)", "parseError", err)
		w.logger.Info("Message sent successfully to endpoint", "securityMode", securityMode, "statusCode", resp.StatusCode)
	}

	return nil
}

// RegisterWithNotifyHub registers this platform with NotifyHub's platform registry
// This function should be called to integrate the external platform
func RegisterWithNotifyHub() error {
	// Register wechatwork platform creator with the global registry
	platform.RegisterPlatform("wechatwork", func(cfg map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
		sender, err := NewWeChatWorkSenderFromMap(cfg, logger)
		if err != nil {
			return nil, err
		}
		// Return the sender as a Platform interface
		return platformAdapter{sender: sender}, nil
	})
	return nil
}

// platformAdapter wraps WeChatWorkSender to implement the Platform interface
type platformAdapter struct {
	sender *WeChatWorkSender
}

func (p platformAdapter) Name() string {
	return p.sender.Name()
}

func (p platformAdapter) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	return p.sender.Send(ctx, msg, targets)
}

func (p platformAdapter) ValidateTarget(target target.Target) error {
	return p.sender.ValidateTarget(target)
}

func (p platformAdapter) IsHealthy(ctx context.Context) error {
	return p.sender.IsHealthy(ctx)
}

func (p platformAdapter) Close() error {
	return p.sender.Close()
}

func (p platformAdapter) GetCapabilities() platform.Capabilities {
	return p.sender.GetCapabilities()
}
