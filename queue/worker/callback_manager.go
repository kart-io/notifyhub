package worker

import (
	"context"

	"github.com/kart-io/notifyhub/queue/callbacks"
	"github.com/kart-io/notifyhub/queue/core"
)

// DefaultCallbackManager 默认回调管理器实现
// 职责：专注于回调执行逻辑
type DefaultCallbackManager struct {
	executor *callbacks.CallbackExecutor
}

// NewDefaultCallbackManager 创建默认回调管理器
func NewDefaultCallbackManager(executor *callbacks.CallbackExecutor) *DefaultCallbackManager {
	if executor == nil {
		executor = callbacks.NewCallbackExecutor()
	}
	return &DefaultCallbackManager{
		executor: executor,
	}
}

// OnMessageSent 消息发送成功回调
func (cm *DefaultCallbackManager) OnMessageSent(ctx context.Context, queueMsg *core.Message, result *ProcessResult) {
	cm.executor.ExecuteCallbacks(
		ctx,
		callbacks.CallbackEventSent,
		queueMsg,
		result.Results,
		nil,
		result.Duration,
	)
}

// OnMessageFailed 消息发送失败回调
func (cm *DefaultCallbackManager) OnMessageFailed(ctx context.Context, queueMsg *core.Message, result *ProcessResult) {
	cm.executor.ExecuteCallbacks(
		ctx,
		callbacks.CallbackEventFailed,
		queueMsg,
		result.Results,
		result.Error,
		result.Duration,
	)
}

// OnMessageRetry 消息重试回调
func (cm *DefaultCallbackManager) OnMessageRetry(ctx context.Context, queueMsg *core.Message, result *ProcessResult) {
	cm.executor.ExecuteCallbacks(
		ctx,
		callbacks.CallbackEventRetry,
		queueMsg,
		result.Results,
		result.Error,
		result.Duration,
	)
}

// OnMaxRetriesExceeded 超过最大重试次数回调
func (cm *DefaultCallbackManager) OnMaxRetriesExceeded(ctx context.Context, queueMsg *core.Message, result *ProcessResult) {
	cm.executor.ExecuteCallbacks(
		ctx,
		callbacks.CallbackEventMaxRetries,
		queueMsg,
		result.Results,
		result.Error,
		result.Duration,
	)
}
