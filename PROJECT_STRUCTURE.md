# NotifyHub - Clean Architecture

## Project Overview

NotifyHub 是一个优雅、统一的多平台通知系统，采用现代化的 Go 架构设计。

## 项目结构

```
notifyhub/
├── pkg/notifyhub/          # 公共 API 层
│   ├── config.go           # 配置工具和预设
│   ├── errors.go           # 公共错误类型
│   ├── hub.go              # 主要 Hub 接口实现
│   ├── message.go          # 消息构建器和 Fluent API
│   ├── options.go          # Hub 配置选项
│   ├── receipt.go          # 发送回执和结果
│   └── target.go           # 目标类型和自动检测
├── internal/platform/      # 内部平台抽象层
│   ├── interface.go        # 平台发送器接口定义
│   ├── manager.go          # 平台管理器和工厂
│   ├── email/             # Email 发送器实现
│   ├── feishu/            # 飞书发送器实现
│   └── sms/               # SMS 发送器实现
├── examples/elegant_api/   # 示例程序
│   ├── main.go            # 展示优雅 API 用法
│   └── README.md          # 示例说明
├── docs/                  # 文档目录
├── go.mod                 # Go 模块定义
├── go.sum                 # 依赖校验和
├── Makefile              # 构建工具
├── CLAUDE.md             # 项目指导文档
└── README.md             # 项目说明
```

## 核心特性

### 1. 优雅的 Fluent API

```go
receipt, err := hub.Send(ctx, notifyhub.NewMessage("Task Completed").
    WithText("Your data processing task has been completed successfully.").
    ToEmail("user@example.com").
    ToFeishu("oc_xxxxxxxx").
    ToPhone("+1234567890").
    Build())
```

### 2. 三层架构设计

- **Public API Layer** (`pkg/notifyhub/`) - 用户友好的公共接口
- **Internal Core Logic** (`internal/platform/`) - 核心平台抽象和管理
- **Platform Implementations** - 具体的平台发送器实现

### 3. 类型安全的配置

```go
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishu(webhookURL, secret),
    notifyhub.WithEmail(host, port, username, password, from),
    notifyhub.WithSMS(provider, apiKey, apiSecret),
    notifyhub.WithTimeout(10*time.Second),
)
```

### 4. 完整的平台支持

- **Email** - SMTP 支持，HTML/文本格式，CC/BCC，附件
- **飞书** - Webhook 消息，卡片，提及，群组消息
- **SMS** - 多提供商支持，模板变量

### 5. 智能目标路由

- 自动检测目标类型和平台
- 并发多平台发送
- 详细的发送结果反馈

## 开发命令

```bash
# 构建项目
make build

# 代码检查
make check

# 格式化代码
make fmt

# 运行 lint
make lint

# 运行示例
cd examples/elegant_api && go run main.go

# 预提交检查
make pre-commit
```

## 快速开始

1. **导入包**

```go
import "github.com/kart-io/notifyhub/pkg/notifyhub"
```

2. **创建 Hub**

```go
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishu(webhookURL, secret),
    notifyhub.WithEmail(host, port, username, password, from),
)
```

3. **发送消息**

```go
receipt, err := hub.Send(ctx, notifyhub.NewMessage("Hello").
    WithText("Hello, World!").
    ToEmail("user@example.com").
    Build())
```

## 设计原则

- **简洁优雅** - 最小化的 API 设计，链式调用
- **类型安全** - 编译时错误检测，强类型配置
- **可扩展性** - 基于接口的设计，易于添加新平台
- **并发安全** - 线程安全的实现
- **错误处理** - 详细的错误信息和类型

## License

MIT License
