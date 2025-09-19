package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/core/hub"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/tests/mocks"
	"github.com/kart-io/notifyhub/tests/utils"
)

func TestHub_Creation(t *testing.T) {
	helper := utils.NewTestHelper(t)

	logger := mocks.NewMockLogger()
	opts := &hub.Options{
		Logger: logger,
	}

	h := hub.NewHub(opts)

	helper.AssertNotNil(h, "Hub should not be nil")
	helper.AssertFalse(h.IsShutdown(), "Hub should not be shutdown initially")
}

func TestHub_TransportRegistration(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 注册传输器
	emailTransport := mocks.NewMockTransport("email")
	feishuTransport := mocks.NewMockTransport("feishu")
	slackTransport := mocks.NewMockTransport("slack")

	h.RegisterTransport(emailTransport)
	h.RegisterTransport(feishuTransport)
	h.RegisterTransport(slackTransport)

	// 验证传输器列表
	transports := h.ListTransports()
	helper.AssertEqual(3, len(transports), "Should have 3 transports registered")

	// 验证传输器名称
	transportMap := make(map[string]bool)
	for _, name := range transports {
		transportMap[name] = true
	}

	helper.AssertTrue(transportMap["email"], "Should have email transport")
	helper.AssertTrue(transportMap["feishu"], "Should have feishu transport")
	helper.AssertTrue(transportMap["slack"], "Should have slack transport")
}

func TestHub_SendMessage(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 注册传输器
	mockTransport := mocks.NewMockTransport("test")
	h.RegisterTransport(mockTransport)

	// 创建消息
	msg := utils.CreateTestMessageWithDetails("Test", "Test message", 3)
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}

	// 发送消息
	ctx := context.Background()
	results, err := h.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send should not error")
	helper.AssertNotNil(results, "Results should not be nil")
	helper.AssertEqual(1, len(results.Results), "Should have 1 result")

	// 验证调用记录
	calls := mockTransport.GetCalls()
	helper.AssertEqual(1, len(calls), "Should have 1 call recorded")
	helper.AssertEqual(msg.ID, calls[0].Message.ID, "Message ID should match")
}

func TestHub_SendToMultipleTargets(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 注册多个传输器
	emailTransport := mocks.NewMockTransport("email")
	feishuTransport := mocks.NewMockTransport("feishu")
	h.RegisterTransport(emailTransport)
	h.RegisterTransport(feishuTransport)

	// 创建消息和多个目标
	msg := utils.CreateTestMessageWithDetails("Multi-target", "Test message", 3)
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "user1@example.com", "email"),
		utils.CreateTestTarget(sending.TargetTypeEmail, "user2@example.com", "email"),
		utils.CreateTestTarget(sending.TargetTypeUser, "user123", "feishu"),
		utils.CreateTestTarget(sending.TargetTypeGroup, "dev-team", "feishu"),
	}

	// 发送消息
	ctx := context.Background()
	results, err := h.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send should not error")
	helper.AssertEqual(4, len(results.Results), "Should have 4 results")

	// 验证每个传输器的调用次数
	emailCalls := emailTransport.GetCalls()
	feishuCalls := feishuTransport.GetCalls()
	helper.AssertEqual(2, len(emailCalls), "Email transport should have 2 calls")
	helper.AssertEqual(2, len(feishuCalls), "Feishu transport should have 2 calls")
}

func TestHub_SendWithUnknownPlatform(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 只注册email传输器
	emailTransport := mocks.NewMockTransport("email")
	h.RegisterTransport(emailTransport)

	// 创建包含未知平台的目标
	msg := utils.CreateTestMessageWithDetails("Test", "Test message", 3)
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "email"),
		utils.CreateTestTarget(sending.TargetTypeUser, "user123", "unknown"), // 未知平台
	}

	// 发送消息
	ctx := context.Background()
	results, err := h.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send should not error (partial failure is OK)")
	helper.AssertEqual(2, len(results.Results), "Should have 2 results")

	// 验证结果
	for _, result := range results.Results {
		switch result.Target.Platform {
		case "email":
			helper.AssertTrue(result.Success, "Email should succeed")
		case "unknown":
			helper.AssertFalse(result.Success, "Unknown platform should fail")
		default:
			helper.AssertTrue(result.Success, "Other platforms should succeed")
		}
	}
}

func TestHub_SendWithTransportError(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 创建会失败的传输器
	mockTransport := mocks.NewMockTransport("test")
	mockTransport.SetError("test@example.com", errors.New("transport failure"))
	h.RegisterTransport(mockTransport)

	// 发送消息
	msg := utils.CreateTestMessageWithDetails("Test", "Test message", 3)
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}

	ctx := context.Background()
	results, err := h.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send should not error at hub level")
	helper.AssertEqual(1, len(results.Results), "Should have 1 result")
	helper.AssertFalse(results.Results[0].Success, "Result should indicate failure")
	helper.AssertNotNil(results.Results[0].Error, "Result should have error")
}

func TestHub_Middleware(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 创建和添加中间件
	middleware1 := mocks.NewMockMiddleware("middleware1")
	middleware2 := mocks.NewMockMiddleware("middleware2")

	var executionOrder []string
	middleware1.SetBeforeFunc(func(ctx context.Context, msg *message.Message, targets []sending.Target) {
		executionOrder = append(executionOrder, "middleware1-before")
	})
	middleware1.SetAfterFunc(func(ctx context.Context, results *sending.SendingResults) {
		executionOrder = append(executionOrder, "middleware1-after")
	})

	middleware2.SetBeforeFunc(func(ctx context.Context, msg *message.Message, targets []sending.Target) {
		executionOrder = append(executionOrder, "middleware2-before")
	})
	middleware2.SetAfterFunc(func(ctx context.Context, results *sending.SendingResults) {
		executionOrder = append(executionOrder, "middleware2-after")
	})

	h.AddMiddleware(middleware1)
	h.AddMiddleware(middleware2)

	// 注册传输器
	mockTransport := mocks.NewMockTransport("test")
	h.RegisterTransport(mockTransport)

	// 发送消息
	msg := utils.CreateTestMessageWithDetails("Test", "Test message", 3)
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}

	ctx := context.Background()
	results, err := h.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send should not error")
	helper.AssertNotNil(results, "Results should not be nil")

	// 验证中间件执行顺序
	expectedOrder := []string{
		"middleware1-before",
		"middleware2-before",
		"middleware2-after",
		"middleware1-after",
	}
	helper.AssertEqual(len(expectedOrder), len(executionOrder), "Execution order length should match")
	for i, expected := range expectedOrder {
		helper.AssertEqual(expected, executionOrder[i], "Execution order should match at index", i)
	}

	// 验证中间件调用记录
	calls1 := middleware1.GetCalls()
	calls2 := middleware2.GetCalls()
	helper.AssertEqual(1, len(calls1), "Middleware1 should have 1 call")
	helper.AssertEqual(1, len(calls2), "Middleware2 should have 1 call")
}

func TestHub_MiddlewareError(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 创建会失败的中间件
	errorMiddleware := mocks.NewMockMiddleware("error-middleware")
	errorMiddleware.SetShouldError(true, "middleware error")
	h.AddMiddleware(errorMiddleware)

	// 注册传输器
	mockTransport := mocks.NewMockTransport("test")
	h.RegisterTransport(mockTransport)

	// 发送消息
	msg := utils.CreateTestMessageWithDetails("Test", "Test message", 3)
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}

	ctx := context.Background()
	_, err := h.Send(ctx, msg, targets)

	helper.AssertError(err, "Send should error when middleware fails")
	helper.AssertContains(err.Error(), "middleware error", "Error should contain middleware error message")

	// 验证传输器未被调用
	calls := mockTransport.GetCalls()
	helper.AssertEqual(0, len(calls), "Transport should not be called when middleware fails")
}

func TestHub_ConcurrentSends(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 注册传输器
	mockTransport := mocks.NewMockTransport("test")
	h.RegisterTransport(mockTransport)

	// 并发发送消息
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			msg := utils.CreateTestMessageWithDetails("Test", "Concurrent message", 3)
			msg.AddMetadata("goroutine", string(rune(id)))

			targets := []sending.Target{
				utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
			}

			ctx := context.Background()
			results, err := h.Send(ctx, msg, targets)

			helper.AssertNoError(err, "Send should not error")
			helper.AssertNotNil(results, "Results should not be nil")

			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 验证所有调用都被记录
	calls := mockTransport.GetCalls()
	helper.AssertEqual(numGoroutines, len(calls), "Should have recorded all concurrent calls")
}

func TestHub_ContextCancellation(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 创建慢速传输器
	mockTransport := mocks.NewMockTransport("test")
	mockTransport.SetDelay(100 * time.Millisecond)
	h.RegisterTransport(mockTransport)

	// 创建快速超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	msg := utils.CreateTestMessageWithDetails("Test", "Test message", 3)
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}

	// 发送消息
	_, err := h.Send(ctx, msg, targets)

	// 应该因为上下文取消而失败
	if err == nil {
		// 如果没有错误，检查是否是因为操作太快
		helper.AssertTrue(false, "Expected context cancellation error")
	}
}

func TestHub_Shutdown(t *testing.T) {
	helper := utils.NewTestHelper(t)

	logger := mocks.NewMockLogger()
	h := hub.NewHub(&hub.Options{Logger: logger})

	// 注册传输器
	mockTransport := mocks.NewMockTransport("test")
	h.RegisterTransport(mockTransport)

	// 关闭Hub
	ctx := context.Background()
	err := h.Shutdown(ctx)

	helper.AssertNoError(err, "Shutdown should not error")
	helper.AssertTrue(h.IsShutdown(), "Hub should be shutdown")

	// 验证日志
	helper.AssertTrue(logger.HasMessage("shutting down hub"), "Should log shutdown")
}

func TestHub_SendAfterShutdown(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)

	// 注册传输器
	mockTransport := mocks.NewMockTransport("test")
	h.RegisterTransport(mockTransport)

	// 关闭Hub
	ctx := context.Background()
	err := h.Shutdown(ctx)
	helper.AssertNoError(err, "Shutdown should not error")

	// 尝试在关闭后发送
	msg := utils.CreateTestMessageWithDetails("Test", "Test message", 3)
	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}

	_, err = h.Send(ctx, msg, targets)
	helper.AssertError(err, "Send should error after shutdown")
}

func TestHub_EmptyTargets(t *testing.T) {
	helper := utils.NewTestHelper(t)

	h := hub.NewHub(nil)
	defer func() { _ = h.Shutdown(context.Background()) }()

	// 注册传输器
	mockTransport := mocks.NewMockTransport("test")
	h.RegisterTransport(mockTransport)

	// 发送没有目标的消息
	msg := utils.CreateTestMessageWithDetails("Test", "Test message", 3)
	targets := []sending.Target{}

	ctx := context.Background()
	results, err := h.Send(ctx, msg, targets)

	helper.AssertNoError(err, "Send should not error with empty targets")
	helper.AssertEqual(0, len(results.Results), "Should have 0 results")

	// 验证传输器未被调用
	calls := mockTransport.GetCalls()
	helper.AssertEqual(0, len(calls), "Transport should not be called with empty targets")
}
