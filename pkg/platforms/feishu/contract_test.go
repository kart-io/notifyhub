// Package feishu provides contract tests for Feishu platform implementation
// This file validates that the Feishu platform implementation conforms to the Platform interface contract
package feishu

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
	"github.com/kart-io/notifyhub/pkg/platform"
)

// TestFeishuPlatformContract validates that Feishu platform implements the Platform interface correctly
func TestFeishuPlatformContract(t *testing.T) {
	// Create test message
	testMessage := message.New()
	testMessage.Title = "Feishu Contract Test"
	testMessage.Body = "This is a contract test message for Feishu platform"
	testMessage.Format = message.FormatText
	testMessage.Priority = message.PriorityNormal

	// Define contract test for Feishu platform
	contractTest := platform.PlatformContractTest{
		PlatformName: "feishu",
		CreatePlatform: func() (platform.Platform, error) {
			config := &config.FeishuConfig{
				WebhookURL: "https://httpbin.org/post", // Use httpbin for testing
				Secret:     "test-secret",
				Keywords:   []string{"test"},
				Timeout:    30 * time.Second,
			}
			logger := logger.NewConsoleLogger("debug")
			return NewFeishuPlatform(config, logger)
		},
		ValidTargets: []target.Target{
			{Type: "feishu", Value: "test-webhook-1", Platform: "feishu"},
			{Type: "webhook", Value: "test-webhook-2", Platform: "feishu"},
			{Type: "feishu", Value: "another-webhook", Platform: "auto"},
			{Type: "webhook", Value: "generic-webhook", Platform: "auto"},
		},
		InvalidTargets: []target.Target{
			{Type: "email", Value: "test@example.com", Platform: "feishu"}, // Wrong type for Feishu
			{Type: "sms", Value: "123456789", Platform: "feishu"},          // Wrong type for Feishu
			{Type: "feishu", Value: "", Platform: "feishu"},                // Empty value
			{Type: "", Value: "test-webhook", Platform: "feishu"},          // Empty type
			{Type: "user", Value: "user123", Platform: "feishu"},           // Unsupported type
		},
		TestMessage: testMessage,
	}

	// Run the contract tests
	platform.RunPlatformContractTests(t, contractTest)
}

// TestFeishuPlatformCapabilities validates specific Feishu capabilities
func TestFeishuPlatformCapabilities(t *testing.T) {
	config := &config.FeishuConfig{
		WebhookURL: "https://example.com/webhook",
	}
	logger := logger.NewConsoleLogger("debug")

	platform, err := NewFeishuPlatform(config, logger)
	if err != nil {
		t.Fatalf("Failed to create Feishu platform: %v", err)
	}
	defer platform.Close()

	capabilities := platform.GetCapabilities()

	// Test Feishu-specific capabilities
	expectedName := "feishu"
	if capabilities.Name != expectedName {
		t.Errorf("Expected platform name %s, got %s", expectedName, capabilities.Name)
	}

	// Verify supported target types include feishu and webhook
	supportedTypes := capabilities.SupportedTargetTypes
	if !contains(supportedTypes, "feishu") {
		t.Error("Feishu platform must support 'feishu' target type")
	}
	if !contains(supportedTypes, "webhook") {
		t.Error("Feishu platform must support 'webhook' target type")
	}

	// Verify supported formats include text and expected Feishu formats
	supportedFormats := capabilities.SupportedFormats
	if !contains(supportedFormats, "text") {
		t.Error("Feishu platform must support 'text' format")
	}

	// Feishu should support reasonable message sizes
	if capabilities.MaxMessageSize < 1000 {
		t.Errorf("Feishu platform message size limit seems too small: %d", capabilities.MaxMessageSize)
	}

	if capabilities.MaxMessageSize > 100000 { // 100KB seems reasonable upper bound
		t.Logf("WARNING: Feishu platform message size limit is very large: %d", capabilities.MaxMessageSize)
	}

	t.Logf("✓ Feishu capabilities validated: %+v", capabilities)
}

// TestFeishuPlatformValidation validates Feishu-specific target validation rules
func TestFeishuPlatformValidation(t *testing.T) {
	config := &config.FeishuConfig{
		WebhookURL: "https://example.com/webhook",
	}
	logger := logger.NewConsoleLogger("debug")

	platform, err := NewFeishuPlatform(config, logger)
	if err != nil {
		t.Fatalf("Failed to create Feishu platform: %v", err)
	}
	defer platform.Close()

	// Test various target validation scenarios
	validTargets := []target.Target{
		{Type: "feishu", Value: "webhook-id-123"},
		{Type: "webhook", Value: "another-webhook"},
		{Type: "feishu", Value: "a", Platform: "feishu"}, // Minimal valid value
	}

	for _, validTarget := range validTargets {
		err := platform.ValidateTarget(validTarget)
		if err != nil {
			t.Errorf("Expected valid target to pass validation: %v, error: %v", validTarget, err)
		}
	}

	invalidTargets := []target.Target{
		{Type: "email", Value: "test@example.com"},     // Wrong type
		{Type: "sms", Value: "123456789"},              // Wrong type
		{Type: "user", Value: "user123"},               // Unsupported type
		{Type: "feishu", Value: ""},                    // Empty value
		{Type: "", Value: "webhook"},                   // Empty type
	}

	for _, invalidTarget := range invalidTargets {
		err := platform.ValidateTarget(invalidTarget)
		if err == nil {
			t.Errorf("Expected invalid target to fail validation: %v", invalidTarget)
		}
	}

	t.Log("✓ Feishu target validation working correctly")
}

// TestFeishuPlatformErrorHandling validates Feishu-specific error scenarios
func TestFeishuPlatformErrorHandling(t *testing.T) {
	// Test platform creation with invalid configuration
	invalidConfigs := []*config.FeishuConfig{
		{WebhookURL: ""},                               // Missing webhook URL
		{WebhookURL: "invalid-url"},                    // Invalid URL format
		{WebhookURL: "https://example.com", Timeout: -1 * time.Second}, // Invalid timeout
	}

	logger := logger.NewConsoleLogger("debug")

	for i, invalidConfig := range invalidConfigs {
		platform, err := NewFeishuPlatform(invalidConfig, logger)
		if err == nil {
			platform.Close()
			t.Errorf("Expected error for invalid config %d: %+v", i, invalidConfig)
		}
	}

	// Test valid platform with error scenarios
	validConfig := &config.FeishuConfig{
		WebhookURL: "https://httpbin.org/status/500", // Will return 500 error
		Timeout:    5 * time.Second,
	}

	platform, err := NewFeishuPlatform(validConfig, logger)
	if err != nil {
		t.Fatalf("Failed to create platform with valid config: %v", err)
	}
	defer platform.Close()

	// Test error handling in Send method with server error
	testMessage := message.New()
	testMessage.Title = "Error Test"
	testMessage.Body = "Testing error handling"

	targets := []target.Target{
		{Type: "feishu", Value: "test-webhook"},
	}

	// This should handle the 500 error gracefully
	results, err := platform.Send(context.Background(), testMessage, targets)

	// Either Send returns an error, or results show failure - both are acceptable
	if err == nil {
		if len(results) == 0 {
			t.Error("Expected results when Send doesn't return error")
		} else if results[0].Success {
			t.Log("WARNING: Expected failure result when server returns 500")
		}
	}

	t.Log("✓ Feishu error handling validated")
}

// TestFeishuPlatformKeywordHandling validates keyword requirement processing
func TestFeishuPlatformKeywordHandling(t *testing.T) {
	// Test platform with keyword requirements
	config := &config.FeishuConfig{
		WebhookURL: "https://httpbin.org/post",
		Keywords:   []string{"urgent", "alert"},
		Timeout:    10 * time.Second,
	}
	logger := logger.NewConsoleLogger("debug")

	platform, err := NewFeishuPlatform(config, logger)
	if err != nil {
		t.Fatalf("Failed to create Feishu platform: %v", err)
	}
	defer platform.Close()

	// Test message with keywords
	messageWithKeywords := message.New()
	messageWithKeywords.Title = "Urgent Alert"
	messageWithKeywords.Body = "This is an urgent alert message"

	targets := []target.Target{
		{Type: "feishu", Value: "test-webhook"},
	}

	// This should work since message contains required keywords
	_, err = platform.Send(context.Background(), messageWithKeywords, targets)
	if err != nil {
		t.Errorf("Message with keywords should send successfully: %v", err)
	}

	// Test message without keywords
	messageWithoutKeywords := message.New()
	messageWithoutKeywords.Title = "Normal Message"
	messageWithoutKeywords.Body = "This is a normal message"

	// This should be handled gracefully (keyword processing should add required keywords)
	_, err = platform.Send(context.Background(), messageWithoutKeywords, targets)
	if err != nil {
		t.Errorf("Message without keywords should be handled gracefully: %v", err)
	}

	t.Log("✓ Feishu keyword handling validated")
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

