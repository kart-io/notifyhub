package registry

import (
	"fmt"
	"sync"

	"github.com/kart-io/notifyhub/platforms/common"
)

// Registry manages platform adapters
// This implements the platform registry pattern from the proposal
type Registry struct {
	adapters map[string]common.PlatformAdapter
	mutex    sync.RWMutex
}

// GlobalRegistry is the global platform registry
var GlobalRegistry = NewRegistry()

// NewRegistry creates a new platform registry
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]common.PlatformAdapter),
	}
}

// Register registers a platform adapter
func (r *Registry) Register(adapter common.PlatformAdapter) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := adapter.Name()
	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("platform %s already registered", name)
	}

	r.adapters[name] = adapter
	return nil
}

// Get returns a platform adapter by name
func (r *Registry) Get(name string) (common.PlatformAdapter, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapter, exists := r.adapters[name]
	if !exists {
		return nil, fmt.Errorf("platform %s not found", name)
	}

	return adapter, nil
}

// List returns all registered platform names
func (r *Registry) List() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// GetAllAdapters returns all registered adapters
func (r *Registry) GetAllAdapters() map[string]common.PlatformAdapter {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]common.PlatformAdapter)
	for name, adapter := range r.adapters {
		result[name] = adapter
	}
	return result
}

// Unregister removes a platform adapter
func (r *Registry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.adapters[name]; !exists {
		return fmt.Errorf("platform %s not found", name)
	}

	delete(r.adapters, name)
	return nil
}

// Global convenience functions

// Register registers a platform adapter globally
func Register(adapter common.PlatformAdapter) error {
	return GlobalRegistry.Register(adapter)
}

// GetPlatform returns a platform adapter by name
func GetPlatform(name string) (common.PlatformAdapter, error) {
	return GlobalRegistry.Get(name)
}

// ListPlatforms returns all registered platform names
func ListPlatforms() []string {
	return GlobalRegistry.List()
}
