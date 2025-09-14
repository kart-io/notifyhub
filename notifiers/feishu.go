package notifiers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/internal"
)

// ================================
// Feishu平台适配器实现
// ================================

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
		client:      &http.Client{Timeout: timeout},
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
		client:      &http.Client{Timeout: timeout},
		rateLimiter: internal.NewRateLimiter(rateLimitConfig),
	}
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

	// Build payload based on message format
	payload := f.buildPayload(message, target)

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

	result.Duration = time.Since(start)
	result.Success = resp.StatusCode == http.StatusOK
	if !result.Success {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

// buildPayload builds Feishu message payload based on format
func (f *FeishuNotifier) buildPayload(message *Message, target Target) map[string]interface{} {
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