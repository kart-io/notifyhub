// Platform registry implementation
package platform

import (
	"context"
	"fmt"
	"sync"

	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// registryImpl implements the Registry interface
type registryImpl struct {
	factories map[string]Factory
	instances map[string]Platform
	configs   map[string]interface{}
	logger    logger.Logger
	mu        sync.RWMutex
}

// NewRegistry creates a new platform registry
func NewRegistry(logger logger.Logger) Registry {
	return &registryImpl{
		factories: make(map[string]Factory),
		instances: make(map[string]Platform),
		configs:   make(map[string]interface{}),
		logger:    logger,
	}
}

// RegisterFactory registers a platform factory
func (r *registryImpl) RegisterFactory(name string, factory Factory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("platform %s already registered", name)
	}

	r.factories[name] = factory
	r.logger.Info("Platform factory registered", "platform", name)
	return nil
}

// SetConfig sets configuration for a platform
func (r *registryImpl) SetConfig(name string, config interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.configs[name] = config

	// If instance already exists, recreate it with new config
	delete(r.instances, name)

	return nil
}

// GetPlatform gets or creates a platform instance
func (r *registryImpl) GetPlatform(name string) (Platform, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Return existing instance if available
	if instance, exists := r.instances[name]; exists {
		return instance, nil
	}

	// Check if factory is registered
	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("platform %s not registered", name)
	}

	// Get configuration
	config, hasConfig := r.configs[name]
	if !hasConfig {
		return nil, fmt.Errorf("no configuration found for platform %s", name)
	}

	// Create instance
	instance, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform %s: %w", name, err)
	}

	r.instances[name] = instance
	r.logger.Info("Platform instance created", "platform", name)
	return instance, nil
}

// ListPlatforms returns a list of registered platform names
func (r *registryImpl) ListPlatforms() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	platforms := make([]string, 0, len(r.factories))
	for name := range r.factories {
		platforms = append(platforms, name)
	}
	return platforms
}

// Health checks the health of all platform instances
func (r *registryImpl) Health(ctx context.Context) map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health := make(map[string]error)
	for name, instance := range r.instances {
		health[name] = instance.IsHealthy(ctx)
	}
	return health
}

// Close closes all platform instances
func (r *registryImpl) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for name, instance := range r.instances {
		if err := instance.Close(); err != nil {
			r.logger.Error("Failed to close platform", "platform", name, "error", err)
			lastErr = err
		}
	}

	// Clear instances
	r.instances = make(map[string]Platform)
	return lastErr
}
