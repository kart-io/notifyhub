// Package client provides the unified client interface for NotifyHub
package client

import (
	"context"

	"github.com/kart-io/notifyhub/pkg/notifyhub/async"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// Client represents the unified notification client interface
// This interface replaces the previous complex calling chain and provides
// both synchronous and asynchronous messaging capabilities
type Client interface {
	// Synchronous interface
	Send(ctx context.Context, msg *message.Message) (*receipt.Receipt, error)
	SendBatch(ctx context.Context, msgs []*message.Message) ([]*receipt.Receipt, error)

	// Asynchronous interface (真正异步 + 回调)
	SendAsync(ctx context.Context, msg *message.Message, opts ...async.AsyncOption) (async.AsyncHandle, error)
	SendAsyncBatch(ctx context.Context, msgs []*message.Message, opts ...async.AsyncOption) (async.AsyncBatchHandle, error)

	// Management interface
	Health(ctx context.Context) (*HealthStatus, error)
	Close() error
}

// HealthStatus represents the health status of the client
type HealthStatus struct {
	Status    string                 `json:"status"`
	Platforms map[string]string      `json:"platforms"`
	Uptime    int64                  `json:"uptime"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
