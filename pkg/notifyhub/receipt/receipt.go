// Package receipt provides receipt management for NotifyHub
package receipt

import (
	"time"
)

// Receipt represents the result of sending a message
type Receipt struct {
	MessageID  string           `json:"message_id"`
	Status     string           `json:"status"`
	Results    []PlatformResult `json:"results"`
	Successful int              `json:"successful"`
	Failed     int              `json:"failed"`
	Total      int              `json:"total"`
	Timestamp  time.Time        `json:"timestamp"`
	Error      error            `json:"error,omitempty"`
}

// PlatformResult represents the result from a specific platform
type PlatformResult struct {
	Platform  string        `json:"platform"`
	Target    string        `json:"target"`
	Success   bool          `json:"success"`
	MessageID string        `json:"message_id,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

// AsyncReceipt represents an asynchronous message receipt
type AsyncReceipt struct {
	MessageID string    `json:"message_id"`
	Status    string    `json:"status"`
	QueuedAt  time.Time `json:"queued_at"`
}
