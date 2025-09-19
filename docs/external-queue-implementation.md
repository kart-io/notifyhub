# 外部队列实现设计方案

## 设计理念

通过**定义最小化接口，完全由外部实现**的方式，解决Redis队列配置中的连接冲突、版本依赖和资源浪费问题。

## 核心优势

### 1. **零内部依赖**
- NotifyHub不依赖任何特定的队列实现
- 完全移除内部Redis客户端依赖
- 用户可以选择任意Redis客户端版本

### 2. **完全可扩展**
- 支持任意类型的队列实现（Redis、RabbitMQ、Kafka、内存队列等）
- 接口设计简洁，易于实现
- 支持注册表模式，运行时动态注册队列类型

### 3. **连接复用**
- 外部管理连接池和生命周期
- 避免重复创建连接
- 支持连接池共享

### 4. **测试友好**
- 接口可轻松Mock
- 支持内存队列用于测试
- 配置验证内置

## 接口设计

### 核心接口

```go
// ExternalQueue 外部队列接口 - 完全由外部实现
type ExternalQueue interface {
    Enqueue(ctx context.Context, message *ExternalMessage) (string, error)
    Dequeue(ctx context.Context) (*ExternalMessage, error)
    Ack(messageID string) error
    Nack(messageID string, retryAfter time.Duration) error
    Size() int
    Health(ctx context.Context) error
    Close() error
}

// ExternalQueueFactory 外部队列工厂接口
type ExternalQueueFactory interface {
    Name() string
    Create(config map[string]interface{}) (ExternalQueue, error)
    Validate(config map[string]interface{}) error
}
```

### 消息结构

```go
// ExternalMessage 外部队列消息结构
type ExternalMessage struct {
    ID         string                 `json:"id"`
    Data       []byte                 `json:"data"`
    Headers    map[string]string      `json:"headers"`
    CreatedAt  time.Time              `json:"created_at"`
    RetryCount int                    `json:"retry_count"`
    NextRetry  time.Time              `json:"next_retry,omitempty"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
```

## 使用方式

### 方式1：直接使用队列实例（最灵活）

```go
// 创建外部Redis队列实例
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

queueInstance := &MyRedisQueue{
    client:     redisClient,
    streamName: "notifications",
}

// 注入到NotifyHub
hub, err := client.New(
    config.WithExternalQueue(queueInstance),
)
```

### 方式2：使用队列工厂（推荐）

```go
// 创建工厂
factory := &RedisQueueFactory{}

// 配置参数
config := map[string]interface{}{
    "addr":        "localhost:6379",
    "password":    "mypassword",
    "db":          0,
    "stream_name": "notifications",
    "buffer_size": 2000,
    "workers":     8,
}

// 使用工厂创建队列
hub, err := client.New(
    config.WithExternalQueueFactory(factory, config),
)
```

### 方式3：全局注册表（生产环境推荐）

```go
// 注册队列工厂
factory := &RedisQueueFactory{}
queue.RegisterExternal(factory)

// 使用注册的工厂
config := map[string]interface{}{
    "addr":        "localhost:6379",
    "stream_name": "prod-notifications",
    "buffer_size": 5000,
    "workers":     16,
}

hub, err := client.New(
    config.WithExternalQueueConfig("redis", config),
)
```

## 实现示例

### Redis队列实现

```go
// RedisQueueFactory Redis队列工厂实现
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
    addr := config["addr"].(string)
    password, _ := config["password"].(string)
    db, _ := config["db"].(int)

    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
    })

    return &RedisQueue{client: client}, nil
}

// RedisQueue Redis队列实现
type RedisQueue struct {
    client *redis.Client
}

func (r *RedisQueue) Enqueue(ctx context.Context, message *queue.ExternalMessage) (string, error) {
    data, _ := json.Marshal(message)
    return r.client.XAdd(ctx, &redis.XAddArgs{
        Stream: "notifications",
        Values: map[string]interface{}{"data": string(data)},
    }).Result()
}

func (r *RedisQueue) Dequeue(ctx context.Context) (*queue.ExternalMessage, error) {
    // Redis Streams实现
    streams, err := r.client.XRead(ctx, &redis.XReadArgs{
        Streams: []string{"notifications", "$"},
        Count:   1,
        Block:   time.Second,
    }).Result()

    if err != nil || len(streams) == 0 {
        return nil, context.DeadlineExceeded
    }

    rawData := streams[0].Messages[0].Values["data"].(string)
    var message queue.ExternalMessage
    json.Unmarshal([]byte(rawData), &message)

    return &message, nil
}

// ... 其他方法实现
```

### 内存队列实现

```go
type MemoryQueue struct {
    messages chan *queue.ExternalMessage
    closed   bool
}

func (m *MemoryQueue) Enqueue(ctx context.Context, message *queue.ExternalMessage) (string, error) {
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
    select {
    case msg := <-m.messages:
        return msg, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

## 配置选项

### 基础配置选项

```go
// 直接使用队列实例
config.WithExternalQueue(queueInstance)

// 使用队列工厂
config.WithExternalQueueFactory(factory, config)

// 使用配置字典（需要先注册工厂）
config.WithExternalQueueConfig("redis", configMap)
```

### 配置参数示例

```go
redisConfig := map[string]interface{}{
    // 连接配置
    "addr":        "localhost:6379",
    "password":    "mypassword",
    "db":          0,

    // Redis Streams配置
    "stream_name":      "notifications",
    "consumer_group":   "workers",
    "consumer_name":    "worker-1",
    "max_len":          10000,
    "claim_min_idle":   "5m",
    "processing_limit": 10,

    // 队列基础配置
    "buffer_size": 2000,
    "workers":     8,
}
```

## 迁移指南

### 从内部Redis实现迁移

**之前的问题方式：**
```go
// 内部管理Redis连接 - 容易冲突
hub, err := client.New(config.WithRedisQueue("localhost:6379", "", 0, 1000, 4))
```

**新的外部实现：**
```go
// 外部管理Redis连接 - 完全控制
redisClient := getSharedRedisClient() // 复用应用的Redis连接

queueInstance := &MyRedisQueue{
    client: redisClient,
    config: MyRedisConfig{},
}

hub, err := client.New(config.WithExternalQueue(queueInstance))
```

### 生产环境最佳实践

```go
// 1. 注册所有队列类型
queue.RegisterExternal(&RedisQueueFactory{})
queue.RegisterExternal(&RabbitMQQueueFactory{})

// 2. 从配置文件读取队列类型
queueType := os.Getenv("QUEUE_TYPE") // "redis" 或 "rabbitmq"

// 3. 动态创建队列
queueConfig := loadQueueConfig(queueType)
hub, err := client.New(
    config.WithExternalQueueConfig(queueType, queueConfig),
)

// 4. 优雅关闭
defer hub.Close()
```

## 架构对比

### 原有架构问题

```
NotifyHub → 内部Redis客户端 → Redis服务器
             ↑ 内部管理连接
             ↑ 版本绑定
             ↑ 配置复杂
```

### 新架构优势

```
NotifyHub → ExternalQueue接口 ← 外部Redis队列实现 → Redis服务器
             ↑ 接口抽象              ↑ 用户管理连接
             ↑ 零依赖                ↑ 版本自由
             ↑ 配置简洁              ↑ 连接复用
```

## 扩展性示例

### 支持新的队列类型

```go
// 实现Kafka队列
type KafkaQueueFactory struct{}

func (f *KafkaQueueFactory) Name() string {
    return "kafka"
}

func (f *KafkaQueueFactory) Create(config map[string]interface{}) (queue.ExternalQueue, error) {
    // Kafka队列实现
    return &KafkaQueue{}, nil
}

// 注册Kafka队列
queue.RegisterExternal(&KafkaQueueFactory{})

// 使用Kafka队列
hub, err := client.New(
    config.WithExternalQueueConfig("kafka", kafkaConfig),
)
```

### 支持多队列类型

```go
// 根据优先级选择队列
switch priority {
case "high":
    hub.UseQueue(config.WithExternalQueueConfig("redis", fastConfig))
case "normal":
    hub.UseQueue(config.WithExternalQueueConfig("rabbitmq", normalConfig))
case "low":
    hub.UseQueue(config.WithExternalQueueConfig("kafka", batchConfig))
}
```

## 总结

这种**外部实现接口方案**完全解决了原有问题：

1. ✅ **零连接冲突** - 外部管理所有连接
2. ✅ **无版本依赖** - 支持任意Redis客户端版本
3. ✅ **完全可扩展** - 支持任意队列类型
4. ✅ **资源高效** - 连接池复用，避免浪费
5. ✅ **测试友好** - 接口Mock，内存队列支持
6. ✅ **生产就绪** - 注册表管理，配置验证

这是一个真正**面向接口编程**的解决方案，体现了依赖倒置原则和接口隔离原则的最佳实践。