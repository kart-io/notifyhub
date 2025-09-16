package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
)

func main() {
	// 使用测试配置创建hub，演示高级功能
	hub, err := client.New(
		config.WithTestDefaults(),
		config.WithFeishu(
			"https://httpbin.org/post", // 用于演示的测试端点
			"", // secret (optional)
		),
	)
	if err != nil {
		log.Fatalf("Failed to create NotifyHub: %v", err)
	}

	// 启动服务
	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("Failed to start NotifyHub: %v", err)
	}
	defer hub.Stop()

	// 发送带有多个目标的复杂消息
	message := client.NewAlert("系统告警", "数据库连接异常，请及时处理！").
		Variable("server", "prod-db-01").
		Variable("error", "connection timeout").
		Variable("environment", "production").
		Metadata("severity", "high").
		Metadata("category", "database").
		Email("admin@company.com").
		FeishuGroup("group-123").
		Build()

	// 同步发送
	options := &client.Options{
		Retry:      true,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}
	results, err := hub.Send(ctx, message, options)

	if err != nil {
		log.Printf("Send error: %v", err)
	} else {
		for _, result := range results {
			log.Printf("Result: %s - %t (took %v)", result.Platform, result.Success, result.Duration)
		}
	}

	// 异步发送
	taskID, err := hub.SendAsync(ctx, message, &client.Options{Async: true})

	if err != nil {
		log.Printf("Async send error: %v", err)
	} else {
		log.Printf("Async task queued: %s", taskID)
	}

	// 获取监控指标
	metrics := hub.GetMetrics()
	log.Printf("Metrics: %+v", metrics)

	// 健康检查
	health := hub.GetHealth(ctx)
	log.Printf("Health: %+v", health)

	// 等待一段时间以便观察异步消息处理
	time.Sleep(5 * time.Second)
}
