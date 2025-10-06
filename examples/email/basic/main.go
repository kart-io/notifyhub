// Package main demonstrates basic email sending with NotifyHub
package main

import (
	"context"

	"github.com/kart/notifyhub/examples/common"
	"github.com/kart/notifyhub/pkg/notifyhub"
	"github.com/kart/notifyhub/pkg/target"
)

const (
	SMTP_HOST     = "smtp.gmail.com"
	SMTP_PORT     = 587
	SMTP_USERNAME = "costa9293@gmail.com"
	SMTP_PASSWORD = ""
	SMTP_FROM     = "costa9293@gmail.com"
)

func main() {
	// Print header
	common.PrintHeader("Email", "Basic")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请修改以下配置为您的实际邮箱信息
	config.Email.Host = SMTP_HOST           // SMTP服务器
	config.Email.Port = SMTP_PORT           // SMTP端口
	config.Email.Username = SMTP_USERNAME   // 您的邮箱
	config.Email.Password = SMTP_PASSWORD   // 应用专用密码
	config.Email.From = SMTP_FROM           // 发件人
	config.Email.To = "longqiuhong@163.com" // 收件人 - 请修改为真实邮箱    // 收件人 - 请修改为真实邮箱

	// Check configuration (skip prompt for debugging)
	logger.Info("跳过配置确认，直接运行测试...")
	// if !common.CheckConfigurationPrompt("email") {
	//	logger.Info("已取消运行")
	//	return
	// }

	// Validate configuration
	if err := common.ValidateConfig(config, "email"); err != nil {
		logger.Error("配置验证失败: %v", err)
		logger.Info("请修改代码中的邮箱配置后重新运行")
		return
	}

	// Create NotifyHub client
	cfg := config.CreateEmailConfig()
	logger.Debug("邮件配置: Host=%s, Port=%d, Username=%s, From=%s, To=%s",
		cfg.Email.Host, cfg.Email.Port, cfg.Email.Username, cfg.Email.From, config.Email.To)

	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("创建NotifyHub客户端失败: %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	logger.Success("NotifyHub客户端创建成功")

	// Example 1: Send basic text email
	if err := sendBasicTextEmail(client, config, logger); err != nil {
		logger.Error("发送基础文本邮件失败: %v", err)
	}

	// Example 2: Send HTML email
	if err := sendHTMLEmail(client, config, logger); err != nil {
		logger.Error("发送HTML邮件失败: %v", err)
	}

	// Example 3: Send Markdown email
	if err := sendMarkdownEmail(client, config, logger); err != nil {
		logger.Error("发送Markdown邮件失败: %v", err)
	}

	// Example 4: Send urgent email
	if err := sendUrgentEmail(client, config, logger); err != nil {
		logger.Error("发送紧急邮件失败: %v", err)
	}

	logger.Success("所有邮件示例运行完成!")
}

// sendBasicTextEmail sends a basic text email
func sendBasicTextEmail(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送基础文本邮件")

	// Create message
	msg := common.CreateTestMessage("Email", "basic")
	msg.Targets = []target.Target{
		common.CreateEmailTarget(config.Email.To),
	}

	logger.Debug("准备发送邮件: Title=%s, To=%s", msg.Title, config.Email.To)
	logger.Debug("邮件内容长度: %d 字符", len(msg.Body))

	// Send message
	ctx := context.Background()
	logger.Info("开始发送基础文本邮件...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		logger.Error("邮件发送过程中出错: %v", err)
		return err
	}

	logger.Success("基础文本邮件发送成功!")
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
		logger.Info("✉️  邮件已发送到 SMTP 服务器，请检查收件箱（包括垃圾邮件文件夹）")
		logger.Info("📧 收件人: %s", config.Email.To)
		logger.Info("📨 邮件主题: %s", msg.Title)
	}
	return nil
}

// sendHTMLEmail sends an HTML formatted email
func sendHTMLEmail(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送HTML格式邮件")

	// Create HTML message
	msg := common.CreateTestMessage("Email", "html")
	msg.Targets = []target.Target{
		common.CreateEmailTarget(config.Email.To),
	}

	logger.Debug("准备发送HTML邮件: Title=%s", msg.Title)

	// Send message
	ctx := context.Background()
	logger.Info("开始发送HTML格式邮件...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		logger.Error("HTML邮件发送过程中出错: %v", err)
		return err
	}

	logger.Success("HTML格式邮件发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendMarkdownEmail sends a Markdown formatted email
func sendMarkdownEmail(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送Markdown格式邮件")

	// Create Markdown message
	msg := common.CreateTestMessage("Email", "markdown")
	msg.Targets = []target.Target{
		common.CreateEmailTarget(config.Email.To),
	}

	logger.Debug("准备发送Markdown邮件: Title=%s", msg.Title)

	// Send message
	ctx := context.Background()
	logger.Info("开始发送Markdown格式邮件...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		logger.Error("Markdown邮件发送过程中出错: %v", err)
		return err
	}

	logger.Success("Markdown格式邮件发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendUrgentEmail sends an urgent priority email
func sendUrgentEmail(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送紧急优先级邮件")

	// Create urgent message
	msg := common.CreateTestMessage("Email", "urgent")
	msg.Targets = []target.Target{
		common.CreateEmailTarget(config.Email.To),
	}

	logger.Debug("准备发送紧急邮件: Title=%s, Priority=%v", msg.Title, msg.Priority)

	// Send message
	ctx := context.Background()
	logger.Info("开始发送紧急优先级邮件...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		logger.Error("紧急邮件发送过程中出错: %v", err)
		return err
	}

	logger.Success("紧急优先级邮件发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}
