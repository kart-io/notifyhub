# 批量操作优化方案

## 概述

批量操作优化方案旨在提高NotifyHub处理大量消息的效率，通过批量发送、并行处理和智能调度等技术，显著提升系统吞吐量并降低资源消耗。

## 需求分析

### 批量操作场景

1. **营销活动通知**：向数万用户发送促销信息
2. **系统公告**：向全体用户发送重要通知
3. **告警批量处理**：同时处理多个系统告警
4. **定时任务**：定期发送报表、提醒等
5. **事件触发**：基于事件批量发送通知

### 性能挑战

- **高并发压力**：大量消息同时处理
- **资源消耗**：内存和CPU使用优化
- **平台限制**：各平台的速率限制
- **错误处理**：批量操作中的错误隔离

## 设计方案

### 1. 批量操作接口设计

```go
// 批量操作构建器
type BatchBuilder interface {
    // 批量消息管理
    Add(builder *SendBuilder) BatchBuilder
    AddMessage(msg *message.Message, targets []sending.Target) BatchBuilder
    AddTemplate(template string, variables []map[string]interface{}, targets []sending.Target) BatchBuilder

    // 批量配置
    WithConcurrency(concurrency int) BatchBuilder
    WithBatchSize(size int) BatchBuilder
    WithRetryPolicy(policy BatchRetryPolicy) BatchBuilder
    WithPriority(priority BatchPriority) BatchBuilder

    // 执行选项
    Execute() (*BatchResults, error)
    ExecuteAsync() (*AsyncBatchHandle, error)
    Stream(callback BatchResultCallback) error
    Validate() (*BatchValidationResult, error)
}

// 批量结果
type BatchResults struct {
    BatchID        string             `json:"batch_id"`
    TotalMessages  int               `json:"total_messages"`
    Sent           int               `json:"sent"`
    Failed         int               `json:"failed"`
    Pending        int               `json:"pending"`
    Results        []BatchItemResult `json:"results"`
    StartTime      time.Time         `json:"start_time"`
    EndTime        time.Time         `json:"end_time"`
    Duration       time.Duration     `json:"duration"`
    Throughput     float64           `json:"throughput"` // 消息/秒
    ErrorSummary   ErrorSummary      `json:"error_summary"`
}

// 批量项目结果
type BatchItemResult struct {
    Index       int                `json:"index"`
    MessageID   string            `json:"message_id"`
    Status      BatchItemStatus   `json:"status"`
    Results     []TargetResult    `json:"results"`
    Error       error             `json:"error,omitempty"`
    ProcessTime time.Duration     `json:"process_time"`
}

type BatchItemStatus string

const (
    BatchItemStatusPending    BatchItemStatus = "pending"
    BatchItemStatusProcessing BatchItemStatus = "processing"
    BatchItemStatusCompleted  BatchItemStatus = "completed"
    BatchItemStatusFailed     BatchItemStatus = "failed"
    BatchItemStatusSkipped    BatchItemStatus = "skipped"
)

// 错误汇总
type ErrorSummary struct {
    TotalErrors      int                    `json:"total_errors"`
    ErrorsByType     map[string]int         `json:"errors_by_type"`
    ErrorsByPlatform map[string]int         `json:"errors_by_platform"`
    CriticalErrors   []BatchItemResult      `json:"critical_errors"`
}

// 批量优先级
type BatchPriority string

const (
    BatchPriorityLow      BatchPriority = "low"
    BatchPriorityNormal   BatchPriority = "normal"
    BatchPriorityHigh     BatchPriority = "high"
    BatchPriorityCritical BatchPriority = "critical"
)
```

### 2. 批量处理器实现

```go
// 批量处理器
type BatchProcessor struct {
    client       *Client
    config       BatchConfig
    scheduler    *BatchScheduler
    rateLimiter  map[string]*RateLimiter
    metrics      *BatchMetrics
    errorHandler *BatchErrorHandler
}

type BatchConfig struct {
    DefaultConcurrency    int           `json:"default_concurrency"`
    DefaultBatchSize      int           `json:"default_batch_size"`
    MaxConcurrency        int           `json:"max_concurrency"`
    MaxBatchSize          int           `json:"max_batch_size"`
    ProcessTimeout        time.Duration `json:"process_timeout"`
    EnableMetrics         bool          `json:"enable_metrics"`
    EnableErrorRecovery   bool          `json:"enable_error_recovery"`
    PlatformRateLimits    map[string]RateLimitConfig `json:"platform_rate_limits"`
}

// 批量构建器实现
type DefaultBatchBuilder struct {
    processor   *BatchProcessor
    items       []BatchItem
    config      BatchExecutionConfig
    validation  *BatchValidation
}

type BatchItem struct {
    Message *message.Message    `json:"message"`
    Targets []sending.Target    `json:"targets"`
    Metadata map[string]string  `json:"metadata"`
    Priority int                `json:"priority"`
}

type BatchExecutionConfig struct {
    Concurrency   int               `json:"concurrency"`
    BatchSize     int               `json:"batch_size"`
    Priority      BatchPriority     `json:"priority"`
    RetryPolicy   BatchRetryPolicy  `json:"retry_policy"`
    Timeout       time.Duration     `json:"timeout"`
    ErrorHandling ErrorHandlingMode `json:"error_handling"`
}

type ErrorHandlingMode string

const (
    ErrorHandlingContinue ErrorHandlingMode = "continue" // 继续处理其他消息
    ErrorHandlingAbort    ErrorHandlingMode = "abort"    // 遇到错误停止
    ErrorHandlingRetry    ErrorHandlingMode = "retry"    // 重试失败项目
)

func (dbb *DefaultBatchBuilder) Add(builder *SendBuilder) BatchBuilder {
    item := BatchItem{
        Message:  builder.message,
        Targets:  builder.targets,
        Metadata: builder.metadata,
        Priority: builder.message.Priority,
    }
    dbb.items = append(dbb.items, item)
    return dbb
}

func (dbb *DefaultBatchBuilder) AddTemplate(template string, variablesList []map[string]interface{}, targets []sending.Target) BatchBuilder {
    for _, variables := range variablesList {
        msg := message.NewMessage()
        msg.Template = template
        msg.Variables = variables

        item := BatchItem{
            Message: msg,
            Targets: targets,
        }
        dbb.items = append(dbb.items, item)
    }
    return dbb
}

func (dbb *DefaultBatchBuilder) Execute() (*BatchResults, error) {
    // 验证批量请求
    if err := dbb.validate(); err != nil {
        return nil, err
    }

    // 创建批量任务
    batch := &BatchTask{
        ID:     generateBatchID(),
        Items:  dbb.items,
        Config: dbb.config,
        Status: BatchStatusPending,
        StartTime: time.Now(),
    }

    // 提交给处理器
    return dbb.processor.ProcessBatch(batch)
}

func (dbb *DefaultBatchBuilder) ExecuteAsync() (*AsyncBatchHandle, error) {
    // 异步执行
    batch := &BatchTask{
        ID:     generateBatchID(),
        Items:  dbb.items,
        Config: dbb.config,
        Status: BatchStatusPending,
        StartTime: time.Now(),
    }

    handle := &AsyncBatchHandle{
        BatchID:    batch.ID,
        ResultChan: make(chan *BatchResults, 1),
        StatusChan: make(chan BatchStatus, 10),
    }

    go func() {
        result, err := dbb.processor.ProcessBatch(batch)
        if err != nil {
            handle.Error = err
        }
        handle.ResultChan <- result
        close(handle.ResultChan)
        close(handle.StatusChan)
    }()

    return handle, nil
}
```

### 3. 智能批量调度器

```go
// 批量调度器
type BatchScheduler struct {
    queues       map[BatchPriority]*PriorityQueue
    workers      []*BatchWorker
    config       SchedulerConfig
    metrics      *SchedulerMetrics
    rateLimiters map[string]*RateLimiter
}

type SchedulerConfig struct {
    WorkerCount        int           `json:"worker_count"`
    QueueCapacity      int           `json:"queue_capacity"`
    ScheduleInterval   time.Duration `json:"schedule_interval"`
    LoadBalanceStrategy string       `json:"load_balance_strategy"`
    PriorityWeights    map[BatchPriority]int `json:"priority_weights"`
}

type BatchWorker struct {
    ID          string
    processor   *BatchProcessor
    currentTask *BatchTask
    status      WorkerStatus
    metrics     *WorkerMetrics
}

type WorkerStatus string

const (
    WorkerStatusIdle       WorkerStatus = "idle"
    WorkerStatusProcessing WorkerStatus = "processing"
    WorkerStatusError      WorkerStatus = "error"
)

func (bs *BatchScheduler) ScheduleBatch(batch *BatchTask) error {
    queue := bs.queues[batch.Config.Priority]
    if queue == nil {
        queue = bs.queues[BatchPriorityNormal]
    }

    return queue.Push(batch)
}

func (bs *BatchScheduler) Start(ctx context.Context) error {
    // 启动工作器
    for i := 0; i < bs.config.WorkerCount; i++ {
        worker := &BatchWorker{
            ID:        fmt.Sprintf("worker-%d", i),
            processor: bs.newProcessorForWorker(),
            status:    WorkerStatusIdle,
        }
        bs.workers = append(bs.workers, worker)

        go bs.runWorker(ctx, worker)
    }

    // 启动调度器
    go bs.runScheduler(ctx)

    return nil
}

func (bs *BatchScheduler) runScheduler(ctx context.Context) {
    ticker := time.NewTicker(bs.config.ScheduleInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            bs.scheduleNextBatch()
        }
    }
}

func (bs *BatchScheduler) scheduleNextBatch() {
    // 按优先级顺序处理
    priorities := []BatchPriority{
        BatchPriorityCritical,
        BatchPriorityHigh,
        BatchPriorityNormal,
        BatchPriorityLow,
    }

    for _, priority := range priorities {
        queue := bs.queues[priority]
        if queue.IsEmpty() {
            continue
        }

        // 查找空闲工作器
        worker := bs.findIdleWorker()
        if worker == nil {
            return // 没有空闲工作器
        }

        // 分配任务
        if batch, ok := queue.Pop().(*BatchTask); ok {
            bs.assignTaskToWorker(worker, batch)
        }
    }
}

func (bs *BatchScheduler) runWorker(ctx context.Context, worker *BatchWorker) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            if worker.currentTask != nil {
                bs.processWorkerTask(worker)
            } else {
                time.Sleep(100 * time.Millisecond)
            }
        }
    }
}
```

### 4. 平台优化的批量发送

```go
// 平台批量适配器
type PlatformBatchAdapter interface {
    GetBatchCapacity() int
    GetRateLimit() RateLimitConfig
    SendBatch(messages []BatchMessage) (*PlatformBatchResult, error)
    SupportsTemplating() bool
    OptimizeBatch(messages []BatchMessage) []BatchMessage
}

// 飞书批量适配器
type FeishuBatchAdapter struct {
    transport Transport
    config    FeishuBatchConfig
}

type FeishuBatchConfig struct {
    MaxBatchSize    int           `json:"max_batch_size"`
    RequestInterval time.Duration `json:"request_interval"`
    Timeout         time.Duration `json:"timeout"`
}

func (fba *FeishuBatchAdapter) SendBatch(messages []BatchMessage) (*PlatformBatchResult, error) {
    // 飞书批量发送优化
    result := &PlatformBatchResult{
        Platform:    "feishu",
        TotalSent:   0,
        TotalFailed: 0,
        Results:     make([]MessageResult, 0),
    }

    // 按webhook分组
    groups := fba.groupByWebhook(messages)

    for webhook, msgs := range groups {
        batchResult, err := fba.sendBatchToWebhook(webhook, msgs)
        if err != nil {
            // 标记所有消息为失败
            for _, msg := range msgs {
                result.Results = append(result.Results, MessageResult{
                    MessageID: msg.ID,
                    Status:    "failed",
                    Error:     err,
                })
                result.TotalFailed++
            }
            continue
        }

        result.Results = append(result.Results, batchResult.Results...)
        result.TotalSent += batchResult.TotalSent
        result.TotalFailed += batchResult.TotalFailed

        // 遵守速率限制
        time.Sleep(fba.config.RequestInterval)
    }

    return result, nil
}

func (fba *FeishuBatchAdapter) groupByWebhook(messages []BatchMessage) map[string][]BatchMessage {
    groups := make(map[string][]BatchMessage)

    for _, msg := range messages {
        for _, target := range msg.Targets {
            if target.Platform == "feishu" {
                webhook := target.GetMetadata("webhook")
                if webhook != "" {
                    groups[webhook] = append(groups[webhook], msg)
                }
            }
        }
    }

    return groups
}

// 邮件批量适配器
type EmailBatchAdapter struct {
    transport Transport
    config    EmailBatchConfig
}

type EmailBatchConfig struct {
    MaxBatchSize      int           `json:"max_batch_size"`
    ConnectionPoolSize int          `json:"connection_pool_size"`
    Timeout           time.Duration `json:"timeout"`
    UseTemplating     bool          `json:"use_templating"`
}

func (eba *EmailBatchAdapter) SendBatch(messages []BatchMessage) (*PlatformBatchResult, error) {
    if eba.config.UseTemplating {
        return eba.sendBatchWithTemplating(messages)
    }
    return eba.sendBatchSequential(messages)
}

func (eba *EmailBatchAdapter) sendBatchWithTemplating(messages []BatchMessage) (*PlatformBatchResult, error) {
    // 使用邮件模板批量发送
    templateGroups := eba.groupByTemplate(messages)

    result := &PlatformBatchResult{Platform: "email"}

    for template, msgs := range templateGroups {
        batchResult, err := eba.sendTemplatedBatch(template, msgs)
        if err != nil {
            continue
        }
        result.merge(batchResult)
    }

    return result, nil
}
```

### 5. 批量错误处理和恢复

```go
// 批量错误处理器
type BatchErrorHandler struct {
    retryQueue    *RetryQueue
    deadLetterBox *DeadLetterBox
    config        BatchErrorConfig
    analyzer      *BatchErrorAnalyzer
}

type BatchErrorConfig struct {
    MaxRetries         int           `json:"max_retries"`
    RetryInterval      time.Duration `json:"retry_interval"`
    EnableDeadLetter   bool          `json:"enable_dead_letter"`
    ErrorThreshold     float64       `json:"error_threshold"`     // 错误率阈值
    CircuitBreakerEnabled bool       `json:"circuit_breaker_enabled"`
}

type BatchErrorAnalyzer struct {
    patterns map[string]ErrorPattern
    metrics  *ErrorMetrics
}

func (beh *BatchErrorHandler) HandleBatchError(batch *BatchTask, errors []BatchItemError) (*BatchErrorRecovery, error) {
    recovery := &BatchErrorRecovery{
        BatchID:        batch.ID,
        RecoveryAction: RecoveryActionNone,
        RetryItems:     make([]BatchItem, 0),
        SkipItems:      make([]BatchItem, 0),
        Recommendations: make([]string, 0),
    }

    // 分析错误模式
    analysis := beh.analyzer.AnalyzeErrors(errors)

    // 根据错误类型决定恢复策略
    if analysis.NetworkErrorRate > 0.8 {
        recovery.RecoveryAction = RecoveryActionRetryAll
        recovery.RetryDelay = 5 * time.Minute
        recovery.Recommendations = append(recovery.Recommendations, "Network issues detected, retry all messages")
    } else if analysis.RateLimitErrorRate > 0.5 {
        recovery.RecoveryAction = RecoveryActionThrottle
        recovery.RetryDelay = 10 * time.Minute
        recovery.Recommendations = append(recovery.Recommendations, "Rate limit exceeded, reduce sending rate")
    } else {
        // 选择性重试
        for _, item := range batch.Items {
            if beh.shouldRetryItem(item, errors) {
                recovery.RetryItems = append(recovery.RetryItems, item)
            } else {
                recovery.SkipItems = append(recovery.SkipItems, item)
            }
        }
        recovery.RecoveryAction = RecoveryActionSelective
    }

    return recovery, nil
}

type RecoveryAction string

const (
    RecoveryActionNone      RecoveryAction = "none"
    RecoveryActionRetryAll  RecoveryAction = "retry_all"
    RecoveryActionSelective RecoveryAction = "selective"
    RecoveryActionThrottle  RecoveryAction = "throttle"
    RecoveryActionCircuitBreak RecoveryAction = "circuit_break"
)

type BatchErrorRecovery struct {
    BatchID         string         `json:"batch_id"`
    RecoveryAction  RecoveryAction `json:"recovery_action"`
    RetryItems      []BatchItem    `json:"retry_items"`
    SkipItems       []BatchItem    `json:"skip_items"`
    RetryDelay      time.Duration  `json:"retry_delay"`
    Recommendations []string       `json:"recommendations"`
}
```

### 6. 批量性能优化

```go
// 批量性能优化器
type BatchPerformanceOptimizer struct {
    config      OptimizationConfig
    profiler    *BatchProfiler
    recommender *PerformanceRecommender
}

type OptimizationConfig struct {
    EnableAutoTuning     bool    `json:"enable_auto_tuning"`
    MinConcurrency       int     `json:"min_concurrency"`
    MaxConcurrency       int     `json:"max_concurrency"`
    TargetThroughput     float64 `json:"target_throughput"`
    TargetLatency        time.Duration `json:"target_latency"`
    AdaptationInterval   time.Duration `json:"adaptation_interval"`
}

type BatchProfiler struct {
    measurements []PerformanceMeasurement
    currentMetrics BatchPerformanceMetrics
}

type PerformanceMeasurement struct {
    Timestamp    time.Time     `json:"timestamp"`
    Concurrency  int           `json:"concurrency"`
    BatchSize    int           `json:"batch_size"`
    Throughput   float64       `json:"throughput"`
    Latency      time.Duration `json:"latency"`
    ErrorRate    float64       `json:"error_rate"`
    CPUUsage     float64       `json:"cpu_usage"`
    MemoryUsage  int64         `json:"memory_usage"`
}

func (bpo *BatchPerformanceOptimizer) OptimizeBatch(batch *BatchTask) *OptimizedBatchConfig {
    profile := bpo.profiler.ProfileBatch(batch)

    config := &OptimizedBatchConfig{
        Concurrency: batch.Config.Concurrency,
        BatchSize:   batch.Config.BatchSize,
    }

    // 基于历史数据调整并发度
    if bpo.config.EnableAutoTuning {
        optimalConcurrency := bpo.calculateOptimalConcurrency(profile)
        if optimalConcurrency != config.Concurrency {
            config.Concurrency = optimalConcurrency
            config.Recommendations = append(config.Recommendations,
                fmt.Sprintf("Adjusted concurrency from %d to %d", batch.Config.Concurrency, optimalConcurrency))
        }

        optimalBatchSize := bpo.calculateOptimalBatchSize(profile)
        if optimalBatchSize != config.BatchSize {
            config.BatchSize = optimalBatchSize
            config.Recommendations = append(config.Recommendations,
                fmt.Sprintf("Adjusted batch size from %d to %d", batch.Config.BatchSize, optimalBatchSize))
        }
    }

    return config
}

type OptimizedBatchConfig struct {
    Concurrency     int      `json:"concurrency"`
    BatchSize       int      `json:"batch_size"`
    Recommendations []string `json:"recommendations"`
    Confidence      float64  `json:"confidence"`
}

func (bpo *BatchPerformanceOptimizer) calculateOptimalConcurrency(profile *BatchProfile) int {
    // 基于吞吐量和错误率计算最优并发度
    measurements := bpo.profiler.measurements

    if len(measurements) < 3 {
        return profile.Concurrency // 数据不足，保持当前值
    }

    maxThroughput := 0.0
    optimalConcurrency := profile.Concurrency

    for _, m := range measurements {
        if m.ErrorRate < 0.05 && m.Throughput > maxThroughput {
            maxThroughput = m.Throughput
            optimalConcurrency = m.Concurrency
        }
    }

    // 限制在配置范围内
    if optimalConcurrency < bpo.config.MinConcurrency {
        optimalConcurrency = bpo.config.MinConcurrency
    } else if optimalConcurrency > bpo.config.MaxConcurrency {
        optimalConcurrency = bpo.config.MaxConcurrency
    }

    return optimalConcurrency
}
```

## 使用示例

### 1. 基础批量操作

```go
// 创建批量构建器
batch := client.Batch(ctx)

// 添加多个消息
for i := 0; i < 1000; i++ {
    batch.Add(
        client.Send(ctx).
            Title(fmt.Sprintf("通知 %d", i)).
            Body("批量发送测试").
            ToEmail(fmt.Sprintf("user%d@example.com", i)),
    )
}

// 配置批量参数
result, err := batch.
    WithConcurrency(10).
    WithBatchSize(50).
    Execute()

if err != nil {
    log.Printf("批量发送失败: %v", err)
} else {
    log.Printf("批量发送完成: 成功 %d, 失败 %d", result.Sent, result.Failed)
}
```

### 2. 模板批量发送

```go
// 用户数据
users := []map[string]interface{}{
    {"name": "张三", "email": "zhangsan@example.com", "balance": 1000},
    {"name": "李四", "email": "lisi@example.com", "balance": 2000},
    // ... 更多用户
}

// 目标用户
targets := make([]sending.Target, len(users))
for i, user := range users {
    targets[i] = sending.NewTarget(sending.TargetTypeEmail, user["email"].(string), "email")
}

// 批量模板发送
result, err := client.Batch(ctx).
    AddTemplate("welcome-template", users, targets).
    WithConcurrency(5).
    Execute()
```

### 3. 异步批量处理

```go
// 异步执行大批量任务
handle, err := client.Batch(ctx).
    Add(/* ... 大量消息 ... */).
    WithConcurrency(20).
    ExecuteAsync()

if err != nil {
    log.Fatal(err)
}

// 监听进度
go func() {
    for status := range handle.StatusChan {
        log.Printf("批量任务状态: %s", status)
    }
}()

// 等待完成
result := <-handle.ResultChan
if handle.Error != nil {
    log.Printf("批量任务失败: %v", handle.Error)
} else {
    log.Printf("批量任务完成: %+v", result)
}
```

### 4. 流式批量处理

```go
// 流式处理大量数据
err := client.Batch(ctx).
    Add(/* ... 消息列表 ... */).
    Stream(func(result BatchItemResult) {
        if result.Status == BatchItemStatusCompleted {
            log.Printf("消息 %s 发送成功", result.MessageID)
        } else {
            log.Printf("消息 %s 发送失败: %v", result.MessageID, result.Error)
        }
    })
```

### 5. 高级配置

```go
// 高级批量配置
result, err := client.Batch(ctx).
    Add(/* ... 消息 ... */).
    WithConcurrency(15).
    WithBatchSize(100).
    WithRetryPolicy(BatchRetryPolicy{
        MaxRetries:    3,
        BackoffFactor: 2.0,
        MaxDelay:      5 * time.Minute,
    }).
    WithPriority(BatchPriorityHigh).
    Execute()
```

## 性能基准

### 预期性能指标

| 指标 | 单个发送 | 批量发送(100条) | 改进倍数 |
|------|---------|---------------|----------|
| **吞吐量** | 10 msg/s | 500 msg/s | 50x |
| **延迟** | 100ms | 200ms | - |
| **CPU使用** | 高 | 低 | 3x优化 |
| **内存使用** | 高 | 低 | 5x优化 |
| **错误恢复** | 单独处理 | 批量恢复 | 10x |

### 平台优化效果

| 平台 | 批量大小 | 性能提升 | 注意事项 |
|------|---------|---------|----------|
| **飞书** | 50条 | 30x | webhook分组 |
| **邮件** | 100条 | 25x | SMTP连接复用 |
| **短信** | 200条 | 40x | 模板批量发送 |

## 总结

批量操作优化方案提供了：

1. **高效批量接口**：简洁易用的批量操作API
2. **智能调度器**：基于优先级和负载的智能调度
3. **平台优化**：针对各平台特点的优化策略
4. **错误恢复**：完善的批量错误处理和恢复机制
5. **性能优化**：自动调优和性能监控

该方案能够将系统吞吐量提升20-50倍，同时保持良好的错误处理和用户体验。