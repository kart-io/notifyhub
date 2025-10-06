// Package main demonstrates basic Slack messaging with NotifyHub
package main

import (
	"context"

	"github.com/kart-io/notifyhub/examples/common"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/target"
)

func main() {
	// Print header
	common.PrintHeader("Slack", "Basic")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请修改以下配置为您的实际Slack信息
	config.Slack.WebhookURL = "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
	config.Slack.Token = ""             // 可选，Bot Token (xoxb-...)
	config.Slack.Channel = "#general"   // 默认频道
	config.Slack.Username = "NotifyHub" // 机器人用户名
	config.Slack.IconEmoji = ":bell:"   // 机器人图标

	// Check configuration
	if !common.CheckConfigurationPrompt("slack") {
		logger.Info("已取消运行")
		return
	}

	// Validate configuration
	if err := common.ValidateConfig(config, "slack"); err != nil {
		logger.Error("配置验证失败: %v", err)
		logger.Info("请修改代码中的Slack配置后重新运行")
		return
	}

	// Create NotifyHub client
	cfg := config.CreateSlackConfig()
	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("创建NotifyHub客户端失败: %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	logger.Success("NotifyHub客户端创建成功")

	// Example 1: Send basic text message
	if err := sendBasicTextMessage(client, config, logger); err != nil {
		logger.Error("发送基础文本消息失败: %v", err)
	}

	// Example 2: Send Markdown message
	if err := sendMarkdownMessage(client, config, logger); err != nil {
		logger.Error("发送Markdown消息失败: %v", err)
	}

	// Example 3: Send urgent message
	if err := sendUrgentMessage(client, config, logger); err != nil {
		logger.Error("发送紧急消息失败: %v", err)
	}

	// Example 4: Send to specific channel
	if err := sendToSpecificChannel(client, config, logger); err != nil {
		logger.Error("发送特定频道消息失败: %v", err)
	}

	logger.Success("所有Slack消息示例运行完成!")
}

// sendBasicTextMessage sends a basic text message to Slack
func sendBasicTextMessage(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送基础文本消息")

	// Create message
	msg := common.CreateTestMessage("Slack", "basic")
	msg.Targets = []target.Target{
		common.CreateSlackTarget(""),
	}

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("基础文本消息发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendMarkdownMessage sends a Markdown formatted message
func sendMarkdownMessage(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送Markdown格式消息")

	// Create Markdown message
	msg := common.CreateTestMessage("Slack", "markdown")
	msg.Targets = []target.Target{
		common.CreateSlackTarget(""),
	}

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("Markdown格式消息发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendUrgentMessage sends an urgent priority message
func sendUrgentMessage(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送紧急优先级消息")

	// Create urgent message
	msg := common.CreateTestMessage("Slack", "urgent")
	msg.Targets = []target.Target{
		common.CreateSlackTarget(""),
	}

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("紧急优先级消息发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}

// sendToSpecificChannel sends a message to a specific channel
func sendToSpecificChannel(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送到特定频道")

	// Create message for specific channel
	msg := common.CreateTestMessage("Slack", "basic")
	msg.Title = "📢 特定频道测试消息"
	msg.Body = "这是一条发送到特定频道的测试消息。\n\n如果您在其他频道看到这条消息，说明频道配置正确！"

	// Send to specific channels (you can modify these)
	msg.Targets = []target.Target{
		common.CreateSlackTarget("#general"), // 发送到 #general 频道
		common.CreateSlackTarget("#random"),  // 发送到 #random 频道 (如果存在)
	}

	// Send message
	ctx := context.Background()
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	logger.Success("特定频道消息发送成功!")
	logger.Debug("发送回执: %+v", receipt)
	return nil
}
