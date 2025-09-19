package registry

import (
	"fmt"

	"github.com/kart-io/notifyhub/platforms/common"
)

// Factory creates platform-specific builders and transports
type Factory struct {
	registry *Registry
}

// NewFactory creates a new platform factory
func NewFactory() *Factory {
	return &Factory{
		registry: GlobalRegistry,
	}
}

// CreateBuilder creates a platform-specific builder
func (f *Factory) CreateBuilder(platform string, base *common.MessageBuilder) (common.PlatformBuilder, error) {
	adapter, err := f.registry.Get(platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform adapter: %w", err)
	}

	return adapter.CreateBuilder(base), nil
}

// CreateTransport creates a platform-specific transport
func (f *Factory) CreateTransport(platform string, config interface{}) (interface{}, error) {
	adapter, err := f.registry.Get(platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform adapter: %w", err)
	}

	return adapter.CreateTransport(config)
}

// RegisterDefaultPlatforms registers all default platform adapters
func (f *Factory) RegisterDefaultPlatforms() error {
	// Note: Platform adapters should be registered by their respective packages
	// during initialization to avoid circular imports
	// This method is kept for interface compatibility
	return nil
}

// GetSupportedFormats returns supported formats for a platform
func (f *Factory) GetSupportedFormats(platform string) ([]string, error) {
	adapter, err := f.registry.Get(platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform adapter: %w", err)
	}

	formats := adapter.Supports()
	return formats, nil
}

// ValidateMessage validates a message for a specific platform
func (f *Factory) ValidateMessage(platform string, msg *common.MessageBuilder) error {
	adapter, err := f.registry.Get(platform)
	if err != nil {
		return fmt.Errorf("failed to get platform adapter: %w", err)
	}

	message := msg.Build()
	return adapter.Validate(message)
}

// Global factory instance
var GlobalFactory = NewFactory()

// Global convenience functions

// CreatePlatformBuilder creates a platform-specific builder
func CreatePlatformBuilder(platform string, base *common.MessageBuilder) (common.PlatformBuilder, error) {
	return GlobalFactory.CreateBuilder(platform, base)
}

// CreatePlatformTransport creates a platform-specific transport
func CreatePlatformTransport(platform string, config interface{}) (interface{}, error) {
	return GlobalFactory.CreateTransport(platform, config)
}

// RegisterDefaults registers all default platform adapters
func RegisterDefaults() error {
	return GlobalFactory.RegisterDefaultPlatforms()
}
