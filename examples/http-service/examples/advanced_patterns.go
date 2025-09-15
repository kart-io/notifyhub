package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
)

// ==========================================
// 高级模式示例 - 展示所有优化特性
// ==========================================

func main() {
	ctx := context.Background()

	// ==========================================
	// 1. 各种初始化方式
	// ==========================================

	// 方式1：零配置（推荐）
	hub1, _ := client.NewAndStart(ctx, config.WithDefaults())
	defer hub1.Stop()

	// 方式2：测试环境
	hub2, _ := client.NewForTestingAndStart(ctx)
	defer hub2.Stop()

	// 方式3：Must 模式（适合演示）
	hub3 := client.MustNewAndStart(ctx, config.WithDefaults())
	defer hub3.Stop()

	// 使用第一个 hub 进行示例
	hub := hub1

	// ==========================================
	// 2. 消息构建模式
	// ==========================================

	// 模式1：快速构建器
	quickEmail := client.QuickEmail("Status Update", "All systems operational", "ops@example.com").Build()

	// 模式2：条件构建
	isUrgent := os.Getenv("URGENT") == "true"
	conditionalMsg := client.NewAlert("Deployment", "v2.0.0 deployed").
		If(isUrgent, func(b *client.MessageBuilder) *client.MessageBuilder {
			return b.Urgent().Email("cto@example.com").ToSlack("#urgent")
		}).
		Unless(isUrgent, func(b *client.MessageBuilder) *client.MessageBuilder {
			return b.Normal().Metadata("priority", "normal")
		}).Build()

	// 模式3：克隆和修改
	baseTemplate := client.NewMessage().
		AsMarkdown().
		Metadata("source", "monitoring").
		Metadata("version", "1.0.0")

	alert1 := baseTemplate.Clone().
		Title("CPU Alert").
		Body("CPU usage > 90%").
		Email("ops@example.com").
		Build()

	alert2 := baseTemplate.Clone().
		Title("Memory Alert").
		Body("Memory usage > 85%").
		Email("ops@example.com").
		Build()

	// 模式4：智能目标检测
	smartMsg := client.NewNotice("Team Update", "Sprint completed").
		ToEmail("team@example.com").           // 自动检测邮件
		ToSlack("#general").                    // 自动处理 Slack 频道
		ToSlack("@john").                       // 自动处理 Slack 用户
		ToFeishu("project-group").              // 飞书群组
		Build()

	// 模式5：Must 构建（测试/演示用）
	mustMsg := client.NewMessage().
		Title("Test").
		Body("Must build pattern").
		Email("test@example.com").
		MustBuild() // 验证失败会 panic

	// ==========================================
	// 3. 发送模式
	// ==========================================

	// 模式1：同步发送
	results, err := hub.Send(ctx, quickEmail, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	}
	fmt.Printf("Sent to %d targets\n", len(results))

	// 模式2：异步发送
	taskID, err := hub.SendAsync(ctx, conditionalMsg, nil)
	if err != nil {
		log.Printf("Async send failed: %v", err)
	}
	fmt.Printf("Queued with task ID: %s\n", taskID)

	// 模式3：带选项发送
	_, err = hub.Send(ctx, alert1, &client.Options{
		Retry:    true,
		MaxRetry: 3,
		Timeout:  30 * time.Second,
		Async:    false,
	})

	// 模式4：批量发送
	messages := []*notifiers.Message{alert1, alert2, smartMsg, mustMsg}
	batchResults, err := hub.SendBatch(ctx, messages, &client.Options{
		Async: true, // 异步批量
	})
	fmt.Printf("Batch sent: %d results\n", len(batchResults))

	// 模式5：便捷函数
	hub.SendText(ctx, "Quick Text", "Simple notification",
		notifiers.Target{Type: notifiers.TargetTypeEmail, Value: "user@example.com"})

	hub.SendAlert(ctx, "ALERT", "Critical issue detected",
		notifiers.Target{Type: notifiers.TargetTypeEmail, Value: "oncall@example.com"},
		notifiers.Target{Type: notifiers.TargetTypeUser, Value: "ops-team", Platform: "slack"})

	hub.SendWithTemplate(ctx, "weekly_report", map[string]interface{}{
		"week":     "2024-W03",
		"sales":    "$125,000",
		"newUsers": 450,
	},
		notifiers.Target{Type: notifiers.TargetTypeEmail, Value: "reports@example.com"})

	// ==========================================
	// 4. 高级特性
	// ==========================================

	// 特性1：延迟发送
	delayedMsg := client.NewNotice("Reminder", "Meeting in 1 hour").
		Email("team@example.com").
		Delay(1 * time.Hour).
		Build()
	hub.SendAsync(ctx, delayedMsg, nil)

	// 特性2：模板消息
	templateMsg := client.TemplatedMessage("alert_template", map[string]interface{}{
		"service":  "payment-api",
		"status":   "down",
		"duration": "5 minutes",
		"impact":   "high",
	}).
		Email("incidents@example.com").
		ToSlack("#incidents").
		Urgent().
		Build()

	// 特性3：格式化消息
	htmlMsg := client.NewMessage().
		Title("HTML Newsletter").
		Body("<h1>Welcome!</h1><p>Thanks for subscribing.</p>").
		AsHTML().
		Email("subscriber@example.com").
		Build()

	markdownMsg := client.NewMessage().
		Title("Markdown Report").
		Body("# Report\n\n- Item 1\n- Item 2\n\n**Important:** Please review").
		AsMarkdown().
		Email("reviewer@example.com").
		Build()

	// 特性4：元数据和变量
	richMsg := client.NewMessage().
		Title("Rich Message").
		Body("Complex notification with metadata").
		Email("admin@example.com").
		Variable("timestamp", time.Now().Unix()).
		Variable("hostname", os.Hostname).
		Variable("environment", os.Getenv("ENV")).
		Metadata("trace_id", "abc-123-def").
		Metadata("correlation_id", "xyz-789").
		Metadata("source_service", "monitoring").
		Build()

	// 特性5：Try 模式（安全降级）
	maybeMsg := client.NewMessage().
		Title("").  // 故意留空，会验证失败
		Body("Test").
		TryBuild() // 返回 nil 而不是 panic

	if maybeMsg == nil {
		log.Println("Message validation failed, using fallback")
		maybeMsg = client.QuickNotice("Fallback", "Default message").Build()
	}

	// ==========================================
	// 5. 监控和健康检查
	// ==========================================

	// 获取健康状态
	health := hub.GetHealth(ctx)
	fmt.Printf("Health status: %v\n", health["status"])

	// 获取指标
	metrics := hub.GetMetrics()
	fmt.Printf("Total sent: %v\n", metrics["total_sent"])
	fmt.Printf("Success rate: %v\n", metrics["success_rate"])

	log.Println("✅ All advanced patterns demonstrated!")
}

// ==========================================
// 辅助函数示例
// ==========================================

// 创建每日报告消息
func createDailyReport(date string, stats map[string]interface{}) *notifiers.Message {
	return client.TemplatedMessage("daily_report", stats).
		Variable("date", date).
		Variable("generated_at", time.Now().Format(time.RFC3339)).
		Email("reports@example.com").
		Email("management@example.com").
		Normal().
		Build()
}

// 创建告警消息（根据严重程度）
func createAlert(severity string, title, body string) *notifiers.Message {
	builder := client.NewAlert(title, body)

	switch severity {
	case "critical":
		builder.Urgent().
			Email("cto@example.com").
			Email("oncall@example.com").
			ToSlack("#critical-alerts").
			Metadata("escalate", "true")
	case "warning":
		builder.High().
			Email("ops@example.com").
			ToSlack("#warnings")
	default:
		builder.Normal().
			Email("monitoring@example.com")
	}

	return builder.
		Metadata("severity", severity).
		Metadata("timestamp", time.Now().Format(time.RFC3339)).
		Build()
}

// 批量创建用户通知
func createUserNotifications(userEmails []string, title, body string) []*notifiers.Message {
	var messages []*notifiers.Message

	for _, email := range userEmails {
		msg := client.QuickEmail(title, body, email).
			Variable("user_email", email).
			Variable("unsubscribe_link", fmt.Sprintf("https://example.com/unsubscribe?email=%s", email)).
			Build()
		messages = append(messages, msg)
	}

	return messages
}