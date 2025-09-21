package transport

import (
	"context"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/core/message"
)

// Transport defines the interface for all platform transports
type Transport interface {
	Send(ctx context.Context, msg *message.Message, target core.Target) (*core.Result, error)
	Name() string
	Shutdown() error
}
