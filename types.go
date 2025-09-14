package notifyhub

import (
	"time"
)

// ================================
// 包特有的类型定义
// ================================

// SendOptions provides options for sending messages
type SendOptions struct {
	Async      bool          `json:"async,omitempty"`
	Retry      bool          `json:"retry,omitempty"`
	MaxRetries int           `json:"max_retries,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
}