// Package async provides queue management for asynchronous message processing
package async

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

// AsyncQueue represents a queue for asynchronous message processing
type AsyncQueue interface {
	// Core queue operations
	Enqueue(ctx context.Context, msg *message.Message, handle AsyncHandle) error
	EnqueueBatch(ctx context.Context, msgs []*message.Message, batchHandle AsyncBatchHandle) error
	Dequeue(ctx context.Context) (*QueueItem, error)

	// Queue status
	Size() int
	IsEmpty() bool
	Health() QueueHealth

	// Lifecycle management
	Close() error
}

// DelayedQueue extends AsyncQueue with scheduling capabilities
type DelayedQueue interface {
	AsyncQueue

	// Delayed operations
	EnqueueDelayed(ctx context.Context, msg *message.Message, handle AsyncHandle, scheduledAt time.Time) error
	EnqueueBatchDelayed(ctx context.Context, msgs []*message.Message, batchHandle AsyncBatchHandle, scheduledAt time.Time) error

	// Scheduling management
	GetScheduledCount() int
	GetNextScheduledTime() *time.Time
	CancelScheduledMessage(messageID string) bool
	GetScheduledMessages() []ScheduledMessageInfo
}

// PriorityQueue extends AsyncQueue with priority-based operations
type PriorityQueue interface {
	AsyncQueue

	// Priority operations
	EnqueueWithPriority(ctx context.Context, msg *message.Message, handle AsyncHandle, priority int) error
	GetHighestPriorityMessage(ctx context.Context) (*QueueItem, error)
	GetQueueByPriority() map[int]int // priority -> count
}

// AdvancedQueue combines all queue capabilities
type AdvancedQueue interface {
	DelayedQueue
	PriorityQueue

	// Batch operations for performance
	DequeueBatch(ctx context.Context, maxItems int) ([]*QueueItem, error)
	EnqueueBulk(ctx context.Context, items []*QueueItem) error

	// Monitoring and statistics
	GetStatistics() QueueStatistics
	GetThroughputMetrics() ThroughputMetrics
}

// QueueItem represents an item in the async queue
type QueueItem struct {
	Message   *message.Message
	Handle    AsyncHandle
	BatchInfo *BatchInfo
	QueuedAt  time.Time
	Priority  int
}

// BatchInfo contains batch-related information for queue items
type BatchInfo struct {
	BatchID     string
	BatchHandle AsyncBatchHandle
	BatchIndex  int
	TotalItems  int
}

// QueueHealth represents the health status of a queue
type QueueHealth struct {
	Healthy        bool  `json:"healthy"`
	Size           int   `json:"size"`
	MaxSize        int   `json:"max_size"`
	ProcessedCount int64 `json:"processed_count"`
	ErrorCount     int64 `json:"error_count"`
}

// QueueStatistics provides detailed queue performance metrics
type QueueStatistics struct {
	TotalEnqueued       int64   `json:"total_enqueued"`
	TotalDequeued       int64   `json:"total_dequeued"`
	TotalProcessed      int64   `json:"total_processed"`
	TotalErrors         int64   `json:"total_errors"`
	CurrentSize         int     `json:"current_size"`
	MaxSize             int     `json:"max_size"`
	ScheduledCount      int     `json:"scheduled_count"`
	AverageWaitTime     float64 `json:"average_wait_time_ms"`
	AverageProcessTime  float64 `json:"average_process_time_ms"`
	PriorityDistribution map[int]int `json:"priority_distribution"`
	LastProcessedAt     int64   `json:"last_processed_at"`
	Uptime              int64   `json:"uptime_seconds"`
}

// ThroughputMetrics provides throughput-related metrics
type ThroughputMetrics struct {
	MessagesPerSecond    float64 `json:"messages_per_second"`
	MessagesPerMinute    float64 `json:"messages_per_minute"`
	MessagesPerHour      float64 `json:"messages_per_hour"`
	ErrorRate            float64 `json:"error_rate"`
	SuccessRate          float64 `json:"success_rate"`
	P50ProcessTime       float64 `json:"p50_process_time_ms"`
	P95ProcessTime       float64 `json:"p95_process_time_ms"`
	P99ProcessTime       float64 `json:"p99_process_time_ms"`
}

// Note: memoryAsyncQueue implementation moved to memory_queue.go

// QueueFactory creates queues based on configuration
type QueueFactory struct {
	logger logger.Logger
}

// NewQueueFactory creates a new queue factory
func NewQueueFactory(logger logger.Logger) *QueueFactory {
	return &QueueFactory{logger: logger}
}

// CreateQueue creates a queue based on the given configuration
func (f *QueueFactory) CreateQueue(queueType string, config map[string]interface{}) (AsyncQueue, error) {
	switch queueType {
	case "memory":
		maxSize := 1000 // default
		if size, ok := config["max_size"].(int); ok {
			maxSize = size
		}

		baseQueue := NewMemoryAsyncQueue(maxSize, f.logger)

		// Check if delayed functionality is requested
		if enableDelayed, ok := config["enable_delayed"].(bool); ok && enableDelayed {
			return NewDelayedAsyncQueue(baseQueue, f.logger), nil
		}

		return baseQueue, nil

	case "redis":
		redisConfig := RedisQueueConfig{
			Address:       "localhost:6379",
			Password:      "",
			DB:            0,
			StreamKey:     "notifyhub:messages",
			ConsumerGroup: "notifyhub",
			ConsumerName:  "worker",
			MaxSize:       1000,
		}

		// Override with provided config
		if addr, ok := config["address"].(string); ok {
			redisConfig.Address = addr
		}
		if password, ok := config["password"].(string); ok {
			redisConfig.Password = password
		}
		if db, ok := config["db"].(int); ok {
			redisConfig.DB = db
		}
		if streamKey, ok := config["stream_key"].(string); ok {
			redisConfig.StreamKey = streamKey
		}
		if consumerGroup, ok := config["consumer_group"].(string); ok {
			redisConfig.ConsumerGroup = consumerGroup
		}
		if consumerName, ok := config["consumer_name"].(string); ok {
			redisConfig.ConsumerName = consumerName
		}
		if maxSize, ok := config["max_size"].(int); ok {
			redisConfig.MaxSize = maxSize
		}

		baseQueue, err := NewRedisAsyncQueue(redisConfig, f.logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create Redis queue: %w", err)
		}

		// Check if delayed functionality is requested
		if enableDelayed, ok := config["enable_delayed"].(bool); ok && enableDelayed {
			return NewDelayedAsyncQueue(baseQueue, f.logger), nil
		}

		return baseQueue, nil

	case "delayed":
		// Legacy support for explicit delayed queue type
		underlyingType := "memory"
		if underlying, ok := config["underlying"].(string); ok {
			underlyingType = underlying
		}

		// Create underlying queue
		underlyingConfig := make(map[string]interface{})
		for k, v := range config {
			if k != "underlying" && k != "enable_delayed" {
				underlyingConfig[k] = v
			}
		}

		underlying, err := f.CreateQueue(underlyingType, underlyingConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create underlying queue: %w", err)
		}

		return NewDelayedAsyncQueue(underlying, f.logger), nil

	default:
		return nil, fmt.Errorf("unknown queue type: %s", queueType)
	}
}
