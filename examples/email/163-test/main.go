// Package main demonstrates 163.com email sending with NotifyHub
package main

import (
	"context"
	"time"

	"github.com/kart/notifyhub/examples/common"
	"github.com/kart/notifyhub/pkg/notifyhub"
	"github.com/kart/notifyhub/pkg/target"
)

const (
	SMTP_HOST     = "smtp.163.com"
	SMTP_PORT     = 25
	SMTP_USERNAME = "longqiuhong@163.com"
	SMTP_PASSWORD = ""
	SMTP_FROM     = "longqiuhong@163.com"
)

func main() {
	// Print header
	common.PrintHeader("163 Email", "Basic")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请修改以下配置为您的实际163邮箱信息
	config.Email.Host = SMTP_HOST         // 163 SMTP服务器
	config.Email.Port = SMTP_PORT         // 163 SMTP端口 (也可以使用587)
	config.Email.Username = SMTP_USERNAME // 您的163邮箱
	config.Email.Password = SMTP_PASSWORD // 163邮箱授权码（不是登录密码）
	config.Email.From = SMTP_FROM         // 发件人

	logger.Info("🔧 163邮箱配置说明：")
	logger.Info("  1. 需要在163邮箱设置中开启SMTP服务")
	logger.Info("  2. 使用授权码而不是登录密码")
	logger.Info("  3. 授权码获取：163邮箱设置 -> POP3/SMTP/IMAP -> 开启服务并设置授权码")
	logger.Info("  4. 推荐使用端口25（STARTTLS）或587")

	// Check configuration
	if !common.CheckConfigurationPrompt("163 email") {
		logger.Info("已取消运行")
		return
	}

	// Validate configuration
	if err := common.ValidateConfig(config, "email"); err != nil {
		logger.Error("配置验证失败: %v", err)
		logger.Info("请修改代码中的163邮箱配置后重新运行")
		return
	}

	// Create NotifyHub client with optimized settings for 163
	cfg := config.CreateEmailConfig()
	cfg.Logger.Level = "debug" // Enable debug logging to see SMTP process

	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("创建NotifyHub客户端失败: %v", err)
		return
	}
	defer client.Close()

	logger.Success("NotifyHub客户端创建成功 (163邮箱)")

	// Test 163 email sending
	if err := send163TestEmail(client, config, logger); err != nil {
		logger.Error("163邮件发送失败: %v", err)
	}

	logger.Success("163邮件测试完成!")
}

// send163TestEmail sends a test email via 163.com SMTP
func send163TestEmail(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送163邮箱测试邮件")

	// Create message optimized for 163
	msg := common.CreateTestMessage("163 Email", "basic")
	msg.Title = "📧 163邮箱 NotifyHub 测试"
	msg.Body = `这是通过163邮箱SMTP服务器发送的测试邮件。

📋 测试信息：
- 发送方式: NotifyHub SMTP客户端
- SMTP服务器: smtp.163.com
- 加密方式: STARTTLS
- 认证方式: PLAIN (授权码)

✅ 如果您收到这封邮件，说明163邮箱SMTP配置成功！

💡 重要提醒：
1. 163邮箱需要使用授权码，不是登录密码
2. 需要在163邮箱设置中开启SMTP服务
3. 建议使用端口25（STARTTLS）或587

--
此邮件由 NotifyHub 发送
发送时间: ` + time.Now().Format("2006-01-02 15:04:05")

	msg.Targets = []target.Target{
		common.CreateEmailTarget("costa9293@gmail.com"),
		common.CreateEmailTarget("costa@hellotalk.cn"),
	}

	logger.Info("📤 开始通过163 SMTP发送邮件...")
	logger.Debug("163配置: Host=%s, Port=%d, From=%s", config.Email.Host, config.Email.Port, config.Email.From)

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("163邮件发送成功!")
	logger.Debug("发送回执: %+v", receipt)

	// 详细分析发送结果
	if receipt.Failed > 0 {
		logger.Warn("有 %d 封邮件发送失败", receipt.Failed)
		for _, result := range receipt.Results {
			if !result.Success {
				logger.Error("发送失败 - Target: %s, Error: %v", result.Target, result.Error)
			}
		}
	} else {
		logger.Info("✉️  邮件已成功发送到163 SMTP服务器")
		logger.Info("📧 收件人: %s", config.Email.To)
		logger.Info("📨 邮件主题: %s", msg.Title)
		logger.Info("💌 请检查收件人邮箱（包括垃圾邮件文件夹）")
	}

	return nil
}
