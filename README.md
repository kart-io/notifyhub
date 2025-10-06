# NotifyHub

[![Go Version](https://img.shields.io/badge/go-1.24.0-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kart-io/notifyhub)](https://goreportcard.com/report/github.com/kart-io/notifyhub)

NotifyHub 是一个现代化的统一通知系统，提供简洁的 API 和强大的异步处理能力。支持多个通知平台（飞书、邮件、Slack、Webhook），具备完整的协程池管理、智能重试机制和丰富的回调功能。

## ✨ 核心特性

### 🚀 统一的通知接口

- **3层架构设计**: `Client → Dispatcher → Platform`，替代复杂的6层调用链
- **多平台支持**: 飞书(Feishu)、邮件(Email)、Slack、Webhook
- **外部平台扩展**: 支持在不修改核心代码的情况下添加新平台
- **统一消息格式**: 跨平台的消息抽象和目标路由

### ⚡ 高性能异步处理

- **真正的异步支持**: 基于队列的异步处理，支持回调机制
- **协程池管理**: 智能协程池，支持动态工作协程数量调整
- **批量操作**: 高效的批量消息发送和处理
- **并发控制**: 支持多协程并发发送，资源管理完善

### 🔄 智能路由与错误处理

- **ML 智能路由**: 基于平台健康度、成功率、响应时间的多因素评分算法
- **自动平台检测**: 根据目标类型智能选择最佳平台
- **用户/组解析**: 支持用户和组的自动展开和路由
- **错误分类处理**: 区分临时错误和永久错误的处理策略

### 📊 完整的观察性

- **实时指标统计**: 自动追踪总发送数、成功率、活跃任务数
- **健康检查**: SMTP 连接测试、平台状态监控、队列健康检查
- **Worker 动态扩缩容**: 根据负载自动调整工作协程数量
- **生命周期管理**: 优雅的启动和关闭流程，防止资源泄漏

## 🏗️ 架构设计

### 整体架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│                 │    │                  │    │                 │
│  Client Layer   │───▶│ Dispatcher Layer │───▶│ Platform Layer  │
│                 │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
        │                       │                       │
        │                       │                       │
        ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ - Send          │    │ - Message Router │    │ - Feishu        │
│ - SendAsync     │    │ - Template Engine│    │ - Email         │
│ - SendBatch     │    │ - Target Resolver│    │ - Slack         │
│ - Health        │    │ - Error Handler  │    │ - Webhook       │
│                 │    │                  │    │ - External      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### 外部平台扩展架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    Plugin Architecture                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                Platform Registry                        │   │
│  │  - RegisterFactory(name, factory)                      │   │
│  │  - SetConfig(name, config)                             │   │
│  │  - GetPlatform(name)                                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Internal   │  │   Internal   │  │   External   │         │
│  │   Platform   │  │   Platform   │  │   Platform   │         │
│  │              │  │              │  │              │         │
│  │   - Feishu   │  │   - Email    │  │  - DingTalk  │         │
│  │   - Slack    │  │   - Webhook  │  │  - Teams     │         │
│  │              │  │              │  │  - Custom    │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│                                                                 │
│  All platforms implement the same Platform interface:          │
│  - Name() string                                               │
│  - GetCapabilities() Capabilities                              │
│  - Send(ctx, msg, targets) ([]*SendResult, error)             │
│  - ValidateTarget(target) error                                │
│  - IsHealthy(ctx) error                                        │
│  - Close() error                                               │
└─────────────────────────────────────────────────────────────────┘
```

### 异步处理架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│                 │    │                  │    │                 │
│ Async Client    │───▶│   Message Queue  │───▶│  Worker Pool    │
│                 │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
        │                       │                       │
        │                       │                       │
        ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ - Handle        │    │ - QueueItem      │    │ - Worker        │
│ - BatchHandle   │    │ - ProcessorFunc  │    │ - ProcessItem   │
│ - Callbacks     │    │ - Stats          │    │ - Lifecycle     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### 协程池管理

```
┌─────────────────────────────────────────────────────────────────┐
│                      Goroutine Pool Manager                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐           │
│  │   Worker 1  │   │   Worker 2  │   │   Worker N  │           │
│  │             │   │             │   │             │           │
│  │ processItem │   │ processItem │   │ processItem │           │
│  │ lifecycle   │   │ lifecycle   │   │ lifecycle   │           │
│  └─────────────┘   └─────────────┘   └─────────────┘           │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Message Queue                         │   │
│  │  [QueueItem] -> [QueueItem] -> [QueueItem] -> ...      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Configuration:                                                 │
│  - Workers: 1-20 (动态调整)                                    │
│  - BufferSize: 100-1000                                        │
│  - UsePool: true/false (池模式开关)                            │
└─────────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 安装

```bash
go get github.com/kart/notifyhub
```

### 基础使用

```go
package main

import (
    "context"
    "fmt"

    "github.com/kart/notifyhub/pkg/notifyhub"
    "github.com/kart/notifyhub/pkg/config"
    "github.com/kart/notifyhub/pkg/message"
    "github.com/kart/notifyhub/pkg/target"
)

func main() {
    // 创建配置
    cfg := &config.Config{
        Feishu: config.FeishuConfig{
            WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url",
        },
    }

    // 创建客户端
    client, err := notifyhub.NewClient(cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 创建消息
    msg := message.New()
    msg.Title = "Hello NotifyHub"
    msg.Body = "这是一条测试消息"
    msg.Targets = []target.Target{
        target.Feishu(""),
    }

    // 发送消息
    ctx := context.Background()
    receipt, err := client.Send(ctx, msg)
    if err != nil {
        panic(err)
    }

    fmt.Printf("消息发送成功: %s\n", receipt.MessageID)
}
```

### 异步发送与回调

```go
// 异步发送消息
handle, err := client.SendAsync(ctx, msg)
if err != nil {
    panic(err)
}

// 设置回调
handle.OnComplete(func(receipt *receipt.Receipt) {
    fmt.Printf("消息发送成功: %s\n", receipt.MessageID)
}).OnError(func(msg *message.Message, err error) {
    fmt.Printf("消息发送失败: %v\n", err)
})

// 等待完成
receipt, err := handle.Wait(ctx)
```

### 协程池配置

```go
cfg := &config.Config{
    Async: config.AsyncConfig{
        Enabled:    true,
        UsePool:    true,        // 启用协程池
        Workers:    8,           // 工作协程数
        BufferSize: 200,         // 队列缓冲区大小
        MinWorkers: 4,           // 最小工作协程数
        MaxWorkers: 16,          // 最大工作协程数
    },
    // ... 其他配置
}
```

## 📖 详细功能

### 支持的平台

#### 1. 飞书 (Feishu)

```go
cfg.Feishu = config.FeishuConfig{
    WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url",
    Secret:     "your-secret",  // 可选
}
```

#### 2. 邮件 (Email)

```go
cfg.Email = config.EmailConfig{
    Host:     "smtp.example.com",
    Port:     587,
    Username: "user@example.com",
    Password: "password",
    From:     "sender@example.com",
    UseTLS:   true,
}
```

#### 3. Slack

```go
cfg.Slack = config.SlackConfig{
    WebhookURL: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    Token:      "xoxb-your-bot-token",  // 可选
}
```

#### 4. Webhook

```go
cfg.Webhook = config.WebhookConfig{
    URL:     "https://your-webhook-endpoint.com/webhook",
    Method:  "POST",
    Headers: map[string]string{
        "Authorization": "Bearer your-token",
    },
}
```

### 消息类型和格式

```go
// 文本消息
msg := message.New()
msg.Title = "标题"
msg.Body = "消息内容"
msg.Format = message.FormatText

// Markdown消息
msg.Format = message.FormatMarkdown
msg.Body = "**加粗文本** 和 `代码`"

// 带优先级的消息
msg.Priority = message.PriorityHigh

// 带变量的模板消息
msg.Variables = map[string]interface{}{
    "user_name": "张三",
    "amount":    100.50,
}
```

### 批量操作

```go
// 同步批量发送
messages := []*message.Message{msg1, msg2, msg3}
receipts, err := client.SendBatch(ctx, messages)

// 异步批量发送
batchHandle, err := client.SendAsyncBatch(ctx, messages)

// 监控批量进度
go func() {
    for progress := range batchHandle.Progress() {
        fmt.Printf("进度: %d/%d (%.1f%%)\n",
            progress.Completed, progress.Total, progress.Progress*100)
    }
}()

// 等待批量完成
receipts, err := batchHandle.Wait(ctx)
```

### 健康检查和监控

```go
// 检查系统健康状态
health, err := client.Health(ctx)
if err != nil {
    panic(err)
}

fmt.Printf("状态: %s\n", health.Status)           // "healthy" 或 "degraded"
fmt.Printf("运行时长: %.0f秒\n", health.Uptime)    // 系统运行时长
fmt.Printf("活跃任务: %d\n", health.ActiveTasks)   // 当前活跃任务数
fmt.Printf("队列深度: %d\n", health.QueueDepth)    // 队列中待处理消息数
fmt.Printf("总发送数: %d\n", health.TotalSent)     // 累计发送消息数
fmt.Printf("成功率: %.2f%%\n", health.SuccessRate) // 发送成功率

// 检查各平台状态
for platform, status := range health.Platforms {
    fmt.Printf("平台 %s: %s\n", platform, status)
}
```

### 智能路由功能

```go
// 自动平台检测 - 根据目标类型智能选择平台
msg := message.New()
msg.Title = "测试消息"
msg.Body = "智能路由测试"

// 不指定平台，系统会自动检测
msg.Targets = []target.Target{
    {Type: "email", Value: "user@example.com"},  // 自动路由到 email 平台
    {Type: "user", Value: "john.doe"},           // 自动路由到 email 平台
    {Type: "group", Value: "developers"},        // 自动展开组成员并路由
}

receipt, err := client.Send(ctx, msg)

// ML 智能路由会根据以下因素选择最佳平台：
// - 平台健康状态（权重 40%）
// - 历史成功率（权重 30%）
// - 平均响应时间（权重 20%）
// - 平台运行时间（权重 10%）
```

### 用户和组解析

```go
// 用户解析 - 自动转换为联系方式
userTarget := target.Target{
    Type:  "user",
    Value: "john.doe@example.com",  // 或者用户ID
}

// 组解析 - 自动展开为组成员
groupTarget := target.Target{
    Type:  "group",
    Value: "developers",  // 支持预定义组: admins, developers, support, all
}

msg.Targets = []target.Target{userTarget, groupTarget}
receipt, err := client.Send(ctx, msg)

// 组 "developers" 会自动展开为所有成员的目标
// 每个成员都会收到独立的通知
```

### Worker 池动态扩缩容

```go
// 配置自动扩缩容
cfg := &config.Config{
    Async: config.AsyncConfig{
        Enabled:    true,
        UsePool:    true,
        Workers:    8,      // 初始工作协程数
        MinWorkers: 4,      // 最小工作协程数
        MaxWorkers: 16,     // 最大工作协程数
        BufferSize: 200,
    },
}

client, _ := notifyhub.NewClient(cfg)

// Worker 池会根据队列负载自动调整：
// - 队列积压增加时，自动增加 worker（不超过 MaxWorkers）
// - 队列空闲时，自动减少 worker（不少于 MinWorkers）
// - 确保资源高效利用和快速响应
```

## 📁 项目结构

```
notifyhub/
├── pkg/                          # 核心包
│   ├── notifyhub/               # 主客户端接口
│   │   ├── client.go            # Client接口定义
│   │   ├── factory.go           # 客户端工厂实现
│   │   └── config.go            # 客户端配置
│   ├── async/                   # 异步处理
│   │   ├── queue.go             # 队列接口和实现
│   │   ├── worker.go            # 工作协程
│   │   ├── handle.go            # 异步句柄
│   │   └── callback.go          # 回调管理
│   ├── platforms/               # 平台适配器
│   │   ├── feishu/              # 飞书平台
│   │   ├── email/               # 邮件平台
│   │   ├── slack/               # Slack平台
│   │   └── webhook/             # Webhook平台
│   ├── message/                 # 消息相关
│   │   ├── message.go           # 消息定义
│   │   ├── builder.go           # 消息构建器
│   │   └── validator.go         # 消息验证
│   ├── target/                  # 目标相关
│   │   ├── target.go            # 目标定义
│   │   ├── resolver.go          # 目标解析
│   │   └── router.go            # 目标路由
│   ├── config/                  # 配置管理
│   │   ├── config.go            # 主配置
│   │   ├── options.go           # 功能选项
│   │   └── platforms/           # 平台配置
│   ├── core/                    # 核心组件
│   │   ├── dispatcher.go        # 消息分发器
│   │   ├── manager.go           # 生命周期管理
│   │   └── router.go            # 消息路由
│   ├── template/                # 模板引擎
│   ├── receipt/                 # 回执处理
│   ├── errors/                  # 错误处理
│   └── utils/                   # 工具包
├── examples/                    # 示例代码
│   ├── feishu/                  # 飞书示例
│   │   ├── basic/               # 基础使用
│   │   └── advanced/            # 高级功能
│   ├── email/                   # 邮件示例
│   ├── slack/                   # Slack示例
│   ├── webhook/                 # Webhook示例
│   ├── async-callbacks/         # 异步回调示例
│   ├── multi-goroutine/         # 多协程示例
│   ├── goroutine-pool/          # 协程池示例
│   └── external-platform-dingtalk/  # 外部平台扩展示例 (钉钉)
├── docs/                        # 文档
├── CLAUDE.md                    # Claude Code 工作指南
├── Makefile                     # 构建脚本
└── README.md                    # 项目说明
```

## 🔧 配置选项

### 基础配置

```go
type Config struct {
    // 平台配置
    Feishu   FeishuConfig   `json:"feishu"`
    Email    EmailConfig    `json:"email"`
    Slack    SlackConfig    `json:"slack"`
    Webhook  WebhookConfig  `json:"webhook"`

    // 异步配置
    Async    AsyncConfig    `json:"async"`

    // 日志配置
    Logger   LoggerConfig   `json:"logger"`
}
```

### 异步配置详解

```go
type AsyncConfig struct {
    Enabled     bool          `json:"enabled"`      // 是否启用异步
    UsePool     bool          `json:"use_pool"`     // 是否使用协程池
    Workers     int           `json:"workers"`      // 工作协程数
    BufferSize  int           `json:"buffer_size"`  // 队列缓冲区大小
    Timeout     time.Duration `json:"timeout"`      // 超时时间
    MinWorkers  int           `json:"min_workers"`  // 最小工作协程数
    MaxWorkers  int           `json:"max_workers"`  // 最大工作协程数
}
```

### 重试策略配置

```go
type RetryPolicy struct {
    MaxRetries      int           `json:"max_retries"`       // 最大重试次数
    InitialInterval time.Duration `json:"initial_interval"`  // 初始重试间隔
    MaxInterval     time.Duration `json:"max_interval"`      // 最大重试间隔
    Multiplier      float64       `json:"multiplier"`        // 重试间隔倍数
    Jitter          bool          `json:"jitter"`            // 是否添加抖动
}
```

## 🔍 示例代码

### 协程池性能对比

```go
// examples/goroutine-pool/main.go
// 对比协程池模式 vs 直接协程模式的性能差异

func comparePoolVsDirect() {
    // 直接协程模式
    cfg1.Async.UsePool = false
    client1, _ := notifyhub.NewClient(cfg1)

    // 协程池模式
    cfg2.Async.UsePool = true
    client2, _ := notifyhub.NewClient(cfg2)

    // 性能测试和对比...
}
```

### 多协程并发发送

```go
// examples/multi-goroutine/main.go
// 演示多协程并发发送和压力测试

func concurrentSingleMessages(client notifyhub.Client) {
    var wg sync.WaitGroup
    concurrency := 10

    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()

            // 创建和发送消息
            handle, err := client.SendAsync(ctx, msg)
            if err != nil {
                return
            }

            // 设置回调
            handle.OnComplete(func(receipt *receipt.Receipt) {
                fmt.Printf("协程 %d 发送成功\n", index+1)
            })

            // 等待完成
            _, err = handle.Wait(ctx)
        }(i)
    }

    wg.Wait()
}
```

### 异步回调处理

```go
// examples/async-callbacks/main.go
// 演示丰富的异步回调功能

handle, err := client.SendAsync(ctx, msg)
if err != nil {
    panic(err)
}

// 链式回调设置
handle.OnComplete(func(receipt *receipt.Receipt) {
    fmt.Printf("✅ 发送成功: %s\n", receipt.MessageID)
}).OnError(func(msg *message.Message, err error) {
    fmt.Printf("❌ 发送失败: %v\n", err)
}).OnProgress(func(stage string, percent float64) {
    fmt.Printf("📊 进度: %s (%.1f%%)\n", stage, percent*100)
})

// 超时等待
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

receipt, err := handle.Wait(ctx)
```

### 外部平台扩展示例

```go
// examples/external-platform-dingtalk/
// 演示如何在不修改 NotifyHub 核心代码的情况下添加钉钉平台

// 1. 实现平台接口
type DingTalkPlatform struct {
    config     DingTalkConfig
    httpClient *http.Client
}

func (d *DingTalkPlatform) Name() string { return "dingtalk" }
func (d *DingTalkPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
    // 实现钉钉消息发送逻辑
}

// 2. 注册外部平台
factory := platform.Factory(NewDingTalkPlatform)
client.RegisterPlatform("dingtalk", factory)

// 3. 配置和使用
config := DingTalkConfig{WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN"}
client.SetPlatformConfig("dingtalk", config)

// 4. 发送消息
msg.Targets = []target.Target{CreateDingTalkTarget(webhookURL)}
receipt, err := client.Send(ctx, msg)
```

## 🔌 外部平台扩展

NotifyHub 设计为可扩展的插件架构，支持在不修改核心代码的情况下添加新的通知平台。

### 扩展原理

通过实现标准的 `Platform` 接口，任何外部平台都可以无缝集成到 NotifyHub 中：

```go
type Platform interface {
    Name() string
    GetCapabilities() Capabilities
    Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error)
    ValidateTarget(target target.Target) error
    IsHealthy(ctx context.Context) error
    Close() error
}
```

### 支持的扩展平台

基于相同的扩展机制，可以支持任何通知平台：

- **钉钉 (DingTalk)** ✅ - [完整示例](examples/external-platform-dingtalk/)
- **微信企业号** - 企业微信群机器人
- **Microsoft Teams** - Teams 频道消息
- **Discord** - Discord 频道通知
- **Telegram** - Telegram Bot API
- **自定义平台** - 任何支持 HTTP API 的服务

### 扩展步骤

1. **实现平台接口**

```go
type CustomPlatform struct {
    config CustomConfig
}

func (c *CustomPlatform) Name() string { return "custom" }
// ... 实现其他接口方法
```

2. **注册平台工厂**

```go
factory := platform.Factory(NewCustomPlatform)
client.RegisterPlatform("custom", factory)
```

3. **配置平台**

```go
config := CustomConfig{/* 平台特定配置 */}
client.SetPlatformConfig("custom", config)
```

4. **使用平台**

```go
msg.Targets = []target.Target{CreateCustomTarget(address)}
receipt, err := client.Send(ctx, msg)
```

### 扩展优势

- **零侵入**: 不需要修改 NotifyHub 核心代码
- **标准化**: 统一的接口和消息格式
- **独立性**: 外部平台可独立开发、测试和维护
- **兼容性**: 完全兼容现有的异步、批量、回调等功能

详细的外部平台扩展指南请参考：[外部平台扩展示例 - 钉钉](examples/external-platform-dingtalk/README.md)

## 🛠️ 开发指南

### 构建和测试

```bash
# 构建所有包
go build ./...

# 运行测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 运行带竞态检测的测试
go test -race ./...

# 运行特定包的测试
go test ./pkg/async
go test ./pkg/platforms/feishu

# 代码格式化
go fmt ./...
gofmt -w .

# 代码检查
go vet ./...

# 运行 linter（需要安装 golangci-lint）
golangci-lint run ./...
```

### 代码质量标准

项目已通过以下质量检查：

- ✅ **编译**: 无错误
- ✅ **单元测试**: 185+ 测试全部通过
- ✅ **代码覆盖率**: 核心包 35-97% 覆盖
- ✅ **静态分析**: golangci-lint 0 issues
- ✅ **竞态检测**: 无数据竞争问题
- ✅ **内存安全**: 无泄漏，正确的资源管理

### 已实现的核心功能

✅ **智能路由系统**

- ML 评分算法：基于健康度、成功率、响应时间的智能平台选择
- 自动平台检测：根据目标类型（email/phone/webhook/user/group）自动路由
- 用户解析：支持用户 ID 到联系方式的智能转换
- 组解析：支持组成员自动展开（内置 admins/developers/support/all 组）

✅ **实时监控指标**

- 运行时长追踪（Uptime）
- 活跃任务计数（使用 atomic.Int64）
- 总发送数、成功数、失败数统计
- 实时成功率计算
- 队列深度监控

✅ **健康检查机制**

- SMTP 连接健康检查（使用 NOOP 命令）
- 平台状态实时监控
- 优雅的错误处理和降级

✅ **Worker 池管理**

- 动态扩容：根据负载自动增加 worker
- 动态缩容：根据空闲自动减少 worker
- 线程安全：使用 sync.Mutex 保护
- 优雅停止：使用 sync.WaitGroup 等待完成

### 添加新平台

1. 在 `pkg/platforms/` 下创建新平台目录
2. 实现 `Platform` 接口:

```go
type Platform interface {
    Send(ctx context.Context, msg *message.Message, target target.Target) (*receipt.Receipt, error)
    Validate(target target.Target) error
    Close() error
}
```

3. 注册平台到注册表
4. 添加配置选项
5. 创建示例代码

### 扩展消息类型

1. 在 `pkg/message/` 中扩展消息结构
2. 更新验证逻辑
3. 在各平台适配器中添加支持
4. 更新模板引擎

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

### 代码风格

- 遵循 Go 语言官方代码风格
- 使用 `golangci-lint` 进行代码检查
- 所有公共 API 必须有文档注释
- 单元测试覆盖率不低于 80%

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

- [Go 语言团队](https://golang.org) 提供优秀的编程语言
- 所有为开源社区做出贡献的开发者们

---

**NotifyHub** - 让通知变得简单而强大！
