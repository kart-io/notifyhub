//go:build integration
// +build integration

package feishu

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestFeishuIntegration tests actual feishu webhook sending
// Run with: go test -tags=integration -run TestFeishuIntegration
// Requires environment variables:
// - FEISHU_WEBHOOK_URL: your feishu bot webhook URL
// - FEISHU_SECRET: your feishu bot secret (optional)
func TestFeishuIntegration(t *testing.T) {
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	if webhookURL == "" {
		t.Skip("FEISHU_WEBHOOK_URL not set, skipping integration test")
	}

	secret := os.Getenv("FEISHU_SECRET") // Optional

	t.Logf("Testing with webhook URL: %s", webhookURL)
	t.Logf("Using secret: %v", secret != "")

	// Test the corrected signature algorithm
	if secret != "" {
		t.Logf("Testing with signature verification enabled using corrected algorithm")
	}

	feishuConfig := &config.FeishuConfig{
		WebhookURL: webhookURL,
		Secret:     secret,
		Timeout:    30 * time.Second,
	}

	sender, err := NewFeishuPlatform(feishuConfig, logger.New().LogMode(logger.Debug))
	if err != nil {
		t.Fatalf("Failed to create feishu platform: %v", err)
	}

	// Test different message types
	testCases := []struct {
		name string
		msg  *message.Message
	}{
		{
			name: "Simple Text Message",
			msg: &message.Message{
				Title:  "Integration Test",
				Body:   "This is a test message from NotifyHub Feishu integration test",
				Format: message.FormatText,
			},
		},
		{
			name: "Markdown Message",
			msg: &message.Message{
				Title:  "Markdown Test",
				Body:   "**Bold text** and *italic text*\n\n- List item 1\n- List item 2",
				Format: message.FormatMarkdown,
			},
		},
		{
			name: "Card Message",
			msg: &message.Message{
				Title: "Card Test",
				PlatformData: map[string]interface{}{
					"feishu_card": map[string]interface{}{
						"elements": []interface{}{
							map[string]interface{}{
								"tag": "div",
								"text": map[string]interface{}{
									"tag":     "lark_md",
									"content": "🎉 **NotifyHub Integration Test**\n\nThis is a test card message",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := sender.Send(context.Background(), tc.msg, []target.Target{
				{Type: "feishu", Value: "integration-test"},
			})

			if err != nil {
				t.Errorf("Send failed: %v", err)
				return
			}

			if len(results) != 1 {
				t.Errorf("Expected 1 result, got %d", len(results))
				return
			}

			result := results[0]
			if !result.Success {
				t.Errorf("Send not successful: %s", result.Error)
				return
			}

			t.Logf("✅ %s sent successfully: MessageID=%s", tc.name, result.MessageID)
		})
	}
}

// TestFeishuHealthCheck tests health check against real webhook
func TestFeishuHealthCheck(t *testing.T) {
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	if webhookURL == "" {
		t.Skip("FEISHU_WEBHOOK_URL not set, skipping health check test")
	}

	feishuConfig := &config.FeishuConfig{
		WebhookURL: webhookURL,
		Timeout:    10 * time.Second,
	}

	sender, err := NewFeishuPlatform(feishuConfig, logger.Discard)
	if err != nil {
		t.Fatalf("Failed to create feishu platform: %v", err)
	}

	err = sender.IsHealthy(context.Background())
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	} else {
		t.Log("✅ Health check passed")
	}
}
