// Package main demonstrates basic feishu messaging with NotifyHub
package main

import (
	"context"

	"github.com/kart/notifyhub/examples/common"
	"github.com/kart/notifyhub/pkg/notifyhub"
	"github.com/kart/notifyhub/pkg/target"
)

func main() {
	// Print header
	common.PrintHeader("Feishu", "Basic")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请修改以下配置为您的实际飞书信息
	config.Feishu.WebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/xxxxx-xxxx-xxxx-xxxxxxxxxxxxxxxxxxxx"
	config.Feishu.Secret = ""                               // 可选，飞书机器人签名校验
	config.Feishu.Keywords = []string{"111", "222"} // 可选，关键词设置

	// Check configuration
	if !common.CheckConfigurationPrompt("feishu") {
		logger.Info("已取消运行")
		return
	}

	// Validate configuration
	if err := common.ValidateConfig(config, "feishu"); err != nil {
		logger.Error("配置验证失败: %v", err)
		logger.Info("请修改代码中的飞书配置后重新运行")
		return
	}

	// Create NotifyHub client
	cfg := config.CreateFeishuConfig()
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

	logger.Success("所有飞书消息示例运行完成!")
}

// sendBasicTextMessage sends a basic text message to feishu
func sendBasicTextMessage(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("发送基础文本消息")

	// Create message
	msg := common.CreateTestMessage("Feishu", "basic")
	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
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
	msg := common.CreateTestMessage("Feishu", "markdown")
	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
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
	msg := common.CreateTestMessage("Feishu", "urgent")
	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
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
