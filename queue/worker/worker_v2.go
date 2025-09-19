package worker

import (
	"context"
	"sync"
	"time"
)

// WorkerV2 重构后的队列工作器
// 职责：专注于工作池管理和goroutine调度
type WorkerV2 struct {
	coordinator WorkerCoordinator
	concurrency int

	// 生命周期管理
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	stopCh  chan struct{}
	workers []chan struct{}
}

// NewWorkerV2 创建重构后的工作器
func NewWorkerV2(coordinator WorkerCoordinator, concurrency int) *WorkerV2 {
	if concurrency <= 0 {
		concurrency = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerV2{
		coordinator: coordinator,
		concurrency: concurrency,
		ctx:         ctx,
		cancel:      cancel,
		stopCh:      make(chan struct{}),
	}
}

// Start 启动工作器
func (w *WorkerV2) Start(ctx context.Context) error {
	// 启动协调器（如果需要）
	if err := w.coordinator.Start(ctx); err != nil {
		return err
	}

	// 启动工作协程
	for i := 0; i < w.concurrency; i++ {
		workerStop := make(chan struct{})
		w.workers = append(w.workers, workerStop)

		w.wg.Add(1)
		go w.workerLoop(w.ctx, workerStop, i)
	}

	return nil
}

// Stop 停止工作器
func (w *WorkerV2) Stop() {
	// 取消context
	w.cancel()

	// 关闭停止信号
	close(w.stopCh)

	// 停止所有工作协程
	for _, workerStop := range w.workers {
		close(workerStop)
	}

	// 等待所有协程结束
	w.wg.Wait()

	// 停止协调器
	_ = w.coordinator.Stop()
}

// workerLoop 工作协程主循环
// 职责：简单的循环调用协调器处理消息
func (w *WorkerV2) workerLoop(ctx context.Context, stopCh chan struct{}, workerID int) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-stopCh:
			return
		default:
			// 委托给协调器处理消息
			if err := w.coordinator.ProcessQueueMessage(ctx); err != nil {
				// 如果出现错误，可以选择记录日志或短暂休眠
				// 避免无限循环的错误
				if isTemporaryError(err) {
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}
}

// isTemporaryError 判断是否为临时错误
func isTemporaryError(err error) bool {
	// 这里可以根据错误类型判断是否为临时错误
	// 例如：队列为空、超时等
	return true // 简化实现，认为所有错误都是临时的
}
