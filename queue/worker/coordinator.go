package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/queue/core"
)

// DefaultWorkerCoordinator 默认工作协调器实现
// 职责：协调各个组件的交互，但不直接处理业务逻辑
type DefaultWorkerCoordinator struct {
	queueManager    QueueManager
	processor       MessageProcessor
	retryManager    RetryManager
	callbackManager CallbackManager

	// 配置
	dequeueTimeout time.Duration
}

// NewDefaultWorkerCoordinator 创建默认工作协调器
func NewDefaultWorkerCoordinator(
	queueManager QueueManager,
	processor MessageProcessor,
	retryManager RetryManager,
	callbackManager CallbackManager,
) *DefaultWorkerCoordinator {
	return &DefaultWorkerCoordinator{
		queueManager:    queueManager,
		processor:       processor,
		retryManager:    retryManager,
		callbackManager: callbackManager,
		dequeueTimeout:  5 * time.Second,
	}
}

// ProcessQueueMessage 处理单个队列消息
// 这是协调器的核心方法，协调各个组件完成消息处理
func (c *DefaultWorkerCoordinator) ProcessQueueMessage(ctx context.Context) error {
	// 1. 从队列获取消息
	queueMsg, err := c.dequeueMessage(ctx)
	if err != nil {
		return err // 可能是超时或队列为空
	}

	// 2. 处理消息
	result, err := c.processor.ProcessMessage(ctx, queueMsg.Message, queueMsg.Targets)
	if err != nil {
		// 处理器内部错误，直接失败
		c.callbackManager.OnMessageFailed(ctx, queueMsg, &ProcessResult{
			Success: false,
			Error:   err,
		})
		_ = c.queueManager.Acknowledge(queueMsg.ID) // 确认消息，避免重复处理
		return err
	}

	// 3. 根据处理结果决定后续行为
	return c.handleProcessResult(ctx, queueMsg, result)
}

// dequeueMessage 从队列出队消息
func (c *DefaultWorkerCoordinator) dequeueMessage(ctx context.Context) (*core.Message, error) {
	dequeueCtx, cancel := context.WithTimeout(ctx, c.dequeueTimeout)
	defer cancel()

	return c.queueManager.Dequeue(dequeueCtx)
}

// handleProcessResult 处理消息处理结果
func (c *DefaultWorkerCoordinator) handleProcessResult(ctx context.Context, queueMsg *core.Message, result *ProcessResult) error {
	if result.Success {
		// 成功处理
		c.callbackManager.OnMessageSent(ctx, queueMsg, result)
		return c.queueManager.Acknowledge(queueMsg.ID)
	}

	// 处理失败，检查是否需要重试
	c.callbackManager.OnMessageFailed(ctx, queueMsg, result)

	if !result.ShouldRetry || !c.retryManager.ShouldRetry(queueMsg.Attempts) {
		// 不需要重试或超过最大重试次数
		c.callbackManager.OnMaxRetriesExceeded(ctx, queueMsg, result)
		return c.queueManager.Acknowledge(queueMsg.ID)
	}

	// 需要重试
	c.callbackManager.OnMessageRetry(ctx, queueMsg, result)

	delay := c.retryManager.CalculateRetryDelay(queueMsg.Attempts)
	if err := c.retryManager.ScheduleRetry(ctx, queueMsg, delay); err != nil {
		return fmt.Errorf("failed to schedule retry: %w", err)
	}

	return nil
}

// Start 启动协调器（如果需要）
func (c *DefaultWorkerCoordinator) Start(ctx context.Context) error {
	// 协调器本身不需要启动逻辑，由Worker管理
	return nil
}

// Stop 停止协调器（如果需要）
func (c *DefaultWorkerCoordinator) Stop() error {
	// 协调器本身不需要停止逻辑，由Worker管理
	return nil
}
