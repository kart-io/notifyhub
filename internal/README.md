# Internal Package

## 功能概述

Internal包提供NotifyHub内部使用的核心工具和辅助功能，包括速率限制、ID生成、工具函数等。这些组件为系统的稳定性和性能提供基础支撑，不对外公开API。

## 核心组件

### 1. 速率限制器 (Rate Limiter)
- **文件**: `ratelimiter.go`
- **算法**: 令牌桶算法 (Token Bucket)
- **功能**: 防止API限流，保护外部服务

### 2. ID生成器 (ID Generator)
- **文件**: `id.go`
- **算法**: 基于时间戳的唯一ID
- **功能**: 生成消息ID、任务ID等唯一标识

### 3. 工具函数 (Utilities)
- **文件**: `utils.go`
- **功能**: 通用工具函数和辅助方法

## 速率限制系统

### 核心结构

#### RateLimiter接口
```go
type RateLimiter interface {
    Consume() error
    ConsumeWithTimeout(ctx context.Context, timeout time.Duration) error
    Available() int
    Reset()
}
```

#### RateLimitConfig配置
```go
type RateLimitConfig struct {
    Limit    int           // 令牌数量（容量）
    Duration time.Duration // 时间窗口
}
```

### 令牌桶实现

#### TokenBucket结构
```go
type TokenBucket struct {
    capacity   int           // 桶容量
    tokens     int           // 当前令牌数
    refillRate int           // 填充速率
    interval   time.Duration // 填充间隔
    lastRefill time.Time     // 上次填充时间
    mu         sync.Mutex    // 并发保护
}
```

### 使用示例

#### 基本使用
```go
// 创建速率限制配置
config := &internal.RateLimitConfig{
    Limit:    100,         // 每分钟100个令牌
    Duration: time.Minute,
}

// 创建令牌桶限制器
limiter := internal.NewRateLimiter(config)

// 消费令牌
err := limiter.Consume()
if err != nil {
    log.Printf("速率限制: %v", err)
    // 处理限流情况
}
```

#### 带超时的速率限制
```go
// 带超时的令牌消费
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := limiter.ConsumeWithTimeout(ctx, 5*time.Second)
switch err {
case nil:
    // 成功获取令牌
case context.DeadlineExceeded:
    // 超时未获取到令牌
case internal.ErrRateLimitExceeded:
    // 速率限制exceeded
}
```

#### 集成到Notifier
```go
// 在Email notifier中使用速率限制
type EmailNotifier struct {
    // ... 其他字段
    rateLimiter internal.RateLimiter
}

func (e *EmailNotifier) Send(ctx context.Context, message *Message) ([]*SendResult, error) {
    // 应用速率限制
    if err := e.rateLimiter.ConsumeWithTimeout(ctx, e.timeout); err != nil {
        return nil, fmt.Errorf("rate limit exceeded: %w", err)
    }

    // 执行实际发送逻辑
    return e.sendEmail(ctx, message)
}
```

### 高级配置

#### 自适应速率限制
```go
type AdaptiveRateLimiter struct {
    baseLimiter *TokenBucket
    errorRate   float64
    adjustment  float64
    mu          sync.RWMutex
}

func (a *AdaptiveRateLimiter) AdjustRate(errorRate float64) {
    a.mu.Lock()
    defer a.mu.Unlock()

    if errorRate > 0.1 { // 错误率超过10%
        a.adjustment = 0.5 // 降低50%速率
    } else if errorRate < 0.01 { // 错误率低于1%
        a.adjustment = 1.2 // 提高20%速率
    }
}
```

#### 分层速率限制
```go
type HierarchicalRateLimiter struct {
    globalLimiter   *TokenBucket  // 全局限制
    platformLimiter map[string]*TokenBucket // 平台特定限制
}

func (h *HierarchicalRateLimiter) Consume(platform string) error {
    // 先检查全局限制
    if err := h.globalLimiter.Consume(); err != nil {
        return err
    }

    // 再检查平台限制
    if limiter, exists := h.platformLimiter[platform]; exists {
        return limiter.Consume()
    }

    return nil
}
```

## ID生成系统

### 生成策略

#### 基于时间戳的ID
```go
// 生成基于纳秒时间戳的ID
func GenerateID() string {
    return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// 生成带前缀的ID
func GenerateIDWithPrefix(prefix string) string {
    timestamp := time.Now().UnixNano()
    return fmt.Sprintf("%s_%s", prefix, strconv.FormatInt(timestamp, 36))
}
```

#### UUID风格ID
```go
// 生成类似UUID的ID
func GenerateUUID() string {
    timestamp := time.Now().UnixNano()
    random := rand.Int63()
    return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
        uint32(timestamp>>32),
        uint16(timestamp>>16),
        uint16(timestamp),
        uint16(random>>48),
        random&0xffffffffffff)
}
```

#### 短ID生成
```go
// 生成短ID（用于任务ID等）
func GenerateShortID() string {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    const length = 8

    b := make([]byte, length)
    for i := range b {
        b[i] = charset[rand.Intn(len(charset))]
    }
    return string(b)
}
```

### 使用示例

#### 消息ID生成
```go
// 在创建消息时生成ID
message := &notifiers.Message{
    ID:    internal.GenerateID(),
    Title: "系统告警",
    Body:  "服务器异常",
}

// 在队列中生成任务ID
taskID := internal.GenerateIDWithPrefix("task")
queueMessage := &queue.Message{
    ID:      taskID,
    Message: message,
}
```

#### 追踪ID生成
```go
// 生成分布式追踪ID
traceID := internal.GenerateUUID()
span.SetAttributes(
    attribute.String("trace.id", traceID),
)
```

## 工具函数

### 字符串处理
```go
// 安全截断字符串
func TruncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen-3] + "..."
}

// 掩码敏感信息
func MaskString(s string, visibleChars int) string {
    if len(s) <= visibleChars {
        return s
    }
    return s[:visibleChars] + strings.Repeat("*", len(s)-visibleChars)
}
```

### 时间处理
```go
// 计算重试延迟（包含抖动）
func CalculateRetryDelay(attempt int, baseDelay time.Duration, maxJitter time.Duration) time.Duration {
    delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay

    if maxJitter > 0 {
        jitter := time.Duration(rand.Int63n(int64(maxJitter)))
        delay += jitter
    }

    return delay
}

// 格式化持续时间
func FormatDuration(d time.Duration) string {
    if d < time.Second {
        return fmt.Sprintf("%dms", d.Milliseconds())
    }
    if d < time.Minute {
        return fmt.Sprintf("%.1fs", d.Seconds())
    }
    return fmt.Sprintf("%.1fm", d.Minutes())
}
```

### 错误处理
```go
// 错误类型检查
func IsRetriableError(err error) bool {
    if err == nil {
        return false
    }

    // 网络错误通常可重试
    if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
        return true
    }

    // HTTP状态码检查
    if httpErr, ok := err.(*HTTPError); ok {
        return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
    }

    return false
}

// 包装错误
func WrapError(err error, operation string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%s failed: %w", operation, err)
}
```

### 并发控制
```go
// 信号量实现
type Semaphore struct {
    ch chan struct{}
}

func NewSemaphore(capacity int) *Semaphore {
    return &Semaphore{
        ch: make(chan struct{}, capacity),
    }
}

func (s *Semaphore) Acquire() {
    s.ch <- struct{}{}
}

func (s *Semaphore) Release() {
    <-s.ch
}

func (s *Semaphore) TryAcquire() bool {
    select {
    case s.ch <- struct{}{}:
        return true
    default:
        return false
    }
}
```

## 性能优化

### 对象池
```go
// 使用sync.Pool优化内存分配
var messagePool = sync.Pool{
    New: func() interface{} {
        return &Message{}
    },
}

func GetMessage() *Message {
    return messagePool.Get().(*Message)
}

func PutMessage(msg *Message) {
    msg.Reset() // 清理状态
    messagePool.Put(msg)
}
```

### 缓存机制
```go
// LRU缓存实现
type LRUCache struct {
    capacity int
    cache    map[string]*list.Element
    list     *list.List
    mu       sync.RWMutex
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if elem, exists := c.cache[key]; exists {
        c.list.MoveToFront(elem)
        return elem.Value, true
    }
    return nil, false
}

func (c *LRUCache) Put(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if elem, exists := c.cache[key]; exists {
        c.list.MoveToFront(elem)
        elem.Value = value
        return
    }

    if c.list.Len() >= c.capacity {
        // 移除最少使用的元素
        back := c.list.Back()
        c.list.Remove(back)
        delete(c.cache, back.Key.(string))
    }

    elem := c.list.PushFront(&CacheEntry{Key: key, Value: value})
    c.cache[key] = elem
}
```

## 监控与调试

### 速率限制监控
```go
// 速率限制统计
type RateLimitStats struct {
    TotalRequests   int64
    RateLimited     int64
    AverageWaitTime time.Duration
}

func (r *TokenBucket) GetStats() RateLimitStats {
    return RateLimitStats{
        TotalRequests:   atomic.LoadInt64(&r.totalRequests),
        RateLimited:     atomic.LoadInt64(&r.rateLimited),
        AverageWaitTime: r.getAverageWaitTime(),
    }
}
```

### 性能指标
```go
// 记录ID生成性能
func BenchmarkIDGeneration() {
    start := time.Now()
    for i := 0; i < 10000; i++ {
        GenerateID()
    }
    duration := time.Since(start)
    log.Printf("生成10000个ID耗时: %v, 平均: %v", duration, duration/10000)
}
```

## 最佳实践

### 1. 速率限制配置
```go
// 根据外部服务的限制配置合适的速率
var platformLimits = map[string]*internal.RateLimitConfig{
    "feishu": {
        Limit:    100,         // 飞书API限制
        Duration: time.Minute,
    },
    "email": {
        Limit:    500,         // SMTP服务器限制
        Duration: time.Hour,
    },
}
```

### 2. 错误处理
```go
// 提供降级机制
func sendWithRateLimit(ctx context.Context, msg *Message) error {
    err := rateLimiter.ConsumeWithTimeout(ctx, 5*time.Second)
    if err == internal.ErrRateLimitExceeded {
        // 降级：延迟发送
        return scheduleDelayedSend(msg, 1*time.Minute)
    }
    return send(ctx, msg)
}
```

### 3. 资源管理
```go
// 及时释放资源
func processWithSemaphore(sem *internal.Semaphore, task func()) {
    sem.Acquire()
    defer sem.Release()

    task()
}
```

## 测试辅助

### Mock实现
```go
// No-op速率限制器（用于测试）
type NoOpRateLimiter struct{}

func (n *NoOpRateLimiter) Consume() error                                                    { return nil }
func (n *NoOpRateLimiter) ConsumeWithTimeout(ctx context.Context, timeout time.Duration) error { return nil }
func (n *NoOpRateLimiter) Available() int                                                   { return 1000 }
func (n *NoOpRateLimiter) Reset()                                                          {}
```

### 测试工具
```go
// 测试ID唯一性
func TestIDUniqueness(t *testing.T) {
    seen := make(map[string]bool)
    for i := 0; i < 10000; i++ {
        id := GenerateID()
        if seen[id] {
            t.Errorf("重复ID: %s", id)
        }
        seen[id] = true
    }
}
```

## 文件说明

- `ratelimiter.go` - 速率限制器实现，包含令牌桶算法和接口定义
- `id.go` - ID生成器实现，提供多种ID生成策略
- `utils.go` - 通用工具函数，包含字符串处理、时间计算等
- `cache.go` - 缓存实现（如存在）
- `semaphore.go` - 并发控制工具（如存在）