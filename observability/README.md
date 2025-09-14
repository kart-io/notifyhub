# Observability Package

## 功能概述

Observability包实现了NotifyHub的可观测性系统，基于OpenTelemetry标准提供分布式追踪、指标收集和监控功能。支持OTLP协议，可以与Jaeger、Prometheus、Grafana等监控系统无缝集成。

## 核心功能

### 1. 分布式追踪 (Distributed Tracing)
- 消息发送全链路追踪
- 跨服务调用追踪
- 错误和异常追踪
- 性能瓶颈识别

### 2. 指标收集 (Metrics)
- 消息发送成功/失败计数
- 发送耗时直方图
- 队列大小监控
- 平台特定指标

### 3. 标准化集成
- OpenTelemetry标准协议
- OTLP HTTP/gRPC导出器
- 自动化资源标识
- 灵活的采样配置

## 核心结构

### TelemetryProvider
```go
type TelemetryProvider struct {
    config        *config.TelemetryConfig
    tracer        trace.Tracer
    meter         metric.Meter
    traceProvider *sdktrace.TracerProvider

    // 指标
    messagesSent     metric.Int64Counter
    messagesEnqueued metric.Int64Counter
    messagesFailed   metric.Int64Counter
    sendDuration     metric.Float64Histogram
    queueSize        metric.Int64UpDownCounter
}
```

### TelemetryConfig
```go
type TelemetryConfig struct {
    ServiceName     string
    ServiceVersion  string
    Environment     string
    OTLPEndpoint    string
    OTLPHeaders     map[string]string
    TracingEnabled  bool
    SampleRate      float64
    MetricsEnabled  bool
    Enabled         bool
}
```

## 使用示例

### 基本配置

```go
// 通过配置启用遥测
hub, err := client.New(
    config.WithTelemetry(
        "notifyhub",           // 服务名
        "v1.2.0",              // 版本
        "production",          // 环境
        "http://localhost:4318", // OTLP端点
    ),
)
```

### 环境变量配置

```bash
# 启用遥测
export NOTIFYHUB_TELEMETRY_ENABLED=true
export NOTIFYHUB_SERVICE_NAME="notifyhub"
export NOTIFYHUB_SERVICE_VERSION="v1.2.0"
export NOTIFYHUB_ENVIRONMENT="production"

# OTLP配置
export NOTIFYHUB_OTLP_ENDPOINT="http://jaeger:4318"
export NOTIFYHUB_OTLP_AUTH="Bearer your-token"

# 追踪配置
export NOTIFYHUB_TRACING_ENABLED=true
export NOTIFYHUB_SAMPLE_RATE=0.1  # 10%采样

# 指标配置
export NOTIFYHUB_METRICS_ENABLED=true
```

```go
// 使用环境变量配置
hub, err := client.New(
    config.WithDefaults(), // 自动加载遥测配置
)
```

## 追踪功能

### 自动追踪

系统自动为关键操作创建追踪Span：

```go
// Hub.SendSync() 自动创建追踪
ctx, span := telemetry.TraceMessageSend(ctx, messageID, "sync", targetCount)
defer span.End()

// Hub.SendAsync() 自动创建追踪
ctx, span := telemetry.TraceMessageEnqueue(ctx, messageID, "async")
defer span.End()
```

### 手动追踪

```go
// 获取遥测提供者
telemetry := hub.GetTelemetryProvider()

// 创建自定义操作追踪
ctx, span := telemetry.TraceOperation(ctx, "custom.operation",
    attribute.String("operation.type", "batch_send"),
    attribute.Int("batch.size", len(messages)),
)
defer span.End()

// 执行业务逻辑
results, err := performBatchOperation(ctx, messages)

// 设置追踪结果
if err != nil {
    telemetry.SetSpanError(span, err)
} else {
    telemetry.SetSpanSuccess(span)
}
```

### 追踪属性

系统自动添加标准追踪属性：

```go
// 消息发送追踪属性
attributes := []attribute.KeyValue{
    attribute.String("notifyhub.message.id", messageID),
    attribute.String("notifyhub.platform", platform),
    attribute.Int("notifyhub.targets.count", targetCount),
    attribute.String("notifyhub.operation", "send"),
}

// 服务资源属性
resource.WithAttributes(
    semconv.ServiceName(config.ServiceName),
    semconv.ServiceVersion(config.ServiceVersion),
    semconv.DeploymentEnvironment(config.Environment),
)
```

## 指标收集

### 内置指标

系统自动收集以下指标：

#### 计数器 (Counters)
```go
// 消息发送成功计数
notifyhub_messages_sent_total{platform="feishu",status="success"} 150

// 消息入队计数
notifyhub_messages_enqueued_total{queue_type="async"} 89

// 消息失败计数
notifyhub_messages_failed_total{platform="email",error_type="timeout"} 3
```

#### 直方图 (Histograms)
```go
// 发送耗时分布
notifyhub_send_duration_seconds{platform="feishu",status="success"}
  bucket{le="0.1"} 45
  bucket{le="0.5"} 120
  bucket{le="1.0"} 148
  bucket{le="+Inf"} 150
```

#### 上下计数器 (UpDown Counters)
```go
// 队列大小
notifyhub_queue_size{queue_type="memory"} 23
```

### 记录指标

```go
// 记录成功发送
telemetry.RecordMessageSent(ctx, "feishu", duration)

// 记录失败发送
telemetry.RecordMessageFailed(ctx, "email", duration, "timeout")

// 记录消息入队
telemetry.RecordMessageEnqueued(ctx, "async")

// 更新队列大小
telemetry.UpdateQueueSize(ctx, "memory", 25)
```

## 监控集成

### Jaeger集成

```yaml
# docker-compose.yml
version: '3.8'
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "4317:4317"    # OTLP gRPC
      - "4318:4318"    # OTLP HTTP
    environment:
      - COLLECTOR_OTLP_ENABLED=true

  notifyhub:
    environment:
      - NOTIFYHUB_TELEMETRY_ENABLED=true
      - NOTIFYHUB_OTLP_ENDPOINT=http://jaeger:4318
```

### Prometheus集成

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'notifyhub'
    static_configs:
      - targets: ['notifyhub:8080']
    metrics_path: /metrics
```

### Grafana仪表盘

```json
{
  "dashboard": {
    "title": "NotifyHub监控",
    "panels": [
      {
        "title": "消息发送成功率",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(notifyhub_messages_sent_total[5m]) / (rate(notifyhub_messages_sent_total[5m]) + rate(notifyhub_messages_failed_total[5m]))"
          }
        ]
      },
      {
        "title": "发送耗时分布",
        "type": "histogram",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(notifyhub_send_duration_seconds_bucket[5m]))"
          }
        ]
      }
    ]
  }
}
```

## 高级配置

### 采样策略

```go
// 配置采样率
config := &config.TelemetryConfig{
    SampleRate: 0.1,  // 10%采样
}

// 基于TraceID的采样
sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1))

// 自定义采样器
customSampler := func(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
    // 高优先级消息总是采样
    if priority, ok := parameters.Attributes["message.priority"]; ok {
        if priority.AsInt64() >= 4 {
            return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
        }
    }

    // 其他消息使用默认采样率
    return sdktrace.TraceIDRatioBased(0.01)(parameters)
}
```

### 资源标识

```go
// 自定义资源属性
resource, err := resource.New(context.Background(),
    resource.WithAttributes(
        semconv.ServiceName("notifyhub"),
        semconv.ServiceVersion("v1.2.0"),
        semconv.DeploymentEnvironment("production"),
        attribute.String("service.instance.id", instanceID),
        attribute.String("host.name", hostname),
        attribute.String("k8s.pod.name", podName),
    ),
)
```

### 导出器配置

```go
// OTLP HTTP导出器
exporter, err := otlptrace.New(context.Background(),
    otlptracehttp.NewClient(
        otlptracehttp.WithEndpoint("https://api.honeycomb.io"),
        otlptracehttp.WithHeaders(map[string]string{
            "x-honeycomb-team": "your-api-key",
        }),
        otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
    ),
)

// 批量处理配置
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter,
        sdktrace.WithMaxExportBatchSize(512),
        sdktrace.WithBatchTimeout(5*time.Second),
        sdktrace.WithMaxExportBatchSize(2048),
    ),
)
```

## 性能优化

### 批量导出

```go
// 配置批量导出参数
batchSpanProcessor := sdktrace.NewBatchSpanProcessor(
    exporter,
    sdktrace.WithMaxExportBatchSize(512),     // 批量大小
    sdktrace.WithBatchTimeout(5*time.Second), // 批量超时
    sdktrace.WithExportTimeout(30*time.Second), // 导出超时
)
```

### 内存管理

```go
// 限制内存使用
spanLimits := sdktrace.SpanLimits{
    AttributeValueLengthLimit:   1000,  // 属性值长度限制
    AttributeCountLimit:         50,    // 属性数量限制
    EventCountLimit:            100,    // 事件数量限制
    LinkCountLimit:             10,     // 链接数量限制
}

tp := sdktrace.NewTracerProvider(
    sdktrace.WithSpanLimits(spanLimits),
)
```

### 异步处理

```go
// 非阻塞遥测记录
go func() {
    telemetry.RecordMessageSent(context.Background(), platform, duration)
}()

// 使用缓冲通道
type AsyncTelemetry struct {
    events chan TelemetryEvent
}

func (at *AsyncTelemetry) RecordAsync(event TelemetryEvent) {
    select {
    case at.events <- event:
        // 成功入队
    default:
        // 队列满，丢弃事件
    }
}
```

## 故障排除

### 常见问题

1. **追踪数据缺失**
   ```go
   // 检查配置
   if !telemetryConfig.TracingEnabled {
       log.Warn("追踪功能未启用")
   }

   // 检查采样率
   if telemetryConfig.SampleRate == 0 {
       log.Warn("采样率为0，不会生成追踪数据")
   }
   ```

2. **指标导出失败**
   ```go
   // 检查端点连通性
   resp, err := http.Get(telemetryConfig.OTLPEndpoint + "/v1/traces")
   if err != nil {
       log.Error("OTLP端点不可达: %v", err)
   }
   ```

3. **性能影响**
   ```go
   // 监控遥测开销
   start := time.Now()
   telemetry.RecordMessageSent(ctx, platform, duration)
   overhead := time.Since(start)

   if overhead > 1*time.Millisecond {
       log.Warn("遥测记录耗时过长: %v", overhead)
   }
   ```

### 调试配置

```go
// 启用详细日志
os.Setenv("OTEL_LOG_LEVEL", "debug")

// 导出到控制台（测试用）
import "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"

consoleExporter, err := stdouttrace.New(
    stdouttrace.WithPrettyPrint(),
)
```

## 最佳实践

### 1. 追踪粒度

```go
// ✅ 合适的追踪粒度
ctx, span := tracer.Start(ctx, "hub.send_message")
defer span.End()

// ❌ 过细的追踪粒度
ctx, span := tracer.Start(ctx, "string.format")
defer span.End()
```

### 2. 属性管理

```go
// ✅ 有意义的属性
span.SetAttributes(
    attribute.String("message.id", msgID),
    attribute.String("platform", "feishu"),
    attribute.Int("targets.count", len(targets)),
)

// ❌ 敏感信息
span.SetAttributes(
    attribute.String("user.email", email), // 可能包含敏感信息
)
```

### 3. 错误处理

```go
// ✅ 完整的错误信息
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    span.SetAttributes(
        attribute.String("error.type", errorType),
        attribute.String("error.message", err.Error()),
    )
}
```

### 4. 资源清理

```go
// 确保遥测提供者正确关闭
func (h *Hub) Stop() error {
    // 其他清理...

    if h.telemetry != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        if err := h.telemetry.Shutdown(ctx); err != nil {
            log.Error("遥测提供者关闭失败: %v", err)
        }
    }

    return nil
}
```

## 文件说明

- `telemetry.go` - 核心遥测提供者实现，包含追踪和指标功能
- `config.go` - 遥测配置结构和默认值（如存在）
- `metrics.go` - 指标定义和记录函数（如存在）
- `tracing.go` - 追踪辅助函数和span管理（如存在）