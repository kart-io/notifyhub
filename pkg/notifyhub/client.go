// Package notifyhub provides the unified client interface and simplified entry point for NotifyHub
package notifyhub

import (
	"context"

	"github.com/kart-io/notifyhub/pkg/notifyhub/async"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Client represents the unified notification client interface
// This interface implements the 3-layer architecture: Client → Dispatcher → Platform
// replacing the complex 6-layer calling chain from the previous implementation
type Client interface {
	// Synchronous interface - immediate message sending
	Send(ctx context.Context, msg *message.Message) (*receipt.Receipt, error)
	SendBatch(ctx context.Context, msgs []*message.Message) ([]*receipt.Receipt, error)

	// Asynchronous interface - true async processing with real queue support
	SendAsync(ctx context.Context, msg *message.Message, opts ...async.AsyncOption) (async.AsyncHandle, error)
	SendAsyncBatch(ctx context.Context, msgs []*message.Message, opts ...async.AsyncOption) (async.AsyncBatchHandle, error)

	// Management interface - health monitoring and lifecycle management
	Health(ctx context.Context) (*HealthStatus, error)
	Close() error
}

// HealthStatus represents the comprehensive health status of the NotifyHub client
type HealthStatus struct {
	Status      string                 `json:"status"`         // "healthy", "degraded", "unhealthy"
	Platforms   map[string]string      `json:"platforms"`      // Platform name -> health status
	Uptime      int64                  `json:"uptime"`         // Uptime in seconds
	ActiveTasks int                    `json:"active_tasks"`   // Number of active async tasks
	QueueDepth  int                    `json:"queue_depth"`    // Current queue depth
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// clientImpl provides the concrete implementation of the Client interface
// This implementation uses instance-level dependencies (no global state)
type clientImpl struct {
	dispatcher   Dispatcher        // Core message dispatcher (2nd layer)
	asyncManager AsyncManager      // Async processing manager
	registry     PlatformRegistry  // Instance-level platform registry
	config       *Config           // Client configuration
	healthMon    HealthMonitor     // Health monitoring system
}

// Send implements synchronous message sending through the simplified call chain
func (c *clientImpl) Send(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	return c.dispatcher.Dispatch(ctx, msg)
}

// SendBatch implements synchronous batch message sending
func (c *clientImpl) SendBatch(ctx context.Context, msgs []*message.Message) ([]*receipt.Receipt, error) {
	receipts := make([]*receipt.Receipt, len(msgs))
	for i, msg := range msgs {
		receipt, err := c.Send(ctx, msg)
		if err != nil {
			return receipts[:i], err
		}
		receipts[i] = receipt
	}
	return receipts, nil
}

// SendAsync implements true asynchronous message sending with real queue support
func (c *clientImpl) SendAsync(ctx context.Context, msg *message.Message, opts ...async.AsyncOption) (async.AsyncHandle, error) {
	return c.asyncManager.ScheduleAsync(ctx, msg, opts...)
}

// SendAsyncBatch implements asynchronous batch message sending
func (c *clientImpl) SendAsyncBatch(ctx context.Context, msgs []*message.Message, opts ...async.AsyncOption) (async.AsyncBatchHandle, error) {
	return c.asyncManager.ScheduleBatch(ctx, msgs, opts...)
}

// Health returns the current health status of the client and all its components
func (c *clientImpl) Health(ctx context.Context) (*HealthStatus, error) {
	return c.healthMon.GetStatus(ctx)
}

// Close gracefully shuts down the client and all its components
func (c *clientImpl) Close() error {
	// Graceful shutdown sequence
	if c.asyncManager != nil {
		if err := c.asyncManager.Close(); err != nil {
			return err
		}
	}

	if c.dispatcher != nil {
		if err := c.dispatcher.Close(); err != nil {
			return err
		}
	}

	if c.registry != nil {
		if err := c.registry.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Dispatcher represents the core message dispatching interface (2nd layer)
// This is the central coordinator that routes messages to appropriate platforms
type Dispatcher interface {
	Dispatch(ctx context.Context, msg *message.Message) (*receipt.Receipt, error)
	Close() error
}

// PlatformRegistry manages the instance-level platform registration
// This replaces the global platform registry that caused multi-instance conflicts
type PlatformRegistry interface {
	Register(name string, platform Platform) error
	Get(name string) (Platform, bool)
	List() []string
	Close() error
}

// Platform represents the unified platform interface (3rd layer)
// All platform implementations must implement this interface
type Platform interface {
	// Platform identification
	Name() string
	GetCapabilities() Capabilities

	// Message sending (core functionality)
	Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error)
	ValidateTarget(target target.Target) error

	// Lifecycle management
	IsHealthy(ctx context.Context) error
	Close() error
}

// Capabilities describes what a platform supports
type Capabilities struct {
	Name                 string   `json:"name"`
	SupportedTargetTypes []string `json:"supported_target_types"`
	SupportedFormats     []string `json:"supported_formats"`
	MaxMessageSize       int      `json:"max_message_size"`
	SupportsScheduling   bool     `json:"supports_scheduling"`
	SupportsAttachments  bool     `json:"supports_attachments"`
	RequiredSettings     []string `json:"required_settings"`
}

// SendResult represents the result of sending to a specific target
type SendResult struct {
	Target    target.Target `json:"target"`
	Success   bool          `json:"success"`
	MessageID string        `json:"message_id,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// AsyncManager handles asynchronous processing with real queue support
type AsyncManager interface {
	ScheduleAsync(ctx context.Context, msg *message.Message, opts ...async.AsyncOption) (async.AsyncHandle, error)
	ScheduleBatch(ctx context.Context, msgs []*message.Message, opts ...async.AsyncOption) (async.AsyncBatchHandle, error)
	Close() error
}

// HealthMonitor provides comprehensive health monitoring
type HealthMonitor interface {
	GetStatus(ctx context.Context) (*HealthStatus, error)
	Close() error
}