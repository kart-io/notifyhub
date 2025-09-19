package queue

import (
	"context"
	"time"
)

// ExternalQueue 外部队列接口 - 完全由外部实现
type ExternalQueue interface {
	// Enqueue 入队消息，返回消息ID
	Enqueue(ctx context.Context, message *ExternalMessage) (string, error)

	// Dequeue 出队消息，如果没有消息则阻塞直到超时
	Dequeue(ctx context.Context) (*ExternalMessage, error)

	// Ack 确认消息处理成功
	Ack(messageID string) error

	// Nack 标记消息处理失败，安排重试
	Nack(messageID string, retryAfter time.Duration) error

	// Size 获取队列当前大小
	Size() int

	// Health 健康检查
	Health(ctx context.Context) error

	// Close 关闭队列连接
	Close() error
}

// ExternalQueueFactory 外部队列工厂接口 - 由外部实现具体队列创建逻辑
type ExternalQueueFactory interface {
	// Name 返回队列类型名称 (如: "redis", "memory", "rabbitmq")
	Name() string

	// Create 根据配置创建队列实例
	Create(config map[string]interface{}) (ExternalQueue, error)

	// Validate 验证配置是否有效
	Validate(config map[string]interface{}) error
}

// ExternalMessage 外部队列消息结构
type ExternalMessage struct {
	ID        string                 `json:"id"`
	Data      []byte                 `json:"data"`
	Headers   map[string]string      `json:"headers"`
	CreatedAt time.Time              `json:"created_at"`
	RetryCount int                   `json:"retry_count"`
	NextRetry  time.Time              `json:"next_retry,omitempty"`

	// 用于存储队列特定的元数据（如Redis的StreamID）
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}