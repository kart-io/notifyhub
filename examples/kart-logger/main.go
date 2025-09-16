package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
)

func main() {
	// 创建NotifyHub实例，使用Kart日志器风格的配置
	hub, err := client.New(config.WithTestDefaults())
	if err != nil {
		log.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("Failed to start NotifyHub: %v", err)
	}
	defer hub.Stop()

	log.Println("=== Kart Logger 风格示例 ===")

	// 发送带有Kart风格标识的通知
	message := client.NewAlert("Kart System Alert", "系统监控发现异常").
		Email("admin@example.com").
		Variable("service", "kart-core").
		Variable("severity", "high").
		Variable("timestamp", time.Now().Format(time.RFC3339)).
		Metadata("source", "kart-monitor").
		Metadata("component", "kart-logger").
		Build()

	results, err := hub.Send(ctx, message, &client.Options{
		Retry:      true,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	})

	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		log.Printf("发送成功，结果数量: %d", len(results))
		for _, result := range results {
			status := "失败"
			if result.Success {
				status = "成功"
			}
			log.Printf("  - 平台: %s, 状态: %s, 耗时: %v",
				result.Platform, status, result.Duration)
		}
	}

	// 发送Kart风格的性能报告
	reportMessage := client.NewReport("Kart Performance Report", "系统性能监控报告").
		Email("ops@example.com").
		Variable("cpu_usage", "45%").
		Variable("memory_usage", "67%").
		Variable("disk_usage", "23%").
		Variable("uptime", "99.95%").
		Metadata("report_type", "performance").
		Metadata("source", "kart-logger").
		Build()

	_, err = hub.Send(ctx, reportMessage, nil)
	if err != nil {
		log.Printf("性能报告发送失败: %v", err)
	} else {
		log.Println("性能报告发送成功")
	}

	// 获取系统指标
	metrics := hub.GetMetrics()
	log.Printf("系统指标: %+v", metrics)

	log.Println("Kart Logger 示例执行完成")
}