package integration

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/api"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/config/routing"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/queue"
	"github.com/kart-io/notifyhub/tests/mocks"
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

	logger := mocks.NewMockLogger()
	opts := &api.Options{Logger: logger}

	// 创建NotifyHub
	hub, err := api.New(cfg, opts)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 创建消息
	msg := hub.NewMessage()
	msg.SetTitle("Integration Test")
	msg.SetBody("This is an end-to-end test message")
	msg.SetPriority(3)
	msg.AddMetadata("source", "integration-test")
	msg.AddVariable("test_id", "e2e-001")

	// 添加目标
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "email"),
		utils.CreateTestTarget(sending.TargetTypeUser, "user123", "feishu"),
		utils.CreateTestTarget(sending.TargetTypeGroup, "alerts", "slack"),
	}

	// 发送通知
	ctx := context.Background()
	results, err := hub.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send failed")
	helper.AssertNotNil(results, "Results should not be nil")
	helper.AssertEqual(len(targets), len(results.Results), "Should have results for all targets")

	// 验证日志
	helper.AssertTrue(logger.HasMessage("sending notification"), "Should log sending")

	// 获取指标
	metrics := hub.GetMetrics()
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

	logger := mocks.NewMockLogger()
	opts := &api.Options{Logger: logger}

	hub, err := api.New(cfg, opts)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 创建消息
	msg := hub.NewMessage()
	msg.SetTitle("Multi-Platform Alert")
	msg.SetBody("This message goes to multiple platforms")
	msg.SetPriority(4)
	msg.SetFormat("markdown")

	// 定义多平台目标
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "admin@company.com", "email"),
		utils.CreateTestTarget(sending.TargetTypeEmail, "manager@company.com", "email"),
		utils.CreateTestTarget(sending.TargetTypeGroup, "alerts", "feishu"),
		utils.CreateTestTarget(sending.TargetTypeUser, "oncall-user", "feishu"),
	}

	// 发送
	ctx := context.Background()
	results, err := hub.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send failed")
	helper.AssertEqual(len(targets), len(results.Results), "Should have results for all targets")

	// 统计平台分布
	platformCounts := make(map[string]int)
	for _, result := range results.Results {
		platformCounts[result.Target.Platform]++
	}

	helper.AssertEqual(2, platformCounts["email"], "Should have 2 email results")
	helper.AssertEqual(2, platformCounts["feishu"], "Should have 2 feishu results")
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

	hub, err := api.New(cfg, nil)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 测试严重告警路由
	criticalMsg := hub.NewMessage()
	criticalMsg.SetTitle("Critical System Failure")
	criticalMsg.SetBody("Database is down")
	criticalMsg.SetPriority(5)
	criticalMsg.AddMetadata("type", "critical")

	originalTargets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeUser, "reporter", "feishu"),
	}

	ctx := context.Background()
	results, err := hub.Send(ctx, criticalMsg, originalTargets)

	helper.AssertNoError(err, "Send failed")
	// 应该有原始目标 + 路由规则添加的目标
	helper.AssertTrue(len(results.Results) >= 3, "Should have additional targets from routing")

	// 测试警告级别路由
	warningMsg := hub.NewMessage()
	warningMsg.SetTitle("High Memory Usage")
	warningMsg.SetBody("Memory usage is at 85%")
	warningMsg.SetPriority(3)

	results, err = hub.Send(ctx, warningMsg, originalTargets)

	helper.AssertNoError(err, "Send failed")
	helper.AssertTrue(len(results.Results) >= 2, "Should have additional targets from routing")

	// 测试信息级别路由
	infoMsg := hub.NewMessage()
	infoMsg.SetTitle("System Update")
	infoMsg.SetBody("System will be updated tonight")
	infoMsg.SetPriority(1)

	results, err = hub.Send(ctx, infoMsg, originalTargets)

	helper.AssertNoError(err, "Send failed")
	helper.AssertTrue(len(results.Results) >= 2, "Should have additional targets from routing")
}

func TestConcurrentNotifications(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 200, 4),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg, nil)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	numMessages := 20
	results := make(chan error, numMessages)

	// 并发发送消息
	for i := 0; i < numMessages; i++ {
		go func(id int) {
			msg := hub.NewMessage()
			msg.SetTitle("Concurrent Message")
			msg.SetBody("Message from goroutine")
			msg.SetPriority(3)
			msg.AddVariable("message_id", id)

			targets := []sending.Target{
				utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
			}

			_, err := hub.Send(ctx, msg, targets)
			results <- err
		}(i)
	}

	// 收集结果
	for i := 0; i < numMessages; i++ {
		err := <-results
		helper.AssertNoError(err, "Message send failed", i)
	}

	// 验证指标
	metrics := hub.GetMetrics()
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

	logger := mocks.NewMockLogger()
	opts := &api.Options{Logger: logger}

	hub, err := api.New(cfg, opts)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	msg := hub.NewMessage()
	msg.SetTitle("Test Failure Handling")
	msg.SetBody("This message will fail to send")
	msg.SetPriority(3)

	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "failing-notifier"),
	}

	ctx := context.Background()
	results, err := hub.Send(ctx, msg, targets)

	// 发送操作本身不应该失败
	helper.AssertNoError(err, "Hub send should not fail")
	helper.AssertNotNil(results, "Results should not be nil")

	// 但具体的结果应该显示失败
	helper.AssertEqual(1, len(results.Results), "Should have 1 result")
	helper.AssertFalse(results.Results[0].Success, "Result should indicate failure")

	// 检查错误日志
	helper.AssertTrue(logger.HasError(), "Should have error logs")
}

func TestTargetExpressionResolution(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithQueue("memory", 100, 1),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg, nil)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// Register mock transports for testing
	testutil.RegisterMockTransports(hub, 10*time.Millisecond)

	msg := hub.NewMessage()
	msg.SetTitle("Target Expression Test")
	msg.SetBody("Testing target expression resolution")
	msg.SetPriority(3)

	// 测试目标表达式
	expressions := []string{
		"direct:email:admin@company.com",
		"direct:feishu:user:john",
		"static:default", // Use static provider for group targets
	}

	ctx := context.Background()
	results, err := hub.SendToTargetExpressions(ctx, msg, expressions)

	helper.AssertNoError(err, "SendToTargetExpressions failed")
	helper.AssertNotNil(results, "Results should not be nil")
	helper.AssertTrue(len(results.Results) > 0, "Should have resolved some targets")
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

	hub, err := api.New(cfg, nil)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 异步发送多个消息
	numMessages := 10
	for i := 0; i < numMessages; i++ {
		msg := hub.NewMessage()
		msg.SetTitle("Queued Message")
		msg.SetBody("Message for queue processing")
		msg.SetPriority(3)
		msg.AddVariable("sequence", i)

		targets := []sending.Target{
			utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
		}

		// 使用Send进行发送
		ctx := context.Background()
		result, err := hub.Send(ctx, msg, targets)
		helper.AssertNoError(err, "Send failed", i)
		helper.AssertNotNil(result, "Should have result")
	}

	// 等待队列处理
	time.Sleep(500 * time.Millisecond)

	// 验证指标
	metrics := hub.GetMetrics()
	helper.AssertNotNil(metrics, "Metrics should not be nil")
}

func TestCompleteLifecycle(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 100, 2),
		config.WithSilentLogger(),
	)

	logger := mocks.NewMockLogger()
	opts := &api.Options{Logger: logger}

	hub, err := api.New(cfg, opts)
	helper.AssertNoError(err, "Failed to create hub")

	// 验证初始状态
	helper.AssertFalse(hub.IsShutdown(), "Hub should not be shutdown initially")

	// 发送消息
	msg := hub.NewMessage()
	msg.SetTitle("Lifecycle Test")
	msg.SetBody("Testing complete lifecycle")
	msg.SetPriority(3)

	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}

	ctx := context.Background()
	results, err := hub.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send failed")
	helper.AssertNotNil(results, "Results should not be nil")

	// 获取传输器列表
	transports := hub.GetTransports()
	helper.AssertTrue(len(transports) > 0, "Should have registered transports")

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = hub.Shutdown(shutdownCtx)
	helper.AssertNoError(err, "Shutdown failed")

	// 验证关闭状态
	helper.AssertTrue(hub.IsShutdown(), "Hub should be shutdown")

	// 验证关闭后操作失败
	_, err = hub.Send(ctx, msg, targets)
	helper.AssertError(err, "Send should fail after shutdown")
}

func TestMessageTemplating(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg, nil)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 创建带模板的消息
	msg := hub.NewMessage()
	msg.SetTemplate("alert-template")
	msg.SetTitle("Alert: {{.service}} is down")
	msg.SetBody("Service {{.service}} in {{.environment}} is experiencing issues. Priority: {{.priority}}")
	msg.AddVariable("service", "user-service")
	msg.AddVariable("environment", "production")
	msg.AddVariable("priority", "high")

	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "admin@company.com", "test"),
	}

	ctx := context.Background()
	results, err := hub.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send failed")
	helper.AssertEqual(1, len(results.Results), "Should have 1 result")

	// 验证模板变量被使用
	helper.AssertEqual(3, len(msg.GetVariables()), "Should have 3 variables")
}
