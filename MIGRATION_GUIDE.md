# NotifyHub v2 迁移指南

## 概览

NotifyHub v2 带来了更简洁、更灵活的 API 设计。主要改进包括：

- ✅ **统一的客户端入口**：单一的 `notifyhub.Client` 接口
- ✅ **配置解耦**：无需导入具体平台包即可使用
- ✅ **多种配置方式**：支持代码、YAML、JSON、环境变量
- ✅ **更好的扩展性**：插件式平台架构
- ✅ **更清晰的包结构**：职责分离，易于理解

## 快速开始

### v1 到 v2 的主要变化

#### 1. 客户端创建

**v1 (旧版)**
```go
import (
    "github.com/kart-io/notifyhub/pkg/notifyhub"
    "github.com/kart-io/notifyhub/pkg/platforms/feishu"  // 必须导入
    "github.com/kart-io/notifyhub/pkg/platforms/email"   // 必须导入
)

hub, err := notifyhub.NewHub(
    feishu.WithFeishu("webhook", feishu.WithFeishuSecret("secret")),
    email.WithEmail("smtp.example.com", 587, "from@example.com"),
)
```

**v2 (新版)**
```go
import "github.com/kart-io/notifyhub/pkg/notifyhub"  // 只需导入主包

client, err := notifyhub.New(
    notifyhub.WithPlatform("feishu", map[string]interface{}{
        "webhook": "webhook-url",
        "secret":  "secret",
    }),
    notifyhub.WithEmail("smtp.example.com", 587, "from@example.com"),
)
```

#### 2. 接口变化

| v1 接口 | v2 接口 | 说明 |
|---------|---------|------|
| `Hub` | `Client` | 统一客户端接口 |
| `NewHub()` | `New()` | 简化的构造函数 |
| `hub.Send()` | `client.Send()` | 方法签名相同 |
| `hub.Close()` | `client.Close()` | 方法签名相同 |
| 无 | `client.SendAsync()` | 新增异步发送 |
| 无 | `client.GetPlatformStatus()` | 新增平台状态查询 |

## 详细迁移步骤

### 步骤 1: 更新导入

```go
// 移除平台特定的导入
- import "github.com/kart-io/notifyhub/pkg/platforms/feishu"
- import "github.com/kart-io/notifyhub/pkg/platforms/email"
- import "github.com/kart-io/notifyhub/pkg/platforms/sms"

// 只保留主包导入
import "github.com/kart-io/notifyhub/pkg/notifyhub"
```

### 步骤 2: 更新客户端创建

#### 方法 A: 使用通用配置（推荐）

```go
// v2 通用配置方式
client, err := notifyhub.New(
    notifyhub.WithPlatform("feishu", map[string]interface{}{
        "webhook": os.Getenv("FEISHU_WEBHOOK"),
        "secret":  os.Getenv("FEISHU_SECRET"),
    }),
    notifyhub.WithPlatform("email", map[string]interface{}{
        "host":     "smtp.gmail.com",
        "port":     587,
        "from":     "noreply@example.com",
        "username": os.Getenv("EMAIL_USER"),
        "password": os.Getenv("EMAIL_PASS"),
    }),
)
```

#### 方法 B: 使用便捷函数

```go
// v2 便捷函数（仍然不需要导入平台包）
client, err := notifyhub.New(
    notifyhub.WithFeishu(webhook, secret),
    notifyhub.WithEmail(host, port, from),
    notifyhub.WithSMS("twilio", apiKey),
    notifyhub.WithSlack(token),
)
```

#### 方法 C: 使用配置文件

```yaml
# config/notifyhub.yaml
platforms:
  feishu:
    webhook: ${FEISHU_WEBHOOK}
    secret: ${FEISHU_SECRET}
  email:
    host: smtp.gmail.com
    port: 587
    from: noreply@example.com
```

```go
// 从配置文件创建
client, err := notifyhub.New(
    notifyhub.WithYAML("config/notifyhub.yaml"),
)
```

### 步骤 3: 更新类型引用

```go
// v1
var hub notifyhub.Hub
var msg *notifyhub.Message

// v2
var client notifyhub.Client
var msg *notifyhub.Message  // Message 类型保持不变
```

### 步骤 4: 更新高级配置

```go
// v2 支持更丰富的配置选项
client, err := notifyhub.New(
    // 平台配置
    notifyhub.WithPlatform("feishu", feishuConfig),

    // 路由规则
    notifyhub.WithRouting(notifyhub.RoutingConfig{
        Rules: []notifyhub.RoutingRule{
            {
                Condition: "priority >= 4",
                Platforms: []string{"feishu", "email"},
            },
        },
    }),

    // 重试策略
    notifyhub.WithRetry(notifyhub.RetryConfig{
        MaxAttempts:     3,
        BackoffStrategy: "exponential",
    }),

    // 限流配置
    notifyhub.WithRateLimit(notifyhub.RateLimitConfig{
        GlobalLimit: &notifyhub.RateLimit{
            RequestsPerSecond: 100,
        },
    }),
)
```

## 迁移示例

### 示例 1: 基础使用

**v1 代码**
```go
package main

import (
    "context"
    "log"
    "github.com/kart-io/notifyhub/pkg/notifyhub"
    "github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

func main() {
    hub, err := notifyhub.NewHub(
        feishu.WithFeishu("webhook-url", feishu.WithFeishuSecret("secret")),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer hub.Close(context.Background())

    msg := notifyhub.NewMessage("Alert").
        WithBody("System error").
        Build()

    receipt, err := hub.Send(context.Background(), msg)
    if err != nil {
        log.Fatal(err)
    }
}
```

**v2 代码**
```go
package main

import (
    "context"
    "log"
    "github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
    client, err := notifyhub.New(
        notifyhub.WithFeishu("webhook-url", "secret"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    msg := notifyhub.NewMessage("Alert").
        WithBody("System error").
        Build()

    receipt, err := client.Send(context.Background(), msg)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 示例 2: 多平台配置

**v1 代码**
```go
import (
    "github.com/kart-io/notifyhub/pkg/notifyhub"
    "github.com/kart-io/notifyhub/pkg/platforms/feishu"
    "github.com/kart-io/notifyhub/pkg/platforms/email"
    "github.com/kart-io/notifyhub/pkg/platforms/sms"
)

hub, err := notifyhub.NewHub(
    feishu.WithFeishu(webhookURL, feishu.WithFeishuSecret(secret)),
    email.WithEmail(smtpHost, smtpPort, fromEmail,
        email.WithSMTPAuth(username, password),
        email.WithTLS(true),
    ),
    sms.WithSMS(sms.ConfigTwilio(apiKey, apiSecret)),
)
```

**v2 代码**
```go
import "github.com/kart-io/notifyhub/pkg/notifyhub"

client, err := notifyhub.New(
    notifyhub.WithPlatform("feishu", map[string]interface{}{
        "webhook": webhookURL,
        "secret":  secret,
    }),
    notifyhub.WithPlatform("email", map[string]interface{}{
        "host":     smtpHost,
        "port":     smtpPort,
        "from":     fromEmail,
        "username": username,
        "password": password,
        "useTLS":   true,
    }),
    notifyhub.WithPlatform("sms", map[string]interface{}{
        "provider":  "twilio",
        "apiKey":    apiKey,
        "apiSecret": apiSecret,
    }),
)
```

## 向后兼容

### 兼容模式

v2 提供了兼容层，允许逐步迁移：

```go
// 旧 API 仍然可用（标记为 deprecated）
hub, err := notifyhub.NewHub(opts...)  // @deprecated: 使用 New()

// 自动转换为新 API
func NewHub(opts ...HubOption) (Hub, error) {
    // 内部调用新 API
    client, err := New(convertOptions(opts...)...)
    return &hubAdapter{client}, err
}
```

### 迁移时间表

| 版本 | 时间 | 说明 |
|------|------|------|
| v1.x | 现在 | 当前稳定版本 |
| v2.0 | 2024 Q1 | 新 API 发布，旧 API 标记为 deprecated |
| v2.x | 2024 Q2-Q3 | 并行支持期，修复 bug |
| v3.0 | 2024 Q4 | 移除旧 API |

## 新功能亮点

### 1. 异步发送

```go
// v2 新增异步发送支持
receiptChan, err := client.SendAsync(ctx, message)
if err != nil {
    return err
}

// 非阻塞继续其他工作
doOtherWork()

// 稍后获取结果
select {
case receipt := <-receiptChan:
    log.Printf("Message sent: %s", receipt.ID)
case <-ctx.Done():
    log.Println("Context cancelled")
}
```

### 2. 平台状态查询

```go
// v2 新增平台状态查询
status, err := client.GetPlatformStatus(ctx, "feishu")
if err != nil {
    return err
}

log.Printf("Platform: %s, Status: %s, Latency: %dms",
    status.Platform,
    status.Status,
    status.Latency.Milliseconds())
```

### 3. 配置验证

```go
// v2 提供配置验证工具
config := notifyhub.Config{
    Platforms: platformConfigs,
}

// 验证配置
if err := notifyhub.ValidateConfig(config); err != nil {
    log.Fatal("Invalid configuration:", err)
}

client, err := notifyhub.NewFromConfig(config)
```

## 常见问题

### Q1: 必须一次性迁移所有代码吗？

**A**: 不需要。v2 提供了兼容层，你可以逐步迁移。旧的 API 会继续工作，但会显示废弃警告。

### Q2: 自定义平台如何迁移？

**A**: 自定义平台需要实现新的接口：

```go
// v1 自定义平台
type CustomPlatform struct {
    // 实现 v1 接口
}

// v2 自定义平台
// 放在 platforms/custom/ 目录下
package custom

func init() {
    // 自动注册到工厂
    notifyhub.RegisterPlatform("custom", NewCustomPlatform)
}

func NewCustomPlatform(config map[string]interface{}) (platform.Sender, error) {
    // 创建平台实例
}
```

### Q3: 配置文件格式有变化吗？

**A**: 配置文件格式更加标准化，但支持从旧格式自动转换：

```go
// 使用迁移工具转换配置
newConfig := notifyhub.MigrateConfig(oldConfig)
```

### Q4: 性能有影响吗？

**A**: v2 的性能优于 v1：
- 减少了不必要的依赖
- 优化了内部实现
- 更好的资源管理

## 获取帮助

### 资源

- 📚 [完整文档](https://github.com/kart-io/notifyhub/docs/v2)
- 💡 [示例代码](https://github.com/kart-io/notifyhub/examples/v2)
- 🐛 [问题反馈](https://github.com/kart-io/notifyhub/issues)
- 💬 [社区讨论](https://github.com/kart-io/notifyhub/discussions)

### 迁移工具

我们提供了自动迁移工具：

```bash
# 安装迁移工具
go install github.com/kart-io/notifyhub/cmd/migrate@latest

# 扫描并更新代码
notifyhub-migrate ./...

# 预览变更（不实际修改）
notifyhub-migrate --dry-run ./...

# 生成迁移报告
notifyhub-migrate --report migration-report.html ./...
```

### 支持

如果在迁移过程中遇到问题：

1. 查看 [迁移 FAQ](https://github.com/kart-io/notifyhub/wiki/Migration-FAQ)
2. 搜索 [已知问题](https://github.com/kart-io/notifyhub/issues?q=label:migration)
3. 提交新的 [Issue](https://github.com/kart-io/notifyhub/issues/new)
4. 加入 [Discord 社区](https://discord.gg/notifyhub)

## 总结

NotifyHub v2 带来了更好的开发体验：

- ✅ **更简洁的 API**：减少样板代码
- ✅ **更灵活的配置**：多种配置方式
- ✅ **更好的扩展性**：插件式架构
- ✅ **更清晰的结构**：职责分离

立即开始迁移，享受更好的开发体验！

---

*最后更新: 2024-01-15*
*版本: v2.0.0*