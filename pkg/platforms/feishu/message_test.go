package feishu

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

func TestNewMessageBuilder(t *testing.T) {
	config := &config.FeishuConfig{
		WebhookURL: "https://example.com/webhook",
		Secret:     "test-secret",
		Keywords:   []string{"test"},
	}
	logger := logger.NewConsoleLogger("debug")

	builder := NewMessageBuilder(config, logger)

	if builder == nil {
		t.Error("Expected MessageBuilder to be created, got nil")
	}

	if builder.config != config {
		t.Error("Expected config to be set correctly")
	}

	if builder.logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	if builder.validator == nil {
		t.Error("Expected validator to be initialized")
	}
}

func TestMessageBuilder_BuildMessage(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	tests := []struct {
		name           string
		message        *message.Message
		expectedType   string
		expectedFields []string
	}{
		{
			name: "simple text message",
			message: &message.Message{
				ID:     "test-1",
				Title:  "Test Title",
				Body:   "Test Body",
				Format: message.FormatText,
			},
			expectedType:   "text",
			expectedFields: []string{"msg_type", "content"},
		},
		{
			name: "markdown message",
			message: &message.Message{
				ID:     "test-2",
				Title:  "**Bold Title**",
				Body:   "Some *italic* text",
				Format: message.FormatMarkdown,
			},
			expectedType:   "post",
			expectedFields: []string{"msg_type", "content"},
		},
		{
			name: "HTML message",
			message: &message.Message{
				ID:     "test-3",
				Title:  "<b>Bold Title</b>",
				Body:   "Some <i>italic</i> text",
				Format: message.FormatHTML,
			},
			expectedType:   "post",
			expectedFields: []string{"msg_type", "content"},
		},
		{
			name: "message with platform-specific card data",
			message: &message.Message{
				ID:    "test-4",
				Title: "Card Title",
				Body:  "Card Body",
				PlatformData: map[string]interface{}{
					"feishu_card": map[string]interface{}{
						"elements": []interface{}{
							map[string]interface{}{
								"tag":  "div",
								"text": "Card content",
							},
						},
					},
				},
			},
			expectedType:   "interactive",
			expectedFields: []string{"msg_type", "content"},
		},
		{
			name: "message with platform-specific rich text data",
			message: &message.Message{
				ID:    "test-5",
				Title: "Rich Title",
				Body:  "Rich Body",
				PlatformData: map[string]interface{}{
					"feishu_rich_text": map[string]interface{}{
						"post": map[string]interface{}{
							"zh_cn": map[string]interface{}{
								"title": "Custom Title",
								"content": [][]interface{}{
									{
										map[string]interface{}{
											"tag":  "text",
											"text": "Custom content",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedType:   "post",
			expectedFields: []string{"msg_type", "content"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feishuMsg, err := builder.BuildMessage(tt.message)
			if err != nil {
				t.Errorf("BuildMessage() error = %v", err)
				return
			}

			if feishuMsg.MsgType != tt.expectedType {
				t.Errorf("Expected message type %s, got %s", tt.expectedType, feishuMsg.MsgType)
			}

			// Check that the message can be marshaled to JSON
			_, err = json.Marshal(feishuMsg)
			if err != nil {
				t.Errorf("Failed to marshal FeishuMessage to JSON: %v", err)
			}

			// Verify content is not nil
			if feishuMsg.Content == nil {
				t.Error("Expected content to be non-nil")
			}
		})
	}
}

func TestMessageBuilder_BuildTextContent(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	tests := []struct {
		name            string
		title           string
		body            string
		expectedContent string
	}{
		{
			name:            "title and body",
			title:           "Test Title",
			body:            "Test Body",
			expectedContent: "Test Title\n\nTest Body",
		},
		{
			name:            "title only",
			title:           "Test Title",
			body:            "",
			expectedContent: "Test Title",
		},
		{
			name:            "body only",
			title:           "",
			body:            "Test Body",
			expectedContent: "Test Body",
		},
		{
			name:            "empty message",
			title:           "",
			body:            "",
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &message.Message{
				ID:     "test",
				Title:  tt.title,
				Body:   tt.body,
				Format: message.FormatText,
			}

			feishuMsg, err := builder.BuildMessage(msg)
			if err != nil {
				t.Errorf("BuildMessage() error = %v", err)
				return
			}

			textContent, ok := feishuMsg.Content.(*FeishuTextContent)
			if !ok {
				t.Error("Expected FeishuTextContent")
				return
			}

			if textContent.Text != tt.expectedContent {
				t.Errorf("Expected content %q, got %q", tt.expectedContent, textContent.Text)
			}
		})
	}
}

func TestMessageBuilder_BuildRichTextContent(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	msg := &message.Message{
		ID:     "test",
		Title:  "Rich Title",
		Body:   "Rich Body Content",
		Format: message.FormatMarkdown,
	}

	feishuMsg, err := builder.BuildMessage(msg)
	if err != nil {
		t.Errorf("BuildMessage() error = %v", err)
		return
	}

	richContent, ok := feishuMsg.Content.(*FeishuRichTextContent)
	if !ok {
		t.Error("Expected FeishuRichTextContent")
		return
	}

	// Check post structure
	if richContent.Post == nil {
		t.Error("Expected post to be non-nil")
		return
	}

	zhCn, ok := richContent.Post["zh_cn"].(map[string]interface{})
	if !ok {
		t.Error("Expected zh_cn section in post")
		return
	}

	if title, ok := zhCn["title"].(string); !ok || title != "Rich Title" {
		t.Errorf("Expected title 'Rich Title', got %v", zhCn["title"])
	}

	content, ok := zhCn["content"].([][]interface{})
	if !ok || len(content) == 0 {
		t.Error("Expected content array to be non-empty")
		return
	}

	firstLine := content[0]
	if len(firstLine) == 0 {
		t.Error("Expected first line to have content")
		return
	}

	textElement, ok := firstLine[0].(map[string]interface{})
	if !ok {
		t.Error("Expected text element to be a map")
		return
	}

	if tag, ok := textElement["tag"].(string); !ok || tag != "text" {
		t.Errorf("Expected tag 'text', got %v", textElement["tag"])
	}

	if text, ok := textElement["text"].(string); !ok || text != "Rich Body Content" {
		t.Errorf("Expected text 'Rich Body Content', got %v", textElement["text"])
	}
}

func TestMessageBuilder_ExtractMessageText(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	tests := []struct {
		name         string
		feishuMsg    *FeishuMessage
		originalMsg  *message.Message
		expectedText string
	}{
		{
			name: "text message",
			feishuMsg: &FeishuMessage{
				MsgType: "text",
				Content: &FeishuTextContent{
					Text: "Hello World",
				},
			},
			originalMsg: &message.Message{
				Title: "Title",
				Body:  "Body",
			},
			expectedText: "Hello World",
		},
		{
			name: "post message",
			feishuMsg: &FeishuMessage{
				MsgType: "post",
				Content: &FeishuRichTextContent{},
			},
			originalMsg: &message.Message{
				Title: "Test Title",
				Body:  "Test Body",
			},
			expectedText: "Test Title Test Body",
		},
		{
			name: "interactive message",
			feishuMsg: &FeishuMessage{
				MsgType: "interactive",
				Content: map[string]interface{}{"elements": []interface{}{}},
			},
			originalMsg: &message.Message{
				Title: "Card Title",
				Body:  "Card Body",
			},
			expectedText: "Card Title Card Body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractedText := builder.ExtractMessageText(tt.feishuMsg, tt.originalMsg)

			if extractedText != tt.expectedText {
				t.Errorf("Expected extracted text %q, got %q", tt.expectedText, extractedText)
			}
		})
	}
}

func TestMessageBuilder_AddKeywordToMessage(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	tests := []struct {
		name      string
		feishuMsg *FeishuMessage
		keyword   string
		expectErr bool
		checkFunc func(*testing.T, *FeishuMessage)
	}{
		{
			name: "add keyword to text message",
			feishuMsg: &FeishuMessage{
				MsgType: "text",
				Content: &FeishuTextContent{
					Text: "Original text",
				},
			},
			keyword:   "ALERT",
			expectErr: false,
			checkFunc: func(t *testing.T, msg *FeishuMessage) {
				content, ok := msg.Content.(*FeishuTextContent)
				if !ok {
					t.Error("Expected FeishuTextContent")
					return
				}
				if !strings.HasPrefix(content.Text, "ALERT ") {
					t.Errorf("Expected text to start with 'ALERT ', got %s", content.Text)
				}
			},
		},
		{
			name: "add keyword to rich text message",
			feishuMsg: &FeishuMessage{
				MsgType: "post",
				Content: &FeishuRichTextContent{
					Post: map[string]interface{}{
						"zh_cn": map[string]interface{}{
							"title": "Title",
							"content": [][]interface{}{
								{
									map[string]interface{}{
										"tag":  "text",
										"text": "Original content",
									},
								},
							},
						},
					},
				},
			},
			keyword:   "NOTICE",
			expectErr: false,
			checkFunc: func(t *testing.T, msg *FeishuMessage) {
				content, ok := msg.Content.(*FeishuRichTextContent)
				if !ok {
					t.Error("Expected FeishuRichTextContent")
					return
				}
				zhCn := content.Post["zh_cn"].(map[string]interface{})
				contentArray := zhCn["content"].([][]interface{})
				firstLine := contentArray[0]
				textElement := firstLine[0].(map[string]interface{})
				text := textElement["text"].(string)
				if !strings.HasPrefix(text, "NOTICE ") {
					t.Errorf("Expected text to start with 'NOTICE ', got %s", text)
				}
			},
		},
		{
			name: "add keyword to interactive message",
			feishuMsg: &FeishuMessage{
				MsgType: "interactive",
				Content: map[string]interface{}{
					"elements": []interface{}{},
				},
			},
			keyword:   "INFO",
			expectErr: false,
			checkFunc: func(t *testing.T, msg *FeishuMessage) {
				// Interactive messages don't modify content, just log warning
			},
		},
		{
			name: "empty keyword",
			feishuMsg: &FeishuMessage{
				MsgType: "text",
				Content: &FeishuTextContent{Text: "test"},
			},
			keyword:   "",
			expectErr: true,
		},
		{
			name: "whitespace keyword",
			feishuMsg: &FeishuMessage{
				MsgType: "text",
				Content: &FeishuTextContent{Text: "test"},
			},
			keyword:   "   ",
			expectErr: true,
		},
		{
			name: "unsupported message type",
			feishuMsg: &FeishuMessage{
				MsgType: "unknown",
				Content: map[string]interface{}{},
			},
			keyword:   "TEST",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := builder.AddKeywordToMessage(tt.feishuMsg, tt.keyword)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.feishuMsg)
			}
		})
	}
}

func TestMessageBuilder_ValidateMessage(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	tests := []struct {
		name      string
		message   *message.Message
		expectErr bool
	}{
		{
			name: "valid message",
			message: &message.Message{
				ID:     "test-123",
				Title:  "Valid Title",
				Body:   "Valid body content",
				Format: message.FormatText,
			},
			expectErr: false,
		},
		{
			name:      "nil message",
			message:   nil,
			expectErr: true,
		},
		{
			name: "empty ID",
			message: &message.Message{
				ID:     "",
				Title:  "Title",
				Body:   "Body",
				Format: message.FormatText,
			},
			expectErr: true,
		},
		{
			name: "title too long",
			message: &message.Message{
				ID:     "test-123",
				Title:  strings.Repeat("a", MaxTitleLength+1),
				Body:   "Body",
				Format: message.FormatText,
			},
			expectErr: true,
		},
		{
			name: "body too long",
			message: &message.Message{
				ID:     "test-123",
				Title:  "Title",
				Body:   strings.Repeat("a", MaxBodyLength+1),
				Format: message.FormatText,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := builder.ValidateMessage(tt.message)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMessageBuilder_SanitizeContent(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	tests := []struct {
		name     string
		input    string
		expected func(string) bool // Function to check if output is acceptable
	}{
		{
			name:  "normal text",
			input: "Hello World",
			expected: func(output string) bool {
				return output == "Hello World"
			},
		},
		{
			name:  "html characters",
			input: "<div>Hello</div>",
			expected: func(output string) bool {
				return strings.Contains(output, "&lt;") && strings.Contains(output, "&gt;")
			},
		},
		{
			name:  "script tag",
			input: "<script>alert('xss')</script>",
			expected: func(output string) bool {
				return !strings.Contains(output, "<script>") && !strings.Contains(output, "alert")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := builder.SanitizeContent(tt.input)

			if !tt.expected(output) {
				t.Errorf("Sanitization failed for input %q, got %q", tt.input, output)
			}
		})
	}
}

func TestMessageBuilder_EstimateMessageSize(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	tests := []struct {
		name        string
		message     *message.Message
		expectError bool
		minSize     int
	}{
		{
			name: "simple text message",
			message: &message.Message{
				ID:     "test-123",
				Title:  "Hello",
				Body:   "World",
				Format: message.FormatText,
			},
			expectError: false,
			minSize:     50, // Should be at least 50 bytes
		},
		{
			name: "rich text message",
			message: &message.Message{
				ID:     "test-456",
				Title:  "Rich Title",
				Body:   "Rich content with more text",
				Format: message.FormatMarkdown,
			},
			expectError: false,
			minSize:     100, // Rich text should be larger
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := builder.EstimateMessageSize(tt.message)

			if size < tt.minSize {
				t.Errorf("Expected size to be at least %d, got %d", tt.minSize, size)
			}

			if size > MaxMessageSize {
				t.Errorf("Estimated size %d exceeds maximum %d", size, MaxMessageSize)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkMessageBuilder_BuildMessage(b *testing.B) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("error") // Reduce log noise
	builder := NewMessageBuilder(config, logger)

	msg := &message.Message{
		ID:        "benchmark-test",
		Title:     "Benchmark Title",
		Body:      "This is a benchmark test message with some content",
		Format:    message.FormatText,
		CreatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.BuildMessage(msg)
		if err != nil {
			b.Errorf("BuildMessage failed: %v", err)
		}
	}
}

func BenchmarkMessageBuilder_ValidateMessage(b *testing.B) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("error")
	builder := NewMessageBuilder(config, logger)

	msg := &message.Message{
		ID:     "benchmark-test",
		Title:  "Benchmark Title",
		Body:   "This is a benchmark test message with some content for validation",
		Format: message.FormatText,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := builder.ValidateMessage(msg)
		if err != nil {
			b.Errorf("ValidateMessage failed: %v", err)
		}
	}
}

// Test Task 0.2 enhanced features: Card message format support
func TestMessageBuilder_CardMessageFormat(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	// Test high priority message automatically uses card format
	highPriorityMsg := &message.Message{
		ID:       "test-card-1",
		Title:    "High Priority Alert",
		Body:     "This is an urgent message",
		Format:   message.FormatText,
		Priority: 3, // Urgent priority
	}

	feishuMsg, err := builder.BuildMessage(highPriorityMsg)
	if err != nil {
		t.Errorf("BuildMessage() error = %v", err)
		return
	}

	if feishuMsg.MsgType != "interactive" {
		t.Errorf("Expected interactive message type for high priority, got %s", feishuMsg.MsgType)
	}

	// Verify card content structure
	cardContent, ok := feishuMsg.Content.(*FeishuCardContent)
	if !ok {
		t.Error("Expected FeishuCardContent for interactive message")
		return
	}

	if cardContent.Header == nil {
		t.Error("Expected header in card content")
	}

	if len(cardContent.Elements) == 0 {
		t.Error("Expected elements in card content")
	}
}

// Test Task 0.2 enhanced features: Message size validation
func TestMessageBuilder_MessageSizeValidation(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	// Test message within size limit
	normalMsg := &message.Message{
		ID:    "test-size-1",
		Title: "Normal message",
		Body:  "This message is within size limits",
	}

	err := builder.ValidateMessageSize(normalMsg)
	if err != nil {
		t.Errorf("ValidateMessageSize() should not error for normal message: %v", err)
	}

	// Test message exceeding size limit
	largeBody := strings.Repeat("A", MaxMessageSize)
	largeMsg := &message.Message{
		ID:    "test-size-2",
		Title: "Large message",
		Body:  largeBody,
	}

	err = builder.ValidateMessageSize(largeMsg)
	if err == nil {
		t.Error("ValidateMessageSize() should error for oversized message")
	}

	// Test GetMaxMessageSize
	maxSize := builder.GetMaxMessageSize()
	if maxSize != MaxMessageSize {
		t.Errorf("GetMaxMessageSize() = %d, expected %d", maxSize, MaxMessageSize)
	}
}

// Test Task 0.2 enhanced features: Format support checking
func TestMessageBuilder_FormatSupport(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	// Test supported formats
	supportedFormats := []string{"text", "markdown", "html"}
	for _, format := range supportedFormats {
		if !builder.SupportsFormat(format) {
			t.Errorf("SupportsFormat(%s) should return true", format)
		}
	}

	// Test unsupported formats
	unsupportedFormats := []string{"json", "xml", "yaml", "pdf"}
	for _, format := range unsupportedFormats {
		if builder.SupportsFormat(format) {
			t.Errorf("SupportsFormat(%s) should return false", format)
		}
	}

	// Test GetSupportedFormats
	formats := builder.GetSupportedFormats()
	if len(formats) != 3 {
		t.Errorf("GetSupportedFormats() should return 3 formats, got %d", len(formats))
	}

	expectedFormats := map[string]bool{
		"text":     true,
		"markdown": true,
		"html":     true,
	}

	for _, format := range formats {
		if !expectedFormats[format] {
			t.Errorf("Unexpected format in supported list: %s", format)
		}
	}
}

// Test Task 0.2 enhanced features: Smart message type determination
func TestMessageBuilder_MessageTypeDetermination(t *testing.T) {
	config := &config.FeishuConfig{WebhookURL: "https://example.com/webhook"}
	logger := logger.NewConsoleLogger("debug")
	builder := NewMessageBuilder(config, logger)

	tests := []struct {
		name         string
		message      *message.Message
		expectedType string
	}{
		{
			name: "metadata preference text",
			message: &message.Message{
				ID:       "test-1",
				Title:    "Test",
				Body:     "Body",
				Format:   message.FormatMarkdown,
				Priority: 3,
				Metadata: map[string]interface{}{
					"feishu_message_type": "text",
				},
			},
			expectedType: "text",
		},
		{
			name: "metadata preference post",
			message: &message.Message{
				ID:       "test-2",
				Title:    "Test",
				Body:     "Body",
				Format:   message.FormatText,
				Priority: 3,
				Metadata: map[string]interface{}{
					"feishu_message_type": "post",
				},
			},
			expectedType: "post",
		},
		{
			name: "high priority auto card",
			message: &message.Message{
				ID:       "test-3",
				Title:    "Urgent",
				Body:     "Alert",
				Format:   message.FormatText,
				Priority: 2, // High priority
			},
			expectedType: "interactive",
		},
		{
			name: "markdown format auto rich text",
			message: &message.Message{
				ID:       "test-4",
				Title:    "Markdown",
				Body:     "**Bold** text",
				Format:   message.FormatMarkdown,
				Priority: 1,
			},
			expectedType: "post",
		},
		{
			name: "default text format",
			message: &message.Message{
				ID:       "test-5",
				Title:    "Simple",
				Body:     "Message",
				Format:   message.FormatText,
				Priority: 1,
			},
			expectedType: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feishuMsg, err := builder.BuildMessage(tt.message)
			if err != nil {
				t.Errorf("BuildMessage() error = %v", err)
				return
			}

			if feishuMsg.MsgType != tt.expectedType {
				t.Errorf("Expected message type %s, got %s", tt.expectedType, feishuMsg.MsgType)
			}
		})
	}
}