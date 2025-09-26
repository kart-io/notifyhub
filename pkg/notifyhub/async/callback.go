// Package async provides callback management for asynchronous operations
package async

import (
	"sync"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// CallbackRegistry manages callbacks for asynchronous operations
type CallbackRegistry struct {
	mutex            sync.RWMutex
	globalCallbacks  *Callbacks
	messageCallbacks map[string]*Callbacks // per-message callbacks
	logger           logger.Logger
}

// Callbacks represents the set of callback functions
type Callbacks struct {
	OnResult   func(*receipt.Receipt)
	OnError    func(*message.Message, error)
	OnProgress func(completed, total int)
	OnComplete func(*BatchSummary)
}

// NewCallbackRegistry creates a new callback registry
func NewCallbackRegistry(logger logger.Logger) *CallbackRegistry {
	return &CallbackRegistry{
		globalCallbacks:  &Callbacks{},
		messageCallbacks: make(map[string]*Callbacks),
		logger:           logger,
	}
}

// RegisterGlobalCallbacks registers global callbacks for all operations
func (r *CallbackRegistry) RegisterGlobalCallbacks(callbacks *Callbacks) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if callbacks.OnResult != nil {
		r.globalCallbacks.OnResult = callbacks.OnResult
	}
	if callbacks.OnError != nil {
		r.globalCallbacks.OnError = callbacks.OnError
	}
	if callbacks.OnProgress != nil {
		r.globalCallbacks.OnProgress = callbacks.OnProgress
	}
	if callbacks.OnComplete != nil {
		r.globalCallbacks.OnComplete = callbacks.OnComplete
	}

	r.logger.Debug("Registered global callbacks")
}

// RegisterMessageCallbacks registers callbacks for a specific message
func (r *CallbackRegistry) RegisterMessageCallbacks(messageID string, callbacks *Callbacks) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.messageCallbacks[messageID] = callbacks
	r.logger.Debug("Registered message callbacks", "message_id", messageID)
}

// TriggerResult triggers result callbacks
func (r *CallbackRegistry) TriggerResult(result *receipt.Receipt) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.logger.Debug("Triggering result callbacks", "message_id", result.MessageID)

	// Trigger message-specific callback
	if callbacks, exists := r.messageCallbacks[result.MessageID]; exists && callbacks.OnResult != nil {
		go r.safeCallback(func() {
			callbacks.OnResult(result)
		}, "message result callback", result.MessageID)
	}

	// Trigger global callback
	if r.globalCallbacks.OnResult != nil {
		go r.safeCallback(func() {
			r.globalCallbacks.OnResult(result)
		}, "global result callback", result.MessageID)
	}
}

// TriggerError triggers error callbacks
func (r *CallbackRegistry) TriggerError(msg *message.Message, err error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.logger.Debug("Triggering error callbacks", "message_id", msg.ID, "error", err)

	// Trigger message-specific callback
	if callbacks, exists := r.messageCallbacks[msg.ID]; exists && callbacks.OnError != nil {
		go r.safeCallback(func() {
			callbacks.OnError(msg, err)
		}, "message error callback", msg.ID)
	}

	// Trigger global callback
	if r.globalCallbacks.OnError != nil {
		go r.safeCallback(func() {
			r.globalCallbacks.OnError(msg, err)
		}, "global error callback", msg.ID)
	}
}

// TriggerProgress triggers progress callbacks
func (r *CallbackRegistry) TriggerProgress(completed, total int) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.logger.Debug("Triggering progress callbacks", "completed", completed, "total", total)

	// Only global progress callback makes sense for batch operations
	if r.globalCallbacks.OnProgress != nil {
		go r.safeCallback(func() {
			r.globalCallbacks.OnProgress(completed, total)
		}, "global progress callback", "")
	}
}

// TriggerComplete triggers completion callbacks
func (r *CallbackRegistry) TriggerComplete(summary *BatchSummary) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.logger.Debug("Triggering completion callbacks", "batch_id", summary.BatchID)

	// Only global completion callback makes sense for batch operations
	if r.globalCallbacks.OnComplete != nil {
		go r.safeCallback(func() {
			r.globalCallbacks.OnComplete(summary)
		}, "global completion callback", summary.BatchID)
	}
}

// CleanupMessageCallbacks removes callbacks for a completed message
func (r *CallbackRegistry) CleanupMessageCallbacks(messageID string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.messageCallbacks, messageID)
	r.logger.Debug("Cleaned up message callbacks", "message_id", messageID)
}

// safeCallback executes a callback function with error recovery
func (r *CallbackRegistry) safeCallback(callback func(), callbackType, messageID string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			r.logger.Error("Callback panicked",
				"callback_type", callbackType,
				"message_id", messageID,
				"panic", recovered)
		}
	}()

	callback()
}

// GetCallbackCount returns the number of registered message callbacks (for monitoring)
func (r *CallbackRegistry) GetCallbackCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.messageCallbacks)
}

// HasGlobalCallbacks returns whether global callbacks are registered
func (r *CallbackRegistry) HasGlobalCallbacks() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.globalCallbacks.OnResult != nil ||
		r.globalCallbacks.OnError != nil ||
		r.globalCallbacks.OnProgress != nil ||
		r.globalCallbacks.OnComplete != nil
}

// ClearGlobalCallbacks clears all global callbacks
func (r *CallbackRegistry) ClearGlobalCallbacks() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.globalCallbacks = &Callbacks{}
	r.logger.Debug("Cleared global callbacks")
}

// ClearAllCallbacks clears all callbacks (global and message-specific)
func (r *CallbackRegistry) ClearAllCallbacks() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.globalCallbacks = &Callbacks{}
	r.messageCallbacks = make(map[string]*Callbacks)
	r.logger.Debug("Cleared all callbacks")
}
