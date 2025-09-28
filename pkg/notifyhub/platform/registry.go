// Package platform provides public APIs for platform registration and extension
package platform

import (
	"log"
	"sync"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// NOTE: Platform, SendResult, and Capabilities are defined in interface.go
// This file imports those definitions to avoid duplication

const (
	NameEmail   = "email"
	NameFeishu  = "feishu"
	NameSMS     = "sms"
	NameSlack   = "slack"
	NameDiscord = "discord"
	NameTeams   = "teams"
	NameWebhook = "webhook"
)

// ExternalSender is deprecated: use Platform interface instead
// This is kept for backward compatibility and will be removed in a future version
type ExternalSender = Platform

// NOTE: Message and Target types are now provided by unified packages:
// - Message: github.com/kart-io/notifyhub/pkg/notifyhub/message.Message
// - Target: github.com/kart-io/notifyhub/pkg/notifyhub/target.Target

// SendResult and Capabilities are now defined in interface.go
// These type aliases are kept for backward compatibility

// PlatformCreator is a function that creates a platform with given configuration
type PlatformCreator func(config map[string]interface{}, logger logger.Logger) (Platform, error)

// ExternalSenderCreator is deprecated: use PlatformCreator instead
// This is kept for backward compatibility and will be removed in a future version
type ExternalSenderCreator = PlatformCreator

// RegisterPlatform registers a platform creator (DEPRECATED)
// This is deprecated in favor of instance-level registration via client configuration.
// Use notifyhub.New() with WithPlatform() options instead.
// This function is kept for backward compatibility and will be removed in a future version.
func RegisterPlatform(platformName string, creator PlatformCreator) {
	// This is now deprecated - platforms should be registered via client configuration
	logDeprecationWarning("platform.RegisterPlatform", "use notifyhub.New() with platform-specific options")

	deprecatedMutex.Lock()
	defer deprecatedMutex.Unlock()

	// Store in deprecated global registry for backward compatibility only
	deprecatedGlobalRegistry[platformName] = creator
}

// Global registry for platform creators (DEPRECATED)
// This global registry is deprecated and will be removed in a future version.
// Use instance-level registries instead.
var deprecatedGlobalRegistry = make(map[string]PlatformCreator)

// GetRegisteredCreators returns all registered platform creators (DEPRECATED)
// This is deprecated - platforms should be registered at the client level.
func GetRegisteredCreators() map[string]PlatformCreator {
	logDeprecationWarning("platform.GetRegisteredCreators", "use instance-level platform management")

	deprecatedMutex.RLock()
	defer deprecatedMutex.RUnlock()

	result := make(map[string]PlatformCreator, len(deprecatedGlobalRegistry))
	for name, creator := range deprecatedGlobalRegistry {
		result[name] = creator
	}
	return result
}

// Note: senderAdapter has been moved to internal package to avoid
// circular dependencies and maintain Go package design principles

// GetRegisteredPlatforms returns a list of all registered platform names (DEPRECATED)
// This is deprecated - use instance-level platform management instead.
func GetRegisteredPlatforms() []string {
	logDeprecationWarning("platform.GetRegisteredPlatforms", "use instance-level platform management")

	deprecatedMutex.RLock()
	defer deprecatedMutex.RUnlock()

	names := make([]string, 0, len(deprecatedGlobalRegistry))
	for name := range deprecatedGlobalRegistry {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a platform is registered (DEPRECATED)
// This is deprecated - use instance-level platform management instead.
func IsRegistered(platformName string) bool {
	logDeprecationWarning("platform.IsRegistered", "use instance-level platform management")

	deprecatedMutex.RLock()
	defer deprecatedMutex.RUnlock()

	_, exists := deprecatedGlobalRegistry[platformName]
	return exists
}

// Mutex for deprecated global registry access
var deprecatedMutex sync.RWMutex

// logDeprecationWarning logs a deprecation warning
func logDeprecationWarning(funcName, replacement string) {
	log.Printf("[DEPRECATED] %s is deprecated. Please use %s instead. This function will be removed in a future version.", funcName, replacement)
}
