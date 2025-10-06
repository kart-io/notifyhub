// Package main demonstrates how to register and use an external DingTalk platform
// This example shows how to extend NotifyHub without modifying core code
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kart-io/notifyhub/examples/external-platform-dingtalk/dingtalk"
	"github.com/kart-io/notifyhub/pkg/config"
	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/receipt"
	"github.com/kart-io/notifyhub/pkg/target"
)

func main() {
	fmt.Println("🤖 NotifyHub External Platform Demo - DingTalk")
	fmt.Println("=============================================")

	// Display two demo modes
	fmt.Println("\nThis demo shows two modes:")
	fmt.Println("1. 🔧 Standalone Platform Demo - Shows how the platform works independently")
	fmt.Println("2. 🔌 Integration Demo - Shows how it would integrate with NotifyHub (conceptual)")

	// Run standalone demo first
	runStandaloneDemo()

	// Run integration demo (conceptual)
	runIntegrationDemo()
}

// runStandaloneDemo demonstrates the DingTalk platform working independently
func runStandaloneDemo() {
	fmt.Println("\n🔧 Standalone DingTalk Platform Demo")
	fmt.Println("===================================")

	// Create DingTalk platform configuration
	config := dingtalk.Config{
		WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_ACCESS_TOKEN",
		Secret:     "", // Optional
		Timeout:    30,
	}

	// Create DingTalk platform instance
	platform, err := dingtalk.New(config)
	if err != nil {
		log.Fatalf("Failed to create DingTalk platform: %v", err)
	}
	defer func() { _ = platform.Close() }()

	fmt.Printf("✅ DingTalk platform created: %s\n", platform.Name())

	// Test platform capabilities
	capabilities := platform.GetCapabilities()
	fmt.Printf("📋 Platform capabilities: %+v\n", capabilities)

	// Create test message
	msg := message.New()
	msg.Title = "🚀 Standalone Platform Test"
	msg.Body = "This message is sent directly through the DingTalk platform without going through NotifyHub core."
	msg.Format = message.FormatText

	// Create targets
	targets := []target.Target{
		dingtalk.CreateTarget(config.WebhookURL),
	}

	// Validate targets
	fmt.Println("\n🔍 Validating targets...")
	for i, target := range targets {
		if err := platform.ValidateTarget(target); err != nil {
			fmt.Printf("❌ Target %d validation failed: %v\n", i+1, err)
		} else {
			fmt.Printf("✅ Target %d is valid\n", i+1)
		}
	}

	// Send message directly through platform
	fmt.Println("\n📤 Sending message directly through DingTalk platform...")
	ctx := context.Background()
	results, err := platform.Send(ctx, msg, targets)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}

	// Process results
	fmt.Println("\n📊 Send results:")
	for i, result := range results {
		if result.Success {
			fmt.Printf("✅ Target %d: Success (ID: %s)\n", i+1, result.MessageID)
		} else {
			fmt.Printf("❌ Target %d: Failed - %v\n", i+1, result.Error)
		}
	}

	// Test health check
	fmt.Println("\n🏥 Health check...")
	if err := platform.IsHealthy(ctx); err != nil {
		fmt.Printf("❌ Platform health check failed: %v\n", err)
	} else {
		fmt.Printf("✅ Platform is healthy\n")
	}

	// Test different message formats
	fmt.Println("\n📝 Testing Markdown message...")
	markdownMsg := message.New()
	markdownMsg.Title = "Markdown Test"
	markdownMsg.Body = `## DingTalk Platform Features

### ✅ Supported Features
- **Text Messages**: Plain text notifications
- **Markdown Messages**: Rich formatted content
- **Target Validation**: Ensures valid webhook URLs
- **Health Monitoring**: Platform status checking

### 📊 Technical Details
| Feature | Status |
|---------|--------|
| HTTP Client | ✅ Working |
| Error Handling | ✅ Working |
| Message Conversion | ✅ Working |

> This is a **standalone** platform demonstration.`
	markdownMsg.Format = message.FormatMarkdown

	markdownResults, err := platform.Send(ctx, markdownMsg, targets)
	if err != nil {
		log.Printf("Failed to send markdown message: %v", err)
	}

	for i, result := range markdownResults {
		if result.Success {
			fmt.Printf("✅ Markdown Target %d: Success (ID: %s)\n", i+1, result.MessageID)
		} else {
			fmt.Printf("❌ Markdown Target %d: Failed - %v\n", i+1, result.Error)
		}
	}

	fmt.Println("\n🎉 Standalone DingTalk platform demo completed!")
	fmt.Println("\nℹ️  Note: This demo shows how the platform works independently.")
	fmt.Println("   To integrate with NotifyHub, the core system would need to provide:")
	fmt.Println("   - Platform registry API")
	fmt.Println("   - Configuration management")
	fmt.Println("   - Unified message routing")
}

func runIntegrationDemo() {
	fmt.Println("\n\n🔌 NotifyHub Integration Demo (Conceptual)")
	fmt.Println("==========================================")

	// Step 1: Create NotifyHub client with basic configuration
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

	// Step 2: Register external DingTalk platform
	// This is the key - we register our external platform without modifying NotifyHub core
	fmt.Println("📋 Registering external DingTalk platform...")

	// Get the platform registry (assuming NotifyHub exposes this)
	// Note: In real implementation, NotifyHub would need to expose the registry
	// or provide a RegisterExternalPlatform method
	err = registerDingTalkPlatform(client)
	if err != nil {
		log.Fatalf("Failed to register DingTalk platform: %v", err)
	}

	fmt.Println("✅ DingTalk platform registered successfully!")

	// Step 3: Configure DingTalk platform
	dingTalkConfig := dingtalk.Config{
		WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_ACCESS_TOKEN",
		Secret:     "", // Optional
		Timeout:    30,
	}

	err = configureDingTalkPlatform(client, dingTalkConfig)
	if err != nil {
		log.Fatalf("Failed to configure DingTalk platform: %v", err)
	}

	fmt.Println("✅ DingTalk platform configured successfully!")

	// Step 4: Test DingTalk platform capabilities
	err = testDingTalkCapabilities(client)
	if err != nil {
		log.Fatalf("Failed to test DingTalk capabilities: %v", err)
	}

	// Step 5: Send messages using DingTalk platform (through NotifyHub)
	fmt.Println("\n📤 Sending test messages through NotifyHub...")
	fmt.Println("⚠️  Note: These will fail because DingTalk platform is not actually registered")

	// Example 1: Basic text message
	err = sendBasicTextMessage(client, dingTalkConfig.WebhookURL)
	if err != nil {
		log.Printf("Expected failure - Failed to send basic text message: %v", err)
	}

	// Example 2: Markdown message
	err = sendMarkdownMessage(client, dingTalkConfig.WebhookURL)
	if err != nil {
		log.Printf("Expected failure - Failed to send markdown message: %v", err)
	}

	// Example 3: Message with @mentions
	err = sendMessageWithMentions(client, dingTalkConfig.WebhookURL)
	if err != nil {
		log.Printf("Expected failure - Failed to send message with mentions: %v", err)
	}

	// Example 4: Async message sending
	err = sendAsyncMessage(client, dingTalkConfig.WebhookURL)
	if err != nil {
		log.Printf("Expected failure - Failed to send async message: %v", err)
	}

	fmt.Println("\n🎉 Integration demo completed!")
	fmt.Println("\nℹ️  Summary:")
	fmt.Println("   ✅ Standalone platform works perfectly")
	fmt.Println("   ⚠️  Integration requires NotifyHub core API changes:")
	fmt.Println("      - client.RegisterPlatform(name, factory)")
	fmt.Println("      - client.SetPlatformConfig(name, config)")
	fmt.Println("      - Expose platform registry for external platforms")
}

// registerDingTalkPlatform registers the DingTalk platform factory
// Note: This assumes NotifyHub provides a way to register external platforms
func registerDingTalkPlatform(client notifyhub.Client) error {
	// In a real implementation, NotifyHub would need to expose platform registration
	// This is a conceptual example of how it would work

	// factory := platform.Factory(NewDingTalkPlatform)
	// return client.RegisterPlatform("dingtalk", factory)

	fmt.Println("⚠️  Note: Platform registration would require NotifyHub to expose registry API")
	return nil // Placeholder
}

// configureDingTalkPlatform sets the configuration for DingTalk platform
func configureDingTalkPlatform(client notifyhub.Client, config dingtalk.Config) error {
	// In a real implementation:
	// return client.SetPlatformConfig("dingtalk", config)

	fmt.Printf("⚙️  DingTalk configuration: WebhookURL=%s, Timeout=%ds\n",
		maskWebhookURL(config.WebhookURL), config.Timeout)
	return nil // Placeholder
}

// testDingTalkCapabilities tests the platform capabilities
func testDingTalkCapabilities(client notifyhub.Client) error {
	fmt.Println("\n🔍 Testing DingTalk platform capabilities...")

	// Create a DingTalk platform instance for testing
	config := dingtalk.Config{
		WebhookURL: "https://example.com/webhook",
		Timeout:    30,
	}

	platform, err := dingtalk.New(config)
	if err != nil {
		return err
	}

	capabilities := platform.GetCapabilities()
	fmt.Printf("📋 Platform Name: %s\n", capabilities.Name)
	fmt.Printf("📋 Supported Targets: %v\n", capabilities.SupportedTargetTypes)
	fmt.Printf("📋 Supported Formats: %v\n", capabilities.SupportedFormats)
	fmt.Printf("📋 Max Message Size: %d bytes\n", capabilities.MaxMessageSize)
	fmt.Printf("📋 Supports Scheduling: %v\n", capabilities.SupportsScheduling)
	fmt.Printf("📋 Supports Attachments: %v\n", capabilities.SupportsAttachments)
	fmt.Printf("📋 Required Settings: %v\n", capabilities.RequiredSettings)

	return nil
}

// sendBasicTextMessage sends a basic text message
func sendBasicTextMessage(client notifyhub.Client, webhookURL string) error {
	fmt.Println("\n📝 Sending basic text message...")

	msg := message.New()
	msg.Title = "NotifyHub 测试消息"
	msg.Body = "这是通过外部 DingTalk 平台发送的测试消息。\n\n时间: " + fmt.Sprintf("%d", 1640995200)
	msg.Format = message.FormatText
	msg.Targets = []target.Target{
		dingtalk.CreateTarget(webhookURL),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Printf("✅ Message sent successfully! ID: %s\n", receipt.MessageID)
	return nil
}

// sendMarkdownMessage sends a markdown formatted message
func sendMarkdownMessage(client notifyhub.Client, webhookURL string) error {
	fmt.Println("\n📝 Sending markdown message...")

	msg := message.New()
	msg.Title = "Markdown 格式测试"
	msg.Body = `## 功能测试报告

### ✅ 已完成
- **外部平台注册**: 成功实现 DingTalk 平台扩展
- **消息发送**: 支持文本和 Markdown 格式
- **目标验证**: 实现目标类型验证

### 📊 统计信息
| 项目 | 数量 |
|------|------|
| 发送成功 | 100% |
| 响应时间 | < 1s |

### 🔗 相关链接
- [NotifyHub](https://github.com/kart-io/notifyhub)
- [DingTalk API](https://developers.dingtalk.com/)

> 这是一条 **Markdown** 格式的测试消息`

	msg.Format = message.FormatMarkdown
	msg.Targets = []target.Target{
		dingtalk.CreateTarget(webhookURL),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send markdown message: %w", err)
	}

	fmt.Printf("✅ Markdown message sent successfully! ID: %s\n", receipt.MessageID)
	return nil
}

// sendMessageWithMentions sends a message with @mentions
func sendMessageWithMentions(client notifyhub.Client, webhookURL string) error {
	fmt.Println("\n📝 Sending message with @mentions...")

	msg := message.New()
	msg.Title = "重要通知"
	msg.Body = "请所有相关人员注意查看这条重要消息！"
	msg.Format = message.FormatText
	msg.Targets = []target.Target{
		dingtalk.CreateTargetWithAt(
			webhookURL,
			true,                                   // @all
			[]string{"13800138000", "13900139000"}, // @mobiles
			[]string{"user123", "user456"},         // @userIds
		),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send message with mentions: %w", err)
	}

	fmt.Printf("✅ Message with mentions sent successfully! ID: %s\n", receipt.MessageID)
	return nil
}

// sendAsyncMessage sends a message asynchronously
func sendAsyncMessage(client notifyhub.Client, webhookURL string) error {
	fmt.Println("\n📝 Sending async message...")

	msg := message.New()
	msg.Title = "异步消息测试"
	msg.Body = "这是一条异步发送的测试消息，演示了 NotifyHub 的异步处理能力。"
	msg.Format = message.FormatText
	msg.Targets = []target.Target{
		dingtalk.CreateTarget(webhookURL),
	}

	ctx := context.Background()
	handle, err := client.SendAsync(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send async message: %w", err)
	}

	fmt.Printf("📤 Async message submitted! Handle ID: %s\n", handle.ID())

	// Set callbacks
	handle.OnComplete(func(receipt *receipt.Receipt) {
		fmt.Printf("✅ Async message completed successfully!\n")
	}).OnError(func(msg *message.Message, err error) {
		fmt.Printf("❌ Async message failed: %v\n", err)
	})

	// Wait for completion
	receipt, err := handle.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for async message: %w", err)
	}

	fmt.Printf("✅ Async message sent successfully! ID: %s\n", receipt.MessageID)
	return nil
}

// maskWebhookURL masks sensitive parts of webhook URL for logging
func maskWebhookURL(url string) string {
	if len(url) > 50 {
		return url[:30] + "..." + url[len(url)-10:]
	}
	return url
}
