package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/api"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/queue"
	"github.com/redis/go-redis/v9"
)

// ===========================
// 外部Redis队列实现示例
// ===========================

// RedisQueueFactory 外部实现的Redis队列工厂
type RedisQueueFactory struct{}

func (f *RedisQueueFactory) Name() string {
	return "redis"
}

func (f *RedisQueueFactory) Validate(config map[string]interface{}) error {
	if config["addr"] == nil || config["addr"].(string) == "" {
		return fmt.Errorf("redis addr is required")
	}
	return nil
}

func (f *RedisQueueFactory) Create(config map[string]interface{}) (queue.ExternalQueue, error) {
	addr, _ := config["addr"].(string)
	password, _ := config["password"].(string)
	db, _ := config["db"].(int)
	streamName, _ := config["stream_name"].(string)
	if streamName == "" {
		streamName = "notifyhub:messages"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis connection failed: %v", err)
	}

	return &RedisQueue{
		client:     client,
		streamName: streamName,
	}, nil
}

// RedisQueue 外部实现的Redis队列
type RedisQueue struct {
	client     *redis.Client
	streamName string
	closed     bool
}

func (r *RedisQueue) Enqueue(ctx context.Context, message *queue.ExternalMessage) (string, error) {
	if r.closed {
		return "", fmt.Errorf("queue is closed")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return "", err
	}

	args := &redis.XAddArgs{
		Stream: r.streamName,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}

	return r.client.XAdd(ctx, args).Result()
}

func (r *RedisQueue) Dequeue(ctx context.Context) (*queue.ExternalMessage, error) {
	if r.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	// 简化实现：直接读取最新消息
	streams, err := r.client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{r.streamName, "$"},
		Count:   1,
		Block:   time.Second,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, context.DeadlineExceeded
		}
		return nil, err
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, context.DeadlineExceeded
	}

	rawData := streams[0].Messages[0].Values["data"].(string)
	var message queue.ExternalMessage
	if err := json.Unmarshal([]byte(rawData), &message); err != nil {
		return nil, err
	}

	// 存储Redis特定的消息ID
	message.Metadata = map[string]interface{}{
		"redis_id": streams[0].Messages[0].ID,
	}

	return &message, nil
}

func (r *RedisQueue) Ack(messageID string) error {
	if r.closed {
		return fmt.Errorf("queue is closed")
	}
	// 简化实现：在实际应用中这里会删除消息
	return nil
}

func (r *RedisQueue) Nack(messageID string, retryAfter time.Duration) error {
	if r.closed {
		return fmt.Errorf("queue is closed")
	}
	// 简化实现：在实际应用中这里会重新排队
	return nil
}

func (r *RedisQueue) Size() int {
	if r.closed {
		return 0
	}
	length, err := r.client.XLen(context.Background(), r.streamName).Result()
	if err != nil {
		return 0
	}
	return int(length)
}

func (r *RedisQueue) Health(ctx context.Context) error {
	if r.closed {
		return fmt.Errorf("queue is closed")
	}
	return r.client.Ping(ctx).Err()
}

func (r *RedisQueue) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	return r.client.Close()
}

// ===========================
// 使用示例
// ===========================

func main() {
	fmt.Println("=== 外部Redis队列实现示例 ===")

	// 方式1：使用Redis队列
	fmt.Println("\n1. 使用Redis队列")
	factory := &RedisQueueFactory{}

	cfg1 := config.New(
		config.WithQueue("redis", 2000, 8),
	)
	_, err := api.New(cfg1)
	if err != nil {
		log.Printf("创建Redis配置失败: %v", err)
	} else {
		fmt.Printf("✓ Redis配置创建成功，队列类型: %s\n", factory.Name())
	}

	// 方式2：使用不同的Redis配置
	fmt.Println("\n2. 使用不同的Redis配置")
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer func() { _ = redisClient.Close() }()

	cfg2 := config.New(
		config.WithQueue("redis", 1000, 4),
	)
	_, err = api.New(cfg2)
	if err != nil {
		log.Printf("创建Redis配置失败: %v", err)
	} else {
		fmt.Printf("✓ Redis配置创建成功\n")
	}

	// 方式3：使用高容量Redis配置
	fmt.Println("\n3. 使用高容量Redis配置")

	// 注册工厂（演示目的）
	if err := queue.RegisterExternal(factory); err != nil {
		log.Printf("注册工厂失败: %v", err)
	} else {
		fmt.Printf("✓ Redis工厂注册成功\n")

		cfg3 := config.New(
			config.WithQueue("redis", 3000, 12),
		)
		_, err := api.New(cfg3)
		if err != nil {
			log.Printf("创建高容量配置失败: %v", err)
		} else {
			fmt.Printf("✓ 高容量配置创建成功\n")
		}
	}

	// 显示注册表信息
	fmt.Printf("\n注册的队列类型: %v\n", queue.ListExternal())

	// 方式4：内存队列示例
	fmt.Println("\n4. 内存队列示例")

	cfg4 := config.New(
		config.WithQueue("memory", 100, 2),
	)
	_, err = api.New(cfg4)
	if err != nil {
		log.Printf("创建内存队列失败: %v", err)
	} else {
		fmt.Printf("✓ 内存队列配置成功\n")
	}

	fmt.Println("\n=== 外部实现的优势 ===")
	fmt.Println("1. 零内部依赖 - NotifyHub不依赖任何特定队列实现")
	fmt.Println("2. 完全可扩展 - 可以实现任意类型的队列")
	fmt.Println("3. 连接复用 - 外部管理连接池和生命周期")
	fmt.Println("4. 版本无关 - 支持任意Redis客户端版本")
	fmt.Println("5. 测试友好 - 可以注入Mock队列")

	fmt.Println("\n=== 示例完成 ===")
}

// ===========================
// 简单内存队列实现示例
// ===========================

type MemoryQueue struct {
	messages chan *queue.ExternalMessage
	closed   bool
}

func (m *MemoryQueue) Enqueue(ctx context.Context, message *queue.ExternalMessage) (string, error) {
	if m.closed {
		return "", fmt.Errorf("queue is closed")
	}

	select {
	case m.messages <- message:
		return message.ID, nil
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return "", fmt.Errorf("queue is full")
	}
}

func (m *MemoryQueue) Dequeue(ctx context.Context) (*queue.ExternalMessage, error) {
	if m.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	select {
	case msg := <-m.messages:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *MemoryQueue) Ack(messageID string) error {
	return nil // 内存队列不需要确认
}

func (m *MemoryQueue) Nack(messageID string, retryAfter time.Duration) error {
	return nil // 简化实现
}

func (m *MemoryQueue) Size() int {
	return len(m.messages)
}

func (m *MemoryQueue) Health(ctx context.Context) error {
	if m.closed {
		return fmt.Errorf("queue is closed")
	}
	return nil
}

func (m *MemoryQueue) Close() error {
	if !m.closed {
		m.closed = true
		close(m.messages)
	}
	return nil
}
