package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kart-io/notifyhub/notifiers"
)

// 飞书简单示例（不使用签名验证）
// 适用于快速测试和本地开发
func main() {
	fmt.Println("=== 飞书简单示例（不使用签名验证）===")

	// 检查环境变量
	webhookURL := "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url"
	fmt.Printf("使用 webhook URL: %s\n", webhookURL)
	fmt.Println("注意：这个示例不使用签名验证，仅适用于开发测试环境")

	// 方法1: 使用最简单的创建方式
	fmt.Println("\n--- 方法1: 最简单的创建方式 ---")
	simpleNotifier := notifiers.NewFeishuNotifierSimple(webhookURL)
	fmt.Printf("创建的通知器: %s\n", simpleNotifier.Name())

	// 方法2: 使用自定义超时的创建方式
	fmt.Println("\n--- 方法2: 自定义超时创建方式 ---")
	timeoutNotifier := notifiers.NewFeishuNotifierWithTimeout(webhookURL, 60*1000*1000*1000) // 60秒
	fmt.Printf("创建的通知器: %s\n", timeoutNotifier.Name())

	// 创建消息并发送
	ctx := context.Background()

	// 基本文本消息
	message := &notifiers.Message{
		Title:  "简单测试消息",
		Body:   "这是一个使用简单方法创建的飞书通知器发送的测试消息",
		Format: notifiers.FormatText,
		Targets: []notifiers.Target{
			{Type: notifiers.TargetTypeGroup, Value: "default", Platform: "feishu"},
		},
	}

	fmt.Println("\n--- 发送测试消息 ---")
	results, err := simpleNotifier.Send(ctx, message)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 消息发送成功 - 平台: %s, 耗时: %v\n",
					result.Platform, result.Duration)
			} else {
				fmt.Printf("❌ 消息发送失败 - 平台: %s, 错误: %s\n",
					result.Platform, result.Error)
			}
		}
	}

	// 健康检查
	fmt.Println("\n--- 健康检查 ---")
	if err := simpleNotifier.Health(ctx); err != nil {
		fmt.Printf("❌ 健康检查失败: %v\n", err)
	} else {
		fmt.Println("✅ 健康检查通过")
	}

	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("提示：")
	fmt.Println("- 这个示例使用了便捷方法，不需要配置 secret")
	fmt.Println("- 适用于本地开发和测试环境")
	fmt.Println("- 生产环境建议使用带签名验证的方法")
}
