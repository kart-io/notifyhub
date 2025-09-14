# Client Package

## 功能概述

Client包是NotifyHub的核心协调器，提供统一的消息发送接口和生命周期管理。Hub作为系统的中央控制器，协调所有子组件完成消息的路由、模板渲染、队列处理和平台发送。

## 核心组件

### Hub结构体
- **核心协调器**：管理所有notifiers、队列、模板引擎和路由引擎
- **生命周期管理**：提供Start()和Stop()方法进行优雅启动和关闭
- **并发安全**：使用sync.RWMutex保护共享状态

### 主要功能

1. **消息发送**
   - `Send()` - 统一发送接口（支持同步/异步）
   - `SendSync()` - 同步发送，立即返回结果
   - `SendAsync()` - 异步发送，消息入队等待处理
   - `SendBatch()` - 批量发送，支持批量优化

2. **便捷方法**
   - `SendText()` - 发送纯文本消息
   - `SendAlert()` - 发送告警消息（高优先级+重试）
   - `SendWithTemplate()` - 使用模板发送消息

3. **监控与健康检查**
   - `GetMetrics()` - 获取发送统计指标
   - `GetHealth()` - 获取系统健康状态
   - 内置健康检查协程

## 使用示例

### 基本使用

```go
// 创建Hub实例
hub, err := client.New(
    config.WithFeishu("webhook-url", "secret"),
    config.WithEmail("smtp.gmail.com", 587, "user", "pass", "from@example.com", true, 30*time.Second),
    config.WithQueue("memory", 1000, 4),
)
if err != nil {
    log.Fatal(err)
}

// 启动服务
ctx := context.Background()
if err := hub.Start(ctx); err != nil {
    log.Fatal(err)
}
defer hub.Stop()

// 发送消息
message := NewMessage().
    Title("Test Message").
    Body("This is a test").
    Email("user@example.com").
    Build()

results, err := hub.Send(ctx, message, nil)
```

### 异步发送

```go
// 异步发送（消息入队）
taskID, err := hub.SendAsync(ctx, message, &Options{
    Timeout: 30 * time.Second,
    Retry:   true,
})
```

### 批量发送

```go
messages := []*notifiers.Message{message1, message2, message3}
results, err := hub.SendBatch(ctx, messages, &Options{
    Async: true,  // 支持异步批量
})
```

## 配置选项

Hub通过functional options模式进行配置：

```go
hub, err := client.New(
    config.WithDefaults(),                    // 从环境变量加载默认配置
    config.WithQueue("memory", 5000, 8),     // 队列配置
    config.WithTelemetry("service", "v1.2.0", "prod", "http://otlp:4318"), // 遥测
    config.WithDefaultLogger(logger.Info),   // 日志级别
)
```

## 错误处理

Hub支持部分失败处理：

```go
results, err := hub.Send(ctx, message, nil)
if err != nil {
    // 检查部分成功的情况
    if results != nil {
        for _, result := range results {
            if !result.Success {
                log.Printf("Platform %s failed: %s", result.Platform, result.Error)
            } else {
                log.Printf("Platform %s succeeded", result.Platform)
            }
        }
    }
}
```

## 架构集成

- **路由引擎**：自动根据消息特征选择发送平台
- **模板引擎**：处理变量替换和格式转换
- **队列系统**：异步处理和重试机制
- **监控系统**：指标收集和健康检查
- **遥测系统**：分布式追踪和指标上报

## 生命周期管理

```go
// 优雅启动
if err := hub.Start(ctx); err != nil {
    return fmt.Errorf("failed to start hub: %w", err)
}

// 优雅关闭（30秒超时）
if err := hub.Stop(); err != nil {
    log.Printf("shutdown error: %v", err)
}
```

## 文件说明

- `hub.go` - Hub核心实现，包含所有发送逻辑和生命周期管理
- `message.go` - 消息构建器，提供链式API构建消息
- `options.go` - 发送选项定义
- `hub_test.go` - 单元测试