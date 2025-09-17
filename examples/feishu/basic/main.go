package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
)

// 飞书基础发送示例
func main() {
	// 从环境变量获取飞书配置
	// 需要设置环境变量：
	// NOTIFYHUB_FEISHU_WEBHOOK_URL=https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url
	// NOTIFYHUB_FEISHU_SECRET=your-webhook-secret

	// 创建 NotifyHub 实例，配置飞书通知器
	hub, err := client.New(
		// 从环境变量自动加载飞书配置（包括签名）
		config.WithFeishuFromEnv(),
		// 配置队列
		config.WithQueue("memory", 1000, 4),
	)
	if err != nil {
		log.Fatalf("创建 NotifyHub 失败: %v", err)
	}

	// 启动服务
	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
	defer hub.Stop()

	// 示例1: 发送简单文本消息到飞书群
	fmt.Println("=== 发送简单文本消息 ===")
	err = hub.FeishuGroup(ctx, "系统通知", "Hello，这是一条来自 NotifyHub 的测试消息！", "default")
	if err != nil {
		log.Printf("发送简单消息失败: %v", err)
	} else {
		fmt.Println("✅ 简单消息发送成功")
	}

	// 示例2: 发送富文本消息
	fmt.Println("\n=== 发送富文本消息 ===")
	message := client.NewMessage().
		Title("📢 系统公告").
		Body("系统将于今晚 22:00 进行维护，预计持续 4 小时").
		Format(notifiers.FormatText).
		Priority(4). // 1=low, 3=normal, 4=high, 5=urgent
		FeishuGroup("default").
		Metadata("source", "ops-team").
		Build()

	results, err := hub.Send(ctx, message, nil)
	if err != nil {
		log.Printf("发送富文本消息失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 平台: %s, 成功: %v, 耗时: %v\n",
					result.Platform, result.Success, result.Duration)
			} else {
				fmt.Printf("❌ 平台: %s, 成功: %v, 耗时: %v, 错误: %s\n",
					result.Platform, result.Success, result.Duration, result.Error)
			}
		}
	}

	// 示例3: 发送警报消息
	fmt.Println("\n=== 发送警报消息 ===")
	alert := client.NewAlert("🚨 系统警报", "CPU 使用率超过 85%").
		Variable("server", "web-server-02").
		Variable("cpu_usage", "87.5%").
		Variable("threshold", "85%").
		Variable("timestamp", time.Now().Format("2006-01-02 15:04:05")).
		FeishuGroup("default").
		Priority(5). // 5=urgent
		Build()

	results, err = hub.Send(ctx, alert, &client.Options{
		Retry:      true,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	})
	if err != nil {
		log.Printf("发送警报失败: %v", err)
	} else {
		successCount := 0
		failCount := 0
		for _, result := range results {
			if result.Success {
				successCount++
			} else {
				failCount++
				fmt.Printf("❌ 警报发送失败: %s\n", result.Error)
			}
		}
		fmt.Printf("✅ 警报发送完成，成功: %d, 失败: %d\n", successCount, failCount)
	}

	// 示例4: 发送通知给特定用户
	fmt.Println("\n=== 发送个人通知 ===")
	personalNotice := client.NewNotice("📋 任务提醒", "您有一个新的代码审查任务等待处理").
		Variable("task_id", "CR-2024-001").
		Variable("assignee", "@张三").
		Variable("due_date", "2024-01-20").
		FeishuUser("default"). // 发送给特定用户
		Build()

	messageID, err := hub.SendAsync(ctx, personalNotice, nil)
	if err != nil {
		log.Printf("发送个人通知失败: %v", err)
	} else {
		fmt.Printf("✅ 个人通知已排队，消息 ID: %s\n", messageID)
	}

	// 示例5: 发送卡片消息
	fmt.Println("\n=== 发送卡片消息 ===")
	cardMessage := client.NewCard("🎯 性能监控报告", "系统性能数据概览").
		Metadata("服务器", "web-01").
		Metadata("状态", "运行正常").
		Metadata("CPU使用率", "45%").
		Metadata("内存使用率", "68%").
		Metadata("响应时间", "120ms").
		Priority(3).
		FeishuGroup("default").
		Build()

	results, err = hub.Send(ctx, cardMessage, nil)
	if err != nil {
		log.Printf("发送卡片消息失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 平台: %s, 成功: %v, 耗时: %v\n",
					result.Platform, result.Success, result.Duration)
			} else {
				fmt.Printf("❌ 平台: %s, 成功: %v, 耗时: %v, 错误: %s\n",
					result.Platform, result.Success, result.Duration, result.Error)
			}
		}
	}

	// 示例6: 自定义卡片结构
	fmt.Println("\n=== 发送自定义卡片消息 ===")
	customCardData := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**📊 自定义监控面板**",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "hr",
			},
			{
				"tag": "div",
				"fields": []map[string]interface{}{
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**服务状态**\n🟢 正常运行",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**响应时间**\n⚡ 95ms",
							"tag":     "lark_md",
						},
					},
				},
			},
			{
				"tag": "action",
				"actions": []map[string]interface{}{
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "查看详情",
							"tag":     "plain_text",
						},
						"type": "primary",
						"url":  "https://monitor.example.com",
					},
				},
			},
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "系统监控",
				"tag":     "plain_text",
			},
			"template": "blue",
		},
	}

	customCard := client.NewMessage().
		Title("自定义卡片").
		Format(notifiers.FormatCard).
		CardData(customCardData).
		FeishuGroup("default").
		Build()

	results, err = hub.Send(ctx, customCard, nil)
	if err != nil {
		log.Printf("发送自定义卡片失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 平台: %s, 成功: %v, 耗时: %v\n",
					result.Platform, result.Success, result.Duration)
			} else {
				fmt.Printf("❌ 平台: %s, 成功: %v, 耗时: %v, 错误: %s\n",
					result.Platform, result.Success, result.Duration, result.Error)
			}
		}
	}

	// 示例7: 使用快捷方法
	fmt.Println("\n=== 使用快捷方法 ===")

	// 快速发送文本
	err = hub.QuickSend(ctx, "快速通知", "这是通过快捷方法发送的消息", "group:default@feishu")
	if err != nil {
		log.Printf("快捷发送失败: %v", err)
	} else {
		fmt.Println("✅ 快捷发送成功")
	}

	// 等待异步消息处理完成
	time.Sleep(2 * time.Second)

	// 检查发送统计
	fmt.Println("\n=== 发送统计 ===")
	metrics := hub.GetMetrics()
	if successRate, ok := metrics["success_rate"].(float64); ok {
		fmt.Printf("成功率: %.1f%%\n", successRate*100)
	}
	if totalSent, ok := metrics["total_sent"].(int64); ok {
		fmt.Printf("总发送数: %d\n", totalSent)
	}
	if platformHealth, ok := metrics["platform_health"].(map[string]bool); ok {
		if feishuHealth, exists := platformHealth["feishu"]; exists {
			fmt.Printf("飞书健康状态: %v\n", feishuHealth)
		}
	}
}

