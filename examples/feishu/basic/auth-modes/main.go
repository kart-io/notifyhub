package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
	fmt.Println("🔐 飞书认证模式演示")
	fmt.Println("=================")

	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatal("请设置环境变量 FEISHU_WEBHOOK_URL")
	}

	ctx := context.Background()

	// 示例1: 无认证模式 (AuthModeNone)
	fmt.Println("\n1️⃣ 无认证模式")
	fmt.Println("-------------")
	runNoAuthExample(ctx, webhookURL)

	// 示例2: 签名认证模式 (AuthModeSignature)
	fmt.Println("\n2️⃣ 签名认证模式")
	fmt.Println("--------------")
	secret := os.Getenv("FEISHU_SECRET")
	if secret != "" {
		runSignatureAuthExample(ctx, webhookURL, secret)
	} else {
		fmt.Println("⚠️  未设置 FEISHU_SECRET，跳过签名认证示例")
	}

	// 示例3: 关键词认证模式 (AuthModeKeywords)
	fmt.Println("\n3️⃣ 关键词认证模式")
	fmt.Println("----------------")
	runKeywordsAuthExample(ctx, webhookURL)

	// 示例4: 显式设置认证模式
	fmt.Println("\n4️⃣ 显式设置认证模式")
	fmt.Println("------------------")
	runExplicitAuthModeExample(ctx, webhookURL)

	fmt.Println("\n✨ 认证模式演示完成!")
}

// runNoAuthExample 演示无认证模式
func runNoAuthExample(ctx context.Context, webhookURL string) {
	hub, err := notifyhub.NewHub(
		notifyhub.WithFeishuFromMap(map[string]interface{}{
			"webhook_url": webhookURL,
			"auth_mode":   "none", // 显式设置为无认证
		}),
		notifyhub.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Printf("创建Hub失败: %v", err)
		return
	}
	defer hub.Close(ctx)

	message := notifyhub.NewMessage("无认证模式测试").
		WithText("这是无认证模式下的测试消息").
		ToFeishuGroup("oc_test_group").
		Build()

	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		fmt.Printf("✅ 发送成功 - 消息ID: %s\n", receipt.MessageID)
	}
}

// runSignatureAuthExample 演示签名认证模式
func runSignatureAuthExample(ctx context.Context, webhookURL, secret string) {
	hub, err := notifyhub.NewHub(
		notifyhub.WithFeishuFromMap(map[string]interface{}{
			"webhook_url": webhookURL,
			"secret":      secret,
			"auth_mode":   "signature", // 显式设置为签名认证
		}),
		notifyhub.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Printf("创建Hub失败: %v", err)
		return
	}
	defer hub.Close(ctx)

	message := notifyhub.NewMessage("签名认证模式测试").
		WithText("这是签名认证模式下的测试消息，包含时间戳和签名").
		ToFeishuGroup("oc_test_group").
		Build()

	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		fmt.Printf("✅ 发送成功 - 消息ID: %s\n", receipt.MessageID)
	}
}

// runKeywordsAuthExample 演示关键词认证模式
func runKeywordsAuthExample(ctx context.Context, webhookURL string) {
	hub, err := notifyhub.NewHub(
		notifyhub.WithFeishuFromMap(map[string]interface{}{
			"webhook_url": webhookURL,
			"keywords":    []string{"告警", "通知"},
			"auth_mode":   "keywords", // 显式设置为关键词认证
		}),
		notifyhub.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Printf("创建Hub失败: %v", err)
		return
	}
	defer hub.Close(ctx)

	message := notifyhub.NewMessage("关键词认证模式测试").
		WithText("这是关键词认证模式下的测试消息，消息会自动添加关键词").
		ToFeishuGroup("oc_test_group").
		Build()

	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		fmt.Printf("✅ 发送成功 - 消息ID: %s\n", receipt.MessageID)
	}
}

// runExplicitAuthModeExample 演示显式设置认证模式
func runExplicitAuthModeExample(ctx context.Context, webhookURL string) {
	// 示例: 即使提供了secret和keywords，也可以显式设置为无认证模式
	hub, err := notifyhub.NewHub(
		notifyhub.WithFeishuFromMap(map[string]interface{}{
			"webhook_url": webhookURL,
			"secret":      "some-secret",    // 提供了secret
			"keywords":    []string{"test"}, // 也提供了keywords
			"auth_mode":   "none",           // 但显式设置为无认证模式
		}),
		notifyhub.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Printf("创建Hub失败: %v", err)
		return
	}
	defer hub.Close(ctx)

	message := notifyhub.NewMessage("显式认证模式测试").
		WithText("显式设置为无认证模式，忽略secret和keywords配置").
		ToFeishuGroup("oc_test_group").
		Build()

	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		fmt.Printf("✅ 发送成功 - 消息ID: %s\n", receipt.MessageID)
	}

	// 演示配置错误的情况
	fmt.Println("\n🚨 演示配置错误:")
	_, err = notifyhub.NewHub(
		notifyhub.WithFeishuFromMap(map[string]interface{}{
			"webhook_url": webhookURL,
			"secret":      "some-secret",
			"keywords":    []string{"test"},
			// 不显式设置auth_mode，同时提供secret和keywords会导致错误
		}),
	)
	if err != nil {
		fmt.Printf("❌ 预期的配置错误: %v\n", err)
	}
}
