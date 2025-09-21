package integration

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/api"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/config/routing"
	"github.com/kart-io/notifyhub/queue"
	"github.com/kart-io/notifyhub/tests/testutil"
	"github.com/kart-io/notifyhub/tests/utils"
)

func TestEndToEndNotificationFlow(t *testing.T) {
	helper := utils.NewTestHelper(t)

	// 创建配置
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 100, 2),
		config.WithSilentLogger(),
	)

	// 创建NotifyHub
	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 发送通知
	results, err := hub.Send().
		Title("Integration Test").
		Body("This is an end-to-end test message").
		Priority(3).
		Meta("source", "integration-test").
		Vars(map[string]interface{}{"test_id": "e2e-001"}).
		To("test@example.com").
		Send(context.Background())

	helper.AssertNoError(err, "Send failed")
	helper.AssertNotNil(results, "Results should not be nil")

	// 获取指标
	metrics := hub.Metrics()
	helper.AssertNotNil(metrics, "Metrics should not be nil")
}

func TestMultiPlatformNotification(t *testing.T) {
	helper := utils.NewTestHelper(t)

	// 配置多个平台
	cfg := config.New(
		config.WithFeishu("https://example.com/webhook", "secret"),
		config.WithEmail("smtp.example.com", 587, "user", "pass", "from@example.com"),
		config.WithQueue("memory", 100, 2),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 发送
	results, err := hub.Send().
		Title("Multi-Platform Alert").
		Body("This message goes to multiple platforms").
		Priority(4).
		Format("markdown").
		To("admin@company.com", "manager@company.com").
		Send(context.Background())

	helper.AssertNoError(err, "Send failed")
	helper.AssertNotNil(results, "Results should not be nil")
}

func TestRoutingIntegration(t *testing.T) {
	helper := utils.NewTestHelper(t)

	// 创建路由规则
	rules := []routing.Rule{
		{
			Name:     "critical_alerts",
			Priority: 100,
			Enabled:  true,
			Conditions: routing.Conditions{
				Priorities: []int{5},
				Metadata: map[string]string{
					"type": "critical",
				},
			},
			Actions: routing.Actions{
				Targets: []routing.Target{
					{Type: "email", Value: "oncall@company.com", Platform: "email"},
					{Type: "group", Value: "critical-alerts", Platform: "feishu"},
				},
			},
		},
		{
			Name:     "warning_alerts",
			Priority: 50,
			Enabled:  true,
			Conditions: routing.Conditions{
				Priorities: []int{3, 4},
			},
			Actions: routing.Actions{
				Targets: []routing.Target{
					{Type: "group", Value: "warnings", Platform: "feishu"},
				},
			},
		},
		{
			Name:     "info_messages",
			Priority: 10,
			Enabled:  true,
			Conditions: routing.Conditions{
				Priorities: []int{1, 2},
			},
			Actions: routing.Actions{
				Targets: []routing.Target{
					{Type: "channel", Value: "info", Platform: "slack"},
				},
			},
		},
	}

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 100, 1),
		config.WithRouting(rules...),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 测试严重告警路由
	_, err = hub.Send().
		Title("Critical System Failure").
		Body("Database is down").
		Priority(5).
		Meta("type", "critical").
		To("reporter@example.com").
		Send(context.Background())

	helper.AssertNoError(err, "Send failed")

	// 测试警告级别路由
	_, err = hub.Send().
		Title("High Memory Usage").
		Body("Memory usage is at 85%").
		Priority(3).
		To("reporter@example.com").
		Send(context.Background())

	helper.AssertNoError(err, "Send failed")

	// 测试信息级别路由
	_, err = hub.Send().
		Title("System Update").
		Body("System will be updated tonight").
		Priority(1).
		To("reporter@example.com").
		Send(context.Background())

	helper.AssertNoError(err, "Send failed")
}

func TestConcurrentNotifications(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 200, 4),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	numMessages := 20
	results := make(chan error, numMessages)

	// 并发发送消息
	for i := 0; i < numMessages; i++ {
		go func(id int) {
			_, err := hub.Send().
				Title("Concurrent Message").
				Body("Message from goroutine").
				Priority(3).
				Vars(map[string]interface{}{"message_id": id}).
				To("test@example.com").
				Send(context.Background())
			results <- err
		}(i)
	}

	// 收集结果
	for i := 0; i < numMessages; i++ {
		err := <-results
		helper.AssertNoError(err, "Message send failed", i)
	}

	// 验证指标
	metrics := hub.Metrics()
	helper.AssertNotNil(metrics, "Metrics should not be nil")
}

func TestErrorHandlingIntegration(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("failing-notifier"),
		config.WithMockNotifierFailure(),
		config.WithQueue("memory", 100, 1),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	results, err := hub.Send().
		Title("Test Failure Handling").
		Body("This message will fail to send").
		Priority(3).
		To("test@example.com").
		Send(context.Background())

	// 发送操作本身不应该失败
	helper.AssertNoError(err, "Hub send should not fail")
	helper.AssertNotNil(results, "Results should not be nil")
}

func TestTargetExpressionResolution(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithQueue("memory", 100, 1),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// Register mock transports for testing
	testutil.RegisterMockTransports(hub, 10*time.Millisecond)

	// Use direct target specification
	results, err := hub.Send().
		Title("Target Expression Test").
		Body("Testing target expression resolution").
		Priority(3).
		To("admin@company.com", "john@example.com").
		Send(context.Background())

	helper.AssertNoError(err, "Send failed")
	helper.AssertNotNil(results, "Results should not be nil")
}

func TestQueuedMessageProcessing(t *testing.T) {
	helper := utils.NewTestHelper(t)

	// 创建带队列的配置
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 100, 2),
		config.WithQueueRetryPolicy(&queue.RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			Multiplier:      2.0,
		}),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 异步发送多个消息
	numMessages := 10
	for i := 0; i < numMessages; i++ {
		// 使用Send进行发送
		result, err := hub.Send().
			Title("Queued Message").
			Body("Message for queue processing").
			Priority(3).
			Vars(map[string]interface{}{"sequence": i}).
			To("test@example.com").
			Send(context.Background())
		helper.AssertNoError(err, "Send failed", i)
		helper.AssertNotNil(result, "Should have result")
	}

	// 等待队列处理
	time.Sleep(500 * time.Millisecond)

	// 验证指标
	metrics := hub.Metrics()
	helper.AssertNotNil(metrics, "Metrics should not be nil")
}

func TestCompleteLifecycle(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 100, 2),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")

	// 发送消息
	results, err := hub.Send().
		Title("Lifecycle Test").
		Body("Testing complete lifecycle").
		Priority(3).
		To("test@example.com").
		Send(context.Background())

	helper.AssertNoError(err, "Send failed")
	helper.AssertNotNil(results, "Results should not be nil")

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = hub.Shutdown(shutdownCtx)
	helper.AssertNoError(err, "Shutdown failed")
}

func TestMessageTemplating(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 创建带模板的消息
	results, err := hub.Send().
		Template("alert-template").
		Title("Alert: {{.service}} is down").
		Body("Service {{.service}} in {{.environment}} is experiencing issues. Priority: {{.priority}}").
		Vars(map[string]interface{}{
			"service":     "user-service",
			"environment": "production",
			"priority":    "high",
		}).
		To("admin@company.com").
		Send(context.Background())

	helper.AssertNoError(err, "Send failed")
	helper.AssertNotNil(results, "Results should not be nil")
}
