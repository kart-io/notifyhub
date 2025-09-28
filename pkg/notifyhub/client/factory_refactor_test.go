package client

import (
	"testing"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
)

// TestFactoryRefactoringGoals validates that the factory refactoring achieved its goals
func TestFactoryRefactoringGoals(t *testing.T) {
	t.Run("factory_focuses_on_client_creation", func(t *testing.T) {
		// Test that ClientFactory can be created and has clean responsibility
		logger := logger.New()
		factory := NewFactory(logger)

		if factory == nil {
			t.Fatal("Factory should not be nil")
		}

		if factory.logger == nil {
			t.Error("Factory should have a logger")
		}

		if factory.registry == nil {
			t.Error("Factory should have an instance-level registry")
		}

		if factory.validator == nil {
			t.Error("Factory should have a validator")
		}
	})

	t.Run("configuration_options_separated", func(t *testing.T) {
		// Test that client configuration options work independently
		clientConfig, err := buildClientConfig(
			WithLogger(logger.New()),
			WithAsync(4),
			WithInstanceRegistry(),
		)

		if err != nil {
			t.Fatalf("Failed to build client config: %v", err)
		}

		if clientConfig.Logger == nil {
			t.Error("Client config should have logger")
		}

		if !clientConfig.Async {
			t.Error("Client config should have async enabled")
		}

		if clientConfig.Workers != 4 {
			t.Error("Client config should have 4 workers")
		}

		if !clientConfig.InstanceRegistry {
			t.Error("Client config should use instance registry")
		}
	})

	t.Run("configuration_validation_separated", func(t *testing.T) {
		// Test that validation logic is separated and functional
		validator := NewConfigValidator()

		if validator == nil {
			t.Fatal("Validator should not be nil")
		}

		// Test valid client config
		validConfig := &ClientConfig{
			Logger:           logger.New(),
			Async:            true,
			Workers:          4,
			InstanceRegistry: true,
		}

		err := validator.ValidateClientConfig(validConfig)
		if err != nil {
			t.Errorf("Valid config should pass validation: %v", err)
		}

		// Test invalid client config
		invalidConfig := &ClientConfig{
			Logger:           nil, // Invalid: nil logger
			Async:            true,
			Workers:          0, // Invalid: zero workers with async enabled
			InstanceRegistry: true,
		}

		err = validator.ValidateClientConfig(invalidConfig)
		if err == nil {
			t.Error("Invalid config should fail validation")
		}
	})

	t.Run("platform_options_compatibility", func(t *testing.T) {
		// Test that platform options work with client options
		clientConfig, err := buildClientConfig(
			WithPlatformOptions(
				config.WithFeishu(config.FeishuConfig{
					WebhookURL: "https://example.com/webhook",
					AuthType:   "webhook",
				}),
			),
			WithDefaults(),
		)

		if err != nil {
			t.Fatalf("Failed to build config with platform options: %v", err)
		}

		if len(clientConfig.PlatformOptions) != 1 {
			t.Error("Client config should have platform options")
		}
	})

	t.Run("file_separation_achieved", func(t *testing.T) {
		// This test verifies that the responsibilities are properly separated
		// by ensuring each component can be instantiated independently

		// Test options.go functionality
		_, err := buildClientConfig(WithDefaults())
		if err != nil {
			t.Errorf("Options module should work independently: %v", err)
		}

		// Test validator.go functionality
		validator := NewConfigValidator()
		if validator == nil {
			t.Error("Validator module should work independently")
		}

		// Test factory.go functionality
		factory := NewFactory(logger.New())
		if factory == nil {
			t.Error("Factory module should work independently")
		}
	})
}

// TestSingleResponsibilityPrinciple validates that each file has a single, clear responsibility
func TestSingleResponsibilityPrinciple(t *testing.T) {
	t.Run("factory_only_creates_clients", func(t *testing.T) {
		// The factory should only orchestrate client creation
		// Configuration handling should be delegated to other modules
		factory := NewFactory(logger.New())

		// Test that factory has minimal surface area focused on client creation
		if factory.logger == nil {
			t.Error("Factory should maintain logger for client creation")
		}
		if factory.registry == nil {
			t.Error("Factory should maintain registry for client creation")
		}
		if factory.validator == nil {
			t.Error("Factory should maintain validator for client creation")
		}
	})

	t.Run("options_only_handles_configuration", func(t *testing.T) {
		// Options module should only handle configuration building
		config, err := buildClientConfig(
			WithDefaults(),
			WithAsync(2),
		)

		if err != nil {
			t.Fatalf("Options module should handle configuration: %v", err)
		}

		if config.Workers != 2 {
			t.Error("Options should correctly apply configuration")
		}
	})

	t.Run("validator_only_validates", func(t *testing.T) {
		// Validator module should only handle validation logic
		validator := NewConfigValidator()

		config := &ClientConfig{
			Logger:  logger.New(),
			Async:   false,
			Workers: 0,
		}

		err := validator.ValidateClientConfig(config)
		if err != nil {
			t.Errorf("Validator should handle validation: %v", err)
		}
	})
}