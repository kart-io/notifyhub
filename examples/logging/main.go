package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/kart-io/notifyhub"
)

func main() {
	log.Println("🚀 NotifyHub 日志系统演示")
	log.Println("=================================")

	ctx := context.Background()

	// ========================================
	// 示例1：使用默认日志器（不同级别）
	// ========================================
	log.Println("\n📝 示例1: 使用默认日志器")
	log.Println("---------------------------------")

	// 创建一个带有Info级别日志的Hub
	hub1, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithDefaultLogger(notifyhub.LogLevelInfo), // Info级别日志
		notifyhub.WithQueue("memory", 100, 1),
	)
	if err != nil {
		log.Fatalf("Failed to create hub1: %v", err)
	}

	if err := hub1.Start(ctx); err != nil {
		log.Fatalf("Failed to start hub1: %v", err)
	}

	// 发送一条消息，观察日志输出
	message1 := notifyhub.NewAlert("测试消息", "这是一条Info级别的日志测试消息").
		FeishuGroup("test-group").
		Build()

	results1, err := hub1.Send(ctx, message1, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		log.Printf("消息发送成功，结果数量: %d", len(results1))
	}

	hub1.Stop()

	// ========================================
	// 示例2：使用Debug级别日志
	// ========================================
	log.Println("\n🐛 示例2: 使用Debug级别日志")
	log.Println("---------------------------------")

	hub2, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithDefaultLogger(notifyhub.LogLevelDebug), // Debug级别日志
		notifyhub.WithQueue("memory", 100, 1),
	)
	if err != nil {
		log.Fatalf("Failed to create hub2: %v", err)
	}

	if err := hub2.Start(ctx); err != nil {
		log.Fatalf("Failed to start hub2: %v", err)
	}

	// 发送消息，观察详细的debug日志
	message2 := notifyhub.NewReport("调试报告", "这是一条Debug级别的详细日志测试").
		Variable("debug_mode", true).
		Variable("timestamp", time.Now().Format(time.RFC3339)).
		FeishuGroup("debug-group").
		Build()

	results2, err := hub2.Send(ctx, message2, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		log.Printf("Debug消息发送成功，结果数量: %d", len(results2))
	}

	hub2.Stop()

	// ========================================
	// 示例3：使用自定义标准日志器
	// ========================================
	log.Println("\n📄 示例3: 使用自定义标准日志器")
	log.Println("---------------------------------")

	// 创建自定义的标准日志器
	customLogger := log.New(os.Stdout, "[NOTIFYHUB] ", log.LstdFlags|log.Lshortfile)

	hub3, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithLogger(
			notifyhub.NewStdLogAdapter(customLogger, notifyhub.LogLevelWarn),
		),
		notifyhub.WithQueue("memory", 100, 1),
	)
	if err != nil {
		log.Fatalf("Failed to create hub3: %v", err)
	}

	if err := hub3.Start(ctx); err != nil {
		log.Fatalf("Failed to start hub3: %v", err)
	}

	message3 := notifyhub.NewNotice("自定义日志", "使用自定义标准日志器的消息").
		FeishuGroup("custom-group").
		Build()

	_, err = hub3.Send(ctx, message3, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		log.Printf("自定义日志消息发送成功")
	}

	hub3.Stop()

	// ========================================
	// 示例4：使用函数适配器
	// ========================================
	log.Println("\n⚙️ 示例4: 使用函数适配器日志")
	log.Println("---------------------------------")

	// 创建自定义日志函数
	logFunc := func(level string, msg string, keyvals ...interface{}) {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		log.Printf("[%s] [%s] %s %v", timestamp, level, msg, keyvals)
	}

	hub4, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithLogger(
			notifyhub.NewFuncAdapter(logFunc, notifyhub.LogLevelInfo),
		),
		notifyhub.WithQueue("memory", 100, 1),
	)
	if err != nil {
		log.Fatalf("Failed to create hub4: %v", err)
	}

	if err := hub4.Start(ctx); err != nil {
		log.Fatalf("Failed to start hub4: %v", err)
	}

	message4 := notifyhub.NewAlert("函数适配器", "使用函数适配器的自定义日志格式").
		Variable("adapter_type", "function").
		FeishuGroup("func-group").
		Build()

	_, err = hub4.Send(ctx, message4, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		log.Printf("函数适配器日志消息发送成功")
	}

	hub4.Stop()

	// ========================================
	// 示例5：静默模式（无日志）
	// ========================================
	log.Println("\n🔇 示例5: 静默模式（无日志输出）")
	log.Println("---------------------------------")

	hub5, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithSilentLogger(), // 静默模式
		notifyhub.WithQueue("memory", 100, 1),
	)
	if err != nil {
		log.Fatalf("Failed to create hub5: %v", err)
	}

	if err := hub5.Start(ctx); err != nil {
		log.Fatalf("Failed to start hub5: %v", err)
	}

	message5 := notifyhub.NewNotice("静默消息", "这条消息不会产生任何NotifyHub日志").
		FeishuGroup("silent-group").
		Build()

	_, err = hub5.Send(ctx, message5, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		log.Printf("静默模式消息发送成功（注意：没有NotifyHub内部日志输出）")
	}

	hub5.Stop()

	// ========================================
	// 示例6：异步消息与日志
	// ========================================
	log.Println("\n⏳ 示例6: 异步消息处理日志")
	log.Println("---------------------------------")

	hub6, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithDefaultLogger(notifyhub.LogLevelInfo),
		notifyhub.WithQueue("memory", 200, 2), // 2个worker
	)
	if err != nil {
		log.Fatalf("Failed to create hub6: %v", err)
	}

	if err := hub6.Start(ctx); err != nil {
		log.Fatalf("Failed to start hub6: %v", err)
	}

	// 发送多个异步消息
	for i := 0; i < 3; i++ {
		asyncMessage := notifyhub.NewNotice("异步消息", "这是第%d条异步消息").
			Variable("message_id", i+1).
			Variable("timestamp", time.Now().Format(time.RFC3339)).
			FeishuGroup("async-group").
			Build()

		taskID, err := hub6.SendAsync(ctx, asyncMessage, notifyhub.NewAsyncOptions())
		if err != nil {
			log.Printf("异步发送失败: %v", err)
		} else {
			log.Printf("异步消息 %d 已入队，任务ID: %s", i+1, taskID)
		}
	}

	// 等待异步处理完成
	log.Println("等待异步消息处理...")
	time.Sleep(3 * time.Second)

	hub6.Stop()

	log.Println("\n🎉 NotifyHub 日志系统演示完成!")
	log.Println("=================================")
	log.Println("💡 总结:")
	log.Println("• 支持多种日志级别：Silent, Error, Warn, Info, Debug")
	log.Println("• 兼容标准log包、logrus、zap等主流日志库")
	log.Println("• 支持自定义日志函数")
	log.Println("• 提供详细的操作追踪和性能监控")
	log.Println("• 可完全静默或自定义日志格式")
}
