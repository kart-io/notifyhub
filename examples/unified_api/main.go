package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub"
)

func main() {
	// 演示统一API的使用方式
	demonstrateUnifiedAPI()
}

func demonstrateUnifiedAPI() {
	ctx := context.Background()
	// 1. 创建客户端 - 统一入口
	client, err := notifyhub.New(
		notifyhub.WithFeishu("https://open.feishu.cn/webhook/xxx", "secret"),
		notifyhub.WithEmail("smtp.example.com", 587, "user", "pass", "noreply@company.com"),
		notifyhub.WithMemoryQueue(1000, 4),
		notifyhub.WithSimpleRetry(3),
		notifyhub.WithDevelopment(),
	)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer func() { _ = client.Shutdown(context.Background()) }()

	// 2. 基础消息发送 - 统一的流畅接口
	log.Println("=== 基础消息发送 ===")
	result, err := client.Send(ctx).
		Title("系统维护通知").
		Body("系统将在今晚22:00进行维护，预计持续2小时").
		Priority(3).
		ToEmail("admin@company.com", "ops@company.com").
		ToFeishu("maintenance-alerts").
		Execute()

	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		log.Printf("发送成功: %d条成功, %d条失败", result.Sent, result.Failed)
	}

	// 3. 告警消息 - 专门的告警API
	log.Println("\n=== 告警消息发送 ===")
	alertResult, err := client.Alert(ctx).
		Title("🚨 数据库连接异常").
		Body("生产环境数据库连接数超过阈值").
		Metadata("service", "database").
		Metadata("environment", "production").
		ToEmail("oncall@company.com").
		ToFeishu("critical-alerts").
		Execute()

	if err != nil {
		log.Printf("告警发送失败: %v", err)
	} else {
		log.Printf("告警发送成功: MessageID=%s", alertResult.MessageID)
	}

	// 4. 通知消息 - 专门的通知API
	log.Println("\n=== 通知消息发送 ===")
	notifResult, err := client.Notification(ctx).
		Title("📊 每日报告").
		Body("今日系统运行正常，处理请求 1,234,567 次").
		ToEmail("team@company.com").
		Execute()

	if err != nil {
		log.Printf("通知发送失败: %v", err)
	} else {
		log.Printf("通知发送成功: %+v", notifResult)
	}

	// 5. 模板消息
	log.Println("\n=== 模板消息发送 ===")
	templateResult, err := client.Send(ctx).
		Template("user-welcome").
		Title("欢迎 {{.username}} 加入我们！").
		Body("Hi {{.username}}, 欢迎加入 {{.company}}！您的账号已激活。").
		Variable("username", "张三").
		Variable("company", "科技公司").
		Variable("activation_url", "https://company.com/activate/xxx").
		ToEmail("zhangsan@company.com").
		Execute()

	if err != nil {
		log.Printf("模板消息发送失败: %v", err)
	} else {
		log.Printf("模板消息发送成功: %+v", templateResult)
	}

	// 6. 延迟发送
	log.Println("\n=== 延迟消息发送 ===")
	delayResult, err := client.Send(ctx).
		Title("⏰ 定时提醒").
		Body("这是一条延迟5秒发送的消息").
		DelayBy(5 * time.Second).
		ToEmail("admin@company.com").
		Execute()

	if err != nil {
		log.Printf("延迟消息发送失败: %v", err)
	} else {
		log.Printf("延迟消息发送成功: %+v", delayResult)
	}

	// 7. 批量发送不同类型
	log.Println("\n=== 批量发送演示 ===")
	go sendMultipleMessages(client)

	// 8. 模拟运行 - 调试功能
	log.Println("\n=== 模拟运行演示 ===")
	dryResult, err := client.Send(ctx).
		Title("测试消息").
		Body("这是一条测试消息").
		ToEmail("test@company.com").
		DryRun()

	if err != nil {
		log.Printf("模拟运行失败: %v", err)
	} else {
		log.Printf("模拟运行结果: Valid=%v, Targets=%d",
			dryResult.Valid, len(dryResult.Targets))
	}

	// 9. 健康检查
	log.Println("\n=== 健康检查 ===")
	health := client.Health()
	log.Printf("系统健康状态: %+v", health)

	// 等待一些异步操作完成
	time.Sleep(2 * time.Second)
}

func sendMultipleMessages(client *notifyhub.Client) {
	ctx := context.Background()

	// 批量发送不同优先级的消息
	messages := []struct {
		title    string
		body     string
		priority int
		targets  func(*notifyhub.SendBuilder) *notifyhub.SendBuilder
	}{
		{
			title:    "高优先级告警",
			body:     "紧急处理",
			priority: 5,
			targets: func(b *notifyhub.SendBuilder) *notifyhub.SendBuilder {
				return b.ToEmail("urgent@company.com").ToFeishu("urgent-alerts")
			},
		},
		{
			title:    "中优先级通知",
			body:     "正常处理",
			priority: 3,
			targets: func(b *notifyhub.SendBuilder) *notifyhub.SendBuilder {
				return b.ToEmail("normal@company.com")
			},
		},
		{
			title:    "低优先级信息",
			body:     "稍后处理",
			priority: 1,
			targets: func(b *notifyhub.SendBuilder) *notifyhub.SendBuilder {
				return b.ToFeishu("info-channel")
			},
		},
	}

	for i, msg := range messages {
		builder := client.Send(ctx).
			Title(msg.title).
			Body(msg.body).
			Priority(msg.priority).
			Metadata("batch_id", "demo_batch").
			Metadata("message_index", string(rune(i)))

		result, err := msg.targets(builder).Execute()
		if err != nil {
			log.Printf("批量消息 %d 发送失败: %v", i+1, err)
		} else {
			log.Printf("批量消息 %d 发送成功: %s", i+1, result.MessageID)
		}

		// 避免发送过快
		time.Sleep(100 * time.Millisecond)
	}
}

// 演示高级配置
