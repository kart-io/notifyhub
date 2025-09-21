package worker

import (
	"context"
	"testing"
	"time"

	coreTypes "github.com/kart-io/notifyhub/core"
	coreMessage "github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/queue/core"
)

// TestRefactorBenefits 验证重构后的主要优势
func TestRefactorBenefits(t *testing.T) {
	t.Run("组件独立性测试", func(t *testing.T) {
		// 测试各组件可以独立创建和测试

		// 1. 消息处理器独立测试
		mockSender := &MockSender{shouldFail: false}
		processor := NewDefaultMessageProcessor(mockSender, 10*time.Second)

		msg := coreMessage.NewMessage()
		targets := []coreTypes.Target{{Platform: "test"}}

		result, err := processor.ProcessMessage(context.Background(), msg, targets)
		if err != nil {
			t.Fatalf("Processor failed: %v", err)
		}
		if !result.Success {
			t.Error("Expected successful processing")
		}

		// 2. 重试管理器独立测试
		mockQueue := &MockQueue{}
		queueManager := NewDefaultQueueManager(mockQueue)
		retryManager := NewDefaultRetryManager(nil, queueManager)

		if !retryManager.ShouldRetry(1) {
			t.Error("Should allow retry for first attempt")
		}

		// 3. 队列管理器独立测试
		_, err = queueManager.Dequeue(context.Background())
		if err == nil {
			t.Error("Expected error for empty queue")
		}
	})

	t.Run("接口替换测试", func(t *testing.T) {
		// 验证可以轻松替换组件实现

		// 创建自定义处理器
		customProcessor := &CustomMessageProcessor{}

		// 验证接口兼容性
		var processor MessageProcessor = customProcessor

		msg := coreMessage.NewMessage()
		targets := []coreTypes.Target{{Platform: "custom"}}

		result, err := processor.ProcessMessage(context.Background(), msg, targets)
		if err != nil {
			t.Fatalf("Custom processor failed: %v", err)
		}

		if result.Success {
			t.Error("Custom processor should simulate failure")
		}
	})

	t.Run("协调器模式测试", func(t *testing.T) {
		// 验证协调器能正确协调各组件

		// 创建所有组件
		queue := &MockQueue{
			messages: []*core.Message{
				{ID: "test1", Message: coreMessage.NewMessage()},
			},
		}

		queueManager := NewDefaultQueueManager(queue)
		processor := NewDefaultMessageProcessor(&MockSender{shouldFail: false}, 5*time.Second)
		retryManager := NewDefaultRetryManager(nil, queueManager)
		callbackManager := NewDefaultCallbackManager(nil)

		// 创建协调器
		coordinator := NewDefaultWorkerCoordinator(
			queueManager, processor, retryManager, callbackManager)

		// 测试消息处理
		err := coordinator.ProcessQueueMessage(context.Background())
		if err != nil {
			t.Fatalf("Coordinator failed: %v", err)
		}
	})

	t.Run("工厂模式测试", func(t *testing.T) {
		// 验证工厂能简化复杂对象创建

		factory := NewWorkerFactory()
		queue := &MockQueue{}
		sender := &MockSender{shouldFail: false}

		// 测试不同创建方式
		worker1 := factory.CreateWorker(queue, sender, DefaultWorkerConfig())
		worker2 := factory.CreateMinimalWorker(queue, sender, 2)

		if worker1 == nil || worker2 == nil {
			t.Error("Factory should create workers")
		}

		// 测试启动停止
		ctx := context.Background()

		if err := worker1.Start(ctx); err != nil {
			t.Fatalf("Worker1 start failed: %v", err)
		}
		worker1.Stop()

		if err := worker2.Start(ctx); err != nil {
			t.Fatalf("Worker2 start failed: %v", err)
		}
		worker2.Stop()
	})
}

// CustomMessageProcessor 自定义处理器实现，用于测试接口替换
type CustomMessageProcessor struct{}

func (c *CustomMessageProcessor) ProcessMessage(ctx context.Context, msg *coreMessage.Message, targets []coreTypes.Target) (*ProcessResult, error) {
	// 模拟自定义处理逻辑
	return &ProcessResult{
		Success:     false, // 模拟失败
		Results:     &coreTypes.SendingResults{Failed: 1},
		Error:       nil,
		Duration:    time.Millisecond * 100,
		ShouldRetry: true,
	}, nil
}

// TestDependencyReduction 验证依赖减少
func TestDependencyReduction(t *testing.T) {
	t.Run("WorkerV2最小依赖", func(t *testing.T) {
		// WorkerV2只依赖WorkerCoordinator接口
		coordinator := &mockCoordinator{}
		worker := NewWorkerV2(coordinator, 1)

		if worker == nil {
			t.Error("Should create worker with minimal dependencies")
		}

		// 验证启动停止
		ctx := context.Background()
		if err := worker.Start(ctx); err != nil {
			t.Fatalf("Worker start failed: %v", err)
		}
		worker.Stop()
	})
}

// mockCoordinator 模拟协调器，验证接口依赖
type mockCoordinator struct{}

func (m *mockCoordinator) ProcessQueueMessage(ctx context.Context) error {
	time.Sleep(time.Millisecond) // 模拟处理时间
	return nil
}

func (m *mockCoordinator) Start(ctx context.Context) error { return nil }
func (m *mockCoordinator) Stop() error                     { return nil }
