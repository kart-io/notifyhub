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
	"net/url"
	"strconv"
	"time"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/core/errors"
)

// Transport implements the Transport interface for Feishu
type Transport struct {
	webhookURL string
	timeout    time.Duration
	client     *http.Client

	// 安全设置 - 四选一
	securityType   SecurityType
	secret         string   // 签名校验
	customKeywords []string // 自定义关键词
	proxyURL       string   // 代理服务器URL
}

// SecurityType represents the type of security setting for Feishu webhook
type SecurityType string

const (
	SecurityTypeNone        SecurityType = "none"         // 无安全设置
	SecurityTypeSignature   SecurityType = "signature"    // 签名校验
	SecurityTypeIPWhitelist SecurityType = "ip_whitelist" // IP白名单
	SecurityTypeKeywords    SecurityType = "keywords"     // 自定义关键词
)

// Config holds the configuration for Feishu transport
type Config struct {
	WebhookURL string
	Timeout    time.Duration

	// 安全设置 - 四选一
	SecurityType   SecurityType
	Secret         string   // 用于签名校验
	CustomKeywords []string // 自定义关键词
	ProxyURL       string   // 代理服务器URL（可选）
}

// NewTransport creates a new Feishu transport
func NewTransport(config *Config) *Transport {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// 设置默认安全类型
	if config.SecurityType == "" {
		if config.Secret != "" {
			config.SecurityType = SecurityTypeSignature
		} else if len(config.CustomKeywords) > 0 {
			config.SecurityType = SecurityTypeKeywords
		} else {
			config.SecurityType = SecurityTypeNone
		}
	}

	// 创建HTTP客户端，支持代理
	transport := &http.Transport{}
	if config.ProxyURL != "" {
		// 使用指定的代理URL
		proxyURL, err := url.Parse(config.ProxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	} else {
		// 使用系统环境变量代理（HTTP_PROXY, HTTPS_PROXY, NO_PROXY等）
		// Go标准库会自动处理大小写变体
		transport.Proxy = http.ProxyFromEnvironment
	}

	return &Transport{
		webhookURL:     config.WebhookURL,
		timeout:        config.Timeout,
		securityType:   config.SecurityType,
		secret:         config.Secret,
		customKeywords: config.CustomKeywords,
		proxyURL:       config.ProxyURL,
		client: &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		},
	}
}

// Name returns the transport name
func (t *Transport) Name() string {
	return "feishu"
}

// Send sends a message through Feishu
func (t *Transport) Send(ctx context.Context, msg *core.Message, target core.Target) (*core.Result, error) {
	result := core.NewResult(msg.ID, target)
	result.SetStatus(core.StatusSending)

	// Convert message to Feishu format
	payload, err := t.buildPayload(msg, target)
	if err != nil {
		result.SetError(fmt.Errorf("failed to build payload: %w", err))
		return result, err
	}

	// Send the request
	response, err := t.sendRequest(ctx, payload)
	if err != nil {
		result.SetError(fmt.Errorf("failed to send request: %w", err))
		return result, err
	}

	// Check response
	if response.Code != 0 {
		// Map Feishu API error codes to standard errors
		var notifyErr *errors.NotifyError
		switch response.Code {
		case 9499: // Invalid webhook URL
			notifyErr = errors.NewFeishuError(errors.CodeInvalidConfig, fmt.Sprintf("Invalid webhook URL: %s", response.Msg))
		case 19001: // Invalid app_id
			notifyErr = errors.NewFeishuError(errors.CodeInvalidCredentials, fmt.Sprintf("Invalid app_id: %s", response.Msg))
		case 19002: // Invalid signature
			notifyErr = errors.NewFeishuError(errors.CodeInvalidCredentials, fmt.Sprintf("Invalid signature: %s", response.Msg))
		case 19003: // Request too frequent
			notifyErr = errors.NewFeishuError(errors.CodeRateLimited, fmt.Sprintf("Rate limited: %s", response.Msg))
		default:
			notifyErr = errors.NewFeishuError(errors.CodeSendingFailed, fmt.Sprintf("Feishu API error (code=%d): %s", response.Code, response.Msg))
		}
		result.SetError(notifyErr)
		return result, notifyErr
	}

	result.Status = core.StatusSent
	result.Response = response
	return result, nil
}

// buildPayload builds the Feishu webhook payload
func (t *Transport) buildPayload(msg *core.Message, target core.Target) (*FeishuPayload, error) {
	payload := &FeishuPayload{
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10), // 使用秒级时间戳
		MsgType:   "text",                                   // Default to text
	}

	// 根据安全类型添加相应的验证信息
	switch t.securityType {
	case SecurityTypeSignature:
		if t.secret != "" {
			signature, err := t.generateSignature(payload.Timestamp)
			if err != nil {
				return nil, fmt.Errorf("failed to generate signature: %w", err)
			}
			payload.Sign = signature
		}
	case SecurityTypeIPWhitelist:
		// IP白名单验证在网络层处理，这里不需要额外处理
	case SecurityTypeKeywords:
		// 自定义关键词会在消息内容中体现
	case SecurityTypeNone:
		// 无安全验证
	}

	// Build content based on message format
	switch msg.Format {
	case core.FormatText:
		payload.Content = &FeishuContent{
			Text: t.buildTextContent(msg),
		}
	case core.FormatMarkdown:
		payload.MsgType = "post"
		payload.Content = &FeishuContent{
			Post: t.buildPostContent(msg),
		}
	case core.FormatCard:
		payload.MsgType = "interactive"
		if msg.CardData != nil {
			payload.Card = msg.CardData
		} else {
			// Build default card
			payload.Card = t.buildDefaultCard(msg)
		}
	default:
		payload.Content = &FeishuContent{
			Text: t.buildTextContent(msg),
		}
	}

	return payload, nil
}

// buildTextContent builds text content
func (t *Transport) buildTextContent(msg *core.Message) string {
	content := ""
	if msg.Title != "" {
		content += msg.Title + "\n"
	}
	content += msg.Body

	// 只有在使用关键词安全模式时才添加关键词标签
	if t.securityType == SecurityTypeKeywords && len(t.customKeywords) > 0 {
		content = t.enhanceContentWithKeywords(content)
	}

	return content
}

// enhanceContentWithKeywords 在消息内容中应用自定义关键词增强
func (t *Transport) enhanceContentWithKeywords(content string) string {
	if len(t.customKeywords) == 0 {
		return content
	}

	// 在消息末尾添加关键词标签
	content += "\n\n🏷️ 标签: "
	for i, keyword := range t.customKeywords {
		if i > 0 {
			content += " | "
		}
		content += "#" + keyword
	}

	return content
}

// buildPostContent builds post (markdown) content
func (t *Transport) buildPostContent(msg *core.Message) *FeishuPost {
	post := &FeishuPost{
		ZhCn: &FeishuPostContent{
			Title: msg.Title,
			Content: [][]FeishuPostElement{
				{
					{
						Tag:  "text",
						Text: msg.Body,
					},
				},
			},
		},
	}
	return post
}

// buildDefaultCard builds a default card
func (t *Transport) buildDefaultCard(msg *core.Message) interface{} {
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": []interface{}{},
	}

	// Add title if present
	if msg.Title != "" {
		titleElement := map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": msg.Title,
				"tag":     "lark_md",
			},
		}
		card["elements"] = append(card["elements"].([]interface{}), titleElement)
	}

	// Add body
	if msg.Body != "" {
		bodyElement := map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": msg.Body,
				"tag":     "lark_md",
			},
		}
		card["elements"] = append(card["elements"].([]interface{}), bodyElement)
	}

	return card
}

// generateSignature generates HMAC-SHA256 signature
// 根据飞书官方算法：使用timestamp+"\n"+secret作为密钥，空字符串作为数据
func (t *Transport) generateSignature(timestamp string) (string, error) {
	// 正确的飞书签名算法：timestamp + "\n" + secret作为密钥，空字符串作为数据
	key := timestamp + "\n" + t.secret
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte("")) // 空字符串作为数据
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}

// sendRequest sends the HTTP request to Feishu
func (t *Transport) sendRequest(ctx context.Context, payload *FeishuPayload) (*FeishuResponse, error) {
	// Marshal payload
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", t.webhookURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, errors.MapNetworkError(err, "feishu")
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.MapNetworkError(err, "feishu")
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, errors.MapHTTPError(resp.StatusCode, string(body), "feishu")
	}

	// Parse response
	var feishuResp FeishuResponse
	if err := json.Unmarshal(body, &feishuResp); err != nil {
		return nil, errors.WrapWithPlatform(errors.CodeProcessingFailed, errors.CategoryTransport, "failed to parse Feishu response", "feishu", err)
	}

	return &feishuResp, nil
}

// Shutdown shuts down the transport
func (t *Transport) Shutdown() error {
	// Close HTTP client if needed
	if t.client != nil {
		t.client.CloseIdleConnections()
	}
	return nil
}

// FeishuPayload represents the Feishu webhook payload
type FeishuPayload struct {
	Timestamp string         `json:"timestamp"`
	Sign      string         `json:"sign,omitempty"`
	MsgType   string         `json:"msg_type"`
	Content   *FeishuContent `json:"content,omitempty"`
	Card      interface{}    `json:"card,omitempty"`
}

// FeishuContent represents the content part of Feishu message
type FeishuContent struct {
	Text string      `json:"text,omitempty"`
	Post *FeishuPost `json:"post,omitempty"`
}

// FeishuPost represents the post content for markdown messages
type FeishuPost struct {
	ZhCn *FeishuPostContent `json:"zh_cn,omitempty"`
	EnUs *FeishuPostContent `json:"en_us,omitempty"`
}

// FeishuPostContent represents the post content
type FeishuPostContent struct {
	Title   string                `json:"title"`
	Content [][]FeishuPostElement `json:"content"`
}

// FeishuPostElement represents an element in post content
type FeishuPostElement struct {
	Tag    string `json:"tag"`
	Text   string `json:"text,omitempty"`
	Href   string `json:"href,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

// FeishuResponse represents the Feishu API response
type FeishuResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		MessageID string `json:"message_id,omitempty"`
	} `json:"data,omitempty"`
}
