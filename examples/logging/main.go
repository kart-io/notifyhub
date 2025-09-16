package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
)

func main() {
	log.Println("=== NotifyHub 日志系统示例 ===")

	// 创建NotifyHub实例（带有详细日志）
	hub, err := client.New(config.WithTestDefaults())
	if err != nil {
		log.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("Failed to start NotifyHub: %v", err)
	}
	defer hub.Stop()

	// 示例1：基本日志消息
	log.Println("\n--- 示例1：基本日志消息 ---")
	message1 := client.NewNotice("日志测试消息", "测试NotifyHub的日志功能").
		Email("logs@example.com").
		Variable("log_level", "INFO").
		Variable("source", "logging-example").
		Build()

	results, err := hub.Send(ctx, message1, nil)
	if err != nil {
		log.Printf("消息发送失败: %v", err)
	} else {
		log.Printf("消息发送成功，共%d个结果", len(results))
	}

	// 示例2：错误日志消息
	log.Println("\n--- 示例2：错误日志消息 ---")
	errorMessage := client.NewAlert("系统错误", "检测到系统异常").
		Email("alerts@example.com").
		Variable("error_code", "E001").
		Variable("severity", "HIGH").
		Variable("timestamp", time.Now().Format(time.RFC3339)).
		Metadata("source", "error-monitor").
		Build()

	_, err = hub.Send(ctx, errorMessage, &client.Options{
		Retry:      true,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	})

	if err != nil {
		log.Printf("错误消息发送失败: %v", err)
	} else {
		log.Println("错误消息发送成功")
	}

	// 示例3：性能日志
	log.Println("\n--- 示例3：性能监控日志 ---")
	perfMessage := client.NewReport("性能报告", "系统性能监控结果").
		Email("performance@example.com").
		Variable("response_time", "250ms").
		Variable("throughput", "1200 req/s").
		Variable("error_rate", "0.02%").
		Metadata("report_type", "performance").
		Build()

	_, err = hub.Send(ctx, perfMessage, nil)
	if err != nil {
		log.Printf("性能报告发送失败: %v", err)
	} else {
		log.Println("性能报告发送成功")
	}

	// 获取系统指标
	metrics := hub.GetMetrics()
	log.Printf("\n系统指标: %+v", metrics)

	// 健康检查
	health := hub.GetHealth(ctx)
	log.Printf("系统健康状况: %+v", health)

	log.Println("\n日志系统示例执行完成")
}