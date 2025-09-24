package sms

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

// ensureRegistered ensures the SMS platform is registered with NotifyHub
func ensureRegistered() {
	registerOnce.Do(func() {
		_ = notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
			Name: platform.NameSMS,
			Creator: func(config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
				return NewSMSSender(config, logger)
			},
			DefaultOpts: func() map[string]interface{} {
				return map[string]interface{}{
					"timeout": 30 * time.Second,
				}
			},
			Validator: func(config map[string]interface{}) error {
				if _, ok := config["provider"].(string); !ok {
					return fmt.Errorf("provider is required")
				}
				if config["provider"].(string) == "" {
					return fmt.Errorf("provider cannot be empty")
				}

				if _, ok := config["api_key"].(string); !ok {
					return fmt.Errorf("api_key is required")
				}
				if config["api_key"].(string) == "" {
					return fmt.Errorf("api_key cannot be empty")
				}

				if _, ok := config["from"].(string); !ok {
					return fmt.Errorf("from is required")
				}
				if config["from"].(string) == "" {
					return fmt.Errorf("from cannot be empty")
				}

				// Validate provider type
				provider := config["provider"].(string)
				switch ProviderType(provider) {
				case ProviderTwilio, ProviderAliyun, ProviderTencent, ProviderAWSSNS:
					// Valid providers
				default:
					return fmt.Errorf("invalid provider: %s", provider)
				}

				return nil
			},
		})
	})
}

// WithSMS creates a HubOption for SMS platform with provider configuration
// This function automatically registers the SMS platform if not already registered
func WithSMS(provider, apiKey, from string, options ...func(map[string]interface{})) notifyhub.HubOption {
	ensureRegistered()

	config := map[string]interface{}{
		"provider": provider,
		"api_key":  apiKey,
		"from":     from,
		"timeout":  30 * time.Second,
	}

	// Apply additional options
	for _, opt := range options {
		opt(config)
	}

	return notifyhub.WithCustomPlatform("sms", config)
}

// WithSMSTwilio creates a HubOption for SMS platform using Twilio provider
func WithSMSTwilio(apiKey, from string, options ...func(map[string]interface{})) notifyhub.HubOption {
	return WithSMS("twilio", apiKey, from, options...)
}

// WithSMSAliyun creates a HubOption for SMS platform using Aliyun provider
func WithSMSAliyun(apiKey, from string, options ...func(map[string]interface{})) notifyhub.HubOption {
	return WithSMS("aliyun", apiKey, from, options...)
}

// WithSMSTencent creates a HubOption for SMS platform using Tencent provider
func WithSMSTencent(apiKey, from string, options ...func(map[string]interface{})) notifyhub.HubOption {
	return WithSMS("tencent", apiKey, from, options...)
}

// WithSMSAWS creates a HubOption for SMS platform using AWS SNS provider
func WithSMSAWS(apiKey, from string, options ...func(map[string]interface{})) notifyhub.HubOption {
	return WithSMS("aws_sns", apiKey, from, options...)
}

// WithSMSAPISecret adds API secret for SMS configuration (required for some providers)
func WithSMSAPISecret(apiSecret string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["api_secret"] = apiSecret
	}
}

// WithSMSRegion sets the region for SMS provider (required for some providers like AWS)
func WithSMSRegion(region string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["region"] = region
	}
}

// WithSMSTemplate sets the SMS template ID for template-based SMS providers
func WithSMSTemplate(template string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["template"] = template
	}
}

// WithSMSSignName sets the SMS signature name (required for some providers like Aliyun)
func WithSMSSignName(signName string) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["sign_name"] = signName
	}
}

// WithSMSTimeout sets the timeout for SMS operations
func WithSMSTimeout(timeout time.Duration) func(map[string]interface{}) {
	return func(config map[string]interface{}) {
		config["timeout"] = timeout
	}
}
