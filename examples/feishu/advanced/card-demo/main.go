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