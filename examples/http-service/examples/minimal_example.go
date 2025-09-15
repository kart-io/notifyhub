package main

import (
	"context"
	"log"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
)

// ==========================================
// 最小化示例 - 5行代码完成通知发送
// ==========================================

func main() {
	// 1. 创建并启动 Hub（自动加载环境变量配置）
	hub, _ := client.NewAndStart(context.Background(), config.WithDefaults())
	defer hub.Stop()

	// 2. 发送简单文本消息
	hub.SendText(context.Background(), "Hello", "NotifyHub is working!",
		notifiers.Target{Type: notifiers.TargetTypeEmail, Value: "admin@example.com"})

	// 3. 发送紧急告警
	client.QuickAlert("ALERT", "Service is down").
		Email("oncall@example.com").
		ToSlack("#incidents").
		Build()

	// 4. 批量邮件
	client.BatchEmail("Newsletter", "Weekly updates",
		"user1@example.com", "user2@example.com", "user3@example.com").
		Build()

	log.Println("✅ Notifications sent!")
}