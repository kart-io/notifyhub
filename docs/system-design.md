# NotifyHub 系统技术方案文档 (v4.0 最终版)

## 1. 概述

NotifyHub 是一个 Go 语言编写的**通知发送库**（Library/Package），可以被其他 Go 项目引入使用。它提供了统一的 API 来发送各类通知消息，而非独立部署的服务。

### 1.1. 定位说明

- **类型**: Go Package/Library（非独立服务）
- **使用方式**: 通过 `go get` 引入项目使用
- **运行环境**: 作为宿主应用的一部分运行
- **依赖管理**: 最小化外部依赖，核心功能零依赖

> 详细的项目背景、目标和需求，请参阅 **[项目需求文档](./project-requirements.md)**。

---

## 2. 核心架构与工作流程

### 2.1. 架构图

```mermaid
graph TD
    subgraph "Host Application (Your Go App)"
        A[业务代码]

        subgraph "NotifyHub Library"
            B[Builder API]
            C[Hub 核心]
            D[Router]

            subgraph "内置模块"
                E[Queue System]
                F[Notifiers]
                G[Template Engine]
                H[Logger]
            end
        end
    end

    subgraph "External Services"
        J[Email Server]
        K[Feishu/Slack/etc]
        L[Redis/Kafka<br>(可选)]
    end

    A -- "notifyhub.New()" --> B;
    B -- "创建实例" --> C;

    C -- "SendSync()" --> D;
    C -- "Send()" --> E;

    D -- "路由" --> F;
    E -- "异步处理" --> F;

    F -- "渲染模板" --> G;
    F -- "发送通知" --> J;
    F -- "发送通知" --> K;

    E -.-> L;

    style A fill:#e1f5fe
    style B fill:#fff9c4
    style C fill:#fff9c4
    style D fill:#fff9c4
    style E fill:#f3e5f5
    style F fill:#f3e5f5
    style G fill:#f3e5f5
    style H fill:#f3e5f5
```

**说明**：
- **蓝色部分**：宿主应用的业务代码
- **黄色部分**：NotifyHub 库的核心API
- **紫色部分**：NotifyHub 的内置功能模块
- **实线**：必需的调用流程
- **虚线**：可选的外部依赖（如使用外部队列）

### 2.2. 工作流程

#### 2.2.1 集成方式
```go
// 1. 安装包
// go get github.com/kart-io/notifyhub

// 2. 在应用中初始化
import "github.com/kart-io/notifyhub"

func main() {
    // 创建 NotifyHub 实例
    hub := notifyhub.New(
        notifyhub.WithFeishu(webhookURL, secret),
        notifyhub.WithEmail(smtpConfig),
        notifyhub.WithQueue(queueConfig),
        notifyhub.WithLogger(logger),
        notifyhub.WithRetryConfig(retryConfig),
    )

    // 在应用生命周期中使用
    defer hub.Stop(context.Background())
}
```

#### 2.2.2 同步发送流程
1. **消息验证**: 检查必填字段、格式验证
2. **路由决策**: Router根据channel选择对应的Notifier
3. **模板渲染**: 如果指定了模板，进行内容渲染
4. **同步发送**: 直接调用Notifier发送消息
5. **结果返回**: 立即返回发送结果给调用者

#### 2.2.3 异步发送流程
1. **消息入队**: 将消息封装为Job推入队列
2. **Worker处理**: 后台Worker从队列取出Job
3. **消息路由**: Router分发到对应Notifier
4. **发送执行**: Notifier执行实际发送
5. **结果处理**:
   - 成功: 调用成功回调，更新Job状态
   - 失败: 根据重试策略决定是否重试
   - 重试: 计算下次重试时间，重新入队
   - 最终失败: 进入死信队列，调用失败回调

#### 2.2.4 错误处理机制
- **可重试错误**: 网络超时、限流、服务暂时不可用
- **不可重试错误**: 认证失败、参数错误、收件人无效
- **熔断机制**: 连续失败达到阈值后暂停该通道
- **降级策略**: 主通道失败时自动切换备用通道

---

## 3. 模块化设计

本系统的核心功能被拆分为多个独立的、可插拔的模块。以下是各模块的职责摘要，详细设计请参阅其独立的子文档。

### 3.1. 核心 Hub 与客户端 ✅ 已实现并验证
`Hub` 是用户与 `notifyhub` 交互的唯一入口，负责协调内部所有模块。

#### 核心Hub结构 (实际实现)
```go
// client/hub.go:18-31
type Hub struct {
    config    *config.Config              // 配置管理
    notifiers map[string]notifiers.Notifier // 通知器映射
    queue     queue.Queue                  // 队列系统
    templates *template.Engine             // 模板引擎
    routing   *config.RoutingEngine       // 路由引擎
    metrics   *monitoring.Metrics         // 监控指标
    logger    logger.Interface            // 日志接口

    mu      sync.RWMutex                  // 读写锁
    started bool                          // 启动状态
    stopCh  chan struct{}                 // 停止信号
    workers []chan struct{}               // 工作协程
}
```

#### 核心API接口 (实际实现)
```go
// 构造函数
func New(opts ...config.Option) (*Hub, error)                // 建造者模式构造

// 生命周期管理
func (h *Hub) Start(ctx context.Context) error               // 启动服务和工作协程
func (h *Hub) Stop() error                                   // 停止服务

// 发送方法
func (h *Hub) Send(ctx, message, options) ([]*SendResult, error)         // 智能发送(支持同步/异步)
func (h *Hub) SendSync(ctx, message, options) ([]*SendResult, error)     // 同步发送
func (h *Hub) SendAsync(ctx, message, options) (string, error)           // 异步发送(返回taskID)
func (h *Hub) SendBatch(ctx, messages, options) ([]*SendResult, error)   // 批量发送(支持同步/异步) ✅ 新增

// 便捷发送方法
func (h *Hub) SendText(ctx, title, body, targets...)                // 发送纯文本
func (h *Hub) SendAlert(ctx, title, body, targets...)               // 发送告警
func (h *Hub) SendWithTemplate(ctx, template, variables, targets...)// 模板发送

// 监控和健康检查
func (h *Hub) GetMetrics() map[string]interface{}            // 获取监控指标
func (h *Hub) GetHealth(ctx) map[string]interface{}          // 健康检查
```

#### 配置选项（建造者模式 - 实际实现）
**平台通知器配置**：
- `WithFeishu(webhookURL, secret)` - 直接配置飞书
- `WithFeishuFromEnv()` - 从环境变量配置飞书
- `WithEmail(host, port, username, password, from)` - 直接配置邮件
- `WithEmailFromEnv()` - 从环境变量配置邮件

**队列和重试配置**：
- `WithQueue(queueType, bufferSize, workers)` - 队列配置
- `WithQueueFromEnv()` - 从环境变量配置队列
- `WithQueueRetryPolicy(*RetryPolicy)` - 重试策略

**路由配置**：
- `WithRouting(rules...)` - 自定义路由规则
- `WithDefaultRouting()` - 默认路由规则

**日志配置**：
- `WithLogger(logger.Interface)` - 自定义日志器
- `WithDefaultLogger(level)` - 默认日志器with级别
- `WithSilentLogger()` - 静默日志

**预设组合**：
- `WithDefaults()` - 从环境变量加载默认配置
- `WithTestDefaults()` - 测试友好的默认配置

### 3.2. 队列系统 ✅ 已实现并验证
队列系统是实现异步处理的核心，支持内置的内存队列和第三方的持久化队列。

#### 核心队列接口 (实际实现)
```go
// queue/queue.go:14-35
type Queue interface {
    Enqueue(ctx context.Context, msg *Message) (string, error)  // Q-FR1: 异步消息入队
    Dequeue(ctx context.Context) (*Message, error)              // 消息出队
    Ack(msgID string) error                                     // 确认处理成功
    Nack(msgID string, nextRetry time.Time) error               // Q-FR3: 失败重试调度
    Close() error                                               // Q-NFR2: 优雅停机
    Size() int                                                  // 队列大小监控
    Health(ctx context.Context) error                           // 健康检查
}
```

#### 内存队列实现 (实际实现)
```go
// queue/queue.go:50-67 - Q-EXT2: 内置内存队列
type SimpleQueue struct {
    messages map[string]*Message    // 消息存储
    pending  chan *Message          // 缓冲队列
    mu       sync.RWMutex          // 并发安全
    closed   bool                  // 关闭状态
}

func NewSimple(bufferSize int) *SimpleQueue  // 配置缓冲区大小
```

#### Worker池系统 (实际实现)
```go
// queue/worker.go:18-46
type Worker struct {
    queue       Queue           // 队列引用
    sender      MessageSender   // 消息发送器
    retryPolicy *RetryPolicy    // Q-FR3: 重试策略
    concurrency int             // 并发Worker数量
    stopCh      chan struct{}   // Q-NFR2: 优雅停机信号
    callbacks   *CallbackExecutor // 回调执行器
}

func NewWorker(queue, sender, retryPolicy, concurrency) *Worker
func (w *Worker) Start(ctx context.Context) error    // 启动Worker池
func (w *Worker) Stop()                             // Q-NFR2: 优雅停机
```

#### 重试策略系统 (实际实现)
```go
// queue/retry.go:6-19 - Q-FR3: 消息重试
type RetryPolicy struct {
    MaxRetries      int           // 最大重试次数
    InitialInterval time.Duration // 初始重试间隔
    Multiplier      float64       // 指数退避倍数
}

func DefaultRetryPolicy() *RetryPolicy              // 默认策略 (3次，30秒，2.0倍)
func ExponentialBackoffPolicy(...) *RetryPolicy    // 指数退避策略
func NoRetryPolicy() *RetryPolicy                  // 禁用重试
func AggressiveRetryPolicy() *RetryPolicy          // 激进重试策略
```

#### 回调系统 (实际实现)
```go
// queue/callback.go:18-50
type CallbackEvent string
const (
    CallbackEventSent       = "sent"        // 发送成功
    CallbackEventFailed     = "failed"      // 发送失败
    CallbackEventRetry      = "retry"       // 重试中
    CallbackEventMaxRetries = "max_retries" // 达到最大重试次数
)

type Callback interface {
    Execute(ctx context.Context, callbackCtx *CallbackContext) error
    Name() string
}
```

**验证结果**:
- ✅ Q-FR1: 异步消息入队 - `Enqueue()` 方法实现
- ✅ Q-FR3: 消息重试 - 完整的重试策略和Worker重试逻辑
- ✅ Q-EXT1: 可插拔后端接口 - 标准Queue接口抽象
- ✅ Q-EXT2: 内置内存队列 - SimpleQueue实现，零外部依赖
- ✅ Q-NFR1: 至少一次投递 - Ack/Nack机制保证
- ✅ Q-NFR2: 优雅停机 - Worker.Stop()和Queue.Close()实现

> **[➡️ 查看队列子系统需求](./queue-system-requirements.md)**

### 3.3. 通知器模块 ✅ Feishu+Email已实现并验证
`Notifier` 是所有通知渠道的统一抽象，采用开放的插件化架构。

#### 核心通知器接口 (实际实现)
```go
// notifiers/base.go:59-65
type Notifier interface {
    Name() string                                           // 通知器名称标识
    Send(ctx, message) ([]*SendResult, error)               // FR2/FR3: 发送通知
    SupportsTarget(target Target) bool                      // NFR4: 目标支持检查
    Health(ctx context.Context) error                       // 健康检查
}
```

#### 消息和目标结构 (实际实现)
```go
// notifiers/base.go:8-46
type Target struct {
    Type     TargetType        // "group", "user", "email"
    Value    string            // 具体标识: user_id, group_id, email
    Platform string            // 平台标识: "feishu", "email"
    Metadata map[string]string // 平台特定元数据
}

type Message struct {
    Title     string                 // 消息标题
    Body      string                 // 消息正文
    Format    MessageFormat          // "text", "markdown", "html"
    Targets   []Target               // 目标列表
    Template  string                 // 模板名称
    Variables map[string]interface{} // 模板变量
    Priority  int                    // 优先级 1-5
}

type SendResult struct {
    Target   Target        // 发送目标
    Platform string        // 平台名称
    Success  bool          // 是否成功
    Error    string        // 错误信息
    Duration time.Duration // 发送耗时
    SentAt   time.Time     // 发送时间
    Attempts int           // 尝试次数
}
```

#### Feishu通知器实现 (实际实现)
```go
// notifiers/feishu.go:17-35
type FeishuNotifier struct {
    webhookURL string        // 飞书Webhook URL
    secret     string        // 签名密钥
    timeout    time.Duration // 请求超时
    client     *http.Client  // HTTP客户端
}

func NewFeishuNotifier(webhookURL, secret, timeout) *FeishuNotifier

// 主要方法实现
func (f *FeishuNotifier) Name() string                          // 返回 "feishu"
func (f *FeishuNotifier) SupportsTarget(target Target) bool     // 支持 group/user 类型
func (f *FeishuNotifier) Send(ctx, message) ([]*SendResult, error) // 发送实现
func (f *FeishuNotifier) Health(ctx context.Context) error      // 健康检查
```

#### Feishu发送功能特性 (实际实现)
- **消息格式支持**: 支持文本和Markdown格式，自动转换为飞书协议格式
- **目标支持**: 支持群组(@all)和用户(@mention)通知
- **错误处理**: 完整的HTTP状态码处理和错误信息返回
- **性能监控**: 记录发送耗时和尝试次数
- **内容格式化**: 自动处理标题+正文格式，支持元数据追加

#### Email通知器实现 (实际实现)
```go
// notifiers/email.go:15-39
type EmailNotifier struct {
    host     string        // SMTP服务器地址
    port     int           // SMTP端口
    username string        // SMTP用户名
    password string        // SMTP密码
    from     string        // 发件人地址
    useTLS   bool          // 是否使用TLS
    timeout  time.Duration // 连接超时
}

func NewEmailNotifier(host, port, username, password, from, useTLS, timeout) *EmailNotifier

// 主要方法实现
func (e *EmailNotifier) Name() string                          // 返回 "email"
func (e *EmailNotifier) SupportsTarget(target Target) bool     // 支持 email 类型和@邮箱地址
func (e *EmailNotifier) Send(ctx, message) ([]*SendResult, error) // SMTP发送实现
func (e *EmailNotifier) Health(ctx context.Context) error      // SMTP连接和认证检查
```

#### Email发送功能特性 (实际实现)
- **SMTP协议支持**: 标准SMTP发送，支持Plain Auth认证
- **格式转换**: 支持Markdown/HTML到纯文本的智能转换
- **邮件结构**: 完整的邮件头（From/To/Subject/MIME）和正文格式
- **健康检查**: 实时SMTP服务器连接和认证验证
- **错误处理**: 详细的SMTP错误信息和连接状态报告
- **内容增强**: 自动添加元数据和时间戳信息

**验证结果**:
- ✅ FR2: 支持飞书通知渠道 - FeishuNotifier完整实现
- ✅ FR3: 支持邮件通知渠道 - EmailNotifier完整实现
- ✅ NFR4: 可扩展性 - 标准Notifier接口，插件化架构
- ✅ 消息格式转换 - 飞书支持text/markdown，Email支持markdown/html到text
- ✅ 目标类型支持 - 飞书群组/用户，Email地址完整支持
- ✅ 错误处理机制 - HTTP/SMTP状态检查和详细错误报告
- ✅ 健康检查机制 - 所有通知器支持实时健康状态检测

> **[➡️ 查看通知器模块详细设计](./notifier-module-design.md)**

### 3.4. 路由与消息 ✅ 已实现并验证
定义了核心数据结构 `Message` 以及 `Router` 组件如何分发消息。

#### 核心路由引擎 (实际实现)
```go
// config/routing.go:10-22
type RoutingEngine struct {
    rules []RoutingRule    // 活跃路由规则列表
}

func NewRoutingEngine(rules []RoutingRule) *RoutingEngine
func (r *RoutingEngine) ProcessMessage(message *notifiers.Message) *notifiers.Message
func (r *RoutingEngine) AddRule(rule RoutingRule)        // 动态添加规则
func (r *RoutingEngine) RemoveRule(name string)          // 动态删除规则
func (r *RoutingEngine) GetRules() []RoutingRule         // 获取当前规则
```

#### 路由规则结构 (实际实现)
```go
// config/options.go:201-217
type RoutingRule struct {
    Name       string         // 规则名称
    Enabled    bool          // 是否启用
    Conditions RuleConditions // 匹配条件
    Actions    []RuleAction   // 执行动作
}

type RuleConditions struct {
    MessageType []string            // 消息类型条件
    Priority    []int               // 优先级条件
    Metadata    map[string]string   // 元数据条件
}

type RuleAction struct {
    Type      string   // 动作类型: "route"
    Platforms []string // 目标平台列表
}
```

#### 路由规则建造者 (实际实现)
```go
// config/routing.go:132-187
type RoutingRuleBuilder struct {
    rule RoutingRule
}

func NewRoutingRule(name) *RoutingRuleBuilder                      // 创建规则建造者
func (b *RoutingRuleBuilder) Enabled(enabled bool) *Builder        // 设置启用状态
func (b *RoutingRuleBuilder) WithPriority(priorities...) *Builder  // 添加优先级条件
func (b *RoutingRuleBuilder) WithMessageType(types...) *Builder    // 添加消息类型条件
func (b *RoutingRuleBuilder) WithMetadata(key, value) *Builder     // 添加元数据条件
func (b *RoutingRuleBuilder) RouteTo(platforms...) *Builder        // 设置路由目标
func (b *RoutingRuleBuilder) Build() RoutingRule                   // 构建规则
```

#### 默认路由规则 (实际实现)
```go
// config/options.go:232-254 - WithDefaultRouting()
{
    Name: "high_priority_all",
    Enabled: true,
    Conditions: RuleConditions{Priority: []int{4, 5}},
    Actions: []RuleAction{{Type: "route", Platforms: []string{"feishu", "email"}}},
},
{
    Name: "alerts_to_feishu",
    Enabled: true,
    Conditions: RuleConditions{Metadata: map[string]string{"type": "alert"}},
    Actions: []RuleAction{{Type: "route", Platforms: []string{"feishu"}}},
}
```

#### 路由处理流程 (实际实现)
1. **规则匹配**: `matchesRule()` 检查优先级、元数据、消息类型条件
2. **规则应用**: `applyRule()` 根据动作过滤和设置目标平台
3. **消息处理**: 在 Hub.SendSync 中调用 `routing.ProcessMessage()`
4. **首个匹配**: 应用第一个匹配的规则后停止 (`break`)

**验证结果**:
- ✅ FR9: 路由机制 - 完整的规则引擎和条件匹配
- ✅ 规则配置 - 支持优先级、元数据、消息类型条件
- ✅ 平台路由 - 动态平台过滤和目标设置
- ✅ 建造者模式 - 流畅的规则构建API
- ✅ 动态管理 - 运行时添加/删除规则支持
- ✅ 默认配置 - 预置高优先级和告警路由规则

> **[➡️ 查看路由与消息详细设计](./routing-message-design.md)**

### 3.5. 模板引擎 ✅ 已实现并验证
负责消息内容的渲染，将模板加载与渲染分离。

#### 核心模板引擎 (实际实现)
```go
// template/engine.go:19-36
type Engine struct {
    textTemplates map[string]*textTemplate.Template  // 文本模板缓存
    htmlTemplates map[string]*htmlTemplate.Template  // HTML模板缓存
    funcMap       textTemplate.FuncMap               // 模板函数集合
}

func NewEngine() *Engine                                                    // 创建引擎
func (e *Engine) RenderMessage(message) (*Message, error)                  // 渲染消息
func (e *Engine) AddTextTemplate(name, templateText string) error          // 添加文本模板
func (e *Engine) AddHTMLTemplate(name, templateText string) error          // 添加HTML模板
```

#### 模板渲染功能 (实际实现)
```go
// template/engine.go:95-129
func (e *Engine) RenderMessage(message *notifiers.Message) (*notifiers.Message, error) {
    // 1. 智能检测：仅在需要时进行模板渲染
    if message.Template == "" && !strings.Contains(message.Title+message.Body, "{{") {
        return message, nil // 无模板内容
    }

    // 2. 创建模板数据上下文
    data := createTemplateData(message)

    // 3. 渲染标题（如果包含模板语法）
    if strings.Contains(message.Title, "{{") {
        rendered.Title = renderString(message.Title, data, format)
    }

    // 4. 渲染正文：命名模板优先，内联模板次之
    if message.Template != "" {
        rendered.Body = renderTemplate(message.Template, data, format)
    } else if strings.Contains(message.Body, "{{") {
        rendered.Body = renderString(message.Body, data, format)
    }
}
```

#### 内置模板函数 (实际实现)
```go
// template/engine.go:39-58 - createFuncMap()
textTemplate.FuncMap{
    "upper":      strings.ToUpper,
    "lower":      strings.ToLower,
    "title":      strings.Title,
    "trim":       strings.TrimSpace,
    "now":        time.Now,
    "formatTime": func(t time.Time, layout string) string,
    "default":    func(defaultValue, value interface{}) interface{},
    "join":       func(sep string, elems []string) string,
    "contains":   strings.Contains,
}
```

#### 内置模板集合 (实际实现)
- **Alert模板**: 告警消息格式化，包含服务器、环境、错误信息
- **Notice模板**: 通知消息格式化，支持变量替换
- **Report模板**: 报告消息格式化，支持数据表格展示

#### 模板数据上下文 (实际实现)
```go
// 为模板提供完整的消息上下文
templateData := map[string]interface{}{
    "Title":     message.Title,
    "Body":      message.Body,
    "Variables": message.Variables,    // 用户自定义变量
    "Metadata":  message.Metadata,    // 消息元数据
    "CreatedAt": message.CreatedAt,   // 创建时间
    "Priority":  message.Priority,    // 优先级
}
```

#### 格式支持 (实际实现)
- **文本模板**: 使用Go标准库`text/template`，适用于纯文本和Markdown
- **HTML模板**: 使用Go标准库`html/template`，自动HTML转义
- **智能选择**: 根据消息格式自动选择对应的模板引擎

**验证结果**:
- ✅ FR5: 模板支持 - 完整的模板渲染系统
- ✅ FR6: 多模板引擎支持 - text/template和html/template双引擎
- ✅ 模板缓存 - 编译后模板内存缓存提升性能
- ✅ 内置函数 - 丰富的字符串、时间、条件处理函数
- ✅ 变量注入 - 完整的消息上下文和用户变量支持
- ✅ 智能渲染 - 仅在需要时执行模板渲染，提升性能
- ✅ 内置模板 - 预置alert/notice/report等常用模板

> **[➡️ 查看模板引擎详细设计](./template-module-design.md)**

### 3.6. 日志模块 ✅ 已实现并验证
提供统一、可插拔的日志记录能力。

#### 核心日志接口 (实际实现)
```go
// logger/interface.go:44-62
type Interface interface {
    LogMode(level LogLevel) Interface                     // 设置日志级别
    Info(ctx, msg string, data ...interface{})          // Info级别日志
    Warn(ctx, msg string, data ...interface{})          // 警告级别日志
    Error(ctx, msg string, data ...interface{})         // 错误级别日志
    Debug(ctx, msg string, data ...interface{})         // 调试级别日志
    Trace(ctx, begin, fc func(), err error)             // 性能追踪日志
}

type LogLevel int
const (
    Silent LogLevel = iota + 1  // 静默模式
    Error                       // 仅错误日志
    Warn                        // 警告和错误
    Info                        // 信息、警告和错误
    Debug                       // 所有级别
)
```

#### 默认日志实现 (实际实现)
```go
// logger/logger.go:29-70
type logger struct {
    Writer                              // 日志输出接口
    Config                              // 日志配置
    infoStr, warnStr, errStr, debugStr string  // 格式字符串
    traceStr, traceErrStr, traceWarnStr string // 追踪格式字符串
}

func NewLogger(writer Writer, config Config) Interface
func Default() Interface                          // 创建默认日志器
func (l *logger) LogMode(level LogLevel) Interface // 设置日志级别
```

#### 适配器系统 (实际实现)
```go
// logger/adapters/adapters.go:16-48
type AdapterBase struct {
    level logger.LogLevel
}

// 支持的适配器类型:
type CustomLogger interface {
    Log(level LogLevel, msg string, fields map[string]interface{})
}

// 预置适配器函数 (notifyhub.go中暴露):
func NewStdLogAdapter(stdLogger, level) Logger      // 标准log适配器
func NewLogrusAdapter(logrusLogger, level) Logger   // Logrus适配器
func NewZapAdapter(zapLogger, level) Logger         // Zap适配器
func NewKartLoggerAdapter(kartLogger, level) Logger // Kart日志适配器
func NewCustomAdapter(customLogger, level) Logger   // 自定义适配器
```

#### Hub中的日志使用 (实际实现)
```go
// client/hub.go 中的广泛使用示例:
hub.logger.Info(ctx, "NotifyHub initializing with config: queue_type=%s, buffer_size=%d, workers=%d", ...)
hub.logger.Info(ctx, "Feishu notifier initialized with webhook: %s", maskWebhookURL(...))
hub.logger.Error(ctx, "No notifiers configured")
h.logger.Warn(ctx, "NotifyHub is already started")
h.logger.Trace(ctx, start, func() (string, int64) { ... }, traceErr)
```

#### 配置选项 (实际实现)
```go
// config/options.go 中的日志配置:
func WithLogger(logger logger.Interface) Option           // 自定义日志器
func WithDefaultLogger(level logger.LogLevel) Option      // 默认日志器+级别
func WithSilentLogger() Option                           // 静默日志器

// 默认配置中的日志设置:
WithDefaults() -> WithDefaultLogger(logger.Warn)        // 默认使用Warn级别
```

#### 性能追踪功能 (实际实现)
- **Trace方法**: 记录操作耗时和结果统计
- **使用场景**: Hub.SendSync中追踪消息发送全流程
- **上下文传递**: 所有日志方法支持context.Context
- **格式化日志**: 支持Printf风格的格式化字符串和参数

**验证结果**:
- ✅ FR13: 统一日志记录 - 完整的日志接口系统
- ✅ NFR5: 可观测性 - 性能追踪和结构化日志
- ✅ 可插拔性 - 适配器模式支持多种日志库集成
- ✅ 日志级别控制 - 支持Silent/Error/Warn/Info/Debug五个级别
- ✅ 上下文支持 - 所有日志方法支持context传递
- ✅ 性能追踪 - Trace方法支持操作耗时和结果统计
- ✅ 彩色输出 - 默认实现支持控制台彩色日志输出

> **[➡️ 查看日志模块详细设计](./logger-module-design.md)**

---

## 4. 核心数据结构

本章节定义系统中最核心的公共结构体和接口。

```go
// Message 是系统中流转的核心数据对象
type Message struct {
    // 基础标识
    ID           string                 `json:"id"`           // 消息唯一标识
    Channel      string                 `json:"channel"`      // 通知渠道: email/feishu/slack等
    Recipients   []string               `json:"recipients"`   // 收件人列表

    // 消息内容
    Subject      string                 `json:"subject,omitempty"`   // 消息主题
    Content      string                 `json:"content,omitempty"`   // 消息正文
    TemplateName string                 `json:"templateName,omitempty"` // 模板名称
    TemplateData interface{}            `json:"templateData,omitempty"` // 模板数据
    Payload      map[string]interface{} `json:"payload,omitempty"`   // 扩展数据

    // 调度控制
    Priority     Priority               `json:"priority"`     // 优先级: Low/Normal/High/Critical
    Delay        time.Duration          `json:"delay,omitempty"` // 延迟发送
    ExpireAt     *time.Time            `json:"expireAt,omitempty"` // 过期时间

    // 重试配置
    Retry        *RetryConfig           `json:"retry,omitempty"` // 重试策略

    // 追踪信息
    TraceID      string                 `json:"traceId,omitempty"` // 分布式追踪ID
    CreatedAt    time.Time             `json:"createdAt"`    // 创建时间
    Metadata     map[string]string      `json:"metadata,omitempty"` // 元数据
}

// Priority 消息优先级
type Priority int

const (
    PriorityLow Priority = iota
    PriorityNormal
    PriorityHigh
    PriorityCritical
)

// RetryConfig 定义了任务失败后的重试策略
type RetryConfig struct {
    MaxRetries      int           `json:"maxRetries"`      // 最大重试次数
    InitialInterval time.Duration `json:"initialInterval"` // 初始重试间隔
    MaxInterval     time.Duration `json:"maxInterval"`     // 最大重试间隔
    Multiplier      float64       `json:"multiplier"`      // 退避倍数
    Jitter          bool          `json:"jitter"`          // 是否添加随机抖动
}

// Result 包含了异步任务的最终执行结果
type Result struct {
    MessageID    string        `json:"messageId"`    // 消息ID
    Success      bool          `json:"success"`      // 是否成功
    Error        error         `json:"error,omitempty"` // 错误信息
    Attempts     int           `json:"attempts"`     // 尝试次数
    Duration     time.Duration `json:"duration"`     // 处理耗时
    ProviderID   string        `json:"providerId,omitempty"` // 第三方返回ID
    Timestamp    time.Time     `json:"timestamp"`    // 完成时间
}

// Callback 是单条消息完成回调的接口
type Callback interface {
    OnSuccess(result *Result)  // 成功回调
    OnFailure(result *Result)  // 失败回调
    OnRetry(result *Result)    // 重试回调
}

// Job 队列中的任务单元
type Job struct {
    ID          string          `json:"id"`
    Message     *Message        `json:"message"`
    Status      JobStatus       `json:"status"`
    CreatedAt   time.Time       `json:"createdAt"`
    UpdatedAt   time.Time       `json:"updatedAt"`
    NextRetryAt *time.Time      `json:"nextRetryAt,omitempty"`
    Attempts    int             `json:"attempts"`
    LastError   string          `json:"lastError,omitempty"`
}

// JobStatus 任务状态
type JobStatus string

const (
    JobStatusPending    JobStatus = "pending"
    JobStatusProcessing JobStatus = "processing"
    JobStatusCompleted  JobStatus = "completed"
    JobStatusFailed     JobStatus = "failed"
    JobStatusDeadLetter JobStatus = "dead_letter"
)
```

---

## 5. 横切关注点

### 5.1. 错误处理与可靠性 ✅ 已实现并验证

#### 重试策略系统 (实际实现)
```go
// queue/retry.go:6-19
type RetryPolicy struct {
    MaxRetries      int           // 最大重试次数
    InitialInterval time.Duration // 初始重试间隔
    Multiplier      float64       // 指数退避倍数
}

func DefaultRetryPolicy() *RetryPolicy                    // 默认策略(3次，30秒，2.0倍)
func ExponentialBackoffPolicy(...) *RetryPolicy           // 指数退避策略
func LinearBackoffPolicy(maxRetries, interval) *RetryPolicy // 线性退避策略
func NoRetryPolicy() *RetryPolicy                         // 禁用重试
func AggressiveRetryPolicy() *RetryPolicy                 // 激进重试策略(5次，10秒，1.5倍)
```

#### 回调事件系统 (实际实现)
```go
// queue/callback.go:18-29
type CallbackEvent string
const (
    CallbackEventSent       = "sent"        // 发送成功
    CallbackEventFailed     = "failed"      // 发送失败
    CallbackEventRetry      = "retry"       // 重试中
    CallbackEventMaxRetries = "max_retries" // 达到最大重试次数
)

type CallbackContext struct {
    MessageID   string                  // 消息ID
    Event       CallbackEvent           // 事件类型
    Message     *notifiers.Message      // 原始消息
    Results     []*notifiers.SendResult // 发送结果
    Error       error                   // 错误信息
    Attempts    int                     // 尝试次数
    ExecutedAt  time.Time               // 执行时间
    Duration    time.Duration           // 处理耗时
}
```

#### Worker错误处理流程 (实际实现)
```go
// queue/worker.go:112-142
if hasFailures {
    // 1. 记录错误信息
    queueMsg.LastError = getErrorMessage(err, results)

    // 2. 执行失败回调
    callbacks.ExecuteCallbacks(ctx, CallbackEventFailed, queueMsg, results, err, duration)

    if retryPolicy.ShouldRetry(queueMsg.Attempts) {
        // 3. 执行重试回调
        callbacks.ExecuteCallbacks(ctx, CallbackEventRetry, queueMsg, results, err, duration)

        // 4. 计算下次重试时间(指数退避)
        nextRetry := retryPolicy.NextRetry(queueMsg.Attempts)

        // 5. 重新入队
        queue.Nack(queueMsg.ID, nextRetry)
        go func() {
            time.Sleep(time.Until(nextRetry))
            queue.Enqueue(context.Background(), queueMsg)
        }()
    } else {
        // 6. 达到最大重试次数
        callbacks.ExecuteCallbacks(ctx, CallbackEventMaxRetries, queueMsg, results, err, duration)
        queue.Ack(queueMsg.ID) // 移出队列
    }
}
```

#### 同步错误处理 (实际实现)
- **直接返回错误**: SendSync()立即返回详细错误信息
- **结果聚合**: 收集所有通知器的发送结果和错误
- **部分成功处理**: 允许部分通知器成功，部分失败
- **超时控制**: 每个通知器独立超时设置

**验证结果**:
- ✅ FR7: 异步重试机制 - 完整的指数退避重试实现
- ✅ FR8: 失败处理机制 - 多层次回调事件系统
- ✅ NFR2: 可靠性 - Ack/Nack机制保证至少一次投递
- ✅ 指数退避算法 - 避免惊群效应的智能重试间隔
- ✅ 回调机制 - 支持sent/failed/retry/max_retries四种事件
- ✅ 超时控制 - 发送和重试的完整超时管理

> 详细设计请参阅 **[错误处理与可靠性技术设计](./error-handling-design.md)**

### 5.2. 安全性
- **凭证管理**:
  - 环境变量注入敏感配置
  - 支持密钥管理服务集成
  - 定期轮转API密钥
- **数据保护**:
  - 传输层TLS加密
  - 敏感信息自动脱敏
  - 审计日志记录
- **访问控制**:
  - API密钥认证
  - 速率限制保护

### 5.3. 可观测性
- **结构化日志**:
  - 统一的日志格式
  - 支持多种日志级别
  - 与现有日志系统集成
- **指标监控**:
  - 发送成功率、延迟分布
  - 队列长度、处理速度
  - 资源使用情况
- **分布式追踪**:
  - TraceID贯穿全流程
  - OpenTelemetry集成
  - 端到端链路可视化

### 5.4. 测试策略
- **单元测试**:
  - 目标覆盖率 > 80%
  - Mock外部依赖
  - 边界条件测试
- **集成测试**:
  - Docker容器化测试环境
  - 端到端流程验证
  - 故障注入测试
- **性能测试**:
  - 压力测试和负载测试
  - 内存泄漏检测
  - 基准测试对比

### 5.5. 性能优化
- **并发处理**: Worker池动态调整
- **批量操作**: 批量发送接口优化
- **缓存策略**: 模板缓存、连接池复用
- **资源限制**: 内存和CPU使用限制

---

## 6. 发展路线图

### Phase 1: 核心功能 (v1.0 - 2个月)
**目标**: 实现基础的同步/异步消息发送能力

- [ ] **Week 1-2**: 核心接口定义
  - Queue、Notifier、Router接口设计
  - Message、Result等核心数据结构
- [ ] **Week 3-4**: 内存队列实现
  - Worker池管理
  - 基础的重试机制
- [ ] **Week 5-6**: 基础通知器
  - Email通知器
  - Feishu通知器
  - Webhook通知器
- [ ] **Week 7-8**: 集成测试
  - 单元测试框架搭建
  - 端到端测试用例
  - CI/CD pipeline配置

### Phase 2: 扩展功能 (v2.0 - 3个月)
**目标**: 丰富通知渠道，增强高级特性

- [ ] **Month 1**: 更多通知器
  - Slack、DingTalk、SMS通知器
  - 通知器健康检查机制
- [ ] **Month 2**: 高级队列特性
  - 消息优先级处理
  - 延迟消息支持
  - 死信队列处理
- [ ] **Month 3**: 模板系统
  - 多模板引擎支持
  - 模板缓存优化
  - 国际化支持

### Phase 3: 生产级特性 (v3.0 - 3个月)
**目标**: 企业级可靠性和可观测性

- [ ] **Month 1**: 持久化队列
  - Redis Queue适配器
  - Kafka Queue适配器
  - 消息持久化和恢复
- [ ] **Month 2**: 可观测性
  - OpenTelemetry集成
  - Prometheus指标导出
  - 监控大盘和告警
- [ ] **Month 3**: 文档和工具
  - 完整的API文档
  - 管理界面原型
  - Docker镜像和Helm charts

### 长期规划 (v4.0+)
- **多租户支持**: 资源隔离、配额管理
- **智能路由**: 基于规则的动态路由
- **A/B测试**: 消息模板A/B测试能力
- **AI增强**: 智能内容生成和优化

---

## 7. 附录

### 7.1. 性能指标

| 指标 | 目标值 | 测量方法 |
|------|--------|----------|
| 消息吞吐量 | 10,000 msg/s | 异步模式下的处理能力 |
| API响应时间 | < 100ms (P99) | 同步发送API延迟 |
| 队列延迟 | < 1s (P95) | 消息入队到开始处理 |
| 重试成功率 | > 95% | 重试后最终成功的比例 |
| 系统可用性 | 99.9% | 月度可用时间百分比 |

### 7.2. 使用示例

#### 基础集成
```go
package main

import (
    "context"
    "github.com/kart-io/notifyhub"
)

func main() {
    // 初始化 NotifyHub
    hub := notifyhub.New(
        notifyhub.WithFeishu(webhookURL, secret),
    )
    defer hub.Stop(context.Background())

    // 发送消息
    message := &notifyhub.Message{
        Channel:    "feishu",
        Recipients: []string{"user@example.com"},
        Subject:    "系统通知",
        Content:    "您的任务已完成",
        Priority:   notifyhub.PriorityNormal,
    }

    // 同步发送
    result, err := hub.SendSync(ctx, message)

    // 异步发送
    err := hub.Send(ctx, message, notifyhub.WithCallback(callback))
}
```

#### 在 Web 应用中使用
```go
// 作为全局单例使用
var notifyHub *notifyhub.Hub

func init() {
    notifyHub = notifyhub.New(
        notifyhub.WithFeishu(os.Getenv("FEISHU_WEBHOOK"), os.Getenv("FEISHU_SECRET")),
        notifyhub.WithQueue(notifyhub.QueueConfig{Workers: 10}),
    )
}

// 在 HTTP Handler 中使用
func HandleOrder(w http.ResponseWriter, r *http.Request) {
    // 业务逻辑...

    // 发送通知
    notifyHub.Send(r.Context(), &notifyhub.Message{
        Channel: "feishu",
        Content: "新订单创建成功",
    })
}
```

#### 批量发送 ✅ 已实现并验证
NotifyHub 支持高性能批量发送功能，支持同步和异步两种模式：

```go
// client/hub.go:262-357 - SendBatch 实现
func (h *Hub) SendBatch(ctx context.Context, messages []*notifiers.Message, options *Options) ([]*notifiers.SendResult, error)

// 同步批量发送
messages := []*notifyhub.Message{
    notifyhub.NewMessage().Title("批量消息1").Body("内容1").Build(),
    notifyhub.NewMessage().Title("批量消息2").Body("内容2").Build(),
    notifyhub.NewMessage().Title("批量消息3").Body("内容3").Build(),
}
results, err := hub.SendBatch(ctx, messages, nil)
if err != nil {
    log.Printf("批量发送失败: %v", err)
} else {
    log.Printf("批量发送完成: %d 个结果", len(results))
}

// 异步批量发送
results, err := hub.SendBatch(ctx, messages, &client.Options{
    Async: true,
    Retry: true,
})

// 批量发送特性:
// ✅ 同步/异步模式支持
// ✅ 部分失败处理 - 单个消息失败不影响其他消息
// ✅ 性能优化 - 批量操作减少系统调用开销
// ✅ 详细错误报告 - 提供每个消息的发送结果
// ✅ 性能追踪 - 记录批量操作总耗时和吞吐量
```

### 7.3. 配置方式

#### 程序化配置（推荐）
```go
// 所有配置通过代码完成，便于类型检查和IDE提示
hub := notifyhub.New(
    // 队列配置
    notifyhub.WithQueue(notifyhub.QueueConfig{
        Type:       "memory",
        Workers:    10,
        BufferSize: 1000,
    }),

    // 重试配置
    notifyhub.WithRetryConfig(notifyhub.RetryConfig{
        MaxRetries:      3,
        InitialInterval: time.Second,
        MaxInterval:     30 * time.Second,
        Multiplier:      2.0,
        Jitter:          true,
    }),

    // 通知器配置
    notifyhub.WithFeishu(webhookURL, secret),
    notifyhub.WithEmail(notifyhub.EmailConfig{
        SMTPHost: "smtp.example.com",
        SMTPPort: 587,
        Username: os.Getenv("SMTP_USER"),
        Password: os.Getenv("SMTP_PASS"),
    }),

    // 日志配置
    notifyhub.WithLogger(logger),
)
```

#### 从配置文件加载（可选）
```go
// 支持从外部配置文件加载
config := notifyhub.LoadConfig("config.yaml")
hub := notifyhub.NewFromConfig(config)
```

### 7.4. 与其他方案对比

| 特性 | NotifyHub | 独立服务方案 | 直接调用SDK |
|------|-----------|------------|------------|
| 部署复杂度 | 低（库引入） | 高（需要独立部署） | 低（直接使用） |
| 统一接口 | ✅ 统一API | ✅ 统一API | ❌ 各自API |
| 可扩展性 | ✅ 插件化设计 | ✅ 可扩展 | ❌ 受限于SDK |
| 性能开销 | 低（进程内） | 高（网络调用） | 低（直接调用） |
| 重试机制 | ✅ 内置 | ✅ 内置 | ❌ 需自行实现 |
| 队列支持 | ✅ 内置 | ✅ 内置 | ❌ 需自行实现 |
| 维护成本 | 低（随应用） | 高（独立运维） | 中（多SDK维护） |

### 7.5. 依赖说明

```go
// go.mod 示例
module your-app

go 1.19

require (
    github.com/kart-io/notifyhub v1.0.0
)

// NotifyHub 核心依赖（自动引入）
// - 无外部强制依赖，核心功能零依赖
// - 可选依赖（按需引入）：
//   - Redis客户端（使用Redis队列时）
//   - Kafka客户端（使用Kafka队列时）
//   - 第三方日志库（使用特定日志适配器时）
```

### 7.6. 相关文档

- [队列系统需求](./queue-system-requirements.md)
- [日志系统文档](../logger/README.md)
- [示例代码](../examples/)
- [API参考](./api-reference.md)
- [快速开始指南](./quick-start.md)

---

📝 **文档版本**: v4.0
🕐 **最后更新**: 2024-01-15
👥 **维护者**: NotifyHub Team
📦 **包地址**: `github.com/kart-io/notifyhub`
