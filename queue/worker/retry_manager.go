package worker

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/queue/core"
	"github.com/kart-io/notifyhub/queue/retry"
)

// DefaultRetryManager 默认重试管理器实现
// 职责：专注于重试策略和重试调度
type DefaultRetryManager struct {
	policy       *retry.RetryPolicy
	queueManager QueueManager
}

// NewDefaultRetryManager 创建默认重试管理器
func NewDefaultRetryManager(policy *retry.RetryPolicy, queueManager QueueManager) *DefaultRetryManager {
	if policy == nil {
		policy = retry.DefaultRetryPolicy()
	}
	return &DefaultRetryManager{
		policy:       policy,
		queueManager: queueManager,
	}
}

// ShouldRetry 判断是否应该重试
func (rm *DefaultRetryManager) ShouldRetry(attempts int) bool {
	return rm.policy.ShouldRetry(attempts)
}

// CalculateRetryDelay 计算重试延迟
func (rm *DefaultRetryManager) CalculateRetryDelay(attempts int) time.Duration {
	nextRetry := rm.policy.NextRetry(attempts)
	if nextRetry.IsZero() {
		return 0
	}
	return time.Until(nextRetry)
}

// ScheduleRetry 调度重试
func (rm *DefaultRetryManager) ScheduleRetry(ctx context.Context, queueMsg *core.Message, delay time.Duration) error {
	nextRetry := time.Now().Add(delay)

	// 拒绝消息并设置下次重试时间
	if err := rm.queueManager.Reject(queueMsg.ID, nextRetry); err != nil {
		return err
	}

	// 异步重新入队（简化实现）
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			// 这里可以通过队列管理器重新入队
			// 具体实现取决于队列类型
		}
	}()

	return nil
}
