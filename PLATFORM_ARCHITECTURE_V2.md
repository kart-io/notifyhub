# NotifyHub v2 平台架构重新设计

## 🎯 问题和目标

### 原架构问题

- **外部开发者无法真正扩展**：必须修改 `api_adapter.go` 才能添加便利函数
- **硬编码平台函数**：`WithSlack`、`WithDiscord` 等硬编码在核心库中
- **API 不一致**：内置平台有便利函数，外部平台只能用通用函数
- **违反开闭原则**：对修改开放，对扩展封闭

### 新架构目标

- ✅ **真正的外部可扩展性**：不修改核心库就能添加新平台
- ✅ **一致的开发体验**：所有平台提供相同质量的 API
- ✅ **清晰的职责分离**：每个平台管理自己的生命周期
- ✅ **向后兼容**：现有代码继续工作，提供平滑迁移路径

## 🏗️ 新架构设计

### 核心原则：**平台包自治**

每个平台在自己的包中提供：

1. **ExternalSender 实现**
2. **便利配置函数**
3. **自动注册机制**
4. **完整文档**

### 目录结构

```
pkg/platforms/                    # 官方平台包
├── slack/
│   ├── sender.go                 # Slack 实现
│   ├── options.go               # WithSlack 便利函数
│   └── README.md                # 使用文档

examples/external_discord_platform/  # 外部平台示例
├── discord/
│   ├── sender.go                # Discord 实现
│   ├── options.go               # WithDiscord 便利函数
│   └── README.md                # 使用文档
```

## 🚀 使用方式对比

### 老方式（硬编码）

```go
import "github.com/kart-io/notifyhub/pkg/notifyhub"

hub := notifyhub.NewHub(
    notifyhub.WithSlack("webhook-url"),  // 硬编码在核心库
)
```

### 新方式（平台包）

```go
import "github.com/kart-io/notifyhub/pkg/notifyhub"
import "github.com/kart-io/notifyhub/pkg/platforms/slack"

hub := notifyhub.NewHub(
    slack.WithSlack("webhook-url"),      // 来自平台包
)
```

### 外部平台（完全独立）

```go
import "github.com/kart-io/notifyhub/pkg/notifyhub"
import "github.com/yourorg/notifyhub-discord"

hub := notifyhub.NewHub(
    discord.WithDiscord("webhook-url"), // 来自外部包
)
```

## 🔧 技术实现

### 1. 自动注册机制

每个平台包使用 `sync.Once` 确保只注册一次：

```go
// 在平台包中
var registerOnce sync.Once

func ensureRegistered() {
    registerOnce.Do(func() {
        notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
            Name:    "slack",
            Creator: NewSlackSender,
            // ... 其他配置
        })
    })
}

func WithSlack(webhookURL string) notifyhub.HubOption {
    ensureRegistered()  // 懒注册
    return notifyhub.WithCustomPlatform("slack", config)
}
```

### 2. 便利函数设计

提供类型安全的配置选项：

```go
func WithSlack(webhookURL string, options ...func(map[string]interface{})) notifyhub.HubOption

func WithSlackTimeout(timeout time.Duration) func(map[string]interface{})
func WithSlackUsername(username string) func(map[string]interface{})
func WithSlackIcon(emoji string) func(map[string]interface{})
```

### 3. 向后兼容

核心库中的函数标记为 deprecated 但继续工作：

```go
// Deprecated: Use github.com/kart-io/notifyhub/pkg/platforms/slack.WithSlack instead
func WithSlack(webhookURL string) HubOption {
    // 继续工作，但提示迁移
}
```

## 📚 平台包开发指南

### 创建新平台包的步骤

1. **实现 ExternalSender 接口**

```go
type MyPlatformSender struct {
    // 平台特定字段
}

func (m *MyPlatformSender) Name() string { return "myplatform" }
func (m *MyPlatformSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error)
func (m *MyPlatformSender) ValidateTarget(target platform.Target) error
func (m *MyPlatformSender) GetCapabilities() platform.Capabilities
func (m *MyPlatformSender) IsHealthy(ctx context.Context) error
func (m *MyPlatformSender) Close() error
```

2. **提供工厂函数**

```go
func NewMyPlatformSender(config map[string]interface{}) (platform.ExternalSender, error) {
    // 创建和配置发送器
}
```

3. **创建便利函数**

```go
func WithMyPlatform(apiKey string, options ...func(map[string]interface{})) notifyhub.HubOption {
    ensureRegistered()
    return notifyhub.WithCustomPlatform("myplatform", config)
}
```

4. **实现自动注册**

```go
var registerOnce sync.Once

func ensureRegistered() {
    registerOnce.Do(func() {
        notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
            Name:    "myplatform",
            Creator: NewMyPlatformSender,
            // ...
        })
    })
}
```

## 🔄 迁移指南

### 阶段 1：立即可用（向后兼容）

现有代码无需修改，继续使用 `notifyhub.WithSlack()` 等函数。

### 阶段 2：推荐迁移

```go
// 老代码
import "github.com/kart-io/notifyhub/pkg/notifyhub"
hub := notifyhub.NewHub(notifyhub.WithSlack("url"))

// 新代码
import "github.com/kart-io/notifyhub/pkg/notifyhub"
import "github.com/kart-io/notifyhub/pkg/platforms/slack"
hub := notifyhub.NewHub(slack.WithSlack("url"))
```

### 阶段 3：移除 deprecated 函数（v3.0）

在未来版本中移除核心库中的硬编码函数。

## 🎯 架构优势

### 1. 真正的外部可扩展性

- 外部开发者可以创建完整的平台包
- 无需修改或fork核心库
- 与内置平台具有相同的API质量

### 2. 清晰的职责分离

- 核心库专注于消息路由和生命周期管理
- 平台包专注于特定平台的实现细节
- 每个组件有明确的边界

### 3. 更好的维护性

- 平台特定的bug不影响核心库
- 平台包可以独立发版和更新
- 降低了核心库的复杂性

### 4. 一致的开发体验

- 所有平台使用相同的注册机制
- 统一的配置模式
- 一致的错误处理

### 5. 社区生态

- 鼓励社区贡献平台包
- 每个平台包可以有自己的维护团队
- 形成丰富的平台生态系统

## 📖 示例和文档

- **Slack 平台包**: `pkg/platforms/slack/`
- **外部 Discord 示例**: `examples/external_discord_platform/`
- **平台包使用示例**: `examples/platform_packages/`
- **迁移示例**: 见各示例目录的 README

## 🚀 未来计划

1. **v2.1**: 完善平台包架构，提供更多官方平台包
2. **v2.2**: 增加平台发现和推荐机制
3. **v3.0**: 移除 deprecated 函数，完全基于平台包架构

这个架构重新设计解决了原始问题，为 NotifyHub 提供了真正的可扩展性，同时保持了优秀的开发体验。
