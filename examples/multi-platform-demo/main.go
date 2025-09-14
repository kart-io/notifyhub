package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub"
)

func main() {
	log.Println("🚀 NotifyHub 多平台通知演示")
	log.Println("=======================================")

	// 配置多个通知平台（使用测试端点）
	hub, err := notifyhub.New(
		// 飞书配置 - 使用测试端点
		notifyhub.WithFeishu(
			"https://httpbin.org/post", // 测试端点，会返回请求信息
			"",
		),
		// 如果有真实的邮件配置，取消注释：
		/*
		notifyhub.WithEmail(
			"smtp.gmail.com",
			587,
			"your@gmail.com",
			"your-app-password",
			"your@gmail.com",
		),
		*/
		// 队列配置
		notifyhub.WithQueue("memory", 500, 2),
		// 设置路由规则
		notifyhub.WithRouting(
			// 高优先级消息发送到所有平台
			notifyhub.NewRoutingRule("high_priority_all").
				WithPriority(4, 5).
				RouteTo("feishu", "email").
				Build(),
			// 告警消息发送到飞书
			notifyhub.NewRoutingRule("alerts_feishu").
				WithMessageType("alert").
				RouteTo("feishu").
				Build(),
		),
	)
	if err != nil {
		log.Fatalf("❌ 创建Hub失败: %v", err)
	}

	// 启动服务
	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("❌ 启动Hub失败: %v", err)
	}
	defer hub.Stop()

	log.Printf("✅ NotifyHub 启动成功，配置的通知平台数量: %d", len(getConfiguredPlatforms(hub)))

	// ========================================
	// 演示1：基础多目标发送
	// ========================================
	log.Println("\n📤 演示1: 基础多目标发送")
	log.Println("-----------------------------------")

	message1 := notifyhub.NewAlert("服务器告警", "CPU使用率超过90%").
		Variable("server", "web-01").
		Variable("cpu_usage", "95%").
		Variable("timestamp", time.Now().Format("2006-01-02 15:04:05")).
		// 同时指定多个目标
		FeishuGroup("ops-team").
		FeishuUser("admin").
		Email("ops@company.com").
		Priority(4). // 触发高优先级路由规则
		Build()

	results1, err := hub.Send(ctx, message1, nil)
	if err != nil {
		log.Printf("❌ 发送失败: %v", err)
	} else {
		log.Printf("✅ 消息发送成功！")
		printResults(results1)
	}

	// ========================================
	// 演示2：不同格式同时发送
	// ========================================
	log.Println("\n📝 演示2: 不同格式同时发送")
	log.Println("-----------------------------------")

	// Markdown格式的消息
	message2 := notifyhub.NewMarkdown("项目更新", `
## 🎉 新功能发布

### ✨ 主要更新
- **多平台通知**: 支持同时发送到飞书和邮件
- **智能路由**: 基于优先级和类型自动路由
- **模板系统**: 支持变量替换和多种格式

### 📊 性能提升
| 指标 | 提升幅度 |
|------|----------|
| 发送速度 | +50% |
| 成功率 | 99.9% |

发布时间: {{.timestamp}}
环境: {{.environment}}
`).
		Variable("timestamp", time.Now().Format("2006-01-02 15:04:05")).
		Variable("environment", "production").
		FeishuGroup("dev-team").
		Email("dev@company.com").
		Priority(3).
		Build()

	results2, err := hub.Send(ctx, message2, nil)
	if err != nil {
		log.Printf("❌ 发送失败: %v", err)
	} else {
		log.Printf("✅ Markdown消息发送成功！")
		printResults(results2)
	}

	// ========================================
	// 演示3：使用内置模板批量发送
	// ========================================
	log.Println("\n📋 演示3: 使用内置模板批量发送")
	log.Println("-----------------------------------")

	message3 := notifyhub.NewReport("每日运营报告", "").
		Template("report"). // 使用内置报告模板
		Variable("metrics", map[string]interface{}{
			"total_users":    "10,240",
			"active_users":   "8,456",
			"new_signups":    "156",
			"revenue":        "$45,230",
			"success_rate":   "99.2%",
		}).
		Variable("summary", "今日系统运行稳定，用户活跃度较高").
		FeishuGroup("management").
		FeishuGroup("ops-team").
		Email("reports@company.com").
		Email("management@company.com").
		Priority(2).
		Build()

	results3, err := hub.Send(ctx, message3, nil)
	if err != nil {
		log.Printf("❌ 发送失败: %v", err)
	} else {
		log.Printf("✅ 报告发送成功！")
		printResults(results3)
	}

	// ========================================
	// 演示4：异步批量通知
	// ========================================
	log.Println("\n⏳ 演示4: 异步批量通知")
	log.Println("-----------------------------------")

	// 创建多个异步任务
	asyncMessages := []struct {
		name string
		msg  *notifyhub.Message
	}{
		{
			name: "用户通知",
			msg: notifyhub.NewNotice("系统维护通知", "系统将于今晚23:00-01:00进行维护").
				Variable("maintenance_time", "23:00-01:00").
				Variable("expected_downtime", "2小时").
				FeishuGroup("all-users").
				Email("users@company.com").
				Build(),
		},
		{
			name: "开发团队通知",
			msg: notifyhub.NewNotice("部署完成", "生产环境部署已完成").
				Variable("version", "v1.2.3").
				Variable("deploy_time", time.Now().Format("15:04:05")).
				FeishuGroup("dev-team").
				Email("dev@company.com").
				Build(),
		},
	}

	for _, asyncMsg := range asyncMessages {
		taskID, err := hub.SendAsync(ctx, asyncMsg.msg, notifyhub.NewAsyncOptions())
		if err != nil {
			log.Printf("❌ %s异步发送失败: %v", asyncMsg.name, err)
		} else {
			log.Printf("✅ %s已加入队列: %s", asyncMsg.name, taskID)
		}
	}

	// 等待异步任务处理
	log.Println("⏳ 等待异步任务处理...")
	time.Sleep(3 * time.Second)

	// ========================================
	// 系统统计
	// ========================================
	log.Println("\n📊 系统统计信息")
	log.Println("=======================================")

	metrics := hub.GetMetrics()
	if totalSent, ok := metrics["total_sent"]; ok {
		log.Printf("📈 总发送数量: %v", totalSent)
	}
	if successRate, ok := metrics["success_rate"]; ok {
		log.Printf("📊 成功率: %.1f%%", successRate.(float64)*100)
	}

	health := hub.GetHealth(ctx)
	status := health["status"].(string)
	statusIcon := "✅"
	if status != "healthy" {
		statusIcon = "⚠️"
	}
	log.Printf("%s 系统健康状态: %s", statusIcon, status)

	log.Println("\n🎉 多平台通知演示完成！")
	log.Println("=======================================")
}

// printResults 打印发送结果
func printResults(results []*notifyhub.SendResult) {
	log.Printf("📊 发送结果统计: 共 %d 条记录", len(results))

	successCount := 0
	for i, result := range results {
		status := "❌ 失败"
		if result.Success {
			status = "✅ 成功"
			successCount++
		}
		log.Printf("  [%d] 平台: %s | 状态: %s | 耗时: %v",
			i+1, result.Platform, status, result.Duration)

		if !result.Success && result.Error != "" {
			log.Printf("      错误信息: %s", result.Error)
		}
	}

	log.Printf("📈 成功率: %.1f%% (%d/%d)",
		float64(successCount)/float64(len(results))*100, successCount, len(results))
}

// getConfiguredPlatforms 获取配置的平台数量（简化实现）
func getConfiguredPlatforms(hub *notifyhub.Hub) []string {
	// 这是一个简化的实现，实际应该从hub内部获取
	return []string{"feishu"} // 在这个演示中只有飞书可以工作
}