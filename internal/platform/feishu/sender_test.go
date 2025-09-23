package feishu

import (
	"testing"

	"github.com/kart-io/notifyhub/internal/platform"
)

func TestParseFeishuConfig_AuthModes(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		expectMode  AuthMode
		expectError bool
		errorMsg    string
	}{
		{
			name: "explicit auth_mode: none",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "none",
			},
			expectMode:  AuthModeNone,
			expectError: false,
		},
		{
			name: "explicit auth_mode: signature with secret",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "signature",
				"secret":      "test_secret",
			},
			expectMode:  AuthModeSignature,
			expectError: false,
		},
		{
			name: "explicit auth_mode: keywords with keywords",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "keywords",
				"keywords":    []string{"trigger", "alert"},
			},
			expectMode:  AuthModeKeywords,
			expectError: false,
		},
		{
			name: "implicit signature mode (secret only)",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"secret":      "test_secret",
			},
			expectMode:  AuthModeSignature,
			expectError: false,
		},
		{
			name: "implicit keywords mode (keywords only)",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"keywords":    []string{"trigger"},
			},
			expectMode:  AuthModeKeywords,
			expectError: false,
		},
		{
			name: "implicit none mode (no auth params)",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
			},
			expectMode:  AuthModeNone,
			expectError: false,
		},
		{
			name: "error: signature mode without secret",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "signature",
			},
			expectError: true,
			errorMsg:    "secret is required when auth_mode is 'signature'",
		},
		{
			name: "error: keywords mode without keywords",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "keywords",
			},
			expectError: true,
			errorMsg:    "keywords are required when auth_mode is 'keywords'",
		},
		{
			name: "error: none mode with secret",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "none",
				"secret":      "should_not_exist",
			},
			expectError: true,
			errorMsg:    "secret should not be provided when auth_mode is 'none'",
		},
		{
			name: "error: none mode with keywords",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "none",
				"keywords":    []string{"should_not_exist"},
			},
			expectError: true,
			errorMsg:    "keywords should not be provided when auth_mode is 'none'",
		},
		{
			name: "error: signature mode with keywords",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "signature",
				"secret":      "test_secret",
				"keywords":    []string{"should_not_exist"},
			},
			expectError: true,
			errorMsg:    "keywords should not be provided when auth_mode is 'signature'",
		},
		{
			name: "error: keywords mode with secret",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "keywords",
				"keywords":    []string{"trigger"},
				"secret":      "should_not_exist",
			},
			expectError: true,
			errorMsg:    "secret should not be provided when auth_mode is 'keywords'",
		},
		{
			name: "error: both secret and keywords without explicit auth_mode",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"secret":      "test_secret",
				"keywords":    []string{"trigger"},
			},
			expectError: true,
			errorMsg:    "both secret and keywords provided, they are mutually exclusive",
		},
		{
			name: "error: invalid auth_mode",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"auth_mode":   "invalid",
			},
			expectError: true,
			errorMsg:    "invalid auth_mode: invalid",
		},
		{
			name: "keywords as string",
			config: map[string]interface{}{
				"webhook_url": "https://open.feishu.cn/webhook/xxx",
				"keywords":    "single_keyword",
			},
			expectMode:  AuthModeKeywords,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parseFeishuConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if config.AuthMode != tt.expectMode {
					t.Errorf("expected auth_mode '%s', got '%s'", tt.expectMode, config.AuthMode)
				}
			}
		})
	}
}

func TestFeishuSender_MessageBuilding_WithAuthModes(t *testing.T) {
	tests := []struct {
		name            string
		authMode        AuthMode
		secret          string
		keywords        []string
		expectTimestamp bool
		expectSign      bool
		expectKeyword   bool
	}{
		{
			name:            "signature mode adds timestamp and sign",
			authMode:        AuthModeSignature,
			secret:          "test_secret",
			expectTimestamp: true,
			expectSign:      true,
			expectKeyword:   false,
		},
		{
			name:            "keywords mode adds keyword to text",
			authMode:        AuthModeKeywords,
			keywords:        []string{"trigger", "alert"},
			expectTimestamp: false,
			expectSign:      false,
			expectKeyword:   true,
		},
		{
			name:            "none mode adds nothing",
			authMode:        AuthModeNone,
			expectTimestamp: false,
			expectSign:      false,
			expectKeyword:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &FeishuSender{
				config: FeishuConfig{
					AuthMode: tt.authMode,
					Secret:   tt.secret,
					Keywords: tt.keywords,
				},
			}

			// Test message building for signature/no auth
			msg := &platform.InternalMessage{
				Title: "Test Title",
				Body:  "Test Body",
			}

			feishuMsg, err := sender.buildFeishuMessage(msg, platform.InternalTarget{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check timestamp and signature
			if tt.expectTimestamp {
				if feishuMsg.Timestamp == "" {
					t.Error("expected timestamp but got empty")
				}
			} else {
				if feishuMsg.Timestamp != "" {
					t.Errorf("expected no timestamp but got: %s", feishuMsg.Timestamp)
				}
			}

			if tt.expectSign {
				if feishuMsg.Sign == "" {
					t.Error("expected signature but got empty")
				}
			} else {
				if feishuMsg.Sign != "" {
					t.Errorf("expected no signature but got: %s", feishuMsg.Sign)
				}
			}

			// Check keyword in text content
			if content, ok := feishuMsg.Content.(FeishuTextContent); ok {
				if tt.expectKeyword {
					if !contains(content.Text, tt.keywords[0]) {
						t.Errorf("expected keyword '%s' in text, got: %s", tt.keywords[0], content.Text)
					}
				} else if len(tt.keywords) > 0 {
					if contains(content.Text, tt.keywords[0]) {
						t.Errorf("unexpected keyword '%s' in text: %s", tt.keywords[0], content.Text)
					}
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(substr) < len(s) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
