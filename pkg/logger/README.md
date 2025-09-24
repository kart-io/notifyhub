# NotifyHub Logger

参考 GORM 和 `slog` 设计的项目级别 Logger，为整个 NotifyHub 提供统一的日志接口。

## 🎯 设计理念

- **统一接口**: 所有平台和组件使用相同的 `slog` 风格的 Logger 接口。
- **依赖注入**: 日志实例通过 `notifyhub.New` 的 `WithLogger` 选项注入，彻底解耦。
- **可扩展**: 允许轻松接入 `slog`, `zap` 等任何第三方日志库。
- **结构化日志**: 日志方法接受键值对参数，方便机器解析和查询。

## 📊 日志级别

```go
const (
    Silent LogLevel = iota + 1  // 无日志输出
    Error                        // 只记录错误
    Warn                         // 记录警告和错误
    Info                         // 记录信息、警告和错误
    Debug                        // 记录所有日志包括调试信息
)
```

## 🚀 快速开始

```go
import (
	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
	// 1. 创建一个 NotifyHub logger 实例
	appLogger := logger.New() // 使用默认的 StandardLogger

	// 2. 通过 WithLogger 选项将其注入 Hub
	hub, err := notifyhub.New(
		// ... 其他平台配置 ...
		notifyhub.WithLogger(appLogger.LogMode(logger.Debug)), // 设置日志级别
	)
	if err != nil {
		log.Fatal(err)
	}

	// 3. 完成！所有平台现在都会使用你提供的 logger 实例。
}
```

## 📝 Logger 接口

```go
type Logger interface {
    LogMode(level LogLevel) Logger
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    Debug(msg string, args ...any)
}
```

## 🔇 静默模式

如果不提供任何 logger，NotifyHub 默认使用 `logger.Discard`，它会忽略所有日志输出。

```go
// 默认情况下，不会输出任何日志
hub, _ := notifyhub.New(...)

// 你也可以显式使用 Discard logger
hub, _ := notifyhub.New(
    notifyhub.WithLogger(logger.Discard),
)
```

## 🏗️ 平台集成

日志记录器现在通过 Hub 的配置自动传递给每个平台。平台作者不再需要（也不应该）创建自己的 logger 实例。

### 实现示例

```go
package myplatform

import "github.com/kart-io/notifyhub/pkg/logger"

// Sender struct 包含一个 logger 字段
type MySender struct {
    logger logger.Logger
    // ... 其他字段
}

// Creator 函数现在接收一个 logger 实例
func NewMySender(config map[string]interface{}, logger logger.Logger) (*MySender, error) {
    sender := &MySender{
        logger: logger, // 直接赋值
    }
    return sender, nil
}

// 在 Send 方法中使用 slog 风格的日志
func (s *MySender) Send(ctx context.Context, msg *Message) error {
    s.logger.Debug("Sending message", "to", msg.To, "messageID", msg.ID)

    if err := s.doSend(msg); err != nil {
        s.logger.Error("Send failed", "error", err, "messageID", msg.ID)
        return err
    }

    s.logger.Info("Message sent successfully", "to", msg.To, "messageID", msg.ID)
    return nil
}
```

## 🧪 测试

在测试中，你可以轻松地将日志输出到 `bytes.Buffer` 以便断言。

```go
import (
    "bytes"
    "log/slog"
    "testing"
    "github.com/kart-io/notifyhub/pkg/logger"
)

func TestMyPlatform(t *testing.T) {
    var buf bytes.Buffer
	slogHandler := slog.NewTextHandler(&buf, nil)
	logger := logger.NewSlogAdapter(slog.New(slogHandler))

    // 将 logger 传递给你的平台
    mySender := &MySender{logger: logger}
    mySender.DoSomethingThatLogs()

    if !strings.Contains(buf.String(), "expected log message") {
        t.Errorf("Log output did not contain expected message. Got: %s", buf.String())
    }
}
```

## 🎯 最佳实践

### 1. 集中配置

在你的应用程序的最高层（例如 `main.go`）配置一次 logger，然后通过 `WithLogger` 将其注入 NotifyHub。让依赖注入来处理日志的传递。

### 2. 使用结构化日志

充分利用键值对格式来记录上下文信息，这会让日志在生产环境中更易于查询和分析。

**好的实践:**
`logger.Error("Failed to process payment", "error", err, "userID", user.ID, "orderID", order.ID)`

**不好的实践:**
`logger.Error(fmt.Sprintf("Error processing payment for user %s, order %s: %v", user.ID, order.ID, err))`

### 3. 通过环境控制级别

```go
level := slog.LevelInfo
if os.Getenv("DEBUG") == "true" {
    level = slog.LevelDebug
}
slogHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
// ...
```

### 4. 输出到文件

```go
file, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
slogHandler := slog.NewJSONHandler(file, nil) // JSON 格式更适合文件
// ...
```

## 🚀 扩展

### 实现自定义 Logger 适配器

如果你想使用 `zap`，只需创建一个 `ZapAdapter`。

```go
import "go.uber.org/zap"

type ZapAdapter struct {
    *zap.SugaredLogger
}

func (a *ZapAdapter) LogMode(level logger.LogLevel) logger.Logger {
    // zap 不支持动态级别切换，可以返回一个新的实例或忽略
    return a
}

func (a *ZapAdapter) Info(msg string, args ...any) {
    a.Infow(msg, args...)
}

// ... 实现 Warn, Error, Debug ...
```
