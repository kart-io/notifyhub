// Package queue provides asynchronous message queue functionality for NotifyHub
// It supports multiple backends including in-memory and Redis implementations
package queue

import (
	"context"
	"errors"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/utils/idgen"
)

var (
	// ErrQueueFull is returned when the queue is at capacity
	ErrQueueFull = errors.New("queue is full")

	// ErrQueueEmpty is returned when attempting to dequeue from an empty queue
	ErrQueueEmpty = errors.New("queue is empty")

	// ErrQueueClosed is returned when operating on a closed queue
	ErrQueueClosed = errors.New("queue is closed")

	// ErrInvalidMessage is returned when the message is invalid
	ErrInvalidMessage = errors.New("invalid message")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timed out")
)

// Priority defines message priority levels
type Priority int

const (
	PriorityLow    Priority = 0
	PriorityNormal Priority = 1
	PriorityHigh   Priority = 2
	PriorityUrgent Priority = 3
)

// QueueMessage represents a queue message wrapper for any payload
// This is distinct from notifyhub.Message which represents notification content
type QueueMessage struct {
	ID          string                 `json:"id"`
	Payload     interface{}            `json:"payload"` // Can be *message.Message or other types
	Priority    Priority               `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Queue defines the interface for message queue implementations
type Queue interface {
	// Enqueue adds a message to the queue
	Enqueue(ctx context.Context, msg *QueueMessage) error

	// EnqueueBatch adds multiple messages to the queue
	EnqueueBatch(ctx context.Context, msgs []*QueueMessage) error

	// Dequeue retrieves and removes a message from the queue
	Dequeue(ctx context.Context) (*QueueMessage, error)

	// DequeueBatch retrieves and removes multiple messages from the queue
	DequeueBatch(ctx context.Context, count int) ([]*QueueMessage, error)

	// Peek retrieves a message without removing it from the queue
	Peek(ctx context.Context) (*QueueMessage, error)

	// Size returns the number of messages in the queue
	Size() int

	// IsEmpty returns true if the queue has no messages
	IsEmpty() bool

	// Clear removes all messages from the queue
	Clear() error

	// Close closes the queue and releases resources
	Close() error
}

// DelayQueue extends Queue with scheduled message support
type DelayQueue interface {
	Queue

	// EnqueueDelayed adds a message to be processed after a delay
	EnqueueDelayed(ctx context.Context, msg *QueueMessage, delay time.Duration) error

	// EnqueueScheduled adds a message to be processed at a specific time
	EnqueueScheduled(ctx context.Context, msg *QueueMessage, scheduledAt time.Time) error
}

// PriorityQueue extends Queue with priority-based processing
type PriorityQueue interface {
	Queue

	// EnqueueWithPriority adds a message with a specific priority
	EnqueueWithPriority(ctx context.Context, msg *QueueMessage, priority Priority) error

	// DequeueByPriority retrieves the highest priority message
	DequeueByPriority(ctx context.Context) (*QueueMessage, error)
}

// RetryQueue extends Queue with retry functionality
type RetryQueue interface {
	Queue

	// Retry moves a message back to the queue with incremented retry count
	Retry(ctx context.Context, msg *QueueMessage) error

	// MoveToDeadLetter moves a message to the dead letter queue
	MoveToDeadLetter(ctx context.Context, msg *QueueMessage) error

	// GetDeadLetterQueue returns the dead letter queue
	GetDeadLetterQueue() Queue
}

// QueueStats represents queue statistics
type QueueStats struct {
	Size            int           `json:"size"`
	EnqueuedCount   int64         `json:"enqueued_count"`
	DequeuedCount   int64         `json:"dequeued_count"`
	RetryCount      int64         `json:"retry_count"`
	DeadLetterCount int64         `json:"dead_letter_count"`
	AverageWaitTime time.Duration `json:"average_wait_time"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// ObservableQueue extends Queue with monitoring capabilities
type ObservableQueue interface {
	Queue

	// GetStats returns queue statistics
	GetStats() *QueueStats

	// GetMetrics returns detailed queue metrics
	GetMetrics() *QueueMetrics

	// Subscribe registers a callback for queue events
	Subscribe(event string, callback func(msg *QueueMessage))

	// Unsubscribe removes a callback for queue events
	Unsubscribe(event string)
}

// QueueMetrics contains detailed queue performance metrics
type QueueMetrics struct {
	Size            int64         `json:"size"`              // 当前队列大小
	EnqueuedCount   int64         `json:"enqueued_count"`    // 总入队数
	DequeuedCount   int64         `json:"dequeued_count"`    // 总出队数
	RetryCount      int64         `json:"retry_count"`       // 总重试次数
	DeadLetterCount int64         `json:"dead_letter_count"` // 死信队列大小
	AverageLatency  time.Duration `json:"average_latency"`   // 平均处理延迟
	ThroughputQPS   float64       `json:"throughput_qps"`    // 处理速率 (QPS)
}

// WorkerMetrics contains worker pool performance metrics
type WorkerMetrics struct {
	ActiveWorkers  int           `json:"active_workers"`  // 活跃工作协程数
	IdleWorkers    int           `json:"idle_workers"`    // 空闲工作协程数
	ProcessedCount int64         `json:"processed_count"` // 总处理消息数
	ErrorCount     int64         `json:"error_count"`     // 总错误数
	AverageTime    time.Duration `json:"average_time"`    // 平均处理时间
}

// HealthStatus represents system health status
type HealthStatus struct {
	Status      string        `json:"status"`       // "healthy" | "degraded" | "unhealthy"
	QueueSize   int           `json:"queue_size"`   // 队列大小
	Workers     int           `json:"workers"`      // 工作协程数
	LastProcess time.Time     `json:"last_process"` // 最后处理时间
	Errors      []HealthError `json:"errors"`       // 错误列表
	Uptime      time.Duration `json:"uptime"`       // 运行时间
}

// HealthError represents a health check error
type HealthError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"` // "warning" | "error" | "critical"
}

// ScalePolicy defines worker pool scaling behavior
type ScalePolicy struct {
	ScaleUpThreshold   int           `json:"scale_up_threshold"`   // 扩容阈值 (队列深度)
	ScaleDownThreshold int           `json:"scale_down_threshold"` // 缩容阈值 (队列深度)
	ScaleUpStep        int           `json:"scale_up_step"`        // 扩容步长
	ScaleDownStep      int           `json:"scale_down_step"`      // 缩容步长
	ScaleInterval      time.Duration `json:"scale_interval"`       // 扩缩容检查间隔
}

// Config represents queue configuration
type Config struct {
	Type            string                 `json:"type"` // "memory", "redis", "rabbitmq", etc.
	Capacity        int                    `json:"capacity"`
	Workers         int                    `json:"workers"`
	RetryPolicy     RetryPolicy            `json:"retry_policy"`
	DeadLetterQueue bool                   `json:"dead_letter_queue"`
	PersistenceDir  string                 `json:"persistence_dir,omitempty"`
	RedisOptions    *RedisOptions          `json:"redis_options,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval"`
	Multiplier      float64       `json:"multiplier"`
	MaxElapsedTime  time.Duration `json:"max_elapsed_time"`
}

// RedisOptions contains Redis-specific configuration
type RedisOptions struct {
	Addr         string        `json:"addr"`
	Password     string        `json:"password,omitempty"`
	DB           int           `json:"db"`
	MaxRetries   int           `json:"max_retries"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	PoolSize     int           `json:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns"`
	MaxConnAge   time.Duration `json:"max_conn_age"`
	KeyPrefix    string        `json:"key_prefix"`
}

// Factory creates queue instances
type Factory interface {
	// CreateQueue creates a new queue instance
	CreateQueue(config *Config) (Queue, error)

	// GetSupportedTypes returns supported queue types
	GetSupportedTypes() []string
}

// Worker processes messages from a queue
type Worker interface {
	// Start begins processing messages
	Start(ctx context.Context) error

	// Stop stops processing messages
	Stop() error

	// Process handles a single message
	Process(ctx context.Context, msg *QueueMessage) error

	// GetID returns the worker ID
	GetID() string

	// IsRunning returns true if the worker is running
	IsRunning() bool
}

// WorkerPool manages multiple workers with auto-scaling
type WorkerPool interface {
	// Start starts all workers
	Start(ctx context.Context) error

	// Stop stops all workers
	Stop() error

	// Scale adjusts the number of workers
	Scale(ctx context.Context, count int) error

	// GetWorkerCount returns the number of workers
	GetWorkerCount() int

	// GetStats returns worker pool statistics
	GetStats() map[string]interface{}

	// GetMetrics returns detailed worker metrics
	GetMetrics() *WorkerMetrics

	// SetScalePolicy updates scaling policy
	SetScalePolicy(policy *ScalePolicy)

	// GetHealth returns health status
	GetHealth() *HealthStatus
}

// MessageHandler defines the interface for message processing
type MessageHandler func(ctx context.Context, msg *QueueMessage) error

// Middleware defines queue middleware
type Middleware func(next MessageHandler) MessageHandler

// Helper functions for message conversion

// NewQueueMessage creates a new queue message from a notification message
func NewQueueMessage(notifyMsg *message.Message) *QueueMessage {
	now := time.Now()

	// Copy metadata to avoid shared references
	metadata := make(map[string]interface{})
	for k, v := range notifyMsg.Metadata {
		metadata[k] = v
	}

	return &QueueMessage{
		ID:          idgen.GenerateMessageID(),
		Payload:     notifyMsg,
		Priority:    convertPriority(notifyMsg.Priority),
		Metadata:    metadata,
		Timestamp:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
		ScheduledAt: notifyMsg.ScheduledAt,
	}
}

// NewQueueMessageWithPayload creates a queue message with arbitrary payload
func NewQueueMessageWithPayload(payload interface{}, priority Priority) *QueueMessage {
	now := time.Now()
	return &QueueMessage{
		ID:        idgen.GenerateMessageID(),
		Payload:   payload,
		Priority:  priority,
		Metadata:  make(map[string]interface{}),
		Timestamp: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetNotificationMessage extracts notification message from queue message payload
func (qm *QueueMessage) GetNotificationMessage() (*message.Message, bool) {
	if notifyMsg, ok := qm.Payload.(*message.Message); ok {
		return notifyMsg, true
	}
	return nil, false
}

// convertPriority converts notification priority to queue priority
func convertPriority(notifyPriority message.Priority) Priority {
	switch notifyPriority {
	case message.PriorityLow:
		return PriorityLow
	case message.PriorityNormal:
		return PriorityNormal
	case message.PriorityHigh:
		return PriorityHigh
	case message.PriorityUrgent:
		return PriorityUrgent
	default:
		return PriorityNormal
	}
}
