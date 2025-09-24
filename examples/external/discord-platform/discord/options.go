package discord

import (
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

var (
	registerOnce sync.Once
)

// ensureRegistered ensures the Discord platform is registered with NotifyHub
func ensureRegistered() {
	registerOnce.Do(func() {
		_ = notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
			Name: platform.NameDiscord,
			Creator: func(config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
				return NewDiscordSender(config, logger)
			},
			DefaultOpts: func() map[string]interface{} {
				return map[string]interface{}{
					"timeout": 30 * time.Second,
				}
			},
			Validator: func(config map[string]interface{}) error {
				if _, ok := config["webhook_url"].(string); !ok {
					return fmt.Errorf("webhook_url is required")
				}
				if config["webhook_url"].(string) == "" {
					return fmt.Errorf("webhook_url cannot be empty")
				}
				return nil
			},
		})
	})
}

// WithDiscord creates a HubOption for Discord platform with webhook URL
// This function automatically registers the Discord platform if not already registered
func WithDiscord(webhookURL string, options ...func(map[string]interface{})) notifyhub.HubOption {
	ensureRegistered()

	config := map[string]interface{}{
		"webhook_url": webhookURL,
		"timeout":     30 * time.Second,
	}

	// Apply additional options
	for _, opt := range options {
		opt(config)
	}

	return notifyhub.WithCustomPlatform("discord", config)
}

// WithDiscordTimeout adds timeout option for Discord configuration
func WithDiscordTimeout(timeout time.Duration) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["timeout"] = timeout
	}
}

// WithDiscordUsername adds default username for Discord messages
func WithDiscordUsername(username string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["default_username"] = username
	}
}

// WithDiscordAvatar adds default avatar URL for Discord messages
func WithDiscordAvatar(avatarURL string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["default_avatar_url"] = avatarURL
	}
}
