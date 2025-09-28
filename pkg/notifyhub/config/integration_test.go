// Package config integration tests for real-world configuration scenarios
package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealWorldConfigurationScenarios tests practical configuration usage patterns
func TestRealWorldConfigurationScenarios(t *testing.T) {
	t.Run("production alerting system", func(t *testing.T) {
		opts := []Option{
			WithTimeout(60 * time.Second),
			WithMaxRetries(5),
			WithMetrics(true),
			WithTracing(true),
			WithFeishu(FeishuConfig{
				WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/prod-alerts",
				Secret:     "prod-secret-12345",
				AuthType:   "webhook",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				RateLimit:  20, // Conservative for production
				SignVerify: true,
				Keywords:   []string{"[ALERT]", "[CRITICAL]", "[ERROR]"},
			}),
			WithEmail(EmailConfig{
				SMTPHost:     "smtp.company.com",
				SMTPPort:     587,
				SMTPUsername: "alerts@company.com",
				SMTPPassword: "smtp-password",
				SMTPFrom:     "alerts@company.com",
				SMTPTLS:      true,
				Timeout:      45 * time.Second,
				MaxRetries:   5,
				RateLimit:    30, // 30 emails per minute
			}),
			WithSlack(SlackConfig{
				WebhookURL:  "https://hooks.slack.com/services/prod/alerts",
				Channel:     "#alerts-production",
				Username:    "AlertBot",
				IconURL:     "https://company.com/alert-icon.png",
				Timeout:     30 * time.Second,
				MaxRetries:  3,
				RateLimit:   50,
				LinkNames:   true,
				UnfurlLinks: true,
			}),
			WithSMS(SMSConfig{
				Provider:   "twilio",
				APIKey:     "prod-twilio-key",
				APISecret:  "prod-twilio-secret",
				From:       "+1234567890",
				Region:     "us-east-1",
				Timeout:    20 * time.Second,
				MaxRetries: 3,
				RateLimit:  10, // Conservative SMS rate
				Templates: map[string]string{
					"critical": "CRITICAL ALERT: {{message}} - {{timestamp}}",
					"resolved": "RESOLVED: {{message}} - {{timestamp}}",
				},
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
				HealthCheckInterval: 30 * time.Second,
				MaxRetries:          5,
				RetryDelay:          2 * time.Second,
				EnableFailover:      true,
			}),
			WithRetryPolicy(5, 1000, 2.0),
		}

		cfg, err := NewConfig(opts...)
		require.NoError(t, err)

		// Verify production-ready configuration
		assert.Equal(t, 60*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 5, cfg.MaxRetries)
		assert.True(t, cfg.EnableMetrics)
		assert.True(t, cfg.EnableTracing)

		// Verify all platforms are configured
		assert.NotNil(t, cfg.Feishu)
		assert.NotNil(t, cfg.Email)
		assert.NotNil(t, cfg.Slack)
		assert.NotNil(t, cfg.SMS)

		// Verify async processing
		assert.True(t, cfg.AsyncConfig.Enabled)
		assert.Equal(t, "redis", cfg.AsyncConfig.QueueType)
		assert.Equal(t, 16, cfg.AsyncConfig.Workers)

		// Verify router configuration
		assert.Equal(t, "weighted", cfg.RouterConfig.LoadBalancing)
		assert.True(t, cfg.RouterConfig.EnableFailover)

		// Verify platform-specific settings
		assert.True(t, cfg.Feishu.SignVerify)
		assert.Equal(t, 20, cfg.Feishu.RateLimit)
		assert.True(t, cfg.Email.SMTPTLS)
		assert.Equal(t, "#alerts-production", cfg.Slack.Channel)
	})

	t.Run("development environment setup", func(t *testing.T) {
		opts := []Option{
			WithTestDefaults(),
			WithFeishuWebhook("http://localhost:8080/feishu/webhook"),
			WithEmailBasic("localhost", 1025, "dev@localhost"), // MailHog
			WithWebhookBasic("http://localhost:8080/webhook"),
			WithAsyncBasic("memory", 2), // Minimal workers for dev
		}

		cfg, err := NewConfig(opts...)
		require.NoError(t, err)

		// Verify development-friendly settings
		assert.Equal(t, 5*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 1, cfg.MaxRetries)
		assert.False(t, cfg.EnableMetrics)
		assert.False(t, cfg.EnableTracing)

		// Verify local endpoints
		assert.Contains(t, cfg.Feishu.WebhookURL, "localhost")
		assert.Equal(t, "localhost", cfg.Email.SMTPHost)
		assert.Equal(t, 1025, cfg.Email.SMTPPort) // MailHog default port
		assert.Contains(t, cfg.Webhook.URL, "localhost")

		// Verify minimal async setup
		assert.True(t, cfg.AsyncConfig.Enabled)
		assert.Equal(t, "memory", cfg.AsyncConfig.QueueType)
		assert.Equal(t, 2, cfg.AsyncConfig.Workers)
	})

	t.Run("multi-tenant SaaS platform", func(t *testing.T) {
		opts := []Option{
			WithTimeout(30 * time.Second),
			WithMaxRetries(3),
			WithMetrics(true),
			WithTracing(true),
			// Support multiple notification channels
			WithFeishu(FeishuConfig{
				WebhookURL: "https://open.feishu.cn/webhook/tenant-{{tenant_id}}",
				AuthType:   "webhook",
				Timeout:    20 * time.Second,
				MaxRetries: 3,
				RateLimit:  100, // Higher rate for SaaS
			}),
			WithEmail(EmailConfig{
				SMTPHost:     "smtp.sendgrid.net",
				SMTPPort:     587,
				SMTPUsername: "apikey",
				SMTPPassword: "sendgrid-api-key",
				SMTPFrom:     "notifications@saas-platform.com",
				SMTPTLS:      true,
				Timeout:      30 * time.Second,
				MaxRetries:   3,
				RateLimit:    100, // High throughput for SaaS
			}),
			WithSlack(SlackConfig{
				WebhookURL:  "https://hooks.slack.com/services/{{tenant_id}}/{{channel}}",
				Timeout:     25 * time.Second,
				MaxRetries:  3,
				RateLimit:   150,
				LinkNames:   true,
				UnfurlLinks: false, // Disable for privacy
			}),
			WithWebhook(WebhookConfig{
				URL:         "https://api.tenant.com/webhook/{{tenant_id}}",
				Method:      "POST",
				Headers:     map[string]string{"Content-Type": "application/json"},
				AuthType:    "bearer",
				AuthValue:   "{{tenant_api_key}}",
				Timeout:     20 * time.Second,
				MaxRetries:  3,
				RateLimit:   200,
				VerifySSL:   true,
				ContentType: "application/json",
			}),
			WithAsyncConfig(AsyncConfig{
				Enabled:        true,
				QueueType:      "redis",
				QueueSize:      50000, // Large queue for high volume
				Workers:        32,    // Many workers for parallel processing
				BatchSize:      100,   // Large batches for efficiency
				ProcessTimeout: 300 * time.Second,
			}),
			WithRouter(RouterConfig{
				LoadBalancing:       "round_robin",
				HealthCheckInterval: 15 * time.Second, // Frequent health checks
				MaxRetries:          3,
				RetryDelay:          500 * time.Millisecond,
				EnableFailover:      true,
			}),
		}

		cfg, err := NewConfig(opts...)
		require.NoError(t, err)

		// Verify SaaS-optimized configuration
		assert.True(t, cfg.EnableMetrics)
		assert.True(t, cfg.EnableTracing)

		// Verify high-throughput settings
		assert.Equal(t, 100, cfg.Feishu.RateLimit)
		assert.Equal(t, 100, cfg.Email.RateLimit)
		assert.Equal(t, 150, cfg.Slack.RateLimit)
		assert.Equal(t, 200, cfg.Webhook.RateLimit)

		// Verify async processing for high volume
		assert.Equal(t, 50000, cfg.AsyncConfig.QueueSize)
		assert.Equal(t, 32, cfg.AsyncConfig.Workers)
		assert.Equal(t, 100, cfg.AsyncConfig.BatchSize)

		// Verify template support for multi-tenancy
		assert.Contains(t, cfg.Feishu.WebhookURL, "{{tenant_id}}")
		assert.Contains(t, cfg.Webhook.URL, "{{tenant_id}}")
	})

	t.Run("minimal notification service", func(t *testing.T) {
		opts := []Option{
			WithFeishuWebhook("https://open.feishu.cn/webhook/simple"),
		}

		cfg, err := NewConfig(opts...)
		require.NoError(t, err)

		// Verify minimal but functional configuration
		assert.Equal(t, 30*time.Second, cfg.DefaultTimeout) // Default
		assert.Equal(t, 3, cfg.MaxRetries)                  // Default
		assert.NotNil(t, cfg.Feishu)
		assert.Equal(t, "webhook", cfg.Feishu.AuthType)

		// Verify defaults are reasonable
		assert.Equal(t, "round_robin", cfg.RouterConfig.LoadBalancing)
		assert.Equal(t, "memory", cfg.AsyncConfig.QueueType)
		assert.Equal(t, 4, cfg.AsyncConfig.Workers)
	})

	t.Run("high-availability enterprise setup", func(t *testing.T) {
		opts := []Option{
			WithTimeout(120 * time.Second), // Long timeout for reliability
			WithMaxRetries(10),             // High retry count
			WithMetrics(true),
			WithTracing(true),
			WithFeishu(FeishuConfig{
				WebhookURL: "https://enterprise.feishu.cn/webhook/ha-primary",
				Secret:     "enterprise-secret",
				AuthType:   "webhook",
				Timeout:    60 * time.Second,
				MaxRetries: 5,
				RateLimit:  50, // Conservative for enterprise
				SignVerify: true,
			}),
			WithEmail(EmailConfig{
				SMTPHost:     "smtp.enterprise.com",
				SMTPPort:     587,
				SMTPUsername: "ha-service@enterprise.com",
				SMTPPassword: "enterprise-smtp-password",
				SMTPFrom:     "alerts@enterprise.com",
				SMTPTLS:      true,
				SMTPSSL:      false,
				Timeout:      90 * time.Second,
				MaxRetries:   8,
				RateLimit:    20, // Conservative rate
			}),
			WithAsyncConfig(AsyncConfig{
				Enabled:        true,
				QueueType:      "redis",
				QueueSize:      100000, // Very large queue
				Workers:        64,     // Many workers for redundancy
				BatchSize:      25,     // Smaller batches for reliability
				ProcessTimeout: 600 * time.Second,
			}),
			WithRouter(RouterConfig{
				LoadBalancing:       "weighted",
				HealthCheckInterval: 10 * time.Second, // Frequent checks
				MaxRetries:          10,               // High retry count
				RetryDelay:          5 * time.Second,  // Longer delay between retries
				EnableFailover:      true,
			}),
			WithRetryPolicy(10, 2000, 1.5), // Conservative exponential backoff
		}

		cfg, err := NewConfig(opts...)
		require.NoError(t, err)

		// Verify enterprise-grade settings
		assert.Equal(t, 120*time.Second, cfg.DefaultTimeout)
		assert.Equal(t, 10, cfg.MaxRetries)
		assert.Equal(t, 10, cfg.RouterConfig.MaxRetries)
		assert.Equal(t, 10, cfg.RetryPolicy.MaxRetries)

		// Verify conservative rate limits
		assert.Equal(t, 50, cfg.Feishu.RateLimit)
		assert.Equal(t, 20, cfg.Email.RateLimit)

		// Verify high-capacity async processing
		assert.Equal(t, 100000, cfg.AsyncConfig.QueueSize)
		assert.Equal(t, 64, cfg.AsyncConfig.Workers)

		// Verify security settings
		assert.True(t, cfg.Feishu.SignVerify)
		assert.True(t, cfg.Email.SMTPTLS)
	})
}

// TestConfigurationMigrationScenarios tests migration from legacy configurations
func TestConfigurationMigrationScenarios(t *testing.T) {
	t.Run("migrate from legacy map-based config", func(t *testing.T) {
		// Simulate legacy configuration pattern
		legacyOpts := []Option{
			WithPlatform("feishu", map[string]interface{}{
				"webhook_url": "https://legacy.feishu.webhook",
				"secret":      "legacy-secret",
				"timeout":     "30s",
			}),
			WithPlatform("email", map[string]interface{}{
				"smtp_host": "smtp.legacy.com",
				"smtp_port": 587,
				"smtp_from": "legacy@example.com",
				"smtp_tls":  true,
			}),
		}

		// Create config with legacy options
		legacyCfg, err := NewConfigWithoutValidation(legacyOpts...)
		require.NoError(t, err)

		// Verify legacy platforms are accessible
		feishuLegacy := legacyCfg.GetPlatformConfig("feishu")
		assert.NotNil(t, feishuLegacy)
		assert.Equal(t, "https://legacy.feishu.webhook", feishuLegacy["webhook_url"])
		assert.Equal(t, "legacy-secret", feishuLegacy["secret"])

		emailLegacy := legacyCfg.GetPlatformConfig("email")
		assert.NotNil(t, emailLegacy)
		assert.Equal(t, "smtp.legacy.com", emailLegacy["smtp_host"])
		assert.Equal(t, 587, emailLegacy["smtp_port"])

		// Now migrate to strong-typed configuration
		modernOpts := []Option{
			WithFeishuWebhook("https://legacy.feishu.webhook", "legacy-secret"),
			WithEmailBasic("smtp.legacy.com", 587, "legacy@example.com"),
		}

		modernCfg, err := NewConfig(modernOpts...)
		require.NoError(t, err)

		// Verify modern configuration provides same functionality
		assert.Equal(t, "https://legacy.feishu.webhook", modernCfg.Feishu.WebhookURL)
		assert.Equal(t, "legacy-secret", modernCfg.Feishu.Secret)
		assert.Equal(t, "smtp.legacy.com", modernCfg.Email.SMTPHost)
		assert.Equal(t, 587, modernCfg.Email.SMTPPort)

		// Verify modern config can be accessed via legacy interface
		feishuModern := modernCfg.GetPlatformConfig("feishu")
		assert.Equal(t, feishuLegacy["webhook_url"], feishuModern["webhook_url"])
		assert.Equal(t, feishuLegacy["secret"], feishuModern["secret"])
	})

	t.Run("gradual migration with mixed configurations", func(t *testing.T) {
		// Simulate partial migration - some platforms migrated, others still legacy
		opts := []Option{
			// Modern strong-typed configurations
			WithFeishuWebhook("https://modern.feishu.webhook", "modern-secret"),
			WithEmailBasic("smtp.modern.com", 587, "modern@example.com"),

			// Legacy map-based configurations
			WithPlatform("custom-platform", map[string]interface{}{
				"endpoint": "https://custom.api.com",
				"api_key":  "custom-key",
				"timeout":  "45s",
			}),
			WithPlatform("legacy-webhook", map[string]interface{}{
				"url":    "https://legacy.webhook.com",
				"method": "POST",
				"headers": map[string]string{
					"Authorization": "Bearer legacy-token",
				},
			}),
		}

		cfg, err := NewConfig(opts...)
		require.NoError(t, err)

		// Verify modern platforms work with strong typing
		assert.NotNil(t, cfg.Feishu)
		assert.Equal(t, "https://modern.feishu.webhook", cfg.Feishu.WebhookURL)
		assert.NotNil(t, cfg.Email)
		assert.Equal(t, "smtp.modern.com", cfg.Email.SMTPHost)

		// Verify legacy platforms still accessible via GetPlatformConfig
		customPlatform := cfg.GetPlatformConfig("custom-platform")
		assert.NotNil(t, customPlatform)
		assert.Equal(t, "https://custom.api.com", customPlatform["endpoint"])

		legacyWebhook := cfg.GetPlatformConfig("legacy-webhook")
		assert.NotNil(t, legacyWebhook)
		assert.Equal(t, "https://legacy.webhook.com", legacyWebhook["url"])

		// Verify modern platforms also accessible via legacy interface
		modernFeishu := cfg.GetPlatformConfig("feishu")
		assert.Equal(t, "https://modern.feishu.webhook", modernFeishu["webhook_url"])
		assert.Equal(t, "modern-secret", modernFeishu["secret"])
	})
}

// TestEnvironmentConfigurationIntegration tests environment-based configuration in realistic scenarios
func TestEnvironmentConfigurationIntegration(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envKeys := []string{
		"NOTIFYHUB_TIMEOUT", "NOTIFYHUB_ENABLE_METRICS", "NOTIFYHUB_ENABLE_TRACING",
		"NOTIFYHUB_EMAIL_HOST", "NOTIFYHUB_EMAIL_PORT", "NOTIFYHUB_EMAIL_FROM",
		"NOTIFYHUB_EMAIL_USERNAME", "NOTIFYHUB_EMAIL_PASSWORD",
		"NOTIFYHUB_FEISHU_WEBHOOK", "NOTIFYHUB_FEISHU_SECRET",
		"NOTIFYHUB_ASYNC_ENABLED", "NOTIFYHUB_QUEUE_TYPE",
	}

	for _, key := range envKeys {
		originalEnv[key] = os.Getenv(key)
	}

	cleanup := func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}
	defer cleanup()

	t.Run("container deployment with environment variables", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		// Set container-style environment variables
		containerEnv := map[string]string{
			"NOTIFYHUB_TIMEOUT":         "60s",
			"NOTIFYHUB_ENABLE_METRICS":  "true",
			"NOTIFYHUB_ENABLE_TRACING":  "true",
			"NOTIFYHUB_EMAIL_HOST":      "smtp.mailgun.org",
			"NOTIFYHUB_EMAIL_PORT":      "587",
			"NOTIFYHUB_EMAIL_FROM":      "noreply@app.com",
			"NOTIFYHUB_EMAIL_USERNAME":  "postmaster@mg.app.com",
			"NOTIFYHUB_EMAIL_PASSWORD":  "mailgun-password",
			"NOTIFYHUB_FEISHU_WEBHOOK":  "https://open.feishu.cn/webhook/container-token",
			"NOTIFYHUB_FEISHU_SECRET":   "container-secret",
			"NOTIFYHUB_ASYNC_ENABLED":   "true",
			"NOTIFYHUB_QUEUE_TYPE":      "redis",
		}

		for key, value := range containerEnv {
			os.Setenv(key, value)
		}

		// Create configuration primarily from environment
		cfg, err := NewConfig(
			WithEnvDefaults(),
			// Override some settings for container environment
			WithAsyncBasic("redis", 8),
			WithRouterBasic("round_robin"),
		)
		require.NoError(t, err)

		// Verify environment variables are loaded
		assert.Equal(t, 60*time.Second, cfg.DefaultTimeout)
		assert.True(t, cfg.EnableMetrics)
		assert.True(t, cfg.EnableTracing)

		// Verify email configuration from environment
		require.NotNil(t, cfg.Email)
		assert.Equal(t, "smtp.mailgun.org", cfg.Email.SMTPHost)
		assert.Equal(t, 587, cfg.Email.SMTPPort)
		assert.Equal(t, "noreply@app.com", cfg.Email.SMTPFrom)
		assert.Equal(t, "postmaster@mg.app.com", cfg.Email.SMTPUsername)
		assert.Equal(t, "mailgun-password", cfg.Email.SMTPPassword)

		// Verify Feishu configuration from environment
		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, "https://open.feishu.cn/webhook/container-token", cfg.Feishu.WebhookURL)
		assert.Equal(t, "container-secret", cfg.Feishu.Secret)
		assert.True(t, cfg.Feishu.SignVerify)

		// Verify async configuration
		assert.True(t, cfg.AsyncConfig.Enabled)
		assert.Equal(t, "redis", cfg.AsyncConfig.QueueType)
		assert.Equal(t, 8, cfg.AsyncConfig.Workers) // From explicit option
	})

	t.Run("kubernetes deployment with secrets and configmaps", func(t *testing.T) {
		// Clean environment
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		// Simulate Kubernetes environment with secrets mounted as env vars
		k8sEnv := map[string]string{
			// From ConfigMap
			"NOTIFYHUB_TIMEOUT":        "45s",
			"NOTIFYHUB_ENABLE_METRICS": "true",
			"NOTIFYHUB_ASYNC_ENABLED":  "true",
			"NOTIFYHUB_QUEUE_TYPE":     "redis",

			// From Secrets
			"NOTIFYHUB_EMAIL_HOST":      "smtp.k8s-mailservice",
			"NOTIFYHUB_EMAIL_USERNAME":  "k8s-service-account",
			"NOTIFYHUB_EMAIL_PASSWORD":  "k8s-secret-password",
			"NOTIFYHUB_EMAIL_FROM":      "alerts@k8s-cluster.com",
			"NOTIFYHUB_FEISHU_WEBHOOK":  "https://open.feishu.cn/webhook/k8s-cluster",
			"NOTIFYHUB_FEISHU_SECRET":   "k8s-feishu-secret",
		}

		for key, value := range k8sEnv {
			os.Setenv(key, value)
		}

		cfg, err := NewConfig(
			WithEnvDefaults(),
			// Add cluster-specific settings
			WithTimeout(45 * time.Second), // Explicit override
			WithAsyncBasic("redis", 12),   // Cluster-optimized workers
		)
		require.NoError(t, err)

		// Verify Kubernetes environment integration
		assert.Equal(t, 45*time.Second, cfg.DefaultTimeout)
		assert.True(t, cfg.EnableMetrics)

		// Verify secrets are loaded correctly
		require.NotNil(t, cfg.Email)
		assert.Equal(t, "smtp.k8s-mailservice", cfg.Email.SMTPHost)
		assert.Equal(t, "k8s-service-account", cfg.Email.SMTPUsername)
		assert.Equal(t, "k8s-secret-password", cfg.Email.SMTPPassword)

		require.NotNil(t, cfg.Feishu)
		assert.Equal(t, "https://open.feishu.cn/webhook/k8s-cluster", cfg.Feishu.WebhookURL)
		assert.Equal(t, "k8s-feishu-secret", cfg.Feishu.Secret)

		// Verify async configuration for cluster environment
		assert.True(t, cfg.AsyncConfig.Enabled)
		assert.Equal(t, "redis", cfg.AsyncConfig.QueueType)
		assert.Equal(t, 12, cfg.AsyncConfig.Workers)
	})
}

// TestConfigurationErrorHandlingIntegration tests error scenarios in realistic configurations
func TestConfigurationErrorHandlingIntegration(t *testing.T) {
	t.Run("partial configuration failures", func(t *testing.T) {
		opts := []Option{
			WithTimeout(30 * time.Second),  // Valid
			WithMaxRetries(3),              // Valid
			WithFeishuWebhook("invalid-url", "secret"), // Invalid URL
			WithEmailBasic("valid.smtp.com", 587, "invalid-email"), // Invalid email
		}

		cfg, err := NewConfig(opts...)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "configuration validation failed")
	})

	t.Run("environment variable parsing errors", func(t *testing.T) {
		// Clean environment
		os.Unsetenv("NOTIFYHUB_TIMEOUT")
		os.Unsetenv("NOTIFYHUB_EMAIL_PORT")

		// Set invalid environment variables
		os.Setenv("NOTIFYHUB_TIMEOUT", "invalid-duration")
		os.Setenv("NOTIFYHUB_EMAIL_HOST", "smtp.test.com")
		os.Setenv("NOTIFYHUB_EMAIL_PORT", "invalid-port")

		defer func() {
			os.Unsetenv("NOTIFYHUB_TIMEOUT")
			os.Unsetenv("NOTIFYHUB_EMAIL_HOST")
			os.Unsetenv("NOTIFYHUB_EMAIL_PORT")
		}()

		cfg, err := NewConfigWithoutValidation(WithEnvDefaults())
		require.NoError(t, err)

		// Invalid timeout should fall back to default
		assert.Equal(t, 30*time.Second, cfg.DefaultTimeout)

		// Invalid port should remain 0
		require.NotNil(t, cfg.Email)
		assert.Equal(t, "smtp.test.com", cfg.Email.SMTPHost)
		assert.Equal(t, 0, cfg.Email.SMTPPort)
	})

	t.Run("mixed valid and invalid platforms", func(t *testing.T) {
		opts := []Option{
			WithFeishuWebhook("https://valid.feishu.webhook", "secret"), // Valid
			WithEmailBasic("", 587, "test@example.com"),                 // Invalid - empty host
			WithSlackWebhook("https://valid.slack.webhook"),             // Valid
		}

		cfg, err := NewConfig(opts...)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})
}