package hub

import (
	"context"
	"fmt"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/internal"
	"github.com/kart-io/notifyhub/logger"
)

// Transport defines the interface for platform transports
type Transport interface {
	Send(ctx context.Context, msg *message.Message, target sending.Target) (*sending.Result, error)
	Name() string
	Shutdown() error
}

// Middleware defines the interface for middleware
type Middleware interface {
	Process(ctx context.Context, msg *message.Message, targets []sending.Target, next ProcessFunc) (*sending.SendingResults, error)
}

// ProcessFunc is the function signature for processing middleware chain
type ProcessFunc func(ctx context.Context, msg *message.Message, targets []sending.Target) (*sending.SendingResults, error)

// Hub is the central orchestrator for message sending
type Hub struct {
	transports   map[string]Transport
	middlewares  []Middleware
	logger       logger.Interface
	idGenerator  IDGenerator
	shutdownChan chan struct{}
}

// HealthStatus represents the health status of the hub
type HealthStatus struct {
	Healthy bool                   `json:"healthy"`
	Details map[string]interface{} `json:"details"`
}

// Logger interface has been replaced by logger.Interface
// This provides better context support and unified logging across the project

// IDGenerator interface for generating unique IDs
type IDGenerator interface {
	Generate() string
}

// internalIDGenerator is the default ID generator implementation
type internalIDGenerator struct{}

// Options for hub configuration
type Options struct {
	Logger      logger.Interface
	IDGenerator IDGenerator
}

func (g *internalIDGenerator) Generate() string {
	return internal.GenerateID()
}

// NewHub creates a new notification hub
func NewHub(opts *Options) *Hub {
	if opts == nil {
		opts = &Options{}
	}

	hub := &Hub{
		transports:   make(map[string]Transport),
		middlewares:  make([]Middleware, 0),
		logger:       opts.Logger,
		idGenerator:  opts.IDGenerator,
		shutdownChan: make(chan struct{}),
	}

	if hub.logger == nil {
		hub.logger = logger.Discard
	}

	if hub.idGenerator == nil {
		hub.idGenerator = &internalIDGenerator{}
	}

	return hub
}

// RegisterTransport registers a platform transport
func (h *Hub) RegisterTransport(transport Transport) {
	h.transports[transport.Name()] = transport
	h.logger.Info(context.Background(), "registered transport", "name", transport.Name())
}

// AddMiddleware adds middleware to the processing chain
func (h *Hub) AddMiddleware(middleware Middleware) {
	h.middlewares = append(h.middlewares, middleware)
	h.logger.Info(context.Background(), "added middleware")
}

// Send sends a message to the specified targets
func (h *Hub) Send(ctx context.Context, msg *message.Message, targets []sending.Target) (*sending.SendingResults, error) {
	// Generate message ID if not set
	if msg.ID == "" {
		msg.ID = h.idGenerator.Generate()
	}

	h.logger.Info(ctx, "sending message", "id", msg.ID, "targets", len(targets))

	// Validate message
	if err := msg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid message: %w", err)
	}

	// Validate targets
	for _, target := range targets {
		if err := target.Validate(); err != nil {
			return nil, fmt.Errorf("invalid target %s: %w", target.String(), err)
		}
	}

	// Process through middleware chain
	processFunc := h.buildProcessChain()
	return processFunc(ctx, msg, targets)
}

// buildProcessChain builds the middleware processing chain
func (h *Hub) buildProcessChain() ProcessFunc {
	// Start with the core sending function
	processFunc := h.coreSend

	// Wrap with middleware in reverse order
	for i := len(h.middlewares) - 1; i >= 0; i-- {
		middleware := h.middlewares[i]
		next := processFunc
		processFunc = func(ctx context.Context, msg *message.Message, targets []sending.Target) (*sending.SendingResults, error) {
			return middleware.Process(ctx, msg, targets, next)
		}
	}

	return processFunc
}

// coreSend is the core sending function that actually sends messages
func (h *Hub) coreSend(ctx context.Context, msg *message.Message, targets []sending.Target) (*sending.SendingResults, error) {
	results := sending.NewSendingResults()

	// Group targets by platform
	targetGroups := h.groupTargetsByPlatform(targets)

	// Send to each platform
	for platform, platformTargets := range targetGroups {
		transport, exists := h.transports[platform]
		if !exists {
			h.logger.Error(ctx, "transport not found", "platform", platform)
			// Add failed results for all targets in this platform
			for _, target := range platformTargets {
				result := sending.NewResult(msg.ID, target)
				result.SetError(fmt.Errorf("transport not found for platform: %s", platform))
				results.AddResult(result)
			}
			continue
		}

		// Send to each target in this platform
		for _, target := range platformTargets {
			result, err := h.sendToTarget(ctx, transport, msg, target)
			results.AddResult(result)
			if err != nil {
				h.logger.Error(ctx, "failed to send to target", "target", target.String(), "error", err)
			} else {
				h.logger.Info(ctx, "successfully sent to target", "target", target.String())
			}
		}
	}

	return results, nil
}

// sendToTarget sends a message to a specific target
func (h *Hub) sendToTarget(ctx context.Context, transport Transport, msg *message.Message, target sending.Target) (*sending.Result, error) {
	result := sending.NewResult(msg.ID, target)
	result.SetStatus(sending.StatusSending)

	// Send through transport
	transportResult, err := transport.Send(ctx, msg, target)
	if err != nil {
		result.SetError(err)
		return result, err
	}

	// Use the result from transport if provided
	if transportResult != nil {
		return transportResult, nil
	}

	// Otherwise mark as sent
	result.SetStatus(sending.StatusSent)
	return result, nil
}

// groupTargetsByPlatform groups targets by their platform
func (h *Hub) groupTargetsByPlatform(targets []sending.Target) map[string][]sending.Target {
	groups := make(map[string][]sending.Target)
	for _, target := range targets {
		platform := target.Platform
		groups[platform] = append(groups[platform], target)
	}
	return groups
}

// GetTransport returns a registered transport by name
func (h *Hub) GetTransport(name string) (Transport, bool) {
	transport, exists := h.transports[name]
	return transport, exists
}

// ListTransports returns all registered transport names
func (h *Hub) ListTransports() []string {
	names := make([]string, 0, len(h.transports))
	for name := range h.transports {
		names = append(names, name)
	}
	return names
}

// Shutdown gracefully shuts down the hub and all transports
func (h *Hub) Shutdown(ctx context.Context) error {
	h.logger.Info(context.Background(), "shutting down hub")

	close(h.shutdownChan)

	// Shutdown all transports
	for name, transport := range h.transports {
		if err := transport.Shutdown(); err != nil {
			h.logger.Error(context.Background(), "failed to shutdown transport", "name", name, "error", err)
		} else {
			h.logger.Info(context.Background(), "transport shutdown complete", "name", name)
		}
	}

	h.logger.Info(context.Background(), "hub shutdown complete")
	return nil
}

// Health returns the health status of the hub
func (h *Hub) Health(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Healthy: true,
		Details: make(map[string]interface{}),
	}

	// Check if hub is shutdown
	if h.IsShutdown() {
		status.Healthy = false
		status.Details["shutdown"] = true
		return status
	}

	// Check transport health
	status.Details["transports"] = len(h.transports)

	return status
}

// IsShutdown returns true if the hub is shut down
func (h *Hub) IsShutdown() bool {
	select {
	case <-h.shutdownChan:
		return true
	default:
		return false
	}
}

// noopLogger has been replaced by logger.Interface
// Using logger.NewUnifiedLogger(logger.Discard) for no-op logging
