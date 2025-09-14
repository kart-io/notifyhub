# 📝 NotifyHub Logger System

NotifyHub 的日志系统采用模块化设计，提供了灵活的日志记录和多样化的适配器支持。

## 🏗️ 架构设计

```
logger/
├── README.md           - 本文档
├── interface.go        - 核心日志接口定义
├── logger.go          - 默认日志器实现
├── adapters/          - 适配器子包
│   ├── README.md      - 适配器使用指南
│   └── adapters.go    - 所有适配器实现
└── examples/          - 使用示例（位于项目根目录下）
```

## ✨ 核心特性

- **统一接口**: 提供标准的日志记录接口，兼容多种日志库
- **多级别支持**: Silent, Error, Warn, Info, Debug 五个级别
- **性能追踪**: 自动记录操作耗时和慢操作告警
- **彩色输出**: 支持控制台彩色日志输出
- **灵活适配**: 支持标准库、第三方库和自定义日志实现

## 🎯 日志级别

```go
const (
    Silent LogLevel = iota + 1  // 静默模式
    Error                       // 仅错误
    Warn                        // 警告及以上
    Info                        // 信息及以上
    Debug                       // 调试及以上（最详细）
)
```

## 🚀 快速开始

### 1. 使用默认日志器

```go
import "github.com/kart-io/notifyhub/logger"

// 创建默认日志器（Warn级别，彩色输出）
log := logger.Default()

// 使用日志器
ctx := context.Background()
log.Info(ctx, "应用启动", "port", 8080)
log.Warn(ctx, "配置缺失", "key", "database.host")
log.Error(ctx, "连接失败", "error", err)
```

### 2. 自定义配置

```go
import (
    "log"
    "os"
    "github.com/kart-io/notifyhub/logger"
)

// 自定义writer和配置
customLogger := logger.New(
    log.New(os.Stdout, "[MyApp] ", log.LstdFlags),
    logger.Config{
        SlowThreshold: 500 * time.Millisecond,  // 慢操作阈值
        LogLevel:      logger.Debug,            // 日志级别
        Colorful:      true,                    // 彩色输出
    },
)
```

### 3. 使用适配器

```go
import (
    "github.com/kart-io/notifyhub"
    "github.com/sirupsen/logrus"
)

// 使用Logrus适配器
logrusLogger := logrus.New()
logrusLogger.SetLevel(logrus.InfoLevel)

hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithLogger(
        notifyhub.NewLogrusAdapter(logrusLogger, notifyhub.LogLevelInfo),
    ),
)
```

## 📊 核心接口

### Logger Interface

```go
type Interface interface {
    LogMode(LogLevel) Interface
    Info(context.Context, string, ...interface{})
    Warn(context.Context, string, ...interface{})
    Error(context.Context, string, ...interface{})
    Debug(context.Context, string, ...interface{})
    Trace(context.Context, time.Time, func() (string, int64), error)
}
```

### Config 结构

```go
type Config struct {
    SlowThreshold             time.Duration  // 慢操作阈值
    LogLevel                  LogLevel       // 日志级别
    Colorful                  bool          // 彩色输出
    IgnoreRecordNotFoundError bool          // 忽略记录未找到错误
}
```

## 🔌 支持的适配器

| 适配器 | 说明 | 特性 |
|--------|------|------|
| **Default** | 内置默认适配器 | 彩色输出、格式化 |
| **StdLog** | 标准库log适配器 | 轻量、简单 |
| **Logrus** | Logrus库适配器 | 结构化日志 |
| **Zap** | Uber Zap适配器 | 高性能 |
| **KartLogger** | Kart企业日志库 | WithField支持 |
| **Custom** | 自定义适配器框架 | 完全自定义 |
| **Function** | 函数式适配器 | 轻量回调 |

详细的适配器使用说明请参考 [`adapters/README.md`](./adapters/README.md)

## 📈 性能监控

日志系统自动提供性能追踪：

```go
// Trace方法会记录操作耗时
log.Trace(ctx, startTime, func() (string, int64) {
    return "发送消息到Feishu", 1
}, err)
```

输出示例：
```
2025-09-14 14:14:50 [info] Operation: Send message to Feishu, Duration: 296.222ms, Affected: 1
2025-09-14 14:14:50 SLOW OPERATION >= 200ms [958.276ms] [rows:1] Send message to multi-targets
```

## 🎨 日志输出格式

### 默认格式输出
```
2025-09-14 13:32:48 [info] NotifyHub initializing with config: queue_type=memory, buffer_size=100
2025-09-14 13:32:48 [warn] Configuration missing: key=database.password
2025-09-14 13:32:48 [error] Connection failed: error=timeout after 30s
```

### 彩色输出
- 🟢 **Info**: 绿色
- 🟡 **Warn**: 黄色/洋红
- 🔴 **Error**: 红色
- 🔵 **Debug**: 蓝色

## 🛠️ 配置选项

### 环境变量支持

```bash
export NOTIFYHUB_LOG_LEVEL=debug
export NOTIFYHUB_LOG_COLORFUL=true
```

### 程序配置

```go
// 详细调试模式
debugLogger := logger.New(writer, logger.Config{
    LogLevel:      logger.Debug,
    Colorful:      true,
    SlowThreshold: 100 * time.Millisecond,
})

// 生产环境模式
prodLogger := logger.New(writer, logger.Config{
    LogLevel:      logger.Warn,
    Colorful:      false,
    SlowThreshold: 1 * time.Second,
})
```

## 💡 最佳实践

### 1. 日志级别选择
- **生产环境**: 使用 `Warn` 或 `Error` 级别
- **开发环境**: 使用 `Debug` 级别获取详细信息
- **测试环境**: 使用 `Silent` 避免日志干扰

### 2. 结构化日志
```go
// 推荐：使用键值对
log.Info(ctx, "用户登录", "user_id", 123, "ip", "192.168.1.1")

// 避免：纯文本拼接
log.Info(ctx, fmt.Sprintf("用户%d从%s登录", 123, "192.168.1.1"))
```

### 3. 错误处理
```go
if err != nil {
    log.Error(ctx, "操作失败", "operation", "send_message", "error", err.Error())
    return err
}
```

### 4. 性能监控
```go
start := time.Now()
defer func() {
    log.Trace(ctx, start, func() (string, int64) {
        return "批量处理用户数据", int64(userCount)
    }, err)
}()
```

## 📚 更多文档

- [适配器使用指南](./adapters/README.md) - 详细的适配器使用说明
- [完整示例代码](../examples/custom-logger/) - 实际使用示例
- [API参考文档](./INTERFACE.md) - 接口详细说明

## 🔗 相关链接

- [NotifyHub 主文档](../README.md)
- [日志系统完整介绍](../LOGGING.md)
- [配置指南](../docs/configuration.md)
- [最佳实践](../docs/best-practices.md)

---

💡 **提示**: 如需更高级的日志需求，请查看自定义适配器框架，它支持任意格式和输出目标。