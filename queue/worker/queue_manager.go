package worker

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/queue/core"
)

// DefaultQueueManager 默认队列管理器实现
// 职责：专注于队列操作的抽象
type DefaultQueueManager struct {
	queue core.Queue
}

// NewDefaultQueueManager 创建默认队列管理器
func NewDefaultQueueManager(queue core.Queue) *DefaultQueueManager {
	return &DefaultQueueManager{
		queue: queue,
	}
}

// Dequeue 从队列中出队消息
func (qm *DefaultQueueManager) Dequeue(ctx context.Context) (*core.Message, error) {
	return qm.queue.Dequeue(ctx)
}

// Acknowledge 确认消息处理成功
func (qm *DefaultQueueManager) Acknowledge(msgID string) error {
	return qm.queue.Ack(msgID)
}

// Reject 拒绝消息并设置重试时间
func (qm *DefaultQueueManager) Reject(msgID string, nextRetry time.Time) error {
	return qm.queue.Nack(msgID, nextRetry)
}
