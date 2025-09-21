package sending

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/logger"
)

// SyncSender handles synchronous message sending
// This implements the proposal's synchronous sending pattern
type SyncSender struct {
	transports map[string]Transport
	logger     Logger
	timeout    time.Duration
}

// Transport interface for platform-specific transports
type Transport interface {
	Send(ctx context.Context, msg *message.Message, target Target) (*Result, error)
	Name() string
	Validate(msg *message.Message) error
}

// Logger is an alias for the unified logger interface
type Logger = logger.Interface

// NewSyncSender creates a new synchronous sender
func NewSyncSender(logger Logger, timeout time.Duration) *SyncSender {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &SyncSender{
		transports: make(map[string]Transport),
		logger:     logger,
		timeout:    timeout,
	}
}

// RegisterTransport registers a transport for a platform
func (s *SyncSender) RegisterTransport(transport Transport) error {
	name := transport.Name()
	if _, exists := s.transports[name]; exists {
		return fmt.Errorf("transport %s already registered", name)
	}
	s.transports[name] = transport
	return nil
}

// Send sends a message synchronously to all targets
func (s *SyncSender) Send(ctx context.Context, msg *message.Message, targets []Target) (*SendingResults, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets provided")
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	results := NewSendingResults()
	startTime := time.Now()

	// Send to each target
	for _, target := range targets {
		result := s.sendToTarget(ctx, msg, target)
		results.AddResult(result)
	}

	results.Duration = time.Since(startTime)
	return results, nil
}

// sendToTarget sends a message to a single target
func (s *SyncSender) sendToTarget(ctx context.Context, msg *message.Message, target Target) *Result {
	result := NewResult(msg.ID, target)
	result.StartTime = time.Now()

	// Get transport for target platform
	transport, exists := s.transports[target.GetPlatform()]
	if !exists {
		result.Error = fmt.Errorf("no transport for platform: %s", target.GetPlatform())
		result.Status = StatusFailed
		result.EndTime = time.Now()
		s.logger.Error(ctx, "Transport not found", "platform", target.GetPlatform(), "target", target.GetValue(), "error", result.Error)
		return result
	}

	// Validate message for this transport
	if err := transport.Validate(msg); err != nil {
		result.Error = fmt.Errorf("message validation failed: %w", err)
		result.Status = StatusFailed
		result.EndTime = time.Now()
		s.logger.Error(ctx, "Message validation failed", "platform", target.Platform, "message", msg.ID, "error", err)
		return result
	}

	// Send through transport
	s.logger.Debug(ctx, "Sending message", "platform", target.Platform, "target", target.Value, "message", msg.ID)

	transportResult, err := transport.Send(ctx, msg, target)
	if err != nil {
		result.Error = err
		result.Status = StatusFailed
		s.logger.Error(ctx, "Send failed", "platform", target.Platform, "target", target.Value, "message", msg.ID, "error", err)
	} else {
		// Use transport result if provided
		if transportResult != nil {
			result = transportResult
		} else {
			result.Status = StatusSent
			result.Success = true
		}
		s.logger.Info(ctx, "Message sent successfully", "platform", target.Platform, "target", target.Value, "message", msg.ID)
	}

	result.EndTime = time.Now()
	return result
}

// GetTransports returns all registered transports
func (s *SyncSender) GetTransports() []string {
	names := make([]string, 0, len(s.transports))
	for name := range s.transports {
		names = append(names, name)
	}
	return names
}
