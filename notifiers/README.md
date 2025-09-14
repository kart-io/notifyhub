# Notifiers Package

## 功能概述

Notifiers包实现了NotifyHub的平台适配器层，定义了统一的通知接口并提供多平台的具体实现。每个notifier负责将抽象的Message转换为特定平台的API调用。

## 核心接口

### Notifier接口
```go
type Notifier interface {
    Name() string
    Send(ctx context.Context, message *Message) ([]*SendResult, error)
    SupportsTarget(target Target) bool
    Health(ctx context.Context) error
    Shutdown(ctx context.Context) error  // v1.1.0新增
}
```

## 支持的平台

### 1. 飞书 (Feishu)
- **文件**: `feishu.go`
- **支持格式**: Text、Markdown、Card
- **目标类型**: 群组、用户
- **特殊功能**: @用户提醒、富文本卡片

### 2. 邮件 (Email)
- **文件**: `email.go`
- **支持格式**: 纯文本（自动转换HTML/Markdown）
- **目标类型**: 邮箱地址
- **特殊功能**: SMTP认证、TLS加密、HTML转文本

## 消息结构

### Message结构体
```go
type Message struct {
    ID          string
    Title       string
    Body        string
    Format      MessageFormat  // Text, Markdown, HTML
    Priority    int           // 1-5优先级
    Targets     []Target      // 目标列表
    Variables   map[string]interface{}  // 模板变量
    Metadata    map[string]string       // 元数据
    Delay       time.Duration          // 延迟发送
    CreatedAt   time.Time
}
```

### Target结构体
```go
type Target struct {
    Type     TargetType  // Email, User, Group
    Value    string      // 具体值（邮箱、用户ID等）
    Platform string      // 指定平台
}
```

## 使用示例

### 创建Notifier实例

```go
// 飞书Notifier
feishuNotifier := notifiers.NewFeishuNotifier(
    "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
    "your-secret",
    30*time.Second,
)

// 邮件Notifier
emailNotifier := notifiers.NewEmailNotifier(
    "smtp.gmail.com", 587,
    "username", "password", "from@example.com",
    true, // 使用TLS
    30*time.Second,
)

// 带速率限制的Notifier
rateLimitConfig := &internal.RateLimitConfig{
    Limit:    100,      // 每分钟100条
    Duration: time.Minute,
}
emailNotifier := notifiers.NewEmailNotifierWithRateLimit(
    host, port, username, password, from, useTLS, timeout,
    rateLimitConfig,
)
```

### 发送消息

```go
message := &notifiers.Message{
    Title: "系统告警",
    Body:  "CPU使用率超过90%",
    Format: notifiers.FormatText,
    Priority: 4,
    Targets: []notifiers.Target{
        {Type: notifiers.TargetTypeEmail, Value: "admin@example.com"},
        {Type: notifiers.TargetTypeUser, Value: "user123", Platform: "feishu"},
    },
}

// 发送消息
results, err := notifier.Send(ctx, message)
for _, result := range results {
    if result.Success {
        log.Printf("发送成功: %s", result.Platform)
    } else {
        log.Printf("发送失败: %s - %s", result.Platform, result.Error)
    }
}
```

## 平台特定功能

### 飞书特性

```go
// @用户提醒
message.Body = "Hello <at user_id=\"ou_xxx\">@张三</at>"

// 富文本卡片
message.Format = notifiers.FormatCard
message.Body = `{
    "config": {"wide_screen_mode": true},
    "elements": [
        {"tag": "div", "text": {"content": "告警信息", "tag": "lark_md"}}
    ]
}`

// 群组消息
target := notifiers.Target{
    Type: notifiers.TargetTypeGroup,
    Value: "oc_xxx", // 群组ID
    Platform: "feishu",
}
```

### 邮件特性

```go
// 自动格式转换
message.Format = notifiers.FormatHTML
message.Body = "<h1>标题</h1><p>内容</p>"
// 会自动转换为纯文本发送

// 添加元数据到邮件
message.Metadata = map[string]string{
    "department": "运维部",
    "severity": "high",
}
```

## 错误处理

### SendResult结构
```go
type SendResult struct {
    Target    Target        // 目标信息
    Platform  string        // 平台名称
    Success   bool          // 是否成功
    Error     string        // 错误信息
    SentAt    time.Time     // 发送时间
    Duration  time.Duration // 耗时
    Attempts  int          // 尝试次数
}
```

### 健康检查

```go
// 检查notifier健康状态
err := notifier.Health(ctx)
if err != nil {
    log.Printf("Notifier不健康: %v", err)
}
```

## 扩展新平台

实现新的通知平台：

```go
type SlackNotifier struct {
    webhookURL string
    timeout    time.Duration
}

func (s *SlackNotifier) Name() string {
    return "slack"
}

func (s *SlackNotifier) Send(ctx context.Context, message *Message) ([]*SendResult, error) {
    // 实现Slack发送逻辑
    return results, nil
}

func (s *SlackNotifier) SupportsTarget(target Target) bool {
    return target.Platform == "slack" || target.Type == TargetTypeSlackChannel
}

func (s *SlackNotifier) Health(ctx context.Context) error {
    // 健康检查逻辑
    return nil
}

func (s *SlackNotifier) Shutdown(ctx context.Context) error {
    // 清理资源
    return nil
}
```

## 速率限制

支持令牌桶算法的速率限制：

```go
// 配置速率限制
rateLimitConfig := &internal.RateLimitConfig{
    Limit:    100,           // 令牌数量
    Duration: time.Minute,   // 时间窗口
}

// 应用到notifier
notifier := NewEmailNotifierWithRateLimit(
    host, port, username, password, from, useTLS, timeout,
    rateLimitConfig,
)
```

## 最佳实践

1. **目标匹配**: 使用`SupportsTarget()`精确匹配支持的目标类型
2. **错误处理**: 返回详细的SendResult，支持部分成功场景
3. **资源管理**: 实现Shutdown()方法进行优雅清理
4. **健康检查**: 定期检查外部服务连接状态
5. **速率限制**: 防止API限流，保护外部服务

## 文件说明

- `base.go` - 核心接口和数据结构定义
- `feishu.go` - 飞书平台适配器实现
- `email.go` - 邮件平台适配器实现
- `notifiers_test.go` - 单元测试和集成测试