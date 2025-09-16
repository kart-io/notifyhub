# NotifyHub HTTP Service - 优化示例

这是一个展示 NotifyHub 最佳实践的 HTTP 服务示例，重点体现了：

- ✨ **减少样板代码** - 简化配置和调用方式
- 🎯 **最佳实践** - 生产级代码结构和错误处理
- 📦 **完整可运行** - 开箱即用的示例服务
- 🔧 **直观易懂** - 清晰的代码结构和注释

## 🚀 快速开始

### 环境配置

```bash
# Feishu 配置
export NOTIFYHUB_FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook"
export NOTIFYHUB_FEISHU_SECRET="your-secret"

# Email 配置
export NOTIFYHUB_SMTP_HOST="smtp.gmail.com"
export NOTIFYHUB_SMTP_PORT="587"
export NOTIFYHUB_SMTP_USERNAME="your-email@gmail.com"
export NOTIFYHUB_SMTP_PASSWORD="your-app-password"
export NOTIFYHUB_SMTP_FROM="your-email@gmail.com"

# 队列配置（可选）
export NOTIFYHUB_QUEUE_TYPE="memory"
export NOTIFYHUB_QUEUE_SIZE="1000"
export NOTIFYHUB_QUEUE_WORKERS="4"

# 服务端口（可选）
export PORT="8080"
```

### 运行服务

```bash
cd examples/http-service
go mod tidy
go run main.go
```

服务启动后，访问：
- 健康检查: http://localhost:8080/api/v1/health
- 指标监控: http://localhost:8080/api/v1/metrics

## 📖 API 使用示例

### 1. 发送简单通知

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "type": "notice",
    "title": "系统通知",
    "message": "这是一条测试消息",
    "targets": [
      {
        "type": "email",
        "value": "user@example.com"
      }
    ]
  }'
```

### 2. 发送告警（展示最佳实践）

```bash
curl -X POST http://localhost:8080/api/v1/alert \
  -H "Content-Type: application/json" \
  -d '{
    "title": "🚨 系统告警",
    "message": "数据库连接异常，请及时处理！",
    "priority": 5,
    "targets": [
      {
        "type": "group",
        "value": "ops-team",
        "platform": "feishu"
      },
      {
        "type": "email",
        "value": "admin@company.com"
      }
    ],
    "variables": {
      "server": "prod-db-01",
      "error": "connection timeout",
      "timestamp": "2024-01-01T10:00:00Z"
    },
    "metadata": {
      "severity": "high",
      "environment": "production"
    },
    "retry_count": 3,
    "timeout_seconds": 30
  }'
```

### 3. 使用模板发送报告

```bash
curl -X POST http://localhost:8080/api/v1/report \
  -H "Content-Type: application/json" \
  -d '{
    "title": "📊 日报",
    "message": "系统运行正常",
    "template": "daily_report",
    "targets": [
      {
        "type": "email",
        "value": "reports@company.com"
      }
    ],
    "variables": {
      "date": "2024-01-01",
      "uptime": "99.9%",
      "requests": 1500000,
      "errors": 5
    }
  }'
```

### 4. 异步发送（推荐用于批量通知）

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "type": "notice",
    "title": "批量通知",
    "message": "这是一条异步消息",
    "async": true,
    "targets": [
      {
        "type": "email",
        "value": "user1@example.com"
      },
      {
        "type": "email", 
        "value": "user2@example.com"
      }
    ]
  }'
```

## 🎯 关键优化特性

### 1. 简化的配置方式

**优化前（样板代码较多）：**
```go
// 需要手动配置每个组件
hub, err := client.New(
    config.WithFeishu("https://...", "secret"),
    config.WithEmail("smtp.gmail.com", 587, "user", "pass", "from", true, 30*time.Second),
    config.WithQueue("memory", 1000, 4),
    // ... 更多配置
)
```

**优化后（自动从环境变量加载）：**
```go
// 自动化配置，减少样板代码
hub, err := client.New(
    config.WithFeishuFromEnv(),     // 自动加载
    config.WithEmailFromEnv(),      // 自动加载
    config.WithQueueFromEnv(),      // 自动加载
    config.WithMockNotifier("dev"), // 开发模式
)
```

### 2. 流式构建器模式

**优化前：**
```go
message := &notifiers.Message{}
message.Title = req.Title
message.Body = req.Message
if req.Priority > 0 {
    message.Priority = req.Priority
}
// ... 大量条件判断和赋值
```

**优化后：**
```go
// 链式调用，清晰直观
builder := client.NewAlert(req.Title, req.Message).
    Priority(req.Priority).
    Variable("server", "prod-01").
    FeishuGroup("ops-team").
    Email("admin@company.com")

message := builder.Build()
```

### 3. 统一的响应格式

```go
type NotificationResponse struct {
    Success   bool                   `json:"success"`
    Message   string                 `json:"message,omitempty"`
    TaskID    string                 `json:"task_id,omitempty"`    // 异步任务
    Results   []PlatformResult       `json:"results,omitempty"`    // 同步结果
    Metadata  map[string]interface{} `json:"metadata,omitempty"`   // 元数据
}
```

### 4. 智能路由选择

```go
// 根据请求自动选择同步/异步模式
if req.Async {
    ns.handleAsyncSend(c, message, req)
} else {
    ns.handleSyncSend(c, message, req)
}
```

## 🔧 生产级特性

### 健康检查和监控

```bash
# 健康检查
curl http://localhost:8080/api/v1/health

# 响应示例
{
  "status": "ok",
  "uptime": "2h30m15s",
  "metrics": {
    "total_sent": 1250,
    "success_rate": 0.995,
    "avg_duration": "150ms"
  },
  "platforms": {
    "feishu": true,
    "email": true
  },
  "version": "1.2.0"
}
```

### 错误处理和重试

```go
// 自动重试配置
retryOpts := client.NewRetryOptions(3)
if req.RetryCount > 0 {
    retryOpts = client.NewRetryOptions(req.RetryCount)
}
if req.Timeout > 0 {
    retryOpts = retryOpts.WithTimeout(time.Duration(req.Timeout) * time.Second)
}
```

### 优雅停机

```go
// 监听系统信号
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// 30秒优雅停机
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
server.Shutdown(ctx)
```

## 🧪 测试示例

### 使用 curl 测试

```bash
# 测试健康检查
curl http://localhost:8080/api/v1/health

# 测试简单通知
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "type": "notice",
    "title": "测试",
    "message": "Hello NotifyHub!",
    "targets": [{"type": "email", "value": "test@example.com"}]
  }'

# 测试快捷告警接口
curl -X POST http://localhost:8080/api/v1/alert \
  -H "Content-Type: application/json" \
  -d '{
    "title": "告警测试",
    "message": "这是一个测试告警",
    "targets": [{"type": "email", "value": "alert@example.com"}]
  }'
```

### 使用 JavaScript (Fetch)

```javascript
// 发送通知
async function sendNotification() {
  const response = await fetch('http://localhost:8080/api/v1/notifications', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      type: 'notice',
      title: '前端通知',
      message: '来自前端的测试消息',
      targets: [
        { type: 'email', value: 'frontend@example.com' }
      ]
    })
  });
  
  const result = await response.json();
  console.log('通知结果:', result);
}

// 检查服务健康状态
async function checkHealth() {
  const response = await fetch('http://localhost:8080/api/v1/health');
  const health = await response.json();
  console.log('服务状态:', health);
}
```

## 📊 与其他示例的对比

| 特性 | 基础示例 | 高级示例 | HTTP服务示例 |
|------|----------|----------|--------------|
| 配置方式 | 手动配置 | 详细配置 | **环境变量自动化** |
| 样板代码 | 较多 | 很多 | **最少** |
| 生产就绪 | ❌ | ⚠️ | **✅** |
| 错误处理 | 基础 | 完善 | **企业级** |
| 监控集成 | ❌ | 基础 | **完整** |
| 可维护性 | 低 | 中 | **高** |

## 🎉 总结

这个 HTTP 服务示例展示了 NotifyHub 的最佳使用方式：

1. **最少样板代码** - 通过环境变量自动配置
2. **直观的 API** - RESTful 设计，易于理解和使用
3. **生产级质量** - 完整的错误处理、监控、优雅停机
4. **开发友好** - 清晰的代码结构和丰富的注释
5. **即开即用** - 完整的示例和文档

通过这个示例，开发者可以快速理解如何在实际项目中优雅地使用 NotifyHub，并可以直接基于此示例构建自己的通知服务。