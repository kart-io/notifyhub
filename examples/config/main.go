package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/queue"
)

func main() {
	ctx := context.Background()

	// ================================
	// 示例1：使用环境变量的默认配置
	// ================================
	log.Println("=== 示例1: 默认配置 (从环境变量) ===")

	hub1, err := client.New(config.WithDefaults())
	if err != nil {
		log.Printf("创建Hub失败: %v", err)
	} else {
		log.Println("✅ 使用默认配置创建Hub成功")
		hub1.Stop() // 清理资源
	}

	// ================================
	// 示例2：手动配置各个组件
	// ================================
	log.Println("\n=== 示例2: 手动配置 ===")

	hub2, err := client.New(
		// Feishu配置
		config.WithFeishu(
			"https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url",
			"your-secret", // 可选
		),
		// 邮件配置
		config.WithEmail(
			"smtp.gmail.com", // SMTP服务器
			587,              // 端口
			"your@gmail.com", // 用户名
			"app-password",   // 密码
			"your@gmail.com", // 发送方
		),
		// 队列配置
		config.WithQueue(
			"memory", // 队列类型
			2000,     // 缓冲区大小
			4,        // 工作器数量
		),
		// 路由规则
		config.WithRouting(
			// 高优先级消息发送到所有平台
			config.NewRoutingRule("high_priority").
				WithPriority(4, 5).
				RouteTo("feishu", "email").
				Build(),
			// 告警消息只发送到飞书
			config.NewRoutingRule("alerts_only_feishu").
				WithMetadata("type", "alert").
				RouteTo("feishu").
				Build(),
		),
	)

	if err != nil {
		log.Printf("创建Hub失败: %v", err)
	} else {
		log.Println("✅ 使用手动配置创建Hub成功")

		// 启动服务
		if err := hub2.Start(ctx); err != nil {
			log.Printf("启动Hub失败: %v", err)
		} else {
			log.Println("✅ Hub服务启动成功")

			// 测试发送消息
			testMessage(ctx, hub2)
		}

		hub2.Stop() // 清理资源
	}

	// ================================
	// 示例3：混合配置（环境变量 + 手动）
	// ================================
	log.Println("\n=== 示例3: 混合配置 ===")

	hub3, err := client.New(
		// 从环境变量加载Feishu和Email配置
		config.WithFeishuFromEnv(),
		config.WithEmailFromEnv(),
		// 手动设置队列配置
		config.WithQueue("memory", 500, 2),
		// 使用默认路由规则
		config.WithDefaultRouting(),
	)

	if err != nil {
		log.Printf("创建Hub失败: %v", err)
	} else {
		log.Println("✅ 使用混合配置创建Hub成功")
		hub3.Stop() // 清理资源
	}

	// ================================
	// 示例4：测试环境配置
	// ================================
	log.Println("\n=== 示例4: 测试环境配置 ===")

	hub4, err := client.New(
		config.WithTestDefaults(), // 测试友好的默认配置
		// 可以添加测试用的通知配置
		config.WithFeishu("https://httpbin.org/post", ""), // 用于测试的端点
	)

	if err != nil {
		log.Printf("创建Hub失败: %v", err)
	} else {
		log.Println("✅ 使用测试配置创建Hub成功")

		// 演示测试环境下的使用
		if err := hub4.Start(ctx); err != nil {
			log.Printf("启动Hub失败: %v", err)
		} else {
			log.Println("✅ 测试Hub服务启动成功")

			// 测试消息
			testMessage := client.NewNotice("测试消息", "这是一条测试消息").
				Variable("environment", "test").
				Build()

			results, err := hub4.Send(ctx, testMessage, nil)
			if err != nil {
				log.Printf("发送测试消息失败: %v", err)
			} else {
				log.Printf("✅ 测试消息发送成功，结果数量: %d", len(results))
			}
		}

		hub4.Stop()
	}

	// ================================
	// 示例5：动态配置和自定义重试策略
	// ================================
	log.Println("\n=== 示例5: 动态配置 ===")

	// 创建自定义重试策略
	aggressiveRetry := queue.ExponentialBackoffPolicy(5, 10*time.Second, 1.5)

	hub5, err := client.New(
		config.WithFeishu("https://httpbin.org/post", ""),
		config.WithQueue("memory", 1000, 3),
		config.WithQueueRetryPolicy(aggressiveRetry),
		// 创建复杂的路由规则
		config.WithRouting(
			config.NewRoutingRule("emergency_all").
				Enabled(true).
				WithPriority(5).
				WithMessageType("emergency", "critical").
				RouteTo("feishu", "email").
				Build(),
			config.NewRoutingRule("normal_feishu_only").
				WithPriority(1, 2, 3).
				RouteTo("feishu").
				Build(),
		),
	)

	if err != nil {
		log.Printf("创建Hub失败: %v", err)
	} else {
		log.Println("✅ 使用动态配置创建Hub成功")
		hub5.Stop()
	}

	log.Println("\n🎉 所有配置示例演示完成!")
}

// testMessage 测试消息发送
func testMessage(ctx context.Context, hub *client.Hub) {
	// 创建测试消息
	message := client.NewAlert("配置测试", "这是一条用于测试配置的消息").
		Variable("timestamp", time.Now().Format(time.RFC3339)).
		Variable("config_test", true).
		Build()

	// 发送消息
	results, err := hub.Send(ctx, message, nil)
	if err != nil {
		log.Printf("❌ 发送消息失败: %v", err)
	} else {
		log.Printf("✅ 消息发送成功，结果数量: %d", len(results))
		for i, result := range results {
			status := "失败"
			if result.Success {
				status = "成功"
			}
			log.Printf("  结果 %d: 平台=%s, 状态=%s, 耗时=%v",
				i+1, result.Platform, status, result.Duration)
		}
	}

	// 显示指标
	metrics := hub.GetMetrics()
	if totalSent, ok := metrics["total_sent"]; ok {
		log.Printf("📊 总发送数量: %v", totalSent)
	}
}
