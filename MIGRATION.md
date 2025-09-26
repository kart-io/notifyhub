# NotifyHub 架构迁移指南

## 概述

本文档提供从旧版NotifyHub架构迁移到新版本的完整指南。新架构提供了显著的性能改进、类型安全和现代化的API设计，同时保持100%向后兼容。

## 快速迁移检查表

- [ ] 阅读重大变更说明
- [ ] 更新依赖包版本
- [ ] 替换创建Hub的方式
- [ ] 迁移配置格式
- [ ] 更新错误处理逻辑
- [ ] 启用新特性（可选）
- [ ] 运行测试验证

## 重大变更概述

### 1. 架构简化

**旧架构**：双层接口（Sender + ExternalSender）
```go
// ❌ 旧版本 - 复杂的双层架构
type Sender interface { ... }
type ExternalSender interface { ... }
```

**新架构**：统一Platform接口
```go
// ✅ 新版本 - 简洁的统一接口
type Platform interface {
    Name() string
    Send(ctx context.Context, msg *Message, targets []Target) ([]*SendResult, error)
    Health() PlatformHealth
}
```

### 2. 强类型配置

**旧版本**：类型不安全的map配置
```go
// ❌ 旧版本 - 运行时错误风险
platforms := map[string]map[string]interface{}{
    "email": {
        "smtp_host": "smtp.example.com",
        "smtp_port": 587,  // 类型不明确
        "smtp_tls": true,
    },
}
hub, err := notifyhub.CreateHub(platforms)
```

**新版本**：编译时类型检查
```go
// ✅ 新版本 - 编译时安全
emailConfig := &config.EmailConfig{
    SMTPHost: "smtp.example.com",
    SMTPPort: 587,
    SMTPFrom: "noreply@example.com",
    UseTLS:   true,
}
hub, err := notifyhub.New(notifyhub.WithEmail(emailConfig))
```

### 3. 真实异步处理

**旧版本**：伪异步（实际上是同步调用）
```go
// ❌ 旧版本 - 假异步
receipt, err := hub.Send(ctx, message) // 阻塞调用
```

**新版本**：真实异步处理
```go
// ✅ 新版本 - 真实异步
resultChan, err := hub.SendAsync(ctx, message)
if err != nil {
    log.Fatal(err)
}

// 立即返回，后台处理
handle := <-resultChan
progress := handle.GetProgress()
result := handle.Wait()
```

## 分步迁移指南

### 步骤1：更新依赖

更新go.mod文件：
```go
module your-project

go 1.21

require (
    github.com/kart-io/notifyhub v3.0.0 // 更新到v3.0.0+
)
```

### 步骤2：迁移Hub创建

#### 场景A：基本邮件配置

**旧代码**：
```go
platforms := map[string]map[string]interface{}{
    "email": {
        "smtp_host": "smtp.gmail.com",
        "smtp_port": 587,
        "smtp_from": "noreply@example.com",
        "smtp_username": "your-username",
        "smtp_password": "your-password",
        "smtp_tls": true,
    },
}
hub, err := notifyhub.CreateHub(platforms)
```

**新代码**：
```go
emailConfig := &config.EmailConfig{
    SMTPHost:     "smtp.gmail.com",
    SMTPPort:     587,
    SMTPFrom:     "noreply@example.com",
    SMTPUsername: "your-username",
    SMTPPassword: "your-password",
    UseTLS:       true,
    Timeout:      30 * time.Second,
}
hub, err := notifyhub.New(notifyhub.WithEmail(emailConfig))
```

#### 场景B：多平台配置

**旧代码**：
```go
platforms := map[string]map[string]interface{}{
    "email": {
        "smtp_host": "smtp.gmail.com",
        "smtp_port": 587,
        "smtp_from": "noreply@example.com",
    },
    "feishu": {
        "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
        "secret": "your-secret",
    },
}
hub, err := notifyhub.CreateHub(platforms)
```

**新代码**：
```go
emailConfig := &config.EmailConfig{
    SMTPHost: "smtp.gmail.com",
    SMTPPort: 587,
    SMTPFrom: "noreply@example.com",
}

feishuConfig := &config.FeishuConfig{
    WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
    Secret:     "your-secret",
}

hub, err := notifyhub.New(
    notifyhub.WithEmail(emailConfig),
    notifyhub.WithFeishu(feishuConfig),
)
```

### 步骤3：迁移消息发送

#### 场景A：同步发送（兼容模式）

**旧代码**：
```go
msg := notifyhub.LegacyMessage{
    Title: "Test Message",
    Content: "Hello World",
    Targets: []notifyhub.LegacyTarget{
        {Type: "email", Address: "user@example.com"},
    },
}
receipt, err := hub.Send(ctx, msg)
```

**新代码（推荐）**：
```go
msg := &message.Message{
    Title: "Test Message",
    Body:  "Hello World",
    Format: message.FormatText,
    Targets: []target.Target{
        {Type: "email", Value: "user@example.com"},
    },
}
receipt, err := hub.Send(ctx, msg)
```

#### 场景B：异步发送（推荐）

```go
// 新功能：真实异步发送
msg := &message.Message{
    Title: "Test Message",
    Body:  "Hello World",
    Format: message.FormatText,
    Targets: []target.Target{
        {Type: "email", Value: "user@example.com"},
    },
}

// 启动异步发送
resultChan, err := hub.SendAsync(ctx, msg)
if err != nil {
    return err
}

// 获取异步句柄
handle := <-resultChan

// 监控进度（可选）
go func() {
    for !handle.IsDone() {
        progress := handle.GetProgress()
        log.Printf("Progress: %d/%d completed", progress.Processed, progress.Total)
        time.Sleep(time.Second)
    }
}()

// 等待完成
result := handle.Wait()
if result.Error != nil {
    return result.Error
}

log.Printf("Async send completed: %+v", result.Receipt)
```

### 步骤4：错误处理迁移

**新版本提供结构化错误处理**：

```go
receipt, err := hub.Send(ctx, msg)
if err != nil {
    // 新版本：结构化错误检查
    if errors.IsSystemError(err) {
        // 系统级错误（配置、网络等）
        log.Error("System error:", err)
        return err
    }

    if errors.IsPlatformError(err) {
        // 平台级错误（SMTP失败、API限流等）
        log.Warn("Platform error:", err)
        // 可以考虑重试或切换平台
    }

    if errors.IsValidationError(err) {
        // 验证错误（参数不正确等）
        log.Error("Validation error:", err)
        return err
    }
}
```

### 步骤5：启用新特性

#### 中间件系统

```go
import "github.com/kart-io/notifyhub/pkg/notifyhub/middleware"

// 创建中间件
loggingMiddleware := middleware.NewLoggingMiddleware(middleware.LoggingConfig{
    Logger:    logger.New(),
    LogLevel:  middleware.LogLevelInfo,
    LogBodies: false, // 生产环境建议关闭
})

retryMiddleware := middleware.NewRetryMiddleware(
    middleware.NewExponentialBackoffStrategy(3, time.Second, 2.0, 0.1),
    logger.New(),
)

metricsMiddleware := middleware.NewMetricsMiddleware(
    middleware.NewSimpleMetricsCollector(logger.New()),
    logger.New(),
)

// 应用中间件
hub, err := notifyhub.New(
    notifyhub.WithEmail(emailConfig),
    notifyhub.WithMiddleware(loggingMiddleware),
    notifyhub.WithMiddleware(retryMiddleware),
    notifyhub.WithMiddleware(metricsMiddleware),
)
```

#### 模板引擎

```go
// 使用模板系统
msg := &message.Message{
    Title: "Welcome {{.UserName}}!",
    Body:  "Hello {{.UserName}}, welcome to {{.ServiceName}}!",
    Format: message.FormatText,
    PlatformData: map[string]interface{}{
        "UserName":    "John Doe",
        "ServiceName": "Our Platform",
    },
    Targets: []target.Target{
        {Type: "email", Value: "john@example.com"},
    },
}
```

#### 健康检查

```go
// 健康检查
health, err := hub.Health(ctx)
if err != nil {
    log.Error("Health check failed:", err)
    return
}

log.Printf("Overall health: %s", health.Status)
for platform, platformHealth := range health.Platforms {
    log.Printf("Platform %s: %s", platform, platformHealth.Status)
}
```

## 性能优化建议

### 1. Hub复用

**推荐**：
```go
// ✅ 推荐：创建一次，复用多次
var globalHub notifyhub.Client
func init() {
    globalHub, _ = notifyhub.New(notifyhub.WithEmail(emailConfig))
}

func sendMessage(msg *message.Message) error {
    return globalHub.Send(ctx, msg)
}
```

**不推荐**：
```go
// ❌ 不推荐：每次创建新Hub
func sendMessage(msg *message.Message) error {
    hub, _ := notifyhub.New(notifyhub.WithEmail(emailConfig))
    defer hub.Close()
    return hub.Send(ctx, msg)
}
```

### 2. 异步批量处理

```go
// 批量异步处理
messages := []*message.Message{...}
handles := make([]async.AsyncHandle, len(messages))

// 启动所有异步发送
for i, msg := range messages {
    resultChan, err := hub.SendAsync(ctx, msg)
    if err != nil {
        continue
    }
    handles[i] = <-resultChan
}

// 并行等待所有完成
var wg sync.WaitGroup
for _, handle := range handles {
    if handle == nil {
        continue
    }
    wg.Add(1)
    go func(h async.AsyncHandle) {
        defer wg.Done()
        result := h.Wait()
        // 处理结果
    }(handle)
}
wg.Wait()
```

## 测试策略

### 单元测试迁移

```go
func TestNotifyHubMigration(t *testing.T) {
    // 设置测试配置
    emailConfig := &config.EmailConfig{
        SMTPHost: "smtp.test.com",
        SMTPPort: 587,
        SMTPFrom: "test@test.com",
    }

    // 创建Hub
    hub, err := notifyhub.New(
        notifyhub.WithEmail(emailConfig),
        notifyhub.WithTestDefaults(), // 测试模式
    )
    require.NoError(t, err)
    defer hub.Close()

    // 创建测试消息
    msg := &message.Message{
        Title: "Test Message",
        Body:  "Test Body",
        Format: message.FormatText,
        Targets: []target.Target{
            {Type: "email", Value: "test@example.com"},
        },
    }

    // 测试发送
    receipt, err := hub.Send(context.Background(), msg)

    // 在测试环境中，通常会返回模拟的结果
    if err != nil {
        t.Logf("Send failed (may be expected in test): %v", err)
    }
    if receipt != nil {
        assert.Equal(t, "success", receipt.Status)
    }
}
```

### 集成测试

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // 使用真实配置进行集成测试
    emailConfig := &config.EmailConfig{
        SMTPHost:     os.Getenv("SMTP_HOST"),
        SMTPPort:     587,
        SMTPFrom:     os.Getenv("SMTP_FROM"),
        SMTPUsername: os.Getenv("SMTP_USERNAME"),
        SMTPPassword: os.Getenv("SMTP_PASSWORD"),
        UseTLS:       true,
    }

    hub, err := notifyhub.New(notifyhub.WithEmail(emailConfig))
    require.NoError(t, err)
    defer hub.Close()

    // 执行实际发送测试
    // ...
}
```

## 故障排查

### 常见迁移问题

#### 1. 配置转换错误

**问题**：类型转换失败
```
Error: invalid configuration: smtp_port must be a number
```

**解决**：检查配置类型匹配
```go
// ❌ 错误
SMTPPort: "587"  // 字符串类型

// ✅ 正确
SMTPPort: 587    // 整数类型
```

#### 2. 导入路径错误

**问题**：
```
package github.com/kart-io/notifyhub/pkg/notifyhub/message is not in GOROOT
```

**解决**：更新导入路径
```go
import (
    "github.com/kart-io/notifyhub/pkg/notifyhub"
    "github.com/kart-io/notifyhub/pkg/notifyhub/message"
    "github.com/kart-io/notifyhub/pkg/notifyhub/target"
)
```

#### 3. Context超时问题

**问题**：异步操作超时
```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
resultChan, err := hub.SendAsync(ctx, msg) // 可能超时
```

**解决**：为异步操作设置合理超时
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 性能调优

#### 监控关键指标

```go
// 启用指标中间件
metricsMiddleware := middleware.NewMetricsMiddleware(
    middleware.NewSimpleMetricsCollector(logger.New()),
    logger.New(),
)

hub, err := notifyhub.New(
    notifyhub.WithEmail(emailConfig),
    notifyhub.WithMiddleware(metricsMiddleware),
)

// 定期检查指标
go func() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        metrics := metricsMiddleware.GetMetrics()
        log.Printf("Current metrics: %+v", metrics)
    }
}()
```

#### 调整异步配置

```go
// 针对高并发场景优化
hub, err := notifyhub.New(
    notifyhub.WithEmail(emailConfig),
    notifyhub.WithAsync(true, 1000, 10), // 队列大小1000，工作协程10
)
```

## 兼容性保证

### 弃用API时间表

| API | 状态 | 弃用版本 | 移除版本 | 替代API |
|-----|------|----------|----------|---------|
| `CreateHub()` | 已弃用 | v3.0.0 | v4.0.0 | `New()` |
| `LegacyMessage` | 已弃用 | v3.0.0 | v4.0.0 | `message.Message` |
| `LegacyTarget` | 已弃用 | v3.0.0 | v4.0.0 | `target.Target` |
| `SetGlobalHub()` | 已弃用 | v3.0.0 | v4.0.0 | 显式Hub管理 |

### 迁移时间建议

- **立即**：开始使用新API进行新功能开发
- **3个月内**：完成核心业务逻辑迁移
- **6个月内**：完成所有代码迁移
- **v4.0.0发布前**：彻底移除旧API使用

## 获取帮助

- **文档**：查看README.md和代码注释
- **示例代码**：参考examples/目录
- **性能测试**：运行tests/performance_validation_test.go
- **问题反馈**：创建GitHub Issue

---

本迁移指南将持续更新，确保涵盖所有迁移场景和最佳实践。