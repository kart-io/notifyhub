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

// 独立的卡片功能测试
func main() {
	fmt.Println("=== 飞书卡片功能专项测试 ===")

	// 创建 NotifyHub 实例
	hub, err := client.New(
		config.WithFeishuFromEnv(),
		config.WithQueue("memory", 100, 2),
	)
	if err != nil {
		log.Fatalf("创建 NotifyHub 失败: %v", err)
	}

	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
	defer hub.Stop()

	// 测试1: 简单卡片（默认模板）
	testSimpleCard(hub, ctx)

	// 等待2秒避免API限制
	time.Sleep(2 * time.Second)

	// 测试2: 完全自定义卡片
	testCustomCard(hub, ctx)

	// 等待2秒
	time.Sleep(2 * time.Second)

	// 测试3: 业务场景卡片
	testBusinessCard(hub, ctx)

	// 等待2秒
	time.Sleep(2 * time.Second)

	// 测试4: @人功能
	testAtMentionFeatures(hub, ctx)

	// 等待异步消息处理完成
	fmt.Println("\n⏳ 等待异步消息处理完成...")
	time.Sleep(3 * time.Second)

	// 显示统计信息
	fmt.Println("\n=== 卡片测试统计 ===")
	metrics := hub.GetMetrics()
	if successRate, ok := metrics["success_rate"].(float64); ok {
		fmt.Printf("成功率: %.1f%%\n", successRate*100)
	}
	if totalSent, ok := metrics["total_sent"].(int64); ok {
		fmt.Printf("总发送数: %d\n", totalSent)
	}
}

// 测试简单卡片（使用默认模板）
func testSimpleCard(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n--- 测试1: 简单卡片（默认模板） ---")

	message := client.NewCard("📊 系统状态报告", "服务器运行状态良好").
		Metadata("服务器", "web-server-01").
		Metadata("CPU", "45%").
		Metadata("内存", "68%").
		Metadata("状态", "🟢 正常").
		Priority(3).
		FeishuGroup("default").
		Build()

	results, err := hub.Send(ctx, message, nil)
	if err != nil {
		log.Printf("❌ 发送简单卡片失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 简单卡片发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ 简单卡片发送失败: %s\n", result.Error)
			}
		}
	}
}

// 测试完全自定义卡片
func testCustomCard(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n--- 测试2: 完全自定义卡片 ---")

	customCardData := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**🚀 部署成功通知**",
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
							"content": "**应用名称**\nuser-service",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**版本**\nv2.1.4",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**环境**\n生产环境",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**部署时间**\n" + time.Now().Format("2006-01-02 15:04:05"),
							"tag":     "lark_md",
						},
					},
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**更新内容**\n• 修复用户登录问题\n• 优化性能\n• 安全漏洞修复",
					"tag":     "lark_md",
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
						"url":  "https://deploy.example.com/details",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "回滚",
							"tag":     "plain_text",
						},
						"type": "danger",
						"url":  "https://deploy.example.com/rollback",
					},
				},
			},
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "部署通知",
				"tag":     "plain_text",
			},
			"template": "green",
		},
	}

	message := client.NewMessage().
		Title("部署成功").
		Format(notifiers.FormatCard).
		CardData(customCardData).
		Priority(4).
		FeishuGroup("default").
		Build()

	results, err := hub.Send(ctx, message, nil)
	if err != nil {
		log.Printf("❌ 发送自定义卡片失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 自定义卡片发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ 自定义卡片发送失败: %s\n", result.Error)
			}
		}
	}
}

// 测试业务场景卡片
func testBusinessCard(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n--- 测试3: 业务场景卡片 ---")

	businessCardData := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**📈 周报 - 开发团队**",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "📅 **报告期间**: " + getWeekRange() + "\n👥 **团队**: 后端开发组",
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
							"content": "**已完成任务**\n✅ 15个",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**进行中任务**\n🔄 8个",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**Bug修复**\n🐛 12个",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**代码提交**\n💻 89次",
							"tag":     "lark_md",
						},
					},
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**重点成果**\n• 🎯 用户登录模块重构完成\n• 🔐 安全认证系统升级\n• ⚡ API响应时间优化30%\n• 📊 新增监控仪表板",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**下周计划**\n• 🚀 开始支付模块开发\n• 🧪 集成测试环境搭建\n• 📝 技术文档更新",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "action",
				"actions": []map[string]interface{}{
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "查看详细报告",
							"tag":     "plain_text",
						},
						"type": "primary",
						"url":  "https://project.example.com/weekly-report",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "任务看板",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://project.example.com/kanban",
					},
				},
			},
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "开发团队周报",
				"tag":     "plain_text",
			},
			"template": "blue",
		},
	}

	message := client.NewCard("团队周报", "开发进度汇总").
		CardData(businessCardData).
		Priority(3).
		FeishuGroup("default").
		Metadata("report_type", "weekly").
		Metadata("team", "backend").
		Build()

	// 使用异步发送
	messageID, err := hub.SendAsync(ctx, message, nil)
	if err != nil {
		log.Printf("❌ 发送业务卡片失败: %v", err)
	} else {
		fmt.Printf("✅ 业务卡片已排队发送，消息ID: %s\n", messageID)
	}
}

// 获取本周时间范围
func getWeekRange() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // 将周日调整为7
	}

	monday := now.AddDate(0, 0, -(weekday-1))
	sunday := monday.AddDate(0, 0, 6)

	return fmt.Sprintf("%s - %s",
		monday.Format("01-02"),
		sunday.Format("01-02"))
}

// 测试@人功能
func testAtMentionFeatures(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n--- 测试4: @人功能演示 ---")

	// 测试1: 文本消息@单个用户
	fmt.Println("\n🔸 测试@单个用户（文本消息）")
	textMessage := client.NewMessage().
		Title("📢 重要通知").
		Body("系统将于今晚22:00进行维护，预计耗时30分钟。").
		AtUser("ou_123456789", "张三").
		Priority(4).
		FeishuGroup("default").
		Build()

	results, err := hub.Send(ctx, textMessage, nil)
	if err != nil {
		log.Printf("❌ 发送@人文本消息失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ @人文本消息发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ @人文本消息发送失败: %s\n", result.Error)
			}
		}
	}

	time.Sleep(2 * time.Second)

	// 测试2: 富文本消息@多个用户
	fmt.Println("\n🔸 测试@多个用户（富文本消息）")
	richTextMessage := client.NewMessage().
		Title("🚨 紧急事件").
		Body("生产环境出现异常，请相关人员立即处理！").
		Format(notifiers.FormatMarkdown).
		AtUsers("ou_111111111", "ou_222222222", "ou_333333333").
		Priority(5).
		FeishuGroup("default").
		Build()

	results, err = hub.Send(ctx, richTextMessage, nil)
	if err != nil {
		log.Printf("❌ 发送@多人富文本消息失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ @多人富文本消息发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ @多人富文本消息发送失败: %s\n", result.Error)
			}
		}
	}

	time.Sleep(2 * time.Second)

	// 测试3: 卡片消息@所有人
	fmt.Println("\n🔸 测试@所有人（卡片消息）")
	cardMessage := client.NewCard("🎉 发版通知", "新版本v2.1.0已成功发布！").
		Metadata("版本", "v2.1.0").
		Metadata("发布时间", time.Now().Format("2006-01-02 15:04:05")).
		Metadata("更新内容", "新增@人功能、修复已知问题").
		AtAll().
		Priority(3).
		FeishuGroup("default").
		Build()

	results, err = hub.Send(ctx, cardMessage, nil)
	if err != nil {
		log.Printf("❌ 发送@所有人卡片消息失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ @所有人卡片消息发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ @所有人卡片消息发送失败: %s\n", result.Error)
			}
		}
	}

	time.Sleep(2 * time.Second)

	// 测试4: 自定义卡片消息@指定用户
	fmt.Println("\n🔸 测试自定义卡片@指定用户")
	customCardData := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**🔔 任务分配通知**",
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
							"content": "**任务标题**\\n优化系统性能",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**截止时间**\\n" + time.Now().AddDate(0, 0, 7).Format("2006-01-02"),
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**优先级**\\n🔴 高",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**负责人**\\n@李四",
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
							"content": "接受任务",
							"tag":     "plain_text",
						},
						"type": "primary",
						"url":  "https://task.example.com/accept",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "查看详情",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://task.example.com/details",
					},
				},
			},
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "任务分配",
				"tag":     "plain_text",
			},
			"template": "orange",
		},
	}

	customCardMessage := client.NewMessage().
		Title("任务分配").
		Format(notifiers.FormatCard).
		CardData(customCardData).
		AtUser("ou_444444444", "李四").
		Priority(4).
		FeishuGroup("default").
		Build()

	results, err = hub.Send(ctx, customCardMessage, nil)
	if err != nil {
		log.Printf("❌ 发送自定义卡片@人消息失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 自定义卡片@人消息发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ 自定义卡片@人消息发送失败: %s\n", result.Error)
			}
		}
	}

	fmt.Println("\n📋 @人功能测试说明：")
	fmt.Println("1. 文本消息支持@单个用户")
	fmt.Println("2. 富文本消息支持@多个用户")
	fmt.Println("3. 卡片消息支持@所有人")
	fmt.Println("4. 自定义卡片支持@指定用户")
	fmt.Println("5. 用户ID需要替换为真实的飞书用户ID")
}