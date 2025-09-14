# Queue Package

## 功能概述

Queue包实现了NotifyHub的消息队列系统，提供灵活的消息缓冲、重试机制、回调系统和多种后端支持。采用模块化架构设计，各功能模块独立开发和维护。

## 📁 包结构

```
queue/
├── README.md                    # 总体文档
├── queue.go                     # 公共API和类型导出
├── core/                        # 核心队列实现
│   ├── interface.go             # Queue接口定义
│   ├── simple.go                # 内存队列实现
│   ├── message.go               # Message结构体
│   └── README.md                # 核心实现文档
├── backends/                    # 不同后端实现
│   ├── redis/                   # Redis队列实现
│   │   ├── redis.go
│   │   └── README.md
│   └── README.md
├── callbacks/                   # 回调系统
│   ├── callback.go             # 回调接口和实现
│   └── README.md
├── worker/                      # 消息处理
│   ├── worker.go               # 工作者实现
│   └── README.md
├── retry/                       # 重试机制
│   ├── policy.go               # 重试策略
│   └── README.md
├── scheduler/                   # 消息调度
│   ├── scheduler.go            # 调度器实现
│   └── README.md
└── tests/                       # 测试文件
    └── queue_test.go
```

## 核心接口

### Queue接口
```go
type Queue interface {
    Enqueue(ctx context.Context, msg *Message) (string, error)
    Dequeue(ctx context.Context) (*Message, error)
    Ack(msgID string) error
    Nack(msgID string, nextRetry time.Time) error
    Close() error
    Size() int
    Health(ctx context.Context) error
}
```

## 模块功能

### 🔧 核心模块 (`core/`)
- **SimpleQueue**: 高性能内存队列实现
- **Queue Interface**: 统一队列接口定义
- **Message**: 消息结构体定义

### 🗄️ 后端实现 (`backends/`)
- **Redis**: Redis Streams持久化队列
- 支持扩展其他后端（PostgreSQL、RabbitMQ等）

### 📞 回调系统 (`callbacks/`)
- 函数回调和HTTP Webhook支持
- 成功/失败/重试/超限回调事件
- HMAC签名验证和超时控制

### 👷 消息处理 (`worker/`)
- 并发消息处理工作者
- 自动重试和错误处理
- 优雅关闭支持

### 🔄 重试机制 (`retry/`)
- 多种重试策略（指数退避、线性退避）
- 抖动支持防止雷群效应
- 可配置重试次数和间隔

### ⏰ 消息调度 (`scheduler/`)
- 延迟消息精确调度
- Min-Heap优先级队列
- 高效时间管理

## 使用示例

### 内存队列

```go
// 创建内存队列
queue := queue.NewSimple(1000) // 缓冲区大小1000

// 入队消息
message := &queue.Message{
    Message: notifierMessage,
    Options: &client.Options{
        Timeout: 30 * time.Second,
        Retry:   true,
    },
}

taskID, err := queue.Enqueue(ctx, message)
if err != nil {
    log.Printf("入队失败: %v", err)
}

// 出队处理
msg, err := queue.Dequeue(ctx)
if err != nil {
    log.Printf("出队失败: %v", err)
}

// 确认处理
err = queue.Ack(msg.ID)
```

### Redis队列

```go
// Redis配置
config := &queue.RedisQueueConfig{
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

// 创建Redis队列
redisQueue, err := queue.NewRedisQueue(config)
if err != nil {
    log.Fatal(err)
}
defer redisQueue.Close()

// 使用方式与内存队列相同
taskID, err := redisQueue.Enqueue(ctx, message)
```

### 延迟消息调度

```go
// 创建延迟消息
message := &queue.Message{
    Message: &notifiers.Message{
        Title: "延迟通知",
        Body:  "这是一个延迟5分钟的消息",
        Delay: 5 * time.Minute, // 延迟5分钟
    },
}

// 调度器会自动处理延迟
taskID, err := queue.Enqueue(ctx, message)
```

## Worker系统

### Worker实现
```go
type Worker struct {
    queue       Queue
    sender      MessageSender  // 通常是Hub实例
    retryPolicy *RetryPolicy
    workers     int
}

func NewWorker(queue Queue, sender MessageSender, retryPolicy *RetryPolicy, workers int) *Worker
```

### 启动Worker

```go
// 创建Worker
worker := queue.NewWorker(
    queue,        // 队列实例
    hub,         // Hub作为MessageSender
    retryPolicy, // 重试策略
    4,           // worker数量
)

// 启动处理
err := worker.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// 优雅停止
worker.Stop()
```

## 重试策略

### RetryPolicy结构
```go
type RetryPolicy struct {
    MaxRetries      int
    InitialInterval time.Duration
    Multiplier      float64
    MaxJitter       time.Duration  // 防雷鸣群
}
```

### 内置策略

```go
// 默认策略（指数退避+jitter）
defaultPolicy := queue.DefaultRetryPolicy()

// 无重试策略
noRetryPolicy := queue.NoRetryPolicy()

// 激进重试（快速重试）
aggressivePolicy := queue.AggressiveRetryPolicy()

// 线性退避
linearPolicy := queue.LinearBackoffPolicy(5, 30*time.Second)

// 自定义策略
customPolicy := &queue.RetryPolicy{
    MaxRetries:      3,
    InitialInterval: 10 * time.Second,
    Multiplier:      2.0,
    MaxJitter:       2 * time.Second,
}
```

### 重试计算

```go
// 计算下次重试时间
nextInterval := policy.CalculateInterval(attempts)
nextRetry := time.Now().Add(nextInterval)

// Nack消息进行重试调度
err := queue.Nack(msgID, nextRetry)
```

## 延迟调度系统

### Min-Heap调度器

MessageScheduler使用最小堆实现精确的时间调度：

```go
type MessageScheduler struct {
    heap     DelayedMessageHeap  // 最小堆
    mu       sync.RWMutex
    queue    Queue
    stopCh   chan struct{}
    ticker   *time.Ticker
}
```

### 调度流程

1. **消息入队**: 延迟消息加入Min-Heap
2. **时间检查**: 定期检查堆顶消息是否到期
3. **消息发送**: 到期消息转入普通队列处理
4. **堆维护**: 自动维护堆结构确保时间顺序

```go
// 创建调度器
scheduler := queue.NewMessageScheduler(queue)

// 调度延迟消息
err := scheduler.ScheduleMessage(&queue.Message{
    Message: msg,
    NextRetry: time.Now().Add(5 * time.Minute),
})

// 停止调度器
scheduler.Stop()
```

## 分布式特性（Redis）

### 消费者组

Redis队列支持消费者组模式，实现分布式处理：

```go
// 多个worker实例可以共享同一个消费者组
config1 := &queue.RedisQueueConfig{
    ConsumerGroup: "notifyhub-workers",
    ConsumerName:  "worker-1",
}

config2 := &queue.RedisQueueConfig{
    ConsumerGroup: "notifyhub-workers", // 相同组名
    ConsumerName:  "worker-2",          // 不同消费者名
}
```

### 故障恢复

- **消息重领**: 自动重领长时间未确认的消息
- **失败重试**: 支持消息级别的重试策略
- **死信处理**: 超过重试次数的消息可进入死信队列

### 监控指标

```go
// 获取队列大小
size := queue.Size()

// 健康检查
err := queue.Health(ctx)

// Redis特有指标
pendingCount := redisQueue.GetPendingCount()
consumerInfo := redisQueue.GetConsumerInfo()
```

## 最佳实践

### 1. 队列选择

```go
// 开发/测试环境
queue := queue.NewSimple(1000)

// 生产环境
redisConfig := queue.DefaultRedisQueueConfig()
redisConfig.MaxLen = 50000  // 根据业务量调整
queue, err := queue.NewRedisQueue(redisConfig)
```

### 2. Worker配置

```go
// CPU密集型：worker数量 = CPU核数
workers := runtime.NumCPU()

// I/O密集型：worker数量 = CPU核数 * 2-4
workers := runtime.NumCPU() * 3

worker := queue.NewWorker(queue, hub, retryPolicy, workers)
```

### 3. 重试策略

```go
// 重要消息：激进重试
criticalPolicy := queue.AggressiveRetryPolicy()

// 一般消息：标准重试
normalPolicy := queue.DefaultRetryPolicy()

// 批量消息：线性重试
batchPolicy := queue.LinearBackoffPolicy(3, 60*time.Second)
```

### 4. 监控告警

```go
// 定期检查队列健康
ticker := time.NewTicker(30 * time.Second)
go func() {
    for range ticker.C {
        if err := queue.Health(ctx); err != nil {
            // 发送告警
        }

        size := queue.Size()
        if size > 5000 { // 队列积压告警
            // 发送告警
        }
    }
}()
```

## 错误处理

### 常见错误类型

1. **队列满**: 入队时缓冲区已满
2. **连接失败**: Redis连接中断
3. **序列化失败**: 消息序列化/反序列化错误
4. **超时**: 操作超时

### 错误恢复

```go
// 重试入队
for attempts := 0; attempts < 3; attempts++ {
    if taskID, err := queue.Enqueue(ctx, msg); err == nil {
        break
    }
    time.Sleep(time.Duration(attempts+1) * time.Second)
}

// 降级处理
if err != nil {
    // 降级到同步发送
    _, err = hub.SendSync(ctx, msg.Message, msg.Options)
}
```

## 文件说明

- `queue.go` - 核心接口定义和SimpleQueue实现
- `redis.go` - Redis Streams队列实现
- `scheduler.go` - Min-Heap延迟消息调度器
- `worker.go` - Worker工作池实现
- `retry.go` - 重试策略实现
- `callbacks.go` - 回调机制实现
- `queue_test.go` - 单元测试和集成测试