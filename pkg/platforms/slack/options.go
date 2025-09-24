package slack

import (
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

var (
	registerOnce sync.Once
)

// ensureRegistered ensures the Slack platform is registered with NotifyHub
func ensureRegistered() {
	registerOnce.Do(func() {
		_ = notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
			Name:    "slack",
			Creator: NewSlackSender,
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

// WithSlack creates a HubOption for Slack platform with webhook URL
// This function automatically registers the Slack platform if not already registered
func WithSlack(webhookURL string, options ...func(map[string]interface{})) notifyhub.HubOption {
	ensureRegistered()

	config := map[string]interface{}{
		"webhook_url": webhookURL,
		"timeout":     30 * time.Second,
	}

	// Apply additional options
	for _, opt := range options {
		opt(config)
	}

	return notifyhub.WithCustomPlatform("slack", config)
}

// WithSlackTimeout adds timeout option for Slack configuration
func WithSlackTimeout(timeout time.Duration) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["timeout"] = timeout
	}
}

// WithSlackUsername adds username option for Slack messages
func WithSlackUsername(username string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["default_username"] = username
	}
}

// WithSlackIcon adds icon emoji option for Slack messages
func WithSlackIcon(iconEmoji string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["default_icon_emoji"] = iconEmoji
	}
}

// WithSlackChannel adds default channel for Slack messages
func WithSlackChannel(channel string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["default_channel"] = channel
	}
}
