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
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	secret := os.Getenv("FEISHU_SECRET")

	if webhookURL == "" {
		log.Fatal("请设置环境变量 FEISHU_WEBHOOK_URL")
	}

	fmt.Println("🚀 飞书高级特性演示")
	fmt.Println("==================")

	// 创建Hub实例
	feishuConfig := map[string]interface{}{
		"webhook_url": webhookURL,
	}
	if secret != "" {
		feishuConfig["secret"] = secret
		feishuConfig["auth_mode"] = "signature"
	}

	hub, err := notifyhub.NewHub(
		notifyhub.WithFeishuFromMap(feishuConfig),
		notifyhub.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatalf("创建Hub失败: %v", err)
	}
	defer hub.Close(context.Background())

	ctx := context.Background()

	// 示例1: 消息模板和变量替换
	fmt.Println("\n📋 示例1: 消息模板和变量")
	fmt.Println("---------------------")
	runTemplateExample(ctx, hub)

	// 示例2: 飞书卡片消息
	fmt.Println("\n🎴 示例2: 飞书卡片消息")
	fmt.Println("---------------------")
	runCardExample(ctx, hub)

	// 示例3: @提及用户
	fmt.Println("\n👥 示例3: @提及用户")
	fmt.Println("------------------")
	runMentionExample(ctx, hub)

	// 示例4: 批量发送和错误处理
	fmt.Println("\n📤 示例4: 批量发送")
	fmt.Println("----------------")
	runBatchExample(ctx, hub)

	// 示例5: 异步发送
	fmt.Println("\n⚡ 示例5: 异步发送")
	fmt.Println("----------------")
	runAsyncExample(ctx, hub)

	// 示例6: 健康检查
	fmt.Println("\n🔍 示例6: 系统健康检查")
	fmt.Println("--------------------")
	runHealthCheckExample(ctx, hub)

	// 示例7: 消息优先级
	fmt.Println("\n⚠️ 示例7: 消息优先级")
	fmt.Println("------------------")
	runPriorityExample(ctx, hub)

	fmt.Println("\n✨ 高级特性演示完成!")
}

// runTemplateExample 演示消息模板和变量
func runTemplateExample(ctx context.Context, hub notifyhub.Hub) {
	message := notifyhub.NewMessage("服务状态报告").
		WithText("服务 {{service_name}} 在 {{environment}} 环境的状态: {{status}}").
		WithVariable("service_name", "用户服务").
		WithVariable("environment", "生产环境").
		WithVariable("status", "🟢 正常").
		WithMetadata("report_type", "status").
		WithMetadata("timestamp", time.Now().Format("2006-01-02 15:04:05")).
		ToFeishuGroup("oc_ops_group").
		Build()

	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		fmt.Printf("✅ 模板消息发送成功 - ID: %s\n", receipt.MessageID)
		printResults(receipt)
	}
}

// runCardExample 演示飞书卡片消息
func runCardExample(ctx context.Context, hub notifyhub.Hub) {
	// 创建复杂的飞书卡片
	cardContent := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "🎯 项目里程碑达成",
				"tag":     "plain_text",
			},
			"template": "blue",
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "恭喜！我们的项目又达成了一个重要里程碑 🎉",
					"tag":     "plain_text",
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
							"content": "**项目名称**\nNotifyHub v2.0",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**完成度**\n85%",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**预计完成**\n2024-02-15",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**团队成员**\n12 人",
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
							"content": "查看项目详情",
							"tag":     "plain_text",
						},
						"type": "primary",
						"url":  "https://project.example.com/notifyhub",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "查看任务看板",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://kanban.example.com/notifyhub",
					},
				},
			},
		},
	}

	message := notifyhub.NewMessage("项目进度更新").
		WithFeishuCard(cardContent).
		ToFeishuGroup("oc_project_group").
		Build()

	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		fmt.Printf("✅ 卡片消息发送成功 - ID: %s\n", receipt.MessageID)
		printResults(receipt)
	}
}

// runMentionExample 演示@提及功能
func runMentionExample(ctx context.Context, hub notifyhub.Hub) {
	// 提及特定用户
	message1 := notifyhub.NewAlert("紧急修复需求").
		WithText("发现严重bug，需要立即处理！").
		WithFeishuMentions("ou_lead_dev", "ou_qa_lead").
		ToFeishuGroup("oc_dev_group").
		Build()

	receipt, err := hub.Send(ctx, message1)
	if err != nil {
		log.Printf("发送提及消息失败: %v", err)
	} else {
		fmt.Printf("✅ 提及消息发送成功 - ID: %s\n", receipt.MessageID)
	}

	// @所有人
	message2 := notifyhub.NewUrgent("系统维护通知").
		WithText("系统将在今晚22:00进行维护，预计持续2小时").
		WithFeishuMentionAll().
		ToFeishuGroup("oc_all_group").
		Build()

	receipt, err = hub.Send(ctx, message2)
	if err != nil {
		log.Printf("发送@所有人消息失败: %v", err)
	} else {
		fmt.Printf("✅ @所有人消息发送成功 - ID: %s\n", receipt.MessageID)
	}
}

// runBatchExample 演示批量发送
func runBatchExample(ctx context.Context, hub notifyhub.Hub) {
	message := notifyhub.NewMessage("多目标通知").
		WithText("这是一条发送到多个目标的通知消息").
		WithMetadata("batch_id", "batch_001").
		WithMetadata("sender", "system").
		ToFeishuUser("ou_user1").
		ToFeishuUser("ou_user2").
		ToFeishuGroup("oc_group1").
		ToFeishuGroup("oc_group2").
		Build()

	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Printf("批量发送失败: %v", err)
	} else {
		fmt.Printf("✅ 批量发送完成 - 总计: %d, 成功: %d, 失败: %d\n",
			receipt.Total, receipt.Successful, receipt.Failed)
		printResults(receipt)
	}
}

// runAsyncExample 演示异步发送
func runAsyncExample(ctx context.Context, hub notifyhub.Hub) {
	message := notifyhub.NewMessage("异步通知").
		WithText("这是一条异步发送的消息，不会阻塞主流程").
		WithMetadata("async", "true").
		ToFeishuGroup("oc_async_group").
		Build()

	asyncReceipt, err := hub.SendAsync(ctx, message)
	if err != nil {
		log.Printf("异步发送失败: %v", err)
	} else {
		fmt.Printf("✅ 异步发送已提交 - ID: %s, 状态: %s\n",
			asyncReceipt.MessageID, asyncReceipt.Status)
		fmt.Printf("  队列时间: %s\n", asyncReceipt.QueuedAt.Format("15:04:05"))
	}
}

// runHealthCheckExample 演示健康检查
func runHealthCheckExample(ctx context.Context, hub notifyhub.Hub) {
	health, err := hub.Health(ctx)
	if err != nil {
		log.Printf("健康检查失败: %v", err)
		return
	}

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

// runPriorityExample 演示消息优先级
func runPriorityExample(ctx context.Context, hub notifyhub.Hub) {
	// 低优先级消息
	lowPriorityMsg := notifyhub.NewMessage("日常通知").
		WithText("这是一条低优先级的日常通知").
		WithPriority(notifyhub.PriorityLow).
		ToFeishuGroup("oc_daily_group").
		Build()

	// 高优先级警告
	highPriorityMsg := notifyhub.NewAlert("性能警告").
		WithText("系统响应时间超过阈值").
		WithPriority(notifyhub.PriorityHigh).
		ToFeishuGroup("oc_ops_group").
		Build()

	// 紧急消息
	urgentMsg := notifyhub.NewUrgent("服务异常").
		WithText("支付服务出现故障，需要立即处理！").
		WithPriority(notifyhub.PriorityUrgent).
		WithFeishuMentionAll().
		ToFeishuGroup("oc_emergency_group").
		Build()

	// 发送消息
	messages := []*notifyhub.Message{lowPriorityMsg, highPriorityMsg, urgentMsg}
	priorities := []string{"低优先级", "高优先级", "紧急"}

	for i, msg := range messages {
		receipt, err := hub.Send(ctx, msg)
		if err != nil {
			log.Printf("发送%s消息失败: %v", priorities[i], err)
		} else {
			fmt.Printf("✅ %s消息发送成功 - ID: %s\n", priorities[i], receipt.MessageID)
		}
	}
}

// printResults 打印发送结果详情
func printResults(receipt *notifyhub.Receipt) {
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