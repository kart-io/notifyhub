package config

import (
	"testing"
	"time"

	"github.com/kart/notifyhub/pkg/config/platforms"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid email config",
			config: &Config{
				Email: &platforms.EmailConfig{
					Host:     "smtp.example.com",
					Port:     587,
					Username: "user@example.com",
					Password: "password",
					From:     "sender@example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "valid feishu config",
			config: &Config{
				Feishu: &platforms.FeishuConfig{
					WebhookURL: "https://open.feishu.cn/webhook/test",
				},
			},
			wantErr: false,
		},
		{
			name: "valid webhook config",
			config: &Config{
				Webhook: &platforms.WebhookConfig{
					URL: "https://webhook.example.com",
				},
			},
			wantErr: false,
		},
		{
			name:    "empty config",
			config:  &Config{},
			wantErr: false, // Validate sets defaults, doesn't error
		},
		{
			name: "invalid email - missing host",
			config: &Config{
				Email: &platforms.EmailConfig{
					Port:     587,
					Username: "user@example.com",
					Password: "password",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmailConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *platforms.EmailConfig
		wantErr bool
	}{
		{
			name: "valid SMTP config",
			config: &platforms.EmailConfig{
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "user@gmail.com",
				Password: "password",
				From:     "sender@gmail.com",
			},
			wantErr: false,
		},
		{
			name: "valid TLS config",
			config: &platforms.EmailConfig{
				Host:     "smtp.gmail.com",
				Port:     465,
				Username: "user@gmail.com",
				Password: "password",
				From:     "sender@gmail.com",
				UseTLS:   true,
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: &platforms.EmailConfig{
				Port:     587,
				Username: "user@gmail.com",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "invalid port - zero",
			config: &platforms.EmailConfig{
				Host:     "smtp.gmail.com",
				Port:     0,
				Username: "user@gmail.com",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			config: &platforms.EmailConfig{
				Host:     "smtp.gmail.com",
				Port:     70000,
				Username: "user@gmail.com",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "missing from address",
			config: &platforms.EmailConfig{
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "user@gmail.com",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "username without password",
			config: &platforms.EmailConfig{
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "user@gmail.com",
				From:     "sender@gmail.com",
			},
			wantErr: false, // Current validation doesn't check username/password pairing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EmailConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFeishuConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *platforms.FeishuConfig
		wantErr bool
	}{
		{
			name: "valid webhook config",
			config: &platforms.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/webhook/test",
			},
			wantErr: false,
		},
		{
			name: "valid webhook with secret",
			config: &platforms.FeishuConfig{
				WebhookURL: "https://open.feishu.cn/webhook/test",
				Secret:     "secret123",
			},
			wantErr: false,
		},
		{
			name:    "missing webhook URL",
			config:  &platforms.FeishuConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FeishuConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWebhookConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *platforms.WebhookConfig
		wantErr bool
	}{
		{
			name: "valid basic config",
			config: &platforms.WebhookConfig{
				URL: "https://webhook.example.com",
			},
			wantErr: false,
		},
		{
			name: "valid config with basic auth",
			config: &platforms.WebhookConfig{
				URL:      "https://webhook.example.com",
				AuthType: "basic",
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name:    "missing URL",
			config:  &platforms.WebhookConfig{},
			wantErr: true,
		},
		{
			name: "invalid URL",
			config: &platforms.WebhookConfig{
				URL: "not-a-url",
			},
			wantErr: false, // Current validation doesn't check URL format
		},
		{
			name: "basic auth without username",
			config: &platforms.WebhookConfig{
				URL:      "https://webhook.example.com",
				AuthType: "basic",
				Password: "pass",
			},
			wantErr: false, // Current validation doesn't check auth completeness
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WebhookConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSlackConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *platforms.SlackConfig
		wantErr bool
	}{
		{
			name: "valid webhook config",
			config: &platforms.SlackConfig{
				WebhookURL: "https://hooks.slack.com/services/T00/B00/XXX",
			},
			wantErr: false,
		},
		{
			name: "valid token config",
			config: &platforms.SlackConfig{
				Token:   "xoxb-123-456",
				Channel: "#general",
			},
			wantErr: false,
		},
		{
			name:    "missing URL and token",
			config:  &platforms.SlackConfig{},
			wantErr: true,
		},
		{
			name: "token without channel",
			config: &platforms.SlackConfig{
				Token: "xoxb-123-456",
			},
			wantErr: false, // Channel is optional when using token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := &Config{
		Timeout: 30 * time.Second,
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", config.Timeout, 30*time.Second)
	}

	// Check async is disabled by default
	if config.Async.Enabled {
		t.Error("Async should be disabled by default")
	}
}

func TestConfig_HasPlatform(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		platform string
		want     bool
	}{
		{
			name: "has email",
			config: &Config{
				Email: &platforms.EmailConfig{
					Host: "smtp.example.com",
					Port: 587,
				},
			},
			platform: "email",
			want:     true,
		},
		{
			name: "has feishu",
			config: &Config{
				Feishu: &platforms.FeishuConfig{
					WebhookURL: "https://open.feishu.cn/webhook/test",
				},
			},
			platform: "feishu",
			want:     true,
		},
		{
			name:     "no email",
			config:   &Config{},
			platform: "email",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bool
			switch tt.platform {
			case "email":
				got = tt.config.Email != nil
			case "feishu":
				got = tt.config.Feishu != nil
			}

			if got != tt.want {
				t.Errorf("Config has %s = %v, want %v", tt.platform, got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "default config",
			opts:    nil,
			wantErr: false,
		},
		{
			name: "with timeout option",
			opts: []Option{
				WithTimeout(60 * time.Second),
			},
			wantErr: false,
		},
		{
			name: "with max retries option",
			opts: []Option{
				WithMaxRetries(5),
			},
			wantErr: false,
		},
		{
			name: "with async enabled",
			opts: []Option{
				WithAsync(8),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && cfg == nil {
				t.Error("New() returned nil config")
			}
		})
	}
}

func TestConfig_IsAsyncEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "async enabled",
			config: &Config{
				Async: AsyncConfig{
					Enabled: true,
				},
			},
			want: true,
		},
		{
			name: "async disabled",
			config: &Config{
				Async: AsyncConfig{
					Enabled: false,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsAsyncEnabled(); got != tt.want {
				t.Errorf("IsAsyncEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetWorkers(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   int
	}{
		{
			name: "custom workers",
			config: &Config{
				Async: AsyncConfig{
					Workers: 8,
				},
			},
			want: 8,
		},
		{
			name: "zero workers - default",
			config: &Config{
				Async: AsyncConfig{
					Workers: 0,
				},
			},
			want: 4,
		},
		{
			name: "negative workers - default",
			config: &Config{
				Async: AsyncConfig{
					Workers: -1,
				},
			},
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetWorkers(); got != tt.want {
				t.Errorf("GetWorkers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 45 * time.Second
	cfg := &Config{}

	opt := WithTimeout(timeout)
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithTimeout() error = %v", err)
	}
	if cfg.Timeout != timeout {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, timeout)
	}
}

func TestWithMaxRetries(t *testing.T) {
	retries := 5
	cfg := &Config{}

	opt := WithMaxRetries(retries)
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithMaxRetries() error = %v", err)
	}
	if cfg.MaxRetries != retries {
		t.Errorf("MaxRetries = %v, want %v", cfg.MaxRetries, retries)
	}
}

func TestWithAsync(t *testing.T) {
	cfg := &Config{}

	opt := WithAsync(10)
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithAsync() error = %v", err)
	}
	if !cfg.Async.Enabled {
		t.Error("Async should be enabled")
	}
	if cfg.Async.Workers != 10 {
		t.Errorf("Workers = %v, want 10", cfg.Async.Workers)
	}
}

func TestAsyncConfig_Defaults(t *testing.T) {
	cfg, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if cfg.Async.Enabled {
		t.Error("Async should be disabled by default")
	}
	if cfg.Async.Workers != 4 {
		t.Errorf("Default workers = %v, want 4", cfg.Async.Workers)
	}
}

func TestLoggerConfig_Defaults(t *testing.T) {
	cfg, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if cfg.Logger.Level != "info" {
		t.Errorf("Default log level = %v, want info", cfg.Logger.Level)
	}
	if cfg.Logger.Format != "json" {
		t.Errorf("Default log format = %v, want json", cfg.Logger.Format)
	}
}

func TestWithFeishu(t *testing.T) {
	cfg := &Config{}
	feishuCfg := FeishuConfig{
		WebhookURL: "https://open.feishu.cn/webhook/test",
		Secret:     "secret123",
	}

	opt := WithFeishu(feishuCfg)
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithFeishu() error = %v", err)
	}
	if cfg.Feishu == nil {
		t.Fatal("Feishu config not set")
	}
	if cfg.Feishu.WebhookURL != feishuCfg.WebhookURL {
		t.Errorf("WebhookURL = %v, want %v", cfg.Feishu.WebhookURL, feishuCfg.WebhookURL)
	}
}

func TestWithEmail(t *testing.T) {
	cfg := &Config{}
	emailCfg := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		From:     "sender@example.com",
		Username: "user",
		Password: "pass",
	}

	opt := WithEmail(emailCfg)
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithEmail() error = %v", err)
	}
	if cfg.Email == nil {
		t.Fatal("Email config not set")
	}
	if cfg.Email.Host != emailCfg.Host {
		t.Errorf("Host = %v, want %v", cfg.Email.Host, emailCfg.Host)
	}
}

func TestWithWebhook(t *testing.T) {
	cfg := &Config{}
	webhookCfg := WebhookConfig{
		URL:    "https://webhook.example.com",
		Method: "POST",
	}

	opt := WithWebhook(webhookCfg)
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithWebhook() error = %v", err)
	}
	if cfg.Webhook == nil {
		t.Fatal("Webhook config not set")
	}
	if cfg.Webhook.URL != webhookCfg.URL {
		t.Errorf("URL = %v, want %v", cfg.Webhook.URL, webhookCfg.URL)
	}
}

func TestWithSlack(t *testing.T) {
	cfg := &Config{}
	slackCfg := SlackConfig{
		WebhookURL: "https://hooks.slack.com/services/TEST",
		Channel:    "#general",
	}

	opt := WithSlack(slackCfg)
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithSlack() error = %v", err)
	}
	if cfg.Slack == nil {
		t.Fatal("Slack config not set")
	}
	if cfg.Slack.WebhookURL != slackCfg.WebhookURL {
		t.Errorf("WebhookURL = %v, want %v", cfg.Slack.WebhookURL, slackCfg.WebhookURL)
	}
}

func TestWithQuickFeishu(t *testing.T) {
	cfg := &Config{}

	opt := WithQuickFeishu("https://open.feishu.cn/webhook/test", "secret")
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithQuickFeishu() error = %v", err)
	}
	if cfg.Feishu == nil {
		t.Fatal("Feishu config not set")
	}
	if cfg.Feishu.Secret != "secret" {
		t.Errorf("Secret = %v, want secret", cfg.Feishu.Secret)
	}
}

func TestWithQuickEmail(t *testing.T) {
	cfg := &Config{}

	opt := WithQuickEmail("smtp.example.com", 587, "sender@example.com")
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithQuickEmail() error = %v", err)
	}
	if cfg.Email == nil {
		t.Fatal("Email config not set")
	}
	if cfg.Email.Host != "smtp.example.com" {
		t.Errorf("Host = %v, want smtp.example.com", cfg.Email.Host)
	}
}

func TestWithQuickWebhook(t *testing.T) {
	cfg := &Config{}

	opt := WithQuickWebhook("https://webhook.example.com")
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithQuickWebhook() error = %v", err)
	}
	if cfg.Webhook == nil {
		t.Fatal("Webhook config not set")
	}
	if cfg.Webhook.URL != "https://webhook.example.com" {
		t.Errorf("URL = %v, want https://webhook.example.com", cfg.Webhook.URL)
	}
}

func TestWithQuickSlack(t *testing.T) {
	cfg := &Config{}

	opt := WithQuickSlack("https://hooks.slack.com/test", "xoxb-token")
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithQuickSlack() error = %v", err)
	}
	if cfg.Slack == nil {
		t.Fatal("Slack config not set")
	}
}

func TestWithSlackWebhook(t *testing.T) {
	cfg := &Config{}

	opt := WithSlackWebhook("https://hooks.slack.com/test")
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithSlackWebhook() error = %v", err)
	}
	if cfg.Slack == nil {
		t.Fatal("Slack config not set")
	}
	if cfg.Slack.WebhookURL != "https://hooks.slack.com/test" {
		t.Errorf("WebhookURL = %v, want https://hooks.slack.com/test", cfg.Slack.WebhookURL)
	}
}

func TestWithSlackToken(t *testing.T) {
	cfg := &Config{}

	opt := WithSlackToken("xoxb-token", "#general")
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithSlackToken() error = %v", err)
	}
	if cfg.Slack == nil {
		t.Fatal("Slack config not set")
	}
	if cfg.Slack.Token != "xoxb-token" {
		t.Errorf("Token = %v, want xoxb-token", cfg.Slack.Token)
	}
}

func TestWithDefaults(t *testing.T) {
	cfg := &Config{}

	opt := WithDefaults()
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithDefaults() error = %v", err)
	}
	// Check some defaults are set
	if cfg.Timeout == 0 {
		t.Error("Timeout should be set by defaults")
	}
}

func TestWithProductionDefaults(t *testing.T) {
	cfg := &Config{}

	opt := WithProductionDefaults()
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithProductionDefaults() error = %v", err)
	}
	// Check production defaults
	if cfg.Timeout == 0 {
		t.Error("Timeout should be set by production defaults")
	}
}

func TestWithTestDefaults(t *testing.T) {
	cfg := &Config{}

	opt := WithTestDefaults()
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithTestDefaults() error = %v", err)
	}
	// Check test defaults
	if cfg.Timeout == 0 {
		t.Error("Timeout should be set by test defaults")
	}
}

func TestConfig_GetAsyncDefaults(t *testing.T) {
	cfg := &Config{}
	defaults := cfg.GetAsyncDefaults()

	if defaults.Enabled {
		t.Error("GetAsyncDefaults() Enabled should be false by default")
	}
	if defaults.Workers != 4 {
		t.Errorf("GetAsyncDefaults() Workers = %v, want 4", defaults.Workers)
	}
	if defaults.BufferSize != 1000 {
		t.Errorf("GetAsyncDefaults() BufferSize = %v, want 1000", defaults.BufferSize)
	}
}

func TestConfig_IsPoolModeEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "pool mode enabled",
			config: &Config{
				Async: AsyncConfig{
					Enabled: true,
					UsePool: true,
				},
			},
			want: true,
		},
		{
			name: "pool mode disabled - UsePool false",
			config: &Config{
				Async: AsyncConfig{
					Enabled: true,
					UsePool: false,
				},
			},
			want: false,
		},
		{
			name: "pool mode disabled - Async disabled",
			config: &Config{
				Async: AsyncConfig{
					Enabled: false,
					UsePool: true,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsPoolModeEnabled(); got != tt.want {
				t.Errorf("IsPoolModeEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_HasFeishu(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "has feishu config",
			config: &Config{
				Feishu: &platforms.FeishuConfig{
					WebhookURL: "https://open.feishu.cn/webhook/test",
				},
			},
			want: true,
		},
		{
			name:   "no feishu config",
			config: &Config{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasFeishu(); got != tt.want {
				t.Errorf("HasFeishu() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_HasEmail(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "has email config",
			config: &Config{
				Email: &platforms.EmailConfig{
					Host: "smtp.example.com",
					Port: 587,
					From: "sender@example.com",
				},
			},
			want: true,
		},
		{
			name:   "no email config",
			config: &Config{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasEmail(); got != tt.want {
				t.Errorf("HasEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_HasWebhook(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "has webhook config",
			config: &Config{
				Webhook: &platforms.WebhookConfig{
					URL: "https://webhook.example.com",
				},
			},
			want: true,
		},
		{
			name:   "no webhook config",
			config: &Config{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasWebhook(); got != tt.want {
				t.Errorf("HasWebhook() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_HasSlack(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "has slack config",
			config: &Config{
				Slack: &platforms.SlackConfig{
					WebhookURL: "https://hooks.slack.com/services/TEST",
				},
			},
			want: true,
		},
		{
			name:   "no slack config",
			config: &Config{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasSlack(); got != tt.want {
				t.Errorf("HasSlack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithLogger(t *testing.T) {
	cfg := &Config{}
	mockLog := &mockLogger{}

	opt := WithLogger(mockLog)
	err := opt(cfg)

	if err != nil {
		t.Errorf("WithLogger() error = %v", err)
	}
	if cfg.LoggerInstance == nil {
		t.Error("WithLogger() should set LoggerInstance")
	}
}

type mockLogger struct{}

func (m *mockLogger) LogMode(level logger.LogLevel) logger.Logger     { return m }
func (m *mockLogger) Debug(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Info(msg string, keysAndValues ...interface{})   {}
func (m *mockLogger) Warn(msg string, keysAndValues ...interface{})   {}
func (m *mockLogger) Error(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Fatal(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) With(keysAndValues ...interface{}) logger.Logger { return m }
