# NotifyHub - 统一通知API

NotifyHub 提供了一个简洁、统一的API来发送跨平台通知。支持邮件、飞书、短信、Slack等多种通知平台。

## ✨ 特性

- 🎯 **统一入口**: 一个包，一个API，简单易用
- 🔧 **类型安全**: 编译时类型检查，减少运行时错误
- 🚀 **高性能**: 异步处理，支持批量发送
- 🔌 **插件化**: 易于扩展新的通知平台
- 🛠️ **调试友好**: 内置调试工具和性能监控
- ⚙️ **配置灵活**: 支持多种配置方式和环境预设

## 🚀 快速开始

### 安装

```bash
go get github.com/kart-io/notifyhub
```

### 基础使用

```go
package main

import (
    "context"
    "log"

    "github.com/kart-io/notifyhub"
)

func main() {
    // 1. 创建客户端 - 统一入口
    client, err := notifyhub.New(
        notifyhub.WithFeishu("https://open.feishu.cn/webhook/xxx", "secret"),
        notifyhub.WithEmail("smtp.example.com", 587, "user", "pass", "noreply@company.com"),
        notifyhub.WithDevelopment(), // 开发环境预设
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // 2. 发送消息 - 流畅的API
    result, err := client.Send(context.Background()).
        Title("系统维护通知").
        Body("系统将在今晚22:00进行维护").
        Priority(3).
        ToEmail("admin@company.com").
        ToFeishu("maintenance-alerts").
        Execute()

    if err != nil {
        log.Printf("发送失败: %v", err)
    } else {
        log.Printf("发送成功: %d条成功, %d条失败", result.Sent, result.Failed)
    }
}
```

## 📖 详细使用指南

### 1. 客户端配置

#### 基础配置

```go
// 基础平台配置
client, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithEmail("smtp.server.com", 587, "user", "pass", "from@company.com"),
    notifyhub.WithSMS("provider", "api-key", "api-secret"),
    notifyhub.WithSlack("webhook-url", "token"),
)
```

#### 高级配置

```go
// 高级邮件配置
client, err := notifyhub.New(
    notifyhub.WithEmailAdvanced(
        "smtp.company.com", 587,
        notifyhub.EmailAuth{
            Username: "noreply@company.com",
            Password: "secure_password",
        },
        notifyhub.EmailConfig{
            From:    "系统通知 <noreply@company.com>",
            UseTLS:  true,
            Timeout: 30 * time.Second,
        },
    ),
    notifyhub.WithRetryPolicy(5, 2*time.Second, true),
    notifyhub.WithRateLimit(50, 100, time.Minute),
)
```

#### 环境预设

```go
// 开发环境
client, err := notifyhub.New(
    notifyhub.WithFeishu("dev-webhook", "dev-secret"),
    notifyhub.WithDevelopment(),
)

// 生产环境
client, err := notifyhub.New(
    notifyhub.WithFeishu("prod-webhook", "prod-secret"),
    notifyhub.WithEmailService("gmail", "noreply@company.com", auth),
    notifyhub.WithProduction(),
)

// 测试环境
client, err := notifyhub.New(
    notifyhub.WithTesting(),
)
```

### 2. 消息发送

#### 基础消息

```go
result, err := client.Send(ctx).
    Title("标题").
    Body("消息内容").
    Priority(3).
    ToEmail("user@company.com").
    ToFeishu("alerts-group").
    Execute()
```

#### 告警消息

```go
result, err := client.Alert(ctx).
    Title("🚨 紧急告警").
    Body("数据库连接异常").
    Critical().
    Metadata("service", "database").
    ToEmail("oncall@company.com").
    ToFeishu("critical-alerts").
    Execute()
```

#### 通知消息

```go
result, err := client.Notification(ctx).
    Title("📊 每日报告").
    Body("今日系统运行正常").
    Normal().
    ToEmail("team@company.com").
    Execute()
```

### 3. 目标管理

#### 类型安全的目标

```go
// 使用类型安全的目标创建函数
result, err := client.Send(ctx).
    Title("消息").
    ToTargets(
        notifyhub.Email("admin@company.com"),
        notifyhub.EmailWithName("admin@company.com", "管理员"),
        notifyhub.Feishu("alerts-group"),
        notifyhub.SMS("+8613800138000"),
        notifyhub.Slack("#alerts"),
    ).
    Execute()
```

#### 目标表达式

```go
// 使用简洁的目标表达式
result, err := client.Send(ctx).
    Title("消息").
    ToExpressions(
        "email:admin@company.com",
        "feishu:alerts-group",
        "sms:+8613800138000",
        "slack:#alerts",
    ).
    Execute()
```

#### 预定义目标组

```go
// 发送到预定义的目标组
result, err := client.Send(ctx).
    Title("消息").
    ToGroup("admins").     // 管理员组
    ToGroup("oncall").     // 值班组
    ToGroup("critical").   // 紧急响应组
    Execute()
```

#### 智能目标解析

```go
// 自动检测目标类型
result, err := client.Send(ctx).
    Title("消息").
    ToSmart(
        "admin@company.com",    // 自动识别为邮箱
        "+8613800138000",       // 自动识别为手机号
        "#alerts",              // 自动识别为Slack频道
        "alerts-group",         // 默认为飞书群组
    ).
    Execute()
```

### 4. 模板和变量

```go
result, err := client.Send(ctx).
    Template("user-welcome").
    Title("欢迎 {{.username}} 加入我们！").
    Body("Hi {{.username}}, 欢迎加入 {{.company}}！").
    Variable("username", "张三").
    Variable("company", "科技公司").
    Variables(map[string]interface{}{
        "activation_url": "https://company.com/activate/xxx",
        "expires_at":     time.Now().Add(24 * time.Hour),
    }).
    ToEmail("zhangsan@company.com").
    Execute()
```

### 5. 延迟发送

```go
// 延迟指定时间
result, err := client.Send(ctx).
    Title("定时提醒").
    Body("这是一条延迟发送的消息").
    DelayBy(5 * time.Minute).
    ToEmail("user@company.com").
    Execute()

// 延迟到指定时间
result, err := client.Send(ctx).
    Title("定时提醒").
    DelayUntil(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)).
    ToEmail("user@company.com").
    Execute()
```

### 6. 调试和监控

#### 调试模式

```go
// 创建调试客户端
debugClient := client.Debug()

// 模拟运行
dryResult, err := debugClient.Send(ctx).
    Title("测试消息").
    ToEmail("test@company.com").
    DryRun()

fmt.Printf("模拟运行结果: Valid=%v, Targets=%d\n",
    dryResult.Valid, len(dryResult.Targets))

// 分析消息配置
builder := debugClient.Send(ctx).Title("测试").ToEmail("test@company.com")
analysis := builder.Analyze()
fmt.Printf("分析结果: %d个问题, %d个建议\n",
    len(analysis.Issues), len(analysis.Suggestions))
```

#### 追踪和指标

```go
// 执行发送并追踪
result, err := debugClient.Send(ctx).
    Title("测试消息").
    ToEmail("test@company.com").
    Execute()

// 打印追踪信息
debugClient.PrintTrace()

// 打印性能指标
debugClient.PrintMetrics()

// 获取详细追踪数据
traces := debugClient.Trace()
for _, trace := range traces {
    fmt.Printf("%s: %s\n", trace.Type, trace.Message)
}
```

### 7. 错误处理

```go
result, err := client.Send(ctx).
    Title("测试消息").
    ToEmail("invalid-email").
    Execute()

if err != nil {
    log.Printf("发送失败: %v", err)
    return
}

// 检查每个目标的发送结果
for _, targetResult := range result.Results {
    if targetResult.Status == notifyhub.StatusFailed {
        if targetResult.Error != nil {
            log.Printf("目标 %s 发送失败: %s (可重试: %v)",
                targetResult.Target.String(),
                targetResult.Error.Message,
                targetResult.Error.Retryable)
        }
    }
}
```

### 8. 健康检查和监控

```go
// 健康检查
health := client.Health()
if !health.Healthy {
    log.Printf("系统不健康: %+v", health.Details)
}

// 优雅关闭
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := client.Shutdown(ctx); err != nil {
    log.Printf("关闭失败: %v", err)
}
```

## 🔧 配置选项完整列表

### 平台配置

- `WithFeishu(webhook, secret)` - 配置飞书
- `WithEmail(host, port, user, pass, from)` - 配置邮件
- `WithEmailAdvanced(host, port, auth, config)` - 高级邮件配置
- `WithSMS(provider, apiKey, apiSecret)` - 配置短信
- `WithSlack(webhook, token)` - 配置Slack

### 队列配置

- `WithMemoryQueue(capacity, concurrency)` - 内存队列
- `WithRedisQueue(addr, password, db, capacity, concurrency)` - Redis队列
- `WithQueue(type, capacity, concurrency)` - 通用队列配置

### 重试和限流

- `WithRetryPolicy(maxAttempts, backoff, jitter)` - 重试策略
- `WithSimpleRetry(maxAttempts)` - 简单重试
- `WithRateLimit(rate, burst, window)` - 限流配置

### 环境预设

- `WithProduction()` - 生产环境预设
- `WithDevelopment()` - 开发环境预设
- `WithTesting()` - 测试环境预设
- `WithDefaults()` - 默认配置（支持环境变量）

### 高级配置

- `WithPlatforms(...platforms)` - 批量平台配置
- `WithLogger(logger)` - 自定义日志器
- `WithChatPlatform(type, webhook, secret)` - 通用聊天平台
- `WithEmailService(provider, from, auth)` - 通用邮件服务
- `WithSMSService(provider, auth)` - 通用短信服务

## 📋 API 参考

### 客户端方法

```go
type Client interface {
    Send(ctx context.Context) *SendBuilder
    Alert(ctx context.Context) *AlertBuilder
    Notification(ctx context.Context) *NotificationBuilder
    Configure(options ...Option) error
    Health() HealthStatus
    Shutdown(ctx context.Context) error
    Debug() *DebugClient
}
```

### 构建器方法

```go
type SendBuilder interface {
    // 消息内容
    Title(string) *SendBuilder
    Body(string) *SendBuilder
    Priority(int) *SendBuilder
    Format(Format) *SendBuilder

    // 模板和变量
    Template(string) *SendBuilder
    Variable(key string, value interface{}) *SendBuilder
    Variables(map[string]interface{}) *SendBuilder

    // 元数据
    Metadata(key, value string) *SendBuilder
    MetadataMap(map[string]string) *SendBuilder

    // 目标设置
    ToEmail(...string) *SendBuilder
    ToFeishu(...string) *SendBuilder
    ToSMS(...string) *SendBuilder
    ToSlack(...string) *SendBuilder
    ToTargets(...TypedTarget) *SendBuilder
    ToGroup(string) *SendBuilder
    ToExpressions(...string) *SendBuilder
    ToSmart(...string) *SendBuilder

    // 延迟发送
    DelayBy(time.Duration) *SendBuilder
    DelayUntil(time.Time) *SendBuilder

    // 执行
    Execute() (*Results, error)
    DryRun() (*DryRunResult, error)
}
```

## 🌟 最佳实践

### 1. 配置管理

```go
// 推荐：使用环境变量配置
client, err := notifyhub.New(
    notifyhub.WithDefaults(),  // 自动读取环境变量
    notifyhub.WithProduction(), // 生产环境优化
)

// 环境变量示例：
// NOTIFYHUB_FEISHU_WEBHOOK_URL=https://...
// NOTIFYHUB_FEISHU_SECRET=...
// NOTIFYHUB_SMTP_HOST=smtp.company.com
// NOTIFYHUB_SMTP_PORT=587
```

### 2. 错误处理

```go
// 推荐：细粒度错误处理
result, err := client.Send(ctx)./*...*/.Execute()
if err != nil {
    // 处理发送错误
    return err
}

// 检查每个目标的结果
for _, tr := range result.Results {
    if tr.Status == notifyhub.StatusFailed && tr.Error.Retryable {
        // 处理可重试的错误
        go retryLater(tr.Target, message)
    }
}
```

### 3. 性能优化

```go
// 推荐：批量发送
messages := []Message{/*...*/}
for _, msg := range messages {
    go func(m Message) {
        client.Send(ctx)./*配置消息*/.Execute()
    }(msg)
}

// 推荐：使用目标组
client.Send(ctx).
    Title("批量通知").
    ToGroup("all-users").  // 预定义的用户组
    Execute()
```

### 4. 监控和调试

```go
// 开发环境：启用调试
if isDevelopment {
    debugClient := client.Debug()
    result, err := debugClient.Send(ctx)./*...*/.Execute()
    debugClient.PrintTrace()
    debugClient.PrintMetrics()
}

// 生产环境：健康检查
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        if !client.Health().Healthy {
            log.Warn("NotifyHub 健康检查失败")
        }
    }
}()
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License