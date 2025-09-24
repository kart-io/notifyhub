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
					"channel":    "#general",
					"username":   "NotifyHub Bot",
					"icon_emoji": ":robot_face:",
					"timeout":    30 * time.Second,
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

// WithSlack creates a HubOption for Slack platform
func WithSlack(webhookURL string, options ...func(map[string]interface{})) notifyhub.HubOption {
	ensureRegistered()

	config := map[string]interface{}{
		"webhook_url": webhookURL,
		"channel":     "#general",
		"username":    "NotifyHub Bot",
		"icon_emoji":  ":robot_face:",
		"timeout":     30 * time.Second,
	}

	// Apply additional options
	for _, opt := range options {
		opt(config)
	}

	return notifyhub.WithCustomPlatform("slack", config)
}

// WithSlackChannel sets the default channel for Slack messages
func WithSlackChannel(channel string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["channel"] = channel
	}
}

// WithSlackUsername sets the bot username for Slack messages
func WithSlackUsername(username string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["username"] = username
	}
}

// WithSlackIconEmoji sets the bot icon emoji for Slack messages
func WithSlackIconEmoji(iconEmoji string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["icon_emoji"] = iconEmoji
	}
}

// WithSlackTimeout sets the timeout for Slack requests
func WithSlackTimeout(timeout time.Duration) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["timeout"] = timeout
	}
}
