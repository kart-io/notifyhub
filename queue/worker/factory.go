package worker

import (
	"time"

	"github.com/kart-io/notifyhub/queue/callbacks"
	"github.com/kart-io/notifyhub/queue/core"
	"github.com/kart-io/notifyhub/queue/retry"
)

// WorkerConfig 工作器配置
type WorkerConfig struct {
	Concurrency    int
	ProcessTimeout time.Duration
	DequeueTimeout time.Duration
	RetryPolicy    *retry.RetryPolicy
}

// DefaultWorkerConfig 返回默认配置
func DefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		Concurrency:    4,
		ProcessTimeout: 30 * time.Second,
		DequeueTimeout: 5 * time.Second,
		RetryPolicy:    retry.DefaultRetryPolicy(),
	}
}

// WorkerFactory 工作器工厂
type WorkerFactory struct{}

// NewWorkerFactory 创建工作器工厂
func NewWorkerFactory() *WorkerFactory {
	return &WorkerFactory{}
}

// CreateWorker 创建完整的工作器实例
// 这个方法封装了组件组装的复杂性
func (f *WorkerFactory) CreateWorker(
	queue core.Queue,
	sender MessageSender,
	config *WorkerConfig,
) *WorkerV2 {
	if config == nil {
		config = DefaultWorkerConfig()
	}

	// 创建各个组件
	queueManager := NewDefaultQueueManager(queue)
	processor := NewDefaultMessageProcessor(sender, config.ProcessTimeout)
	retryManager := NewDefaultRetryManager(config.RetryPolicy, queueManager)
	callbackManager := NewDefaultCallbackManager(callbacks.NewCallbackExecutor())

	// 创建协调器
	coordinator := NewDefaultWorkerCoordinator(
		queueManager,
		processor,
		retryManager,
		callbackManager,
	)

	// 创建工作器
	return NewWorkerV2(coordinator, config.Concurrency)
}

// CreateWorkerWithCustomComponents 使用自定义组件创建工作器
// 允许用户替换特定的组件实现
func (f *WorkerFactory) CreateWorkerWithCustomComponents(
	queueManager QueueManager,
	processor MessageProcessor,
	retryManager RetryManager,
	callbackManager CallbackManager,
	concurrency int,
) *WorkerV2 {
	coordinator := NewDefaultWorkerCoordinator(
		queueManager,
		processor,
		retryManager,
		callbackManager,
	)

	return NewWorkerV2(coordinator, concurrency)
}

// CreateMinimalWorker 创建最小化的工作器（用于简单场景）
func (f *WorkerFactory) CreateMinimalWorker(
	queue core.Queue,
	sender MessageSender,
	concurrency int,
) *WorkerV2 {
	config := &WorkerConfig{
		Concurrency:    concurrency,
		ProcessTimeout: 30 * time.Second,
		RetryPolicy:    retry.NoRetryPolicy(), // 不重试
	}

	return f.CreateWorker(queue, sender, config)
}
