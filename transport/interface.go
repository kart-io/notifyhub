package transport

import (
	"context"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
)

// Transport defines the interface for all platform transports
type Transport interface {
	Send(ctx context.Context, msg *message.Message, target sending.Target) (*sending.Result, error)
	Name() string
	Shutdown() error
}
