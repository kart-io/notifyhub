package notifyhub

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/config"
	"github.com/kart-io/notifyhub/pkg/config/platforms"
	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid email config",
			config: &config.Config{
				Email: &platforms.EmailConfig{
					Host:     "smtp.example.com",
					Port:     587,
					Username: "user@example.com",
					Password: "password",
					From:     "sender@example.com",
				},
				LoggerInstance: logger.New(),
			},
			wantErr: false,
		},
		{
			name: "valid feishu config",
			config: &config.Config{
				Feishu: &platforms.FeishuConfig{
					WebhookURL: "https://open.feishu.cn/webhook/test",
				},
				LoggerInstance: logger.New(),
			},
			wantErr: false,
		},
		{
			name: "missing logger - auto-created",
			config: &config.Config{
				Email: &platforms.EmailConfig{
					Host:     "smtp.example.com",
					Port:     587,
					Username: "user@example.com",
					Password: "password",
					From:     "sender@example.com",
				},
			},
			wantErr: false, // Validate() creates default logger
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client")
			}
			// Cleanup
			if client != nil {
				_ = client.Close()
			}
		})
	}
}

func TestNewClientFromOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    []config.Option
		wantErr bool
	}{
		{
			name: "valid email options",
			opts: []config.Option{
				config.WithEmail(platforms.EmailConfig{
					Host:     "smtp.example.com",
					Port:     587,
					Username: "user@example.com",
					Password: "password",
					From:     "sender@example.com",
				}),
				config.WithLogger(logger.New()),
			},
			wantErr: false,
		},
		{
			name: "missing logger - auto-created",
			opts: []config.Option{
				config.WithEmail(platforms.EmailConfig{
					Host:     "smtp.example.com",
					Port:     587,
					Username: "user@example.com",
					Password: "password",
					From:     "sender@example.com",
				}),
			},
			wantErr: false, // Validate() creates default logger
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClientFromOptions(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClientFromOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClientFromOptions() returned nil client")
			}
			// Cleanup
			if client != nil {
				_ = client.Close()
			}
		})
	}
}

func TestClientImpl_Close(t *testing.T) {
	cfg := &config.Config{
		Email: &platforms.EmailConfig{
			Host:     "smtp.example.com",
			Port:     587,
			Username: "user@example.com",
			Password: "password",
			From:     "sender@example.com",
		},
		LoggerInstance: logger.New(),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Close again should not error
	err = client.Close()
	if err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
}

func TestRegisterPlatformFactories(t *testing.T) {
	cfg := &config.Config{
		Email: &platforms.EmailConfig{
			Host:     "smtp.example.com",
			Port:     587,
			Username: "user@example.com",
			Password: "password",
			From:     "sender@example.com",
		},
		Feishu: &platforms.FeishuConfig{
			WebhookURL: "https://open.feishu.cn/webhook/test",
		},
		Webhook: &platforms.WebhookConfig{
			URL: "https://webhook.example.com",
		},
		LoggerInstance: logger.New(),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Verify all platforms are registered
	impl := client.(*clientImpl)
	platformNames := impl.platformRegistry.ListPlatforms()

	expectedPlatforms := []string{"email", "feishu", "webhook"}
	for _, name := range expectedPlatforms {
		found := false
		for _, p := range platformNames {
			if p == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Platform %s not registered", name)
		}
	}
}

func TestWithFeishu(t *testing.T) {
	cfg := &Config{}
	feishuCfg := FeishuConfig{
		WebhookURL: "https://open.feishu.cn/webhook/test",
	}
	opt := WithFeishu(feishuCfg)

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithFeishu() error = %v", err)
	}
	if cfg.Feishu == nil {
		t.Error("WithFeishu() should set Feishu config")
	}
}

func TestWithEmail(t *testing.T) {
	cfg := &Config{}
	emailCfg := EmailConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "sender@example.com",
	}
	opt := WithEmail(emailCfg)

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithEmail() error = %v", err)
	}
	if cfg.Email == nil {
		t.Error("WithEmail() should set Email config")
	}
}

func TestWithWebhook(t *testing.T) {
	cfg := &Config{}
	webhookCfg := WebhookConfig{
		URL: "https://webhook.example.com",
	}
	opt := WithWebhook(webhookCfg)

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithWebhook() error = %v", err)
	}
	if cfg.Webhook == nil {
		t.Error("WithWebhook() should set Webhook config")
	}
}

func TestWithAsync(t *testing.T) {
	cfg := &Config{}
	opt := WithAsync(8)

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithAsync() error = %v", err)
	}
	if !cfg.Async.Enabled {
		t.Error("WithAsync() should enable async")
	}
	if cfg.Async.Workers != 8 {
		t.Errorf("WithAsync() workers = %d, want 8", cfg.Async.Workers)
	}
}

func TestWithTimeout(t *testing.T) {
	cfg := &Config{}
	opt := WithTimeout(60 * time.Second)

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithTimeout() error = %v", err)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("WithTimeout() timeout = %v, want 60s", cfg.Timeout)
	}
}

func TestWithMaxRetries(t *testing.T) {
	cfg := &Config{}
	opt := WithMaxRetries(5)

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithMaxRetries() error = %v", err)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("WithMaxRetries() retries = %d, want 5", cfg.MaxRetries)
	}
}

func TestWithLogger(t *testing.T) {
	cfg := &Config{}
	log := logger.New()
	opt := WithLogger(log)

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithLogger() error = %v", err)
	}
	if cfg.LoggerInstance == nil {
		t.Error("WithLogger() should set LoggerInstance")
	}
}

func TestWithLoggerConfig(t *testing.T) {
	cfg := &Config{}
	opt := WithLoggerConfig("debug", "text")

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithLoggerConfig() error = %v", err)
	}
	if cfg.Logger.Level != "debug" {
		t.Errorf("WithLoggerConfig() level = %s, want debug", cfg.Logger.Level)
	}
	if cfg.Logger.Format != "text" {
		t.Errorf("WithLoggerConfig() format = %s, want text", cfg.Logger.Format)
	}
}

func TestWithFeishuWebhook(t *testing.T) {
	cfg := &Config{}
	opt := WithFeishuWebhook("https://open.feishu.cn/webhook/test", "secret123")

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithFeishuWebhook() error = %v", err)
	}
	if cfg.Feishu == nil || cfg.Feishu.WebhookURL != "https://open.feishu.cn/webhook/test" {
		t.Error("WithFeishuWebhook() should set webhook URL")
	}
	if cfg.Feishu.Secret != "secret123" {
		t.Error("WithFeishuWebhook() should set secret")
	}
}

func TestWithFeishuApp(t *testing.T) {
	cfg := &Config{}
	opt := WithFeishuApp("app123", "secret456")

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithFeishuApp() error = %v", err)
	}
	if cfg.Feishu == nil {
		t.Fatal("WithFeishuApp() should create Feishu config")
	}
	if cfg.Feishu.AppID != "app123" {
		t.Errorf("WithFeishuApp() AppID = %s, want app123", cfg.Feishu.AppID)
	}
	if cfg.Feishu.AppSecret != "secret456" {
		t.Errorf("WithFeishuApp() AppSecret = %s, want secret456", cfg.Feishu.AppSecret)
	}
}

func TestWithFeishuAdvanced(t *testing.T) {
	cfg := &Config{}
	opt := WithFeishuAdvanced(
		"https://open.feishu.cn/webhook/test",
		"secret123",
		[]string{"alert", "urgent"},
		30*time.Second,
		3,
		100,
		true,
	)

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithFeishuAdvanced() error = %v", err)
	}
	if cfg.Feishu == nil {
		t.Fatal("WithFeishuAdvanced() should create Feishu config")
	}
	if cfg.Feishu.WebhookURL != "https://open.feishu.cn/webhook/test" {
		t.Error("WithFeishuAdvanced() should set webhook URL")
	}
	if cfg.Feishu.Secret != "secret123" {
		t.Error("WithFeishuAdvanced() should set secret")
	}
	if !cfg.Feishu.SignVerify {
		t.Error("WithFeishuAdvanced() should set SignVerify")
	}
}

func TestWithEmailSMTP(t *testing.T) {
	cfg := &Config{}
	opt := WithEmailSMTP("smtp.example.com", 587, "user@example.com", "password", "sender@example.com")

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithEmailSMTP() error = %v", err)
	}
	if cfg.Email == nil {
		t.Fatal("WithEmailSMTP() should create Email config")
	}
	if cfg.Email.Host != "smtp.example.com" {
		t.Error("WithEmailSMTP() should set host")
	}
	if cfg.Email.Port != 587 {
		t.Error("WithEmailSMTP() should set port")
	}
}

func TestWithWebhookBasic(t *testing.T) {
	cfg := &Config{}
	opt := WithWebhookBasic("https://webhook.example.com")

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithWebhookBasic() error = %v", err)
	}
	if cfg.Webhook == nil || cfg.Webhook.URL != "https://webhook.example.com" {
		t.Error("WithWebhookBasic() should set webhook URL")
	}
}

func TestWithDefaults(t *testing.T) {
	cfg := &Config{}
	opt := WithDefaults()

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithDefaults() error = %v", err)
	}
	if cfg.Timeout == 0 {
		t.Error("WithDefaults() should set timeout")
	}
	if cfg.MaxRetries == 0 {
		t.Error("WithDefaults() should set max retries")
	}
}

func TestWithTestDefaults(t *testing.T) {
	cfg := &Config{}
	opt := WithTestDefaults()

	err := opt(cfg)
	if err != nil {
		t.Errorf("WithTestDefaults() error = %v", err)
	}
	if cfg.Timeout == 0 {
		t.Error("WithTestDefaults() should set timeout")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid email config",
			config: &Config{
				Email: &EmailConfig{
					Host: "smtp.example.com",
					Port: 587,
					From: "sender@example.com",
				},
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid feishu config",
			config: &Config{
				Feishu: &FeishuConfig{
					WebhookURL: "https://open.feishu.cn/webhook/test",
				},
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty config with defaults",
			config: &Config{
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_GetConfiguredPlatformCount(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   int
	}{
		{
			name:   "no platforms",
			config: &Config{},
			want:   0,
		},
		{
			name: "one platform",
			config: &Config{
				Email: &EmailConfig{
					Host: "smtp.example.com",
					Port: 587,
					From: "sender@example.com",
				},
			},
			want: 1,
		},
		{
			name: "multiple platforms",
			config: &Config{
				Email: &EmailConfig{
					Host: "smtp.example.com",
					Port: 587,
					From: "sender@example.com",
				},
				Feishu: &FeishuConfig{
					WebhookURL: "https://open.feishu.cn/webhook/test",
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetConfiguredPlatformCount(); got != tt.want {
				t.Errorf("GetConfiguredPlatformCount() = %v, want %v", got, tt.want)
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
	cfg := &Config{
		Async: AsyncConfig{
			Workers: 8,
		},
	}

	if got := cfg.GetWorkers(); got != 8 {
		t.Errorf("GetWorkers() = %v, want 8", got)
	}
}

func TestConfig_HasFeishu(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "has feishu",
			config: &Config{
				Feishu: &FeishuConfig{
					WebhookURL: "https://open.feishu.cn/webhook/test",
				},
			},
			want: true,
		},
		{
			name:   "no feishu",
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
			name: "has email",
			config: &Config{
				Email: &EmailConfig{
					Host: "smtp.example.com",
					Port: 587,
					From: "sender@example.com",
				},
			},
			want: true,
		},
		{
			name:   "no email",
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
			name: "has webhook",
			config: &Config{
				Webhook: &WebhookConfig{
					URL: "https://webhook.example.com",
				},
			},
			want: true,
		},
		{
			name:   "no webhook",
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
