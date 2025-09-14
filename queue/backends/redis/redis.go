package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisQueueConfig contains Redis queue configuration
type RedisQueueConfig struct {
	// Redis connection options
	Addr     string `json:"addr" yaml:"addr"`         // Redis server address
	Password string `json:"password" yaml:"password"` // Redis password
	DB       int    `json:"db" yaml:"db"`             // Redis database number

	// Stream configuration
	StreamName      string        `json:"stream_name" yaml:"stream_name"`           // Redis stream name
	ConsumerGroup   string        `json:"consumer_group" yaml:"consumer_group"`     // Consumer group name
	ConsumerName    string        `json:"consumer_name" yaml:"consumer_name"`       // Consumer name
	MaxLen          int64         `json:"max_len" yaml:"max_len"`                   // Maximum stream length
	ClaimMinIdle    time.Duration `json:"claim_min_idle" yaml:"claim_min_idle"`     // Minimum idle time before claiming
	ProcessingLimit int           `json:"processing_limit" yaml:"processing_limit"` // Max messages to process at once
}

// DefaultRedisQueueConfig returns default Redis queue configuration
func DefaultRedisQueueConfig() *RedisQueueConfig {
	return &RedisQueueConfig{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		StreamName:      "notifyhub:messages",
		ConsumerGroup:   "notifyhub-workers",
		ConsumerName:    "worker-1",
		MaxLen:          10000,
		ClaimMinIdle:    5 * time.Minute,
		ProcessingLimit: 10,
	}
}

// RedisQueue implements Queue interface using Redis Streams
type RedisQueue struct {
	client     *redis.Client
	config     *RedisQueueConfig
	ctx        context.Context
	cancel     context.CancelFunc
	closed     bool
}

// NewRedisQueue creates a new Redis queue
func NewRedisQueue(config *RedisQueueConfig) (*RedisQueue, error) {
	if config == nil {
		config = DefaultRedisQueueConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	q := &RedisQueue{
		client: client,
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize stream and consumer group
	if err := q.initializeStream(); err != nil {
		cancel()
		client.Close()
		return nil, fmt.Errorf("initialize stream failed: %v", err)
	}

	return q, nil
}

// initializeStream creates the stream and consumer group if they don't exist
func (r *RedisQueue) initializeStream() error {
	// Create consumer group (will fail if it already exists, which is fine)
	err := r.client.XGroupCreateMkStream(r.ctx, r.config.StreamName, r.config.ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %v", err)
	}
	return nil
}

// Enqueue adds a message to the Redis stream
func (r *RedisQueue) Enqueue(ctx context.Context, msg *Message) (string, error) {
	if r.closed {
		return "", fmt.Errorf("queue closed")
	}

	// Generate ID if not set
	if msg.ID == "" {
		msg.ID = generateID()
	}

	// Set creation time if not set
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// Serialize message
	data, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("serialize message: %v", err)
	}

	// Add to stream
	args := &redis.XAddArgs{
		Stream: r.config.StreamName,
		MaxLen: r.config.MaxLen,
		Approx: true,
		Values: map[string]interface{}{
			"id":         msg.ID,
			"data":       string(data),
			"created_at": msg.CreatedAt.Unix(),
		},
	}

	streamID, err := r.client.XAdd(ctx, args).Result()
	if err != nil {
		return "", fmt.Errorf("add to stream: %v", err)
	}

	return streamID, nil
}

// Dequeue retrieves a message from the Redis stream
func (r *RedisQueue) Dequeue(ctx context.Context) (*Message, error) {
	if r.closed {
		return nil, fmt.Errorf("queue closed")
	}

	// First, try to claim any pending messages that have been idle too long
	r.claimIdleMessages()

	// Read new messages from the stream
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    r.config.ConsumerGroup,
		Consumer: r.config.ConsumerName,
		Streams:  []string{r.config.StreamName, ">"},
		Count:    1,
		Block:    time.Second, // Block for 1 second
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, context.DeadlineExceeded // Timeout
		}
		return nil, fmt.Errorf("read from stream: %v", err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, context.DeadlineExceeded // No messages
	}

	// Parse the message
	streamMsg := streams[0].Messages[0]
	return r.parseStreamMessage(streamMsg)
}

// parseStreamMessage converts Redis stream message to our Message format
func (r *RedisQueue) parseStreamMessage(streamMsg redis.XMessage) (*Message, error) {
	dataStr, ok := streamMsg.Values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message format: missing data field")
	}

	var msg Message
	if err := json.Unmarshal([]byte(dataStr), &msg); err != nil {
		return nil, fmt.Errorf("deserialize message: %v", err)
	}

	// Store the stream ID for acknowledgment
	msg.StreamID = streamMsg.ID

	return &msg, nil
}

// Ack acknowledges successful processing of a message
func (r *RedisQueue) Ack(msgID string) error {
	if r.closed {
		return fmt.Errorf("queue closed")
	}

	// For Redis streams, we need the stream ID, not the message ID
	// We assume msgID contains the stream ID (stored in Message.StreamID)
	return r.client.XAck(r.ctx, r.config.StreamName, r.config.ConsumerGroup, msgID).Err()
}

// Nack indicates failed processing and schedules retry
func (r *RedisQueue) Nack(msgID string, nextRetry time.Time) error {
	if r.closed {
		return fmt.Errorf("queue closed")
	}

	// For Redis streams, we don't explicitly NACK
	// Failed messages will be reclaimed by claimIdleMessages()
	// We could optionally add the message back to stream for retry
	return nil
}

// claimIdleMessages claims messages that have been pending too long
func (r *RedisQueue) claimIdleMessages() {
	pending, err := r.client.XPendingExt(r.ctx, &redis.XPendingExtArgs{
		Stream: r.config.StreamName,
		Group:  r.config.ConsumerGroup,
		Start:  "-",
		End:    "+",
		Count:  r.config.ProcessingLimit,
	}).Result()

	if err != nil || len(pending) == 0 {
		return
	}

	var idsToProcess []string
	for _, p := range pending {
		if p.Idle >= r.config.ClaimMinIdle {
			idsToProcess = append(idsToProcess, p.ID)
		}
	}

	if len(idsToProcess) > 0 {
		r.client.XClaim(r.ctx, &redis.XClaimArgs{
			Stream:   r.config.StreamName,
			Group:    r.config.ConsumerGroup,
			Consumer: r.config.ConsumerName,
			Messages: idsToProcess,
			MinIdle:  r.config.ClaimMinIdle,
		})
	}
}

// Close closes the Redis queue
func (r *RedisQueue) Close() error {
	if r.closed {
		return nil
	}

	r.closed = true
	r.cancel()

	return r.client.Close()
}

// Size returns the current stream length
func (r *RedisQueue) Size() int {
	length, err := r.client.XLen(r.ctx, r.config.StreamName).Result()
	if err != nil {
		return 0
	}
	return int(length)
}

// Health checks Redis connection health
func (r *RedisQueue) Health(ctx context.Context) error {
	if r.closed {
		return fmt.Errorf("queue is closed")
	}

	return r.client.Ping(ctx).Err()
}

// generateID generates a unique ID for messages
func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// Update Message struct to include StreamID for Redis
// This should be added to the Message struct in queue.go