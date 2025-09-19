package worker

import (
	"github.com/kart-io/notifyhub/queue/core"
	"github.com/kart-io/notifyhub/queue/retry"
)

// Deprecated: Use WorkerV2 and WorkerFactory instead.
// This is kept for backward compatibility and will be removed in a future version.

// Worker 已废弃：请使用 WorkerV2 和 WorkerFactory
// 这个类型别名提供向后兼容性，将在未来版本中移除
type Worker = WorkerV2

// NewWorker 已废弃：请使用 WorkerFactory.CreateWorker()
// 这个函数提供向后兼容性，将在未来版本中移除
//
// 迁移示例:
//
//	旧版本: worker := NewWorker(queue, sender, retryPolicy, concurrency)
//	新版本:
//	  factory := NewWorkerFactory()
//	  config := &WorkerConfig{
//	      Concurrency: concurrency,
//	      RetryPolicy: retryPolicy,
//	  }
//	  worker := factory.CreateWorker(queue, sender, config)
func NewWorker(queue core.Queue, sender MessageSender, retryPolicy *retry.RetryPolicy, concurrency int) *WorkerV2 {
	config := &WorkerConfig{
		Concurrency: concurrency,
		RetryPolicy: retryPolicy,
	}

	factory := NewWorkerFactory()
	return factory.CreateWorker(queue, sender, config)
}
