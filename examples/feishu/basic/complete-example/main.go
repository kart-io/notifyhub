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
	// 从环境变量获取飞书配置
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	secret := os.Getenv("FEISHU_SECRET")

	if webhookURL == "" {
		log.Fatal("请设置环境变量 FEISHU_WEBHOOK_URL")
	}

	fmt.Println("🚀 飞书通知示例")
	fmt.Println("===============")
	fmt.Printf("Webhook URL: %s\n", webhookURL)
	if secret != "" {
		fmt.Println("Secret: 已配置")
	}

	// 创建 Hub 实例，配置飞书
	var options []notifyhub.HubOption

	// 配置飞书平台
	feishuConfig := map[string]interface{}{
		"webhook_url": webhookURL,
	}
	if secret != "" {
		feishuConfig["secret"] = secret
		feishuConfig["auth_mode"] = "signature"
	}

	options = append(options, notifyhub.WithFeishuFromMap(feishuConfig))
	options = append(options, notifyhub.WithTimeout(30*time.Second))

	hub, err := notifyhub.NewHub(options...)
	if err != nil {
		log.Fatalf("创建 Hub 失败: %v", err)
	}
	defer func() {
		if err := hub.Close(context.Background()); err != nil {
			log.Printf("关闭 Hub 失败: %v", err)
		}
	}()

	ctx := context.Background()

	// 示例1: 发送简单文本消息
	fmt.Println("\n📝 示例1: 简单文本消息")
	fmt.Println("------------------")

	textMessage := notifyhub.NewMessage("系统通知").
		WithText("这是一条来自 NotifyHub 的测试消息 📢").
		ToFeishuGroup("oc_test_group").
		Build()

	receipt, err := hub.Send(ctx, textMessage)
	if err != nil {
		log.Printf("发送文本消息失败: %v", err)
	} else {
		fmt.Printf("✅ 消息发送成功 - ID: %s, 状态: %s\n", receipt.MessageID, receipt.Status)
		// 显示详细结果
		if len(receipt.Results) > 0 {
			fmt.Println("  详细结果:")
			for _, result := range receipt.Results {
				status := "✅"
				if !result.Success {
					status = "❌"
				}
				fmt.Printf("    %s Target: %s, Platform: %s, Duration: %dms\n",
					status, result.Target, result.Platform, result.Duration.Milliseconds())
				if result.Error != "" {
					fmt.Printf("      错误: %s\n", result.Error)
				}
			}
		}
	}

	// 示例2: 发送Markdown格式消息
	fmt.Println("\n📋 示例2: Markdown 格式消息")
	fmt.Println("------------------------")

	markdownContent := `## 系统状态报告 📊

**服务状态**: 🟢 正常运行
**响应时间**: 45ms
**活跃用户**: 1,250

### 最近更新
- ✅ 用户认证系统优化
- ✅ 消息推送性能提升
- 🔄 数据库备份进行中

*更新时间: 2024-01-20 14:30*`

	markdownMessage := notifyhub.NewAlert("系统状态更新").
		WithMarkdown(markdownContent).
		ToFeishu("ou_test_user").
		Build()

	receipt, err = hub.Send(ctx, markdownMessage)
	if err != nil {
		log.Printf("发送Markdown消息失败: %v", err)
	} else {
		fmt.Printf("✅ Markdown消息发送成功 - ID: %s\n", receipt.MessageID)
	}

	// 示例3: 发送卡片消息
	fmt.Println("\n🎴 示例3: 飞书卡片消息")
	fmt.Println("--------------------")

	cardContent := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**🎉 部署成功通知**\n\n应用已成功部署到生产环境",
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
							"content": "**版本号**\nv2.1.4",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**部署时间**\n2024-01-20 14:30:00",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**部署环境**\n生产环境 (Production)",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**状态**\n🟢 运行正常",
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
						"url":  "https://console.example.com/deployment/12345",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "查看日志",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://logs.example.com/app/production",
					},
				},
			},
		},
	}

	cardMessage := notifyhub.NewUrgent("部署通知").
		WithFeishuCard(cardContent).
		WithFeishuMentions("ou_admin_user").
		ToFeishuGroup("oc_devops_group").
		Build()

	receipt, err = hub.Send(ctx, cardMessage)
	if err != nil {
		log.Printf("发送卡片消息失败: %v", err)
	} else {
		fmt.Printf("✅ 卡片消息发送成功 - ID: %s\n", receipt.MessageID)
	}

	// 示例4: 批量发送消息
	fmt.Println("\n📤 示例4: 批量发送")
	fmt.Println("----------------")

	batchMessage := notifyhub.NewMessage("批量通知").
		WithText("这是一条批量发送的通知消息").
		WithMetadata("batch_id", "batch_001").
		WithMetadata("source", "automated_system").
		ToFeishu("ou_user1").
		ToFeishu("ou_user2").
		ToFeishuGroup("oc_group1").
		ToFeishuGroup("oc_group2").
		Build()

	receipt, err = hub.Send(ctx, batchMessage)
	if err != nil {
		log.Printf("批量发送失败: %v", err)
	} else {
		fmt.Printf("✅ 批量发送完成 - 总计: %d, 成功: %d, 失败: %d\n",
			receipt.Total, receipt.Successful, receipt.Failed)

		// 显示详细结果
		for _, result := range receipt.Results {
			status := "✅"
			if !result.Success {
				status = "❌"
			}
			fmt.Printf("  %s Target: %s, Platform: %s, Duration: %dms\n",
				status, result.Target, result.Platform, result.Duration.Milliseconds())
		}
	}

	// 示例5: 异步发送
	fmt.Println("\n⚡ 示例5: 异步发送")
	fmt.Println("----------------")

	asyncMessage := notifyhub.NewMessage("异步通知").
		WithText("这是一条异步发送的消息，不会阻塞主流程").
		WithVariable("async_id", "async_001").
		ToFeishuGroup("oc_async_group").
		Build()

	asyncReceipt, err := hub.SendAsync(ctx, asyncMessage)
	if err != nil {
		log.Printf("异步发送失败: %v", err)
	} else {
		fmt.Printf("✅ 异步发送已提交 - ID: %s, 状态: %s\n",
			asyncReceipt.MessageID, asyncReceipt.Status)
		fmt.Printf("  队列时间: %s\n", asyncReceipt.QueuedAt.Format("15:04:05"))
	}

	// 示例6: 健康检查
	fmt.Println("\n🔍 示例6: 系统健康检查")
	fmt.Println("--------------------")

	health, err := hub.Health(ctx)
	if err != nil {
		log.Printf("健康检查失败: %v", err)
	} else {
		fmt.Printf("✅ 系统状态: %s\n", health.Status)
		fmt.Printf("  检查时间: %s\n", health.Timestamp.Format("2006-01-02 15:04:05"))

		for platform, status := range health.Platforms {
			healthStatus := "🟢 健康"
			if !status.Available {
				healthStatus = fmt.Sprintf("🔴 异常: %s", status.Status)
			}
			fmt.Printf("  %s: %s\n", platform, healthStatus)
		}
	}

	fmt.Println("\n✨ 飞书通知示例运行完成!")
	fmt.Println("========================")
}
