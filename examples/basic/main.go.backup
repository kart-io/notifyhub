package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
)

func main() {
	// 创建配置（使用测试默认配置便于演示）
	hub, err := client.New(config.WithTestDefaults())
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
		notifiers.Target{Type: notifiers.TargetTypeEmail, Value: "user@example.com"})
	if err != nil {
		log.Printf("Send failed: %v", err)
	}

	// 发送警报
	err = hub.SendAlert(ctx, "System Alert", "CPU usage is high",
		notifiers.Target{Type: notifiers.TargetTypeGroup, Value: "ops-team", Platform: "test-mock"})
	if err != nil {
		log.Printf("Alert failed: %v", err)
	}

	// 使用模板发送消息
	err = hub.SendWithTemplate(ctx, "alert", map[string]interface{}{
		"server":      "web-01",
		"environment": "production",
	}, notifiers.Target{Type: notifiers.TargetTypeGroup, Value: "default", Platform: "test-mock"})
	if err != nil {
		log.Printf("Template send failed: %v", err)
	}
}

func ExampleMessageBuilder() {
	hub, _ := client.New(config.WithTestDefaults())
	ctx := context.Background()
	hub.Start(ctx)
	defer hub.Stop()

	// 使用构建器模式创建复杂消息
	message := client.NewAlert("Production Issue", "Database connection lost").
		Variable("server", "db-01").
		Variable("error_code", 1042).
		Metadata("environment", "production").
		Metadata("severity", "critical").
		Email("dba@company.com").
		FeishuGroup("dba-alerts").
		Build()

	// 同步发送
	options := &client.Options{
		Retry:      true,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}
	results, err := hub.Send(ctx, message, options)

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
	hub, _ := client.New(config.WithTestDefaults())
	ctx := context.Background()
	hub.Start(ctx)
	defer hub.Stop()

	// 异步发送报告
	message := client.NewReport("Daily Report", "All systems running normally").
		Variable("uptime", "99.9%").
		Variable("requests", 1500000).
		Email("reports@company.com").
		Build()

	// 异步发送
	messageID, err := hub.SendAsync(ctx, message, nil)

	if err != nil {
		log.Printf("Async send failed: %v", err)
	} else {
		fmt.Printf("Message queued successfully with ID: %s\n", messageID)
	}

	// 等待处理
	time.Sleep(2 * time.Second)

	// 检查指标
	metrics := hub.GetMetrics()
	fmt.Printf("Metrics: %+v\n", metrics)
}

func ExampleNotifyHub_health() {
	hub, _ := client.New(config.WithTestDefaults())
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
