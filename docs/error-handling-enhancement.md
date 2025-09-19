# 错误处理机制增强方案

## 概述

当前NotifyHub的错误处理机制相对基础，本文档提出了一套完整的错误处理增强方案，旨在提供更精细的错误分类、智能重试策略和用户友好的错误信息。

## 现状分析

### 当前错误处理的局限性

```go
// 当前的基础错误结构
type NotificationError struct {
    Code      ErrorCode `json:"code"`
    Message   string    `json:"message"`
    Platform  string    `json:"platform"`
    Target    string    `json:"target"`
    Retryable bool      `json:"retryable"`
}
```

**存在的问题：**
- 错误信息过于简单，缺乏上下文
- 重试策略不够智能
- 缺乏错误恢复建议
- 没有错误聚合和分析能力

## 增强设计方案

### 1. 详细错误结构

```go
// 增强的错误结构
type DetailedError struct {
    Code         ErrorCode                `json:"code"`
    Message      string                   `json:"message"`
    Platform     string                   `json:"platform"`
    Target       string                   `json:"target"`
    Retryable    bool                     `json:"retryable"`

    // 新增字段
    Context      map[string]interface{}   `json:"context"`      // 错误上下文
    Suggestion   string                   `json:"suggestion"`   // 解决建议
    RetryAfter   *time.Duration          `json:"retry_after"`  // 建议重试时间
    Category     ErrorCategory           `json:"category"`     // 错误类别
    Severity     ErrorSeverity           `json:"severity"`     // 严重程度
    RequestID    string                  `json:"request_id"`   // 请求ID
    Timestamp    time.Time               `json:"timestamp"`    // 错误时间
    StackTrace   []StackFrame            `json:"stack_trace"`  // 调用栈
}

// 错误类别
type ErrorCategory string

const (
    CategoryNetwork      ErrorCategory = "network"       // 网络错误
    CategoryAuth         ErrorCategory = "authentication" // 认证错误
    CategoryValidation   ErrorCategory = "validation"    // 验证错误
    CategoryRateLimit    ErrorCategory = "rate_limit"    // 限流错误
    CategoryPlatform     ErrorCategory = "platform"      // 平台错误
    CategoryConfiguration ErrorCategory = "configuration" // 配置错误
    CategoryInternal     ErrorCategory = "internal"      // 内部错误
)

// 错误严重程度
type ErrorSeverity string

const (
    SeverityCritical ErrorSeverity = "critical" // 严重错误
    SeverityHigh     ErrorSeverity = "high"     // 高优先级错误
    SeverityMedium   ErrorSeverity = "medium"   // 中等错误
    SeverityLow      ErrorSeverity = "low"      // 低优先级错误
    SeverityInfo     ErrorSeverity = "info"     // 信息性错误
)

// 调用栈帧
type StackFrame struct {
    Function string `json:"function"`
    File     string `json:"file"`
    Line     int    `json:"line"`
}
```

### 2. 智能重试策略

```go
// 错误恢复策略接口
type ErrorRecoveryStrategy interface {
    ShouldRetry(err *DetailedError, attempt int) bool
    NextRetryDelay(err *DetailedError, attempt int) time.Duration
    MaxAttempts(err *DetailedError) int
    FallbackAction(err *DetailedError) []RecoveryAction
}

// 恢复动作
type RecoveryAction interface {
    Execute(ctx context.Context, originalMessage *message.Message) error
    Description() string
    Cost() RecoveryCost
}

type RecoveryCost string

const (
    CostLow    RecoveryCost = "low"    // 低成本
    CostMedium RecoveryCost = "medium" // 中等成本
    CostHigh   RecoveryCost = "high"   // 高成本
)

// 智能重试策略实现
type IntelligentRetryStrategy struct {
    baseDelay       time.Duration
    maxDelay        time.Duration
    multiplier      float64
    jitterFactor    float64
    categoryLimits  map[ErrorCategory]int
    platformLimits  map[string]int
}

func (irs *IntelligentRetryStrategy) ShouldRetry(err *DetailedError, attempt int) bool {
    // 基于错误类别判断是否重试
    switch err.Category {
    case CategoryNetwork:
        return attempt < 5
    case CategoryRateLimit:
        return attempt < 3
    case CategoryAuth:
        return false // 认证错误不重试
    case CategoryValidation:
        return false // 验证错误不重试
    default:
        return attempt < 3
    }
}

func (irs *IntelligentRetryStrategy) NextRetryDelay(err *DetailedError, attempt int) time.Duration {
    // 基于错误类型调整延迟
    baseDelay := irs.baseDelay

    switch err.Category {
    case CategoryRateLimit:
        if err.RetryAfter != nil {
            baseDelay = *err.RetryAfter
        } else {
            baseDelay = time.Minute // 限流错误默认1分钟
        }
    case CategoryNetwork:
        baseDelay = irs.baseDelay * time.Duration(attempt) // 网络错误线性增长
    }

    // 指数退避 + 抖动
    delay := time.Duration(float64(baseDelay) * math.Pow(irs.multiplier, float64(attempt-1)))
    if delay > irs.maxDelay {
        delay = irs.maxDelay
    }

    // 添加抖动
    jitter := time.Duration(rand.Float64() * float64(delay) * irs.jitterFactor)
    return delay + jitter
}

func (irs *IntelligentRetryStrategy) FallbackAction(err *DetailedError) []RecoveryAction {
    var actions []RecoveryAction

    switch err.Category {
    case CategoryRateLimit:
        actions = append(actions, &DelayedRetryAction{
            delay: time.Hour, // 1小时后重试
        })
    case CategoryPlatform:
        actions = append(actions, &AlternatePlatformAction{
            excludePlatform: err.Platform,
        })
    case CategoryNetwork:
        actions = append(actions, &QueueForLaterAction{
            priority: message.PriorityHigh,
        })
    }

    return actions
}
```

### 3. 恢复动作实现

```go
// 延迟重试动作
type DelayedRetryAction struct {
    delay time.Duration
}

func (dra *DelayedRetryAction) Execute(ctx context.Context, msg *message.Message) error {
    // 将消息加入延迟队列
    return scheduleDelayedMessage(ctx, msg, dra.delay)
}

func (dra *DelayedRetryAction) Description() string {
    return fmt.Sprintf("重试发送，延迟 %v", dra.delay)
}

func (dra *DelayedRetryAction) Cost() RecoveryCost {
    return CostLow
}

// 备用平台动作
type AlternatePlatformAction struct {
    excludePlatform string
}

func (apa *AlternatePlatformAction) Execute(ctx context.Context, msg *message.Message) error {
    // 选择备用平台发送
    return sendToAlternatePlatform(ctx, msg, apa.excludePlatform)
}

func (apa *AlternatePlatformAction) Description() string {
    return fmt.Sprintf("使用备用平台发送（排除 %s）", apa.excludePlatform)
}

func (apa *AlternatePlatformAction) Cost() RecoveryCost {
    return CostMedium
}

// 队列重试动作
type QueueForLaterAction struct {
    priority int
}

func (qfla *QueueForLaterAction) Execute(ctx context.Context, msg *message.Message) error {
    // 加入高优先级队列稍后重试
    return queueMessageWithPriority(ctx, msg, qfla.priority)
}

func (qfla *QueueForLaterAction) Description() string {
    return "加入队列稍后重试"
}

func (qfla *QueueForLaterAction) Cost() RecoveryCost {
    return CostLow
}
```

### 4. 错误分析器

```go
// 错误分析器
type ErrorAnalyzer struct {
    errorHistory []ErrorEntry
    patterns     map[string]ErrorPattern
    statistics   ErrorStatistics
}

type ErrorEntry struct {
    Error     *DetailedError `json:"error"`
    Message   *message.Message `json:"message"`
    Timestamp time.Time      `json:"timestamp"`
    Resolved  bool           `json:"resolved"`
}

type ErrorPattern struct {
    Signature   string        `json:"signature"`   // 错误特征
    Frequency   int           `json:"frequency"`   // 出现频率
    LastSeen    time.Time     `json:"last_seen"`   // 最后出现时间
    Resolution  string        `json:"resolution"`  // 解决方案
    Prevention  string        `json:"prevention"`  // 预防措施
}

type ErrorStatistics struct {
    TotalErrors    int                      `json:"total_errors"`
    ErrorsByCode   map[ErrorCode]int        `json:"errors_by_code"`
    ErrorsByCategory map[ErrorCategory]int  `json:"errors_by_category"`
    ErrorsByPlatform map[string]int         `json:"errors_by_platform"`
    ErrorRate      float64                  `json:"error_rate"`
    MTTR           time.Duration            `json:"mttr"` // 平均修复时间
}

func (ea *ErrorAnalyzer) AnalyzeError(err *DetailedError, msg *message.Message) *ErrorAnalysis {
    signature := ea.generateErrorSignature(err)

    analysis := &ErrorAnalysis{
        Error:           err,
        Signature:       signature,
        SimilarErrors:   ea.findSimilarErrors(signature),
        SuggestedActions: ea.generateSuggestions(err),
        RootCause:       ea.analyzeRootCause(err),
        Impact:          ea.assessImpact(err, msg),
    }

    return analysis
}

type ErrorAnalysis struct {
    Error            *DetailedError    `json:"error"`
    Signature        string           `json:"signature"`
    SimilarErrors    []ErrorEntry     `json:"similar_errors"`
    SuggestedActions []string         `json:"suggested_actions"`
    RootCause        *RootCause       `json:"root_cause"`
    Impact           *ImpactAssessment `json:"impact"`
}

type RootCause struct {
    Category    string   `json:"category"`
    Description string   `json:"description"`
    Components  []string `json:"components"`
    Confidence  float64  `json:"confidence"`
}

type ImpactAssessment struct {
    AffectedUsers    int     `json:"affected_users"`
    BusinessImpact   string  `json:"business_impact"`
    TechnicalImpact  string  `json:"technical_impact"`
    UrgencyLevel     string  `json:"urgency_level"`
}
```

## 实现指南

### 1. 错误处理器集成

```go
// 在Hub中集成错误处理器
type Hub struct {
    // ... 现有字段
    errorHandler    *ErrorHandler
    errorAnalyzer   *ErrorAnalyzer
    retryStrategy   ErrorRecoveryStrategy
}

type ErrorHandler struct {
    analyzer       *ErrorAnalyzer
    retryStrategy  ErrorRecoveryStrategy
    notifications  []ErrorNotification
}

func (eh *ErrorHandler) HandleError(ctx context.Context, err error, msg *message.Message, target sending.Target) (*ErrorHandlingResult, error) {
    // 转换为详细错误
    detailedErr := eh.convertToDetailedError(err, msg, target)

    // 分析错误
    analysis := eh.analyzer.AnalyzeError(detailedErr, msg)

    // 确定恢复策略
    shouldRetry := eh.retryStrategy.ShouldRetry(detailedErr, msg.AttemptCount)

    result := &ErrorHandlingResult{
        Error:           detailedErr,
        Analysis:        analysis,
        ShouldRetry:     shouldRetry,
        RetryDelay:      eh.retryStrategy.NextRetryDelay(detailedErr, msg.AttemptCount),
        FallbackActions: eh.retryStrategy.FallbackAction(detailedErr),
    }

    // 发送错误通知（如果配置）
    eh.notifyError(detailedErr, analysis)

    return result, nil
}

type ErrorHandlingResult struct {
    Error           *DetailedError     `json:"error"`
    Analysis        *ErrorAnalysis     `json:"analysis"`
    ShouldRetry     bool              `json:"should_retry"`
    RetryDelay      time.Duration     `json:"retry_delay"`
    FallbackActions []RecoveryAction  `json:"fallback_actions"`
}
```

### 2. 客户端API集成

```go
// 为客户端添加错误处理配置
func WithErrorHandling(config ErrorHandlingConfig) Option {
    return func(cfg *Config) {
        cfg.ErrorHandling = &config
    }
}

type ErrorHandlingConfig struct {
    EnableAnalysis     bool                    `json:"enable_analysis"`
    RetryStrategy     ErrorRecoveryStrategy   `json:"retry_strategy"`
    NotificationConfig *ErrorNotificationConfig `json:"notification_config"`
    LogErrors         bool                    `json:"log_errors"`
    PersistErrors     bool                    `json:"persist_errors"`
}

// 使用示例
client, err := notifyhub.New(
    notifyhub.WithFeishu("webhook", "secret"),
    notifyhub.WithErrorHandling(ErrorHandlingConfig{
        EnableAnalysis: true,
        RetryStrategy: &IntelligentRetryStrategy{
            baseDelay:    time.Second,
            maxDelay:     time.Minute * 10,
            multiplier:   2.0,
            jitterFactor: 0.1,
        },
        LogErrors:     true,
        PersistErrors: true,
    }),
)
```

### 3. 错误通知系统

```go
// 错误通知配置
type ErrorNotificationConfig struct {
    Enabled     bool              `json:"enabled"`
    Channels    []NotifyChannel   `json:"channels"`
    Filters     []ErrorFilter     `json:"filters"`
    Throttling  ThrottlingConfig  `json:"throttling"`
}

type NotifyChannel struct {
    Type     string            `json:"type"`     // "email", "slack", "webhook"
    Config   map[string]string `json:"config"`
    Severity []ErrorSeverity   `json:"severity"` // 通知的错误级别
}

// 错误过滤器
type ErrorFilter interface {
    ShouldNotify(err *DetailedError) bool
}

// 频率过滤器
type FrequencyFilter struct {
    MaxPerMinute int `json:"max_per_minute"`
    MaxPerHour   int `json:"max_per_hour"`
}

func (ff *FrequencyFilter) ShouldNotify(err *DetailedError) bool {
    // 检查错误频率是否超过阈值
    return true
}
```

## 使用示例

### 基础错误处理

```go
// 发送消息并处理错误
result, err := client.Send(ctx).
    Title("测试消息").
    ToEmail("invalid-email").
    Execute()

if err != nil {
    log.Printf("发送失败: %v", err)
    return
}

// 检查详细结果
for _, targetResult := range result.Results {
    if targetResult.Status == notifyhub.StatusFailed {
        if detailedErr, ok := targetResult.Error.(*DetailedError); ok {
            log.Printf("目标 %s 发送失败:", targetResult.Target.String())
            log.Printf("  错误类别: %s", detailedErr.Category)
            log.Printf("  严重程度: %s", detailedErr.Severity)
            log.Printf("  建议: %s", detailedErr.Suggestion)

            if detailedErr.Retryable && detailedErr.RetryAfter != nil {
                log.Printf("  建议重试时间: %v", *detailedErr.RetryAfter)
            }
        }
    }
}
```

### 自定义错误处理

```go
// 自定义重试策略
customStrategy := &CustomRetryStrategy{
    maxAttempts: map[ErrorCategory]int{
        CategoryNetwork:   5,
        CategoryRateLimit: 2,
        CategoryAuth:      0,
    },
}

client, err := notifyhub.New(
    notifyhub.WithFeishu("webhook", "secret"),
    notifyhub.WithErrorHandling(ErrorHandlingConfig{
        RetryStrategy: customStrategy,
        NotificationConfig: &ErrorNotificationConfig{
            Enabled: true,
            Channels: []NotifyChannel{
                {
                    Type: "email",
                    Config: map[string]string{
                        "to": "ops@company.com",
                    },
                    Severity: []ErrorSeverity{SeverityCritical, SeverityHigh},
                },
            },
        },
    }),
)
```

## 性能考虑

### 1. 异步错误分析

```go
// 异步错误分析避免影响主流程
type AsyncErrorAnalyzer struct {
    analyzer    *ErrorAnalyzer
    errorQueue  chan ErrorAnalysisTask
    workers     int
}

type ErrorAnalysisTask struct {
    Error   *DetailedError
    Message *message.Message
    Done    chan *ErrorAnalysis
}

func (aea *AsyncErrorAnalyzer) AnalyzeAsync(err *DetailedError, msg *message.Message) <-chan *ErrorAnalysis {
    done := make(chan *ErrorAnalysis, 1)
    task := ErrorAnalysisTask{
        Error:   err,
        Message: msg,
        Done:    done,
    }

    select {
    case aea.errorQueue <- task:
    default:
        // 队列满时快速失败
        close(done)
    }

    return done
}
```

### 2. 错误聚合和批量处理

```go
// 错误聚合器
type ErrorAggregator struct {
    buffer    []ErrorEntry
    ticker    *time.Ticker
    batchSize int
}

func (ea *ErrorAggregator) AddError(err *DetailedError, msg *message.Message) {
    ea.buffer = append(ea.buffer, ErrorEntry{
        Error:     err,
        Message:   msg,
        Timestamp: time.Now(),
    })

    if len(ea.buffer) >= ea.batchSize {
        ea.processBatch()
    }
}

func (ea *ErrorAggregator) processBatch() {
    // 批量处理错误
    for _, entry := range ea.buffer {
        // 分析和处理
    }
    ea.buffer = ea.buffer[:0]
}
```

## 监控和指标

### 错误指标收集

```go
type ErrorMetrics struct {
    TotalErrors      prometheus.Counter
    ErrorsByCategory prometheus.CounterVec
    ErrorsByPlatform prometheus.CounterVec
    ErrorRate        prometheus.Gauge
    MTTR             prometheus.Histogram
}

func (em *ErrorMetrics) RecordError(err *DetailedError) {
    em.TotalErrors.Inc()
    em.ErrorsByCategory.WithLabelValues(string(err.Category)).Inc()
    em.ErrorsByPlatform.WithLabelValues(err.Platform).Inc()
}
```

## 总结

这套错误处理增强方案提供了：

1. **详细的错误信息**：包含上下文、建议和调用栈
2. **智能重试策略**：基于错误类型的差异化重试
3. **自动错误恢复**：多种恢复动作和备用方案
4. **错误分析能力**：模式识别和根因分析
5. **完善的监控**：错误指标和通知系统

该方案既保持了API的简洁性，又提供了企业级的错误处理能力，为生产环境的稳定运行提供了强有力的保障。