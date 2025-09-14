# 📋 NotifyHub Logger Interface Documentation

本文档详细介绍了 NotifyHub 日志系统的核心接口定义和使用说明。

## 🎯 核心接口

### Logger Interface

主要的日志记录接口，所有日志实现都必须满足此接口：

```go
type Interface interface {
    // LogMode 设置日志级别并返回新的日志器实例
    LogMode(level LogLevel) Interface

    // Info 记录信息级别日志
    Info(ctx context.Context, msg string, data ...interface{})

    // Warn 记录警告级别日志
    Warn(ctx context.Context, msg string, data ...interface{})

    // Error 记录错误级别日志
    Error(ctx context.Context, msg string, data ...interface{})

    // Debug 记录调试级别日志
    Debug(ctx context.Context, msg string, data ...interface{})

    // Trace 记录操作追踪，包含耗时统计
    Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error)
}
```

### 方法详解

#### LogMode(level LogLevel) Interface

设置日志级别并返回新的日志器实例。

**参数**:
- `level`: 新的日志级别

**返回值**:
- 新的日志器实例（不修改原实例）

**示例**:
```go
debugLogger := logger.LogMode(logger.Debug)
prodLogger := logger.LogMode(logger.Warn)
```

#### Info/Warn/Error/Debug 方法

记录指定级别的日志消息。

**参数**:
- `ctx`: 上下文对象，可用于传递请求ID等信息
- `msg`: 日志消息文本
- `data`: 可变参数，支持键值对格式 (key1, value1, key2, value2, ...)

**示例**:
```go
// 基础使用
logger.Info(ctx, "用户登录成功")

// 带结构化数据
logger.Info(ctx, "用户登录", "user_id", 12345, "ip", "192.168.1.100")

// 错误日志
logger.Error(ctx, "数据库连接失败", "error", err.Error(), "retry_count", 3)
```

#### Trace(ctx, begin, fc, err)

记录操作追踪信息，包含耗时统计和性能监控。

**参数**:
- `ctx`: 上下文对象
- `begin`: 操作开始时间
- `fc`: 回调函数，返回操作描述和影响行数
- `err`: 操作过程中的错误（如果有）

**回调函数签名**:
```go
func() (operation string, affected int64)
```

**示例**:
```go
start := time.Now()

// 执行操作...
result, err := performOperation()

// 记录追踪
logger.Trace(ctx, start, func() (string, int64) {
    return "发送消息到Feishu", int64(result.Count)
}, err)
```

## 📊 日志级别

### LogLevel 类型

```go
type LogLevel int

const (
    Silent LogLevel = iota + 1  // 1: 静默模式，不输出任何日志
    Error                       // 2: 仅输出错误日志
    Warn                        // 3: 输出警告和错误日志
    Info                        // 4: 输出信息、警告和错误日志
    Debug                       // 5: 输出所有级别日志（最详细）
)
```

### 级别说明

| 级别 | 值 | 说明 | 使用场景 |
|------|----|----- |----------|
| `Silent` | 1 | 静默模式 | 测试环境，不需要日志输出 |
| `Error` | 2 | 仅错误 | 生产环境，只关注错误 |
| `Warn` | 3 | 警告及以上 | 生产环境默认级别 |
| `Info` | 4 | 信息及以上 | 开发和调试环境 |
| `Debug` | 5 | 调试级别 | 详细调试信息 |

### 级别比较逻辑

```go
// 当 logger.LogLevel >= 目标级别时，日志会被输出
if logger.LogLevel >= Info {
    // 输出Info级别日志
}

// 示例：如果 LogLevel = Warn (3)
// - Error (2) 会被输出 (因为 3 >= 2)
// - Warn (3) 会被输出 (因为 3 >= 3)
// - Info (4) 不会被输出 (因为 3 < 4)
// - Debug (5) 不会被输出 (因为 3 < 5)
```

## ⚙️ 配置结构

### Config 结构体

```go
type Config struct {
    // SlowThreshold 慢操作阈值，超过此时间的操作会被标记为慢操作
    SlowThreshold time.Duration

    // LogLevel 日志输出级别
    LogLevel LogLevel

    // Colorful 是否启用彩色输出（仅对控制台输出有效）
    Colorful bool

    // IgnoreRecordNotFoundError 是否忽略"记录未找到"错误
    IgnoreRecordNotFoundError bool
}
```

### 配置说明

#### SlowThreshold
- 类型: `time.Duration`
- 默认值: `200ms`
- 说明: 当操作耗时超过此阈值时，会在日志中标记为"SLOW OPERATION"

```go
config := Config{
    SlowThreshold: 500 * time.Millisecond,  // 500ms以上视为慢操作
}
```

#### LogLevel
- 类型: `LogLevel`
- 默认值: `Warn`
- 说明: 控制日志输出的详细程度

#### Colorful
- 类型: `bool`
- 默认值: `true`
- 说明: 是否在控制台使用彩色输出

彩色方案：
- 🟢 Info: 绿色
- 🟡 Warn: 黄色/洋红
- 🔴 Error: 红色
- 🔵 Debug: 蓝色

#### IgnoreRecordNotFoundError
- 类型: `bool`
- 默认值: `false`
- 说明: 是否忽略 ErrRecordNotFound 错误的日志输出

## 🔧 Writer 接口

### Writer 定义

```go
type Writer interface {
    Printf(string, ...interface{})
}
```

标准的写入器接口，兼容 `log.Logger` 和其他实现了 `Printf` 方法的对象。

### 常用实现

```go
import "log"
import "os"

// 标准输出
stdWriter := log.New(os.Stdout, "", log.LstdFlags)

// 文件输出
file, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
fileWriter := log.New(file, "", log.LstdFlags)

// 自定义前缀
customWriter := log.New(os.Stdout, "[MyApp] ", log.LstdFlags)
```

## 📈 Trace 方法详解

### 使用模式

#### 基本模式
```go
start := time.Now()
defer logger.Trace(ctx, start, func() (string, int64) {
    return "操作描述", affectedRows
}, err)

// 执行具体操作
// ...
```

#### 详细模式
```go
func SendNotification(ctx context.Context, message string) error {
    start := time.Now()
    var err error
    var count int64

    defer func() {
        logger.Trace(ctx, start, func() (string, int64) {
            return fmt.Sprintf("Send message '%s' to %d notifiers", message, count), count
        }, err)
    }()

    // 执行发送逻辑
    count, err = doSend(message)
    return err
}
```

### 输出格式

#### 正常操作
```
2025-09-14 14:14:50 [info] Operation: Send message to Feishu, Duration: 296.222ms, Affected: 1
```

#### 慢操作警告
```
2025-09-14 14:14:50 SLOW OPERATION >= 200ms [958.276ms] [rows:1] Send message to multi-targets
```

#### 操作失败
```
2025-09-14 14:14:50 [error] Operation failed: Connect to database, Duration: 5000.123ms, Affected: 0, Error: connection timeout
```

## 🎨 日志格式化

### 默认格式
```
时间戳 [级别] 消息内容
```

示例：
```
2025-09-14 13:32:48 [info] NotifyHub initializing with config: queue_type=memory
2025-09-14 13:32:48 [warn] Configuration missing: key=database.password
2025-09-14 13:32:48 [error] Connection failed: error=timeout after 30s
```

### 彩色格式（Colorful=true）
- 时间戳：绿色
- 级别标签：各级别对应颜色
- 消息内容：默认颜色
- 数据字段：蓝色高亮

### 追踪格式
```
时间戳 [info] Operation: 操作描述, Duration: 耗时ms, Affected: 影响行数
时间戳 SLOW OPERATION >= 阈值 [实际耗时ms] [rows:行数] 操作描述
时间戳 [error] Operation failed: 操作描述, Duration: 耗时ms, Affected: 行数, Error: 错误信息
```

## 🔗 相关接口

### 适配器接口
请参考 [adapters/README.md](./adapters/README.md) 了解各种适配器接口的详细说明。

### 常量定义
```go
// 预定义错误
var ErrRecordNotFound = errors.New("record not found")

// 颜色常量（用于彩色输出）
const (
    Reset       = "\033[0m"
    Red         = "\033[31m"
    Green       = "\033[32m"
    Yellow      = "\033[33m"
    Blue        = "\033[34m"
    // ...更多颜色定义
)
```

## 💡 使用建议

### 1. 上下文使用
```go
// 推荐：传递有意义的上下文
ctx := context.WithValue(ctx, "request_id", requestID)
logger.Info(ctx, "处理请求", "user_id", userID)

// 简单场景：使用空上下文
logger.Info(context.Background(), "系统启动")
```

### 2. 结构化数据
```go
// 推荐：键值对形式
logger.Info(ctx, "用户操作", "action", "login", "user_id", 123, "duration", "1.2s")

// 避免：字符串拼接
logger.Info(ctx, fmt.Sprintf("用户%d执行%s操作，耗时%s", 123, "login", "1.2s"))
```

### 3. 错误处理
```go
if err != nil {
    logger.Error(ctx, "操作失败",
        "operation", "send_message",
        "error", err.Error(),
        "retry_count", retryCount,
    )
    return err
}
```

---

📚 **更多文档**: [返回主文档](./README.md) | [适配器指南](./adapters/README.md)