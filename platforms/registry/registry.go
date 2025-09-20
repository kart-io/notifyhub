package registry

import (
	"fmt"
	"sync"

	"github.com/kart-io/notifyhub/platforms"
)

// PlatformInfo contains information about a registered platform
type PlatformInfo struct {
	Name         string
	Description  string
	Capabilities platforms.Capabilities
}

// Registry manages platform implementations
// This implements the enhanced platform registry pattern
type Registry struct {
	platforms map[string]platforms.Platform
	mutex     sync.RWMutex
}

// GlobalRegistry is the global platform registry
var GlobalRegistry = NewRegistry()

// NewRegistry creates a new platform registry
func NewRegistry() *Registry {
	return &Registry{
		platforms: make(map[string]platforms.Platform),
	}
}

// Register registers a platform implementation
func (r *Registry) Register(platform platforms.Platform) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := platform.Name()
	if _, exists := r.platforms[name]; exists {
		return fmt.Errorf("platform %s already registered", name)
	}

	r.platforms[name] = platform
	return nil
}

// Get returns a platform by name
func (r *Registry) Get(name string) (platforms.Platform, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	platform, exists := r.platforms[name]
	if !exists {
		return nil, fmt.Errorf("platform %s not found", name)
	}

	return platform, nil
}

// List returns all registered platform names
func (r *Registry) List() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.platforms))
	for name := range r.platforms {
		names = append(names, name)
	}
	return names
}

// ListWithInfo returns information about all registered platforms
func (r *Registry) ListWithInfo() []PlatformInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	info := make([]PlatformInfo, 0, len(r.platforms))
	for _, platform := range r.platforms {
		info = append(info, PlatformInfo{
			Name:         platform.Name(),
			Description:  platform.Description(),
			Capabilities: platform.Capabilities(),
		})
	}
	return info
}

// GetAll returns all registered platforms
func (r *Registry) GetAll() map[string]platforms.Platform {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]platforms.Platform)
	for name, platform := range r.platforms {
		result[name] = platform
	}
	return result
}

// GetCapabilities returns the capabilities of a specific platform
func (r *Registry) GetCapabilities(name string) (platforms.Capabilities, error) {
	platform, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	return platform.Capabilities(), nil
}

// Unregister removes a platform
func (r *Registry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.platforms[name]; !exists {
		return fmt.Errorf("platform %s not found", name)
	}

	delete(r.platforms, name)
	return nil
}

// Clear removes all registered platforms
func (r *Registry) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.platforms = make(map[string]platforms.Platform)
}

// Global convenience functions

// Register registers a platform globally
func Register(platform platforms.Platform) error {
	return GlobalRegistry.Register(platform)
}

// GetPlatform returns a platform by name
func GetPlatform(name string) (platforms.Platform, error) {
	return GlobalRegistry.Get(name)
}

// ListPlatforms returns all registered platform names
func ListPlatforms() []string {
	return GlobalRegistry.List()
}

// ListPlatformsWithInfo returns information about all registered platforms
func ListPlatformsWithInfo() []PlatformInfo {
	return GlobalRegistry.ListWithInfo()
}

// GetPlatformCapabilities returns the capabilities of a specific platform
func GetPlatformCapabilities(name string) (platforms.Capabilities, error) {
	return GlobalRegistry.GetCapabilities(name)
}
