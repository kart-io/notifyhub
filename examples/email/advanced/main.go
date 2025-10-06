// Package main demonstrates advanced email features with NotifyHub
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/examples/common"
	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/target"
)

func main() {
	// Print header
	common.PrintHeader("Email", "Advanced")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请修改以下配置为您的实际邮箱信息
	config.Email.Host = "smtp.gmail.com"
	config.Email.Port = 587
	config.Email.Username = "your_email@gmail.com"
	config.Email.Password = "your_app_password"
	config.Email.From = "your_email@gmail.com"
	config.Email.To = "recipient@example.com"

	// Check configuration
	if !common.CheckConfigurationPrompt("email") {
		logger.Info("已取消运行")
		return
	}

	// Validate configuration
	if err := common.ValidateConfig(config, "email"); err != nil {
		logger.Error("配置验证失败: %v", err)
		return
	}

	// Create NotifyHub client with async support
	cfg := config.CreateEmailConfig()
	cfg.Async.Enabled = true
	cfg.Async.Workers = 4
	cfg.Logger.Level = "debug"

	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("创建NotifyHub客户端失败: %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	logger.Success("NotifyHub客户端创建成功 (异步模式)")

	// Advanced examples
	examples := []struct {
		name string
		fn   func(notifyhub.Client, *common.ExampleConfig, *common.Logger) error
	}{
		{"批量邮件发送", sendBatchEmails},
		{"带附件的邮件", sendEmailWithAttachments},
		{"多收件人邮件", sendMultiRecipientEmail},
		{"异步邮件发送", sendAsyncEmail},
		{"模板化邮件", sendTemplatedEmail},
		{"带元数据的邮件", sendEmailWithMetadata},
	}

	for _, example := range examples {
		if err := example.fn(client, config, logger); err != nil {
			logger.Error("%s失败: %v", example.name, err)
		}

		// 添加延迟避免频率限制
		time.Sleep(2 * time.Second)
	}

	logger.Success("所有高级邮件示例运行完成!")
}

// sendBatchEmails demonstrates batch email sending
func sendBatchEmails(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("批量邮件发送")

	messages := []*message.Message{
		createWelcomeEmail(config.Email.To),
		createNewsletterEmail(config.Email.To),
		createReminderEmail(config.Email.To),
	}

	ctx := context.Background()
	receipts, err := client.SendBatch(ctx, messages)
	if err != nil {
		return err
	}

	logger.Success("批量邮件发送成功! 共发送 %d 封邮件", len(receipts))
	return nil
}

// sendEmailWithAttachments demonstrates email with attachments (simulated)
func sendEmailWithAttachments(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送带附件的邮件")

	msg := message.New()
	msg.Title = "📎 带附件的邮件测试"
	msg.Body = `这封邮件包含了一些附件文件（模拟）：

1. 文档文件 (document.pdf)
2. 表格文件 (spreadsheet.xlsx)
3. 图片文件 (image.png)

附件通过 platform_data 字段模拟实现。`
	msg.Format = message.FormatText
	msg.Priority = message.PriorityNormal

	// Simulate attachments using platform_data
	msg.PlatformData = map[string]interface{}{
		"email": map[string]interface{}{
			"attachments": []map[string]interface{}{
				{
					"name":         "document.pdf",
					"content_type": "application/pdf",
					"content":      "JVBERi0xLjQK", // Base64 sample
					"inline":       false,
				},
				{
					"name":         "spreadsheet.xlsx",
					"content_type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
					"content":      "UEsDBBQA", // Base64 sample
					"inline":       false,
				},
			},
		},
	}

	msg.Targets = []target.Target{
		common.CreateEmailTarget(config.Email.To),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("带附件的邮件发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendMultiRecipientEmail demonstrates multi-recipient email
func sendMultiRecipientEmail(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送多收件人邮件")

	msg := message.New()
	msg.Title = "👥 多收件人邮件测试"
	msg.Body = `这是一封发送给多个收件人的测试邮件。

通过 platform_data 可以设置：
- TO: 主要收件人
- CC: 抄送收件人
- BCC: 密送收件人

当前配置为发送给主收件人，实际使用时可以配置多个地址。`
	msg.Format = message.FormatText

	// Configure multiple recipients through platform_data
	msg.PlatformData = map[string]interface{}{
		"email": map[string]interface{}{
			"cc":  []string{}, // 可以添加抄送地址
			"bcc": []string{}, // 可以添加密送地址
		},
	}

	msg.Targets = []target.Target{
		common.CreateEmailTarget(config.Email.To),
		// 可以添加更多目标地址
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("多收件人邮件发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendAsyncEmail demonstrates asynchronous email sending
func sendAsyncEmail(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("异步邮件发送")

	msg := common.CreateTestMessage("Email", "basic")
	msg.Title = "⚡ 异步邮件发送测试"
	msg.Body = "这是一封异步方式发送的邮件。\n\n异步功能已经实现，邮件将在后台处理。"
	msg.Targets = []target.Target{
		common.CreateEmailTarget(config.Email.To),
	}

	logger.Info("🚀 使用异步模式发送邮件")

	ctx := context.Background()
	handle, err := client.SendAsync(ctx, msg)
	if err != nil {
		return err
	}

	logger.Info("异步邮件已提交，消息ID: %s", handle.ID())

	// Wait for the result
	receipt, err := handle.Wait(ctx)
	if err != nil {
		logger.Error("异步邮件发送失败: %v", err)
		return err
	}

	logger.Success("异步邮件发送完成!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendTemplatedEmail demonstrates templated email
func sendTemplatedEmail(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送模板化邮件")

	// Template variables
	templateVars := map[string]interface{}{
		"user_name":    "张三",
		"product_name": "NotifyHub",
		"login_time":   time.Now().Format("2006-01-02 15:04:05"),
		"ip_address":   "192.168.1.100",
	}

	msg := message.New()
	msg.Title = "🔐 安全登录通知"
	msg.Body = buildLoginNotificationTemplate(templateVars)
	msg.Format = message.FormatHTML
	msg.Variables = templateVars

	msg.Targets = []target.Target{
		common.CreateEmailTarget(config.Email.To),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("模板化邮件发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendEmailWithMetadata demonstrates email with rich metadata
func sendEmailWithMetadata(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送带元数据的邮件")

	msg := message.New()
	msg.Title = "📊 带元数据的邮件测试"
	msg.Body = `这封邮件展示了如何使用丰富的元数据。

元数据用途：
- 邮件分类和跟踪
- 统计分析
- 自动化处理
- 业务标识

邮件ID: ` + msg.ID
	msg.Format = message.FormatText

	// Rich metadata
	msg.Metadata = map[string]interface{}{
		"campaign_id":    "metadata_demo_001",
		"sender_dept":    "技术部",
		"category":       "测试",
		"tracking_pixel": true,
		"utm_source":     "notifyhub",
		"utm_campaign":   "advanced_demo",
		"priority_score": 85,
		"tags":           []string{"demo", "metadata", "advanced"},
	}

	msg.Targets = []target.Target{
		common.CreateEmailTarget(config.Email.To),
	}

	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("带元数据的邮件发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	logger.Debug("元数据: %+v", msg.Metadata)
	return nil
}

// Helper functions for creating different types of emails

func createWelcomeEmail(to string) *message.Message {
	msg := message.New()
	msg.Title = "🎉 欢迎使用 NotifyHub"
	msg.Body = "欢迎使用 NotifyHub 邮件服务！感谢您的信任和支持。"
	msg.Format = message.FormatText
	msg.Targets = []target.Target{common.CreateEmailTarget(to)}
	msg.Metadata = map[string]interface{}{"type": "welcome"}
	return msg
}

func createNewsletterEmail(to string) *message.Message {
	msg := message.New()
	msg.Title = "📰 NotifyHub 技术周报"
	msg.Body = `# NotifyHub 技术周报

## 本周更新
- 邮件平台优化
- 新增批量发送功能
- 性能提升 20%

## 即将推出
- 短信平台支持
- 模板引擎
- 更多统计功能

感谢关注！`
	msg.Format = message.FormatMarkdown
	msg.Targets = []target.Target{common.CreateEmailTarget(to)}
	msg.Metadata = map[string]interface{}{"type": "newsletter"}
	return msg
}

func createReminderEmail(to string) *message.Message {
	msg := message.New()
	msg.Title = "⏰ 友好提醒"
	msg.Body = "这是一个友好的提醒消息，用于测试批量发送功能。"
	msg.Format = message.FormatText
	msg.Priority = message.PriorityLow
	msg.Targets = []target.Target{common.CreateEmailTarget(to)}
	msg.Metadata = map[string]interface{}{"type": "reminder"}
	return msg
}

func buildLoginNotificationTemplate(vars map[string]interface{}) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>安全登录通知</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .info { background-color: #f8f9fa; padding: 15px; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 安全登录通知</h1>
        </div>
        <div class="content">
            <p>尊敬的 <strong>%s</strong>，您好！</p>
            <p>您的 %s 账户刚刚登录成功。</p>
            <div class="info">
                <p><strong>登录时间:</strong> %s</p>
                <p><strong>IP地址:</strong> %s</p>
            </div>
            <p>如果这不是您本人的操作，请立即联系我们。</p>
        </div>
    </div>
</body>
</html>`,
		vars["user_name"], vars["product_name"], vars["login_time"], vars["ip_address"])
}
