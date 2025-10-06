// Package main demonstrates basic webhook sending with NotifyHub
package main

import (
	"context"

	"github.com/kart/notifyhub/examples/common"
	"github.com/kart/notifyhub/pkg/notifyhub"
	"github.com/kart/notifyhub/pkg/target"
)

func main() {
	// Print header
	common.PrintHeader("Webhook", "Basic")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请修改以下配置为您的实际Webhook信息
	config.Webhook.URL = "https://httpbin.org/post" // 测试用的webhook地址
	config.Webhook.Method = "POST"                  // HTTP方法
	config.Webhook.Headers = map[string]string{     // 自定义请求头
		"Content-Type":    "application/json",
		"Authorization":   "Bearer your_token_here", // 如果需要认证
		"X-Custom-Header": "NotifyHub-Example",
	}
	config.Webhook.AuthType = "bearer"       // 认证类型: bearer, basic, none
	config.Webhook.Token = "your_token_here" // Bearer token

	// Check configuration
	if !common.CheckConfigurationPrompt("webhook") {
		logger.Info("已取消运行")
		return
	}

	// Validate configuration
	if err := common.ValidateConfig(config, "webhook"); err != nil {
		logger.Error("配置验证失败: %v", err)
		logger.Info("请修改代码中的Webhook配置后重新运行")
		return
	}

	// Create NotifyHub client
	cfg := config.CreateWebhookConfig()
	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("创建NotifyHub客户端失败: %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	logger.Success("NotifyHub客户端创建成功")

	// Example 1: Send basic JSON payload
	if err := sendBasicJSON(client, config, logger); err != nil {
		logger.Error("发送基础JSON数据失败: %v", err)
	}

	// Example 2: Send text message
	if err := sendTextMessage(client, config, logger); err != nil {
		logger.Error("发送文本消息失败: %v", err)
	}

	// Example 3: Send structured data
	if err := sendStructuredData(client, config, logger); err != nil {
		logger.Error("发送结构化数据失败: %v", err)
	}

	// Example 4: Send urgent notification
	if err := sendUrgentNotification(client, config, logger); err != nil {
		logger.Error("发送紧急通知失败: %v", err)
	}

	logger.Success("所有Webhook示例运行完成!")
}

// sendBasicJSON sends a basic JSON payload
func sendBasicJSON(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送基础JSON数据")

	// Create message
	msg := common.CreateTestMessage("Webhook", "basic")
	msg.Targets = []target.Target{
		common.CreateWebhookTarget(config.Webhook.URL),
	}

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("基础JSON数据发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendTextMessage sends a plain text message
func sendTextMessage(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送纯文本消息")

	// Create text message
	msg := common.CreateTestMessage("Webhook", "basic")
	msg.Title = "Webhook 文本消息测试"
	msg.Body = "这是一条通过Webhook发送的纯文本消息。"

	// Override platform data for plain text
	msg.PlatformData = map[string]interface{}{
		"webhook": map[string]interface{}{
			"content_type": "text/plain",
			"payload":      msg.Body,
		},
	}

	msg.Targets = []target.Target{
		common.CreateWebhookTarget(config.Webhook.URL),
	}

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("纯文本消息发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendStructuredData sends structured data payload
func sendStructuredData(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送结构化数据")

	// Create structured message
	msg := common.CreateTestMessage("Webhook", "basic")
	msg.Title = "Webhook 结构化数据测试"
	msg.Body = "这是一条包含结构化数据的Webhook消息。"

	// Add structured platform data
	msg.PlatformData = map[string]interface{}{
		"webhook": map[string]interface{}{
			"event_type": "notification",
			"timestamp":  "2023-12-01T12:00:00Z",
			"source":     "notifyhub",
			"data": map[string]interface{}{
				"user_id":     "user_123",
				"action":      "message_sent",
				"object_type": "webhook",
				"object_id":   "webhook_001",
				"metadata": map[string]interface{}{
					"ip_address": "192.168.1.100",
					"user_agent": "NotifyHub/1.0",
					"session_id": "sess_abc123",
				},
			},
		},
	}

	msg.Targets = []target.Target{
		common.CreateWebhookTarget(config.Webhook.URL),
	}

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("结构化数据发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendUrgentNotification sends an urgent notification
func sendUrgentNotification(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送紧急通知")

	// Create urgent message
	msg := common.CreateTestMessage("Webhook", "urgent")
	msg.Targets = []target.Target{
		common.CreateWebhookTarget(config.Webhook.URL),
	}

	// Add urgent notification data
	msg.PlatformData = map[string]interface{}{
		"webhook": map[string]interface{}{
			"alert_level":           "urgent",
			"severity":              "high",
			"requires_ack":          true,
			"escalation":            true,
			"notification_channels": []string{"email", "sms", "phone"},
		},
	}

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("紧急通知发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}
