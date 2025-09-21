package core

import (
	"time"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/core/message"
)

// Message represents a message in the queue
type Message struct {
	ID          string           `json:"id"`
	Message     *message.Message `json:"message"`
	Targets     []core.Target    `json:"targets"`
	Options     interface{}      `json:"options"`   // Can be send options
	Callbacks   interface{}      `json:"callbacks"` // Will be *callbacks.CallbackOptions when set
	Attempts    int              `json:"attempts"`
	CreatedAt   time.Time        `json:"created_at"`
	NextRetry   time.Time        `json:"next_retry"`
	LastError   string           `json:"last_error"`
	LastResults []*core.Result   `json:"last_results,omitempty"`
	StreamID    string           `json:"stream_id,omitempty"` // For Redis streams support
}
