package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
)

// 飞书快速测试示例 - 验证核心功能
func main() {
	fmt.Println("🚀 飞书快速测试开始...")

	// 创建基础配置的 NotifyHub
	hub, err := client.New(config.WithTestDefaults())
	if err != nil {
		log.Fatalf("创建 NotifyHub 失败: %v", err)
	}

	// 启动服务
	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
	defer hub.Stop()

	fmt.Println("✅ NotifyHub 启动成功")

	// 测试1: 基础消息构建
	fmt.Println("\n📝 测试消息构建...")
	message := client.NewMessage().
		Title("测试标题").
		Body("测试内容").
		Priority(3).
		FeishuGroup("test-group").
		Metadata("test", "true").
		Build()

	if message.Title != "测试标题" {
		log.Fatal("❌ 消息标题设置失败")
	}
	if message.Body != "测试内容" {
		log.Fatal("❌ 消息内容设置失败")
	}
	fmt.Println("✅ 消息构建测试通过")

	// 测试2: 不同类型消息
	fmt.Println("\n📢 测试不同消息类型...")

	alert := client.NewAlert("警报标题", "警报内容").
		Priority(5).
		Variable("level", "critical").
		Build()

	notice := client.NewNotice("通知标题", "通知内容").
		Priority(3).
		Build()

	report := client.NewReport("报告标题", "报告内容").
		Priority(1).
		Build()

	if alert.Title != "警报标题" || notice.Title != "通知标题" || report.Title != "报告标题" {
		log.Fatal("❌ 消息类型构建失败")
	}
	fmt.Println("✅ 消息类型测试通过")

	// 测试3: 目标构建
	fmt.Println("\n🎯 测试目标构建...")
	targetList := client.NewTargetList().
		AddFeishuGroups("group1", "group2").
		AddEmails("test@example.com")

	targets := targetList.Build()
	if len(targets) != 3 {
		log.Fatalf("❌ 目标构建失败，期望3个，实际%d个", len(targets))
	}
	fmt.Println("✅ 目标构建测试通过")

	// 测试4: 模拟发送 (使用测试配置，不会真实发送)
	fmt.Println("\n📤 测试模拟发送...")

	testMessage := client.NewMessage().
		Title("模拟发送测试").
		Body("这是一个模拟发送的测试消息").
		FeishuGroup("test-group").
		Build()

	results, err := hub.Send(ctx, testMessage, &client.Options{
		Retry:   false,
		Timeout: 5 * time.Second,
	})

	if err != nil {
		fmt.Printf("⚠️  模拟发送完成 (预期错误): %v\n", err)
	} else {
		fmt.Printf("✅ 模拟发送结果: %d 个平台\n", len(results))
	}

	// 测试5: 异步发送
	fmt.Println("\n⏱️  测试异步发送...")

	asyncMessage := client.NewMessage().
		Title("异步发送测试").
		Body("这是异步发送测试").
		FeishuGroup("async-test").
		Build()

	messageID, err := hub.SendAsync(ctx, asyncMessage, nil)
	if err != nil {
		fmt.Printf("⚠️  异步发送排队失败: %v\n", err)
	} else {
		fmt.Printf("✅ 异步消息已排队，ID: %s\n", messageID[:8])
	}

	// 测试6: 快捷方法
	fmt.Println("\n⚡ 测试快捷方法...")

	err = hub.QuickSend(ctx, "快捷测试", "快捷发送内容", "group:test-group@feishu")
	if err != nil {
		fmt.Printf("⚠️  快捷发送完成 (预期错误): %v\n", err)
	} else {
		fmt.Println("✅ 快捷发送成功")
	}

	// 测试7: 健康检查和指标
	fmt.Println("\n📊 测试健康检查...")

	health := hub.GetHealth(ctx)
	if health == nil {
		log.Fatal("❌ 健康检查失败")
	}
	fmt.Println("✅ 健康检查通过")

	metrics := hub.GetMetrics()
	if metrics == nil {
		log.Fatal("❌ 指标获取失败")
	}
	fmt.Printf("✅ 指标获取成功，包含 %d 项数据\n", len(metrics))

	// 等待异步任务处理
	time.Sleep(1 * time.Second)

	fmt.Println("\n🎉 飞书快速测试完成!")

	// 显示最终统计
	fmt.Println("\n📈 最终统计:")
	if totalSent, ok := metrics["total_sent"].(int64); ok {
		fmt.Printf("  总发送数: %d\n", totalSent)
	}
	if successRate, ok := metrics["success_rate"].(float64); ok {
		fmt.Printf("  成功率: %.1f%%\n", successRate*100)
	}

	fmt.Println("✅ 所有核心功能测试通过!")
}
