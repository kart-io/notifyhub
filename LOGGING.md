# 📝 NotifyHub 日志系统

NotifyHub 提供了强大而灵活的日志系统，参考 GORM 的设计，支持多种日志级别和适配器，完全兼容调用者现有的日志库。

## ✨ 主要特性

- **多级别日志**: Silent, Error, Warn, Info, Debug
- **兼容性**: 支持标准 log、logrus、zap 等主流日志库
- **性能监控**: 自动记录操作耗时和慢操作告警
- **色彩输出**: 支持彩色控制台输出
- **灵活配置**: 可完全静默或自定义日志格式
- **详细追踪**: 提供完整的操作链路追踪

## 🎯 日志级别

```go
const (
    LogLevelSilent = logger.Silent  // 静默模式
    LogLevelError  = logger.Error   // 仅错误
    LogLevelWarn   = logger.Warn    // 警告及以上
    LogLevelInfo   = logger.Info    // 信息及以上
    LogLevelDebug  = logger.Debug   // 调试及以上
)
```

## 🚀 快速开始

### 1. 使用默认日志器

```go
// 使用默认日志器，Warn 级别
hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    // 默认已包含 Warn 级别日志
)

// 或者显式指定级别
hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithDefaultLogger(notifyhub.LogLevelInfo),
)
```

### 2. 静默模式（无日志）

```go
hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithSilentLogger(), // 完全静默
)
```

### 3. 调试模式（详细日志）

```go
hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithDefaultLogger(notifyhub.LogLevelDebug),
)
```

## 🔌 日志适配器

NotifyHub 支持多种主流日志库，包括专门为 `github.com/kart-io/logger` 设计的适配器。

### 标准 log 包

```go
import "log"

customLogger := log.New(os.Stdout, "[NOTIFYHUB] ", log.LstdFlags)

hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(
        notifyhub.NewStdLogAdapter(customLogger, notifyhub.LogLevelInfo),
    ),
)
```

### Logrus 适配器

```go
import "github.com/sirupsen/logrus"

logrusLogger := logrus.New()
logrusLogger.SetLevel(logrus.InfoLevel)

hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(
        notifyhub.NewLogrusAdapter(logrusLogger, notifyhub.LogLevelInfo),
    ),
)
```

### Zap 适配器

```go
import "go.uber.org/zap"

zapLogger, _ := zap.NewProduction()
sugar := zapLogger.Sugar()

hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(
        notifyhub.NewZapAdapter(sugar, notifyhub.LogLevelInfo),
    ),
)
```

### Kart Logger 适配器（github.com/kart-io/logger）

NotifyHub 提供了专门的适配器来支持 `github.com/kart-io/logger`：

```go
import "github.com/kart-io/logger"

// 方式1: 使用完整的 Kart Logger（支持 WithField/WithFields）
kartLogger := logger.New() // 你的 Kart Logger 实例

hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(
        notifyhub.NewKartLoggerAdapter(kartLogger, notifyhub.LogLevelInfo),
    ),
)

// 方式2: 使用简化版（不支持 WithField 方法的实现）
simpleKartLogger := logger.NewSimple() // 简化版实例

hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(
        notifyhub.NewSimpleKartLoggerAdapter(simpleKartLogger, notifyhub.LogLevelInfo),
    ),
)
```

**Kart Logger 接口要求：**

完整版适配器期望的接口：
```go
type KartLogger interface {
    // 基础日志方法
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})

    // 格式化日志方法
    Debugf(format string, args ...interface{})
    Infof(format string, args ...interface{})
    Warnf(format string, args ...interface{})
    Errorf(format string, args ...interface{})

    // 结构化日志方法（可选）
    WithField(key string, value interface{}) interface{}
    WithFields(fields map[string]interface{}) interface{}
}
```

简化版适配器接口：
```go
type SimpleKartLogger interface {
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})

    Debugf(format string, args ...interface{})
    Infof(format string, args ...interface{})
    Warnf(format string, args ...interface{})
    Errorf(format string, args ...interface{})
}
```

### 自定义适配器框架（全新模块化设计）

NotifyHub 的日志系统已经完全重构为模块化架构，提供了极其灵活的自定义适配器框架。新架构将所有适配器拆分为独立文件，便于维护和扩展。

#### 🏗️ 模块化架构

```
logger/
├── adapter_base.go     - 基础适配器和自定义框架
├── adapter_std.go      - 标准 log 包适配器
├── adapter_logrus.go   - Logrus 适配器
├── adapter_zap.go      - Zap 适配器
└── adapter_kart.go     - Kart Logger 适配器
```

#### 🎯 自定义适配器接口

创建自定义适配器只需实现一个简单的接口：

```go
import "github.com/kart-io/notifyhub/logger"

// CustomLogger 接口 - 只需实现一个方法！
type CustomLogger interface {
    Log(level LogLevel, msg string, fields map[string]interface{})
}

// 实现你的自定义日志器
type MyCustomLogger struct {
    // 你的配置
}

func (m *MyCustomLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
    // 实现你的日志逻辑
    // level: 日志级别 (Silent, Error, Warn, Info, Debug)
    // msg: 日志消息
    // fields: 结构化字段数据
}

// 使用自定义适配器
customLogger := &MyCustomLogger{}
hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(
        notifyhub.NewCustomAdapter(customLogger, notifyhub.LogLevelInfo),
    ),
)
```

#### 🌟 完整示例集合

NotifyHub 提供了多种开箱即用的自定义适配器实现：

**1. 控制台日志器（带前缀）：**
```go
type ConsoleLogger struct {
    prefix string
}

func (c *ConsoleLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    output := fmt.Sprintf("[%s] [%s] %s%s", timestamp, level.String(), c.prefix, msg)

    if len(fields) > 0 {
        output += " fields:"
        for k, v := range fields {
            output += fmt.Sprintf(" %s=%v", k, v)
        }
    }

    fmt.Println(output)
}
```

**2. JSON 结构化日志器：**
```go
type JSONLogger struct {
    serviceName string
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

**3. 文件日志器：**
```go
type FileLogger struct {
    file   *os.File
    prefix string
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
```

**4. 多目标日志器：**
```go
type MultiTargetLogger struct {
    targets []logger.CustomLogger
}

func (m *MultiTargetLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
    // 同时输出到多个目标
    for _, target := range m.targets {
        target.Log(level, msg, fields)
    }
}

// 创建多目标日志器
consoleTarget := NewConsoleLogger("[MULTI-CONSOLE] ")
jsonTarget := NewJSONLogger("multi-target-service")
fileTarget, _ := NewFileLogger("/tmp/notifyhub-multi.log", "[MULTI-FILE] ")

multiLogger := NewMultiTargetLogger(consoleTarget, jsonTarget, fileTarget)

hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(
        notifyhub.NewCustomAdapter(multiLogger, notifyhub.LogLevelInfo),
    ),
)
```

### 函数适配器（轻量级）

如果你只需要简单的函数式日志，可以使用函数适配器：

```go
// 自定义日志函数
logFunc := func(level string, msg string, keyvals ...interface{}) {
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    fmt.Printf("[%s] [%s] %s %v\\n", timestamp, level, msg, keyvals)
}

hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(
        notifyhub.NewFuncAdapter(logFunc, notifyhub.LogLevelInfo),
    ),
)
```

## 📊 日志输出示例

### Info 级别输出

```
2025-09-14 13:32:48 [info] NotifyHub initializing with config: queue_type=memory, buffer_size=100, workers=1
2025-09-14 13:32:48 [info] Feishu notifier initialized with webhook: https://httpbin.org/***n.org/post
2025-09-14 13:32:48 [info] NotifyHub initialized successfully with 1 notifiers: [feishu]
2025-09-14 13:32:48 [info] Starting NotifyHub services...
2025-09-14 13:32:48 [info] Queue workers started successfully: 1 workers
2025-09-14 13:32:48 [info] NotifyHub started successfully
2025-09-14 13:32:50 [info] Notifier feishu succeeded: 1 results (took 1.24751325s)
2025-09-14 13:32:50 [info] Message send completed: 1 successful results, 0 errors
```

### Debug 级别输出（更详细）

```
2025-09-14 13:32:50 [debug] Starting synchronous message send: title='调试报告', priority=2, targets=1
2025-09-14 13:32:50 [debug] Message processed through routing and template rendering
2025-09-14 13:32:50 [debug] Sending message via feishu notifier
2025-09-14 13:32:50 [info] Notifier feishu succeeded: 1 results (took 319.788958ms)
```

### 性能追踪（慢操作告警）

```
2025-09-14 13:32:50 SLOW OPERATION >= 200ms [1247.630ms] [rows:1] Send message '测试消息' to 1 notifiers
```

## ⚡ 性能监控

日志系统自动监控和记录：

- **操作耗时**: 每个通知器的发送时间
- **慢操作告警**: 超过 200ms 的操作会被标记
- **成功/失败统计**: 详细的发送结果统计
- **错误追踪**: 完整的错误信息和堆栈

## 🎨 自定义日志器

如果现有适配器不满足需求，可以实现自定义日志器：

```go
import "github.com/kart-io/notifyhub/logger"

type MyCustomLogger struct {
    // 你的日志器实现
}

// 实现 logger.Interface 接口
func (l *MyCustomLogger) LogMode(level logger.LogLevel) logger.Interface {
    // 实现逻辑
}

func (l *MyCustomLogger) Info(ctx context.Context, msg string, data ...interface{}) {
    // 实现逻辑
}

func (l *MyCustomLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
    // 实现逻辑
}

func (l *MyCustomLogger) Error(ctx context.Context, msg string, data ...interface{}) {
    // 实现逻辑
}

func (l *MyCustomLogger) Debug(ctx context.Context, msg string, data ...interface{}) {
    // 实现逻辑
}

func (l *MyCustomLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
    // 实现逻辑
}

// 使用自定义日志器
hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(&MyCustomLogger{}),
)
```

## 🔧 配置选项

### 环境变量支持

```bash
# 可以通过环境变量控制日志级别
export NOTIFYHUB_LOG_LEVEL=debug
export NOTIFYHUB_LOG_COLORFUL=true
```

### 程序配置

```go
// 方式1: 使用预设配置
hub, err := notifyhub.New(
    notifyhub.WithDefaults(), // 包含 Warn 级别日志
)

// 方式2: 测试友好配置
hub, err := notifyhub.New(
    notifyhub.WithTestDefaults(), // 包含 Debug 级别日志
)

// 方式3: 完全自定义
hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithDefaultLogger(notifyhub.LogLevelInfo),
)
```

## 🔍 日志内容

日志系统记录以下关键信息：

### 系统级别
- Hub 初始化和配置信息
- 通知器初始化状态
- 服务启动/停止状态
- 队列工作器状态

### 消息级别
- 消息发送开始/完成
- 路由和模板处理
- 每个通知器的执行结果
- 异步消息入队状态

### 错误级别
- 通知器连接失败
- 消息发送失败详情
- 模板渲染错误
- 系统异常信息

### 性能级别
- 操作执行时间
- 慢操作告警
- 吞吐量统计
- 资源使用情况

## 💡 最佳实践

1. **生产环境**: 建议使用 `Warn` 或 `Error` 级别
2. **开发环境**: 可以使用 `Debug` 级别获取详细信息
3. **测试环境**: 可以使用 `Silent` 避免日志干扰
4. **性能监控**: 关注慢操作告警，优化性能
5. **错误处理**: 通过 `Error` 级别日志快速定位问题

## 📚 示例代码

完整的示例代码请参考：

- `examples/custom-logger/main.go` - **🆕 自定义适配器框架完整演示**
- `examples/logging/main.go` - 完整日志演示
- `examples/kart-logger/main.go` - Kart Logger 适配器演示
- `examples/config/main.go` - 配置示例
- `examples/multi-platform-demo/main.go` - 实际使用场景

### 🚀 新增自定义适配器示例

`examples/custom-logger/main.go` 包含了完整的自定义适配器框架演示：

1. **控制台日志器** - 带前缀的彩色控制台输出
2. **JSON日志器** - 结构化JSON格式输出
3. **文件日志器** - 写入文件的日志器
4. **多目标日志器** - 同时输出到控制台、JSON和文件

运行示例：
```bash
cd examples/custom-logger
go run main.go
```

输出效果：
```
# 控制台输出
[2025-09-14 13:55:28] [info] [NOTIFYHUB-CONSOLE] NotifyHub initializing with config fields: queue_type=memory buffer_size=50

# JSON 输出
{"level":"info","service":"notifyhub-service","timestamp":"2025-09-14T13:55:29+08:00","message":"NotifyHub initializing with config","queue_type":"memory","buffer_size":50}

# 文件输出到 /tmp/notifyhub.log 和 /tmp/notifyhub-multi.log
[2025-09-14 13:55:30] [info] [MULTI-FILE] NotifyHub initialized successfully with 1 notifiers: [feishu]
```

---

💡 **总结**: NotifyHub 的日志系统已完全重构为**模块化架构**，提供了：

- **🏗️ 模块化设计**: 将大型适配器文件拆分为多个专门文件，便于维护和扩展
- **🎯 简化接口**: 自定义适配器只需实现一个 `Log` 方法，极其简单
- **🌟 丰富示例**: 提供控制台、JSON、文件、多目标等多种完整实现
- **🔌 GORM级兼容性**: 支持标准log、logrus、zap、Kart Logger等主流日志库
- **⚡ 高性能**: 基于接口的设计，运行时开销最小
- **🎨 极度灵活**: 支持任意格式和输出目标的自定义日志器

新架构让你可以无缝集成到现有的日志体系中，同时获得详细的操作监控和性能追踪能力。