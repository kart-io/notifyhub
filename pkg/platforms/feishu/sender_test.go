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

func TestNewFeishuPlatform_WithSignature(t *testing.T) {
	feishuConfig := &config.FeishuConfig{
		WebhookURL: "https://httpbin.org/post", // Use httpbin for testing instead of real webhook
		Secret:     "gQURr67BPOsTZlI7jBn0Jh",
		Timeout:    30 * time.Second,
	}

	// Create a debug logger to see what's happening
	debugLogger := logger.New().LogMode(logger.Debug)

	sender, err := NewFeishuPlatform(feishuConfig, debugLogger)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sender == nil {
		t.Fatal("Expected sender to be created")
	}

	results, err := sender.Send(context.Background(), &message.Message{
		Title: "Test Title",
		Body:  "Test Body",
	}, []target.Target{
		{Type: "feishu", Value: "feishu-target"},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("Expected successful send, got error: %s", results[0].Error)
	}

	t.Logf("Send result: Success=%v, MessageID=%s", results[0].Success, results[0].MessageID)
}

func TestNewFeishuPlatform_WithKeywords(t *testing.T) {
	feishuConfig := &config.FeishuConfig{
		WebhookURL: "https://httpbin.org/post", // Use httpbin for testing instead of real webhook
		Keywords:   []string{"notification"},
		Timeout:    30 * time.Second,
	}

	// Create a debug logger to see what's happening
	debugLogger := logger.New().LogMode(logger.Debug)

	sender, err := NewFeishuPlatform(feishuConfig, debugLogger)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sender == nil {
		t.Fatal("Expected sender to be created")
	}

	results, err := sender.Send(context.Background(), &message.Message{
		Title: "Test Title",
		Body:  "Test Body",
	}, []target.Target{
		{Type: "feishu", Value: "feishu-target"},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("Expected successful send, got error: %s", results[0].Error)
	}

	t.Logf("Send result: Success=%v, MessageID=%s", results[0].Success, results[0].MessageID)
}

// TestFeishuSender_ThreeSecurityModes tests the three security modes
func TestFeishuSender_ThreeSecurityModes(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.FeishuConfig
		message     *message.Message
		description string
	}{
		{
			name: "NoSecurity",
			config: &config.FeishuConfig{
				WebhookURL: "https://httpbin.org/post",
				// No secret, no keywords
				Timeout: 30 * time.Second,
			},
			message: &message.Message{
				Title: "Test Message",
				Body:  "This is a test message without any security requirements",
			},
			description: "情况1: 无需配置签名与自定义关键词",
		},
		{
			name: "WithSignature",
			config: &config.FeishuConfig{
				WebhookURL: "https://httpbin.org/post",
				Secret:     "test-secret-key",
				Timeout:    30 * time.Second,
			},
			message: &message.Message{
				Title: "Test Message",
				Body:  "This is a test message with signature verification",
			},
			description: "情况2: 配置签名验证",
		},
		{
			name: "WithKeywords",
			config: &config.FeishuConfig{
				WebhookURL: "https://httpbin.org/post",
				Keywords:   []string{"报警", "通知", "测试"},
				Timeout:    30 * time.Second,
			},
			message: &message.Message{
				Title: "Test Message",
				Body:  "This is a test message without any keywords",
			},
			description: "情况3: 自定义关键词 - 消息会自动添加关键词",
		},
		{
			name: "WithKeywordsMatched",
			config: &config.FeishuConfig{
				WebhookURL: "https://httpbin.org/post",
				Keywords:   []string{"报警", "通知", "测试"},
				Timeout:    30 * time.Second,
			},
			message: &message.Message{
				Title: "报警通知",
				Body:  "This is a test alert message",
			},
			description: "情况3: 自定义关键词 - 消息已包含关键词",
		},
		{
			name: "WithSignatureAndKeywords",
			config: &config.FeishuConfig{
				WebhookURL: "https://httpbin.org/post",
				Secret:     "test-secret-key",
				Keywords:   []string{"报警", "通知", "测试"},
				Timeout:    30 * time.Second,
			},
			message: &message.Message{
				Title: "Test Message",
				Body:  "This message has both signature and keyword requirements",
			},
			description: "情况2+3: 同时配置签名和关键词",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debugLogger := logger.New().LogMode(logger.Debug)
			sender, err := NewFeishuPlatform(tt.config, debugLogger)
			if err != nil {
				t.Fatalf("Failed to create feishu platform: %v", err)
			}

			results, err := sender.Send(context.Background(), tt.message, []target.Target{
				{Type: "feishu", Value: "test-target"},
			})

			if err != nil {
				t.Errorf("Send failed: %v", err)
				return
			}

			if len(results) != 1 {
				t.Errorf("Expected 1 result, got %d", len(results))
				return
			}

			if !results[0].Success {
				t.Errorf("Send not successful: %s", results[0].Error)
				return
			}

			t.Logf("✅ %s - %s: Success=%v, MessageID=%s", tt.name, tt.description, results[0].Success, results[0].MessageID)
		})
	}
}

func TestNewFeishuPlatform_RequiredFields(t *testing.T) {
	feishuConfig := &config.FeishuConfig{
		// Missing WebhookURL
		Secret:  "test-secret",
		Timeout: 30 * time.Second,
	}

	_, err := NewFeishuPlatform(feishuConfig, logger.Discard)
	if err == nil {
		t.Fatal("Expected error for missing webhook URL")
	}
}

func TestNewFeishuSender(t *testing.T) {
	config := map[string]interface{}{
		"webhook_url": "https://httpbin.org/post", // Use httpbin for testing
		"secret":      "test-secret",
		"timeout":     "30s",
	}

	sender, err := NewFeishuSender(config, logger.Discard)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sender == nil {
		t.Fatal("Expected sender to be created")
	}
}

func TestNewFeishuSender_RequiredFields(t *testing.T) {
	config := map[string]interface{}{
		"secret": "test-secret",
		// Missing webhook_url
	}

	_, err := NewFeishuSender(config, logger.Discard)
	if err == nil {
		t.Fatal("Expected error for missing webhook_url")
	}
}

func TestFeishuSender_GetCapabilities(t *testing.T) {
	sender := &FeishuSender{
		webhookURL: "https://test.com",
		logger:     logger.Discard,
	}

	caps := sender.GetCapabilities()

	if caps.Name != "feishu" {
		t.Errorf("Expected name 'feishu', got %s", caps.Name)
	}

	expectedTargetTypes := []string{"feishu", "webhook"}
	if len(caps.SupportedTargetTypes) != len(expectedTargetTypes) {
		t.Errorf("Expected %d supported target types, got %d", len(expectedTargetTypes), len(caps.SupportedTargetTypes))
	}

	expectedFormats := []string{"text", "markdown", "card", "rich_text"}
	if len(caps.SupportedFormats) != len(expectedFormats) {
		t.Errorf("Expected %d supported formats, got %d", len(expectedFormats), len(caps.SupportedFormats))
	}

	if caps.MaxMessageSize != 4000 {
		t.Errorf("Expected max message size 4000, got %d", caps.MaxMessageSize)
	}
}

func TestFeishuSender_filterFeishuTargets(t *testing.T) {
	sender := &FeishuSender{
		logger: logger.Discard,
	}

	targets := []target.Target{
		{Type: "email", Value: "test@example.com"},
		{Type: "feishu", Value: "feishu-target"},
		{Type: "webhook", Value: "webhook-target"},
		{Type: "slack", Value: "slack-target"},
	}

	filtered := sender.filterFeishuTargets(targets)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 feishu targets, got %d", len(filtered))
	}

	// Check that only feishu and webhook targets are included
	for _, target := range filtered {
		if target.Type != "feishu" && target.Type != "webhook" {
			t.Errorf("Unexpected target type: %s", target.Type)
		}
	}
}

func TestFeishuSender_buildFeishuMessage(t *testing.T) {
	sender := &FeishuSender{
		logger: logger.Discard,
	}

	t.Run("TextMessage", func(t *testing.T) {
		msg := &message.Message{
			Title:  "Test Title",
			Body:   "Test Body",
			Format: message.FormatText,
		}

		feishuMsg, err := sender.buildFeishuMessage(msg)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if feishuMsg.MsgType != "text" {
			t.Errorf("Expected msg_type 'text', got %s", feishuMsg.MsgType)
		}

		content, ok := feishuMsg.Content.(*FeishuTextContent)
		if !ok {
			t.Fatal("Expected FeishuTextContent")
		}

		expectedText := "Test Title\n\nTest Body"
		if content.Text != expectedText {
			t.Errorf("Expected text %q, got %q", expectedText, content.Text)
		}
	})

	t.Run("MarkdownMessage", func(t *testing.T) {
		msg := &message.Message{
			Title:  "Test Title",
			Body:   "**Test Body**",
			Format: message.FormatMarkdown,
		}

		feishuMsg, err := sender.buildFeishuMessage(msg)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if feishuMsg.MsgType != "post" {
			t.Errorf("Expected msg_type 'post', got %s", feishuMsg.MsgType)
		}
	})

	t.Run("CardMessage", func(t *testing.T) {
		cardData := map[string]interface{}{
			"elements": []interface{}{
				map[string]interface{}{
					"tag":  "div",
					"text": map[string]interface{}{"tag": "lark_md", "content": "Test card"},
				},
			},
		}

		msg := &message.Message{
			Title:        "Test Card",
			PlatformData: map[string]interface{}{"feishu_card": cardData},
		}

		feishuMsg, err := sender.buildFeishuMessage(msg)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if feishuMsg.MsgType != "interactive" {
			t.Errorf("Expected msg_type 'interactive', got %s", feishuMsg.MsgType)
		}

		// We can't directly compare maps, so just check the type
		if feishuMsg.Content == nil {
			t.Error("Expected content to be set")
		}
	})
}

// TestFeishuSender_SecurityModeDetection tests security mode detection
func TestFeishuSender_SecurityModeDetection(t *testing.T) {
	tests := []struct {
		name         string
		secret       string
		keywords     []string
		expectedMode string
	}{
		{
			name:         "NoSecurity",
			secret:       "",
			keywords:     []string{},
			expectedMode: "no_security",
		},
		{
			name:         "SignatureOnly",
			secret:       "test-secret",
			keywords:     []string{},
			expectedMode: "signature_only",
		},
		{
			name:         "KeywordsOnly",
			secret:       "",
			keywords:     []string{"报警", "通知"},
			expectedMode: "keywords_only",
		},
		{
			name:         "SignatureAndKeywords",
			secret:       "test-secret",
			keywords:     []string{"报警", "通知"},
			expectedMode: "signature_and_keywords",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &FeishuSender{
				secret:   tt.secret,
				keywords: tt.keywords,
				logger:   logger.Discard,
			}

			mode := sender.determineSecurityMode()
			if mode != tt.expectedMode {
				t.Errorf("Expected security mode %s, got %s", tt.expectedMode, mode)
			}
		})
	}
}

func TestFeishuSender_Close(t *testing.T) {
	sender := &FeishuSender{
		logger: logger.Discard,
	}

	err := sender.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}
}

func TestFeishuSender_ValidateTarget(t *testing.T) {
	sender := &FeishuSender{
		logger: logger.Discard,
	}

	t.Run("ValidFeishuTarget", func(t *testing.T) {
		target := target.Target{Type: "feishu", Value: "test-target"}
		err := sender.ValidateTarget(target)
		if err != nil {
			t.Errorf("Expected no error for valid feishu target, got %v", err)
		}
	})

	t.Run("ValidWebhookTarget", func(t *testing.T) {
		target := target.Target{Type: "webhook", Value: "test-webhook"}
		err := sender.ValidateTarget(target)
		if err != nil {
			t.Errorf("Expected no error for valid webhook target, got %v", err)
		}
	})

	t.Run("InvalidTargetType", func(t *testing.T) {
		target := target.Target{Type: "email", Value: "test@example.com"}
		err := sender.ValidateTarget(target)
		if err == nil {
			t.Error("Expected error for invalid target type")
		}
	})

	t.Run("EmptyTargetValue", func(t *testing.T) {
		target := target.Target{Type: "feishu", Value: ""}
		err := sender.ValidateTarget(target)
		if err == nil {
			t.Error("Expected error for empty target value")
		}
	})
}

func TestFeishuSender_IsHealthy(t *testing.T) {
	t.Run("HealthyWithWebhookURL", func(t *testing.T) {
		sender := &FeishuSender{
			webhookURL: "https://test.com/webhook",
			logger:     logger.Discard,
		}

		ctx := context.Background()
		err := sender.IsHealthy(ctx)
		if err != nil {
			t.Errorf("Expected no error for healthy sender, got %v", err)
		}
	})

	t.Run("UnhealthyWithoutWebhookURL", func(t *testing.T) {
		sender := &FeishuSender{
			webhookURL: "",
			logger:     logger.Discard,
		}

		ctx := context.Background()
		err := sender.IsHealthy(ctx)
		if err == nil {
			t.Error("Expected error for sender without webhook URL")
		}
	})
}
