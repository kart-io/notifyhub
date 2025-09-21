package worker

import (
	"context"
	"time"

	coreTypes "github.com/kart-io/notifyhub/core"
	coreMessage "github.com/kart-io/notifyhub/core/message"
)

// DefaultMessageProcessor 默认消息处理器实现
// 职责：专注于消息发送逻辑
type DefaultMessageProcessor struct {
	sender  MessageSender
	timeout time.Duration
}

// NewDefaultMessageProcessor 创建默认消息处理器
func NewDefaultMessageProcessor(sender MessageSender, timeout time.Duration) *DefaultMessageProcessor {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &DefaultMessageProcessor{
		sender:  sender,
		timeout: timeout,
	}
}

// ProcessMessage 处理消息发送
func (p *DefaultMessageProcessor) ProcessMessage(ctx context.Context, msg *coreMessage.Message, targets []coreTypes.Target) (*ProcessResult, error) {
	start := time.Now()

	// 创建带超时的context
	sendCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// 发送消息
	results, err := p.sender.SendSync(sendCtx, msg, targets)
	duration := time.Since(start)

	// 构建处理结果
	result := &ProcessResult{
		Success:     err == nil && p.isSuccessful(results),
		Results:     results,
		Error:       err,
		Duration:    duration,
		ShouldRetry: p.shouldRetry(err, results),
	}

	return result, nil
}

// isSuccessful 检查发送结果是否成功
func (p *DefaultMessageProcessor) isSuccessful(results *coreTypes.SendingResults) bool {
	if results == nil {
		return false
	}
	return results.Failed == 0
}

// shouldRetry 判断是否应该重试
func (p *DefaultMessageProcessor) shouldRetry(err error, results *coreTypes.SendingResults) bool {
	// 如果有发送错误，需要重试
	if err != nil {
		return true
	}

	// 如果有失败的结果，需要重试
	if results != nil && results.Failed > 0 {
		return true
	}

	return false
}
