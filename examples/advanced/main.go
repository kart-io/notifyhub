package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub"
)

func main() {
	// 创建自定义重试策略
	customRetry := notifyhub.ExponentialBackoffPolicy(5, 10*time.Second, 1.5)

	// 使用option pattern创建配置
	hub, err := notifyhub.New(
		notifyhub.WithFeishu(
			"https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url",
			"", // secret (optional)
		),
		notifyhub.WithEmail(
			"smtp.gmail.com",       // host
			587,                    // port
			"your-email@gmail.com", // username
			"your-app-password",    // password
			"your-email@gmail.com", // from
		),
		notifyhub.WithQueue("memory", 1000, 2),
		notifyhub.WithQueueRetryPolicy(customRetry),
		notifyhub.WithRouting(
			notifyhub.NewRoutingRule("high_priority").
				WithPriority(4, 5).
				RouteTo("feishu", "email").
				Build(),
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
	message := notifyhub.NewAlert("系统告警", "数据库连接异常，请及时处理！").
		Variable("server", "prod-db-01").
		Variable("error", "connection timeout").
		Variable("environment", "production").
		Metadata("severity", "high").
		Metadata("category", "database").
		Email("admin@company.com").
		FeishuGroup("group-123").
		Build()

	// 同步发送
	results, err := hub.Send(ctx, message, notifyhub.NewRetryOptions(3).WithTimeout(30*time.Second))

	if err != nil {
		log.Printf("Send error: %v", err)
	} else {
		for _, result := range results {
			log.Printf("Result: %s - %t (took %v)", result.Platform, result.Success, result.Duration)
		}
	}

	// 异步发送
	taskID, err := hub.SendAsync(ctx, message, notifyhub.NewAsyncOptions())

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
