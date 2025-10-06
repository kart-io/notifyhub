// Package notifyhub provides the unified client interface and entry point for NotifyHub
// This implements the 3-layer architecture: Client → Dispatcher → Platform
package notifyhub

import (
	"context"

	"github.com/kart/notifyhub/pkg/async"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/receipt"
)

// Client represents the unified notification client interface
// This interface implements the 3-layer architecture: Client → Dispatcher → Platform
// replacing the complex 6-layer calling chain from the previous implementation
type Client interface {
	// Synchronous interface - immediate message sending
	Send(ctx context.Context, msg *message.Message) (*receipt.Receipt, error)
	SendBatch(ctx context.Context, msgs []*message.Message) ([]*receipt.Receipt, error)

	// Asynchronous interface - true async processing with real queue support
	SendAsync(ctx context.Context, msg *message.Message, opts ...async.Option) (async.Handle, error)
	SendAsyncBatch(ctx context.Context, msgs []*message.Message, opts ...async.Option) (async.BatchHandle, error)

	// Management interface - health monitoring and lifecycle management
	Health(ctx context.Context) (*HealthStatus, error)
	Close() error
}

// HealthStatus represents the comprehensive health status of the NotifyHub client
type HealthStatus struct {
	Status      string                 `json:"status"`       // "healthy", "degraded", "unhealthy"
	Platforms   map[string]string      `json:"platforms"`    // Platform name -> health status
	Uptime      float64                `json:"uptime"`       // Uptime in seconds
	ActiveTasks int64                  `json:"active_tasks"` // Number of active async tasks
	QueueDepth  int64                  `json:"queue_depth"`  // Current queue depth
	TotalSent   int64                  `json:"total_sent"`   // Total messages sent
	SuccessRate float64                `json:"success_rate"` // Success rate percentage
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
