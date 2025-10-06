// Package main demonstrates how to implement an external SMS platform
// This example shows how to extend NotifyHub with SMS capabilities
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart/notifyhub/examples/external-platform-sms/sms"
	"github.com/kart/notifyhub/pkg/config"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/notifyhub"
	"github.com/kart/notifyhub/pkg/receipt"
	"github.com/kart/notifyhub/pkg/target"
)

func main() {
	fmt.Println("📱 NotifyHub External Platform Demo - SMS")
	fmt.Println("=========================================")

	// Display demo modes
	fmt.Println("\nThis demo shows SMS platform capabilities:")
	fmt.Println("1. 🔧 Standalone SMS Platform - Works independently")
	fmt.Println("2. 🏢 Multiple Provider Support - Aliyun, Tencent, Twilio, Nexmo")
	fmt.Println("3. 🚦 Rate Limiting - Protect against spam")
	fmt.Println("4. 📋 Template Support - SMS templates with variables")
	fmt.Println("5. 🔌 Integration Demo - NotifyHub integration concept")

	// Run standalone demos
	runStandaloneDemos()

	// Run integration demo
	runIntegrationDemo()
}

// runStandaloneDemos demonstrates SMS platform working independently
func runStandaloneDemos() {
	fmt.Println("\n🔧 Standalone SMS Platform Demos")
	fmt.Println("=================================")

	// Demo 1: Basic SMS with Mock Provider
	fmt.Println("\n📤 Demo 1: Basic SMS with Mock Provider")
	demoBasicSMS()

	// Demo 2: Multiple Providers
	fmt.Println("\n🏢 Demo 2: Multiple SMS Providers")
	demoMultipleProviders()

	// Demo 3: Rate Limiting
	fmt.Println("\n🚦 Demo 3: Rate Limiting")
	demoRateLimiting()

	// Demo 4: Template SMS
	fmt.Println("\n📋 Demo 4: Template SMS")
	demoTemplateSMS()

	// Demo 5: Error Handling
	fmt.Println("\n❌ Demo 5: Error Handling")
	demoErrorHandling()
}

// demoBasicSMS demonstrates basic SMS sending
func demoBasicSMS() {
	// Create SMS platform with mock provider
	config := sms.Config{
		Provider: sms.ProviderMock,
		Credentials: map[string]string{
			"should_fail": "false",
			"delay":       "500ms",
		},
		Timeout: 30,
	}

	platform, err := sms.New(config)
	if err != nil {
		log.Printf("Failed to create SMS platform: %v", err)
		return
	}
	defer func() { _ = platform.Close() }()

	fmt.Printf("✅ SMS platform created: %s\n", platform.Name())

	// Test platform capabilities
	capabilities := platform.GetCapabilities()
	fmt.Printf("📋 Capabilities: %+v\n", capabilities)

	// Create test message
	msg := message.New()
	msg.Title = "NotifyHub"
	msg.Body = "这是一条测试短信，演示外部SMS平台功能。"
	msg.Format = message.FormatText

	// Create targets
	targets := []target.Target{
		sms.CreateTarget("+86 138 0013 8000"),
		sms.CreateMobileTarget("+1 555 123 4567"),
	}

	// Send SMS
	ctx := context.Background()
	results, err := platform.Send(ctx, msg, targets)
	if err != nil {
		log.Printf("Failed to send SMS: %v", err)
		return
	}

	// Process results
	fmt.Println("📊 Send results:")
	for i, result := range results {
		if result.Success {
			fmt.Printf("✅ Target %d (%s): Success - %s\n", i+1, result.Target.Value, result.Response)
		} else {
			fmt.Printf("❌ Target %d (%s): Failed - %v\n", i+1, result.Target.Value, result.Error)
		}
	}
}

// demoMultipleProviders demonstrates different SMS providers
func demoMultipleProviders() {
	providers := []struct {
		name        string
		provider    sms.Provider
		credentials map[string]string
	}{
		{
			name:     "阿里云短信",
			provider: sms.ProviderAliyun,
			credentials: map[string]string{
				"access_key_id":     "LTAI_test_key",
				"access_key_secret": "test_secret",
				"sign_name":         "NotifyHub",
			},
		},
		{
			name:     "腾讯云短信",
			provider: sms.ProviderTencent,
			credentials: map[string]string{
				"secret_id":  "AKID_test_id",
				"secret_key": "test_key",
				"app_id":     "1400123456",
				"sign_name":  "NotifyHub",
			},
		},
		{
			name:     "Twilio",
			provider: sms.ProviderTwilio,
			credentials: map[string]string{
				"account_sid": "AC_test_sid",
				"auth_token":  "test_token",
				"from_number": "+1234567890",
			},
		},
	}

	for _, p := range providers {
		fmt.Printf("\n🔍 Testing %s:\n", p.name)

		config := sms.Config{
			Provider:    p.provider,
			Credentials: p.credentials,
			Timeout:     30,
		}

		platform, err := sms.New(config)
		if err != nil {
			fmt.Printf("❌ Failed to create %s platform: %v\n", p.name, err)
			continue
		}

		// Test health
		ctx := context.Background()
		if err := platform.IsHealthy(ctx); err != nil {
			fmt.Printf("⚠️  %s health check failed: %v\n", p.name, err)
		} else {
			fmt.Printf("✅ %s is healthy\n", p.name)
		}

		// Test sending
		msg := message.New()
		msg.Body = fmt.Sprintf("来自%s的测试短信", p.name)
		targets := []target.Target{sms.CreateTarget("+86 138 0013 8000")}

		results, err := platform.Send(ctx, msg, targets)
		if err != nil {
			fmt.Printf("❌ %s send failed: %v\n", p.name, err)
		} else if len(results) > 0 && results[0].Success {
			fmt.Printf("✅ %s send success: %s\n", p.name, results[0].Response)
		} else if len(results) > 0 {
			fmt.Printf("❌ %s send failed: %v\n", p.name, results[0].Error)
		}

		_ = platform.Close()
	}
}

// demoRateLimiting demonstrates SMS rate limiting
func demoRateLimiting() {
	// Create SMS platform with rate limiting
	config := sms.Config{
		Provider: sms.ProviderMock,
		Credentials: map[string]string{
			"should_fail": "false",
			"delay":       "100ms",
		},
		RateLimit: sms.RateLimitConfig{
			Enabled:    true,
			MaxPerHour: 3,  // 每小时最多3条
			MaxPerDay:  10, // 每天最多10条
		},
		Timeout: 30,
	}

	platform, err := sms.New(config)
	if err != nil {
		log.Printf("Failed to create SMS platform: %v", err)
		return
	}
	defer func() { _ = platform.Close() }()

	fmt.Printf("✅ SMS platform with rate limiting created\n")

	// Test multiple sends to same number
	msg := message.New()
	msg.Body = "限流测试短信"
	phone := "+86 138 0013 8000"
	targets := []target.Target{sms.CreateTarget(phone)}

	ctx := context.Background()

	// Send 5 messages to test rate limiting
	for i := 1; i <= 5; i++ {
		fmt.Printf("\n📤 尝试发送第 %d 条短信到 %s:\n", i, phone)

		results, err := platform.Send(ctx, msg, targets)
		if err != nil {
			fmt.Printf("❌ 发送失败: %v\n", err)
			continue
		}

		if len(results) > 0 {
			if results[0].Success {
				fmt.Printf("✅ 发送成功: %s\n", results[0].Response)
			} else {
				fmt.Printf("❌ 被限流: %v\n", results[0].Error)
			}
		}

		time.Sleep(200 * time.Millisecond)
	}
}

// demoTemplateSMS demonstrates template-based SMS
func demoTemplateSMS() {
	// Create SMS platform with templates
	config := sms.Config{
		Provider: sms.ProviderMock,
		Credentials: map[string]string{
			"should_fail": "false",
		},
		Templates: map[string]string{
			"verification": "您的验证码是{{code}}，请在{{minutes}}分钟内使用。",
			"welcome":      "欢迎{{name}}注册NotifyHub！",
			"notification": "{{title}}: {{content}}",
		},
		Timeout: 30,
	}

	platform, err := sms.New(config)
	if err != nil {
		log.Printf("Failed to create SMS platform: %v", err)
		return
	}
	defer func() { _ = platform.Close() }()

	fmt.Printf("✅ SMS platform with templates created\n")

	// Test different templates
	templates := []struct {
		name      string
		template  string
		variables map[string]interface{}
	}{
		{
			name:     "验证码短信",
			template: "verification",
			variables: map[string]interface{}{
				"code":    "123456",
				"minutes": "5",
			},
		},
		{
			name:     "欢迎短信",
			template: "welcome",
			variables: map[string]interface{}{
				"name": "张三",
			},
		},
		{
			name:     "通知短信",
			template: "notification",
			variables: map[string]interface{}{
				"title":   "系统维护",
				"content": "今晚22:00-24:00系统维护，请合理安排时间。",
			},
		},
	}

	ctx := context.Background()
	targets := []target.Target{sms.CreateTarget("+86 138 0013 8000")}

	for _, tmpl := range templates {
		fmt.Printf("\n📋 测试%s模板:\n", tmpl.name)

		// Create message with template
		msg := message.New()
		msg.Variables = tmpl.variables
		msg.Metadata = map[string]interface{}{
			"template": tmpl.template,
		}

		results, err := platform.Send(ctx, msg, targets)
		if err != nil {
			fmt.Printf("❌ 发送失败: %v\n", err)
			continue
		}

		if len(results) > 0 {
			if results[0].Success {
				fmt.Printf("✅ 发送成功: %s\n", results[0].Response)
			} else {
				fmt.Printf("❌ 发送失败: %v\n", results[0].Error)
			}
		}
	}
}

// demoErrorHandling demonstrates error handling
func demoErrorHandling() {
	// Test various error scenarios
	scenarios := []struct {
		name        string
		config      sms.Config
		phone       string
		expectError bool
	}{
		{
			name: "无效的提供商",
			config: sms.Config{
				Provider: "invalid_provider",
			},
			expectError: true,
		},
		{
			name: "缺少认证信息",
			config: sms.Config{
				Provider:    sms.ProviderAliyun,
				Credentials: map[string]string{},
			},
			expectError: true,
		},
		{
			name: "无效的手机号",
			config: sms.Config{
				Provider: sms.ProviderMock,
				Credentials: map[string]string{
					"should_fail": "false",
				},
			},
			phone:       "invalid_phone",
			expectError: true,
		},
		{
			name: "提供商发送失败",
			config: sms.Config{
				Provider: sms.ProviderMock,
				Credentials: map[string]string{
					"should_fail": "false",
				},
			},
			phone:       "+86 138 0013 fail",
			expectError: true,
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\n🧪 测试场景: %s\n", scenario.name)

		platform, err := sms.New(scenario.config)
		if err != nil {
			if scenario.expectError {
				fmt.Printf("✅ 预期错误: %v\n", err)
			} else {
				fmt.Printf("❌ 意外错误: %v\n", err)
			}
			continue
		}

		if scenario.phone != "" {
			msg := message.New()
			msg.Body = "测试短信"
			targets := []target.Target{sms.CreateTarget(scenario.phone)}

			ctx := context.Background()
			results, err := platform.Send(ctx, msg, targets)
			if err != nil {
				fmt.Printf("✅ 发送失败 (预期): %v\n", err)
			} else if len(results) > 0 && results[0].Error != nil {
				fmt.Printf("✅ 目标失败 (预期): %v\n", results[0].Error)
			} else {
				fmt.Printf("❌ 意外成功\n")
			}
		}

		if platform != nil {
			_ = platform.Close()
		}
	}
}

// runIntegrationDemo demonstrates integration with NotifyHub
func runIntegrationDemo() {
	fmt.Println("\n\n🔌 NotifyHub Integration Demo (Conceptual)")
	fmt.Println("==========================================")

	// Create NotifyHub client
	cfg := &config.Config{
		Logger: config.LoggerConfig{
			Level: "info",
		},
	}

	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create NotifyHub client: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Simulate SMS platform registration
	fmt.Println("📋 Registering external SMS platform...")
	err = registerSMSPlatform(client)
	if err != nil {
		log.Fatalf("Failed to register SMS platform: %v", err)
	}
	fmt.Println("✅ SMS platform registered successfully!")

	// Configure SMS platform
	smsConfig := sms.Config{
		Provider: sms.ProviderMock,
		Credentials: map[string]string{
			"should_fail": "false",
		},
		Templates: map[string]string{
			"verification": "验证码: {{code}}",
		},
		RateLimit: sms.RateLimitConfig{
			Enabled:    true,
			MaxPerHour: 100,
			MaxPerDay:  1000,
		},
	}

	err = configureSMSPlatform(client, smsConfig)
	if err != nil {
		log.Fatalf("Failed to configure SMS platform: %v", err)
	}
	fmt.Println("✅ SMS platform configured successfully!")

	// Test SMS platform capabilities
	err = testSMSCapabilities(client)
	if err != nil {
		log.Fatalf("Failed to test SMS capabilities: %v", err)
	}

	// Send test messages
	fmt.Println("\n📤 Sending test SMS messages through NotifyHub...")
	fmt.Println("⚠️  Note: These will show errors because SMS platform is not actually registered")

	// Example 1: Basic SMS
	err = sendBasicSMS(client, "+86 138 0013 8000")
	if err != nil {
		log.Printf("Expected failure - Basic SMS: %v", err)
	}

	// Example 2: Template SMS
	err = sendTemplateSMS(client, "+86 138 0013 8000")
	if err != nil {
		log.Printf("Expected failure - Template SMS: %v", err)
	}

	// Example 3: Async SMS
	err = sendAsyncSMS(client, "+86 138 0013 8000")
	if err != nil {
		log.Printf("Expected failure - Async SMS: %v", err)
	}

	fmt.Println("\n🎉 Integration demo completed!")
	fmt.Println("\nℹ️  Summary:")
	fmt.Println("   ✅ Standalone SMS platform works perfectly")
	fmt.Println("   🏢 Multiple provider support (Aliyun, Tencent, Twilio, Nexmo)")
	fmt.Println("   🚦 Rate limiting prevents spam")
	fmt.Println("   📋 Template system with variable substitution")
	fmt.Println("   ⚠️  Integration requires NotifyHub core API changes:")
	fmt.Println("      - client.RegisterPlatform(name, factory)")
	fmt.Println("      - client.SetPlatformConfig(name, config)")
	fmt.Println("      - SMS-specific routing and validation")
}

// Helper functions for integration demo

func registerSMSPlatform(client notifyhub.Client) error {
	// In a real implementation:
	// factory := platform.Factory(sms.New)
	// return client.RegisterPlatform("sms", factory)

	fmt.Println("⚠️  Note: Platform registration would require NotifyHub to expose registry API")
	return nil
}

func configureSMSPlatform(client notifyhub.Client, config sms.Config) error {
	// In a real implementation:
	// return client.SetPlatformConfig("sms", config)

	fmt.Printf("⚙️  SMS configuration: Provider=%s, Templates=%d, RateLimit=%v\n",
		config.Provider, len(config.Templates), config.RateLimit.Enabled)
	return nil
}

func testSMSCapabilities(client notifyhub.Client) error {
	fmt.Println("\n🔍 Testing SMS platform capabilities...")

	// Create SMS platform instance for testing
	config := sms.Config{
		Provider: sms.ProviderMock,
		Credentials: map[string]string{
			"should_fail": "false",
		},
	}

	platform, err := sms.New(config)
	if err != nil {
		return err
	}
	defer func() { _ = platform.Close() }()

	capabilities := platform.GetCapabilities()
	fmt.Printf("📋 Platform Name: %s\n", capabilities.Name)
	fmt.Printf("📋 Supported Targets: %v\n", capabilities.SupportedTargetTypes)
	fmt.Printf("📋 Supported Formats: %v\n", capabilities.SupportedFormats)
	fmt.Printf("📋 Max Message Size: %d characters\n", capabilities.MaxMessageSize)
	fmt.Printf("📋 Supports Scheduling: %v\n", capabilities.SupportsScheduling)
	fmt.Printf("📋 Required Settings: %v\n", capabilities.RequiredSettings)

	return nil
}

func sendBasicSMS(client notifyhub.Client, phone string) error {
	fmt.Println("\n📝 Sending basic SMS...")

	msg := message.New()
	msg.Title = "NotifyHub"
	msg.Body = "这是通过外部SMS平台发送的基础短信。"
	msg.Format = message.FormatText
	msg.Targets = []target.Target{
		sms.CreateTarget(phone),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Basic SMS sent successfully! ID: %s\n", receipt.MessageID)
	return nil
}

func sendTemplateSMS(client notifyhub.Client, phone string) error {
	fmt.Println("\n📋 Sending template SMS...")

	msg := message.New()
	msg.Variables = map[string]interface{}{
		"code": "123456",
	}
	msg.Metadata = map[string]interface{}{
		"template": "verification",
	}
	msg.Targets = []target.Target{
		sms.CreateTarget(phone),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Template SMS sent successfully! ID: %s\n", receipt.MessageID)
	return nil
}

func sendAsyncSMS(client notifyhub.Client, phone string) error {
	fmt.Println("\n⚡ Sending async SMS...")

	msg := message.New()
	msg.Body = "这是一条异步发送的短信，演示了NotifyHub的异步处理能力。"
	msg.Targets = []target.Target{
		sms.CreateTarget(phone),
	}

	ctx := context.Background()
	handle, err := client.SendAsync(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Printf("📤 Async SMS submitted! Handle ID: %s\n", handle.ID())

	// Set callbacks
	handle.OnComplete(func(receipt *receipt.Receipt) {
		fmt.Printf("✅ Async SMS completed successfully!\n")
	}).OnError(func(msg *message.Message, err error) {
		fmt.Printf("❌ Async SMS failed: %v\n", err)
	})

	// Wait for completion
	receipt, err := handle.Wait(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Async SMS sent successfully! ID: %s\n", receipt.MessageID)
	return nil
}
