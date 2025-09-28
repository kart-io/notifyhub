// Package client provides factory functions to create NotifyHub clients
// This file focuses solely on client creation responsibility
package client

import (
	"fmt"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/async"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
	"github.com/kart-io/notifyhub/pkg/platform"
)

// ClientFactory creates NotifyHub clients with specified configuration
// Focuses solely on client creation, delegating configuration handling to other modules
type ClientFactory struct {
	logger    logger.Logger
	registry  *platform.Registry
	validator *ConfigValidator
}

// NewFactory creates a new client factory with instance-level platform registry
func NewFactory(logger logger.Logger) *ClientFactory {
	// Create instance-level registry instead of relying on global state
	registry := platform.NewRegistry(logger)
	validator := NewConfigValidator()

	return &ClientFactory{
		logger:    logger,
		registry:  registry,
		validator: validator,
	}
}

// CreateClient creates a new client with the given configuration options
// This method focuses solely on orchestrating client creation
func (f *ClientFactory) CreateClient(opts ...config.Option) (Client, error) {
	// Build and validate configuration
	cfg, err := config.NewConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// Validate configuration for registry compatibility
	if err := f.validator.ValidateRegistryCompatibility(f.registry, cfg); err != nil {
		return nil, fmt.Errorf("registry compatibility validation failed: %w", err)
	}

	// Register platforms from configuration
	if err := f.registerPlatformsFromConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to register platforms: %w", err)
	}

	// Create dispatcher (simplified call chain)
	dispatcher, err := f.createDispatcher(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispatcher: %w", err)
	}

	// Create appropriate client type based on configuration
	return f.createClientImplementation(cfg, dispatcher)
}

// registerPlatformsFromConfig registers platforms based on configuration
// This method handles the platform registration logic extracted from the giant factory
func (f *ClientFactory) registerPlatformsFromConfig(cfg *config.Config) error {
	// TODO: This will integrate with actual platform implementations
	// For now, this is a placeholder that maintains the interface

	// The platform registration logic will be implemented as platform packages become available
	// This separates the concern of platform registration from client creation

	return nil
}

// createDispatcher creates the core dispatcher with registry
func (f *ClientFactory) createDispatcher(cfg *config.Config) (core.Dispatcher, error) {
	// This can be extended when core.NewDispatcherWithRegistry is implemented
	return nil, fmt.Errorf("dispatcher creation not yet implemented")
}

// createClientImplementation creates the appropriate client implementation
func (f *ClientFactory) createClientImplementation(cfg *config.Config, dispatcher core.Dispatcher) (Client, error) {
	// Create sync client as base
	syncClient := NewSyncClient(dispatcher, f.logger)

	// If async is enabled, wrap with async client
	if cfg.IsAsyncEnabled() {
		// Create async executor
		queueSize := 1000 // default queue size
		workerCount := cfg.Workers()

		executor := async.NewAsyncExecutor(queueSize, workerCount, dispatcher, f.logger)
		return NewAsyncClient(syncClient, executor, f.logger), nil
	}

	return syncClient, nil
}

// Convenience functions for client creation

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
