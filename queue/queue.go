// Package queue provides message queuing functionality with support for
// different backends, callbacks, retry policies, and delayed message scheduling.
package queue

import (
	"github.com/kart-io/notifyhub/queue/callbacks"
	"github.com/kart-io/notifyhub/queue/core"
	"github.com/kart-io/notifyhub/queue/retry"
	"github.com/kart-io/notifyhub/queue/scheduler"
	"github.com/kart-io/notifyhub/queue/worker"
)

// Re-export core types and interfaces for backward compatibility
type (
	// Queue defines the interface for message queues
	Queue = core.Queue

	// Message represents a message in the queue
	Message = core.Message

	// MessageSender defines interface for sending messages
	MessageSender = worker.MessageSender

	// Worker processes messages from the queue
	Worker = worker.Worker

	// RetryPolicy defines retry behavior
	RetryPolicy = retry.RetryPolicy

	// Callback defines the interface for message processing callbacks
	Callback = callbacks.Callback

	// CallbackOptions defines callback configuration for a message
	CallbackOptions = callbacks.CallbackOptions

	// CallbackEvent represents the event type for callbacks
	CallbackEvent = callbacks.CallbackEvent

	// CallbackContext contains information about the callback execution
	CallbackContext = callbacks.CallbackContext

	// MessageScheduler handles delayed message scheduling
	MessageScheduler = scheduler.MessageScheduler

	// EnhancedQueue wraps a basic queue with scheduling capabilities
	EnhancedQueue = scheduler.EnhancedQueue
)

// Re-export callback events for backward compatibility
const (
	CallbackEventSent       = callbacks.CallbackEventSent
	CallbackEventFailed     = callbacks.CallbackEventFailed
	CallbackEventRetry      = callbacks.CallbackEventRetry
	CallbackEventMaxRetries = callbacks.CallbackEventMaxRetries
)

// Re-export constructor functions for backward compatibility

// NewSimple creates a new in-memory queue
func NewSimple(bufferSize int) Queue {
	return core.NewSimple(bufferSize)
}

// NewWorker creates a new queue worker
func NewWorker(queue Queue, sender MessageSender, retryPolicy *RetryPolicy, concurrency int) *Worker {
	return worker.NewWorker(queue, sender, retryPolicy, concurrency)
}

// Retry policy constructors
var (
	DefaultRetryPolicy       = retry.DefaultRetryPolicy
	ExponentialBackoffPolicy = retry.ExponentialBackoffPolicy
	LinearBackoffPolicy      = retry.LinearBackoffPolicy
	NoRetryPolicy            = retry.NoRetryPolicy
	AggressiveRetryPolicy    = retry.AggressiveRetryPolicy
)

// Callback constructors
var (
	NewCallbackFunc     = callbacks.NewCallbackFunc
	NewLoggingCallback  = callbacks.NewLoggingCallback
	NewCallbackExecutor = callbacks.NewCallbackExecutor
)

// Scheduler constructors
var (
	NewMessageScheduler = scheduler.NewMessageScheduler
	NewEnhancedQueue    = scheduler.NewEnhancedQueue
)
