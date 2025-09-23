package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
	// 从环境变量获取飞书配置
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	secret := os.Getenv("FEISHU_SECRET")

	if webhookURL == "" {
		log.Fatal("❌ 请设置环境变量 FEISHU_WEBHOOK_URL")
	}

	fmt.Println("🧪 飞书真实发送测试")
	fmt.Println("==================")
	fmt.Printf("Webhook URL: %s\n", webhookURL)
	fmt.Printf("Secret: %s\n", secret)
	fmt.Println()

	// 首先测试 Webhook URL 的连通性
	fmt.Println("🔗 测试 Webhook 连通性...")
	resp, err := http.Get(webhookURL)
	if err != nil {
		fmt.Printf("❌ Webhook URL 不可达: %v\n", err)
		fmt.Println("请检查:")
		fmt.Println("1. URL 是否正确")
		fmt.Println("2. 网络连接是否正常")
		fmt.Println("3. 飞书服务是否可访问")
		return
	}
	resp.Body.Close()
	fmt.Printf("✅ Webhook URL 可达 (HTTP %d)\n\n", resp.StatusCode)

	// 创建 NotifyHub 实例
	fmt.Println("🚀 创建 NotifyHub 实例...")
	hub, err := notifyhub.NewHub(
		notifyhub.WithFeishuFromMap(map[string]interface{}{
			"webhook_url": webhookURL,
			"secret":      secret,
		}),
		notifyhub.WithTimeout(15000), // 15秒超时
	)
	if err != nil {
		log.Fatalf("❌ 创建 NotifyHub 失败: %v", err)
	}
	defer func() {
		if err := hub.Close(context.Background()); err != nil {
			log.Printf("关闭 Hub 失败: %v", err)
		}
	}()

	ctx := context.Background()

	// 测试系统健康检查
	fmt.Println("🔍 系统健康检查...")
	health, err := hub.Health(ctx)
	if err != nil {
		fmt.Printf("❌ 健康检查失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 系统状态: %s\n", health.Status)
	for platform, status := range health.Platforms {
		statusText := "🟢 健康"
		if !status.Available {
			statusText = fmt.Sprintf("🔴 异常: %s", status.Status)
		}
		fmt.Printf("  %s: %s\n", platform, statusText)
	}
	fmt.Println()

	// 发送测试消息
	fmt.Println("📝 发送测试消息...")

	// 注意: 这里使用一个通用的群组 ID 前缀，实际使用时需要替换为真实的群组 ID
	testMessage := notifyhub.NewMessage("NotifyHub 测试").
		WithMarkdown("## 🧪 测试消息\n\n这是来自 **NotifyHub** 的测试消息。\n\n- 时间: "+
			fmt.Sprintf("%s", health.Timestamp.Format("2006-01-02 15:04:05"))+"\n- 状态: ✅ 正常").
		WithMetadata("test_id", "real-test-001").
		Build()

	// 由于我们不知道真实的群组 ID，这里先尝试发送到一个测试 ID
	// 实际使用时，用户需要替换为真实的群组或用户 ID
	testMessage.Targets = []notifyhub.Target{
		notifyhub.AutoDetectTarget("oc_test_group"), // 替换为真实群组 ID
	}

	receipt, err := hub.Send(ctx, testMessage)
	if err != nil {
		fmt.Printf("❌ 消息发送失败: %v\n", err)
		fmt.Println("\n🔧 故障排除建议:")
		fmt.Println("1. 检查 Webhook URL 是否正确")
		fmt.Println("2. 检查签名密钥是否匹配")
		fmt.Println("3. 确认机器人已添加到目标群聊")
		fmt.Println("4. 检查群组/用户 ID 是否正确")
		fmt.Println("5. 确认网络连接正常")

		// 检查是否是特定的错误类型
		if notifyhub.IsValidationError(err) {
			fmt.Println("\n📝 验证错误: 请检查消息格式")
		}

		return
	}

	// 发送成功
	fmt.Printf("✅ 消息发送成功!\n")
	fmt.Printf("  消息 ID: %s\n", receipt.MessageID)
	fmt.Printf("  状态: %s\n", receipt.Status)
	fmt.Printf("  总计: %d, 成功: %d, 失败: %d\n",
		receipt.Total, receipt.Successful, receipt.Failed)

	if len(receipt.Results) > 0 {
		fmt.Println("  详细结果:")
		for _, result := range receipt.Results {
			status := "✅"
			if !result.Success {
				status = "❌"
			}
			fmt.Printf("    %s %s -> %s (%dms)\n",
				status, result.Target, result.Platform, result.Duration.Milliseconds())
		}
	}

	// 如果有错误，显示错误信息
	if receipt.Error != nil {
		fmt.Printf("  错误: %s\n", receipt.Error)
	}

	fmt.Println("\n✨ 测试完成!")
	fmt.Println("\n📋 使用真实配置的步骤:")
	fmt.Println("1. 在飞书中创建群聊机器人，获取 Webhook URL")
	fmt.Println("2. 设置环境变量:")
	fmt.Println("   export FEISHU_WEBHOOK_URL=\"your-real-webhook-url\"")
	fmt.Println("   export FEISHU_SECRET=\"your-real-secret\"")
	fmt.Println("3. 修改代码中的群组/用户 ID 为真实值")
	fmt.Println("4. 重新运行: go run test-real.go")
}
