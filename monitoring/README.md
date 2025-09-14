# Monitoring Package

## 功能概述

Monitoring包实现了NotifyHub的监控指标收集系统，提供实时的性能监控、成功率统计、错误追踪和健康状态检查。为系统运维和性能优化提供数据支撑。

## 核心功能

### 1. 指标收集
- 消息发送成功/失败统计
- 发送耗时和性能指标
- 平台特定监控数据
- 队列状态和积压监控

### 2. 健康检查
- 组件健康状态监控
- 外部服务连通性检查
- 系统资源使用情况
- 整体服务可用性评估

### 3. 实时统计
- 成功率计算
- 平均响应时间
- 错误率分析
- 吞吐量监控

## 核心结构

### Metrics结构体
```go
type Metrics struct {
    // 基础计数器
    totalSent     int64
    totalFailed   int64
    totalDuration int64

    // 平台特定指标
    platformStats map[string]*PlatformStats

    // 时间统计
    startTime time.Time
    lastReset time.Time

    // 并发保护
    mu sync.RWMutex
}
```

### PlatformStats结构体
```go
type PlatformStats struct {
    Sent       int64         // 发送成功数
    Failed     int64         // 发送失败数
    TotalTime  time.Duration // 总耗时
    LastError  string        // 最后错误
    LastSent   time.Time     // 最后发送时间
    Healthy    bool          // 健康状态
}
```

## 使用示例

### 基本指标记录

```go
// 创建监控实例
metrics := monitoring.NewMetrics()

// 记录成功发送
metrics.RecordSend("feishu", true, 150*time.Millisecond, "")

// 记录失败发送
metrics.RecordSend("email", false, 5*time.Second, "SMTP timeout")

// 记录健康检查
metrics.RecordHealth("feishu", true)
metrics.RecordHealth("email", false)
```

### 获取监控快照

```go
// 获取完整监控快照
snapshot := metrics.GetSnapshot()

fmt.Printf("总发送数: %d\n", snapshot["total_sent"])
fmt.Printf("成功率: %.2f%%\n", snapshot["success_rate"].(float64)*100)
fmt.Printf("平均耗时: %v\n", snapshot["avg_duration"])

// 获取平台特定统计
platformStats := snapshot["platform_stats"].(map[string]*monitoring.PlatformStats)
for platform, stats := range platformStats {
    fmt.Printf("平台 %s: 成功 %d, 失败 %d\n", platform, stats.Sent, stats.Failed)
}
```

### 健康状态检查

```go
// 检查整体健康状态
health := hub.GetHealth(ctx)
fmt.Printf("服务状态: %s\n", health["status"])
fmt.Printf("运行时间: %s\n", health["uptime"])

// 成功率低于80%会被标记为不健康
if health["status"] == "unhealthy" {
    log.Warn("服务健康状态异常，需要关注")
}
```

## 详细指标

### 基础指标

#### 发送统计
```go
// 记录发送结果
func (m *Metrics) RecordSend(platform string, success bool, duration time.Duration, errorMsg string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if success {
        atomic.AddInt64(&m.totalSent, 1)
    } else {
        atomic.AddInt64(&m.totalFailed, 1)
    }

    atomic.AddInt64(&m.totalDuration, int64(duration))

    // 更新平台特定统计
    if stats, exists := m.platformStats[platform]; exists {
        if success {
            atomic.AddInt64(&stats.Sent, 1)
        } else {
            atomic.AddInt64(&stats.Failed, 1)
            stats.LastError = errorMsg
        }
        stats.TotalTime += duration
        stats.LastSent = time.Now()
    }
}
```

#### 性能指标
```go
// 计算成功率
func (m *Metrics) GetSuccessRate() float64 {
    total := atomic.LoadInt64(&m.totalSent) + atomic.LoadInt64(&m.totalFailed)
    if total == 0 {
        return 1.0
    }
    return float64(atomic.LoadInt64(&m.totalSent)) / float64(total)
}

// 计算平均耗时
func (m *Metrics) GetAverageLatency() time.Duration {
    totalOps := atomic.LoadInt64(&m.totalSent) + atomic.LoadInt64(&m.totalFailed)
    if totalOps == 0 {
        return 0
    }
    totalDuration := atomic.LoadInt64(&m.totalDuration)
    return time.Duration(totalDuration / totalOps)
}
```

### 高级指标

#### 错误分析
```go
type ErrorStats struct {
    ErrorCounts map[string]int64  // 错误类型统计
    LastErrors  []ErrorRecord     // 最近错误记录
    mu          sync.RWMutex
}

type ErrorRecord struct {
    Timestamp time.Time
    Platform  string
    Error     string
    Message   string
}

func (e *ErrorStats) RecordError(platform, errorType, message string) {
    e.mu.Lock()
    defer e.mu.Unlock()

    // 统计错误类型
    e.ErrorCounts[errorType]++

    // 记录最近错误（保留最近100条）
    record := ErrorRecord{
        Timestamp: time.Now(),
        Platform:  platform,
        Error:     errorType,
        Message:   message,
    }

    e.LastErrors = append(e.LastErrors, record)
    if len(e.LastErrors) > 100 {
        e.LastErrors = e.LastErrors[1:]
    }
}
```

#### 吞吐量监控
```go
type ThroughputMonitor struct {
    windows []TimeWindow
    mu      sync.RWMutex
}

type TimeWindow struct {
    StartTime time.Time
    Count     int64
    Duration  time.Duration
}

func (t *ThroughputMonitor) RecordEvent() {
    t.mu.Lock()
    defer t.mu.Unlock()

    now := time.Now()

    // 找到当前时间窗口
    for i := range t.windows {
        if now.Sub(t.windows[i].StartTime) < t.windows[i].Duration {
            t.windows[i].Count++
            return
        }
    }

    // 创建新的时间窗口
    t.windows = append(t.windows, TimeWindow{
        StartTime: now,
        Count:     1,
        Duration:  time.Minute,
    })

    // 清理过期窗口
    t.cleanupExpiredWindows(now)
}

func (t *ThroughputMonitor) GetThroughput(duration time.Duration) float64 {
    t.mu.RLock()
    defer t.mu.RUnlock()

    var totalCount int64
    cutoff := time.Now().Add(-duration)

    for _, window := range t.windows {
        if window.StartTime.After(cutoff) {
            totalCount += window.Count
        }
    }

    return float64(totalCount) / duration.Minutes()
}
```

## 监控快照

### 快照数据结构
```go
type Snapshot struct {
    // 基础统计
    TotalSent     int64     `json:"total_sent"`
    TotalFailed   int64     `json:"total_failed"`
    SuccessRate   float64   `json:"success_rate"`
    AvgDuration   string    `json:"avg_duration"`

    // 时间信息
    Uptime        string    `json:"uptime"`
    LastReset     time.Time `json:"last_reset"`

    // 平台统计
    PlatformStats map[string]*PlatformStats `json:"platform_stats"`

    // 错误信息
    ErrorCounts   map[string]int64 `json:"error_counts"`
    LastErrors    []ErrorRecord    `json:"last_errors"`

    // 性能指标
    ThroughputPM  float64   `json:"throughput_per_minute"`
    P95Latency    string    `json:"p95_latency"`
    P99Latency    string    `json:"p99_latency"`
}
```

### 生成快照
```go
func (m *Metrics) GetSnapshot() map[string]interface{} {
    m.mu.RLock()
    defer m.mu.RUnlock()

    totalSent := atomic.LoadInt64(&m.totalSent)
    totalFailed := atomic.LoadInt64(&m.totalFailed)
    successRate := m.GetSuccessRate()
    avgDuration := m.GetAverageLatency()

    snapshot := map[string]interface{}{
        "total_sent":      totalSent,
        "total_failed":    totalFailed,
        "total_messages":  totalSent + totalFailed,
        "success_rate":    successRate,
        "avg_duration":    avgDuration,
        "uptime":         time.Since(m.startTime),
        "last_reset":     m.lastReset,
        "platform_stats": m.copyPlatformStats(),
        "timestamp":      time.Now(),
    }

    return snapshot
}
```

## 性能监控

### 延迟监控
```go
type LatencyMonitor struct {
    samples []time.Duration
    mu      sync.RWMutex
}

func (l *LatencyMonitor) Record(duration time.Duration) {
    l.mu.Lock()
    defer l.mu.Unlock()

    l.samples = append(l.samples, duration)

    // 保留最近1000个样本
    if len(l.samples) > 1000 {
        l.samples = l.samples[500:] // 保留后500个
    }
}

func (l *LatencyMonitor) GetPercentile(p float64) time.Duration {
    l.mu.RLock()
    defer l.mu.RUnlock()

    if len(l.samples) == 0 {
        return 0
    }

    // 复制并排序样本
    sorted := make([]time.Duration, len(l.samples))
    copy(sorted, l.samples)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i] < sorted[j]
    })

    index := int(float64(len(sorted)) * p)
    if index >= len(sorted) {
        index = len(sorted) - 1
    }

    return sorted[index]
}
```

### 资源监控
```go
type ResourceMonitor struct {
    CPUUsage    float64
    MemoryUsage int64
    GoroutineCount int
    lastUpdate  time.Time
    mu          sync.RWMutex
}

func (r *ResourceMonitor) Update() {
    r.mu.Lock()
    defer r.mu.Unlock()

    // 获取CPU使用率
    r.CPUUsage = getCPUUsage()

    // 获取内存使用
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    r.MemoryUsage = int64(m.Alloc)

    // 获取协程数量
    r.GoroutineCount = runtime.NumGoroutine()

    r.lastUpdate = time.Now()
}

func (r *ResourceMonitor) GetSnapshot() map[string]interface{} {
    r.mu.RLock()
    defer r.mu.RUnlock()

    return map[string]interface{}{
        "cpu_usage":        r.CPUUsage,
        "memory_usage_mb":  r.MemoryUsage / 1024 / 1024,
        "goroutine_count":  r.GoroutineCount,
        "last_update":      r.lastUpdate,
    }
}
```

## 告警机制

### 阈值监控
```go
type AlertRule struct {
    Name        string
    Condition   func(*Snapshot) bool
    Message     string
    Cooldown    time.Duration
    LastFired   time.Time
}

type AlertManager struct {
    rules []AlertRule
    hub   NotificationSender
    mu    sync.RWMutex
}

func (a *AlertManager) CheckAlerts(snapshot *Snapshot) {
    a.mu.Lock()
    defer a.mu.Unlock()

    for i := range a.rules {
        rule := &a.rules[i]

        // 检查冷却时间
        if time.Since(rule.LastFired) < rule.Cooldown {
            continue
        }

        // 检查条件
        if rule.Condition(snapshot) {
            a.fireAlert(rule, snapshot)
            rule.LastFired = time.Now()
        }
    }
}

func (a *AlertManager) fireAlert(rule *AlertRule, snapshot *Snapshot) {
    message := &notifiers.Message{
        Title:    fmt.Sprintf("NotifyHub Alert: %s", rule.Name),
        Body:     rule.Message,
        Priority: 4, // 高优先级
        Metadata: map[string]string{
            "alert_type": "monitoring",
            "rule_name":  rule.Name,
        },
    }

    go a.hub.SendAlert(context.Background(), message.Title, message.Body)
}
```

### 预定义告警规则
```go
var DefaultAlertRules = []AlertRule{
    {
        Name: "Low Success Rate",
        Condition: func(s *Snapshot) bool {
            return s.SuccessRate < 0.8 // 成功率低于80%
        },
        Message: "消息发送成功率低于80%，当前成功率: %.2f%%",
        Cooldown: 5 * time.Minute,
    },
    {
        Name: "High Error Rate",
        Condition: func(s *Snapshot) bool {
            total := s.TotalSent + s.TotalFailed
            return total > 100 && float64(s.TotalFailed)/float64(total) > 0.1
        },
        Message: "错误率过高，失败数: %d，总数: %d",
        Cooldown: 5 * time.Minute,
    },
    {
        Name: "High Latency",
        Condition: func(s *Snapshot) bool {
            return s.P95Latency > (5 * time.Second).String()
        },
        Message: "发送延迟过高，P95延迟: %s",
        Cooldown: 10 * time.Minute,
    },
}
```

## HTTP监控端点

### 监控API
```go
func SetupMonitoringEndpoints(hub *client.Hub) *http.ServeMux {
    mux := http.NewServeMux()

    // 健康检查端点
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        health := hub.GetHealth(r.Context())
        w.Header().Set("Content-Type", "application/json")

        if health["status"] == "healthy" {
            w.WriteHeader(http.StatusOK)
        } else {
            w.WriteHeader(http.StatusServiceUnavailable)
        }

        json.NewEncoder(w).Encode(health)
    })

    // 指标端点
    mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        metrics := hub.GetMetrics()
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(metrics)
    })

    // Prometheus指标端点
    mux.HandleFunc("/metrics/prometheus", func(w http.ResponseWriter, r *http.Request) {
        metrics := hub.GetMetrics()
        w.Header().Set("Content-Type", "text/plain")
        w.WriteHeader(http.StatusOK)

        // 输出Prometheus格式
        fmt.Fprintf(w, "notifyhub_messages_sent_total %d\n", metrics["total_sent"])
        fmt.Fprintf(w, "notifyhub_messages_failed_total %d\n", metrics["total_failed"])
        fmt.Fprintf(w, "notifyhub_success_rate %.6f\n", metrics["success_rate"])
    })

    return mux
}
```

## 最佳实践

### 1. 指标收集频率
```go
// 避免过于频繁的指标更新
type BatchMetrics struct {
    buffer []MetricEvent
    mu     sync.Mutex
}

func (b *BatchMetrics) Record(event MetricEvent) {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.buffer = append(b.buffer, event)

    if len(b.buffer) >= 100 { // 批量处理
        b.flush()
    }
}
```

### 2. 内存管理
```go
// 定期清理历史数据
func (m *Metrics) cleanup() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        m.mu.Lock()

        // 清理超过24小时的错误记录
        cutoff := time.Now().Add(-24 * time.Hour)
        filtered := m.errorRecords[:0]

        for _, record := range m.errorRecords {
            if record.Timestamp.After(cutoff) {
                filtered = append(filtered, record)
            }
        }

        m.errorRecords = filtered
        m.mu.Unlock()
    }
}
```

### 3. 监控告警
```go
// 集成外部监控系统
func (m *Metrics) SendToPrometheus() {
    // 推送指标到Pushgateway
    pusher := push.New(gatewayURL, "notifyhub")

    registry := prometheus.NewRegistry()
    registry.MustRegister(m.sentCounter)
    registry.MustRegister(m.failedCounter)

    if err := pusher.Gatherer(registry).Push(); err != nil {
        log.Printf("Failed to push metrics: %v", err)
    }
}
```

## 文件说明

- `metrics.go` - 核心监控指标实现，包含基础统计和性能监控
- `health.go` - 健康检查实现（如存在）
- `alerts.go` - 告警规则和管理（如存在）
- `endpoints.go` - HTTP监控端点（如存在）