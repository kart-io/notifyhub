package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
	"github.com/kart-io/notifyhub/queue"
)

// 飞书高级功能示例
func main() {
	// 创建高级配置的 NotifyHub
	hub, err := client.New(
		config.WithFeishuFromEnv(),
		config.WithQueue("memory", 2000, 8), // 更大的队列和更多工作协程
		config.WithTelemetry("feishu-example", "1.0.0", "production", ""),
	)
	if err != nil {
		log.Fatalf("创建 NotifyHub 失败: %v", err)
	}

	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
	defer hub.Stop()

	// 示例1: 使用模板发送复杂消息
	demonstrateTemplates(hub, ctx)

	// 示例2: 消息路由和优先级
	demonstrateRouting(hub, ctx)

	// 示例3: 重试和错误处理
	demonstrateRetryHandling(hub, ctx)

	// 示例4: 回调和监控
	demonstrateCallbacks(hub, ctx)

	// 示例5: 延迟发送
	demonstrateDelayedSending(hub, ctx)

	// 示例6: 批量增强发送
	demonstrateEnhancedBatch(hub, ctx)

	// 示例7: 高级卡片功能
	demonstrateAdvancedCards(hub, ctx)

	// 等待所有异步任务完成
	time.Sleep(5 * time.Second)

	// 显示最终统计
	showFinalStats(hub)
}

// 演示模板使用
func demonstrateTemplates(hub *client.Hub, ctx context.Context) {
	fmt.Println("=== 模板消息示例 ===")

	// 添加自定义模板
	templates := hub.Templates()
	metadata := &client.TemplateMetadata{
		Name:        "incident_alert",
		Description: "紧急事件通知模板",
		Version:     "1.0.0",
		Author:      "NotifyHub Example",
		Tags:        []string{"incident", "alert", "emergency"},
		Variables: []client.VariableInfo{
			{Name: "severity", Type: "string", Required: true, Description: "事件级别"},
			{Name: "service", Type: "string", Required: true, Description: "影响服务"},
			{Name: "start_time", Type: "string", Required: true, Description: "开始时间"},
			{Name: "description", Type: "string", Required: true, Description: "事件描述"},
			{Name: "assignee", Type: "string", Required: false, Description: "负责人"},
			{Name: "eta", Type: "string", Required: false, Description: "预计恢复时间"},
		},
		Platforms:    []string{"feishu"},
		MessageTypes: []string{"alert"},
	}

	err := templates.RegisterTemplate("incident_alert", `🚨 **紧急事件通知**

**事件级别:** {{.severity}}
**影响服务:** {{.service}}
**开始时间:** {{.start_time}}
**描述:** {{.description}}

**负责人:** {{.assignee}}
**预计恢复:** {{.eta}}

---
请相关人员立即处理。`, metadata)

	if err != nil {
		log.Printf("添加模板失败: %v", err)
		return
	}

	// 使用模板发送事件通知
	templateVars := map[string]interface{}{
		"severity":    "critical",
		"service":     "用户认证服务",
		"start_time":  time.Now().Add(-10 * time.Minute).Format("2006-01-02 15:04:05"),
		"description": "用户登录接口响应超时，影响用户正常登录",
		"assignee":    "运维团队",
		"eta":         "30分钟内",
	}

	// 渲染模板获取内容
	content, err := hub.RenderTemplate(ctx, "incident_alert", templateVars)
	if err != nil {
		log.Printf("渲染模板失败: %v", err)
		return
	}

	// 解析渲染后的内容
	lines := strings.Split(content, "\n")
	title := "🚨 紧急事件通知"
	body := content
	if len(lines) > 0 && strings.HasPrefix(lines[0], "TITLE:") {
		title = strings.TrimPrefix(lines[0], "TITLE:")
		title = strings.TrimSpace(title)
		if len(lines) > 1 {
			body = strings.Join(lines[1:], "\n")
		}
	}

	incident := client.NewAlert(title, body).
		FeishuGroup("incident-response").
		Priority(5). // 5=urgent
		Build()

	results, err := hub.Send(ctx, incident, &client.Options{
		Retry:      true,
		MaxRetries: 5,
		Timeout:    45 * time.Second,
	})

	if err != nil {
		log.Printf("发送事件通知失败: %v", err)
	} else {
		fmt.Printf("✅ 事件通知发送成功，耗时: %v\n", results[0].Duration)
	}
}

// 演示消息路由和优先级
func demonstrateRouting(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 消息路由示例 ===")

	// 不同优先级的消息会路由到不同的群组
	messages := []struct {
		priority int
		title    string
		group    string
	}{
		{1, "日常维护通知", "general-notifications"}, // 1=low
		{3, "部署完成通知", "deployment-updates"},    // 3=normal
		{4, "性能警告", "performance-alerts"},      // 4=high
		{5, "服务中断", "incident-response"},       // 5=urgent
	}

	for _, msg := range messages {
		message := client.NewMessage().
			Title(msg.title).
			Body(fmt.Sprintf("这是一个 %s 级别的消息示例", getPriorityName(msg.priority))).
			Priority(msg.priority).
			FeishuGroup(msg.group).
			Metadata("routing_demo", "true").
			Build()

		// 异步发送以演示并发处理
		messageID, err := hub.SendAsync(ctx, message, nil)
		if err != nil {
			log.Printf("路由消息失败 (%s): %v", msg.title, err)
		} else {
			fmt.Printf("✅ %s 消息已路由到 %s (ID: %s)\n",
				getPriorityName(msg.priority), msg.group, messageID[:8])
		}
	}
}

// 演示重试和错误处理
func demonstrateRetryHandling(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 重试机制示例 ===")

	// 创建一个可能失败的消息（使用无效的群组ID）
	unreliableMessage := client.NewMessage().
		Title("重试测试").
		Body("这条消息可能需要重试").
		FeishuGroup("invalid-group-id"). // 故意使用无效ID
		Priority(3).                     // 3=normal
		Build()

	// 配置重试选项
	retryOptions := &client.Options{
		Retry:      true,
		MaxRetries: 3,
		Timeout:    10 * time.Second,
	}

	start := time.Now()
	results, err := hub.Send(ctx, unreliableMessage, retryOptions)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ 消息发送失败（预期行为），总耗时: %v\n", duration)
		fmt.Printf("错误详情: %v\n", err)
	} else {
		for _, result := range results {
			fmt.Printf("结果: 平台=%s, 成功=%v, 错误=%s\n",
				result.Platform, result.Success, result.Error)
		}
	}
}

// 演示回调和监控
func demonstrateCallbacks(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 回调监控示例 ===")

	// 创建带回调的消息
	callbackMessage := client.NewNotice("回调测试", "这条消息将触发回调函数").
		FeishuGroup("callback-test").
		Metadata("callback_demo", "true").
		Build()

	// 配置回调选项
	callbackOptions := &queue.CallbackOptions{
		WebhookURL:      "https://your-webhook-endpoint.com/callback",
		WebhookSecret:   "your-webhook-secret",
		CallbackTimeout: 5 * time.Second,
	}

	// 添加成功回调
	successCallback := queue.NewCallbackFunc("success-logger", func(ctx context.Context, callbackCtx *queue.CallbackContext) error {
		fmt.Printf("🎉 回调触发 - 事件: %s, 消息ID: %s, 耗时: %v\n",
			callbackCtx.Event, callbackCtx.MessageID, callbackCtx.Duration)

		// 可以在这里记录到外部系统
		logData := map[string]interface{}{
			"event":      callbackCtx.Event,
			"message_id": callbackCtx.MessageID,
			"platform":   callbackCtx.Results[0].Platform,
			"success":    callbackCtx.Results[0].Success,
			"timestamp":  time.Now(),
		}

		jsonData, _ := json.MarshalIndent(logData, "", "  ")
		fmt.Printf("回调数据: %s\n", jsonData)

		return nil
	})

	callbackOptions.AddCallback(queue.CallbackEventSent, successCallback)

	// 发送消息
	messageID, err := hub.SendAsync(ctx, callbackMessage, &client.Options{
		Callbacks: callbackOptions,
	})

	if err != nil {
		log.Printf("回调消息发送失败: %v", err)
	} else {
		fmt.Printf("✅ 回调消息已排队，ID: %s\n", messageID)
	}
}

// 演示延迟发送
func demonstrateDelayedSending(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 延迟发送示例 ===")

	delays := []time.Duration{
		2 * time.Second,  // 2秒后
		5 * time.Second,  // 5秒后
		10 * time.Second, // 10秒后
	}

	for i, delay := range delays {
		delayedMessage := client.NewMessage().
			Title(fmt.Sprintf("延迟消息 #%d", i+1)).
			Body(fmt.Sprintf("这条消息延迟了 %v 后发送", delay)).
			Delay(delay).
			FeishuGroup("delayed-messages").
			Metadata("delay_seconds", fmt.Sprintf("%.0f", delay.Seconds())).
			Build()

		messageID, err := hub.SendAsync(ctx, delayedMessage, nil)
		if err != nil {
			log.Printf("延迟消息排队失败: %v", err)
		} else {
			fmt.Printf("⏰ 延迟消息已排队 (延迟%v): %s\n", delay, messageID[:8])
		}
	}
}

// 演示批量增强发送
func demonstrateEnhancedBatch(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 批量增强发送示例 ===")

	// 使用增强批量构建器
	batch := hub.NewEnhancedBatch()

	// 添加多种类型的消息到批次
	messages := []struct {
		msgType string
		title   string
		content string
		target  string
	}{
		{"alert", "数据库连接警告", "数据库连接池使用率达到80%", "dba-team"},
		{"notice", "部署通知", "应用版本 v2.1.0 已部署到生产环境", "dev-team"},
		{"report", "性能报告", "今日系统性能表现良好", "management"},
	}

	for _, msg := range messages {
		var message *notifiers.Message

		switch msg.msgType {
		case "alert":
			message = client.NewAlert(msg.title, msg.content).
				Variable("threshold", "80%").
				Variable("current", "82%").
				FeishuGroup(msg.target).
				Priority(4). // 4=high
				Build()
		case "notice":
			message = client.NewNotice(msg.title, msg.content).
				Variable("version", "v2.1.0").
				Variable("environment", "production").
				FeishuGroup(msg.target).
				Build()
		case "report":
			message = client.NewReport(msg.title, msg.content).
				Variable("date", time.Now().Format("2006-01-02")).
				Variable("uptime", "99.9%").
				FeishuGroup(msg.target).
				Build()
		}

		target := notifiers.Target{Type: notifiers.TargetTypeGroup, Value: msg.target, Platform: "feishu"}
		batch.AddMessage(message, []notifiers.Target{target}, &client.Options{
			Retry:      true,
			MaxRetries: 3,
		})

		fmt.Printf("📝 添加消息到批次: %s (类型: %s)\n", msg.title, msg.msgType)
	}

	// 执行批量发送
	fmt.Println("🚀 开始批量发送...")
	start := time.Now()

	batchResults, err := batch.SendAll(ctx)
	duration := time.Since(start)

	if err != nil {
		log.Printf("批量发送失败: %v", err)
	} else {
		fmt.Printf("✅ 批量发送完成，总耗时: %v\n", duration)

		successCount := 0
		for _, result := range batchResults {
			if result.Success {
				successCount++
			}
		}

		fmt.Printf("📊 发送统计: 成功 %d/%d, 成功率: %.1f%%\n",
			successCount, len(batchResults),
			float64(successCount)/float64(len(batchResults))*100)
	}
}

// 显示最终统计信息
func showFinalStats(hub *client.Hub) {
	fmt.Println("\n=== 最终统计信息 ===")

	ctx := context.Background()
	health := hub.GetHealth(ctx)
	fmt.Printf("系统健康状态: %v\n", health)

	metrics := hub.GetMetrics()

	if totalSent, ok := metrics["total_sent"].(int64); ok {
		fmt.Printf("总发送消息数: %d\n", totalSent)
	}

	if totalFailed, ok := metrics["total_failed"].(int64); ok {
		fmt.Printf("总失败消息数: %d\n", totalFailed)
	}

	if successRate, ok := metrics["success_rate"].(float64); ok {
		fmt.Printf("总体成功率: %.1f%%\n", successRate*100)
	}

	if avgDuration, ok := metrics["avg_duration"].(string); ok {
		fmt.Printf("平均发送耗时: %s\n", avgDuration)
	}

	if platformHealth, ok := metrics["platform_health"].(map[string]bool); ok {
		fmt.Println("平台健康状态:")
		for platform, healthy := range platformHealth {
			status := "❌"
			if healthy {
				status = "✅"
			}
			fmt.Printf("  %s %s\n", status, platform)
		}
	}
}

// 辅助函数：获取优先级名称
func getPriorityName(priority int) string {
	switch priority {
	case 1:
		return "低"
	case 2:
		return "较低"
	case 3:
		return "普通"
	case 4:
		return "高"
	case 5:
		return "紧急"
	default:
		return "未知"
	}
}

// 演示高级卡片功能
func demonstrateAdvancedCards(hub *client.Hub, ctx context.Context) {
	fmt.Println("=== 高级卡片功能示例 ===")

	// 1. 监控仪表板卡片
	demonstrateMonitoringCard(hub, ctx)

	// 2. 事件处理卡片
	demonstrateIncidentCard(hub, ctx)

	// 3. 审批流程卡片
	demonstrateApprovalCard(hub, ctx)

	// 4. 动态数据卡片
	demonstrateDynamicCard(hub, ctx)

	// 5. 复杂交互卡片
	demonstrateInteractiveCard(hub, ctx)
}

// 监控仪表板卡片
func demonstrateMonitoringCard(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n--- 监控仪表板卡片 ---")

	monitoringData := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**📊 系统监控仪表板**",
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
							"content": "**CPU使用率**\n🟢 45%",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**内存使用率**\n🟡 68%",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**磁盘使用率**\n🟢 32%",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**网络流量**\n🔵 1.2GB/s",
							"tag":     "lark_md",
						},
					},
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**服务状态**\n✅ Web服务: 正常\n✅ 数据库: 正常\n⚠️ 缓存: 性能降级\n❌ 邮件服务: 故障",
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
						"url":  "https://monitor.example.com/dashboard",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "性能报告",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://monitor.example.com/reports",
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

	message := client.NewCard("监控仪表板", "系统状态概览").
		CardData(monitoringData).
		Priority(3).
		FeishuGroup("default").
		Metadata("type", "monitoring").
		Metadata("timestamp", time.Now().Format("2006-01-02 15:04:05")).
		Build()

	results, err := hub.Send(ctx, message, &client.Options{
		Retry:      true,
		MaxRetries: 2,
		Timeout:    45 * time.Second,
	})

	if err != nil {
		log.Printf("发送监控卡片失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 监控卡片发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ 监控卡片发送失败: %s\n", result.Error)
			}
		}
	}
}

// 事件处理卡片
func demonstrateIncidentCard(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n--- 事件处理卡片 ---")

	incidentData := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**🚨 紧急事件通知**",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**事件ID**: INC-2024-001\n**级别**: 🔴 P1 - 严重\n**影响服务**: 用户登录系统\n**开始时间**: " + time.Now().Format("2006-01-02 15:04:05"),
					"tag":     "lark_md",
				},
			},
			{
				"tag": "hr",
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**问题描述**\n用户无法正常登录系统，登录接口返回500错误。初步排查发现是数据库连接池满载导致。",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**当前状态**: 🔍 调查中\n**负责人**: @张三\n**预计恢复**: 30分钟内",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "action",
				"actions": []map[string]interface{}{
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "认领处理",
							"tag":     "plain_text",
						},
						"type": "primary",
						"url":  "https://incident.example.com/claim/INC-2024-001",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "状态跟踪",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://incident.example.com/track/INC-2024-001",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "通知更新",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://incident.example.com/notify/INC-2024-001",
					},
				},
			},
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "紧急事件 - INC-2024-001",
				"tag":     "plain_text",
			},
			"template": "red",
		},
	}

	message := client.NewCard("紧急事件", "用户登录系统故障").
		CardData(incidentData).
		Priority(5). // 紧急
		FeishuGroup("default").
		Metadata("incident_id", "INC-2024-001").
		Metadata("severity", "P1").
		Build()

	results, err := hub.Send(ctx, message, &client.Options{
		Retry:      true,
		MaxRetries: 3,
		Timeout:    10 * time.Second,
	})

	if err != nil {
		log.Printf("发送事件卡片失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 事件卡片发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ 事件卡片发送失败: %s\n", result.Error)
			}
		}
	}
}

// 审批流程卡片
func demonstrateApprovalCard(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n--- 审批流程卡片 ---")

	approvalData := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**📋 待审批申请**",
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
							"content": "**申请人**\n李四",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**申请类型**\n生产环境部署",
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**申请时间**\n" + time.Now().Format("2006-01-02 15:04"),
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": "**紧急程度**\n🟡 中等",
							"tag":     "lark_md",
						},
					},
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**部署内容**\n更新用户管理模块，修复登录异常问题\n\n**版本信息**\nv2.1.3 → v2.1.4\n\n**影响范围**\n用户登录、个人信息管理",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**测试状态**: ✅ 通过\n**代码审查**: ✅ 通过\n**安全扫描**: ✅ 通过",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "action",
				"actions": []map[string]interface{}{
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "✅ 批准",
							"tag":     "plain_text",
						},
						"type": "primary",
						"url":  "https://approval.example.com/approve/APR-2024-001",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "❌ 拒绝",
							"tag":     "plain_text",
						},
						"type": "danger",
						"url":  "https://approval.example.com/reject/APR-2024-001",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "💬 留言",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://approval.example.com/comment/APR-2024-001",
					},
				},
			},
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "审批申请 - APR-2024-001",
				"tag":     "plain_text",
			},
			"template": "orange",
		},
	}

	message := client.NewCard("审批申请", "生产环境部署申请").
		CardData(approvalData).
		Priority(4).
		FeishuGroup("default").
		Metadata("approval_id", "APR-2024-001").
		Metadata("applicant", "李四").
		Build()

	// 异步发送
	messageID, err := hub.SendAsync(ctx, message, &client.Options{
		Retry:      true,
		MaxRetries: 2,
	})

	if err != nil {
		log.Printf("发送审批卡片失败: %v", err)
	} else {
		fmt.Printf("✅ 审批卡片已排队发送，消息ID: %s\n", messageID)
	}
}

// 动态数据卡片
func demonstrateDynamicCard(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n--- 动态数据卡片 ---")

	// 模拟实时数据
	currentTime := time.Now()
	cpuUsage := 42 + (currentTime.Second() % 30) // 42-72之间变化
	memUsage := 65 + (currentTime.Second() % 20) // 65-85之间变化

	dynamicData := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**📈 实时性能数据**",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": fmt.Sprintf("🕒 更新时间: %s", currentTime.Format("15:04:05")),
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
							"content": fmt.Sprintf("**CPU使用率**\n%s %d%%", getStatusIcon(cpuUsage), cpuUsage),
							"tag":     "lark_md",
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"content": fmt.Sprintf("**内存使用率**\n%s %d%%", getStatusIcon(memUsage), memUsage),
							"tag":     "lark_md",
						},
					},
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**活跃连接数**: 1,234\n**请求响应时间**: 95ms\n**错误率**: 0.01%",
					"tag":     "lark_md",
				},
			},
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "实时监控数据",
				"tag":     "plain_text",
			},
			"template": "green",
		},
	}

	message := client.NewCard("实时监控", "动态性能数据").
		CardData(dynamicData).
		Priority(2).
		FeishuGroup("default").
		Metadata("update_time", currentTime.Format("2006-01-02 15:04:05")).
		Metadata("cpu_usage", fmt.Sprintf("%d", cpuUsage)).
		Build()

	results, err := hub.Send(ctx, message, nil)
	if err != nil {
		log.Printf("发送动态卡片失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 动态卡片发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ 动态卡片发送失败: %s\n", result.Error)
			}
		}
	}
}

// 复杂交互卡片
func demonstrateInteractiveCard(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n--- 复杂交互卡片 ---")

	interactiveData := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**🎮 多媒体交互卡片**",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "hr",
			},
			{
				"tag": "img",
				"img_key":   "img_placeholder",
				"alt":       map[string]interface{}{"tag": "plain_text", "content": "示例图片"},
				"mode":      "fit_horizontal",
				"compact":   true,
				"custom_width": 300,
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**功能演示**\n这是一个包含多种交互元素的复杂卡片，展示了飞书卡片的高级功能。",
					"tag":     "lark_md",
				},
			},
			{
				"tag": "div",
				"fields": []map[string]interface{}{
					{
						"is_short": false,
						"text": map[string]interface{}{
							"content": "**选项列表**\n• 🔍 查看详细信息\n• 📊 生成分析报告\n• 🔔 设置提醒通知\n• 📤 分享给团队",
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
							"content": "🔍 详情",
							"tag":     "plain_text",
						},
						"type": "primary",
						"url":  "https://example.com/details",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "📊 报告",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://example.com/report",
					},
				},
			},
			{
				"tag": "action",
				"actions": []map[string]interface{}{
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "🔔 提醒",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://example.com/remind",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"content": "📤 分享",
							"tag":     "plain_text",
						},
						"type": "default",
						"url":  "https://example.com/share",
					},
				},
			},
			{
				"tag": "note",
				"elements": []map[string]interface{}{
					{
						"tag": "plain_text",
						"content": "💡 提示：点击按钮可以执行对应操作，此卡片展示了飞书的完整交互能力。",
					},
				},
			},
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "交互式功能演示",
				"tag":     "plain_text",
			},
			"template": "purple",
		},
	}

	message := client.NewCard("交互演示", "多功能交互卡片").
		CardData(interactiveData).
		Priority(3).
		FeishuGroup("default").
		Metadata("card_type", "interactive").
		Metadata("features", "buttons,images,notes").
		Build()

	results, err := hub.Send(ctx, message, &client.Options{
		Retry:      true,
		MaxRetries: 2,
		Timeout:    30 * time.Second,
	})

	if err != nil {
		log.Printf("发送交互卡片失败: %v", err)
	} else {
		for _, result := range results {
			if result.Success {
				fmt.Printf("✅ 交互卡片发送成功，耗时: %v\n", result.Duration)
			} else {
				fmt.Printf("❌ 交互卡片发送失败: %s\n", result.Error)
			}
		}
	}
}

// 根据使用率获取状态图标
func getStatusIcon(usage int) string {
	if usage < 50 {
		return "🟢"
	} else if usage < 80 {
		return "🟡"
	} else {
		return "🔴"
	}
}
