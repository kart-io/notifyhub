# NotifyHub Public API

这是NotifyHub的公共API层，提供统一、稳定的接口供用户使用。

## 三层架构设计

### 1. 公共API层 (`pkg/notifyhub/`)

- **职责**: 提供给用户的统一且稳定的接口
- **特点**:
  - 平台无关的消息构建
  - 统一的配置接口
  - 简化的错误处理
  - 向后兼容保证

### 2. 内部核心逻辑层 (`internal/dispatcher/`, `internal/platform/`)

- **职责**: 负责消息的路由和分发，隔离平台差异
- **特点**:
  - 消息路由和转换
  - 平台适配和抽象
  - 错误处理和重试
  - 性能优化

### 3. 基础服务层 (`internal/queue/`, `internal/logger/`, `internal/config/`)

- **职责**: 提供支撑系统运行的基础模块
- **特点**:
  - 队列管理
  - 日志记录
  - 配置加载
  - 监控和度量

## 核心特性

### 统一的消息API

```go
// 一个Message对象支持所有平台特性
message := notifyhub.NewMessage("Title", "Body").
    // Email特性
    WithCC("cc@example.com").
    WithAttachment("file.pdf", "content").
    // Feishu特性
    WithCard(`{"type": "card"}`).
    WithMentions("user1", "user2").
    // SMS特性
    WithSMSFrom("+1234567890").
    WithSMSFlash()
```

### 平台无关的目标设置

```go
message.ToEmail("user@example.com").        // 明确指定平台
       ToPhone("+1234567890").              // 明确指定平台
       ToUser("user123").                   // 由路由决定平台
       ToGroup("group456")                  // 由路由决定平台
```

### 类型安全的配置

```go
client, err := notifyhub.NewClient(
    notifyhub.WithPlatform("feishu", notifyhub.PlatformConfig{
        Enabled: true,
        Settings: map[string]interface{}{
            "webhook_url": "https://...",
            "secret": "secret",
        },
    }),
    notifyhub.WithQueue("memory", 1000, 4),
    notifyhub.WithTimeout(30*time.Second),
)
```

## 使用示例

### 基本用法

```go
// 创建客户端
client, err := notifyhub.NewClient(
    notifyhub.WithPlatform("email", emailConfig),
    notifyhub.WithPlatform("feishu", feishuConfig),
)
if err != nil {
    log.Fatal(err)
}

// 发送消息
message := notifyhub.NewMessage("Alert", "System down").
    ToEmail("admin@example.com").
    ToFeishuGroup("ops-team")

result, err := client.Send(context.Background(), message)
if err != nil {
    log.Printf("Send failed: %v", err)
}

fmt.Printf("Sent to %d targets successfully\n", result.Successful)
```

### 高级特性

```go
// 模板消息
message := notifyhub.NewMessage("", "").
    WithTemplate("welcome_email").
    WithVariable("user_name", "John").
    WithVariable("login_url", "https://...").
    ToEmail("john@example.com")

// 定时发送
message := notifyhub.NewMessage("Reminder", "Meeting in 1 hour").
    WithDelay(time.Hour).
    ToUser("user123")

// 异步发送
asyncResult, err := client.SendAsync(context.Background(), message)
```

## 迁移指南

从旧API迁移到新API：

### 旧方式（已废弃）

```go
// 旧的platform-specific builders
feishuBuilder := feishu.NewBuilder().
    AtUser("user123").
    ToGroup("group456")

emailBuilder := email.NewBuilder().
    CC("cc@example.com").
    ToEmail("user@example.com")
```

### 新方式（推荐）

```go
// 统一的Message API
message := notifyhub.NewMessage("Title", "Body").
    // Feishu特性
    WithMentions("user123").
    ToFeishuGroup("group456").
    // Email特性
    WithCC("cc@example.com").
    ToEmail("user@example.com")
```

## 优势

1. **去耦合**: 用户代码不再直接依赖特定平台
2. **统一接口**: 一个Message对象支持所有平台特性
3. **类型安全**: 避免map[string]interface{}的使用
4. **易于扩展**: 新增平台不影响用户代码
5. **向后兼容**: 提供迁移路径和兼容层
6. **测试友好**: 更容易进行单元测试和集成测试

## 性能特性

- **惰性转换**: 只在发送时转换平台特定数据
- **智能路由**: 根据目标类型自动选择最佳平台
- **并行发送**: 多平台消息并行处理
- **连接池**: 复用平台连接减少开销
- **批量处理**: 支持批量消息发送（计划中）
