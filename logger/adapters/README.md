# 🔌 NotifyHub Logger Adapters

NotifyHub 日志适配器提供了灵活的方式来集成各种日志库，支持从标准库到企业级日志解决方案的无缝接入。

## 📋 适配器列表

| 适配器 | 接口 | 特点 | 使用场景 |
|--------|------|------|----------|
| **CustomAdapter** | `CustomLogger` | 极简接口，只需实现一个方法 | 自定义日志需求 |
| **FuncAdapter** | `LogFunc` | 函数式，轻量级 | 简单回调场景 |
| **StdLogAdapter** | `StdLogger` | 兼容标准库 | 基础项目 |
| **LogrusAdapter** | `LogrusLogger` | 结构化日志 | 中型项目 |
| **ZapAdapter** | `ZapLogger` | 高性能 | 高并发场景 |
| **KartLoggerAdapter** | `KartLogger` | 企业级，WithField支持 | 企业项目 |
| **SimpleKartLoggerAdapter** | `SimpleKartLogger` | 简化版Kart Logger | 轻量企业场景 |

## 🎯 自定义适配器框架

### 核心接口

自定义适配器只需实现一个简单的接口：

```go
type CustomLogger interface {
    Log(level logger.LogLevel, msg string, fields map[string]interface{})
}
```

### 快速开始

```go
package main

import (
    "fmt"
    "time"
    "github.com/kart-io/notifyhub"
    "github.com/kart-io/notifyhub/logger"
    "github.com/kart-io/notifyhub/logger/adapters"
)

// 1. 实现CustomLogger接口
type MyLogger struct {
    prefix string
}

func (m *MyLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    output := fmt.Sprintf("[%s] [%s] %s%s", timestamp, level.String(), m.prefix, msg)

    if len(fields) > 0 {
        output += " fields:"
        for k, v := range fields {
            output += fmt.Sprintf(" %s=%v", k, v)
        }
    }

    fmt.Println(output)
}

// 2. 使用自定义适配器
func main() {
    customLogger := &MyLogger{prefix: "[MyApp] "}

    hub, err := notifyhub.New(
        notifyhub.WithFeishu("webhook-url", "secret"),
        notifyhub.WithLogger(
            notifyhub.NewCustomAdapter(customLogger, notifyhub.LogLevelInfo),
        ),
    )

    // ... 使用hub
}
```

## 📋 标准适配器使用

### 1. 标准库适配器

```go
import (
    "log"
    "os"
    "github.com/kart-io/notifyhub"
)

func useStdLogAdapter() {
    stdLogger := log.New(os.Stdout, "[NOTIFYHUB] ", log.LstdFlags)

    hub, err := notifyhub.New(
        notifyhub.WithFeishu("webhook-url", "secret"),
        notifyhub.WithLogger(
            notifyhub.NewStdLogAdapter(stdLogger, notifyhub.LogLevelInfo),
        ),
    )
}
```

### 2. Logrus适配器

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/kart-io/notifyhub"
)

func useLogrusAdapter() {
    logrusLogger := logrus.New()
    logrusLogger.SetLevel(logrus.InfoLevel)
    logrusLogger.SetFormatter(&logrus.JSONFormatter{})

    hub, err := notifyhub.New(
        notifyhub.WithFeishu("webhook-url", "secret"),
        notifyhub.WithLogger(
            notifyhub.NewLogrusAdapter(logrusLogger, notifyhub.LogLevelInfo),
        ),
    )
}
```

### 3. Zap适配器

```go
import (
    "go.uber.org/zap"
    "github.com/kart-io/notifyhub"
)

func useZapAdapter() {
    zapLogger, _ := zap.NewProduction()
    sugar := zapLogger.Sugar()

    hub, err := notifyhub.New(
        notifyhub.WithFeishu("webhook-url", "secret"),
        notifyhub.WithLogger(
            notifyhub.NewZapAdapter(sugar, notifyhub.LogLevelInfo),
        ),
    )
}
```

### 4. Kart Logger适配器

```go
import (
    "github.com/kart-io/logger"  // 企业内部日志库
    "github.com/kart-io/notifyhub"
)

func useKartLoggerAdapter() {
    kartLogger := logger.New()  // 你的Kart Logger实例

    hub, err := notifyhub.New(
        notifyhub.WithFeishu("webhook-url", "secret"),
        notifyhub.WithLogger(
            notifyhub.NewKartLoggerAdapter(kartLogger, notifyhub.LogLevelInfo),
        ),
    )
}

func useSimpleKartLoggerAdapter() {
    simpleKartLogger := logger.NewSimple()  // 简化版实例

    hub, err := notifyhub.New(
        notifyhub.WithFeishu("webhook-url", "secret"),
        notifyhub.WithLogger(
            notifyhub.NewSimpleKartLoggerAdapter(simpleKartLogger, notifyhub.LogLevelInfo),
        ),
    )
}
```

### 5. 函数适配器

```go
func useFuncAdapter() {
    logFunc := func(level string, msg string, keyvals ...interface{}) {
        timestamp := time.Now().Format("2006-01-02 15:04:05")
        fmt.Printf("[%s] [%s] %s %v\n", timestamp, level, msg, keyvals)
    }

    hub, err := notifyhub.New(
        notifyhub.WithFeishu("webhook-url", "secret"),
        notifyhub.WithLogger(
            notifyhub.NewFuncAdapter(logFunc, notifyhub.LogLevelInfo),
        ),
    )
}
```

## 🌟 高级自定义适配器示例

### JSON结构化日志器

```go
type JSONLogger struct {
    serviceName string
}

func NewJSONLogger(serviceName string) *JSONLogger {
    return &JSONLogger{serviceName: serviceName}
}

func (j *JSONLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
    logEntry := map[string]interface{}{
        "timestamp": time.Now().Format(time.RFC3339),
        "level":     level.String(),
        "service":   j.serviceName,
        "message":   msg,
    }

    // 合并字段
    for k, v := range fields {
        logEntry[k] = v
    }

    jsonData, _ := json.Marshal(logEntry)
    fmt.Println(string(jsonData))
}
```

### 文件日志器

```go
type FileLogger struct {
    file   *os.File
    prefix string
}

func NewFileLogger(filename, prefix string) (*FileLogger, error) {
    file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return nil, err
    }

    return &FileLogger{
        file:   file,
        prefix: prefix,
    }, nil
}

func (f *FileLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    output := fmt.Sprintf("[%s] [%s] %s%s", timestamp, level.String(), f.prefix, msg)

    if len(fields) > 0 {
        output += " fields:"
        for k, v := range fields {
            output += fmt.Sprintf(" %s=%v", k, v)
        }
    }

    output += "\n"
    f.file.WriteString(output)
}

func (f *FileLogger) Close() error {
    return f.file.Close()
}
```

### 多目标日志器

```go
type MultiTargetLogger struct {
    targets []adapters.CustomLogger
}

func NewMultiTargetLogger(targets ...adapters.CustomLogger) *MultiTargetLogger {
    return &MultiTargetLogger{targets: targets}
}

func (m *MultiTargetLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
    // 同时输出到多个目标
    for _, target := range m.targets {
        target.Log(level, msg, fields)
    }
}

// 使用示例
func useMultiTargetLogger() {
    consoleLogger := NewConsoleLogger("[CONSOLE] ")
    jsonLogger := NewJSONLogger("my-service")
    fileLogger, _ := NewFileLogger("/tmp/app.log", "[FILE] ")

    multiLogger := NewMultiTargetLogger(consoleLogger, jsonLogger, fileLogger)

    hub, err := notifyhub.New(
        notifyhub.WithFeishu("webhook-url", "secret"),
        notifyhub.WithLogger(
            notifyhub.NewCustomAdapter(multiLogger, notifyhub.LogLevelInfo),
        ),
    )
}
```

## 🔧 接口规范

### CustomLogger接口
```go
type CustomLogger interface {
    Log(level logger.LogLevel, msg string, fields map[string]interface{})
}
```

**参数说明**:
- `level`: 日志级别 (Silent, Error, Warn, Info, Debug)
- `msg`: 日志消息
- `fields`: 结构化字段数据

### LogFunc类型
```go
type LogFunc func(level string, msg string, keyvals ...interface{})
```

**参数说明**:
- `level`: 日志级别字符串 ("silent", "error", "warn", "info", "debug")
- `msg`: 日志消息
- `keyvals`: 键值对参数 (key1, value1, key2, value2, ...)

### 标准库接口

**StdLogger**:
```go
type StdLogger interface {
    Print(v ...interface{})
    Printf(format string, v ...interface{})
}
```

**LogrusLogger**:
```go
type LogrusLogger interface {
    Debug(args ...interface{})
    Info(args ...interface{})
    Warn(args ...interface{})
    Error(args ...interface{})
    Debugf(format string, args ...interface{})
    Infof(format string, args ...interface{})
    Warnf(format string, args ...interface{})
    Errorf(format string, args ...interface{})
}
```

## 📊 性能对比

| 适配器 | 内存占用 | 性能 | 功能丰富度 | 推荐场景 |
|--------|----------|------|------------|----------|
| Custom | 最低 | 最高 | ⭐⭐⭐⭐⭐ | 定制需求 |
| Func | 低 | 高 | ⭐⭐⭐ | 简单场景 |
| StdLog | 低 | 中 | ⭐⭐ | 基础项目 |
| Logrus | 中 | 中 | ⭐⭐⭐⭐ | 中型项目 |
| Zap | 低 | 最高 | ⭐⭐⭐⭐ | 高性能场景 |
| KartLogger | 中 | 高 | ⭐⭐⭐⭐⭐ | 企业项目 |

## 💡 最佳实践

### 1. 适配器选择
- **高性能要求**: 选择Zap或Custom适配器
- **企业环境**: 选择KartLogger适配器
- **快速原型**: 选择Func或StdLog适配器
- **结构化日志**: 选择Logrus或Custom适配器

### 2. 错误处理
```go
func (c *CustomLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
    defer func() {
        if r := recover(); r != nil {
            // 记录适配器内部错误，避免影响主程序
            fmt.Fprintf(os.Stderr, "Logger adapter error: %v\n", r)
        }
    }()

    // 你的日志逻辑
}
```

### 3. 资源管理
```go
type FileLogger struct {
    file *os.File
    mu   sync.Mutex  // 保护并发写入
}

func (f *FileLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
    f.mu.Lock()
    defer f.mu.Unlock()

    // 安全的文件写入
    f.file.WriteString(formatLog(level, msg, fields))
}
```

## 🧪 测试

完整的适配器示例和测试代码位于：
- [`../../examples/custom-logger/main.go`](../../examples/custom-logger/main.go)

运行示例：
```bash
cd examples/custom-logger
go run main.go
```

## 📚 参考资料

- [Logger核心接口](../interface.go)
- [适配器实现](./adapters.go)
- [使用示例](../../examples/custom-logger/)
- [主文档](../README.md)

---

💡 **提示**: 自定义适配器框架的设计目标是简单易用，如果你只需要基本的日志功能，一个10行的实现就足够了！