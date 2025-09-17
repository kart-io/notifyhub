package notifiers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/kart-io/notifyhub/internal"
)

// ================================
// Feishu平台适配器实现
// ================================

// createOptimizedHTTPClient creates an HTTP client optimized for webhook calls
func createOptimizedHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,  // 连接超时
				KeepAlive: 30 * time.Second, // 保持连接
			}).DialContext,
			TLSHandshakeTimeout:   3 * time.Second, // TLS握手超时
			ResponseHeaderTimeout: 5 * time.Second, // 响应头超时
			ExpectContinueTimeout: 1 * time.Second, // Expect 100-continue超时
			MaxIdleConns:          10,              // 最大空闲连接
			MaxIdleConnsPerHost:   2,               // 每个host最大空闲连接
			IdleConnTimeout:       30 * time.Second, // 空闲连接超时
			DisableCompression:    true,            // 禁用压缩以减少CPU使用
		},
	}
}

// FeishuNotifier implements Feishu webhook notifications
type FeishuNotifier struct {
	webhookURL  string
	secret      string
	timeout     time.Duration
	client      *http.Client
	rateLimiter internal.RateLimiter
}

// NewFeishuNotifier creates a new Feishu notifier
func NewFeishuNotifier(webhookURL, secret string, timeout time.Duration) *FeishuNotifier {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &FeishuNotifier{
		webhookURL:  webhookURL,
		secret:      secret,
		timeout:     timeout,
		client:      createOptimizedHTTPClient(timeout),
		rateLimiter: &internal.NoOpRateLimiter{}, // Default: no rate limiting
	}
}

// NewFeishuNotifierWithRateLimit creates a new Feishu notifier with rate limiting
func NewFeishuNotifierWithRateLimit(webhookURL, secret string, timeout time.Duration, rateLimitConfig *internal.RateLimitConfig) *FeishuNotifier {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &FeishuNotifier{
		webhookURL:  webhookURL,
		secret:      secret,
		timeout:     timeout,
		client:      createOptimizedHTTPClient(timeout),
		rateLimiter: internal.NewRateLimiter(rateLimitConfig),
	}
}

// NewFeishuNotifierSimple creates a new Feishu notifier without secret (不启用签名验证)
// 适用于开发测试环境或不需要签名验证的场景
func NewFeishuNotifierSimple(webhookURL string) *FeishuNotifier {
	return NewFeishuNotifier(webhookURL, "", 30*time.Second)
}

// NewFeishuNotifierWithTimeout creates a new Feishu notifier without secret but with custom timeout
// 适用于需要自定义超时时间但不需要签名验证的场景
func NewFeishuNotifierWithTimeout(webhookURL string, timeout time.Duration) *FeishuNotifier {
	return NewFeishuNotifier(webhookURL, "", timeout)
}

// Name returns the notifier name
func (f *FeishuNotifier) Name() string {
	return "feishu"
}

// SupportsTarget checks if this notifier supports the given target
func (f *FeishuNotifier) SupportsTarget(target Target) bool {
	if target.Platform != "" && target.Platform != "feishu" {
		return false
	}
	return target.Type == TargetTypeGroup || target.Type == TargetTypeUser
}

// Send sends a message via Feishu webhook
func (f *FeishuNotifier) Send(ctx context.Context, message *Message) ([]*SendResult, error) {
	var results []*SendResult

	// If no targets, send to default webhook
	if len(message.Targets) == 0 {
		result := f.sendToWebhook(ctx, message, Target{Type: TargetTypeGroup, Value: "default"})
		results = append(results, result)
		return results, nil
	}

	// Send to each supported target
	for _, target := range message.Targets {
		if f.SupportsTarget(target) {
			result := f.sendToWebhook(ctx, message, target)
			results = append(results, result)
		}
	}

	return results, nil
}

// sendToWebhook sends message to Feishu webhook
func (f *FeishuNotifier) sendToWebhook(ctx context.Context, message *Message, target Target) *SendResult {
	start := time.Now()
	result := &SendResult{
		Target:   target,
		Platform: f.Name(),
		SentAt:   start,
		Attempts: 1,
	}

	// Apply rate limiting
	if err := f.rateLimiter.ConsumeWithTimeout(ctx, f.timeout); err != nil {
		result.Error = fmt.Sprintf("rate limit exceeded: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// 准备签名信息（如果配置了secret）
	var timestamp, signature string
	if f.secret != "" {
		timestamp = strconv.FormatInt(time.Now().Unix(), 10)
		signature = f.generateSignature(timestamp)
		// 调试信息：fmt.Printf("飞书签名调试: timestamp=%s, secret=%s, signature=%s\n", timestamp, f.secret, signature)
	}

	// Build payload based on message format (包含签名信息)
	payload := f.buildPayload(message, target, timestamp, signature)

	data, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Sprintf("marshal payload: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	req, err := http.NewRequestWithContext(ctx, "POST", f.webhookURL, bytes.NewBuffer(data))
	if err != nil {
		result.Error = fmt.Sprintf("create request: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("send request: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	// 读取响应体以获取详细的错误信息
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("read response body: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// 调试信息：fmt.Printf("飞书API响应: Status=%d, Body=%s\n", resp.StatusCode, string(body))

	result.Duration = time.Since(start)
	result.Success = resp.StatusCode == http.StatusOK
	if !result.Success {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	} else {
		// 即使是200状态码，也要检查飞书API返回的具体错误码
		var responseData map[string]interface{}
		if err := json.Unmarshal(body, &responseData); err == nil {
			if code, ok := responseData["code"].(float64); ok && code != 0 {
				result.Success = false
				if msg, exists := responseData["msg"].(string); exists {
					result.Error = fmt.Sprintf("飞书API错误: code=%v, msg=%s", code, msg)
				} else {
					result.Error = fmt.Sprintf("飞书API错误: code=%v", code)
				}
			}
		}
	}

	return result
}

// buildPayload builds Feishu message payload based on format
func (f *FeishuNotifier) buildPayload(message *Message, target Target, timestamp, signature string) map[string]interface{} {
	var payload map[string]interface{}

	switch message.Format {
	case FormatText:
		payload = map[string]interface{}{
			"msg_type": "text",
			"content": map[string]string{
				"text": f.formatTextContent(message),
			},
		}
	case FormatMarkdown:
		payload = map[string]interface{}{
			"msg_type": "post",
			"content": map[string]interface{}{
				"post": map[string]interface{}{
					"zh_cn": map[string]interface{}{
						"title": message.Title,
						"content": [][]map[string]interface{}{
							{{"tag": "text", "text": message.Body}},
						},
					},
				},
			},
		}
	case FormatCard:
		payload = f.buildCardPayload(message)
	default:
		payload = map[string]interface{}{
			"msg_type": "text",
			"content": map[string]string{
				"text": f.formatTextContent(message),
			},
		}
	}

	// Add user mention if needed
	f.addUserMention(payload, target)

	// Add signature verification fields if configured
	if f.secret != "" && timestamp != "" && signature != "" {
		payload["timestamp"] = timestamp
		payload["sign"] = signature
	}

	return payload
}

// formatTextContent formats message as plain text
func (f *FeishuNotifier) formatTextContent(message *Message) string {
	content := ""
	if message.Title != "" {
		content = fmt.Sprintf("%s\n\n%s", message.Title, message.Body)
	} else {
		content = message.Body
	}

	// Add metadata if present
	if len(message.Metadata) > 0 {
		content += "\n\n---\n"
		for key, value := range message.Metadata {
			content += fmt.Sprintf("%s: %s\n", key, value)
		}
	}

	return content
}

// addUserMention adds user mention to the message if target is user
func (f *FeishuNotifier) addUserMention(payload map[string]interface{}, target Target) {
	if target.Type == TargetTypeUser && target.Value != "default" {
		if content, ok := payload["content"].(map[string]string); ok {
			content["text"] = fmt.Sprintf("<at user_id=\"%s\">@user</at> %s", target.Value, content["text"])
		}
	}
}

// Health checks if Feishu notifier is healthy
func (f *FeishuNotifier) Health(ctx context.Context) error {
	if f.webhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}
	// 注意：secret 是可选的，如果配置了则启用签名验证
	// 这提供了额外的安全性，但不是必需的
	return nil
}

// Shutdown gracefully shuts down the Feishu notifier
func (f *FeishuNotifier) Shutdown(ctx context.Context) error {
	// Close HTTP client connections
	if f.client != nil {
		f.client.CloseIdleConnections()
	}
	return nil
}

// buildCardPayload builds Feishu interactive card payload
func (f *FeishuNotifier) buildCardPayload(message *Message) map[string]interface{} {
	payload := map[string]interface{}{
		"msg_type": "interactive",
	}

	// 如果有自定义的CardData，直接使用
	if message.CardData != nil {
		if cardData, ok := message.CardData.(map[string]interface{}); ok {
			payload["card"] = cardData
			return payload
		}
	}

	// 否则构建默认的卡片结构
	payload["card"] = f.buildDefaultCard(message)
	return payload
}

// buildDefaultCard builds a default card structure for Feishu
func (f *FeishuNotifier) buildDefaultCard(message *Message) map[string]interface{} {
	elements := []map[string]interface{}{}

	// 添加标题元素
	if message.Title != "" {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": message.Title,
				"tag":     "lark_md",
			},
		})
	}

	// 添加内容元素
	if message.Body != "" {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": message.Body,
				"tag":     "lark_md",
			},
		})
	}

	// 添加元数据信息
	if len(message.Metadata) > 0 {
		elements = append(elements, map[string]interface{}{
			"tag": "hr",
		})

		for key, value := range message.Metadata {
			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": fmt.Sprintf("**%s**: %s", key, value),
					"tag":     "lark_md",
				},
			})
		}
	}

	return map[string]interface{}{
		"elements": elements,
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": message.Title,
				"tag":     "plain_text",
			},
		},
	}
}

// generateSignature 生成飞书 webhook 签名
// 根据飞书官方文档方式三：设置签名校验
// 参考: https://blog.csdn.net/jinborui2/article/details/133940051
// 飞书特殊实现：key = timestamp + "\n" + secret, data = 空字符串
func (f *FeishuNotifier) generateSignature(timestamp string) string {
	if f.secret == "" {
		return ""
	}

	// 构造HMAC密钥：timestamp + "\n" + secret
	key := timestamp + "\n" + f.secret

	// 使用飞书的特殊HMAC实现（对应PHP的hash_hmac('sha256', '', $sign, true)）：
	// key = timestamp + "\n" + secret
	// data = 空字符串（必须明确写入）
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte("")) // 明确写入空字符串

	// Base64 编码
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
