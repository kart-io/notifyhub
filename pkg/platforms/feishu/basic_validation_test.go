// Package feishu - Basic validation tests that don't require network access
package feishu

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestBasicValidation performs basic validation tests without network dependencies
func TestBasicValidation(t *testing.T) {
	// Test configuration creation
	cfg := &config.FeishuConfig{
		WebhookURL: "https://test.example.com/webhook",
		Secret:     "test-secret",
		Keywords:   []string{"test"},
		Timeout:    30 * time.Second,
	}

	// Test platform creation
	logger := &simpleLogger{}
	platform, err := NewFeishuPlatform(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	// Test platform name
	if platform.Name() != "feishu" {
		t.Errorf("Expected platform name 'feishu', got '%s'", platform.Name())
	}

	// Test capabilities
	caps := platform.GetCapabilities()
	if caps.Name != "feishu" {
		t.Errorf("Expected capabilities name 'feishu', got '%s'", caps.Name)
	}

	// Test target validation
	validTarget := target.Target{Type: "webhook", Value: "https://example.com"}
	if err := platform.ValidateTarget(validTarget); err != nil {
		t.Errorf("Valid target validation failed: %v", err)
	}

	invalidTarget := target.Target{Type: "invalid", Value: ""}
	if err := platform.ValidateTarget(invalidTarget); err == nil {
		t.Error("Invalid target should fail validation")
	}

	// Test health check
	if err := platform.IsHealthy(context.Background()); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	t.Log("✅ Basic platform validation passed")
}

// TestComponentCreation validates that all components can be created
func TestComponentCreation(t *testing.T) {
	cfg := &FeishuConfig{
		WebhookURL: "https://test.example.com",
		Secret:     "secret",
		Keywords:   []string{"test"},
		Timeout:    30 * time.Second,
	}

	logger := &simpleLogger{}

	// Test AuthHandler creation
	auth := NewAuthHandler(cfg.Secret, cfg.Keywords)
	if auth == nil {
		t.Error("Failed to create AuthHandler")
	}

	// Test MessageBuilder creation
	builder := NewMessageBuilder(cfg, logger)
	if builder == nil {
		t.Error("Failed to create MessageBuilder")
	}

	// Test MessageValidator creation
	validator := NewMessageValidator(logger)
	if validator == nil {
		t.Error("Failed to create MessageValidator")
	}

	t.Log("✅ All components created successfully")
}

// TestMessageBuilding validates message building functionality
func TestMessageBuilding(t *testing.T) {
	cfg := &FeishuConfig{
		WebhookURL: "https://test.example.com",
		Timeout:    30 * time.Second,
	}

	logger := &simpleLogger{}
	builder := NewMessageBuilder(cfg, logger)

	// Test text message building
	msg := &message.Message{
		ID:     "test-123",
		Title:  "Test Title",
		Body:   "Test Body",
		Format: message.FormatText,
	}

	feishuMsg, err := builder.BuildMessage(msg)
	if err != nil {
		t.Errorf("Failed to build message: %v", err)
	}

	if feishuMsg.MsgType != "text" {
		t.Errorf("Expected text message type, got %s", feishuMsg.MsgType)
	}

	t.Log("✅ Message building validation passed")
}

// TestBackwardCompatibilityConfig validates backward compatibility
func TestBackwardCompatibilityConfig(t *testing.T) {
	configMap := map[string]interface{}{
		"webhook_url": "https://test.example.com/webhook",
		"secret":      "test-secret",
		"keywords":    []string{"test"},
		"timeout":     "30s",
	}

	cfg, err := NewConfigFromMap(configMap)
	if err != nil {
		t.Errorf("Backward compatibility config failed: %v", err)
	}

	if cfg.WebhookURL != "https://test.example.com/webhook" {
		t.Error("Config mapping failed for webhook_url")
	}

	if cfg.Secret != "test-secret" {
		t.Error("Config mapping failed for secret")
	}

	t.Log("✅ Backward compatibility validation passed")
}

// simpleLogger implements logger.Logger for testing
type simpleLogger struct{}

func (l *simpleLogger) Debug(msg string, args ...interface{}) {}
func (l *simpleLogger) Info(msg string, args ...interface{})  {}
func (l *simpleLogger) Warn(msg string, args ...interface{})  {}
func (l *simpleLogger) Error(msg string, args ...interface{}) {}