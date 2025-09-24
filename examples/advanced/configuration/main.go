// Package main demonstrates advanced configuration patterns
// This shows enterprise-ready configuration management for NotifyHub
package main

import (
	"os"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/email"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
	"github.com/kart-io/notifyhub/pkg/platforms/sms"
)

var log = logger.New().LogMode(logger.Info)

func main() {

	log.Info("⚙️  Advanced Configuration Patterns Demo")
	log.Info("======================================")

	// Part 1: Environment-Based Configuration
	log.Info("🌍 Part 1: Environment-Based Configuration")
	log.Info("----------------------------------------")

	// Production configuration
	prodHub := createProductionHub()
	if prodHub != nil {
		defer func() { _ = prodHub.Close() }()
		log.Info("✅ Production hub configured")
	}

	// Development configuration
	devHub := createDevelopmentHub()
	if devHub != nil {
		defer func() { _ = devHub.Close() }()
		log.Info("✅ Development hub configured")
	}

	// Test configuration
	testHub := createTestHub()
	if testHub != nil {
		defer func() { _ = testHub.Close() }()
		log.Info("✅ Test hub configured")
	}

	// Part 2: Configuration Builders
	log.Info("🏗️  Part 2: Configuration Builders")
	log.Info("--------------------------------")

	// Fluent configuration builder (conceptual example)
	builderHub := NewConfigBuilder().
		WithFeishu("https://example.com/feishu/webhook").
		WithFeishuSecret("prod-secret").
		WithFeishuTimeout(30*time.Second).
		WithEmail("smtp.company.com", 587, "notifications@company.com").
		WithEmailAuth("smtp-user", "smtp-pass").
		WithEmailTLS(true).
		WithSMSTwilio("twilio-key", "+1234567890").
		WithSMSTimeout(20 * time.Second).
		Build()

	if builderHub != nil {
		defer func() { _ = builderHub.Close() }()
		log.Info("✅ Builder-pattern hub created")
	}

	// Part 3: Configuration Validation
	log.Info("🔍 Part 3: Configuration Validation")
	log.Info("---------------------------------")

	// Valid configuration
	validConfig := map[string]interface{}{
		"feishu": map[string]interface{}{
			"webhook_url": "https://example.com/webhook",
			"secret":      "valid-secret",
			"timeout":     30 * time.Second,
		},
		"email": map[string]interface{}{
			"smtp_host": "smtp.company.com",
			"smtp_port": 587,
			"smtp_from": "notifications@company.com",
		},
	}

	if validateConfiguration(validConfig) {
		log.Info("✅ Configuration validation passed")
	} else {
		log.Info("❌ Configuration validation failed")
	}

	// Invalid configuration
	invalidConfig := map[string]interface{}{
		"feishu": map[string]interface{}{
			"webhook_url": "", // Invalid: empty URL
			"secret":      "secret",
		},
	}

	if validateConfiguration(invalidConfig) {
		log.Info("✅ Invalid config unexpectedly passed")
	} else {
		log.Info("❌ Invalid configuration correctly rejected")
	}

	// Part 4: Configuration Templates
	log.Info("📋 Part 4: Configuration Templates")
	log.Info("--------------------------------")

	// Team communication template
	if createTeamTemplate() != nil {
		log.Info("✅ Team communication template created")
	}

	// Operations alert template
	if createOpsTemplate() != nil {
		log.Info("✅ Operations alert template created")
	}

	// Customer notification template
	if createCustomerTemplate() != nil {
		log.Info("✅ Customer notification template created")
	}

	// Part 5: Dynamic Configuration
	log.Info("🔄 Part 5: Dynamic Configuration")
	log.Info("------------------------------")

	// Configuration that changes based on conditions
	dynamicHub := createDynamicConfiguration()
	if dynamicHub != nil {
		defer func() { _ = dynamicHub.Close() }()
		log.Info("✅ Dynamic configuration created")
	}

	// Summary
	log.Info("⚙️  Configuration Best Practices")
	log.Info("==============================")
	log.Info("✅ ENVIRONMENT SEPARATION:")
	log.Info("   • Different configs for prod/dev/test")
	log.Info("   • Environment variable usage")
	log.Info("   • Secure credential management")
	log.Info("✅ CONFIGURATION VALIDATION:")
	log.Info("   • Early validation prevents runtime errors")
	log.Info("   • Clear error messages for debugging")
	log.Info("   • Type safety and required field checks")
	log.Info("✅ CONFIGURATION TEMPLATES:")
	log.Info("   • Reusable configuration patterns")
	log.Info("   • Team-specific setups")
	log.Info("   • Use case optimizations")
	log.Info("✅ DYNAMIC CONFIGURATION:")
	log.Info("   • Runtime configuration changes")
	log.Info("   • Conditional platform selection")
	log.Info("   • Feature flag integration")

	log.Info("⚙️  Advanced Configuration Demo Complete!")
}

// Environment-based configuration functions

func createProductionHub() notifyhub.Client {
	// Production uses environment variables for security
	feishuWebhook := os.Getenv("FEISHU_WEBHOOK_URL")
	feishuSecret := os.Getenv("FEISHU_SECRET")
	emailHost := os.Getenv("SMTP_HOST")
	emailUser := os.Getenv("SMTP_USER")
	emailPass := os.Getenv("SMTP_PASS")
	twilioKey := os.Getenv("TWILIO_API_KEY")

	if feishuWebhook == "" {
		feishuWebhook = "https://example.com/feishu/webhook" // Fallback for demo
	}

	hub, err := notifyhub.New(
		feishu.WithFeishu(feishuWebhook,
			feishu.WithFeishuSecret(feishuSecret),
			feishu.WithFeishuTimeout(30*time.Second),
		),
		email.WithEmail(getOrDefault(emailHost, "smtp.company.com"), 587,
			"notifications@company.com",
			email.WithEmailAuth(emailUser, emailPass),
			email.WithEmailTLS(true),
			email.WithEmailTimeout(45*time.Second),
		),
		sms.WithSMSTwilio(getOrDefault(twilioKey, "demo-key"), "+1234567890",
			sms.WithSMSTimeout(30*time.Second),
		),
	)

	if err != nil {
		log.Error("Failed to create production hub", "error", err)
		return nil
	}
	return hub
}

func createDevelopmentHub() notifyhub.Client {
	// Development uses relaxed settings and mock endpoints
	hub, err := notifyhub.New(
		feishu.WithFeishu("https://dev.example.com/feishu/webhook",
			feishu.WithFeishuTimeout(60*time.Second), // Longer timeout for debugging
		),
		email.WithEmail("localhost", 1025, "dev@example.com", // MailHog for local dev
			email.WithEmailTLS(false),
			email.WithEmailTimeout(10*time.Second),
		),
	)

	if err != nil {
		log.Error("Failed to create development hub", "error", err)
		return nil
	}
	return hub
}

func createTestHub() notifyhub.Client {
	// Test uses mock configurations
	hub, err := notifyhub.New(
		feishu.WithFeishu("https://test.example.com/webhook"),
		email.WithEmail("test-smtp", 587, "test@example.com"),
	)

	if err != nil {
		log.Error("Failed to create test hub", "error", err)
		return nil
	}
	return hub
}

// Configuration builder extension (conceptual)
type ConfigBuilder struct {
	feishuConfig map[string]interface{}
	emailConfig  map[string]interface{}
	smsConfig    map[string]interface{}
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		feishuConfig: make(map[string]interface{}),
		emailConfig:  make(map[string]interface{}),
		smsConfig:    make(map[string]interface{}),
	}
}

func (c *ConfigBuilder) WithFeishu(webhookURL string) *ConfigBuilder {
	c.feishuConfig["webhook_url"] = webhookURL
	return c
}

func (c *ConfigBuilder) WithFeishuSecret(secret string) *ConfigBuilder {
	c.feishuConfig["secret"] = secret
	return c
}

func (c *ConfigBuilder) WithFeishuTimeout(timeout time.Duration) *ConfigBuilder {
	c.feishuConfig["timeout"] = timeout
	return c
}

func (c *ConfigBuilder) WithEmail(host string, port int, from string) *ConfigBuilder {
	c.emailConfig["smtp_host"] = host
	c.emailConfig["smtp_port"] = port
	c.emailConfig["smtp_from"] = from
	return c
}

func (c *ConfigBuilder) WithEmailAuth(username, password string) *ConfigBuilder {
	c.emailConfig["smtp_username"] = username
	c.emailConfig["smtp_password"] = password
	return c
}

func (c *ConfigBuilder) WithEmailTLS(useTLS bool) *ConfigBuilder {
	c.emailConfig["smtp_tls"] = useTLS
	return c
}

func (c *ConfigBuilder) WithSMSTwilio(apiKey, from string) *ConfigBuilder {
	c.smsConfig["provider"] = "twilio"
	c.smsConfig["api_key"] = apiKey
	c.smsConfig["from"] = from
	return c
}

func (c *ConfigBuilder) WithSMSTimeout(timeout time.Duration) *ConfigBuilder {
	c.smsConfig["timeout"] = timeout
	return c
}

func (c *ConfigBuilder) Build() notifyhub.Client {
	var options []notifyhub.HubOption

	if len(c.feishuConfig) > 0 {
		options = append(options, feishu.WithFeishu(
			c.feishuConfig["webhook_url"].(string),
			func(config map[string]interface{}) {
				for k, v := range c.feishuConfig {
					if k != "webhook_url" {
						config[k] = v
					}
				}
			},
		))
	}

	if len(c.emailConfig) > 0 {
		options = append(options, email.WithEmail(
			c.emailConfig["smtp_host"].(string),
			c.emailConfig["smtp_port"].(int),
			c.emailConfig["smtp_from"].(string),
			func(config map[string]interface{}) {
				for k, v := range c.emailConfig {
					if k != "smtp_host" && k != "smtp_port" && k != "smtp_from" {
						config[k] = v
					}
				}
			},
		))
	}

	if len(c.smsConfig) > 0 {
		options = append(options, sms.WithSMS(
			c.smsConfig["provider"].(string),
			c.smsConfig["api_key"].(string),
			c.smsConfig["from"].(string),
			func(config map[string]interface{}) {
				for k, v := range c.smsConfig {
					if k != "provider" && k != "api_key" && k != "from" {
						config[k] = v
					}
				}
			},
		))
	}

	hub, err := notifyhub.New(options...)
	if err != nil {
		log.Error("Failed to build hub", "error", err)
		return nil
	}
	return hub
}

// Configuration validation
func validateConfiguration(config map[string]interface{}) bool {

	// Validate Feishu configuration
	if feishuConfig, ok := config["feishu"].(map[string]interface{}); ok {
		if webhookURL, exists := feishuConfig["webhook_url"].(string); !exists || webhookURL == "" {
			log.Error("   ❌ Feishu webhook_url is required and cannot be empty")
			return false
		}
	}

	// Validate Email configuration
	if emailConfig, ok := config["email"].(map[string]interface{}); ok {
		if host, exists := emailConfig["smtp_host"].(string); !exists || host == "" {
			log.Error("   ❌ Email smtp_host is required")
			return false
		}
		if _, exists := emailConfig["smtp_port"].(int); !exists {
			log.Error("   ❌ Email smtp_port is required")
			return false
		}
	}

	return true
}

// Configuration templates
func createTeamTemplate() notifyhub.Client {
	// Team communication focused configuration
	hub, _ := notifyhub.New(
		feishu.WithFeishu("https://team.example.com/webhook",
			feishu.WithFeishuTimeout(15*time.Second), // Fast for team communication
		),
	)
	return hub
}

func createOpsTemplate() notifyhub.Client {
	// Operations alerts with multiple channels
	hub, _ := notifyhub.New(
		feishu.WithFeishu("https://ops.example.com/webhook"),
		email.WithEmail("smtp.company.com", 587, "ops@company.com"),
		sms.WithSMSTwilio("ops-twilio-key", "+1234567890"),
	)
	return hub
}

func createCustomerTemplate() notifyhub.Client {
	// Customer notifications (email + SMS)
	hub, _ := notifyhub.New(
		email.WithEmail("smtp.mailgun.com", 587, "noreply@company.com",
			email.WithEmailAuth("api", "mailgun-key"),
		),
		sms.WithSMSTwilio("customer-twilio-key", "+1234567890"),
	)
	return hub
}

// Dynamic configuration
func createDynamicConfiguration() notifyhub.Client {
	environment := getOrDefault(os.Getenv("ENVIRONMENT"), "development")

	var options []notifyhub.HubOption

	// Always include Feishu for team communication
	options = append(options, feishu.WithFeishu("https://example.com/webhook"))

	// Production gets all platforms
	if environment == "production" {
		options = append(options,
			email.WithEmail("smtp.company.com", 587, "prod@company.com"),
			sms.WithSMSTwilio("prod-twilio", "+1234567890"),
		)
	}

	// Development gets limited platforms
	if environment == "development" {
		options = append(options,
			email.WithEmail("localhost", 1025, "dev@example.com"),
		)
	}

	hub, err := notifyhub.New(options...)
	if err != nil {
		log.Error("Failed to create dynamic hub", "error", err)
		return nil
	}
	return hub
}

// Utility functions
func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
