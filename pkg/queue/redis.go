// Package queue provides Redis-based queue implementation
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// redisQueue implements a Redis-backed queue with priority and delay support
type redisQueue struct {
	client          *redis.Client
	keyPrefix       string
	mainQueue       string
	priorityQueue   string
	delayedQueue    string
	processingQueue string
	deadLetterQueue string
	capacity        int
	closed          int32
	stats           *QueueStats
	retryPolicy     RetryPolicy
	metrics         *MetricsCollector
	logger          logger.Logger
	stopCh          chan struct{}
}

// NewRedisQueue creates a new Redis-backed queue
func NewRedisQueue(opts *RedisOptions, capacity int, log logger.Logger) (Queue, error) {
	if log == nil {
		log = logger.Discard
	}

	if opts == nil {
		return nil, errors.New("redis options cannot be nil")
	}

	// Set defaults
	if opts.KeyPrefix == "" {
		opts.KeyPrefix = "notifyhub:queue:"
	}
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 5 * time.Second
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 3 * time.Second
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = 3 * time.Second
	}

	client := redis.NewClient(&redis.Options{
		Addr:         opts.Addr,
		Password:     opts.Password,
		DB:           opts.DB,
		MaxRetries:   opts.MaxRetries,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		PoolSize:     opts.PoolSize,
		MinIdleConns: opts.MinIdleConns,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Error("Failed to connect to Redis", "error", err)
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	q := &redisQueue{
		client:          client,
		keyPrefix:       opts.KeyPrefix,
		mainQueue:       opts.KeyPrefix + "main",
		priorityQueue:   opts.KeyPrefix + "priority",
		delayedQueue:    opts.KeyPrefix + "delayed",
		processingQueue: opts.KeyPrefix + "processing",
		deadLetterQueue: opts.KeyPrefix + "dlq",
		capacity:        capacity,
		stats: &QueueStats{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		retryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			MaxElapsedTime:  5 * time.Minute,
		},
		stopCh:  make(chan struct{}),
		logger:  log,
		metrics: NewMetricsCollector(),
	}

	// Start background processor for delayed messages
	go q.processDelayedMessages()

	log.Info("Redis queue created",
		"addr", opts.Addr,
		"keyPrefix", opts.KeyPrefix,
		"capacity", capacity)

	return q, nil
}

// Enqueue adds a message to the queue
func (q *redisQueue) Enqueue(ctx context.Context, msg *QueueMessage) error {
	if atomic.LoadInt32(&q.closed) == 1 {
		q.logger.Error("Attempted to enqueue to closed queue")
		return ErrQueueClosed
	}

	if msg == nil {
		q.logger.Error("Attempted to enqueue nil message")
		return ErrInvalidMessage
	}

	// Check capacity
	size, err := q.getSize(ctx)
	if err != nil {
		return fmt.Errorf("failed to get queue size: %w", err)
	}
	if q.capacity > 0 && size >= q.capacity {
		q.logger.Error("Queue is full", "capacity", q.capacity, "size", size)
		return ErrQueueFull
	}

	// Set timestamps if not set
	now := time.Now()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	msg.UpdatedAt = now
	msg.Timestamp = now

	// Serialize message
	data, err := json.Marshal(msg)
	if err != nil {
		q.logger.Error("Failed to serialize message", "error", err)
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Handle delayed messages
	if msg.ScheduledAt != nil && msg.ScheduledAt.After(now) {
		score := float64(msg.ScheduledAt.Unix())
		err = q.client.ZAdd(ctx, q.delayedQueue, redis.Z{
			Score:  score,
			Member: data,
		}).Err()
		if err != nil {
			q.logger.Error("Failed to add delayed message", "error", err)
			return fmt.Errorf("failed to add delayed message: %w", err)
		}
		atomic.AddInt64(&q.stats.EnqueuedCount, 1)
		q.logger.Debug("Message enqueued for delayed processing",
			"messageID", msg.ID,
			"scheduledAt", msg.ScheduledAt)
		return nil
	}

	// Handle priority messages
	if msg.Priority > PriorityNormal {
		err = q.client.LPush(ctx, q.priorityQueue, data).Err()
		if err != nil {
			q.logger.Error("Failed to add priority message", "error", err)
			return fmt.Errorf("failed to add priority message: %w", err)
		}
		atomic.AddInt64(&q.stats.EnqueuedCount, 1)
		q.logger.Debug("High priority message enqueued",
			"messageID", msg.ID,
			"priority", msg.Priority)
		return nil
	}

	// Regular enqueue
	err = q.client.LPush(ctx, q.mainQueue, data).Err()
	if err != nil {
		q.logger.Error("Failed to add message", "error", err)
		return fmt.Errorf("failed to add message: %w", err)
	}

	atomic.AddInt64(&q.stats.EnqueuedCount, 1)
	q.logger.Debug("Message enqueued", "messageID", msg.ID)
	return nil
}

// EnqueueBatch adds multiple messages to the queue
func (q *redisQueue) EnqueueBatch(ctx context.Context, msgs []*QueueMessage) error {
	if atomic.LoadInt32(&q.closed) == 1 {
		return ErrQueueClosed
	}

	q.logger.Debug("Enqueueing batch", "count", len(msgs))

	// Use pipeline for batch operations
	pipe := q.client.Pipeline()

	for _, msg := range msgs {
		if msg == nil {
			continue
		}

		// Set timestamps
		now := time.Now()
		if msg.CreatedAt.IsZero() {
			msg.CreatedAt = now
		}
		msg.UpdatedAt = now
		msg.Timestamp = now

		data, err := json.Marshal(msg)
		if err != nil {
			q.logger.Error("Failed to serialize message in batch", "error", err)
			continue
		}

		// Route to appropriate queue
		if msg.ScheduledAt != nil && msg.ScheduledAt.After(now) {
			score := float64(msg.ScheduledAt.Unix())
			pipe.ZAdd(ctx, q.delayedQueue, redis.Z{
				Score:  score,
				Member: data,
			})
		} else if msg.Priority > PriorityNormal {
			pipe.LPush(ctx, q.priorityQueue, data)
		} else {
			pipe.LPush(ctx, q.mainQueue, data)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		q.logger.Error("Failed to execute batch enqueue", "error", err)
		return fmt.Errorf("failed to execute batch enqueue: %w", err)
	}

	atomic.AddInt64(&q.stats.EnqueuedCount, int64(len(msgs)))
	q.logger.Debug("Batch enqueued successfully", "count", len(msgs))
	return nil
}

// Dequeue retrieves and removes a message from the queue
func (q *redisQueue) Dequeue(ctx context.Context) (*QueueMessage, error) {
	if atomic.LoadInt32(&q.closed) == 1 {
		return nil, ErrQueueClosed
	}

	// Try priority queue first
	data, err := q.client.RPop(ctx, q.priorityQueue).Result()
	if err == nil {
		msg, err := q.deserializeMessage([]byte(data))
		if err == nil {
			atomic.AddInt64(&q.stats.DequeuedCount, 1)
			q.logger.Debug("Priority message dequeued",
				"messageID", msg.ID,
				"priority", msg.Priority)
			return msg, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		q.logger.Error("Failed to dequeue from priority queue", "error", err)
		return nil, fmt.Errorf("failed to dequeue from priority queue: %w", err)
	}

	// Try main queue
	data, err = q.client.RPop(ctx, q.mainQueue).Result()
	if err == nil {
		msg, err := q.deserializeMessage([]byte(data))
		if err == nil {
			atomic.AddInt64(&q.stats.DequeuedCount, 1)
			q.logger.Debug("Message dequeued", "messageID", msg.ID)
			return msg, nil
		}
	} else if errors.Is(err, redis.Nil) {
		return nil, ErrQueueEmpty
	}

	q.logger.Error("Failed to dequeue message", "error", err)
	return nil, fmt.Errorf("failed to dequeue message: %w", err)
}

// DequeueBatch retrieves and removes multiple messages from the queue
func (q *redisQueue) DequeueBatch(ctx context.Context, count int) ([]*QueueMessage, error) {
	if atomic.LoadInt32(&q.closed) == 1 {
		return nil, ErrQueueClosed
	}

	q.logger.Debug("Dequeueing batch", "requestedCount", count)

	messages := make([]*QueueMessage, 0, count)

	// Try priority queue first
	for i := 0; i < count && len(messages) < count; i++ {
		data, err := q.client.RPop(ctx, q.priorityQueue).Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				q.logger.Error("Error dequeueing from priority queue", "error", err)
			}
			break
		}

		msg, err := q.deserializeMessage([]byte(data))
		if err == nil {
			messages = append(messages, msg)
		}
	}

	// Fill remaining from main queue
	remaining := count - len(messages)
	if remaining > 0 {
		// Use pipeline for efficiency
		pipe := q.client.Pipeline()
		for i := 0; i < remaining; i++ {
			pipe.RPop(ctx, q.mainQueue)
		}

		results, err := pipe.Exec(ctx)
		if err != nil && !errors.Is(err, redis.Nil) {
			q.logger.Error("Error executing batch dequeue", "error", err)
		}

		for _, result := range results {
			if result.Err() == nil {
				if strCmd, ok := result.(*redis.StringCmd); ok {
					data, _ := strCmd.Result()
					if data != "" {
						msg, err := q.deserializeMessage([]byte(data))
						if err == nil {
							messages = append(messages, msg)
						}
					}
				}
			}
		}
	}

	if len(messages) == 0 {
		return nil, ErrQueueEmpty
	}

	atomic.AddInt64(&q.stats.DequeuedCount, int64(len(messages)))
	q.logger.Debug("Batch dequeued", "count", len(messages))
	return messages, nil
}

// Peek retrieves a message without removing it from the queue
func (q *redisQueue) Peek(ctx context.Context) (*QueueMessage, error) {
	if atomic.LoadInt32(&q.closed) == 1 {
		return nil, ErrQueueClosed
	}

	// Try priority queue first
	data, err := q.client.LRange(ctx, q.priorityQueue, -1, -1).Result()
	if err == nil && len(data) > 0 {
		msg, err := q.deserializeMessage([]byte(data[0]))
		if err == nil {
			return msg, nil
		}
	}

	// Try main queue
	data, err = q.client.LRange(ctx, q.mainQueue, -1, -1).Result()
	if err == nil && len(data) > 0 {
		msg, err := q.deserializeMessage([]byte(data[0]))
		if err == nil {
			return msg, nil
		}
	}

	return nil, ErrQueueEmpty
}

// Size returns the number of messages in the queue
func (q *redisQueue) Size() int {
	ctx := context.Background()
	size, _ := q.getSize(ctx)
	return size
}

// IsEmpty returns true if the queue has no messages
func (q *redisQueue) IsEmpty() bool {
	return q.Size() == 0
}

// Clear removes all messages from the queue
func (q *redisQueue) Clear() error {
	ctx := context.Background()

	pipe := q.client.Pipeline()
	pipe.Del(ctx, q.mainQueue)
	pipe.Del(ctx, q.priorityQueue)
	pipe.Del(ctx, q.delayedQueue)
	pipe.Del(ctx, q.processingQueue)

	_, err := pipe.Exec(ctx)
	if err != nil {
		q.logger.Error("Failed to clear queue", "error", err)
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	q.logger.Info("Queue cleared")
	return nil
}

// Close closes the queue and releases resources
func (q *redisQueue) Close() error {
	if !atomic.CompareAndSwapInt32(&q.closed, 0, 1) {
		return nil // Already closed
	}

	q.logger.Info("Closing Redis queue")

	close(q.stopCh)

	// Close Redis client
	if err := q.client.Close(); err != nil {
		q.logger.Error("Error closing Redis client", "error", err)
		return fmt.Errorf("failed to close Redis client: %w", err)
	}

	q.logger.Info("Redis queue closed")
	return nil
}

// Helper methods

func (q *redisQueue) getSize(ctx context.Context) (int, error) {
	pipe := q.client.Pipeline()
	mainLen := pipe.LLen(ctx, q.mainQueue)
	priorityLen := pipe.LLen(ctx, q.priorityQueue)
	delayedLen := pipe.ZCard(ctx, q.delayedQueue)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	total := int(mainLen.Val() + priorityLen.Val() + delayedLen.Val())
	return total, nil
}

func (q *redisQueue) deserializeMessage(data []byte) (*QueueMessage, error) {
	var msg QueueMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to deserialize message: %w", err)
	}
	return &msg, nil
}

// processDelayedMessages processes delayed messages in the background
func (q *redisQueue) processDelayedMessages() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopCh:
			return
		case <-ticker.C:
			q.moveReadyDelayedMessages()
		}
	}
}

func (q *redisQueue) moveReadyDelayedMessages() {
	ctx := context.Background()
	now := time.Now().Unix()

	// Get messages that are ready
	results, err := q.client.ZRangeByScoreWithScores(ctx, q.delayedQueue, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", now),
	}).Result()

	if err != nil || len(results) == 0 {
		return
	}

	// Move ready messages to appropriate queues
	pipe := q.client.Pipeline()

	for _, z := range results {
		data := z.Member.(string)
		msg, err := q.deserializeMessage([]byte(data))
		if err != nil {
			continue
		}

		// Remove from delayed queue
		pipe.ZRem(ctx, q.delayedQueue, data)

		// Add to appropriate queue
		if msg.Priority > PriorityNormal {
			pipe.LPush(ctx, q.priorityQueue, data)
		} else {
			pipe.LPush(ctx, q.mainQueue, data)
		}

		q.logger.Debug("Delayed message ready for processing", "messageID", msg.ID)
	}

	_, _ = pipe.Exec(ctx)
}

// GetStats returns queue statistics (ObservableQueue interface)
func (q *redisQueue) GetStats() *QueueStats {
	ctx := context.Background()
	size, _ := q.getSize(ctx)

	return &QueueStats{
		Size:            size,
		EnqueuedCount:   q.stats.EnqueuedCount,
		DequeuedCount:   q.stats.DequeuedCount,
		RetryCount:      q.stats.RetryCount,
		DeadLetterCount: q.stats.DeadLetterCount,
		AverageWaitTime: q.stats.AverageWaitTime,
		CreatedAt:       q.stats.CreatedAt,
		UpdatedAt:       time.Now(),
	}
}

// GetMetrics returns detailed queue metrics (ObservableQueue interface)
func (q *redisQueue) GetMetrics() *QueueMetrics {
	return q.metrics.GetQueueMetrics()
}

// Subscribe registers a callback for queue events (ObservableQueue interface)
func (q *redisQueue) Subscribe(event string, callback func(msg *QueueMessage)) {
	q.metrics.Subscribe(event, callback)
}

// Unsubscribe removes a callback for queue events (ObservableQueue interface)
func (q *redisQueue) Unsubscribe(event string) {
	q.metrics.Unsubscribe(event)
}
