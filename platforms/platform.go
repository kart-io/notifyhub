package platforms

import (
	"context"
)

// Platform defines the unified interface for all notification platforms.
// It combines transport, configuration, and capability management.
type Platform interface {
	// Basic platform information
	Name() string
	Description() string

	// Capabilities returns the platform's capabilities
	Capabilities() Capabilities

	// Configuration management
	ValidateConfig(config interface{}) error
	CreateTransport(config interface{}) (Transport, error)

	// Default configuration for the platform
	DefaultConfig() interface{}
}

// Transport defines the interface for sending messages to a platform
type Transport interface {
	// Send sends a message to a target on this platform
	Send(ctx context.Context, msg interface{}, target interface{}) (interface{}, error)

	// Name returns the transport name (usually same as platform name)
	Name() string

	// Shutdown gracefully shuts down the transport
	Shutdown() error

	// Health checks if the transport is healthy
	Health(ctx context.Context) error
}

// BasePlatform provides common functionality for platform implementations
type BasePlatform struct {
	name         string
	description  string
	capabilities Capabilities
}

// NewBasePlatform creates a new base platform with common functionality
func NewBasePlatform(name, description string, caps Capabilities) *BasePlatform {
	return &BasePlatform{
		name:         name,
		description:  description,
		capabilities: caps,
	}
}

// Name returns the platform name
func (p *BasePlatform) Name() string {
	return p.name
}

// Description returns the platform description
func (p *BasePlatform) Description() string {
	return p.description
}

// Capabilities returns the platform capabilities
func (p *BasePlatform) Capabilities() Capabilities {
	return p.capabilities
}
