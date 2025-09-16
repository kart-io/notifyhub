package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
)

func main() {
	// 创建NotifyHub实例
	hub, err := client.New(config.WithTestDefaults())
	if err != nil {
		log.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("Failed to start NotifyHub: %v", err)
	}
	defer hub.Stop()

	// 创建发送选项
	options := &client.Options{
		Async:      true,
		Retry:      true,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}

	// 创建消息
	message := client.NewAlert("回调测试", "这是一条用于测试回调功能的消息").
		Email("test@example.com").
		Variable("test_time", time.Now().Format(time.RFC3339)).
		Build()

	// 异步发送消息
	taskID, err := hub.SendAsync(ctx, message, options)
	if err != nil {
		log.Printf("发送消息失败: %v", err)
	} else {
		log.Printf("消息已加入队列，任务ID: %s", taskID)
	}

	// 等待处理完成
	time.Sleep(3 * time.Second)

	// 检查指标
	metrics := hub.GetMetrics()
	log.Printf("指标: %+v", metrics)

	log.Println("回调示例执行完成")
}