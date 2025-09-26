// Package client provides factory functions to create NotifyHub clients
package client

import (
	"fmt"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/async"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
)

// ClientFactory creates NotifyHub clients with specified configuration
type ClientFactory struct {
	logger logger.Logger
}

// NewFactory creates a new client factory
func NewFactory(logger logger.Logger) *ClientFactory {
	return &ClientFactory{
		logger: logger,
	}
}

// CreateClient creates a new client with the given configuration options
func (f *ClientFactory) CreateClient(opts ...config.Option) (Client, error) {
	// Create configuration from options
	cfg, err := config.NewConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// Create core dispatcher
	dispatcher, err := core.NewDispatcher(cfg, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispatcher: %w", err)
	}

	// Create sync client implementation
	syncClient := &SyncClient{
		dispatcher: dispatcher,
		logger:     f.logger,
	}

	// If async is enabled, create async client
	if cfg.IsAsyncEnabled() {
		// Create async executor
		queueSize := 1000 // default queue size
		workerCount := cfg.Workers()

		executor := async.NewAsyncExecutor(queueSize, workerCount, dispatcher, f.logger)
		asyncClient := NewAsyncClient(syncClient, executor, f.logger)

		return asyncClient, nil
	}

	return syncClient, nil
}

// New creates a new NotifyHub client with the given configuration options
// This is a convenience function that uses the default logger
func New(opts ...config.Option) (Client, error) {
	defaultLogger := logger.New() // Use default logger constructor
	factory := NewFactory(defaultLogger)
	return factory.CreateClient(opts...)
}

// NewWithLogger creates a new NotifyHub client with custom logger
func NewWithLogger(logger logger.Logger, opts ...config.Option) (Client, error) {
	factory := NewFactory(logger)
	return factory.CreateClient(opts...)
}
