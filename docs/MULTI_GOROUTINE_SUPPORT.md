# NotifyHub 多协程 (Goroutine) 支持详解

NotifyHub 的异步实现完全支持多协程并发处理，提供了高性能的消息发送能力。

## 🏗️ 架构设计

### 多层协程架构

```
用户代码
    ↓
NotifyHub Client (SendAsync/SendAsyncBatch)
    ↓
个人协程 (每个异步操作一个 goroutine)
    ↓
平台发送 (可能并发到多个平台)
    ↓
Worker Pool (可选，用于负载均衡)
```

### 核心协程使用场景

1. **单条异步消息**: 每次 `SendAsync()` 调用创建 1 个 goroutine
2. **批量异步消息**: 每次 `SendAsyncBatch()` 调用创建 N+1 个 goroutines (N=消息数量)
3. **回调处理**: 回调函数在发送 goroutine 中执行
4. **工作池**: 可配置的 Worker Pool 进行负载均衡

## 🚀 协程使用模式

### 1. 单条消息异步发送

```go
// 每次调用创建 1 个 goroutine
handle, err := client.SendAsync(ctx, msg)
```

**协程模型:**
```
Main Goroutine
    └── SendAsync() creates 1 goroutine
            ├── 调用 Send() 方法
            ├── 处理平台发送
            ├── 触发回调
            └── 设置结果
```

### 2. 批量消息异步发送

```go
// 创建 N+1 个 goroutines (N=消息数量)
batchHandle, err := client.SendAsyncBatch(ctx, messages)
```

**协程模型:**
```
Main Goroutine
    └── SendAsyncBatch() creates 1 coordinator goroutine
            ├── Message 1 → goroutine 1
            ├── Message 2 → goroutine 2
            ├── Message 3 → goroutine 3
            └── Message N → goroutine N
```

### 3. 并发单条消息

```go
// 用户代码并发调用，每个调用独立创建 goroutine
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(index int) {
        defer wg.Done()
        handle, _ := client.SendAsync(ctx, createMessage(index))
        handle.Wait(ctx)
    }(i)
}
wg.Wait()
```

**协程模型:**
```
Main Goroutine
    ├── User goroutine 1 → SendAsync() → processing goroutine 1
    ├── User goroutine 2 → SendAsync() → processing goroutine 2
    └── User goroutine N → SendAsync() → processing goroutine N
```

## ⚙️ 配置参数

### Worker Pool 配置

```go
cfg := config.CreateFeishuConfig()
cfg.Async.Enabled = true
cfg.Async.Workers = 8          // 工作协程数
cfg.Async.MaxWorkers = 16      // 最大协程数
cfg.Async.MinWorkers = 2       // 最小协程数
```

### 协程数量计算

| 操作类型 | 协程数量 | 说明 |
|---------|---------|------|
| SendAsync() | 1 | 每次调用创建1个处理协程 |
| SendAsyncBatch(N条消息) | N+1 | 1个协调协程 + N个处理协程 |
| 并发调用SendAsync() M次 | M | M个处理协程并行运行 |
| 并发调用SendAsyncBatch() | (N₁+1) + (N₂+1) + ... | 每个批次独立的协程组 |

## 📊 性能特征

### 协程开销

- **创建开销**: 每个 goroutine ~2KB 栈空间
- **上下文切换**: Go 运行时高效调度
- **内存管理**: 栈动态增长，GC 自动管理

### 吞吐量优化

```go
// 高吞吐量场景：批量发送
messages := make([]*message.Message, 100)
// ... 填充消息
batchHandle, err := client.SendAsyncBatch(ctx, messages)

// 高并发场景：并发单条发送
const concurrency = 50
sem := make(chan struct{}, concurrency)
for _, msg := range messages {
    sem <- struct{}{}
    go func(m *message.Message) {
        defer func() { <-sem }()
        handle, _ := client.SendAsync(ctx, m)
        handle.Wait(ctx)
    }(msg)
}
```

## 🛡️ 线程安全

### 安全保证

- ✅ **Client 线程安全**: 可以从多个 goroutine 并发调用
- ✅ **Handle 线程安全**: 可以从多个 goroutine 访问
- ✅ **回调线程安全**: 回调函数在独立 goroutine 中执行
- ✅ **状态管理**: 使用 channel 和 sync 包确保数据一致性

### 最佳实践

```go
// ✅ 正确：并发调用 SendAsync
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(index int) {
        defer wg.Done()
        handle, err := client.SendAsync(ctx, createMessage(index))
        if err != nil {
            return
        }
        handle.Wait(ctx)
    }(i)
}
wg.Wait()

// ✅ 正确：共享 Handle 对象
handle, err := client.SendAsync(ctx, msg)
go func() {
    result := <-handle.Result()
    // 处理结果
}()
receipt, err := handle.Wait(ctx)
```

## 🔧 调试和监控

### 协程数量监控

```go
import "runtime"

// 发送前
beforeGoroutines := runtime.NumGoroutine()
fmt.Printf("发送前协程数: %d\n", beforeGoroutines)

// 执行异步发送
handle, err := client.SendAsync(ctx, msg)

// 发送后
afterGoroutines := runtime.NumGoroutine()
fmt.Printf("发送后协程数: %d\n", afterGoroutines)

// 等待完成
handle.Wait(ctx)

// 完成后
finalGoroutines := runtime.NumGoroutine()
fmt.Printf("完成后协程数: %d\n", finalGoroutines)
```

### 性能分析

```go
import _ "net/http/pprof"

// 启用 pprof
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// 查看协程信息
// http://localhost:6060/debug/pprof/goroutine?debug=1
```

## 🧪 测试用例

### 压力测试

```go
func TestConcurrentSending(t *testing.T) {
    client := createTestClient()

    const (
        totalMessages = 1000
        concurrency = 50
    )

    var wg sync.WaitGroup
    sem := make(chan struct{}, concurrency)

    start := time.Now()

    for i := 0; i < totalMessages; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()

            sem <- struct{}{}
            defer func() { <-sem }()

            msg := createTestMessage(index)
            handle, err := client.SendAsync(context.Background(), msg)
            require.NoError(t, err)

            _, err = handle.Wait(context.Background())
            assert.NoError(t, err)
        }(i)
    }

    wg.Wait()
    duration := time.Since(start)

    throughput := float64(totalMessages) / duration.Seconds()
    t.Logf("吞吐量: %.2f 消息/秒", throughput)
    t.Logf("平均延迟: %v", duration/totalMessages)
}
```

## 📈 性能基准

### 典型性能指标

| 场景 | 吞吐量 | 协程数 | 内存使用 |
|------|-------|--------|---------|
| 单条异步发送 | ~100 msg/s | +1 per msg | ~2KB per goroutine |
| 批量发送 (100条) | ~1000 msg/s | +101 total | ~200KB total |
| 并发发送 (50并发) | ~2000 msg/s | +50 active | ~100KB total |

### 资源使用建议

- **小规模**: 直接使用 SendAsync，简单高效
- **中等规模**: 使用 SendAsyncBatch，平衡性能和资源
- **大规模**: 结合信号量控制并发数，避免协程数爆炸

## 🚨 注意事项

### 协程泄漏预防

```go
// ❌ 错误：可能导致协程泄漏
handle, err := client.SendAsync(ctx, msg)
// 忘记调用 handle.Wait() 或监听 handle.Result()

// ✅ 正确：确保清理资源
handle, err := client.SendAsync(ctx, msg)
defer func() {
    // 设置超时避免永久阻塞
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    handle.Wait(ctx)
}()
```

### 内存管理

```go
// 大批量发送后建议手动触发 GC
batchHandle, err := client.SendAsyncBatch(ctx, largeMessageBatch)
batchHandle.Wait(ctx)
runtime.GC() // 可选：强制垃圾回收
```

## 📚 相关示例

- `examples/multi-goroutine/main.go` - 完整的多协程使用示例
- `examples/async-callbacks/main.go` - 异步回调示例
- `examples/feishu/advanced/main.go` - Feishu 平台异步示例
- `examples/email/advanced/main.go` - Email 平台异步示例

NotifyHub 的多协程支持为高性能消息发送提供了强大的基础，合理使用可以显著提升应用的吞吐量和响应性。