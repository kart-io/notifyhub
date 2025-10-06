// Package main demonstrates async callbacks functionality with NotifyHub
package main

import (
	"context"
	"strings"
	"time"

	"github.com/kart/notifyhub/examples/common"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/notifyhub"
	"github.com/kart/notifyhub/pkg/receipt"
	"github.com/kart/notifyhub/pkg/target"
)

func main() {
	// Print header
	common.PrintHeader("Async Callbacks", "Demo")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请根据需要修改配置
	config.Feishu.WebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/b6bd1f02-01a7-4adc-9cd0-f043414dd5f1"

	config.Feishu.Keywords = []string{"notification", "通知"}

	// Check configuration
	if !common.CheckConfigurationPrompt("feishu") {
		logger.Info("已取消运行")
		return
	}

	// Create NotifyHub client
	cfg := config.CreateFeishuConfig()
	cfg.Logger.Level = "debug"

	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("创建NotifyHub客户端失败: %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	logger.Success("NotifyHub客户端创建成功")

	// Callback examples
	examples := []struct {
		name string
		fn   func(notifyhub.Client, *common.ExampleConfig, *common.Logger) error
	}{
		{"单条消息异步回调", singleMessageCallback},
		{"批量消息异步回调", batchMessagesCallback},
		{"完整回调功能演示", fullCallbackDemo},
	}

	for _, example := range examples {
		logger.Info("\n" + strings.Repeat("=", 50))
		logger.Info("开始执行: %s", example.name)
		logger.Info(strings.Repeat("=", 50))

		if err := example.fn(client, config, logger); err != nil {
			logger.Error("%s失败: %v", example.name, err)
		}

		// 添加延迟
		time.Sleep(3 * time.Second)
	}

	logger.Success("所有异步回调示例运行完成!")
}

// singleMessageCallback demonstrates single message with callbacks
func singleMessageCallback(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("单条消息异步回调演示")

	msg := message.New()
	msg.Title = "🔔 异步回调测试"
	msg.Body = "这条消息将演示异步回调功能。"
	msg.Format = message.FormatText
	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
	}

	ctx := context.Background()

	// Create handle and set up callbacks
	handle, err := client.SendAsync(ctx, msg)
	if err != nil {
		return err
	}

	// Set up callbacks using the fluent interface
	handle.OnComplete(func(receiptData *receipt.Receipt) {
		logger.Success("✅ 消息发送成功回调触发!")
		logger.Info("回执信息: MessageID=%s, 成功数=%d",
			receiptData.MessageID, len(receiptData.Results))
	}).OnError(func(messageData *message.Message, err error) {
		logger.Error("❌ 消息发送失败回调触发!")
		logger.Error("错误信息: %v", err)
		logger.Error("消息ID: %s", messageData.ID)
	})

	logger.Info("异步消息已提交，等待回调...")

	// Wait for completion
	_, err = handle.Wait(ctx)
	if err != nil {
		logger.Error("等待异步结果失败: %v", err)
		return err
	}

	logger.Info("异步操作完成，最终结果已获取")
	return nil
}

// batchMessagesCallback demonstrates batch messages with callbacks
func batchMessagesCallback(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("批量消息异步回调演示")

	// Create multiple messages
	messages := []*message.Message{
		createCallbackTestMessage("消息1", "这是第一条测试消息"),
		createCallbackTestMessage("消息2", "这是第二条测试消息"),
		createCallbackTestMessage("消息3", "这是第三条测试消息"),
	}

	ctx := context.Background()

	// Send batch with callbacks
	batchHandle, err := client.SendAsyncBatch(ctx, messages)
	if err != nil {
		return err
	}

	logger.Info("批量异步消息已提交，监听进度...")

	// Monitor progress
	go func() {
		for progress := range batchHandle.Progress() {
			logger.Info("📊 批量进度: %d/%d 完成 (%.1f%%)",
				progress.Completed, progress.Total, progress.Progress*100)
		}
	}()

	// Monitor results
	go func() {
		resultCount := 0
		for result := range batchHandle.Results() {
			resultCount++
			if result.Error != nil {
				logger.Error("📦 批量结果 %d: 失败 - %v", resultCount, result.Error)
			} else {
				logger.Success("📦 批量结果 %d: 成功", resultCount)
			}
		}
	}()

	// Wait for all to complete
	receipts, err := batchHandle.Wait(ctx)
	if err != nil {
		logger.Error("等待批量异步结果失败: %v", err)
		return err
	}

	logger.Success("批量异步操作完成，共处理 %d 条消息", len(receipts))
	for i, receiptData := range receipts {
		if receiptData != nil {
			logger.Info("收据 %d: MessageID=%s", i+1, receiptData.MessageID)
		}
	}
	return nil
}

// fullCallbackDemo demonstrates full callback functionality
func fullCallbackDemo(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("完整回调功能演示")

	msg := message.New()
	msg.Title = "🎯 完整回调演示"
	msg.Body = "这条消息演示所有类型的回调功能。"
	msg.Format = message.FormatText
	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
	}

	ctx := context.Background()

	// Send with full callback setup
	handle, err := client.SendAsync(ctx, msg)
	if err != nil {
		return err
	}

	// Set up comprehensive callbacks
	handle.OnComplete(func(receiptData *receipt.Receipt) {
		logger.Success("🎉 完成回调: 消息发送成功!")
		logger.Info("   - 消息ID: %s", receiptData.MessageID)
		logger.Info("   - 发送时间: %s", receiptData.Timestamp.Format("2006-01-02 15:04:05"))
		logger.Info("   - 结果数量: %d", len(receiptData.Results))

		for i, result := range receiptData.Results {
			logger.Info("   - 结果%d: 平台=%s, 目标=%s, 成功=%v",
				i+1, result.Platform, result.Target, result.Success)
		}
	}).OnError(func(messageData *message.Message, err error) {
		logger.Error("💥 错误回调: 消息发送失败!")
		logger.Error("   - 消息ID: %s", messageData.ID)
		logger.Error("   - 消息标题: %s", messageData.Title)
		logger.Error("   - 错误信息: %v", err)
		logger.Error("   - 目标数量: %d", len(messageData.Targets))
	}).OnProgress(func(completed, total int) {
		progress := float64(completed) / float64(total) * 100
		logger.Info("🔄 进度回调: %d/%d 完成 (%.1f%%)", completed, total, progress)
	})

	logger.Info("等待完整回调演示...")

	// Wait for completion
	_, err = handle.Wait(ctx)
	if err != nil {
		logger.Error("等待失败: %v", err)
		return err
	}

	logger.Success("完整回调演示完成!")
	return nil
}

// Helper function to create test messages
func createCallbackTestMessage(title, body string) *message.Message {
	msg := message.New()
	msg.Title = title
	msg.Body = body
	msg.Format = message.FormatText
	msg.Targets = []target.Target{
		common.CreateFeishuTarget(),
	}
	return msg
}
