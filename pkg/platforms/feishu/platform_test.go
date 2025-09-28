package feishu

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

func TestNewFeishuPlatform(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.FeishuConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.FeishuConfig{
				WebhookURL: "https://example.com/webhook",
				Secret:     "test-secret",
				Keywords:   []string{"test"},
				Timeout:    30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "missing webhook URL",
			config: &config.FeishuConfig{
				Secret:   "test-secret",
				Keywords: []string{"test"},
				Timeout:  30 * time.Second,
			},
			expectError: true,
		},
		{
			name: "default timeout",
			config: &config.FeishuConfig{
				WebhookURL: "https://example.com/webhook",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logger.NewConsoleLogger("debug")
			platform, err := NewFeishuPlatform(tt.config, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if platform == nil {
				t.Errorf("expected platform but got nil")
				return
			}

			// Test platform interface methods
			if platform.Name() != "feishu" {
				t.Errorf("expected platform name 'feishu', got '%s'", platform.Name())
			}

			capabilities := platform.GetCapabilities()
			if capabilities.Name != "feishu" {
				t.Errorf("expected capabilities name 'feishu', got '%s'", capabilities.Name)
			}
		})
	}
}

func TestFeishuPlatform_ValidateTarget(t *testing.T) {
	config := &config.FeishuConfig{
		WebhookURL: "https://example.com/webhook",
	}
	logger := logger.NewConsoleLogger("debug")
	platform, err := NewFeishuPlatform(config, logger)
	if err != nil {
		t.Fatalf("failed to create platform: %v", err)
	}

	tests := []struct {
		name        string
		target      target.Target
		expectError bool
	}{
		{
			name:        "valid feishu target",
			target:      target.Target{Type: "feishu", Value: "test-value"},
			expectError: false,
		},
		{
			name:        "valid webhook target",
			target:      target.Target{Type: "webhook", Value: "test-value"},
			expectError: false,
		},
		{
			name:        "invalid target type",
			target:      target.Target{Type: "email", Value: "test@example.com"},
			expectError: true,
		},
		{
			name:        "empty target value",
			target:      target.Target{Type: "feishu", Value: ""},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := platform.ValidateTarget(tt.target)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestFeishuPlatform_IsHealthy(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.FeishuConfig
		expectError bool
	}{
		{
			name: "healthy with webhook URL",
			config: &config.FeishuConfig{
				WebhookURL: "https://example.com/webhook",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logger.NewConsoleLogger("debug")
			platform, err := NewFeishuPlatform(tt.config, logger)
			if err != nil {
				t.Fatalf("failed to create platform: %v", err)
			}

			ctx := context.Background()
			err = platform.IsHealthy(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestFeishuPlatform_Send(t *testing.T) {
	config := &config.FeishuConfig{
		WebhookURL: "https://httpbin.org/post", // Use httpbin for testing
	}
	logger := logger.NewConsoleLogger("debug")
	platform, err := NewFeishuPlatform(config, logger)
	if err != nil {
		t.Fatalf("failed to create platform: %v", err)
	}

	tests := []struct {
		name            string
		message         *message.Message
		targets         []target.Target
		expectSuccess   int
		expectFailure   int
	}{
		{
			name: "valid message to feishu target",
			message: &message.Message{
				ID:    "test-123",
				Title: "Test Message",
				Body:  "This is a test message",
			},
			targets: []target.Target{
				{Type: "feishu", Value: "test-webhook"},
			},
			expectSuccess: 1,
			expectFailure: 0,
		},
		{
			name: "message to non-feishu target",
			message: &message.Message{
				ID:    "test-456",
				Title: "Test Message",
				Body:  "This is a test message",
			},
			targets: []target.Target{
				{Type: "email", Value: "test@example.com"},
			},
			expectSuccess: 0,
			expectFailure: 1,
		},
		{
			name: "mixed targets",
			message: &message.Message{
				ID:    "test-789",
				Title: "Test Message",
				Body:  "This is a test message",
			},
			targets: []target.Target{
				{Type: "feishu", Value: "test-webhook"},
				{Type: "email", Value: "test@example.com"},
				{Type: "webhook", Value: "test-webhook-2"},
			},
			expectSuccess: 2,
			expectFailure: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := platform.Send(ctx, tt.message, tt.targets)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(results) != len(tt.targets) {
				t.Errorf("expected %d results, got %d", len(tt.targets), len(results))
				return
			}

			successCount := 0
			failureCount := 0
			for _, result := range results {
				if result.Success {
					successCount++
				} else {
					failureCount++
				}
			}

			if successCount != tt.expectSuccess {
				t.Errorf("expected %d successes, got %d", tt.expectSuccess, successCount)
			}

			if failureCount != tt.expectFailure {
				t.Errorf("expected %d failures, got %d", tt.expectFailure, failureCount)
			}
		})
	}
}