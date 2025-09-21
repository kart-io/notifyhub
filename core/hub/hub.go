package hub

import (
	"context"
	"fmt"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/internal"
	"github.com/kart-io/notifyhub/logger"
)

// Transport is an alias for core.Transport for backward compatibility
type Transport = core.Transport

// Middleware defines the interface for middleware
type Middleware interface {
	Process(ctx context.Context, msg *core.Message, targets []core.Target, next ProcessFunc) (*core.SendingResults, error)
}

// ProcessFunc is the function signature for processing middleware chain
type ProcessFunc func(ctx context.Context, msg *core.Message, targets []core.Target) (*core.SendingResults, error)

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
func (h *Hub) Send(ctx context.Context, msg *core.Message, targets []core.Target) (*core.SendingResults, error) {
	// Check if hub is shutdown
	if h.IsShutdown() {
		return nil, fmt.Errorf("hub is shutdown")
	}

	// Generate message ID if not set
	if msg.ID == "" {
		msg.ID = h.idGenerator.Generate()
	}

	h.logger.Info(ctx, "sending message", "id", msg.ID, "targets", len(targets))

	// Handle empty targets case
	if len(targets) == 0 {
		// If both targets parameter and message targets are empty, that's an error
		if len(msg.Targets) == 0 {
			return nil, fmt.Errorf("invalid message: at least one target is required")
		}
		// If message has targets but targets parameter is empty, return empty results
		// (This handles cases where someone passes empty slice but message has targets)
		return core.NewSendingResults(), nil
	}

	// Sync targets to message if message doesn't have targets but targets parameter is provided
	if len(msg.Targets) == 0 && len(targets) > 0 {
		msg.Targets = targets
	}

	// Validate message (including targets)
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
		processFunc = func(ctx context.Context, msg *core.Message, targets []core.Target) (*core.SendingResults, error) {
			return middleware.Process(ctx, msg, targets, next)
		}
	}

	return processFunc
}

// coreSend is the core sending function that actually sends messages
func (h *Hub) coreSend(ctx context.Context, msg *core.Message, targets []core.Target) (*core.SendingResults, error) {
	results := core.NewSendingResults()

	// Group targets by platform
	targetGroups := h.groupTargetsByPlatform(targets)

	// Send to each platform
	for platform, platformTargets := range targetGroups {
		transport, exists := h.transports[platform]
		if !exists {
			h.logger.Error(ctx, "transport not found", "platform", platform)
			// Add failed results for all targets in this platform
			for _, target := range platformTargets {
				result := core.NewResult(msg.ID, target)
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
				h.logger.Error(ctx, "failed to send to target: target=%s, error=%v", target.String(), err)
			} else {
				h.logger.Info(ctx, "successfully sent to target: %s", target.String())
			}
		}
	}

	return results, nil
}

// sendToTarget sends a message to a specific target
func (h *Hub) sendToTarget(ctx context.Context, transport Transport, msg *core.Message, target core.Target) (*core.Result, error) {
	result := core.NewResult(msg.ID, target)
	result.SetStatus(core.StatusSending)

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
	result.SetStatus(core.StatusSent)
	return result, nil
}

// groupTargetsByPlatform groups targets by their platform
func (h *Hub) groupTargetsByPlatform(targets []core.Target) map[string][]core.Target {
	groups := make(map[string][]core.Target)
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
