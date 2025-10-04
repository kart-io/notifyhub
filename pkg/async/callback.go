// Package async provides callback management for async operations
package async

import (
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/receipt"
)

// CompletionCallback is called when an async operation completes successfully
type CompletionCallback func(*receipt.Receipt)

// ErrorCallback is called when an async operation fails
type ErrorCallback func(*message.Message, error)

// ProgressCallback is called to report progress of batch operations
type ProgressCallback func(completed, total int)

// RetryCallback is called before retrying a failed operation
type RetryCallback func(*message.Message, int, error)

// CallbackManager manages callbacks for async operations
type CallbackManager struct {
	onComplete CompletionCallback
	onError    ErrorCallback
	onProgress ProgressCallback
	onRetry    RetryCallback
}

// NewCallbackManager creates a new callback manager
func NewCallbackManager() *CallbackManager {
	return &CallbackManager{}
}

// OnComplete sets the completion callback
func (cm *CallbackManager) OnComplete(callback CompletionCallback) *CallbackManager {
	cm.onComplete = callback
	return cm
}

// OnError sets the error callback
func (cm *CallbackManager) OnError(callback ErrorCallback) *CallbackManager {
	cm.onError = callback
	return cm
}

// OnProgress sets the progress callback
func (cm *CallbackManager) OnProgress(callback ProgressCallback) *CallbackManager {
	cm.onProgress = callback
	return cm
}

// OnRetry sets the retry callback
func (cm *CallbackManager) OnRetry(callback RetryCallback) *CallbackManager {
	cm.onRetry = callback
	return cm
}

// TriggerComplete calls the completion callback if set
func (cm *CallbackManager) TriggerComplete(receipt *receipt.Receipt) {
	if cm.onComplete != nil {
		cm.onComplete(receipt)
	}
}

// TriggerError calls the error callback if set
func (cm *CallbackManager) TriggerError(msg *message.Message, err error) {
	if cm.onError != nil {
		cm.onError(msg, err)
	}
}

// TriggerProgress calls the progress callback if set
func (cm *CallbackManager) TriggerProgress(completed, total int) {
	if cm.onProgress != nil {
		cm.onProgress(completed, total)
	}
}

// TriggerRetry calls the retry callback if set
func (cm *CallbackManager) TriggerRetry(msg *message.Message, attempt int, err error) {
	if cm.onRetry != nil {
		cm.onRetry(msg, attempt, err)
	}
}

// HasCallbacks returns true if any callbacks are set
func (cm *CallbackManager) HasCallbacks() bool {
	return cm.onComplete != nil || cm.onError != nil || cm.onProgress != nil || cm.onRetry != nil
}

// CallbackOptions provides fluent options for setting callbacks
type CallbackOptions struct {
	manager *CallbackManager
}

// NewCallbackOptions creates new callback options
func NewCallbackOptions() *CallbackOptions {
	return &CallbackOptions{
		manager: NewCallbackManager(),
	}
}

// OnComplete sets the completion callback
func (co *CallbackOptions) OnComplete(callback CompletionCallback) *CallbackOptions {
	co.manager.OnComplete(callback)
	return co
}

// OnError sets the error callback
func (co *CallbackOptions) OnError(callback ErrorCallback) *CallbackOptions {
	co.manager.OnError(callback)
	return co
}

// OnProgress sets the progress callback
func (co *CallbackOptions) OnProgress(callback ProgressCallback) *CallbackOptions {
	co.manager.OnProgress(callback)
	return co
}

// OnRetry sets the retry callback
func (co *CallbackOptions) OnRetry(callback RetryCallback) *CallbackOptions {
	co.manager.OnRetry(callback)
	return co
}

// Build returns the configured callback manager
func (co *CallbackOptions) Build() *CallbackManager {
	return co.manager
}
