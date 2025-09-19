package feishu

import "errors"

// Feishu-specific errors
var (
	ErrInvalidConfig = errors.New("invalid Feishu config")
	ErrEmptyMessage  = errors.New("message cannot be empty")
	ErrInvalidUser   = errors.New("invalid user ID")
	ErrInvalidGroup  = errors.New("invalid group ID")
)

// FeishuConfig holds Feishu-specific configuration
type FeishuConfig struct {
	WebhookURL string
	Secret     string
	AppID      string
	AppSecret  string
}

// FeishuTarget represents a Feishu-specific target
type FeishuTarget struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	ChatID string `json:"chat_id,omitempty"`
}

// FeishuCard represents a Feishu interactive card
type FeishuCard struct {
	Config   FeishuCardConfig    `json:"config"`
	Elements []FeishuCardElement `json:"elements"`
}

// FeishuCardConfig represents card configuration
type FeishuCardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
	EnableForward  bool `json:"enable_forward"`
}

// FeishuCardElement represents a card element
type FeishuCardElement struct {
	Tag    string      `json:"tag"`
	Text   interface{} `json:"text,omitempty"`
	Fields interface{} `json:"fields,omitempty"`
}

// FeishuMessage represents a complete Feishu message
type FeishuMessage struct {
	MsgType   string      `json:"msg_type"`
	Content   interface{} `json:"content"`
	Card      *FeishuCard `json:"card,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
	Sign      string      `json:"sign,omitempty"`
}
