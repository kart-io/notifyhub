package worker

import (
	"context"
	"time"

	coreMessage "github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/queue/core"
)

// MessageSender defines interface for sending messages
type MessageSender interface {
	SendSync(ctx context.Context, message *coreMessage.Message, targets []sending.Target) (*sending.SendingResults, error)
}

// MessageProcessor 定义消息处理接口
// 负责处理单个消息的发送逻辑
type MessageProcessor interface {
	ProcessMessage(ctx context.Context, msg *coreMessage.Message, targets []sending.Target) (*ProcessResult, error)
}

// ProcessResult 表示消息处理结果
type ProcessResult struct {
	Success     bool
	Results     *sending.SendingResults
	Error       error
	Duration    time.Duration
	ShouldRetry bool
}

// RetryManager 定义重试管理接口
// 负责重试策略和重试调度
type RetryManager interface {
	ShouldRetry(attempts int) bool
	CalculateRetryDelay(attempts int) time.Duration
	ScheduleRetry(ctx context.Context, queueMsg *core.Message, delay time.Duration) error
}

// CallbackManager 定义回调管理接口
// 负责执行各种事件回调
type CallbackManager interface {
	OnMessageSent(ctx context.Context, queueMsg *core.Message, result *ProcessResult)
	OnMessageFailed(ctx context.Context, queueMsg *core.Message, result *ProcessResult)
	OnMessageRetry(ctx context.Context, queueMsg *core.Message, result *ProcessResult)
	OnMaxRetriesExceeded(ctx context.Context, queueMsg *core.Message, result *ProcessResult)
}

// QueueManager 定义队列管理接口
// 负责队列操作的抽象
type QueueManager interface {
	Dequeue(ctx context.Context) (*core.Message, error)
	Acknowledge(msgID string) error
	Reject(msgID string, nextRetry time.Time) error
}

// WorkerCoordinator 协调器接口
// 负责协调各个组件的交互
type WorkerCoordinator interface {
	ProcessQueueMessage(ctx context.Context) error
	Start(ctx context.Context) error
	Stop() error
}
