# NotifyHub 优化实现建议

基于HTTP服务案例分析，以下是具体的、可立即实施的改进建议：

## 🎯 高优先级改进（立即可实施）

### 1. 改进测试配置
**位置**: `config/options.go`
```go
// 当前的WithTestDefaults缺少notifier
func WithTestDefaults() Option {
    return optionFunc(func(c *Config) {
        WithQueue("memory", 100, 1).apply(c)
        WithQueueRetryPolicy(queue.NoRetryPolicy()).apply(c)
        WithMockNotifier().apply(c) // ← 添加这一行
        if c.logger == nil {
            WithDefaultLogger(logger.Debug).apply(c)
        }
    })
}

// 新增mock notifier选项
func WithMockNotifier() Option {
    return optionFunc(func(c *Config) {
        // 添加一个总是成功的mock notifier
        mockNotifier := &MockNotifier{name: "mock"}
        c.notifiers = append(c.notifiers, mockNotifier)
    })
}
```

### 2. 添加便捷构造函数
**位置**: `client/hub.go`
```go
// 组合创建+启动
func NewAndStart(ctx context.Context, options ...config.Option) (*Hub, error) {
    hub, err := New(options...)
    if err != nil {
        return nil, err
    }

    if err := hub.Start(ctx); err != nil {
        hub.Stop() // 清理资源
        return nil, err
    }

    return hub, nil
}

// 测试专用构造函数
func NewForTesting() (*Hub, error) {
    return NewAndStart(context.Background(), config.WithTestDefaults())
}
```

### 3. 简化消息转换
**位置**: `notifiers/message_builder.go` (新文件)
```go
package notifiers

import (
    "encoding/json"
    "io"
    "net/http"
    "fmt"
    "time"
)

// 从HTTP请求直接创建消息
func NewMessageFromHTTPRequest(r *http.Request) (*Message, error) {
    var req struct {
        Title     string                 `json:"title"`
        Body      string                 `json:"body"`
        Targets   []TargetRequest        `json:"targets"`
        Priority  int                    `json:"priority,omitempty"`
        Format    string                 `json:"format,omitempty"`
        Template  string                 `json:"template,omitempty"`
        Variables map[string]interface{} `json:"variables,omitempty"`
        Metadata  map[string]interface{} `json:"metadata,omitempty"`
    }

    body, err := io.ReadAll(r.Body)
    if err != nil {
        return nil, err
    }

    if err := json.Unmarshal(body, &req); err != nil {
        return nil, err
    }

    return convertToMessage(&req), nil
}

// 从结构体创建消息（通用转换函数）
func convertToMessage(req interface{}) *Message {
    // 使用反射或类型断言进行通用转换
    // 减少handlers中的重复代码
}
```

### 4. 改善错误反馈
**位置**: `client/hub.go`
```go
// 改进SendResult结构
type DetailedSendResult struct {
    Target      Target
    Notifier    string
    Status      ResultStatus // Success, Failed, Skipped
    Reason      string       // 详细原因
    Error       error
    Duration    time.Duration
}

type SendResults struct {
    Total      int
    Successful int
    Failed     int
    Skipped    int
    Details    []DetailedSendResult
}

// 发送前验证
func (h *Hub) ValidateMessage(message *Message) *ValidationResult {
    result := &ValidationResult{Valid: true}

    for _, target := range message.Targets {
        compatible := h.findCompatibleNotifiers(target)
        if len(compatible) == 0 {
            result.Valid = false
            result.Errors = append(result.Errors, ValidationError{
                Target: target,
                Reason: fmt.Sprintf("No notifier supports target type '%s'", target.Type),
                Suggestion: h.suggestAlternatives(target),
            })
        }
    }

    return result
}

type ValidationResult struct {
    Valid  bool
    Errors []ValidationError
}

type ValidationError struct {
    Target     Target
    Reason     string
    Suggestion string
}
```

## 🚀 中优先级改进（后续实施）

### 5. 建造者模式API
**位置**: `client/builder.go` (新文件)
```go
type HubBuilder struct {
    options []config.Option
    autoStart bool
    ctx context.Context
}

func NewBuilder() *HubBuilder {
    return &HubBuilder{}
}

func (b *HubBuilder) WithFeishu(webhook, secret string) *HubBuilder {
    b.options = append(b.options, config.WithFeishu(webhook, secret))
    return b
}

func (b *HubBuilder) WithEmail(host string, port int, username, password, from string) *HubBuilder {
    b.options = append(b.options, config.WithEmail(host, port, username, password, from))
    return b
}

func (b *HubBuilder) AutoStart(ctx context.Context) *HubBuilder {
    b.autoStart = true
    b.ctx = ctx
    return b
}

func (b *HubBuilder) Build() (*Hub, error) {
    if b.autoStart && b.ctx != nil {
        return NewAndStart(b.ctx, b.options...)
    }
    return New(b.options...)
}
```

### 6. 消息建造者改进
**位置**: `notifiers/message_builder.go`
```go
type MessageBuilder struct {
    message *Message
}

func NewMessage() *MessageBuilder {
    return &MessageBuilder{
        message: &Message{
            ID:        generateID(),
            CreatedAt: time.Now(),
            Priority:  1,
            Format:    FormatText,
        },
    }
}

func (b *MessageBuilder) Title(title string) *MessageBuilder {
    b.message.Title = title
    return b
}

func (b *MessageBuilder) Body(body string) *MessageBuilder {
    b.message.Body = body
    return b
}

func (b *MessageBuilder) ToEmail(email string) *MessageBuilder {
    b.message.Targets = append(b.message.Targets, Target{
        Type:  TargetTypeEmail,
        Value: email,
    })
    return b
}

func (b *MessageBuilder) ToFeishu(target string) *MessageBuilder {
    b.message.Targets = append(b.message.Targets, Target{
        Type:     TargetTypeUser,
        Value:    target,
        Platform: "feishu",
    })
    return b
}

func (b *MessageBuilder) WithPriority(priority int) *MessageBuilder {
    b.message.Priority = priority
    return b
}

func (b *MessageBuilder) Build() *Message {
    return b.message
}
```

## 📊 使用效果对比

### 改进前 (当前状态)
```go
// 测试代码 - 复杂且容易出错
hub, err := client.New(
    config.WithTestDefaults(),
    config.WithFeishu("https://test", ""),
)
if err != nil { /* handle */ }
if err := hub.Start(ctx); err != nil { /* handle */ }
defer hub.Stop()

// 消息创建 - 需要40+行转换代码
message := h.convertToMessage(&req) // 复杂转换函数

// 错误反馈 - 不够详细
results, err := hub.Send(ctx, message, nil)
// 只知道"skipped"，不知道具体原因
```

### 改进后 (优化版本)
```go
// 测试代码 - 简洁明了
hub, err := client.NewForTesting()
if err != nil { /* handle */ }
defer hub.Close()

// 消息创建 - 一行代码
message, err := notifiers.NewMessageFromHTTPRequest(httpReq)
// 或者使用建造者
message := notifiers.NewMessage().
    Title(req.Title).
    Body(req.Body).
    ToEmail(req.Email).
    Build()

// 错误反馈 - 详细且有建议
validation := hub.ValidateMessage(message)
if !validation.Valid {
    for _, err := range validation.Errors {
        log.Printf("Target %s failed: %s. Suggestion: %s",
            err.Target.Value, err.Reason, err.Suggestion)
    }
}
```

## ⚡ 立即可执行的改进

1. **修改** `config/options.go` 中的 `WithTestDefaults()` 函数
2. **添加** `client/convenience.go` 文件，包含 `NewForTesting()` 等函数
3. **创建** `notifiers/http_helpers.go` 文件，包含HTTP请求转换函数
4. **扩展** `client/hub.go` 中的返回结果，提供更详细的错误信息

这些改进都是向后兼容的，不会破坏现有代码，但能显著改善使用体验。