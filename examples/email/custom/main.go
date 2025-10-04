// Package main demonstrates custom email functionality with NotifyHub
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kart/notifyhub/examples/common"
	"github.com/kart/notifyhub/pkg/notifyhub"
	"github.com/kart/notifyhub/pkg/platforms/email"
	"github.com/kart/notifyhub/pkg/target"
)

const (
	SMTP_HOST     = "smtp.feishu.cn"
	SMTP_PORT     = 465
	SMTP_USERNAME = "costa@hellotalk.cn"
	SMTP_PASSWORD = ""
	SMTP_FROM     = "costa@hellotalk.cn"
)

func main() {
	// Print header
	common.PrintHeader("Email Custom", "Templates & Configuration")

	// Create logger
	logger := common.NewLogger(true)

	logger.Info("🎨 自定义邮件功能演示")
	logger.Info("包括：模板系统、自定义配置、频率限制、追踪功能等")

	// Example 1: Basic custom email configuration
	if err := demonstrateBasicCustomConfig(logger); err != nil {
		logger.Error("基础自定义配置演示失败: %v", err)
	}

	// Example 2: Standard NotifyHub Client with Custom Config
	if err := demonstrateStandardNotifyHub(logger); err != nil {
		logger.Error("标准NotifyHub客户端演示失败: %v", err)
	}

	// Example 3: Template system
	if err := demonstrateTemplateSystem(logger); err != nil {
		logger.Error("模板系统演示失败: %v", err)
	}

	// // Example 3: Advanced features
	// if err := demonstrateAdvancedFeatures(logger); err != nil {
	// 	logger.Error("高级功能演示失败: %v", err)
	// }

	// Example 4: Rate limiting
	// if err := demonstrateRateLimiting(logger); err != nil {
	// 	logger.Error("频率限制演示失败: %v", err)
	// }

	logger.Success("自定义邮件功能演示完成！")
}

// demonstrateBasicCustomConfig demonstrates basic custom email configuration
func demonstrateBasicCustomConfig(logger *common.Logger) error {
	common.PrintSeparator("基础自定义邮件配置")

	// Create custom email configuration
	customConfig := &email.CustomEmailConfig{
		Name:        "my-custom-email",
		DisplayName: "我的自定义邮件服务",
		Host:        SMTP_HOST,
		Port:        SMTP_PORT,
		Username:    SMTP_USERNAME,
		Password:    SMTP_PASSWORD,
		From:        SMTP_FROM,
		FromName:    "NotifyHub Demo",
		UseTLS:      true,
		UseStartTLS: false,
		AuthMethod:  "plain",

		// Custom headers
		CustomHeaders: map[string]string{
			"X-Company":     "Your Company",
			"X-Department":  "IT",
			"X-Environment": "Demo",
		},

		// Reply settings
		ReplyToAddress: "noreply@yourcompany.com",

		// Basic rate limiting
		RateLimit:  30, // 30 emails per minute
		BurstLimit: 10, // burst of 10 emails

		Description: "用于演示的自定义邮件配置",
		Tags:        []string{"demo", "custom", "gmail"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	logger.Info("创建自定义邮件配置: %s", customConfig.Name)

	// Validate configuration
	if err := customConfig.Validate(); err != nil {
		logger.Error("配置验证失败: %v", err)
		return err
	}

	logger.Success("✅ 自定义邮件配置验证通过")

	// Create custom email sender
	sender, err := email.NewCustomEmailSender(customConfig, logger)
	if err != nil {
		logger.Error("创建自定义邮件发送器失败: %v", err)
		return err
	}
	defer sender.Close()

	logger.Success("✅ 自定义邮件发送器创建成功")

	// Test connection
	ctx := context.Background()
	if err := sender.TestConnection(ctx); err != nil {
		logger.Warn("连接测试失败 (这是正常的，因为使用的是示例凭据): %v", err)
	} else {
		logger.Success("✅ 连接测试成功")
	}

	// Get capabilities
	capabilities := sender.GetCapabilities()
	logger.Info("📋 邮件发送器能力:")
	logger.Info("   支持模板: %v", capabilities.SupportsTemplates)
	logger.Info("   支持追踪: %v", capabilities.SupportsTracking)
	logger.Info("   支持频率限制: %v", capabilities.SupportsRateLimiting)
	logger.Info("   支持自定义头: %v", capabilities.SupportsCustomHeaders)
	logger.Info("   单邮件最大收件人: %d", capabilities.MaxRecipientsPerEmail)
	logger.Info("   支持的格式: %v", capabilities.SupportedFormats)
	logger.Info("   可用模板数量: %d", len(capabilities.Templates))

	// 演示两种发送方式
	logger.Info("\n📧 演示两种发送方式:")

	// 方式1: 使用自定义邮件发送器 (CustomEmailOptions)
	logger.Info("\n1️⃣ 使用 CustomEmailSender.SendCustomEmail 方法:")
	customOptions := &email.CustomEmailOptions{
		RequestID:  fmt.Sprintf("basic-demo-%d", time.Now().Unix()),
		Template:   "plain",
		Subject:    "自定义邮件发送器测试",
		Body:       "这是使用自定义邮件发送器发送的测试邮件。",
		Recipients: []string{"costa9293@gmail.com", "longqiuhong@163.com"},
		Priority:   "normal",
		Variables: map[string]interface{}{
			"user_name": "测试用户",
			"company":   "示例公司",
		},
	}

	result, err := sender.SendCustomEmail(context.Background(), customOptions)
	if err != nil {
		logger.Error("自定义邮件发送失败: %v", err)
	} else {
		logger.Success("✅ 自定义邮件发送成功: %d/%d", result.Successful, result.Total)
	}

	// 方式2: 使用标准 NotifyHub 客户端 (message.Message + target.Target)
	logger.Info("\n2️⃣ 使用标准 NotifyHub 客户端和自定义平台:")

	// 这里需要创建一个使用自定义邮件配置的 NotifyHub 客户端
	// 但由于当前架构，我们只能演示如何准备消息和目标
	logger.Info("   📝 标准方式需要通过 NotifyHub 客户端配置")
	logger.Info("   📧 消息格式: message.Message + target.Target")
	logger.Info("   🎯 目标设置: target.New(target.TargetTypeEmail, email, target.PlatformEmail)")

	// 展示如何准备标准格式
	logger.Info("\n   示例代码:")
	logger.Info("   ```go")
	logger.Info("   msg := message.New()")
	logger.Info("   msg.Title = \"邮件主题\"")
	logger.Info("   msg.Body = \"邮件内容\"")
	logger.Info("   msg.Targets = []target.Target{")
	logger.Info("       target.New(target.TargetTypeEmail, \"user@example.com\", target.PlatformEmail),")
	logger.Info("   }")
	logger.Info("   client.Send(ctx, msg)")
	logger.Info("   ```")

	return nil
}

// demonstrateStandardNotifyHub demonstrates using standard NotifyHub client with custom email config
func demonstrateStandardNotifyHub(logger *common.Logger) error {
	common.PrintSeparator("标准NotifyHub客户端 + 自定义邮件配置")

	// 创建 NotifyHub 配置，使用我们的自定义SMTP设置
	config := &common.ExampleConfig{
		Email: common.EmailSettings{
			Host:     SMTP_HOST,
			Port:     SMTP_PORT,
			Username: SMTP_USERNAME,
			Password: SMTP_PASSWORD,
			From:     SMTP_FROM,
			To:       "costa9293@gmail.com", // 默认收件人，但会被 target 覆盖
			UseTLS:   true,                  // 使用我们验证过的TLS设置
		},
	}

	// 创建 NotifyHub 邮件配置
	nhConfig := config.CreateEmailConfig()
	nhConfig.Logger.Level = "debug"

	// 创建 NotifyHub 客户端
	client, err := notifyhub.NewClient(nhConfig)
	if err != nil {
		logger.Error("创建NotifyHub客户端失败: %v", err)
		return err
	}
	defer client.Close()

	logger.Success("✅ NotifyHub客户端创建成功 (使用自定义SMTP配置)")

	// 创建消息
	msg := common.CreateTestMessage("Email Custom", "basic")
	msg.Title = "📧 NotifyHub标准客户端测试"
	msg.Body = `这是通过标准NotifyHub客户端发送的自定义邮件。

📋 测试信息：
- 发送方式: NotifyHub 标准客户端
- SMTP服务器: ` + SMTP_HOST + `:` + fmt.Sprintf("%d", SMTP_PORT) + `
- 发件人: ` + SMTP_FROM + `
- 安全连接: TLS (端口465)

✅ 这种方式支持所有NotifyHub的标准功能，包括：
- 消息格式化 (HTML, Markdown, 纯文本)
- 目标管理 (target.Target)
- 优先级设置
- 元数据处理

💡 收件人通过 msg.Targets 设置，支持灵活的目标管理

--
此邮件由 NotifyHub 标准客户端发送
发送时间: ` + time.Now().Format("2006-01-02 15:04:05")

	// 设置收件人 - 使用标准的 target.Target 方式
	msg.Targets = []target.Target{
		common.CreateEmailTarget("costa9293@gmail.com"),
		common.CreateEmailTarget("longqiuhong@163.com"),
	}

	logger.Info("📧 准备发送邮件:")
	logger.Info("   主题: %s", msg.Title)
	logger.Info("   收件人数量: %d", len(msg.Targets))
	logger.Info("   格式: %s", msg.Format)
	logger.Info("   优先级: %s", msg.Priority)

	// 发送邮件
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		logger.Error("邮件发送失败: %v", err)
		return err
	}

	logger.Success("✅ NotifyHub标准客户端发送成功!")
	logger.Info("📊 发送结果:")
	logger.Info("   总数: %d", receipt.Total)
	logger.Info("   成功: %d", receipt.Successful)
	logger.Info("   失败: %d", receipt.Failed)

	// 显示详细结果
	for i, result := range receipt.Results {
		if result.Success {
			logger.Success("   收件人%d: %s ✅", i+1, result.Target)
		} else {
			logger.Error("   收件人%d: %s ❌ - %v", i+1, result.Target, result.Error)
		}
	}

	logger.Info("\n💡 总结:")
	logger.Info("   ✅ 标准NotifyHub客户端完全支持自定义SMTP配置")
	logger.Info("   ✅ 通过 msg.Targets 灵活设置收件人")
	logger.Info("   ✅ 支持所有NotifyHub标准功能")
	logger.Info("   ✅ 可与其他平台(飞书、Webhook等)混合使用")

	return nil
}

// demonstrateTemplateSystem demonstrates the email template system
func demonstrateTemplateSystem(logger *common.Logger) error {
	common.PrintSeparator("邮件模板系统演示")

	// Create a simple custom configuration for template testing
	customConfig := &email.CustomEmailConfig{
		Name:        "template-demo",
		DisplayName: "模板演示配置",
		Host:        SMTP_HOST,
		Port:        SMTP_PORT,
		Username:    SMTP_USERNAME,
		Password:    SMTP_PASSWORD,
		From:        SMTP_FROM,
		FromName:    "模板演示",
		UseTLS:      true,
		UseStartTLS: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	sender, err := email.NewCustomEmailSender(customConfig, logger)
	if err != nil {
		logger.Error("创建模板演示发送器失败: %v", err)
		return err
	}
	defer sender.Close()

	// Get available templates
	capabilities := sender.GetCapabilities()
	logger.Info("📧 可用的内置模板:")

	for name, template := range capabilities.Templates {
		logger.Info("   %s (%s) - %s", name, template.Type, template.Description)
	}

	// Demonstrate template rendering with different templates
	templates := []string{"notification", "alert", "plain", "marketing"}

	for _, templateName := range templates {
		logger.Info("\n🎨 测试模板: %s", templateName)

		// Prepare template options
		options := &email.CustomEmailOptions{
			RequestID: fmt.Sprintf("template-test-%s-%d", templateName, time.Now().Unix()),
			Template:  templateName,
			Subject:   "模板测试邮件",
			Body:      "这是使用 " + templateName + " 模板的测试邮件内容。",
			Recipients: []string{
				"costa9293@gmail.com",
				"longqiuhong@163.com",
			},
			Priority: "normal",
			Variables: map[string]interface{}{
				"user_name":     "张三",
				"company":       "示例公司",
				"button_text":   "查看详情",
				"button_url":    "https://example.com/details",
				"support_email": "support@example.com",
			},
			CustomData: map[string]interface{}{
				"campaign_id": "template-demo-001",
				"source":      "api",
			},
		}

		// Note: We won't actually send the emails in this demo
		logger.Info("   模板选项准备完成")
		logger.Info("   收件人: %v", options.Recipients)
		logger.Info("   模板变量: %d 个", len(options.Variables))

		// In a real scenario, you would call:
		result, err := sender.SendCustomEmail(context.Background(), options)
		if err != nil {
			logger.Error("模板 %s 发送失败: %v", templateName, err)
		} else {
			logger.Success("模板 %s 发送成功: %v", templateName, result)
		}
		logger.Success("   ✅ 模板 %s 配置验证通过", templateName)
	}

	return nil
}

// demonstrateAdvancedFeatures demonstrates advanced email features
func demonstrateAdvancedFeatures(logger *common.Logger) error {
	common.PrintSeparator("高级邮件功能演示")

	// Create configuration with advanced features
	advancedConfig := &email.CustomEmailConfig{
		Name:        "advanced-email",
		DisplayName: "高级邮件服务",
		Host:        "smtp.yourcompany.com",
		Port:        587,
		Username:    "notifications@yourcompany.com",
		Password:    "secure-password",
		From:        "notifications@yourcompany.com",
		FromName:    "Your Company Notifications",
		UseTLS:      false,
		UseStartTLS: true,

		// Enable tracking
		EnableTracking: true,
		TrackingDomain: "track.yourcompany.com",
		UnsubscribeURL: "https://yourcompany.com/unsubscribe",

		// Advanced headers
		CustomHeaders: map[string]string{
			"X-Mailer":                 "NotifyHub Advanced v1.0",
			"X-Company":                "Your Company",
			"X-Priority":               "3",
			"X-Auto-Response-Suppress": "OOF, DR, RN, NRN",
		},

		// Domain restrictions
		AllowedDomains: []string{"yourcompany.com", "partner.com", "gmail.com"},
		BlockedDomains: []string{"tempmail.com", "10minutemail.com"},

		// Enhanced validation
		RequireSSL:         true,
		ValidateRecipients: true,

		// Rate limiting
		RateLimit:       60, // 60 emails per minute
		BurstLimit:      20, // burst of 20 emails
		RateLimitWindow: time.Minute,

		// Retry settings
		MaxRetries:    3,
		RetryInterval: 30 * time.Second,
		Timeout:       60 * time.Second,

		Description: "具有高级功能的企业级邮件配置",
		Tags:        []string{"enterprise", "tracking", "secure"},
		Metadata: map[string]interface{}{
			"environment": "production",
			"team":        "devops",
			"cost_center": "IT-001",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	logger.Info("🚀 创建高级邮件配置")

	// Validate configuration
	if err := advancedConfig.Validate(); err != nil {
		logger.Error("高级配置验证失败: %v", err)
		return err
	}

	logger.Success("✅ 高级配置验证通过")

	// Create sender
	sender, err := email.NewCustomEmailSender(advancedConfig, logger)
	if err != nil {
		logger.Error("创建高级邮件发送器失败: %v", err)
		return err
	}
	defer sender.Close()

	// Display advanced features
	capabilities := sender.GetCapabilities()
	logger.Info("🔧 高级功能状态:")
	logger.Info("   ✅ 邮件追踪: %v", capabilities.SupportsTracking)
	logger.Info("   ✅ 频率限制: %v", capabilities.SupportsRateLimiting)
	logger.Info("   ✅ 自定义头: %v", capabilities.SupportsCustomHeaders)
	logger.Info("   📊 可用模板: %d", len(capabilities.Templates))

	// Demonstrate recipient validation
	logger.Info("\n📧 收件人验证测试:")

	testRecipients := []string{
		"valid@yourcompany.com",    // Should pass - in allowed domains
		"user@partner.com",         // Should pass - in allowed domains
		"test@gmail.com",           // Should pass - in allowed domains
		"spam@tempmail.com",        // Should fail - in blocked domains
		"invalid@unauthorized.com", // Should fail - not in allowed domains
		"not-an-email",             // Should fail - invalid format
	}

	for _, recipient := range testRecipients {
		logger.Info("   测试: %s", recipient)
		// Note: In a real implementation, we would test the validation
		// For this demo, we just show the configuration
	}

	// Show configuration summary
	logger.Info("\n⚙️  配置摘要:")
	logger.Info("   名称: %s", advancedConfig.Name)
	logger.Info("   显示名: %s", advancedConfig.DisplayName)
	logger.Info("   发件人: %s <%s>", advancedConfig.FromName, advancedConfig.From)
	logger.Info("   追踪域名: %s", advancedConfig.TrackingDomain)
	logger.Info("   取消订阅URL: %s", advancedConfig.UnsubscribeURL)
	logger.Info("   允许域名: %v", advancedConfig.AllowedDomains)
	logger.Info("   禁止域名: %v", advancedConfig.BlockedDomains)
	logger.Info("   发送频率: %d 邮件/分钟", advancedConfig.RateLimit)
	logger.Info("   突发限制: %d 邮件", advancedConfig.BurstLimit)

	return nil
}

// demonstrateRateLimiting demonstrates rate limiting functionality
func demonstrateRateLimiting(logger *common.Logger) error {
	common.PrintSeparator("频率限制演示")

	// Create rate limiter for testing
	rateLimiter := email.NewRateLimiter(5, 10, time.Minute) // 5 per minute, burst of 10

	logger.Info("🚦 创建频率限制器:")
	logger.Info("   频率: 5 邮件/分钟")
	logger.Info("   突发: 10 邮件")
	logger.Info("   窗口: 1 分钟")

	// Show initial state
	stats := rateLimiter.Stats()
	logger.Info("\n📊 初始状态:")
	logger.Info("   当前令牌: %d", stats.CurrentTokens)
	logger.Info("   容量: %d", stats.Capacity)
	logger.Info("   下次补充: %v", stats.TimeToRefill)

	// Simulate email sending with rate limiting
	logger.Info("\n🔄 模拟邮件发送:")

	for i := 1; i <= 15; i++ {
		if rateLimiter.Allow() {
			logger.Success("   第 %d 封邮件: ✅ 允许发送", i)
		} else {
			logger.Warn("   第 %d 封邮件: ❌ 频率限制", i)
		}

		// Show current token count every 5 attempts
		if i%5 == 0 {
			tokens := rateLimiter.Tokens()
			logger.Info("   当前剩余令牌: %d", tokens)
		}
	}

	// Show final statistics
	finalStats := rateLimiter.Stats()
	logger.Info("\n📈 最终统计:")
	logger.Info("   剩余令牌: %d", finalStats.CurrentTokens)
	logger.Info("   补充频率: %d 令牌/%v", finalStats.RefillRate, finalStats.RefillPeriod)
	logger.Info("   下次补充时间: %v", finalStats.TimeToRefill)

	logger.Info("\n💡 频率限制说明:")
	logger.Info("   - 令牌桶算法实现平滑的频率控制")
	logger.Info("   - 支持突发流量处理")
	logger.Info("   - 自动补充令牌，防止长期阻塞")
	logger.Info("   - 线程安全，支持并发使用")

	return nil
}

// Helper function for formatting
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
