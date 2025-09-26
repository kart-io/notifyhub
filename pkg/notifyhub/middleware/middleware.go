// Package middleware provides middleware support for NotifyHub operations
package middleware

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Middleware defines the interface for NotifyHub middleware
type Middleware interface {
	// Name returns the middleware name for identification
	Name() string

	// HandleSend is called before/after sending a message
	HandleSend(ctx context.Context, msg *message.Message, targets []target.Target, next SendHandler) (*receipt.Receipt, error)
}

// SendHandler represents the next handler in the middleware chain
type SendHandler func(ctx context.Context, msg *message.Message, targets []target.Target) (*receipt.Receipt, error)

// Chain represents a middleware chain
type Chain struct {
	middlewares []Middleware
}

// NewChain creates a new middleware chain
func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{
		middlewares: middlewares,
	}
}

// Then executes the middleware chain with the final handler
func (c *Chain) Then(final SendHandler) SendHandler {
	if len(c.middlewares) == 0 {
		return final
	}

	return c.buildChain(0, final)
}

// buildChain recursively builds the middleware chain
func (c *Chain) buildChain(index int, final SendHandler) SendHandler {
	if index >= len(c.middlewares) {
		return final
	}

	middleware := c.middlewares[index]
	next := c.buildChain(index+1, final)

	return func(ctx context.Context, msg *message.Message, targets []target.Target) (*receipt.Receipt, error) {
		return middleware.HandleSend(ctx, msg, targets, next)
	}
}

// Add adds middleware to the chain
func (c *Chain) Add(middleware ...Middleware) *Chain {
	c.middlewares = append(c.middlewares, middleware...)
	return c
}

// GetMiddlewares returns all middlewares in the chain
func (c *Chain) GetMiddlewares() []Middleware {
	return c.middlewares[:]
}

// Manager manages middleware chains for different operations
type Manager struct {
	sendChain     *Chain
	defaultChains map[string]*Chain
}

// NewManager creates a new middleware manager
func NewManager() *Manager {
	return &Manager{
		sendChain:     NewChain(),
		defaultChains: make(map[string]*Chain),
	}
}

// SetSendMiddleware sets the middleware chain for send operations
func (m *Manager) SetSendMiddleware(middlewares ...Middleware) {
	m.sendChain = NewChain(middlewares...)
}

// AddSendMiddleware adds middleware to the send chain
func (m *Manager) AddSendMiddleware(middleware ...Middleware) {
	m.sendChain.Add(middleware...)
}

// GetSendChain returns the send middleware chain
func (m *Manager) GetSendChain() *Chain {
	return m.sendChain
}

// ExecuteSend executes the send middleware chain with the given handler
func (m *Manager) ExecuteSend(ctx context.Context, msg *message.Message, targets []target.Target, handler SendHandler) (*receipt.Receipt, error) {
	return m.sendChain.Then(handler)(ctx, msg, targets)
}

// Context keys for middleware
type contextKey string

const (
	// MiddlewareStartTimeKey stores the start time for operations
	MiddlewareStartTimeKey contextKey = "middleware_start_time"

	// MiddlewareMetadataKey stores middleware-specific metadata
	MiddlewareMetadataKey contextKey = "middleware_metadata"

	// MiddlewareTraceIDKey stores trace ID for distributed tracing
	MiddlewareTraceIDKey contextKey = "middleware_trace_id"
)

// Utility functions for middleware context management

// SetStartTime sets the start time in context
func SetStartTime(ctx context.Context) context.Context {
	return context.WithValue(ctx, MiddlewareStartTimeKey, time.Now())
}

// GetStartTime gets the start time from context
func GetStartTime(ctx context.Context) (time.Time, bool) {
	if t, ok := ctx.Value(MiddlewareStartTimeKey).(time.Time); ok {
		return t, true
	}
	return time.Time{}, false
}

// SetMetadata sets middleware metadata in context
func SetMetadata(ctx context.Context, key string, value interface{}) context.Context {
	metadata := make(map[string]interface{})
	if existing, ok := ctx.Value(MiddlewareMetadataKey).(map[string]interface{}); ok {
		// Copy existing metadata
		for k, v := range existing {
			metadata[k] = v
		}
	}
	metadata[key] = value
	return context.WithValue(ctx, MiddlewareMetadataKey, metadata)
}

// GetMetadata gets middleware metadata from context
func GetMetadata(ctx context.Context, key string) (interface{}, bool) {
	if metadata, ok := ctx.Value(MiddlewareMetadataKey).(map[string]interface{}); ok {
		if value, exists := metadata[key]; exists {
			return value, true
		}
	}
	return nil, false
}

// SetTraceID sets the trace ID in context
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, MiddlewareTraceIDKey, traceID)
}

// GetTraceID gets the trace ID from context
func GetTraceID(ctx context.Context) (string, bool) {
	if traceID, ok := ctx.Value(MiddlewareTraceIDKey).(string); ok {
		return traceID, true
	}
	return "", false
}

// Base middleware struct that can be embedded
type BaseMiddleware struct {
	name string
}

// NewBaseMiddleware creates a new base middleware
func NewBaseMiddleware(name string) BaseMiddleware {
	return BaseMiddleware{name: name}
}

// Name returns the middleware name
func (bm BaseMiddleware) Name() string {
	return bm.name
}

// NoOpMiddleware is a middleware that does nothing (useful for testing)
type NoOpMiddleware struct {
	BaseMiddleware
}

// NewNoOpMiddleware creates a no-op middleware
func NewNoOpMiddleware() *NoOpMiddleware {
	return &NoOpMiddleware{
		BaseMiddleware: NewBaseMiddleware("noop"),
	}
}

// HandleSend implements the Middleware interface
func (nm *NoOpMiddleware) HandleSend(ctx context.Context, msg *message.Message, targets []target.Target, next SendHandler) (*receipt.Receipt, error) {
	return next(ctx, msg, targets)
}
