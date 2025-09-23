# NotifyHub API 参考文档

本文档提供了 NotifyHub 公共 API 的详细参考信息。

## 目录

- [核心接口](#核心接口)
- [配置选项](#配置选项)
- [消息构建](#消息构建)
- [目标类型](#目标类型)
- [回执和状态](#回执和状态)
- [健康检查](#健康检查)
- [平台扩展](#平台扩展)
- [错误处理](#错误处理)

## 核心接口

### Hub

`Hub` 是 NotifyHub 的核心接口，提供消息发送和管理功能。

```go
type Hub interface {
    Send(ctx context.Context, message *Message) (*Receipt, error)
    SendAsync(ctx context.Context, message *Message) (*AsyncReceipt, error)
    Health(ctx context.Context) (*HealthStatus, error)
    Close(ctx context.Context) error
}
```

#### 方法详情

##### Send

同步发送消息到指定目标。

```go
func Send(ctx context.Context, message *Message) (*Receipt, error)
```

**参数:**

- `ctx`: 上下文，用于超时控制和取消操作
- `message`: 要发送的消息对象

**返回值:**

- `*Receipt`: 发送回执，包含详细的发送结果
- `error`: 发送过程中的错误

**示例:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

msg := NewMessage("测试通知").
    Body("这是一条测试消息").
    AddTarget(NewFeishuUserTarget("user123")).
    Build()

receipt, err := hub.Send(ctx, msg)
if err != nil {
    log.Printf("发送失败: %v", err)
    return
}

log.Printf("发送结果: %s, 成功: %d, 失败: %d",
    receipt.Status, receipt.Successful, receipt.Failed)
```

##### SendAsync

异步发送消息，立即返回而不等待发送完成。

```go
func SendAsync(ctx context.Context, message *Message) (*AsyncReceipt, error)
```

**参数:**

- `ctx`: 上下文
- `message`: 要发送的消息对象

**返回值:**

- `*AsyncReceipt`: 异步回执，包含消息 ID 和状态
- `error`: 提交过程中的错误

**示例:**

```go
receipt, err := hub.SendAsync(ctx, msg)
if err != nil {
    log.Printf("提交失败: %v", err)
    return
}

log.Printf("消息已提交: %s, 状态: %s", receipt.MessageID, receipt.Status)
```

##### Health

检查 Hub 和所有注册平台的健康状态。

```go
func Health(ctx context.Context) (*HealthStatus, error)
```

**返回值:**

- `*HealthStatus`: 健康状态信息
- `error`: 健康检查过程中的错误

**示例:**

```go
health, err := hub.Health(ctx)
if err != nil {
    log.Printf("健康检查失败: %v", err)
    return
}

if !health.Healthy {
    log.Printf("Hub 状态异常: %s", health.Status)
    for platform, status := range health.Platforms {
        if !status.Available {
            log.Printf("平台 %s 不可用: %s", platform, status.Status)
        }
    }
}
```

##### Close

优雅关闭 Hub，释放所有资源。

```go
func Close(ctx context.Context) error
```

**参数:**

- `ctx`: 上下文，用于控制关闭超时

**返回值:**

- `error`: 关闭过程中的错误

**示例:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := hub.Close(ctx); err != nil {
    log.Printf("关闭 Hub 失败: %v", err)
}
```

## 配置选项

### HubOption

`HubOption` 是配置 Hub 的函数类型，使用函数式选项模式。

```go
type HubOption func(*HubConfig)
```

### 内置配置选项

#### WithFeishu

配置飞书平台。

```go
func WithFeishu(webhookURL, secret string) HubOption
```

**参数:**

- `webhookURL`: 飞书机器人 Webhook URL
- `secret`: 签名密钥（可选）

**示例:**

```go
hub, err := NewHub(
    WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook", "your-secret"),
)
```

#### WithEmail

配置邮件平台。

```go
func WithEmail(host string, port int, username, password, from string, useTLS bool, timeout time.Duration) HubOption
```

**参数:**

- `host`: SMTP 服务器地址
- `port`: SMTP 端口
- `username`: 用户名
- `password`: 密码
- `from`: 发件人地址
- `useTLS`: 是否使用 TLS
- `timeout`: 连接超时

**示例:**

```go
hub, err := NewHub(
    WithEmail("smtp.gmail.com", 587, "user@gmail.com", "password", "notifications@company.com", true, 30*time.Second),
)
```

#### WithTimeout

设置默认超时时间。

```go
func WithTimeout(timeout time.Duration) HubOption
```

**示例:**

```go
hub, err := NewHub(
    WithTimeout(45*time.Second),
)
```

#### WithPlatformConfig

使用通用配置添加平台。

```go
func WithPlatformConfig(platformName string, config map[string]interface{}) HubOption
```

**示例:**

```go
hub, err := NewHub(
    WithPlatformConfig("custom", map[string]interface{}{
        "api_key": "your-api-key",
        "endpoint": "https://api.custom.com",
    }),
)
```

## 消息构建

### Message

消息结构体包含所有发送所需的信息。

```go
type Message struct {
    ID           string                 `json:"id"`
    Title        string                 `json:"title"`
    Body         string                 `json:"body"`
    Format       string                 `json:"format"`       // "text", "markdown", "html"
    Priority     Priority               `json:"priority"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    Variables    map[string]interface{} `json:"variables,omitempty"`
    PlatformData map[string]interface{} `json:"platform_data,omitempty"`
    Targets      []Target               `json:"targets"`
}
```

### MessageBuilder

消息构建器提供流式 API 来构建消息。

```go
type MessageBuilder struct {
    // 内部字段
}
```

#### 构建器方法

##### Body

设置消息内容。

```go
func (b *MessageBuilder) Body(body string) *MessageBuilder
```

##### Priority

设置消息优先级。

```go
func (b *MessageBuilder) Priority(priority Priority) *MessageBuilder
```

**优先级常量:**

- `PriorityLow`: 低优先级
- `PriorityNormal`: 普通优先级（默认）
- `PriorityHigh`: 高优先级
- `PriorityUrgent`: 紧急优先级

##### AddTarget

添加发送目标。

```go
func (b *MessageBuilder) AddTarget(target Target) *MessageBuilder
```

##### AddTargets

批量添加发送目标。

```go
func (b *MessageBuilder) AddTargets(targets ...Target) *MessageBuilder
```

##### WithMetadata

添加元数据。

```go
func (b *MessageBuilder) WithMetadata(key string, value interface{}) *MessageBuilder
```

##### WithVariable

添加模板变量。

```go
func (b *MessageBuilder) WithVariable(key string, value interface{}) *MessageBuilder
```

##### Build

构建最终的消息对象。

```go
func (b *MessageBuilder) Build() *Message
```

### 消息创建函数

#### NewMessage

创建普通消息构建器。

```go
func NewMessage(title ...string) *MessageBuilder
```

**示例:**

```go
msg := NewMessage("系统通知").
    Body("服务器重启完成").
    Priority(PriorityNormal).
    AddTarget(NewEmailTarget("admin@company.com")).
    Build()
```

#### NewAlert

创建警告消息构建器（高优先级）。

```go
func NewAlert(title string) *MessageBuilder
```

**示例:**

```go
msg := NewAlert("磁盘空间不足").
    Body("服务器磁盘使用率已达到 90%").
    AddTarget(NewFeishuUserTarget("admin")).
    Build()
```

#### NewUrgent

创建紧急消息构建器（最高优先级）。

```go
func NewUrgent(title string) *MessageBuilder
```

**示例:**

```go
msg := NewUrgent("服务宕机").
    Body("支付服务当前不可用").
    AddTarget(NewEmailTarget("oncall@company.com")).
    AddTarget(NewFeishuUserTarget("oncall")).
    Build()
```

## 目标类型

### Target

目标结构体定义了消息的接收者。

```go
type Target struct {
    Type     string `json:"type"`     // "email", "user", "group", "webhook", "phone"
    Value    string `json:"value"`    // 目标标识符
    Platform string `json:"platform"` // 平台名称
}
```

### 目标类型常量

```go
const (
    TargetTypeEmail   = "email"
    TargetTypePhone   = "phone"
    TargetTypeUser    = "user"
    TargetTypeGroup   = "group"
    TargetTypeChannel = "channel"
    TargetTypeWebhook = "webhook"
)
```

### 平台常量

```go
const (
    PlatformFeishu  = "feishu"
    PlatformEmail   = "email"
    PlatformSMS     = "sms"
    PlatformWebhook = "webhook"
    PlatformAuto    = "auto"
)
```

### 目标创建函数

#### NewTarget

创建通用目标。

```go
func NewTarget(targetType, value, platform string) Target
```

#### NewEmailTarget

创建邮件目标。

```go
func NewEmailTarget(email string) Target
```

**示例:**

```go
target := NewEmailTarget("user@company.com")
```

#### NewPhoneTarget

创建电话目标。

```go
func NewPhoneTarget(phone string) Target
```

**示例:**

```go
target := NewPhoneTarget("+86-13800138000")
```

#### NewFeishuUserTarget

创建飞书用户目标。

```go
func NewFeishuUserTarget(userID string) Target
```

**示例:**

```go
target := NewFeishuUserTarget("ou_7d8a6e6df7e6a6e6df7e6a6e6df7e6a")
```

#### NewFeishuGroupTarget

创建飞书群组目标。

```go
func NewFeishuGroupTarget(groupID string) Target
```

**示例:**

```go
target := NewFeishuGroupTarget("oc_a0553eda9014c201e6969b478895c230")
```

#### NewWebhookTarget

创建 Webhook 目标。

```go
func NewWebhookTarget(url string) Target
```

**示例:**

```go
target := NewWebhookTarget("https://example.com/webhook/your-id")
```

#### AutoDetectTarget

自动检测目标类型。

```go
func AutoDetectTarget(value string) Target
```

**示例:**

```go
target := AutoDetectTarget("user@company.com")  // 自动识别为邮件
target := AutoDetectTarget("+86-13800138000")   // 自动识别为电话
```

## 回执和状态

### Receipt

同步发送的回执信息。

```go
type Receipt struct {
    MessageID  string           `json:"message_id"`
    Status     string           `json:"status"`     // "success", "failed", "partial"
    Results    []PlatformResult `json:"results"`
    Successful int              `json:"successful"`
    Failed     int              `json:"failed"`
    Total      int              `json:"total"`
    Timestamp  time.Time        `json:"timestamp"`
}
```

### AsyncReceipt

异步发送的回执信息。

```go
type AsyncReceipt struct {
    MessageID string    `json:"message_id"`
    Status    string    `json:"status"`    // "queued", "processing", "completed", "failed"
    QueuedAt  time.Time `json:"queued_at"`
}
```

### PlatformResult

单个平台的发送结果。

```go
type PlatformResult struct {
    Platform  string        `json:"platform"`
    Target    string        `json:"target"`
    Success   bool          `json:"success"`
    MessageID string        `json:"message_id,omitempty"`
    Error     string        `json:"error,omitempty"`
    Timestamp time.Time     `json:"timestamp"`
    Duration  time.Duration `json:"duration"`
}
```

## 健康检查

### HealthStatus

整体健康状态。

```go
type HealthStatus struct {
    Healthy   bool                      `json:"healthy"`
    Status    string                    `json:"status"`
    Platforms map[string]PlatformHealth `json:"platforms"`
    Queue     QueueHealth               `json:"queue"`
    Timestamp time.Time                 `json:"timestamp"`
}
```

### PlatformHealth

单个平台的健康状态。

```go
type PlatformHealth struct {
    Available bool              `json:"available"`
    Status    string            `json:"status"`
    Details   map[string]string `json:"details,omitempty"`
}
```

### QueueHealth

队列系统的健康状态。

```go
type QueueHealth struct {
    Available bool `json:"available"`
}
```

## 平台扩展

### ExternalSender

外部平台扩展接口。

```go
type ExternalSender interface {
    Name() string
    Send(ctx context.Context, msg *Message, targets []Target) ([]*SendResult, error)
    ValidateTarget(target Target) error
    GetCapabilities() Capabilities
    IsHealthy(ctx context.Context) error
    Close() error
}
```

### 注册平台

```go
func RegisterPlatform(platformName string, creator ExternalSenderCreator)
```

**示例:**

```go
func init() {
    platform.RegisterPlatform("custom", func(config map[string]interface{}) (platform.ExternalSender, error) {
        return &CustomSender{
            apiKey: config["api_key"].(string),
        }, nil
    })
}
```

## 错误处理

### 错误类型

NotifyHub 定义了几种错误类型：

- **配置错误**: 平台配置不正确
- **验证错误**: 目标验证失败
- **网络错误**: 网络连接问题
- **平台错误**: 平台特定的错误
- **超时错误**: 操作超时

### 错误检查

```go
// 检查是否为验证错误
if IsValidationError(err) {
    // 处理验证错误
}

// 检查发送结果
receipt, err := hub.Send(ctx, msg)
if err != nil {
    return err
}

if receipt.Status == "failed" {
    return fmt.Errorf("所有平台发送失败")
}

if receipt.Status == "partial" {
    // 部分成功，检查具体失败原因
    for _, result := range receipt.Results {
        if !result.Success {
            log.Printf("平台 %s 发送失败: %s", result.Platform, result.Error)
        }
    }
}
```

## 最佳实践

### 1. 资源管理

始终确保正确关闭资源：

```go
hub, err := NewHub(options...)
if err != nil {
    return err
}
defer func() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := hub.Close(ctx); err != nil {
        log.Printf("关闭 Hub 失败: %v", err)
    }
}()
```

### 2. 上下文管理

使用带超时的上下文：

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

receipt, err := hub.Send(ctx, msg)
```

### 3. 错误处理

检查所有可能的错误情况：

```go
receipt, err := hub.Send(ctx, msg)
if err != nil {
    return fmt.Errorf("发送消息失败: %w", err)
}

switch receipt.Status {
case "success":
    log.Printf("消息发送成功")
case "partial":
    log.Printf("部分发送成功: %d/%d", receipt.Successful, receipt.Total)
case "failed":
    return fmt.Errorf("消息发送失败")
}
```

### 4. 批量发送

使用批量目标提高效率：

```go
targets := []Target{
    NewEmailTarget("user1@company.com"),
    NewEmailTarget("user2@company.com"),
    NewFeishuUserTarget("user3"),
}

msg := NewMessage("批量通知").
    Body("这是一条批量消息").
    AddTargets(targets...).
    Build()
```

### 5. 平台特定配置

使用平台特定数据：

```go
msg := NewMessage("富文本消息").
    Body("普通文本内容").
    WithPlatformData("feishu", map[string]interface{}{
        "card": map[string]interface{}{
            "elements": []map[string]interface{}{
                {
                    "tag": "div",
                    "text": map[string]interface{}{
                        "content": "**富文本内容**",
                        "tag": "lark_md",
                    },
                },
            },
        },
    }).
    Build()
```

这份 API 参考文档涵盖了 NotifyHub 的所有公共接口和使用方法。更多示例和高级用法请参考项目的示例代码。
