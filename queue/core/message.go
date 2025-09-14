package core

import (
	"time"

	"github.com/kart-io/notifyhub/notifiers"
)

// Message represents a message in the queue
type Message struct {
	ID          string                  `json:"id"`
	Message     *notifiers.Message      `json:"message"`
	Options     interface{}             `json:"options"` // Can be client.Options or other options
	Callbacks   interface{}             `json:"callbacks"` // Will be *callbacks.CallbackOptions when set
	Attempts    int                     `json:"attempts"`
	CreatedAt   time.Time               `json:"created_at"`
	NextRetry   time.Time               `json:"next_retry"`
	LastError   string                  `json:"last_error"`
	LastResults []*notifiers.SendResult `json:"last_results,omitempty"`
	StreamID    string                  `json:"stream_id,omitempty"` // For Redis streams support
}