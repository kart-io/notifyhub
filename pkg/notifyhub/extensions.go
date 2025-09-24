// Package notifyhub extensions provide plugin-style architecture for external platforms
package notifyhub

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// PlatformExtension provides a standardized way to add external platforms
type PlatformExtension struct {
	Name        string
	Creator     platform.ExternalSenderCreator
	DefaultOpts func() map[string]interface{}
	Validator   func(config map[string]interface{}) error
}

// ExtensionRegistry manages platform extensions
type ExtensionRegistry struct {
	mu         sync.RWMutex
	extensions map[string]*PlatformExtension
}

var globalRegistry = &ExtensionRegistry{
	extensions: make(map[string]*PlatformExtension),
}

// RegisterExtension registers a new platform extension
func RegisterExtension(ext *PlatformExtension) error {
	if ext == nil {
		return fmt.Errorf("extension cannot be nil")
	}
	if ext.Name == "" {
		return fmt.Errorf("extension name cannot be empty")
	}
	if ext.Creator == nil {
		return fmt.Errorf("extension creator cannot be nil")
	}

	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if _, exists := globalRegistry.extensions[ext.Name]; exists {
		return fmt.Errorf("platform %s already registered", ext.Name)
	}

	globalRegistry.extensions[ext.Name] = ext

	// Also register with the platform registry
	platform.RegisterPlatform(ext.Name, ext.Creator)

	return nil
}

// GetExtensions returns all registered extensions
func GetExtensions() map[string]*PlatformExtension {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make(map[string]*PlatformExtension, len(globalRegistry.extensions))
	for k, v := range globalRegistry.extensions {
		result[k] = v
	}
	return result
}

// GetExtension returns a specific extension by name
func GetExtension(name string) (*PlatformExtension, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	ext, exists := globalRegistry.extensions[name]
	return ext, exists
}

// IsExtensionRegistered checks if an extension is registered
func IsExtensionRegistered(name string) bool {
	_, exists := GetExtension(name)
	return exists
}

// WithExtension creates a HubOption for an external platform using generic configuration
func WithExtension(platformName string, config map[string]interface{}) HubOption {
	return func(cfg *HubConfig) error {
		// Validate configuration if validator exists
		if ext, exists := GetExtension(platformName); exists && ext.Validator != nil {
			if err := ext.Validator(config); err != nil {
				// Return validation error
				return fmt.Errorf("validation failed for platform %s: %w", platformName, err)
			}
		}

		cfg.Platforms[platformName] = PlatformConfig(config)
		return nil
	}
}

// WithExtensionDefaults creates a HubOption for an external platform using default configuration
func WithExtensionDefaults(platformName string, overrides ...map[string]interface{}) HubOption {
	return func(cfg *HubConfig) error {
		ext, exists := GetExtension(platformName)
		if !exists {
			return fmt.Errorf("platform %s is not registered", platformName)
		}

		// Start with defaults
		config := make(map[string]interface{})
		if ext.DefaultOpts != nil {
			config = ext.DefaultOpts()
		}

		// Apply overrides
		for _, override := range overrides {
			for k, v := range override {
				config[k] = v
			}
		}

		// Use WithExtension to apply configuration with validation
		return WithExtension(platformName, config)(cfg)
	}
}

// ConfigBuilder provides a fluent interface for building platform configurations
type ConfigBuilder struct {
	config map[string]interface{}
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: make(map[string]interface{}),
	}
}

// Set sets a configuration value
func (cb *ConfigBuilder) Set(key string, value interface{}) *ConfigBuilder {
	cb.config[key] = value
	return cb
}

// SetIf conditionally sets a configuration value
func (cb *ConfigBuilder) SetIf(condition bool, key string, value interface{}) *ConfigBuilder {
	if condition {
		cb.config[key] = value
	}
	return cb
}

// Merge merges another configuration map
func (cb *ConfigBuilder) Merge(other map[string]interface{}) *ConfigBuilder {
	for k, v := range other {
		cb.config[k] = v
	}
	return cb
}

// Build returns the built configuration
func (cb *ConfigBuilder) Build() map[string]interface{} {
	result := make(map[string]interface{}, len(cb.config))
	for k, v := range cb.config {
		result[k] = v
	}
	return result
}

// Helper functions for common configuration patterns

// SimpleWebhookConfig creates configuration for webhook-based platforms
func SimpleWebhookConfig(webhookURL string, options ...func(map[string]interface{})) map[string]interface{} {
	config := map[string]interface{}{
		"webhook_url": webhookURL,
	}

	for _, opt := range options {
		opt(config)
	}

	return config
}

// WithSecret adds a secret to webhook configuration
func WithSecret(secret string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["secret"] = secret
	}
}

// WithTimeout adds a timeout to configuration
func WithTimeoutConfig(timeout interface{}) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["timeout"] = timeout
	}
}

// WithHeaders adds headers to configuration
func WithHeaders(headers map[string]string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["headers"] = headers
	}
}

// AutoDetectAndRegister attempts to auto-detect and register platform from configuration
func AutoDetectAndRegister(platformName string, config map[string]interface{}, creator platform.ExternalSenderCreator) error {
	ext := &PlatformExtension{
		Name:    platformName,
		Creator: creator,
		DefaultOpts: func() map[string]interface{} {
			return make(map[string]interface{})
		},
	}

	return RegisterExtension(ext)
}

// GetPlatformInfo returns information about all registered platforms
type PlatformInfo struct {
	Name         string                `json:"name"`
	IsBuiltIn    bool                  `json:"is_built_in"`
	IsExtension  bool                  `json:"is_extension"`
	Capabilities platform.Capabilities `json:"capabilities,omitempty"`
}

// GetAllPlatformInfo returns information about all available platforms
func GetAllPlatformInfo() []PlatformInfo {
	var infos []PlatformInfo

	// Built-in platforms
	builtins := []string{"feishu", "email", "sms"}
	for _, name := range builtins {
		infos = append(infos, PlatformInfo{
			Name:        name,
			IsBuiltIn:   true,
			IsExtension: false,
		})
	}

	// Extension platforms
	for name := range GetExtensions() {
		// Skip if already in builtins
		isBuiltIn := false
		for _, builtin := range builtins {
			if name == builtin {
				isBuiltIn = true
				break
			}
		}

		if !isBuiltIn {
			infos = append(infos, PlatformInfo{
				Name:        name,
				IsBuiltIn:   false,
				IsExtension: true,
			})
		}
	}

	return infos
}

// ConfigTemplate provides configuration templates for common scenarios
type ConfigTemplate struct {
	Name        string
	Description string
	Template    map[string]interface{}
	Required    []string
	Optional    []string
}

// GetConfigTemplate returns a configuration template for a platform
func GetConfigTemplate(platformName string) (*ConfigTemplate, error) {
	ext, exists := GetExtension(platformName)
	if !exists {
		return nil, fmt.Errorf("platform %s not found", platformName)
	}

	template := &ConfigTemplate{
		Name:        platformName,
		Description: fmt.Sprintf("Configuration template for %s platform", platformName),
		Required:    []string{},
		Optional:    []string{},
	}

	if ext.DefaultOpts != nil {
		template.Template = ext.DefaultOpts()

		// Analyze template to determine required/optional fields
		for key, value := range template.Template {
			if value == nil || isZeroValue(value) {
				template.Required = append(template.Required, key)
			} else {
				template.Optional = append(template.Optional, key)
			}
		}
	}

	return template, nil
}

// isZeroValue checks if a value is a zero value
func isZeroValue(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Slice, reflect.Map, reflect.Chan:
		return v.Len() == 0
	default:
		return false
	}
}
