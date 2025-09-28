// Package config benchmark tests for performance verification
package config

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// BenchmarkNewConfig benchmarks configuration creation performance
func BenchmarkNewConfig(b *testing.B) {
	opts := []Option{
		WithTimeout(30 * time.Second),
		WithMaxRetries(3),
		WithMetrics(true),
		WithTracing(true),
		WithFeishuWebhook("https://benchmark.webhook", "secret"),
		WithEmailBasic("smtp.benchmark.com", 587, "test@benchmark.com"),
		WithAsyncBasic("memory", 4),
		WithRouterBasic("round_robin"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := NewConfig(opts...)
		if err != nil {
			b.Fatal(err)
		}
		_ = cfg
	}
}

// BenchmarkNewConfigWithoutValidation benchmarks config creation without validation
func BenchmarkNewConfigWithoutValidation(b *testing.B) {
	opts := []Option{
		WithTimeout(30 * time.Second),
		WithMaxRetries(3),
		WithMetrics(true),
		WithTracing(true),
		WithFeishuWebhook("https://benchmark.webhook", "secret"),
		WithEmailBasic("smtp.benchmark.com", 587, "test@benchmark.com"),
		WithAsyncBasic("memory", 4),
		WithRouterBasic("round_robin"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := NewConfigWithoutValidation(opts...)
		if err != nil {
			b.Fatal(err)
		}
		_ = cfg
	}
}

// BenchmarkConfigValidation benchmarks configuration validation performance
func BenchmarkConfigValidation(b *testing.B) {
	cfg := &Config{
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
		EnableMetrics:  true,
		EnableTracing:  true,
		Email: &EmailConfig{
			SMTPHost:     "smtp.benchmark.com",
			SMTPPort:     587,
			SMTPUsername: "benchmark@example.com",
			SMTPPassword: "password123",
			SMTPFrom:     "benchmark@example.com",
			SMTPTLS:      true,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
			RateLimit:    60,
		},
		Feishu: &FeishuConfig{
			WebhookURL: "https://open.feishu.cn/webhook/benchmark",
			Secret:     "benchmark-secret",
			AuthType:   "webhook",
			Timeout:    20 * time.Second,
			MaxRetries: 2,
			RateLimit:  100,
			Keywords:   []string{"benchmark", "test"},
		},
		SMS: &SMSConfig{
			Provider:   "twilio",
			APIKey:     "benchmark-key",
			APISecret:  "benchmark-secret",
			From:       "+1234567890",
			Region:     "us-east-1",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			RateLimit:  30,
			Templates:  map[string]string{"verification": "Code: {{code}}"},
		},
		Slack: &SlackConfig{
			WebhookURL:  "https://hooks.slack.com/benchmark",
			Token:       "xoxb-benchmark",
			Channel:     "#benchmark",
			Username:    "BenchmarkBot",
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			RateLimit:   100,
			LinkNames:   true,
			UnfurlLinks: true,
		},
		Webhook: &WebhookConfig{
			URL:         "https://api.benchmark.com/webhook",
			Method:      "POST",
			Headers:     map[string]string{"X-API-Key": "benchmark"},
			AuthType:    "bearer",
			AuthValue:   "benchmark-token",
			Timeout:     20 * time.Second,
			MaxRetries:  3,
			RateLimit:   100,
			VerifySSL:   true,
			ContentType: "application/json",
		},
		AsyncConfig: AsyncConfig{
			Enabled:        true,
			QueueType:      "memory",
			QueueSize:      1000,
			Workers:        4,
			BatchSize:      10,
			ProcessTimeout: 30 * time.Second,
		},
		RouterConfig: RouterConfig{
			LoadBalancing:       "round_robin",
			HealthCheckInterval: 30 * time.Second,
			MaxRetries:          3,
			RetryDelay:          1 * time.Second,
			EnableFailover:      true,
		},
		RetryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			Multiplier:      2.0,
			MaxInterval:     30 * time.Second,
		},
		Platforms: make(map[string]map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cfg.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOptionApplication benchmarks individual option application
func BenchmarkOptionApplication(b *testing.B) {
	cfg := &Config{
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
		Platforms:      make(map[string]map[string]interface{}),
	}

	b.Run("WithTimeout", func(b *testing.B) {
		opt := WithTimeout(45 * time.Second)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = opt(cfg)
		}
	})

	b.Run("WithFeishuWebhook", func(b *testing.B) {
		opt := WithFeishuWebhook("https://benchmark.webhook", "secret")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = opt(cfg)
		}
	})

	b.Run("WithEmailBasic", func(b *testing.B) {
		opt := WithEmailBasic("smtp.benchmark.com", 587, "test@benchmark.com")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = opt(cfg)
		}
	})

	b.Run("WithAsyncConfig", func(b *testing.B) {
		asyncCfg := AsyncConfig{
			Enabled:        true,
			QueueType:      "redis",
			QueueSize:      2000,
			Workers:        8,
			BatchSize:      20,
			ProcessTimeout: 60 * time.Second,
		}
		opt := WithAsyncConfig(asyncCfg)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = opt(cfg)
		}
	})
}

// BenchmarkComplexConfiguration benchmarks complex configuration scenarios
func BenchmarkComplexConfiguration(b *testing.B) {
	b.Run("ProductionLike", func(b *testing.B) {
		opts := []Option{
			WithTimeout(60 * time.Second),
			WithMaxRetries(5),
			WithMetrics(true),
			WithTracing(true),
			WithFeishu(FeishuConfig{
				WebhookURL: "https://open.feishu.cn/webhook/production",
				Secret:     "production-secret-123456789",
				AuthType:   "webhook",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				RateLimit:  20,
				SignVerify: true,
				Keywords:   []string{"[ALERT]", "[ERROR]", "[CRITICAL]", "[WARNING]"},
			}),
			WithEmail(EmailConfig{
				SMTPHost:     "smtp.production.com",
				SMTPPort:     587,
				SMTPUsername: "alerts@production.com",
				SMTPPassword: "production-password-123",
				SMTPFrom:     "alerts@production.com",
				SMTPTLS:      true,
				SMTPSSL:      false,
				Timeout:      45 * time.Second,
				MaxRetries:   5,
				RateLimit:    30,
			}),
			WithSlack(SlackConfig{
				WebhookURL:  "https://hooks.slack.com/production",
				Token:       "xoxb-production-token",
				Channel:     "#alerts-production",
				Username:    "ProductionAlerts",
				IconURL:     "https://production.com/alert-icon.png",
				Timeout:     30 * time.Second,
				MaxRetries:  3,
				RateLimit:   50,
				LinkNames:   true,
				UnfurlLinks: true,
			}),
			WithSMS(SMSConfig{
				Provider:   "twilio",
				APIKey:     "production-twilio-key",
				APISecret:  "production-twilio-secret",
				From:       "+1234567890",
				Region:     "us-east-1",
				Timeout:    20 * time.Second,
				MaxRetries: 3,
				RateLimit:  10,
				Templates: map[string]string{
					"alert":        "ALERT: {{message}}",
					"verification": "Code: {{code}}",
					"notification": "{{title}}: {{body}}",
				},
			}),
			WithWebhook(WebhookConfig{
				URL:         "https://api.production.com/webhook",
				Method:      "POST",
				Headers:     map[string]string{"X-API-Key": "prod-key", "X-Source": "notifyhub"},
				AuthType:    "bearer",
				AuthValue:   "production-bearer-token",
				Timeout:     30 * time.Second,
				MaxRetries:  3,
				RateLimit:   100,
				VerifySSL:   true,
				ContentType: "application/json",
			}),
			WithAsyncConfig(AsyncConfig{
				Enabled:        true,
				QueueType:      "redis",
				QueueSize:      10000,
				Workers:        16,
				BatchSize:      50,
				ProcessTimeout: 120 * time.Second,
			}),
			WithRouter(RouterConfig{
				LoadBalancing:       "weighted",
				HealthCheckInterval: 60 * time.Second,
				MaxRetries:          5,
				RetryDelay:          2 * time.Second,
				EnableFailover:      true,
			}),
			WithRetryPolicy(5, 1000, 2.0),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg, err := NewConfig(opts...)
			if err != nil {
				b.Fatal(err)
			}
			_ = cfg
		}
	})

	b.Run("MinimalConfiguration", func(b *testing.B) {
		opts := []Option{
			WithFeishuWebhook("https://minimal.webhook"),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg, err := NewConfig(opts...)
			if err != nil {
				b.Fatal(err)
			}
			_ = cfg
		}
	})

	b.Run("TestConfiguration", func(b *testing.B) {
		opts := []Option{
			WithTestDefaults(),
			WithFeishuWebhook("http://localhost:8080/feishu"),
			WithWebhook(WebhookConfig{
				URL:         "http://localhost:8080/webhook",
				Method:      "POST",
				Headers:     make(map[string]string),
				AuthType:    "bearer",
				AuthValue:   "test-token",
				Timeout:     30 * time.Second,
				MaxRetries:  3,
				RateLimit:   100,
				VerifySSL:   false, // For localhost
				ContentType: "application/json",
			}),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg, err := NewConfig(opts...)
			if err != nil {
				b.Fatal(err)
			}
			_ = cfg
		}
	})
}

// BenchmarkConfigSerialization benchmarks configuration serialization
func BenchmarkConfigSerialization(b *testing.B) {
	cfg := &Config{
		DefaultTimeout: 45 * time.Second,
		MaxRetries:     5,
		EnableMetrics:  true,
		EnableTracing:  true,
		Email: &EmailConfig{
			SMTPHost:     "smtp.benchmark.com",
			SMTPPort:     587,
			SMTPUsername: "benchmark@example.com",
			SMTPPassword: "password123",
			SMTPFrom:     "benchmark@example.com",
			SMTPTLS:      true,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
			RateLimit:    60,
		},
		Feishu: &FeishuConfig{
			WebhookURL: "https://open.feishu.cn/webhook/benchmark",
			Secret:     "benchmark-secret",
			AuthType:   "webhook",
			Timeout:    20 * time.Second,
			MaxRetries: 2,
			RateLimit:  100,
			Keywords:   []string{"benchmark", "test", "performance"},
		},
		AsyncConfig: AsyncConfig{
			Enabled:        true,
			QueueType:      "redis",
			QueueSize:      2000,
			Workers:        8,
			BatchSize:      20,
			ProcessTimeout: 60 * time.Second,
		},
		RouterConfig: RouterConfig{
			LoadBalancing:       "weighted",
			HealthCheckInterval: 60 * time.Second,
			MaxRetries:          5,
			RetryDelay:          2 * time.Second,
			EnableFailover:      true,
		},
		RetryPolicy: RetryPolicy{
			MaxRetries:      5,
			InitialInterval: 2 * time.Second,
			Multiplier:      2.5,
			MaxInterval:     60 * time.Second,
		},
	}

	b.Run("JSONSerialization", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(cfg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkLegacyPlatformConversion benchmarks legacy platform config conversion
func BenchmarkLegacyPlatformConversion(b *testing.B) {
	cfg := &Config{
		Email: &EmailConfig{
			SMTPHost:     "smtp.benchmark.com",
			SMTPPort:     587,
			SMTPUsername: "benchmark@example.com",
			SMTPPassword: "password123",
			SMTPFrom:     "benchmark@example.com",
			SMTPTLS:      true,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
			RateLimit:    60,
		},
		Feishu: &FeishuConfig{
			WebhookURL: "https://open.feishu.cn/webhook/benchmark",
			Secret:     "benchmark-secret",
			AuthType:   "webhook",
			Timeout:    20 * time.Second,
			MaxRetries: 2,
			RateLimit:  100,
			Keywords:   []string{"benchmark", "test"},
		},
		SMS: &SMSConfig{
			Provider:   "twilio",
			APIKey:     "benchmark-key",
			APISecret:  "benchmark-secret",
			From:       "+1234567890",
			Region:     "us-east-1",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			RateLimit:  30,
			Templates:  map[string]string{"verification": "Code: {{code}}"},
		},
		Slack: &SlackConfig{
			WebhookURL:  "https://hooks.slack.com/benchmark",
			Token:       "xoxb-benchmark",
			Channel:     "#benchmark",
			Username:    "BenchmarkBot",
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			RateLimit:   100,
			LinkNames:   true,
			UnfurlLinks: true,
		},
		Webhook: &WebhookConfig{
			URL:         "https://api.benchmark.com/webhook",
			Method:      "POST",
			Headers:     map[string]string{"X-API-Key": "benchmark"},
			AuthType:    "bearer",
			AuthValue:   "benchmark-token",
			Timeout:     20 * time.Second,
			MaxRetries:  3,
			RateLimit:   100,
			VerifySSL:   true,
			ContentType: "application/json",
		},
	}

	platforms := []string{"email", "feishu", "sms", "slack", "webhook"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, platform := range platforms {
			_ = cfg.GetPlatformConfig(platform)
		}
	}
}

// BenchmarkEnvironmentVariableLoading benchmarks environment variable loading
func BenchmarkEnvironmentVariableLoading(b *testing.B) {
	// Setup comprehensive environment variables
	envVars := map[string]string{
		"NOTIFYHUB_TIMEOUT":         "45s",
		"NOTIFYHUB_ENABLE_METRICS":  "true",
		"NOTIFYHUB_ENABLE_TRACING":  "true",
		"NOTIFYHUB_EMAIL_HOST":      "smtp.benchmark.com",
		"NOTIFYHUB_EMAIL_PORT":      "587",
		"NOTIFYHUB_EMAIL_FROM":      "benchmark@example.com",
		"NOTIFYHUB_EMAIL_USERNAME":  "benchmark_user",
		"NOTIFYHUB_EMAIL_PASSWORD":  "benchmark_password",
		"NOTIFYHUB_FEISHU_WEBHOOK":  "https://benchmark.feishu.webhook",
		"NOTIFYHUB_FEISHU_SECRET":   "benchmark-feishu-secret",
		"NOTIFYHUB_ASYNC_ENABLED":   "true",
		"NOTIFYHUB_QUEUE_TYPE":      "redis",
	}

	// Set environment variables
	for key, value := range envVars {
		os.Setenv(key, value)
	}

	defer func() {
		// Cleanup
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := &Config{
			Platforms: make(map[string]map[string]interface{}),
		}
		err := WithEnvDefaults()(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryUsage benchmarks memory allocation during configuration creation
func BenchmarkMemoryUsage(b *testing.B) {
	opts := []Option{
		WithTimeout(30 * time.Second),
		WithMaxRetries(3),
		WithMetrics(true),
		WithFeishuWebhook("https://benchmark.webhook", "secret"),
		WithEmailBasic("smtp.benchmark.com", 587, "test@benchmark.com"),
		WithSlackWebhook("https://slack.webhook", "#benchmark"),
		WithAsyncBasic("memory", 4),
		WithRouterBasic("round_robin"),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := NewConfig(opts...)
		if err != nil {
			b.Fatal(err)
		}
		_ = cfg
	}
}

// BenchmarkConcurrentConfigCreation benchmarks concurrent configuration creation
func BenchmarkConcurrentConfigCreation(b *testing.B) {
	opts := []Option{
		WithTimeout(30 * time.Second),
		WithFeishuWebhook("https://concurrent.webhook", "secret"),
		WithEmailBasic("smtp.concurrent.com", 587, "test@concurrent.com"),
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cfg, err := NewConfig(opts...)
			if err != nil {
				b.Fatal(err)
			}
			_ = cfg
		}
	})
}