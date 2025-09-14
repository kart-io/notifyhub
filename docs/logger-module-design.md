# NotifyHub 日志模块技术设计 (v1.1)

> **对应需求**: `FR13` (可插拔日志系统), `NFR5` (可观测性)

## 1. 设计目标

日志模块是系统可观测性的基础。其设计旨在实现以下目标：

*   **统一接口**: `notifyhub` 内部所有组件都使用此标准接口记录日志，避免直接依赖任何具体的日志库。
*   **可插拔性**: 允许库的使用者（开发者）轻松地将 `notifyhub` 的日志输出集成到其项目已有的日志系统中（如 `zap`, `logrus`）。
*   **结构化日志**: 接口设计上鼓励和支持通过 `key-value` 形式的字段来记录上下文信息，便于后续的日志查询和分析。
*   **日志级别**: 支持标准的日志级别（Debug, Info, Warn, Error）。

## 2. 核心接口定义 (`logger/interface.go`)

```go
package logger

// Field 是结构化日志的一个键值对字段
type Field struct {
    Key   string
    Value interface{}
}

// Logger 定义了 notifyhub 内部使用的标准日志接口
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)

    // With 返回一个新的 Logger 实例，该实例会自动附加一些固定的上下文字段
    With(fields ...Field) Logger
}
```

## 3. 实现模式

### 3.1. 默认实现 (`logger/logger.go`)

*   `notifyhub` 会提供一个开箱即用的默认日志实现。
*   这个实现非常简单，通常是基于 Go 标准库的 `log` 包进行封装，将日志信息格式化后输出到标准输出（`stdout`）。
*   它的主要目的是保证用户在不进行任何日志配置的情况下，系统也能正常输出关键信息。

### 3.2. 适配器模式 (`logger/adapters/`)

为了让用户能够集成自己偏好的、功能更强大的日志库，系统采用适配器模式。

*   **理念**: 用户可以创建一个结构体，它内嵌了用户自己的日志库实例（如 `*zap.Logger`），并让这个结构体实现 `logger.Logger` 接口。
*   **示例：实现一个 `zap` 适配器**

    ```go
    package adapters

    import (
        "github.com/kart/notifyhub/logger"
        "go.uber.org/zap"
    )

    type ZapAdapter struct {
        zapLogger *zap.Logger
    }

    func NewZapAdapter(l *zap.Logger) *ZapAdapter {
        return &ZapAdapter{zapLogger: l}
    }

    func (a *ZapAdapter) Info(msg string, fields ...logger.Field) {
        zapFields := a.toZapFields(fields)
        a.zapLogger.Info(msg, zapFields...)
    }

    // ... 实现 Debug, Warn, Error, With 等其他方法 ...

    func (a *ZapAdapter) toZapFields(fields []logger.Field) []zap.Field {
        // ... 将 logger.Field 转换为 zap.Field 的逻辑 ...
        return zapFields
    }
    ```

## 4. 使用与配置

用户可以在初始化 `Hub` 时，通过 `WithLogger` 选项，将自己实现的日志适配器实例注入到系统中，从而全面接管日志输出。

```go
// 在用户的 main.go 中

import (
    "go.uber.org/zap"
    "github.com/kart/notifyhub"
    "my-project/log_adapters" // 假设这是用户自己的适配器包
)

func main() {
    // 1. 初始化用户自己的日志库
    zapLogger, _ := zap.NewProduction()
    defer zapLogger.Sync()

    // 2. 创建适配器实例
    myLogger := log_adapters.NewZapAdapter(zapLogger)

    // 3. 通过选项注入 Hub
    hub, err := notifyhub.New(
        notifyhub.WithLogger(myLogger),
        // ... 其他配置
    )
    // ...
}
```
