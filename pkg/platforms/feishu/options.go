package feishu

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

// ensureRegistered ensures the Feishu platform is registered with NotifyHub
func ensureRegistered() {
	registerOnce.Do(func() {
		_ = notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
			Name: platform.NameFeishu,
			Creator: func(config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
				return NewFeishuSender(config, logger)
			},
			DefaultOpts: func() map[string]interface{} {
				return map[string]interface{}{
					"timeout":   30 * time.Second,
					"auth_mode": "none",
				}
			},
			Validator: func(config map[string]interface{}) error {
				if _, ok := config["webhook_url"].(string); !ok {
					return fmt.Errorf("webhook_url is required")
				}
				if config["webhook_url"].(string) == "" {
					return fmt.Errorf("webhook_url cannot be empty")
				}

				// Validate auth mode
				if authMode, ok := config["auth_mode"].(string); ok {
					switch AuthMode(authMode) {
					case AuthModeNone, AuthModeSignature, AuthModeKeywords:
						// Valid auth modes
					default:
						return fmt.Errorf("invalid auth_mode: %s", authMode)
					}
				}

				return nil
			},
		})
	})
}

// WithFeishu creates a HubOption for Feishu platform with webhook URL
// This function automatically registers the Feishu platform if not already registered
func WithFeishu(webhookURL string, options ...func(map[string]interface{})) notifyhub.HubOption {
	ensureRegistered()

	config := map[string]interface{}{
		"webhook_url": webhookURL,
		"timeout":     30 * time.Second,
		"auth_mode":   "none",
	}

	// Apply additional options
	for _, opt := range options {
		opt(config)
	}

	return notifyhub.WithCustomPlatform("feishu", config)
}

// WithFeishuSecret adds secret for signature authentication
func WithFeishuSecret(secret string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["secret"] = secret
		config["auth_mode"] = "signature"
	}
}

// WithFeishuKeywords adds keywords for keyword authentication
func WithFeishuKeywords(keywords []string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["keywords"] = keywords
		config["auth_mode"] = "keywords"
	}
}

// WithFeishuAuthMode sets the authentication mode explicitly
func WithFeishuAuthMode(mode AuthMode) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["auth_mode"] = string(mode)
	}
}

// WithFeishuTimeout adds timeout option for Feishu configuration
func WithFeishuTimeout(timeout time.Duration) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["timeout"] = timeout
	}
}

// Legacy compatibility functions (these will call the new platform-specific ones)

// WithFeishuSignature is a legacy function that configures Feishu with signature auth
// Deprecated: Use WithFeishu with WithFeishuSecret instead
func WithFeishuSignature(webhookURL, secret string) notifyhub.HubOption {
	return WithFeishu(webhookURL, WithFeishuSecret(secret))
}

// WithFeishuKeywordAuth is a legacy function that configures Feishu with keyword auth
// Deprecated: Use WithFeishu with WithFeishuKeywords instead
func WithFeishuKeywordAuth(webhookURL string, keywords []string) notifyhub.HubOption {
	return WithFeishu(webhookURL, WithFeishuKeywords(keywords))
}
