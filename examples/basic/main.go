package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub"
)

func main() {
	// 创建配置（从环境变量加载）
	// 需要设置：NOTIFYHUB_FEISHU_WEBHOOK_URL 和/或 NOTIFYHUB_SMTP_HOST 等
	hub, err := notifyhub.NewWithDefaults()
	if err != nil {
		log.Printf("Failed to create NotifyHub: %v", err)
		return
	}

	// 启动服务
	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Printf("Failed to start: %v", err)
		return
	}
	defer hub.Stop()

	// 发送简单文本消息
	err = hub.SendText(ctx, "Hello", "This is a test message",
		notifyhub.Target{Type: notifyhub.TargetTypeEmail, Value: "user@example.com"})
	if err != nil {
		log.Printf("Send failed: %v", err)
	}

	// 发送警报
	err = hub.SendAlert(ctx, "System Alert", "CPU usage is high",
		notifyhub.Target{Type: notifyhub.TargetTypeGroup, Value: "ops-team", Platform: "feishu"})
	if err != nil {
		log.Printf("Alert failed: %v", err)
	}

	// 使用模板发送消息
	err = hub.SendWithTemplate(ctx, "alert", map[string]interface{}{
		"server":      "web-01",
		"environment": "production",
	}, notifyhub.Target{Type: notifyhub.TargetTypeGroup, Value: "default", Platform: "feishu"})
	if err != nil {
		log.Printf("Template send failed: %v", err)
	}
}

func ExampleMessageBuilder() {
	hub, _ := notifyhub.NewWithDefaults()
	ctx := context.Background()
	hub.Start(ctx)
	defer hub.Stop()

	// 使用构建器模式创建复杂消息
	message := notifyhub.NewAlert("Production Issue", "Database connection lost").
		Variable("server", "db-01").
		Variable("error_code", 1042).
		Metadata("environment", "production").
		Metadata("severity", "critical").
		Email("dba@company.com").
		FeishuGroup("dba-alerts").
		Build()

	// 同步发送
	results, err := hub.Send(ctx, message, notifyhub.NewRetryOptions(3).WithTimeout(30*time.Second))

	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		for _, result := range results {
			fmt.Printf("Platform: %s, Success: %v, Duration: %v\n",
				result.Platform, result.Success, result.Duration)
		}
	}
}

func ExampleNotifyHub_async() {
	hub, _ := notifyhub.NewWithDefaults()
	ctx := context.Background()
	hub.Start(ctx)
	defer hub.Stop()

	// 异步发送报告
	message := notifyhub.NewReport("Daily Report", "All systems running normally").
		Variable("uptime", "99.9%").
		Variable("requests", 1500000).
		Email("reports@company.com").
		Build()

	// 异步发送（通过队列）
	results, err := hub.Send(ctx, message, notifyhub.NewAsyncOptions())

	if err != nil {
		log.Printf("Async send failed: %v", err)
	} else {
		fmt.Printf("Message queued successfully: %v\n", results[0].Success)
	}

	// 等待处理
	time.Sleep(2 * time.Second)

	// 检查指标
	metrics := hub.GetMetrics()
	fmt.Printf("Metrics: %+v\n", metrics)
}

func ExampleNotifyHub_health() {
	hub, _ := notifyhub.NewWithDefaults()
	ctx := context.Background()
	hub.Start(ctx)
	defer hub.Stop()

	// 检查健康状态
	health := hub.GetHealth(ctx)
	fmt.Printf("Health: %+v\n", health)

	// 获取详细指标
	metrics := hub.GetMetrics()
	fmt.Printf("Success Rate: %.2f%%\n", metrics["success_rate"].(float64)*100)
	fmt.Printf("Total Sent: %d\n", metrics["total_sent"])
	fmt.Printf("Uptime: %s\n", metrics["uptime"])
}
