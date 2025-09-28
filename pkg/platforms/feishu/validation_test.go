package feishu

import (
	"strings"
	"testing"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

func TestNewMessageValidator(t *testing.T) {
	logger := logger.NewConsoleLogger("debug")
	validator := NewMessageValidator(logger)

	if validator == nil {
		t.Error("Expected MessageValidator to be created, got nil")
	}

	if validator.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
}

func TestMessageValidator_ValidateMessage(t *testing.T) {
	logger := logger.NewConsoleLogger("debug")
	validator := NewMessageValidator(logger)

	tests := []struct {
		name      string
		message   *message.Message
		expectErr bool
		errSubstr string
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
			errSubstr: "cannot be nil",
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
			errSubstr: "ID cannot be empty",
		},
		{
			name: "whitespace only ID",
			message: &message.Message{
				ID:     "   ",
				Title:  "Title",
				Body:   "Body",
				Format: message.FormatText,
			},
			expectErr: true,
			errSubstr: "ID cannot be empty",
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
			errSubstr: "title exceeds maximum length",
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
			errSubstr: "body exceeds maximum length",
		},
		{
			name: "message too large",
			message: &message.Message{
				ID:     "test-123",
				Title:  strings.Repeat("a", MaxMessageSize/2),
				Body:   strings.Repeat("b", MaxMessageSize/2+1),
				Format: message.FormatText,
			},
			expectErr: true,
			errSubstr: "exceeds maximum size",
		},
		{
			name: "dangerous script content",
			message: &message.Message{
				ID:     "test-123",
				Title:  "Title",
				Body:   "<script>alert('xss')</script>",
				Format: message.FormatText,
			},
			expectErr: true,
			errSubstr: "dangerous content",
		},
		{
			name: "javascript URL",
			message: &message.Message{
				ID:     "test-123",
				Title:  "Title",
				Body:   "Click here: javascript:alert('xss')",
				Format: message.FormatText,
			},
			expectErr: true,
			errSubstr: "dangerous content",
		},
		{
			name: "event handler",
			message: &message.Message{
				ID:     "test-123",
				Title:  "Title",
				Body:   "<div onclick='alert()'>Click me</div>",
				Format: message.FormatText,
			},
			expectErr: true,
			errSubstr: "dangerous content",
		},
		{
			name: "text format with HTML tags",
			message: &message.Message{
				ID:     "test-123",
				Title:  "<b>Bold Title</b>",
				Body:   "Body",
				Format: message.FormatText,
			},
			expectErr: true,
			errSubstr: "should not contain HTML tags",
		},
		{
			name: "invalid HTML format",
			message: &message.Message{
				ID:     "test-123",
				Title:  "<div>Unclosed tag",
				Body:   "Body",
				Format: message.FormatHTML,
			},
			expectErr: true,
			errSubstr: "invalid HTML format",
		},
		{
			name: "unmatched markdown code blocks",
			message: &message.Message{
				ID:     "test-123",
				Title:  "Title",
				Body:   "```go\ncode block without closing",
				Format: message.FormatMarkdown,
			},
			expectErr: true,
			errSubstr: "unmatched code block syntax",
		},
		{
			name: "unmatched markdown inline code",
			message: &message.Message{
				ID:     "test-123",
				Title:  "Title",
				Body:   "Some `inline code without closing",
				Format: message.FormatMarkdown,
			},
			expectErr: true,
			errSubstr: "unmatched inline code syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateMessage(tt.message)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("Expected error to contain %q, got %q", tt.errSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMessageValidator_ValidateSecurity(t *testing.T) {
	logger := logger.NewConsoleLogger("debug")
	validator := NewMessageValidator(logger)

	tests := []struct {
		name      string
		message   *message.Message
		expectErr bool
	}{
		{
			name: "safe content",
			message: &message.Message{
				ID:    "test-1",
				Title: "Safe Title",
				Body:  "Safe body content",
			},
			expectErr: false,
		},
		{
			name: "script tag",
			message: &message.Message{
				ID:    "test-2",
				Title: "Title",
				Body:  "<script>alert('xss')</script>",
			},
			expectErr: true,
		},
		{
			name: "javascript protocol",
			message: &message.Message{
				ID:    "test-3",
				Title: "Title",
				Body:  "javascript:void(0)",
			},
			expectErr: true,
		},
		{
			name: "event handler",
			message: &message.Message{
				ID:    "test-4",
				Title: "Title",
				Body:  "onclick=alert()",
			},
			expectErr: true,
		},
		{
			name: "high special character ratio",
			message: &message.Message{
				ID:    "test-5",
				Title: "Title",
				Body:  "<>&\"'<>&\"'<>&\"'",
			},
			expectErr: false, // Should not error, just warn
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateSecurity(tt.message)

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

func TestMessageValidator_ValidateFormat(t *testing.T) {
	logger := logger.NewConsoleLogger("debug")
	validator := NewMessageValidator(logger)

	tests := []struct {
		name      string
		message   *message.Message
		expectErr bool
	}{
		{
			name: "valid text format",
			message: &message.Message{
				ID:     "test-1",
				Title:  "Plain title",
				Body:   "Plain body",
				Format: message.FormatText,
			},
			expectErr: false,
		},
		{
			name: "text format with HTML",
			message: &message.Message{
				ID:     "test-2",
				Title:  "<b>Bold</b>",
				Body:   "Plain body",
				Format: message.FormatText,
			},
			expectErr: true,
		},
		{
			name: "valid HTML format",
			message: &message.Message{
				ID:     "test-3",
				Title:  "<b>Bold</b>",
				Body:   "<i>Italic</i>",
				Format: message.FormatHTML,
			},
			expectErr: false,
		},
		{
			name: "invalid HTML format",
			message: &message.Message{
				ID:     "test-4",
				Title:  "<b>Unclosed",
				Body:   "Body",
				Format: message.FormatHTML,
			},
			expectErr: true,
		},
		{
			name: "valid markdown",
			message: &message.Message{
				ID:     "test-5",
				Title:  "**Bold**",
				Body:   "`code` and ```block```",
				Format: message.FormatMarkdown,
			},
			expectErr: false,
		},
		{
			name: "invalid markdown",
			message: &message.Message{
				ID:     "test-6",
				Title:  "Title",
				Body:   "```unclosed code block",
				Format: message.FormatMarkdown,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateFormat(tt.message)

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

func TestMessageValidator_IsValidHTML(t *testing.T) {
	logger := logger.NewConsoleLogger("debug")
	validator := NewMessageValidator(logger)

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "empty content",
			content:  "",
			expected: true,
		},
		{
			name:     "plain text",
			content:  "Hello world",
			expected: true,
		},
		{
			name:     "valid HTML",
			content:  "<b>Bold</b>",
			expected: true,
		},
		{
			name:     "multiple tags",
			content:  "<div><p>Paragraph</p></div>",
			expected: true,
		},
		{
			name:     "unclosed tag",
			content:  "<b>Bold",
			expected: false,
		},
		{
			name:     "extra closing tag",
			content:  "Bold</b>",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isValidHTML(tt.content)
			if result != tt.expected {
				t.Errorf("Expected %v for content %q, got %v", tt.expected, tt.content, result)
			}
		})
	}
}

func TestMessageValidator_ValidateMarkdown(t *testing.T) {
	logger := logger.NewConsoleLogger("debug")
	validator := NewMessageValidator(logger)

	tests := []struct {
		name      string
		message   *message.Message
		expectErr bool
	}{
		{
			name: "valid markdown",
			message: &message.Message{
				Title: "**Bold**",
				Body:  "`inline` and ```\ncode block\n```",
			},
			expectErr: false,
		},
		{
			name: "unmatched code block",
			message: &message.Message{
				Title: "Title",
				Body:  "```\ncode without closing",
			},
			expectErr: true,
		},
		{
			name: "unmatched inline code",
			message: &message.Message{
				Title: "Title",
				Body:  "`inline without closing",
			},
			expectErr: true,
		},
		{
			name: "multiple code blocks",
			message: &message.Message{
				Title: "Title",
				Body:  "```\nblock1\n``` and ```\nblock2\n```",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateMarkdown(tt.message)

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

func TestMessageValidator_SanitizeContent(t *testing.T) {
	logger := logger.NewConsoleLogger("debug")
	validator := NewMessageValidator(logger)

	tests := []struct {
		name     string
		input    string
		checkLen bool
		contains []string
		notContains []string
	}{
		{
			name:     "plain text",
			input:    "Hello world",
			checkLen: true,
		},
		{
			name:        "HTML entities",
			input:       "<div>Hello</div>",
			contains:    []string{"&lt;", "&gt;"},
			notContains: []string{"<div>", "</div>"},
		},
		{
			name:        "script tag removal",
			input:       "<script>alert('xss')</script>",
			notContains: []string{"<script>", "alert"},
		},
		{
			name:        "javascript protocol removal",
			input:       "javascript:void(0)",
			notContains: []string{"javascript:"},
		},
		{
			name:        "event handler removal",
			input:       "onclick=alert()",
			notContains: []string{"onclick="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := validator.SanitizeContent(tt.input)

			if tt.checkLen && len(output) == 0 && len(tt.input) > 0 {
				t.Error("Expected non-empty output for non-empty input")
			}

			for _, substr := range tt.contains {
				if !strings.Contains(output, substr) {
					t.Errorf("Expected output to contain %q, got %q", substr, output)
				}
			}

			for _, substr := range tt.notContains {
				if strings.Contains(output, substr) {
					t.Errorf("Expected output to NOT contain %q, got %q", substr, output)
				}
			}
		})
	}
}

func TestMessageValidator_EstimateMessageSize(t *testing.T) {
	logger := logger.NewConsoleLogger("debug")
	validator := NewMessageValidator(logger)

	tests := []struct {
		name    string
		message *message.Message
		msgType string
		minSize int
	}{
		{
			name: "text message",
			message: &message.Message{
				Title: "Hello",
				Body:  "World",
			},
			msgType: "text",
			minSize: 20,
		},
		{
			name: "rich text message",
			message: &message.Message{
				Title: "Rich Title",
				Body:  "Rich content",
			},
			msgType: "post",
			minSize: 50,
		},
		{
			name: "interactive message",
			message: &message.Message{
				Title: "Card Title",
				Body:  "Card content",
			},
			msgType: "interactive",
			minSize: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := validator.EstimateMessageSize(tt.message, tt.msgType)

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
func BenchmarkMessageValidator_ValidateMessage(b *testing.B) {
	logger := logger.NewConsoleLogger("error")
	validator := NewMessageValidator(logger)

	msg := &message.Message{
		ID:     "benchmark-test",
		Title:  "Benchmark Title",
		Body:   "This is a benchmark test message with some content for validation testing",
		Format: message.FormatText,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateMessage(msg)
		if err != nil {
			b.Errorf("ValidateMessage failed: %v", err)
		}
	}
}

func BenchmarkMessageValidator_SanitizeContent(b *testing.B) {
	logger := logger.NewConsoleLogger("error")
	validator := NewMessageValidator(logger)

	content := "<div>Some content with <script>alert('test')</script> and other <b>HTML</b> tags</div>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.SanitizeContent(content)
	}
}