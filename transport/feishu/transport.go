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

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
)

// Transport implements the Transport interface for Feishu
type Transport struct {
	webhookURL string
	secret     string
	timeout    time.Duration
	client     *http.Client
}

// Config holds the configuration for Feishu transport
type Config struct {
	WebhookURL string
	Secret     string
	Timeout    time.Duration
}

// NewTransport creates a new Feishu transport
func NewTransport(config *Config) *Transport {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Transport{
		webhookURL: config.WebhookURL,
		secret:     config.Secret,
		timeout:    config.Timeout,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Name returns the transport name
func (t *Transport) Name() string {
	return "feishu"
}

// Send sends a message through Feishu
func (t *Transport) Send(ctx context.Context, msg *message.Message, target sending.Target) (*sending.Result, error) {
	result := sending.NewResult(msg.ID, target)
	result.SetStatus(sending.StatusSending)

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
		err = fmt.Errorf("feishu API error: code=%d, msg=%s", response.Code, response.Msg)
		result.SetError(err)
		return result, err
	}

	result.SetStatus(sending.StatusSent)
	result.SetResponse(response)
	return result, nil
}

// buildPayload builds the Feishu webhook payload
func (t *Transport) buildPayload(msg *message.Message, target sending.Target) (*FeishuPayload, error) {
	payload := &FeishuPayload{
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10),
		MsgType:   "text", // Default to text
	}

	// Add signature if secret is configured
	if t.secret != "" {
		signature, err := t.generateSignature(payload.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to generate signature: %w", err)
		}
		payload.Sign = signature
	}

	// Build content based on message format
	switch msg.Format {
	case message.FormatText:
		payload.Content = &FeishuContent{
			Text: t.buildTextContent(msg),
		}
	case message.FormatMarkdown:
		payload.MsgType = "post"
		payload.Content = &FeishuContent{
			Post: t.buildPostContent(msg),
		}
	case message.FormatCard:
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
func (t *Transport) buildTextContent(msg *message.Message) string {
	content := ""
	if msg.Title != "" {
		content += msg.Title + "\n"
	}
	content += msg.Body
	return content
}

// buildPostContent builds post (markdown) content
func (t *Transport) buildPostContent(msg *message.Message) *FeishuPost {
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
func (t *Transport) buildDefaultCard(msg *message.Message) interface{} {
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
func (t *Transport) generateSignature(timestamp string) (string, error) {
	stringToSign := timestamp + "\n" + t.secret
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte(stringToSign))
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
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

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
