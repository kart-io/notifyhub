package worker

import (
	"context"
	"testing"
	"time"

	coreMessage "github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/queue/core"
	"github.com/kart-io/notifyhub/queue/retry"
)

// MockSender 模拟消息发送器
type MockSender struct {
	shouldFail bool
}

func (m *MockSender) SendSync(ctx context.Context, message *coreMessage.Message, targets []sending.Target) (*sending.SendingResults, error) {
	if m.shouldFail {
		return &sending.SendingResults{Failed: 1}, nil
	}
	return &sending.SendingResults{Success: 1}, nil
}

// MockQueue 模拟队列
type MockQueue struct {
	messages []*core.Message
	index    int
}

func (m *MockQueue) Enqueue(ctx context.Context, msg *core.Message) (string, error) {
	return msg.ID, nil
}

func (m *MockQueue) Dequeue(ctx context.Context) (*core.Message, error) {
	if m.index >= len(m.messages) {
		return nil, context.DeadlineExceeded
	}
	msg := m.messages[m.index]
	m.index++
	return msg, nil
}

func (m *MockQueue) Ack(msgID string) error                       { return nil }
func (m *MockQueue) Nack(msgID string, nextRetry time.Time) error { return nil }
func (m *MockQueue) Close() error                                 { return nil }
func (m *MockQueue) Size() int                                    { return len(m.messages) - m.index }
func (m *MockQueue) Health(ctx context.Context) error             { return nil }

// TestWorkerV2_BasicUsage 测试基本使用方式
func TestWorkerV2_BasicUsage(t *testing.T) {
	// 创建模拟组件
	queue := &MockQueue{
		messages: []*core.Message{
			{ID: "msg1", Message: coreMessage.NewMessage()},
		},
	}
	sender := &MockSender{shouldFail: false}

	// 使用工厂创建工作器
	factory := NewWorkerFactory()
	worker := factory.CreateWorker(queue, sender, DefaultWorkerConfig())

	// 启动工作器
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := worker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// 等待一段时间让工作器处理消息
	time.Sleep(100 * time.Millisecond)

	// 停止工作器
	worker.Stop()
}

// TestWorkerV2_WithCustomComponents 测试使用自定义组件
func TestWorkerV2_WithCustomComponents(t *testing.T) {
	// 创建自定义组件
	queue := &MockQueue{}
	queueManager := NewDefaultQueueManager(queue)

	sender := &MockSender{shouldFail: false}
	processor := NewDefaultMessageProcessor(sender, 10*time.Second)

	retryPolicy := retry.ExponentialBackoffPolicy(3, 5*time.Second, 2.0)
	retryManager := NewDefaultRetryManager(retryPolicy, queueManager)

	callbackManager := NewDefaultCallbackManager(nil)

	// 使用工厂创建工作器
	factory := NewWorkerFactory()
	worker := factory.CreateWorkerWithCustomComponents(
		queueManager,
		processor,
		retryManager,
		callbackManager,
		2, // 2个并发工作协程
	)

	// 测试启动和停止
	ctx := context.Background()
	err := worker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	worker.Stop()
}

// TestWorkerV2_MinimalSetup 测试最小化设置
func TestWorkerV2_MinimalSetup(t *testing.T) {
	queue := &MockQueue{}
	sender := &MockSender{shouldFail: false}

	// 创建最小化工作器
	factory := NewWorkerFactory()
	worker := factory.CreateMinimalWorker(queue, sender, 1)

	ctx := context.Background()
	err := worker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	worker.Stop()
}

// Example 展示重构后的使用方式
func Example() {
	// 1. 准备依赖
	var queue core.Queue     // 你的队列实现
	var sender MessageSender // 你的发送器实现

	// 2. 配置工作器
	config := &WorkerConfig{
		Concurrency:    4,
		ProcessTimeout: 30 * time.Second,
		RetryPolicy:    retry.ExponentialBackoffPolicy(3, 10*time.Second, 2.0),
	}

	// 3. 创建工作器
	factory := NewWorkerFactory()
	worker := factory.CreateWorker(queue, sender, config)

	// 4. 启动工作器
	ctx := context.Background()
	if err := worker.Start(ctx); err != nil {
		panic(err)
	}

	// 5. 优雅停止
	defer worker.Stop()

	// 工作器现在会自动处理队列中的消息
}
