# NotifyHub Public API

NotifyHub 是一个统一的通知系统，提供了模块化、可扩展的架构，支持多种通知平台。

## 功能特性

- 🚀 **统一接口**: 单一 API 支持多种通知平台
- 🔌 **可插拔架构**: 支持内部和外部平台扩展
- 🔄 **异步处理**: 支持同步和异步消息发送
- 💪 **健壮性**: 内置重试机制和错误处理
- 📊 **监控**: 完整的健康检查和状态监控
- 🎯 **类型安全**: 完全的 Go 类型安全设计

## 架构概览

NotifyHub 采用符合 Go 设计原则的分层架构：

```
pkg/notifyhub/      # 公共 API (可被外部导入)
├── core/           # 核心 Hub 实现
│   ├── hub.go      # Hub 接口定义
│   ├── impl.go     # Hub 实现
│   ├── init.go     # 内部平台初始化
│   ├── manager.go  # 平台管理器
│   └── health.go   # 健康检查
├── platform/       # 平台注册机制
│   └── registry.go # 平台注册和接口
├── config/         # 配置管理
├── message/        # 消息类型
├── target/         # 目标类型
└── receipt/        # 回执类型

internal/pkg/       # 内部实现 (遵循 Go 包设计原则)
├── adapters/       # 内部平台适配器
│   └── feishu.go   # 飞书适配器
└── register/       # 自动注册
    └── register.go # 内部平台注册
```

### 设计原则

- **Go 包设计合规**: `pkg/` 包只依赖公共接口，`internal/` 包处理内部实现
- **清晰的职责分离**: 核心逻辑、平台适配、配置管理各司其职
- **可扩展性**: 支持外部平台通过公共接口进行扩展
- **类型安全**: 完整的类型定义和接口约束

## 快速开始

### 安装

```bash
go get github.com/kart-io/notifyhub/pkg/notifyhub
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"

    "github.com/kart-io/notifyhub/pkg/notifyhub"
    "github.com/kart-io/notifyhub/pkg/notifyhub/config"
    "github.com/kart-io/notifyhub/pkg/notifyhub/message"
    "github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

func main() {
    // 创建 Hub 配置
    cfg := &config.HubConfig{
        Platforms: map[string]config.PlatformConfig{
            "feishu": {
                "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook",
                "secret":      "your-secret",
            },
        },
    }

    // 创建 Hub 实例
    hub, err := notifyhub.NewHub(cfg)
    if err != nil {
        panic(err)
    }
    defer hub.Close(context.Background())

    // 创建消息
    msg := &message.Message{
        ID:    "msg-001",
        Title: "测试通知",
        Body:  "这是一条测试消息",
        Targets: []target.Target{
            {
                Type:     "webhook",
                Value:    "feishu-bot",
                Platform: "feishu",
            },
        },
    }

    // 发送消息
    receipt, err := hub.Send(context.Background(), msg)
    if err != nil {
        panic(err)
    }

    fmt.Printf("发送结果: %s, 成功: %d, 失败: %d\n",
        receipt.Status, receipt.Successful, receipt.Failed)
}
```

## 支持的平台

### 内置平台

- **飞书 (Feishu)**: 企业内部通知
- **Email**: 邮件通知 (开发中)
- **SMS**: 短信通知 (开发中)

### 外部平台扩展

支持通过实现 `platform.ExternalSender` 接口来扩展新的通知平台:

```go
package main

import (
    "context"
    "fmt"

    "github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// 自定义平台实现
type CustomSender struct {
    name string
}

func (c *CustomSender) Name() string {
    return c.name
}

func (c *CustomSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
    // 实现自定义发送逻辑
    results := make([]*platform.SendResult, len(targets))
    for i, target := range targets {
        results[i] = &platform.SendResult{
            Target:    target,
            Success:   true,
            MessageID: fmt.Sprintf("custom-%d", i),
        }
    }
    return results, nil
}

func (c *CustomSender) ValidateTarget(target platform.Target) error {
    return nil
}

func (c *CustomSender) GetCapabilities() platform.Capabilities {
    return platform.Capabilities{
        Name:                 c.name,
        SupportedTargetTypes: []string{"user", "channel"},
        SupportedFormats:     []string{"text", "markdown"},
    }
}

func (c *CustomSender) IsHealthy(ctx context.Context) error {
    return nil
}

func (c *CustomSender) Close() error {
    return nil
}

func init() {
    // 注册自定义平台
    platform.RegisterPlatform("custom", func(config map[string]interface{}) (platform.ExternalSender, error) {
        return &CustomSender{name: "custom"}, nil
    })
}
```

## 配置

### 平台配置

每个平台都有自己的配置要求:

#### 飞书配置

```go
"feishu": {
    "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook",
    "secret":      "your-secret",          // 可选
    "timeout":     "30s",                  // 可选，默认 30 秒
}
```

### 环境变量

支持通过环境变量进行配置:

```bash
FEISHU_WEBHOOK_URL=https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook
FEISHU_SECRET=your-secret
```

## API 参考

### 核心接口

#### Hub

主要的通知发送接口:

```go
type Hub interface {
    Send(ctx context.Context, message *message.Message) (*receipt.Receipt, error)
    SendAsync(ctx context.Context, message *message.Message) (*receipt.AsyncReceipt, error)
    Health(ctx context.Context) (*HealthStatus, error)
    Close(ctx context.Context) error
}
```

#### ExternalSender

外部平台扩展接口:

```go
type ExternalSender interface {
    Name() string
    Send(ctx context.Context, msg *Message, targets []Target) ([]*SendResult, error)
    ValidateTarget(target Target) error
    GetCapabilities() Capabilities
    IsHealthy(ctx context.Context) error
    Close() error
}
```

### 消息类型

#### Message

```go
type Message struct {
    ID           string                 `json:"id"`
    Title        string                 `json:"title"`
    Body         string                 `json:"body"`
    Format       string                 `json:"format"`       // "text", "markdown", "html"
    Priority     int                    `json:"priority"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    Variables    map[string]interface{} `json:"variables,omitempty"`
    PlatformData map[string]interface{} `json:"platform_data,omitempty"`
    Targets      []target.Target        `json:"targets"`
}
```

#### Target

```go
type Target struct {
    Type     string `json:"type"`     // "email", "user", "group", "webhook", "phone"
    Value    string `json:"value"`    // 目标标识符
    Platform string `json:"platform"` // 平台名称
}
```

#### Receipt

```go
type Receipt struct {
    MessageID  string            `json:"message_id"`
    Status     string            `json:"status"`     // "success", "failed", "partial"
    Results    []PlatformResult  `json:"results"`
    Successful int               `json:"successful"`
    Failed     int               `json:"failed"`
    Total      int               `json:"total"`
    Timestamp  time.Time         `json:"timestamp"`
}
```

## 健康检查

Hub 提供了完整的健康检查机制:

```go
health, err := hub.Health(context.Background())
if err != nil {
    log.Fatal(err)
}

fmt.Printf("整体健康状态: %v\n", health.Healthy)
for platform, status := range health.Platforms {
    fmt.Printf("平台 %s: %v\n", platform, status.Available)
}
```

## 错误处理

NotifyHub 提供了详细的错误信息和状态码:

```go
receipt, err := hub.Send(ctx, msg)
if err != nil {
    // 处理发送错误
    log.Printf("发送失败: %v", err)
    return
}

// 检查部分失败
if receipt.Status == "partial" {
    log.Printf("部分发送失败: 成功 %d, 失败 %d",
        receipt.Successful, receipt.Failed)

    for _, result := range receipt.Results {
        if !result.Success {
            log.Printf("平台 %s 发送失败: %s",
                result.Platform, result.Error)
        }
    }
}
```

## 最佳实践

### 1. 资源管理

始终确保正确关闭 Hub 资源:

```go
hub, err := notifyhub.NewHub(cfg)
if err != nil {
    return err
}
defer func() {
    if err := hub.Close(context.Background()); err != nil {
        log.Printf("关闭 Hub 失败: %v", err)
    }
}()
```

### 2. 上下文管理

使用带超时的上下文:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

receipt, err := hub.Send(ctx, msg)
```

### 3. 错误处理

检查所有可能的错误情况:

```go
receipt, err := hub.Send(ctx, msg)
if err != nil {
    return fmt.Errorf("发送消息失败: %w", err)
}

if receipt.Failed > 0 {
    return fmt.Errorf("部分发送失败: %d/%d", receipt.Failed, receipt.Total)
}
```

### 4. 平台扩展

注册自定义平台时使用 init 函数:

```go
func init() {
    platform.RegisterPlatform("myplatform", NewMyPlatformSender)
}
```

## 示例

更多示例请参考:

- [飞书通知示例](../../examples/feishu/)
- [外部平台扩展示例](../../examples/external_platform/)
- [高级配置示例](../../examples/advanced/)

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](../../LICENSE) 文件。

## 贡献

欢迎贡献代码！请参考 [CONTRIBUTING.md](../../CONTRIBUTING.md) 了解详细信息。