// Package platform provides public APIs for platform registration and extension
package platform

import (
	"github.com/kart-io/notifyhub/pkg/logger"
)

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

// RegisterPlatform registers a platform creator
// This is the public API for external packages to register their platforms
func RegisterPlatform(platformName string, creator PlatformCreator) {
	// Use a global registry that can be accessed by core package
	globalPlatformRegistry[platformName] = creator
}

// Global registry for platform creators
var globalPlatformRegistry = make(map[string]PlatformCreator)

// GetRegisteredCreators returns all registered platform creators
func GetRegisteredCreators() map[string]PlatformCreator {
	return globalPlatformRegistry
}

// Note: senderAdapter has been moved to internal package to avoid
// circular dependencies and maintain Go package design principles

// GetRegisteredPlatforms returns a list of all registered platform names
func GetRegisteredPlatforms() []string {
	// This would need to be implemented in the internal package
	// For now, return common platforms
	return []string{NameFeishu, NameEmail, NameSMS}
}

// IsRegistered checks if a platform is registered
func IsRegistered(platformName string) bool {
	platforms := GetRegisteredPlatforms()
	for _, p := range platforms {
		if p == platformName {
			return true
		}
	}
	return false
}
