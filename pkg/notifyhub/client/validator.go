// Package client provides configuration validation for NotifyHub clients
package client

import (
	"fmt"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/platform"
)

// ConfigValidator handles validation of client and platform configurations
type ConfigValidator struct{}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateClientConfig validates client-specific configuration
func (v *ConfigValidator) ValidateClientConfig(cfg *ClientConfig) error {
	if cfg == nil {
		return fmt.Errorf("client config cannot be nil")
	}

	// Validate logger
	if cfg.Logger == nil {
		return fmt.Errorf("logger cannot be nil")
	}

	// Validate async configuration
	if cfg.Async && cfg.Workers <= 0 {
		return fmt.Errorf("worker count must be positive when async is enabled, got: %d", cfg.Workers)
	}

	// Validate platform options if any
	if len(cfg.PlatformOptions) > 0 {
		if err := v.validatePlatformOptions(cfg.PlatformOptions); err != nil {
			return fmt.Errorf("platform options validation failed: %w", err)
		}
	}

	return nil
}

// validatePlatformOptions validates platform configuration options
func (v *ConfigValidator) validatePlatformOptions(opts []config.Option) error {
	// Create a temporary config to test the options
	tempConfig, err := config.NewConfig(opts...)
	if err != nil {
		return fmt.Errorf("failed to build config from options: %w", err)
	}

	// The config package already validates the configuration
	// This is just an additional layer to catch issues early
	return v.validateBuiltConfig(tempConfig)
}

// validateBuiltConfig validates a fully built configuration
func (v *ConfigValidator) validateBuiltConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Check that at least one platform is configured
	if cfg.Feishu == nil && cfg.Email == nil && cfg.Webhook == nil {
		return fmt.Errorf("at least one platform must be configured")
	}

	// Validate individual platform configurations
	if cfg.Feishu != nil {
		if err := v.validateFeishuConfig(cfg.Feishu); err != nil {
			return fmt.Errorf("feishu config validation failed: %w", err)
		}
	}

	if cfg.Email != nil {
		if err := v.validateEmailConfig(cfg.Email); err != nil {
			return fmt.Errorf("email config validation failed: %w", err)
		}
	}

	if cfg.Webhook != nil {
		if err := v.validateWebhookConfig(cfg.Webhook); err != nil {
			return fmt.Errorf("webhook config validation failed: %w", err)
		}
	}

	return nil
}

// validateFeishuConfig validates Feishu platform configuration
func (v *ConfigValidator) validateFeishuConfig(cfg *config.FeishuConfig) error {
	if cfg == nil {
		return fmt.Errorf("feishu config cannot be nil")
	}

	// The config package already does detailed validation
	// This can be extended with additional client-specific validation if needed
	return nil
}

// validateEmailConfig validates email platform configuration
func (v *ConfigValidator) validateEmailConfig(cfg *config.EmailConfig) error {
	if cfg == nil {
		return fmt.Errorf("email config cannot be nil")
	}

	// Additional validation beyond what's in the config package
	if cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 {
		return fmt.Errorf("email port must be between 1 and 65535, got: %d", cfg.SMTPPort)
	}

	return nil
}

// validateWebhookConfig validates webhook platform configuration
func (v *ConfigValidator) validateWebhookConfig(cfg *config.WebhookConfig) error {
	if cfg == nil {
		return fmt.Errorf("webhook config cannot be nil")
	}

	// Additional validation beyond what's in the config package
	validMethods := map[string]bool{
		"GET":    true,
		"POST":   true,
		"PUT":    true,
		"PATCH":  true,
		"DELETE": true,
	}

	if !validMethods[cfg.Method] {
		return fmt.Errorf("invalid webhook method: %s", cfg.Method)
	}

	return nil
}

// ValidateRegistryCompatibility validates that platforms can be registered
func (v *ConfigValidator) ValidateRegistryCompatibility(registry *platform.Registry, cfg *config.Config) error {
	if registry == nil {
		return fmt.Errorf("platform registry cannot be nil")
	}

	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Check for registry state conflicts
	existingPlatforms := registry.ListRegistered()

	// Warn about potential conflicts
	if cfg.Feishu != nil {
		for _, existing := range existingPlatforms {
			if existing == "feishu" {
				return fmt.Errorf("feishu platform already registered in registry")
			}
		}
	}

	if cfg.Email != nil {
		for _, existing := range existingPlatforms {
			if existing == "email" {
				return fmt.Errorf("email platform already registered in registry")
			}
		}
	}

	if cfg.Webhook != nil {
		for _, existing := range existingPlatforms {
			if existing == "webhook" {
				return fmt.Errorf("webhook platform already registered in registry")
			}
		}
	}

	return nil
}

// ValidatePlatformRequirements validates that all required platform dependencies are available
func (v *ConfigValidator) ValidatePlatformRequirements(cfg *config.Config) error {
	// This can be extended to check for platform-specific requirements
	// For example, checking if required environment variables are set
	// or if external dependencies are available

	return nil
}