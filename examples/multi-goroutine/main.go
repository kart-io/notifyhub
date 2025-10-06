// Package main demonstrates multi-goroutine usage with NotifyHub async features
package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/examples/common"
	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/receipt"
	"github.com/kart-io/notifyhub/pkg/target"
)

func main() {
	// Print header
	common.PrintHeader("Multi-Goroutine", "Async Demo")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	config := common.DefaultExampleConfig()

	// 请根据需要修改配置
	config.Feishu.WebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url"

	// Check configuration
	if !common.CheckConfigurationPrompt("feishu") {
		logger.Info("已取消运行")
		return
	}

	// Create NotifyHub client with async support
	cfg := config.CreateFeishuConfig()
	cfg.Async.Enabled = true
	cfg.Async.Workers = 8 // 使用8个工作协程
	cfg.Logger.Level = "debug"

	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		logger.Error("创建NotifyHub客户端失败: %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	logger.Success("NotifyHub客户端创建成功 (多协程模式)")
	logger.Info("当前系统协程数: %d", runtime.NumGoroutine())

	// Multi-goroutine examples
	examples := []struct {
		name string
		fn   func(notifyhub.Client, *common.ExampleConfig, *common.Logger) error
	}{
		{"并发单条消息发送", concurrentSingleMessages},
		{"大批量并发发送", largeBatchConcurrent},
		{"混合并发模式", mixedConcurrentMode},
		{"协程池压力测试", goroutinePoolStressTest},
	}

	for _, example := range examples {
		logger.Info("\n" + strings.Repeat("=", 60))
		logger.Info("开始执行: %s", example.name)
		logger.Info("协程数 (开始前): %d", runtime.NumGoroutine())
		logger.Info(strings.Repeat("=", 60))

		start := time.Now()
		if err := example.fn(client, config, logger); err != nil {
			logger.Error("%s失败: %v", example.name, err)
		}
		duration := time.Since(start)

		logger.Info("协程数 (完成后): %d", runtime.NumGoroutine())
		logger.Info("执行时间: %v", duration)
		logger.Info(strings.Repeat("=", 60))

		// 添加延迟和清理
		time.Sleep(2 * time.Second)
		runtime.GC() // 强制垃圾回收
	}

	logger.Success("所有多协程异步示例运行完成!")
	logger.Info("最终协程数: %d", runtime.NumGoroutine())
}

// concurrentSingleMessages demonstrates concurrent single message sending
func concurrentSingleMessages(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("并发单条消息发送演示")

	ctx := context.Background()
	concurrency := 10 // 并发数
	var wg sync.WaitGroup

	logger.Info("🚀 启动 %d 个并发协程发送消息", concurrency)

	// 使用协程池并发发送单条消息
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			msg := message.New()
			msg.Title = fmt.Sprintf("🔀 并发消息 #%d", index+1)
			msg.Body = fmt.Sprintf("这是第 %d 个并发发送的消息，时间: %s",
				index+1, time.Now().Format("15:04:05.000"))
			msg.Format = message.FormatText
			msg.Targets = []target.Target{
				common.CreateFeishuTarget(),
			}

			// 异步发送
			handle, err := client.SendAsync(ctx, msg)
			if err != nil {
				logger.Error("协程 %d 发送失败: %v", index+1, err)
				return
			}

			// 设置回调
			handle.OnComplete(func(receipt *receipt.Receipt) {
				logger.Success("✅ 协程 %d 消息发送成功: %s", index+1, receipt.MessageID)
			}).OnError(func(message *message.Message, err error) {
				logger.Error("❌ 协程 %d 消息发送失败: %v", index+1, err)
			})

			// 等待完成
			_, err = handle.Wait(ctx)
			if err != nil {
				logger.Error("协程 %d 等待失败: %v", index+1, err)
			}

			logger.Info("协程 %d 完成", index+1)
		}(i)
	}

	// 等待所有协程完成
	wg.Wait()
	logger.Success("✅ 所有 %d 个并发协程已完成", concurrency)
	return nil
}

// largeBatchConcurrent demonstrates large batch concurrent processing
func largeBatchConcurrent(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("大批量并发发送演示")

	ctx := context.Background()
	batchSize := 20 // 每批次消息数
	batchCount := 3 // 批次数
	var wg sync.WaitGroup

	logger.Info("🚀 启动 %d 个批次，每批次 %d 条消息", batchCount, batchSize)

	// 并发处理多个批次
	for batch := 0; batch < batchCount; batch++ {
		wg.Add(1)
		go func(batchIndex int) {
			defer wg.Done()

			// 创建批次消息
			messages := make([]*message.Message, batchSize)
			for i := 0; i < batchSize; i++ {
				msg := message.New()
				msg.Title = fmt.Sprintf("📦 批次%d-消息%d", batchIndex+1, i+1)
				msg.Body = fmt.Sprintf("批次 %d 中的第 %d 条消息", batchIndex+1, i+1)
				msg.Format = message.FormatText
				msg.Targets = []target.Target{
					common.CreateFeishuTarget(),
				}
				messages[i] = msg
			}

			// 异步批量发送
			batchHandle, err := client.SendAsyncBatch(ctx, messages)
			if err != nil {
				logger.Error("批次 %d 发送失败: %v", batchIndex+1, err)
				return
			}

			logger.Info("批次 %d 已提交，BatchID: %s", batchIndex+1, batchHandle.BatchID())

			// 监控进度
			go func() {
				for progress := range batchHandle.Progress() {
					logger.Info("📊 批次 %d 进度: %d/%d (%.1f%%)",
						batchIndex+1, progress.Completed, progress.Total, progress.Progress*100)
				}
			}()

			// 等待批次完成
			receipts, err := batchHandle.Wait(ctx)
			if err != nil {
				logger.Error("批次 %d 等待失败: %v", batchIndex+1, err)
				return
			}

			successCount := 0
			for _, receipt := range receipts {
				if receipt != nil {
					successCount++
				}
			}

			logger.Success("✅ 批次 %d 完成: %d/%d 成功", batchIndex+1, successCount, len(receipts))
		}(batch)
	}

	// 等待所有批次完成
	wg.Wait()
	logger.Success("✅ 所有 %d 个批次已完成", batchCount)
	return nil
}

// mixedConcurrentMode demonstrates mixed concurrent patterns
func mixedConcurrentMode(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("混合并发模式演示")

	ctx := context.Background()
	var wg sync.WaitGroup

	logger.Info("🔀 启动混合并发模式：单条 + 批量")

	// 1. 并发单条消息 (高优先级)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			msg := message.New()
			msg.Title = fmt.Sprintf("⚡ 高优先级消息 #%d", index+1)
			msg.Body = "这是高优先级的单条消息"
			msg.Priority = message.PriorityHigh
			msg.Format = message.FormatText
			msg.Targets = []target.Target{
				common.CreateFeishuTarget(),
			}

			handle, err := client.SendAsync(ctx, msg)
			if err != nil {
				logger.Error("高优先级消息 %d 发送失败: %v", index+1, err)
				return
			}

			handle.OnComplete(func(receipt *receipt.Receipt) {
				logger.Success("🔥 高优先级消息 %d 发送成功", index+1)
			})

			_, _ = handle.Wait(ctx)
		}(i)
	}

	// 2. 并发批量消息 (普通优先级)
	for batch := 0; batch < 2; batch++ {
		wg.Add(1)
		go func(batchIndex int) {
			defer wg.Done()

			messages := make([]*message.Message, 8)
			for i := 0; i < 8; i++ {
				msg := message.New()
				msg.Title = fmt.Sprintf("📋 普通批量消息 B%d-M%d", batchIndex+1, i+1)
				msg.Body = "这是普通优先级的批量消息"
				msg.Priority = message.PriorityNormal
				msg.Format = message.FormatText
				msg.Targets = []target.Target{
					common.CreateFeishuTarget(),
				}
				messages[i] = msg
			}

			batchHandle, err := client.SendAsyncBatch(ctx, messages)
			if err != nil {
				logger.Error("普通批量 %d 发送失败: %v", batchIndex+1, err)
				return
			}

			// 简单等待，不监控详细进度
			receipts, _ := batchHandle.Wait(ctx)
			logger.Info("📦 普通批量 %d 完成: %d 条消息", batchIndex+1, len(receipts))
		}(batch)
	}

	// 3. 延迟发送 (低优先级)
	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(1 * time.Second) // 延迟1秒

		msg := message.New()
		msg.Title = "⏰ 延迟低优先级消息"
		msg.Body = "这是延迟发送的低优先级消息"
		msg.Priority = message.PriorityLow
		msg.Format = message.FormatText
		msg.Targets = []target.Target{
			common.CreateFeishuTarget(),
		}

		handle, err := client.SendAsync(ctx, msg)
		if err != nil {
			logger.Error("延迟消息发送失败: %v", err)
			return
		}

		handle.OnComplete(func(receipt *receipt.Receipt) {
			logger.Success("⏰ 延迟消息发送成功")
		})

		_, _ = handle.Wait(ctx)
	}()

	// 等待所有混合模式完成
	wg.Wait()
	logger.Success("✅ 混合并发模式完成")
	return nil
}

// goroutinePoolStressTest demonstrates goroutine pool under stress
func goroutinePoolStressTest(client notifyhub.Client, config *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("协程池压力测试")

	ctx := context.Background()
	totalMessages := 50 // 总消息数
	concurrency := 15   // 并发协程数
	var wg sync.WaitGroup

	logger.Info("🧪 压力测试: %d 条消息，%d 个并发协程", totalMessages, concurrency)
	logger.Info("测试开始前协程数: %d", runtime.NumGoroutine())

	// 信号量控制并发数
	sem := make(chan struct{}, concurrency)

	start := time.Now()

	// 发送所有消息
	for i := 0; i < totalMessages; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			msg := message.New()
			msg.Title = fmt.Sprintf("🧪 压力测试消息 #%d", index+1)
			msg.Body = fmt.Sprintf("压力测试消息 %d/%d", index+1, totalMessages)
			msg.Format = message.FormatText
			msg.Targets = []target.Target{
				common.CreateFeishuTarget(),
			}

			handle, err := client.SendAsync(ctx, msg)
			if err != nil {
				logger.Error("压力测试消息 %d 发送失败: %v", index+1, err)
				return
			}

			// 简单的回调
			handle.OnComplete(func(receipt *receipt.Receipt) {
				if index%10 == 0 { // 每10条消息打印一次
					logger.Info("✅ 压力测试进度: 消息 %d 发送成功", index+1)
				}
			})

			_, err = handle.Wait(ctx)
			if err != nil {
				logger.Error("压力测试消息 %d 等待失败: %v", index+1, err)
			}
		}(i)
	}

	// 等待所有消息完成
	wg.Wait()

	duration := time.Since(start)
	throughput := float64(totalMessages) / duration.Seconds()

	logger.Success("✅ 压力测试完成!")
	logger.Info("总时间: %v", duration)
	logger.Info("吞吐量: %.2f 消息/秒", throughput)
	logger.Info("测试完成后协程数: %d", runtime.NumGoroutine())

	return nil
}
