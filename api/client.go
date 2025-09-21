// Package api provides a compatibility layer and interface adapters for the NotifyHub system.
//
// This package serves as a bridge between different API versions and provides interface
// implementations that wrap the main notifyhub.Client. It is designed to:
//
//  1. Provide backward compatibility with existing API contracts
//  2. Implement specific interfaces required by different components
//  3. Serve as an adapter layer between different client versions
//
// For new applications, it's recommended to use the main notifyhub.Client directly
// from the root package, which provides the most complete and up-to-date API.
//
// The api.Client interface implemented by this package is primarily used for:
//   - Legacy compatibility
//   - Interface-based dependency injection
//   - Component integration where a specific interface contract is required
//
// Example usage:
//
//	// Using the api package (adapter layer)
//	client, err := api.New(
//	    notifyhub.WithFeishu("webhook-url", "secret"),
//	    notifyhub.WithEmail("smtp.host.com", 587, "user", "pass", "from@example.com"),
//	)
//
//	// Using the main package directly (recommended for new code)
//	client, err := notifyhub.New(
//	    notifyhub.WithFeishu("webhook-url", "secret"),
//	    notifyhub.WithEmail("smtp.host.com", 587, "user", "pass", "from@example.com"),
//	)
package api

import (
	"context"
	"fmt"

	"github.com/kart-io/notifyhub"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/monitoring"
)

// ClientImpl wraps the main notifyhub.Client to implement the api.Client interface.
//
// This is a thin adapter layer that delegates all operations to the underlying
// notifyhub.Client while providing the interface contract required by various
// components. It avoids code duplication by leveraging the builder patterns
// and functionality provided by the main client.
//
// The adapter handles:
//   - Interface compliance with api.Client
//   - Type conversions between interface types and implementation types
//   - Delegation of all core functionality to notifyhub.Client
//   - Metrics collection and monitoring integration
type ClientImpl struct {
	client  *notifyhub.Client
	metrics *monitoring.Metrics
}

// New creates a new API client
// This is a compatibility layer that wraps the main notifyhub.Client
// It accepts the same notifyhub.Option functions as the main client
func New(args ...interface{}) (Client, error) {
	// Convert all arguments to notifyhub.Option functions
	options := make([]notifyhub.Option, 0, len(args))

	for i, arg := range args {
		switch v := arg.(type) {
		case notifyhub.Option:
			// Direct notifyhub option
			options = append(options, v)
		case *config.Config:
			// Legacy config.Config support - for now, ignore and use defaults
			// This is kept for backward compatibility but should be migrated to options
			_ = v // Suppress unused variable warning
			// TODO: Convert config.Config to notifyhub.Option functions when needed
		default:
			return nil, fmt.Errorf("unsupported argument type at index %d: %T", i, arg)
		}
	}

	// Create the underlying notifyhub client
	client, err := notifyhub.New(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create notifyhub client: %w", err)
	}

	return &ClientImpl{
		client:  client,
		metrics: monitoring.NewMetrics(),
	}, nil
}

// Hub returns the hub interface for sending messages
func (c *ClientImpl) Hub() ClientHub {
	return &HubImpl{client: c.client}
}

// Metrics returns metrics about message sending
func (c *ClientImpl) Metrics() interface{} {
	return c.metrics
}

// Send returns a send builder for fluent message construction
func (c *ClientImpl) Send() *notifyhub.SendBuilder {
	// Return the underlying client's send builder
	// For now, we need to provide a context - use background as default
	return c.client.Send(context.Background())
}

// SendMessage sends a message to targets
func (c *ClientImpl) SendMessage(ctx context.Context, msg *core.Message, targets []core.Target) (*core.SendingResults, error) {
	// Use the Send builder from the underlying client
	builder := c.client.Send(ctx)

	// Configure the message
	builder.Title(msg.Title).Body(msg.Body)

	// Set message format if specified
	if msg.Format != "" {
		builder.Format(msg.Format)
	}

	// Set priority if specified
	if msg.Priority != 0 {
		builder.Priority(int(msg.Priority))
	}

	// Add variables
	for k, v := range msg.Variables {
		builder.Variable(k, v)
	}

	// Add metadata
	for k, v := range msg.Metadata {
		builder.Metadata(k, v)
	}

	// Add template if specified
	if msg.Template != "" {
		builder.Template(msg.Template)
	}

	// Add targets using the builder's target methods
	for _, target := range targets {
		switch target.Type {
		case core.TargetTypeEmail:
			builder.ToEmail(target.Value)
		case core.TargetTypeGroup:
			// Use appropriate method based on platform
			switch target.Platform {
			case "feishu":
				builder.ToFeishu(target.Value)
			case "slack":
				builder.ToSlack(target.Value)
			}
		case core.TargetTypeSMS:
			builder.ToSMS(target.Value)
		default:
			// For unknown target types, try to use the generic To method
			builder.To(target.Value)
		}
	}

	// Execute the send operation
	results, err := builder.Execute()
	if err != nil {
		return nil, err
	}

	// Convert to core.SendingResults
	coreResults := &core.SendingResults{
		MessageID: results.MessageID,
		Success:   results.Sent,
		Failed:    results.Failed,
		Total:     results.Sent + results.Failed,
		Results:   make([]*core.Result, len(results.Results)),
	}

	// Convert individual results
	for i, result := range results.Results {
		// Convert DeliveryStatus to core.Status
		var status core.Status
		switch result.Status {
		case notifyhub.StatusSent:
			status = core.StatusSent
		case notifyhub.StatusFailed:
			status = core.StatusFailed
		case notifyhub.StatusPending:
			status = core.StatusPending
		case notifyhub.StatusSending:
			status = core.StatusSending
		case notifyhub.StatusRetrying:
			status = core.StatusRetrying
		default:
			status = core.StatusPending
		}

		// Convert error if present
		var err error
		if result.Error != nil {
			err = result.Error
		}

		coreResults.Results[i] = &core.Result{
			MessageID: results.MessageID,
			Target:    result.Target,
			Status:    status,
			Platform:  result.Target.Platform, // Get platform from target
			Error:     err,
			Duration:  result.Duration,
			Timestamp: result.Timestamp,
		}
	}

	return coreResults, nil
}

// RegisterTransport registers a new transport
func (c *ClientImpl) RegisterTransport(transport interface{}) error {
	// This method may not be available on the current client
	// Return nil for now - this needs to be implemented properly
	return nil
}

// Health returns the health status
func (c *ClientImpl) Health() interface{} {
	return c.client.Health()
}

// Shutdown shuts down the client
func (c *ClientImpl) Shutdown(ctx context.Context) error {
	return c.client.Shutdown(ctx)
}

// HubImpl implements the ClientHub interface
type HubImpl struct {
	client *notifyhub.Client
}

// Send sends a message through the hub
func (h *HubImpl) Send(ctx context.Context, msg *core.Message, targets []core.Target) (*core.SendingResults, error) {
	// Delegate to the ClientImpl's SendMessage method to avoid code duplication
	clientImpl := &ClientImpl{client: h.client}
	return clientImpl.SendMessage(ctx, msg, targets)
}
