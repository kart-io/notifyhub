// Package core provides the core Hub interface and types
package core

import (
	"context"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// Hub represents the main interface for sending notifications
type Hub interface {
	Send(ctx context.Context, message *message.Message) (*receipt.Receipt, error)
	SendAsync(ctx context.Context, message *message.Message) (*receipt.AsyncReceipt, error)
	Health(ctx context.Context) (*HealthStatus, error)
	Close(ctx context.Context) error
}
