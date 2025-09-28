// +build redis

// Package async provides Redis-based queue implementation for distributed scenarios
package async

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

// RedisAsyncQueue implements AsyncQueue using Redis Streams
type RedisAsyncQueue struct {
	client         *redis.Client
	streamKey      string
	consumerGroup  string
	consumerName   string
	logger         logger.Logger
	maxSize        int
	processedCount int64
	errorCount     int64
	closed         bool
}

// RedisQueueConfig represents Redis queue configuration
type RedisQueueConfig struct {
	Address       string
	Password      string
	DB            int
	StreamKey     string
	ConsumerGroup string
	ConsumerName  string
	MaxSize       int
}

// NewRedisAsyncQueue creates a new Redis-based async queue
func NewRedisAsyncQueue(config RedisQueueConfig, logger logger.Logger) (AsyncQueue, error) {
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	queue := &RedisAsyncQueue{
		client:        client,
		streamKey:     config.StreamKey,
		consumerGroup: config.ConsumerGroup,
		consumerName:  config.ConsumerName,
		logger:        logger,
		maxSize:       config.MaxSize,
	}

	// Create consumer group if it doesn't exist
	if err := queue.ensureConsumerGroup(ctx); err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return queue, nil
}

// ensureConsumerGroup creates the consumer group if it doesn't exist
func (q *RedisAsyncQueue) ensureConsumerGroup(ctx context.Context) error {
	// Try to create the consumer group
	err := q.client.XGroupCreateMkStream(ctx, q.streamKey, q.consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (q *RedisAsyncQueue) Enqueue(ctx context.Context, msg *message.Message, handle AsyncHandle) error {
	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	// Check stream length limit
	streamLen, err := q.client.XLen(ctx, q.streamKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check stream length: %w", err)
	}

	if streamLen >= int64(q.maxSize) {
		return fmt.Errorf("queue is full (size: %d)", q.maxSize)
	}

	// Serialize queue item
	item := &QueueItem{
		Message:  msg,
		Handle:   handle,
		QueuedAt: time.Now(),
		Priority: int(msg.Priority),
	}

	itemData, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to serialize queue item: %w", err)
	}

	// Add to Redis Stream with priority-based ID
	// Higher priority gets lower timestamp to appear first
	priorityScore := 1000 - item.Priority // Invert priority for Redis sorting
	streamID := fmt.Sprintf("%d-%d", time.Now().UnixMilli()-int64(priorityScore*1000), 0)

	fields := map[string]interface{}{
		"data":      string(itemData),
		"priority":  item.Priority,
		"queued_at": item.QueuedAt.Unix(),
	}

	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey,
		ID:     streamID,
		Values: fields,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to add message to stream: %w", err)
	}

	q.logger.Debug("Message enqueued to Redis",
		"message_id", msg.ID,
		"stream_key", q.streamKey,
		"priority", item.Priority)

	// Update handle status
	if handleImpl, ok := handle.(*HandleImpl); ok {
		handleImpl.UpdateStatus(StatusPending, 0.0, nil, nil)
	}

	return nil
}

func (q *RedisAsyncQueue) EnqueueBatch(ctx context.Context, msgs []*message.Message, batchHandle AsyncBatchHandle) error {
	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	// Check capacity
	streamLen, err := q.client.XLen(ctx, q.streamKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check stream length: %w", err)
	}

	if streamLen+int64(len(msgs)) > int64(q.maxSize) {
		return fmt.Errorf("batch would exceed queue capacity")
	}

	batchID := batchHandle.BatchID()

	// Get individual handles from batch handle
	var handles []AsyncHandle
	if batchImpl, ok := batchHandle.(*BatchHandleImpl); ok {
		handles = batchImpl.GetHandles()
	} else {
		return fmt.Errorf("invalid batch handle type")
	}

	// Use pipeline for batch operations
	pipe := q.client.Pipeline()

	for i, msg := range msgs {
		if i >= len(handles) {
			return fmt.Errorf("mismatch between messages and handles")
		}

		item := &QueueItem{
			Message:  msg,
			Handle:   handles[i],
			QueuedAt: time.Now(),
			Priority: int(msg.Priority),
			BatchInfo: &BatchInfo{
				BatchID:     batchID,
				BatchHandle: batchHandle,
				BatchIndex:  i,
				TotalItems:  len(msgs),
			},
		}

		itemData, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to serialize queue item %d: %w", i, err)
		}

		priorityScore := 1000 - item.Priority
		streamID := fmt.Sprintf("%d-%d", time.Now().UnixMilli()-int64(priorityScore*1000), i)

		fields := map[string]interface{}{
			"data":      string(itemData),
			"priority":  item.Priority,
			"queued_at": item.QueuedAt.Unix(),
			"batch_id":  batchID,
		}

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: q.streamKey,
			ID:     streamID,
			Values: fields,
		})
	}

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute batch enqueue: %w", err)
	}

	q.logger.Debug("Batch enqueued to Redis",
		"batch_id", batchID,
		"count", len(msgs),
		"stream_key", q.streamKey)

	return nil
}

func (q *RedisAsyncQueue) Dequeue(ctx context.Context) (*QueueItem, error) {
	if q.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	// Read from consumer group
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.consumerGroup,
		Consumer: q.consumerName,
		Streams:  []string{q.streamKey, ">"},
		Count:    1,
		Block:    time.Second, // Block for 1 second
	}).Result()

	if err != nil {
		if err == redis.Nil {
			// No messages available
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil // No messages
	}

	msg := streams[0].Messages[0]

	// Deserialize queue item
	dataStr, ok := msg.Values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message data format")
	}

	var item QueueItem
	if err := json.Unmarshal([]byte(dataStr), &item); err != nil {
		return nil, fmt.Errorf("failed to deserialize queue item: %w", err)
	}

	// Acknowledge the message
	if err := q.client.XAck(ctx, q.streamKey, q.consumerGroup, msg.ID).Err(); err != nil {
		q.logger.Warn("Failed to acknowledge message", "message_id", msg.ID, "error", err)
	}

	q.logger.Debug("Message dequeued from Redis",
		"message_id", item.Message.ID,
		"stream_id", msg.ID)

	// Update handle status
	if handleImpl, ok := item.Handle.(*HandleImpl); ok {
		handleImpl.UpdateStatus(StatusProcessing, 0.1, nil, nil)
	}

	return &item, nil
}

func (q *RedisAsyncQueue) Size() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	length, err := q.client.XLen(ctx, q.streamKey).Result()
	if err != nil {
		q.logger.Error("Failed to get queue size", "error", err)
		return 0
	}

	return int(length)
}

func (q *RedisAsyncQueue) IsEmpty() bool {
	return q.Size() == 0
}

func (q *RedisAsyncQueue) Close() error {
	if q.closed {
		return nil
	}

	q.closed = true
	q.logger.Info("Redis async queue closed", "stream_key", q.streamKey)

	return q.client.Close()
}

func (q *RedisAsyncQueue) Health() QueueHealth {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	healthy := true
	size := 0

	// Test Redis connection
	if err := q.client.Ping(ctx).Err(); err != nil {
		healthy = false
		q.logger.Error("Redis health check failed", "error", err)
	} else {
		// Get current stream length
		if length, err := q.client.XLen(ctx, q.streamKey).Result(); err == nil {
			size = int(length)
		}
	}

	return QueueHealth{
		Healthy:        healthy && !q.closed,
		Size:           size,
		MaxSize:        q.maxSize,
		ProcessedCount: q.processedCount,
		ErrorCount:     q.errorCount,
	}
}

// IncrementProcessed increments the processed count
func (q *RedisAsyncQueue) IncrementProcessed() {
	q.processedCount++
}

// IncrementError increments the error count
func (q *RedisAsyncQueue) IncrementError() {
	q.errorCount++
}

// CleanupProcessedMessages removes old processed messages from the stream
func (q *RedisAsyncQueue) CleanupProcessedMessages(ctx context.Context, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge).UnixMilli()
	cutoffID := fmt.Sprintf("%d-0", cutoff)

	count, err := q.client.XTrimApprox(ctx, q.streamKey, cutoffID, 0).Result()
	if err != nil {
		return fmt.Errorf("failed to cleanup processed messages: %w", err)
	}

	q.logger.Debug("Cleaned up processed messages",
		"stream_key", q.streamKey,
		"count", count,
		"cutoff", cutoffID)

	return nil
}

// GetPendingMessages returns the count of pending messages for this consumer
func (q *RedisAsyncQueue) GetPendingMessages(ctx context.Context) (int64, error) {
	pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   q.streamKey,
		Group:    q.consumerGroup,
		Start:    "-",
		End:      "+",
		Count:    1,
		Consumer: q.consumerName,
	}).Result()

	if err != nil {
		return 0, fmt.Errorf("failed to get pending messages: %w", err)
	}

	return int64(len(pending)), nil
}