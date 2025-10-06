// Package main demonstrates goroutine pool usage with NotifyHub
package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kart/notifyhub/examples/common"
	"github.com/kart/notifyhub/pkg/async"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/notifyhub"
	"github.com/kart/notifyhub/pkg/target"
)

func main() {
	// Print header
	common.PrintHeader("Goroutine Pool", "Demo")

	// Create logger
	logger := common.NewLogger(true)

	// Create example configuration
	exampleConfig := common.DefaultExampleConfig()

	// 请根据需要修改配置
	exampleConfig.Feishu.WebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url"

	// Check configuration
	if !common.CheckConfigurationPrompt("feishu") {
		logger.Info("已取消运行")
		return
	}

	logger.Info("初始协程数: %d", runtime.NumGoroutine())

	// Goroutine pool examples
	examples := []struct {
		name string
		fn   func(*common.ExampleConfig, *common.Logger) error
	}{
		{"协程池模式 vs 直接协程模式对比", comparePoolVsDirect},
		{"协程池配置演示", demonstratePoolConfiguration},
		{"高负载协程池测试", highLoadPoolTest},
		{"协程池资源管理", poolResourceManagement},
	}

	for _, example := range examples {
		logger.Info("\n" + strings.Repeat("=", 60))
		logger.Info("开始执行: %s", example.name)
		logger.Info("协程数 (开始前): %d", runtime.NumGoroutine())
		logger.Info(strings.Repeat("=", 60))

		start := time.Now()
		if err := example.fn(exampleConfig, logger); err != nil {
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

	logger.Success("所有协程池示例运行完成!")
	logger.Info("最终协程数: %d", runtime.NumGoroutine())
}

// comparePoolVsDirect compares goroutine pool mode vs direct goroutine mode
func comparePoolVsDirect(exampleConfig *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("协程池模式 vs 直接协程模式对比")

	messageCount := 50
	messages := createTestMessages(messageCount, "对比测试")

	// Test 1: Direct goroutine mode
	logger.Info("🔥 测试 1: 直接协程模式")
	cfg1 := exampleConfig.CreateFeishuConfig()
	cfg1.Async.Enabled = true
	cfg1.Async.UsePool = false // 禁用协程池
	cfg1.Async.Workers = 8
	cfg1.Logger.Level = "info"

	start1 := time.Now()
	goroutinesBefore1 := runtime.NumGoroutine()

	client1, err := notifyhub.NewClient(cfg1)
	if err != nil {
		return err
	}

	// 发送消息
	var wg1 sync.WaitGroup
	for i, msg := range messages {
		wg1.Add(1)
		go func(index int, message *message.Message) {
			defer wg1.Done()
			ctx := context.Background()
			handle, err := client1.SendAsync(ctx, message)
			if err != nil {
				logger.Error("直接模式消息 %d 发送失败: %v", index+1, err)
				return
			}
			_, _ = handle.Wait(ctx)
		}(i, msg)
	}

	wg1.Wait()
	_ = client1.Close()

	duration1 := time.Since(start1)
	goroutinesAfter1 := runtime.NumGoroutine()
	goroutinesDiff1 := goroutinesAfter1 - goroutinesBefore1

	logger.Info("直接协程模式结果:")
	logger.Info("  - 耗时: %v", duration1)
	logger.Info("  - 协程变化: %d", goroutinesDiff1)
	logger.Info("  - 平均每条消息: %v", duration1/time.Duration(messageCount))

	// 等待一下让协程清理
	time.Sleep(2 * time.Second)
	runtime.GC()

	// Test 2: Goroutine pool mode
	logger.Info("\n🏊 测试 2: 协程池模式")
	cfg2 := exampleConfig.CreateFeishuConfig()
	cfg2.Async.Enabled = true
	cfg2.Async.UsePool = true // 启用协程池
	cfg2.Async.Workers = 8
	cfg2.Async.BufferSize = 100
	cfg2.Logger.Level = "info"

	start2 := time.Now()
	goroutinesBefore2 := runtime.NumGoroutine()

	client2, err := notifyhub.NewClient(cfg2)
	if err != nil {
		return err
	}

	// 发送消息
	var wg2 sync.WaitGroup
	for i, msg := range messages {
		wg2.Add(1)
		go func(index int, message *message.Message) {
			defer wg2.Done()
			ctx := context.Background()
			handle, err := client2.SendAsync(ctx, message)
			if err != nil {
				logger.Error("协程池模式消息 %d 发送失败: %v", index+1, err)
				return
			}
			_, _ = handle.Wait(ctx)
		}(i, msg)
	}

	wg2.Wait()
	_ = client2.Close()

	duration2 := time.Since(start2)
	goroutinesAfter2 := runtime.NumGoroutine()
	goroutinesDiff2 := goroutinesAfter2 - goroutinesBefore2

	logger.Info("协程池模式结果:")
	logger.Info("  - 耗时: %v", duration2)
	logger.Info("  - 协程变化: %d", goroutinesDiff2)
	logger.Info("  - 平均每条消息: %v", duration2/time.Duration(messageCount))

	// 对比分析
	logger.Info("\n📊 对比分析:")
	if duration2 < duration1 {
		improvement := float64(duration1-duration2) / float64(duration1) * 100
		logger.Success("协程池模式快 %.1f%% (%v)", improvement, duration1-duration2)
	} else {
		degradation := float64(duration2-duration1) / float64(duration1) * 100
		logger.Info("直接协程模式快 %.1f%% (%v)", degradation, duration2-duration1)
	}

	goroutineImprovement := goroutinesDiff1 - goroutinesDiff2
	if goroutineImprovement > 0 {
		logger.Success("协程池模式减少了 %d 个协程创建", goroutineImprovement)
	} else {
		logger.Info("协程池模式增加了 %d 个协程", -goroutineImprovement)
	}

	return nil
}

// demonstratePoolConfiguration demonstrates different pool configurations
func demonstratePoolConfiguration(exampleConfig *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("协程池配置演示")

	messageCount := 30
	messages := createTestMessages(messageCount, "配置测试")

	configs := []struct {
		name    string
		workers int
		buffer  int
	}{
		{"小型池 (2 workers, 50 buffer)", 2, 50},
		{"中型池 (8 workers, 200 buffer)", 8, 200},
		{"大型池 (16 workers, 500 buffer)", 16, 500},
	}

	for _, poolConfig := range configs {
		logger.Info("🔧 测试配置: %s", poolConfig.name)

		cfg := exampleConfig.CreateFeishuConfig()
		cfg.Async.Enabled = true
		cfg.Async.UsePool = true
		cfg.Async.Workers = poolConfig.workers
		cfg.Async.BufferSize = poolConfig.buffer
		cfg.Logger.Level = "info"

		start := time.Now()
		goroutinesBefore := runtime.NumGoroutine()

		client, err := notifyhub.NewClient(cfg)
		if err != nil {
			logger.Error("创建客户端失败: %v", err)
			continue
		}

		// 批量发送
		ctx := context.Background()
		batchHandle, err := client.SendAsyncBatch(ctx, messages)
		if err != nil {
			logger.Error("批量发送失败: %v", err)
			_ = client.Close()
			continue
		}

		// 等待完成
		receipts, err := batchHandle.Wait(ctx)
		if err != nil {
			logger.Error("等待批量完成失败: %v", err)
		}

		_ = client.Close()

		duration := time.Since(start)
		goroutinesAfter := runtime.NumGoroutine()
		goroutinesDiff := goroutinesAfter - goroutinesBefore

		successCount := 0
		for _, receipt := range receipts {
			if receipt != nil {
				successCount++
			}
		}

		logger.Info("配置结果: %s", poolConfig.name)
		logger.Info("  - 成功: %d/%d", successCount, len(receipts))
		logger.Info("  - 耗时: %v", duration)
		logger.Info("  - 协程变化: %d", goroutinesDiff)
		logger.Info("  - 吞吐量: %.2f msg/s", float64(messageCount)/duration.Seconds())

		// 清理间隔
		time.Sleep(1 * time.Second)
		runtime.GC()
	}

	return nil
}

// highLoadPoolTest tests goroutine pool under high load
func highLoadPoolTest(exampleConfig *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("高负载协程池测试")

	// 创建大量消息
	messageCount := 200
	batchSize := 50
	messages := createTestMessages(messageCount, "高负载测试")

	cfg := exampleConfig.CreateFeishuConfig()
	cfg.Async.Enabled = true
	cfg.Async.UsePool = true
	cfg.Async.Workers = 12
	cfg.Async.BufferSize = 300
	cfg.Async.MinWorkers = 8
	cfg.Async.MaxWorkers = 20
	cfg.Logger.Level = "info"

	logger.Info("🚀 高负载测试配置:")
	logger.Info("  - 总消息数: %d", messageCount)
	logger.Info("  - 批次大小: %d", batchSize)
	logger.Info("  - 工作协程: %d", cfg.Async.Workers)
	logger.Info("  - 缓冲区大小: %d", cfg.Async.BufferSize)

	start := time.Now()
	goroutinesBefore := runtime.NumGoroutine()

	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	var wg sync.WaitGroup
	totalSuccess := 0
	var mutex sync.Mutex

	// 分批并发发送
	for i := 0; i < messageCount; i += batchSize {
		end := i + batchSize
		if end > messageCount {
			end = messageCount
		}

		batchMessages := messages[i:end]
		wg.Add(1)

		go func(batch []*message.Message, batchIndex int) {
			defer wg.Done()

			batchHandle, err := client.SendAsyncBatch(ctx, batch)
			if err != nil {
				logger.Error("批次 %d 发送失败: %v", batchIndex+1, err)
				return
			}

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

			mutex.Lock()
			totalSuccess += successCount
			mutex.Unlock()

			logger.Info("批次 %d 完成: %d/%d 成功", batchIndex+1, successCount, len(batch))
		}(batchMessages, i/batchSize)
	}

	// 等待所有批次完成
	wg.Wait()

	duration := time.Since(start)
	goroutinesAfter := runtime.NumGoroutine()
	goroutinesDiff := goroutinesAfter - goroutinesBefore

	logger.Success("高负载测试结果:")
	logger.Info("  - 总成功: %d/%d", totalSuccess, messageCount)
	logger.Info("  - 总耗时: %v", duration)
	logger.Info("  - 协程变化: %d", goroutinesDiff)
	logger.Info("  - 平均吞吐量: %.2f msg/s", float64(totalSuccess)/duration.Seconds())
	logger.Info("  - 成功率: %.1f%%", float64(totalSuccess)/float64(messageCount)*100)

	return nil
}

// poolResourceManagement demonstrates pool resource management
func poolResourceManagement(exampleConfig *common.ExampleConfig, logger *common.Logger) error {
	common.PrintSeparator("协程池资源管理")

	cfg := exampleConfig.CreateFeishuConfig()
	cfg.Async.Enabled = true
	cfg.Async.UsePool = true
	cfg.Async.Workers = 6
	cfg.Async.BufferSize = 100
	cfg.Logger.Level = "debug"

	logger.Info("🛠️ 创建协程池客户端...")
	goroutinesBefore := runtime.NumGoroutine()

	client, err := notifyhub.NewClient(cfg)
	if err != nil {
		return err
	}

	goroutinesAfterCreate := runtime.NumGoroutine()
	logger.Info("客户端创建后协程数: %d (增加 %d)", goroutinesAfterCreate, goroutinesAfterCreate-goroutinesBefore)

	// 发送一些消息
	messages := createTestMessages(20, "资源管理测试")
	ctx := context.Background()

	logger.Info("📤 发送测试消息...")
	handles := make([]async.Handle, len(messages))
	for i, msg := range messages {
		handle, err := client.SendAsync(ctx, msg)
		if err != nil {
			logger.Error("消息 %d 发送失败: %v", i+1, err)
			continue
		}
		handles[i] = handle
	}

	goroutinesAfterSend := runtime.NumGoroutine()
	logger.Info("消息发送后协程数: %d (增加 %d)", goroutinesAfterSend, goroutinesAfterSend-goroutinesAfterCreate)

	// 等待所有消息完成
	logger.Info("⏳ 等待消息处理完成...")
	for i, handle := range handles {
		if handle != nil {
			_, err := handle.Wait(ctx)
			if err != nil {
				logger.Error("消息 %d 等待失败: %v", i+1, err)
			}
		}
	}

	goroutinesAfterWait := runtime.NumGoroutine()
	logger.Info("消息完成后协程数: %d (变化 %d)", goroutinesAfterWait, goroutinesAfterWait-goroutinesAfterSend)

	// 关闭客户端
	logger.Info("🔒 关闭客户端...")
	_ = client.Close()

	// 等待资源清理
	time.Sleep(1 * time.Second)
	runtime.GC()

	goroutinesAfterClose := runtime.NumGoroutine()
	logger.Info("客户端关闭后协程数: %d (减少 %d)", goroutinesAfterClose, goroutinesAfterCreate-goroutinesAfterClose)

	logger.Success("资源管理测试完成:")
	logger.Info("  - 初始协程数: %d", goroutinesBefore)
	logger.Info("  - 创建后协程数: %d (+%d)", goroutinesAfterCreate, goroutinesAfterCreate-goroutinesBefore)
	logger.Info("  - 发送后协程数: %d (+%d)", goroutinesAfterSend, goroutinesAfterSend-goroutinesAfterCreate)
	logger.Info("  - 完成后协程数: %d (%d)", goroutinesAfterWait, goroutinesAfterWait-goroutinesAfterSend)
	logger.Info("  - 关闭后协程数: %d (-%d)", goroutinesAfterClose, goroutinesAfterCreate-goroutinesAfterClose)

	return nil
}

// Helper function to create test messages
func createTestMessages(count int, prefix string) []*message.Message {
	messages := make([]*message.Message, count)
	for i := 0; i < count; i++ {
		msg := message.New()
		msg.Title = fmt.Sprintf("🧪 %s #%d", prefix, i+1)
		msg.Body = fmt.Sprintf("这是第 %d 条%s消息，时间: %s", i+1, prefix, time.Now().Format("15:04:05.000"))
		msg.Format = message.FormatText
		msg.Targets = []target.Target{
			common.CreateFeishuTarget(),
		}
		messages[i] = msg
	}
	return messages
}
