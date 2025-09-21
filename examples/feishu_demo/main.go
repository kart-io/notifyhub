package main

import (
	"context"
	"log"
	"os"

	"github.com/kart-io/notifyhub"

	"github.com/kart-io/notifyhub/platforms/email"
	"github.com/kart-io/notifyhub/platforms/feishu"
	"github.com/kart-io/notifyhub/platforms/registry"
)

func main() {
	feishuDemo()
}

func init() {
	// Register built-in platforms
	if err := registry.Register(feishu.NewFeishuPlatform()); err != nil {
		log.Fatalf("failed to register feishu platform: %v", err)
	}
	if err := registry.Register(email.NewEmailPlatform()); err != nil {
		log.Fatalf("failed to register email platform: %v", err)
	}
}

func feishuDemo() {
	// 使用环境变量配置，或使用默认测试值
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	secret := os.Getenv("FEISHU_SECRET")

	// 如果没有设置环境变量，使用示例中的测试值
	if webhookURL == "" {
		webhookURL = ""
	}
	if secret == "" {
		secret = ""
	}

	log.Printf("Using webhook URL: %s", webhookURL)
	log.Printf("Using secret: %s", secret)

	client, err := notifyhub.New(
		notifyhub.WithFeishu(webhookURL, secret),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	result, err := client.Send(ctx).
		Title("Hello World").
		Body("This is a test notification").
		ToFeishu("webhook-id").
		Execute()

	if err != nil {
		log.Printf("❌ 发送失败: %v", err)
		log.Printf("\n📋 故障排除:")
		log.Printf("1. 确认webhook URL是否有效")
		log.Printf("2. 确认secret是否正确")
		log.Printf("3. 确认飞书机器人是否已启用")
		log.Printf("4. 尝试设置环境变量:")
		log.Printf("   export FEISHU_WEBHOOK_URL=\"your-webhook-url\"")
		log.Printf("   export FEISHU_SECRET=\"your-secret\"")
		return
	}

	log.Printf("✅ 消息发送成功! 结果: %+v", result)
}
