// Package main demonstrates how to configure sender and recipients for custom email
package main

import (
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/examples/common"
	"github.com/kart-io/notifyhub/pkg/platforms/email"
)

func configureSenderAndRecipients() error {
	logger := common.NewLogger(true)

	// ================================
	// 1. 发件人配置 (Sender Configuration)
	// ================================
	customConfig := &email.CustomEmailConfig{
		Name:        "my-email-service",
		DisplayName: "我的邮件服务",

		// 🏷️ 发件人设置
		From:     "sender@yourcompany.com", // 发件人邮箱地址 (必填)
		FromName: "Your Company Name",      // 发件人显示名称 (可选)
		Username: "sender@yourcompany.com", // SMTP认证用户名 (通常与From相同)
		Password: "your-app-password",      // 邮箱密码或应用专用密码 (必填)

		// 🌐 SMTP服务器设置
		Host: "smtp.yourcompany.com", // SMTP服务器地址 (必填)
		Port: 587,                    // SMTP端口 (必填)

		// 🔒 安全设置
		UseTLS:      false,   // 是否使用TLS
		UseStartTLS: true,    // 是否使用STARTTLS (推荐)
		AuthMethod:  "plain", // 认证方式

		// 📧 回复和退信设置
		ReplyToAddress: "noreply@yourcompany.com", // 回复地址 (可选)
		BounceAddress:  "bounce@yourcompany.com",  // 退信地址 (可选)

		// 📝 自定义邮件头
		CustomHeaders: map[string]string{
			"X-Company":    "Your Company",
			"X-Department": "IT Department",
			"X-Priority":   "normal",
		},

		// 🚦 频率限制 (可选)
		RateLimit:  60, // 每分钟60封邮件
		BurstLimit: 20, // 突发限制20封

		// 🛡️ 域名限制 (可选)
		AllowedDomains: []string{
			"yourcompany.com",
			"partner.com",
			"gmail.com",
		},
		BlockedDomains: []string{
			"tempmail.com",
			"spam.com",
		},

		// 📊 追踪设置 (可选)
		EnableTracking: true,
		TrackingDomain: "track.yourcompany.com",
		UnsubscribeURL: "https://yourcompany.com/unsubscribe",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	logger.Info("🔧 发件人配置完成:")
	logger.Info("   发件人: %s <%s>", customConfig.FromName, customConfig.From)
	logger.Info("   SMTP服务器: %s:%d", customConfig.Host, customConfig.Port)
	logger.Info("   安全连接: STARTTLS=%v, TLS=%v", customConfig.UseStartTLS, customConfig.UseTLS)

	// 创建邮件发送器
	sender, err := email.NewCustomEmailSender(customConfig, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := sender.Close(); err != nil {
			fmt.Printf("关闭发送器失败: %v\n", err)
		}
	}()

	// ================================
	// 2. 收件人配置 (Recipients Configuration)
	// ================================

	// 方式1: 基础收件人设置
	basicOptions := &email.CustomEmailOptions{
		RequestID: "basic-email-001",
		Subject:   "基础邮件测试",
		Body:      "这是一封测试邮件",

		// 📧 收件人列表 (必填)
		Recipients: []string{
			"user1@yourcompany.com",
			"user2@partner.com",
			"user3@gmail.com",
		},

		Priority: "normal",
	}

	logger.Info("\n📧 基础收件人配置:")
	logger.Info("   收件人数量: %d", len(basicOptions.Recipients))
	for i, recipient := range basicOptions.Recipients {
		logger.Info("   收件人%d: %s", i+1, recipient)
	}

	// 方式2: 使用模板的收件人设置
	templateOptions := &email.CustomEmailOptions{
		RequestID: "template-email-001",
		Template:  "notification", // 使用内置模板
		Subject:   "重要通知",

		// 📧 多个收件人
		Recipients: []string{
			"manager@yourcompany.com",
			"team-lead@yourcompany.com",
			"developer1@yourcompany.com",
			"developer2@yourcompany.com",
		},

		// 📝 模板变量
		Variables: map[string]interface{}{
			"user_name": "团队成员",
			"company":   "Your Company",
			"message":   "系统维护通知",
		},

		Priority: "high",
	}

	logger.Info("\n🎨 模板收件人配置:")
	logger.Info("   模板名称: %s", templateOptions.Template)
	logger.Info("   收件人数量: %d", len(templateOptions.Recipients))
	logger.Info("   优先级: %s", templateOptions.Priority)

	// 方式3: 批量收件人 (大量发送)
	var batchRecipients []string

	// 模拟生成大量收件人
	departments := []string{"sales", "marketing", "support", "engineering"}
	for _, dept := range departments {
		for i := 1; i <= 10; i++ {
			batchRecipients = append(batchRecipients,
				fmt.Sprintf("%s%d@yourcompany.com", dept, i))
		}
	}

	batchOptions := &email.CustomEmailOptions{
		RequestID:  "batch-email-001",
		Template:   "newsletter",
		Subject:    "公司月报",
		Recipients: batchRecipients,
		Variables: map[string]interface{}{
			"month":          "12月",
			"year":           "2024",
			"newsletter_url": "https://yourcompany.com/newsletter/202412",
		},
		Priority: "low",
	}

	logger.Info("\n📨 批量收件人配置:")
	logger.Info("   收件人总数: %d", len(batchOptions.Recipients))
	logger.Info("   前5个收件人:")
	for i := 0; i < 5 && i < len(batchOptions.Recipients); i++ {
		logger.Info("     %d. %s", i+1, batchOptions.Recipients[i])
	}

	// ================================
	// 3. 发送邮件示例
	// ================================

	logger.Info("\n🚀 发送邮件示例 (仅演示，不实际发送):")

	// 注意: 在实际环境中取消注释以下代码来发送邮件
	// ctx := context.Background()

	// 发送基础邮件
	// result1, err := sender.SendCustomEmail(ctx, basicOptions)
	// if err != nil {
	//     logger.Error("基础邮件发送失败: %v", err)
	// } else {
	//     logger.Success("基础邮件发送成功: %d/%d", result1.Successful, result1.Total)
	// }

	// 发送模板邮件
	// result2, err := sender.SendCustomEmail(ctx, templateOptions)
	// if err != nil {
	//     logger.Error("模板邮件发送失败: %v", err)
	// } else {
	//     logger.Success("模板邮件发送成功: %d/%d", result2.Successful, result2.Total)
	// }

	// 批量发送 (建议分批处理)
	// batchSize := 50
	// for i := 0; i < len(batchRecipients); i += batchSize {
	//     end := i + batchSize
	//     if end > len(batchRecipients) {
	//         end = len(batchRecipients)
	//     }
	//
	//     batchOptions.Recipients = batchRecipients[i:end]
	//     result, err := sender.SendCustomEmail(ctx, batchOptions)
	//     if err != nil {
	//         logger.Error("批量邮件发送失败: %v", err)
	//     } else {
	//         logger.Info("批量邮件发送: %d/%d", result.Successful, result.Total)
	//     }
	// }

	logger.Success("✅ 发件人和收件人配置演示完成!")

	return nil
}

// ================================
// 4. 常见邮件服务商配置示例
// ================================

// Gmail配置示例
func gmailConfig() *email.CustomEmailConfig {
	return &email.CustomEmailConfig{
		Name:        "gmail-service",
		Host:        "smtp.gmail.com",
		Port:        587,
		Username:    "your-email@gmail.com",
		Password:    "your-app-password", // 使用应用专用密码
		From:        "your-email@gmail.com",
		FromName:    "Your Name",
		UseStartTLS: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// 163邮箱配置示例
func email163Config() *email.CustomEmailConfig {
	return &email.CustomEmailConfig{
		Name:        "163-service",
		Host:        "smtp.163.com",
		Port:        25, // 或587
		Username:    "your-email@163.com",
		Password:    "your-auth-code", // 使用授权码
		From:        "your-email@163.com",
		FromName:    "Your Name",
		UseStartTLS: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// QQ邮箱配置示例
func qqMailConfig() *email.CustomEmailConfig {
	return &email.CustomEmailConfig{
		Name:        "qq-service",
		Host:        "smtp.qq.com",
		Port:        587,
		Username:    "your-email@qq.com",
		Password:    "your-auth-code", // 使用授权码
		From:        "your-email@qq.com",
		FromName:    "Your Name",
		UseStartTLS: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// 企业邮箱配置示例
func enterpriseConfig() *email.CustomEmailConfig {
	return &email.CustomEmailConfig{
		Name:        "enterprise-service",
		Host:        "smtp.yourcompany.com",
		Port:        587,
		Username:    "your-email@yourcompany.com",
		Password:    "your-password",
		From:        "noreply@yourcompany.com",
		FromName:    "Your Company System",
		UseStartTLS: true,
		RequireSSL:  true, // 企业级安全要求

		// 企业级设置
		CustomHeaders: map[string]string{
			"X-Company":     "Your Company",
			"X-System":      "NotifyHub",
			"X-Environment": "Production",
		},

		// 严格的域名限制
		AllowedDomains:     []string{"yourcompany.com"},
		ValidateRecipients: true,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Suppress unused function warnings for example code
var _ = func() {
	_ = configureSenderAndRecipients
	_ = gmailConfig
	_ = email163Config
	_ = qqMailConfig
	_ = enterpriseConfig
}
