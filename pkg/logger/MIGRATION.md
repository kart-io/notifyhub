# Logger Refactoring & Migration Guide (v2)

## 📋 变更总结

为了实现完全解耦和更现代化的日志体验，Logger 模块进行了重大重构。本次重构的核心思想是**依赖注入**和**统一接口**。

**主要变更点:**

1.  **接口简化**: `Logger` 接口被简化，采用了类似 `slog` 的风格，使用 `(msg string, args ...any)` 进行结构化日志记录。
2.  **移除全局实例**: 全局的 `logger.Default` 和 `logger.NewWithPrefix` 已被移除。现在必须通过依赖注入来提供 logger。
3.  **中央配置**: 日志记录器在创建 `notifyhub.Hub` 时通过 `notifyhub.WithLogger()` 一次性配置，并自动传递给所有平台。
4.  **平台解耦**: 平台（如 Email, Feishu）不再创建自己的 logger 实例，而是接收从 Hub 传递过来的 logger。
5.  **移除特定于平台的日志选项**: `WithEmailLogger` 和 `WithEmailLogLevel` 等函数已被移除，以支持统一的 `WithLogger`。

## 🔄 迁移步骤

### 对于 NotifyHub 的使用者

你的日志配置方式需要改变。之前你可能为每个平台单独配置日志，现在你只需要在创建 Hub 时配置一次。

**迁移前:**

```go
// 旧方式：为每个平台单独设置日志级别
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.example.com", 587, "from@example.com",
        email.WithEmailLogLevel(logger.Debug),
    ),
    feishu.WithFeishu("https://...",
        feishu.WithFeishuLogLevel(logger.Info),
    ),
)
```

**迁移后:**

```go
import (
	"os"
	"log/slog"
	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

// 新方式：创建中心 logger 并注入

// 1. 创建你选择的 logger (例如 slog)
slogHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
slogLogger := slog.New(slogHandler)

// 2. 包装在适配器中
notifyHubLogger := logger.NewSlogAdapter(slogLogger)

// 3. 在创建 Hub 时通过 WithLogger 注入
hub, err := notifyhub.New(
    notifyhub.WithEmail("smtp.example.com", 587, "from@example.com"),
    notifyhub.WithFeishu("https://..."),
    notifyhub.WithLogger(notifyHubLogger), // 在此处统一注入
)
```

### 对于平台开发者

如果你是平台（Platform）的开发者，你需要更新你的代码以接收注入的 logger，而不是自己创建它。

**迁移前:**

```go
// 旧的平台 Sender 创建函数
func NewMySender(config map[string]interface{}) (*MySender, error) {
    sender := &MySender{}

    // 自己创建 logger
    if l, ok := config["logger"].(logger.Logger); ok {
        sender.logger = l
    } else {
        sender.logger = logger.NewWithPrefix("[myplatform]", logger.Warn)
    }
    return sender, nil
}

// 旧的日志调用方式
func (s *MySender) Send(ctx context.Context, msg *Message) error {
    s.logger.Debug(ctx, "Sending message to %s", msg.To)
}
```

**迁移后:**

```go
// 新的平台 Creator 函数签名，接收一个 logger
func NewMySender(config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
    sender := &MySender{
        logger: logger, // 直接从参数赋值
    }
    // 如果 logger 为 nil，在方法调用时处理
    return sender, nil
}

// 新的日志调用方式 (slog 风格)
func (s *MySender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
    if s.logger == nil {
        s.logger = logger.Discard // 安全保护
    }
    s.logger.Debug("Sending message", "target_count", len(targets), "messageID", msg.ID)
    // ...
    return nil, nil
}
```

## 🎯 新的 API

### Logger 接口

```go
type Logger interface {
    LogMode(level LogLevel) Logger
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    Debug(msg string, args ...any)
}
```

### Hub 配置

```go
// 推荐的配置方式
hub, err := notifyhub.New(
    // ... platforms
    notifyhub.WithLogger(myLogger),
)
```

## ✅ 检查清单

- [ ] 移除所有对 `logger.Default` 和 `logger.NewWithPrefix` 的调用。
- [ ] 更新 `notifyhub.New` 或 `notifyhub.NewHub` 的调用，使用 `notifyhub.WithLogger` 注入一个中心 logger。
- [ ] 移除所有特定于平台的日志选项，如 `email.WithEmailLogger` 和 `email.WithEmailLogLevel`。
- [ ] 如果你是平台开发者，请更新你的 `Creator` 函数以接收 `logger.Logger`。
- [ ] 更新所有日志调用为 `slog` 风格：`logger.Info("message", "key", value)`。