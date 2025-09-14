# 💡 NotifyHub Logger Best Practices

本指南提供了使用 NotifyHub 日志系统的最佳实践，帮助你构建高效、可维护的日志记录方案。

## 🎯 日志级别策略

### 环境分层策略

```go
// 生产环境配置
func newProdLogger() logger.Interface {
    return logger.New(writer, logger.Config{
        LogLevel:      logger.Warn,           // 只记录警告和错误
        SlowThreshold: 1 * time.Second,       // 较高的慢操作阈值
        Colorful:      false,                 // 禁用彩色输出
    })
}

// 开发环境配置
func newDevLogger() logger.Interface {
    return logger.New(writer, logger.Config{
        LogLevel:      logger.Debug,          // 记录所有日志
        SlowThreshold: 100 * time.Millisecond, // 较低的慢操作阈值
        Colorful:      true,                  // 启用彩色输出
    })
}

// 测试环境配置
func newTestLogger() logger.Interface {
    return logger.New(writer, logger.Config{
        LogLevel:      logger.Silent,         // 静默模式
        SlowThreshold: 0,                     // 禁用慢操作检测
        Colorful:      false,
    })
}
```

### 级别选择指南

| 场景 | 推荐级别 | 原因 |
|------|----------|------|
| 生产环境 | `Warn` | 平衡性能和可观测性 |
| 预生产环境 | `Info` | 更多运行时信息 |
| 开发环境 | `Debug` | 完整的调试信息 |
| 单元测试 | `Silent` | 避免测试输出干扰 |
| 集成测试 | `Error` | 只关注错误信息 |

## 📝 日志消息设计

### 消息格式规范

```go
// ✅ 推荐：简洁明确的消息
logger.Info(ctx, "用户登录成功", "user_id", 12345, "duration", "120ms")

// ✅ 推荐：使用动宾结构
logger.Info(ctx, "消息发送完成", "target", "feishu", "count", 5)

// ❌ 避免：冗长的描述
logger.Info(ctx, "用户ID为12345的用户在经过120毫秒的处理后成功完成了登录操作")

// ❌ 避免：无意义的信息
logger.Info(ctx, "这里是日志", "data", someData)
```

### 结构化数据最佳实践

```go
// ✅ 推荐：使用标准的键名
logger.Info(ctx, "HTTP请求处理",
    "method", "POST",
    "path", "/api/users",
    "status_code", 200,
    "duration_ms", 45,
    "user_id", userID,
    "request_id", requestID,
)

// ✅ 推荐：错误信息包含上下文
if err != nil {
    logger.Error(ctx, "数据库操作失败",
        "operation", "insert_user",
        "table", "users",
        "error", err.Error(),
        "user_data", userData,
        "retry_count", retryCount,
    )
}

// ❌ 避免：嵌套复杂对象
logger.Info(ctx, "复杂对象", "object", complexStruct) // 可能导致序列化问题

// ✅ 推荐：提取关键字段
logger.Info(ctx, "用户创建",
    "user_id", user.ID,
    "username", user.Username,
    "email", user.Email,
    "created_at", user.CreatedAt.Format(time.RFC3339),
)
```

## 🚀 性能优化

### 条件日志记录

```go
// ✅ 推荐：昂贵操作的条件记录
if logger.LogLevel >= logger.Debug {
    expensiveData := computeExpensiveDebugData()
    logger.Debug(ctx, "详细调试信息", "data", expensiveData)
}

// ✅ 推荐：使用懒加载函数
logger.Debug(ctx, "调试信息", "computed_data", func() interface{} {
    return computeExpensiveDebugData() // 仅在Debug级别时计算
})
```

### Trace方法优化

```go
// ✅ 推荐：使用defer模式
func ProcessMessage(ctx context.Context, msg string) error {
    start := time.Now()
    var processedCount int64
    var err error

    defer func() {
        logger.Trace(ctx, start, func() (string, int64) {
            return fmt.Sprintf("Process message '%s'", msg), processedCount
        }, err)
    }()

    // 业务逻辑
    processedCount, err = doProcess(msg)
    return err
}

// ✅ 推荐：批量操作的统计
func BatchProcess(ctx context.Context, items []Item) error {
    start := time.Now()
    successCount := int64(0)
    var lastErr error

    defer func() {
        logger.Trace(ctx, start, func() (string, int64) {
            return fmt.Sprintf("Batch process %d items", len(items)), successCount
        }, lastErr)
    }()

    for _, item := range items {
        if err := processItem(item); err != nil {
            lastErr = err
        } else {
            successCount++
        }
    }

    return lastErr
}
```

## 🔧 适配器选择策略

### 按场景选择

```go
// 高性能场景：选择Zap
func setupHighPerformanceLogger() logger.Interface {
    zapLogger, _ := zap.NewProduction()
    return notifyhub.NewZapAdapter(zapLogger.Sugar(), notifyhub.LogLevelInfo)
}

// 企业环境：选择Kart Logger
func setupEnterpriseLogger() logger.Interface {
    kartLogger := kart.New().WithServiceName("notifyhub")
    return notifyhub.NewKartLoggerAdapter(kartLogger, notifyhub.LogLevelWarn)
}

// 云原生环境：选择JSON格式
func setupCloudNativeLogger() logger.Interface {
    jsonLogger := &JSONLogger{serviceName: "notifyhub"}
    return notifyhub.NewCustomAdapter(jsonLogger, notifyhub.LogLevelInfo)
}

// 本地开发：选择彩色输出
func setupDevelopmentLogger() logger.Interface {
    return notifyhub.NewDefaultLogger().LogMode(notifyhub.LogLevelDebug)
}
```

### 多环境适配器

```go
func setupLoggerByEnv(env string) logger.Interface {
    switch env {
    case "production":
        // 生产环境：JSON格式，写入文件
        return setupProductionLogger()
    case "staging":
        // 预生产：结构化日志，包含更多信息
        return setupStagingLogger()
    case "development":
        // 开发环境：彩色控制台输出
        return setupDevelopmentLogger()
    case "test":
        // 测试环境：静默模式
        return notifyhub.NewSilentLogger()
    default:
        return notifyhub.NewDefaultLogger()
    }
}
```

## 🛡️ 错误处理最佳实践

### 分层错误记录

```go
// 服务层：记录业务错误
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) error {
    user, err := s.validateAndCreate(req)
    if err != nil {
        // 记录业务层错误，包含业务上下文
        logger.Error(ctx, "用户创建失败",
            "operation", "create_user",
            "username", req.Username,
            "email", req.Email,
            "error", err.Error(),
            "validation_errors", getValidationErrors(err),
        )
        return fmt.Errorf("创建用户失败: %w", err)
    }

    logger.Info(ctx, "用户创建成功",
        "user_id", user.ID,
        "username", user.Username,
    )
    return nil
}

// 数据层：记录技术错误
func (r *UserRepository) Insert(ctx context.Context, user User) error {
    if err := r.db.Insert(&user); err != nil {
        // 记录数据层错误，包含技术细节
        logger.Error(ctx, "数据库插入失败",
            "operation", "db_insert",
            "table", "users",
            "sql_error", err.Error(),
            "user_data", sanitizeUserData(user), // 注意数据脱敏
        )
        return fmt.Errorf("数据库插入失败: %w", err)
    }
    return nil
}
```

### 错误恢复记录

```go
func (h *Handler) HandleRequest(ctx context.Context, req Request) (resp Response, err error) {
    defer func() {
        if r := recover(); r != nil {
            // 记录panic信息
            logger.Error(ctx, "处理请求时发生panic",
                "request_id", getRequestID(ctx),
                "panic", r,
                "stack", string(debug.Stack()),
            )
            err = fmt.Errorf("内部服务器错误")
        }
    }()

    // 正常处理逻辑
    return h.process(ctx, req)
}
```

## 📊 监控和告警

### 结构化监控数据

```go
// ✅ 推荐：包含监控友好的字段
func logNotificationSent(ctx context.Context, notifier string, success bool, duration time.Duration) {
    logger.Info(ctx, "消息发送完成",
        "notifier", notifier,
        "success", success,
        "duration_ms", duration.Milliseconds(),
        "timestamp", time.Now().Unix(),
        // 监控系统可以基于这些字段创建指标
    )
}

// 慢操作专门记录
func logSlowOperation(ctx context.Context, operation string, duration time.Duration, threshold time.Duration) {
    logger.Warn(ctx, "检测到慢操作",
        "operation", operation,
        "duration_ms", duration.Milliseconds(),
        "threshold_ms", threshold.Milliseconds(),
        "slowness_ratio", float64(duration) / float64(threshold),
        // 用于监控系统创建慢操作告警
    )
}
```

### 业务指标记录

```go
// 记录业务关键指标
func logBusinessMetrics(ctx context.Context, metrics BusinessMetrics) {
    logger.Info(ctx, "业务指标",
        "active_users", metrics.ActiveUsers,
        "messages_sent", metrics.MessagesSent,
        "error_rate", metrics.ErrorRate,
        "avg_response_time", metrics.AvgResponseTime,
        "timestamp", time.Now().Unix(),
        // 用于业务监控大盘
    )
}
```

## 🔒 安全和合规

### 数据脱敏

```go
// 敏感数据脱敏处理
func sanitizeUserData(user User) map[string]interface{} {
    return map[string]interface{}{
        "id":       user.ID,
        "username": user.Username,
        "email":    maskEmail(user.Email),    // admin@example.com -> a***n@e***e.com
        "phone":    maskPhone(user.Phone),    // 13812345678 -> 138****5678
        // 不记录密码、token等敏感信息
    }
}

// 请求日志脱敏
func logHTTPRequest(ctx context.Context, req *http.Request) {
    headers := make(map[string]string)
    for k, v := range req.Header {
        if isSensitiveHeader(k) {
            headers[k] = "***"  // 脱敏敏感头部
        } else {
            headers[k] = strings.Join(v, ",")
        }
    }

    logger.Info(ctx, "HTTP请求",
        "method", req.Method,
        "path", req.URL.Path,
        "headers", headers,
        "user_agent", req.UserAgent(),
    )
}

func isSensitiveHeader(name string) bool {
    sensitiveHeaders := []string{
        "authorization", "cookie", "x-api-key", "x-auth-token",
    }
    name = strings.ToLower(name)
    for _, sensitive := range sensitiveHeaders {
        if name == sensitive {
            return true
        }
    }
    return false
}
```

### 审计日志

```go
// 审计日志专门记录
func logAuditEvent(ctx context.Context, event AuditEvent) {
    logger.Info(ctx, "审计事件",
        "event_type", event.Type,           // "user_login", "data_export", etc.
        "user_id", event.UserID,
        "resource_id", event.ResourceID,
        "action", event.Action,
        "result", event.Result,             // "success", "failure", "denied"
        "ip_address", event.IPAddress,
        "user_agent", event.UserAgent,
        "timestamp", event.Timestamp.Unix(),
        "session_id", event.SessionID,
    )
}
```

## 🧪 测试友好的日志

### 测试中的日志管理

```go
// 测试专用的日志配置
func setupTestLogger(t *testing.T) logger.Interface {
    // 选项1：静默模式
    if !testing.Verbose() {
        return logger.NewSilentLogger()
    }

    // 选项2：测试专用输出
    testWriter := &TestWriter{t: t}
    return logger.New(testWriter, logger.Config{
        LogLevel:  logger.Debug,
        Colorful:  false,
    })
}

type TestWriter struct {
    t *testing.T
}

func (w *TestWriter) Printf(format string, args ...interface{}) {
    w.t.Logf(format, args...)
}

// 在测试中验证日志输出
func TestUserCreation(t *testing.T) {
    logBuf := &bytes.Buffer{}
    testLogger := logger.New(log.New(logBuf, "", 0), logger.Config{
        LogLevel: logger.Info,
    })

    service := NewUserService(testLogger)
    err := service.CreateUser(ctx, validRequest)

    assert.NoError(t, err)
    assert.Contains(t, logBuf.String(), "用户创建成功")
}
```

## 📈 日志分析和优化

### 性能分析

```go
// 记录性能关键点
func logPerformanceMetrics(ctx context.Context, operation string, metrics PerformanceMetrics) {
    logger.Info(ctx, "性能指标",
        "operation", operation,
        "duration_ms", metrics.Duration.Milliseconds(),
        "memory_mb", metrics.MemoryMB,
        "cpu_percent", metrics.CPUPercent,
        "goroutines", metrics.Goroutines,
        "gc_count", metrics.GCCount,
        // 用于性能分析和优化
    )
}
```

### 日志轮转和清理

```go
// 生产环境日志配置示例
func setupProductionFileLogger() logger.Interface {
    // 使用日志轮转
    logFile := &lumberjack.Logger{
        Filename:   "/var/log/notifyhub/app.log",
        MaxSize:    100,  // MB
        MaxBackups: 7,    // 保留7个备份
        MaxAge:     30,   // 天
        Compress:   true, // 压缩旧日志
    }

    return logger.New(
        log.New(logFile, "", log.LstdFlags),
        logger.Config{
            LogLevel:      logger.Info,
            SlowThreshold: 500 * time.Millisecond,
            Colorful:      false,
        },
    )
}
```

## 📚 团队协作规范

### 日志消息规范

1. **使用英文动词开头**: "User login failed" 而不是 "Failed user login"
2. **保持消息简洁**: 不超过50个字符
3. **使用统一的业务术语**: 建立团队词汇表
4. **包含关键上下文**: user_id, request_id, operation等

### 代码审查检查点

- [ ] 是否选择了合适的日志级别？
- [ ] 敏感数据是否已脱敏？
- [ ] 错误日志是否包含足够的上下文？
- [ ] 是否避免了日志中的用户输入注入？
- [ ] 性能敏感路径是否考虑了日志开销？

---

🔗 **相关文档**: [主文档](./README.md) | [接口说明](./INTERFACE.md) | [适配器指南](./adapters/README.md)