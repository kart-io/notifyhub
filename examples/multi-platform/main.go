package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub"
)

func main() {
	// 同时配置飞书和邮件
	hub, err := notifyhub.New(
		// 飞书配置
		notifyhub.WithFeishu(
			"https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url",
			"your-secret", // 可选
		),
		// 邮件配置
		notifyhub.WithEmail(
			"smtp.gmail.com",    // SMTP服务器
			587,                 // 端口
			"your@gmail.com",    // 用户名
			"your-app-password", // 密码
			"your@gmail.com",    // 发送方
		),
		// 队列配置
		notifyhub.WithQueue("memory", 1000, 2),
		// 路由规则：高优先级消息同时发送到飞书和邮件
		notifyhub.WithRouting(
			notifyhub.NewRoutingRule("high_priority_all").
				WithPriority(4, 5).
				RouteTo("feishu", "email"). // 同时路由到两个平台
				Build(),
			notifyhub.NewRoutingRule("normal_feishu_only").
				WithPriority(1, 2, 3).
				RouteTo("feishu"). // 普通消息只发飞书
				Build(),
		),
	)
	if err != nil {
		log.Fatalf("创建Hub失败: %v", err)
	}

	// 启动服务
	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("启动Hub失败: %v", err)
	}
	defer hub.Stop()

	// ========================================
	// 示例1：使用路由规则自动分发到多平台
	// ========================================
	log.Println("=== 示例1: 高优先级告警 - 自动发送到飞书和邮件 ===")

	alertMessage := notifyhub.NewAlert("系统严重告警", "数据库服务器宕机").
		Variable("server", "prod-db-01").
		Variable("environment", "production").
		Variable("error", "connection refused").
		Priority(5). // 高优先级，会触发路由规则发送到两个平台
		Build()

	results, err := hub.Send(ctx, alertMessage, nil)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		log.Printf("✅ 发送成功，共发送到 %d 个平台:", len(results))
		for _, result := range results {
			status := "失败"
			if result.Success {
				status = "成功"
			}
			log.Printf("  - 平台: %s, 状态: %s, 耗时: %v", result.Platform, status, result.Duration)
		}
	}

	// ========================================
	// 示例2：手动指定多个目标平台
	// ========================================
	log.Println("\n=== 示例2: 手动指定目标 - 同时发送到指定的飞书群和邮箱 ===")

	manualMessage := notifyhub.NewNotice("项目更新通知", "新版本 v2.1.0 已发布").
		Variable("version", "v2.1.0").
		Variable("features", []string{"新增用户管理", "优化性能", "修复bug"}).
		Variable("release_date", time.Now().Format("2006-01-02")).
		// 手动指定多个目标
		FeishuGroup("dev-team").      // 发送到飞书开发团队群
		FeishuUser("zhang_san").      // 发送到飞书用户
		Email("team@company.com").    // 发送到团队邮箱
		Email("manager@company.com"). // 发送到管理员邮箱
		Priority(3).                  // 普通优先级
		Build()

	results, err = hub.Send(ctx, manualMessage, nil)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		log.Printf("✅ 发送成功，共发送到 %d 个目标:", len(results))
		for _, result := range results {
			status := "失败"
			if result.Success {
				status = "成功"
			}
			log.Printf("  - 平台: %s, 状态: %s, 耗时: %v", result.Platform, status, result.Duration)
		}
	}

	// ========================================
	// 示例3：使用模板同时发送到多平台
	// ========================================
	log.Println("\n=== 示例3: 使用模板 - 报告同时发送到多个平台 ===")

	reportMessage := notifyhub.NewReport("每日系统报告", "").
		Template("report"). // 使用内置报告模板
		Variable("metrics", map[string]interface{}{
			"cpu_usage":    "45%",
			"memory_usage": "78%",
			"disk_usage":   "32%",
			"uptime":       "99.9%",
		}).
		Variable("summary", "系统运行正常，无异常").
		FeishuGroup("ops-team").
		Email("ops@company.com").
		Email("cto@company.com").
		Priority(2).
		Build()

	results, err = hub.Send(ctx, reportMessage, nil)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		log.Printf("✅ 发送成功，共发送到 %d 个目标:", len(results))
		for _, result := range results {
			status := "失败"
			if result.Success {
				status = "成功"
			}
			log.Printf("  - 平台: %s, 状态: %s, 耗时: %v", result.Platform, status, result.Duration)
		}
	}

	// ========================================
	// 示例4：异步批量发送到多平台
	// ========================================
	log.Println("\n=== 示例4: 异步批量发送 ===")

	batchMessage := notifyhub.NewNotice("批量通知", "这是一条异步批量通知").
		Variable("batch_id", "batch_001").
		FeishuGroup("all-staff").
		Email("all@company.com").
		Build()

	// 异步发送
	taskID, err := hub.SendAsync(ctx, batchMessage, notifyhub.NewAsyncOptions())
	if err != nil {
		log.Printf("异步发送失败: %v", err)
	} else {
		log.Printf("✅ 异步任务已入队: %s", taskID)
	}

	// 等待异步处理完成
	time.Sleep(3 * time.Second)

	// ========================================
	// 监控和指标
	// ========================================
	log.Println("\n=== 系统指标 ===")

	metrics := hub.GetMetrics()
	if totalSent, ok := metrics["total_sent"]; ok {
		log.Printf("📊 总发送数量: %v", totalSent)
	}
	if successRate, ok := metrics["success_rate"]; ok {
		log.Printf("📊 成功率: %.2f%%", successRate.(float64)*100)
	}

	// 健康检查
	health := hub.GetHealth(ctx)
	log.Printf("🏥 系统健康状态: %v", health["status"])

	log.Println("\n🎉 多平台通知演示完成!")
}
