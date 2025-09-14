# NotifyHub 系统技术方案文档 (v3.2 修订版)

## 1. 概述

### 1.1. 项目目标

NotifyHub 旨在成为一个功能强大、高度可扩展且易于使用的统一通知发送解决方案。它致力于解决在不同业务场景下，向多种渠道（如邮件、即时通讯工具、短信等）发送通知的复杂性。

### 1.2. 设计原则

*   **灵活性与可扩展性**: 系统的核心模块均采用基于接口的插件化设计，支持同步和异步两种发送模式，并允许开发者轻松添加新的通知渠道、队列后端、日志记录器等。
*   **高可靠性**: 通过异步处理、消息重试和死信队列等机制，确保通知消息在各种异常情况下都能被可靠地投递。
*   **出色的可观测性**: 提供详细的结构化日志、丰富的监控指标和分布式追踪能力，使系统在生产环境中的运行状态清晰可见，易于排错和维护。

---

## 2. 核心架构与工作流程

### 2.1. 架构图

```mermaid
graph TD
    subgraph "Application Layer"
        A[用户应用]
    end

    subgraph "NotifyHub Core"
        B[Client/Builder]
        C[Hub (核心协调器)]
        D[Router]
    end

    subgraph "Functional Modules"
        E[Queue System]
        F[Notifiers]
        G[Template Engine]
        H[Observability]
    end

    subgraph "External Services"
        J[Email, Feishu, etc.]
        K[Kafka, Redis, etc.]
    end

    A -- "1. New(With...)" --> B;
    B -- "2. Build Hub" --> C;

    C -- "3a. SendSync(msg)" --> D;
    C -- "3b. Send(msg) / SendBatch(msgs)" --> E;

    D -- "4a. Route to" --> F;
    E -- "4b. Enqueue Job(s)" --> K;
    E -- "5b. Worker Dequeue & Process" --> F

    F -- "Render with" --> G;
    F -- "Send to" --> J;

    C -- "Records" --> H;
    E -- "Records" --> H;
    F -- "Records" --> H;
```

#### 架构组件说明

- **Client/Builder**: 负责系统初始化和配置管理，采用建造者模式提供灵活的配置方式
- **Hub**: 核心协调器，提供统一的发送API，协调各个模块的工作
- **Router**: 消息路由器，根据消息类型和配置将消息分发到对应的通知器
- **Queue System**: 异步消息队列，支持内存队列和第三方队列后端
- **Notifiers**: 通知器插件，负责具体的消息发送到各个渠道
- **Template Engine**: 模板渲染引擎，支持动态内容生成
- **Observability**: 可观测性模块，提供日志、指标和追踪能力

### 2.2. 工作流程详解

> **对应需求**: `FR1`, `FR4`, `FR9`

#### 同步发送流程
1. **初始化**: 用户应用通过 `notifyhub.New(With...)` 创建并配置 `Hub` 实例
2. **同步发送**: 调用 `hub.SendSync(msg)` 发起同步通知
3. **消息路由**: Router根据消息类型选择对应的Notifier
4. **模板渲染**: Template Engine渲染消息内容
5. **立即发送**: Notifier立即发送到目标渠道
6. **返回结果**: 同步返回发送结果给调用者

#### 异步发送流程
1. **异步发送**: 调用 `hub.Send(msg)` 或 `hub.SendBatch(msgs)` 发起异步通知
2. **消息入队**: 消息被推入队列系统
3. **后台处理**: Worker从队列中取出消息进行处理
4. **消息路由**: Router将消息分发到对应的Notifier
5. **模板渲染**: Template Engine渲染消息内容
6. **异步发送**: Notifier异步发送到目标渠道
7. **回调通知**: 通过回调机制通知发送结果

#### 错误处理与重试机制
- **重试机制**: 支持指数退避算法的自动重试
- **死信队列**: 重试失败的消息进入死信队列
- **熔断保护**: 防止级联失败的熔断机制
- **优雅降级**: 在部分渠道失败时的降级策略
- **错误分类**: 区分可重试错误（网络超时、限流）和不可重试错误（认证失败、格式错误）

---

## 3. 模块详解

本系统的核心功能被拆分为多个独立的、可插拔的模块。以下是各模块的职责摘要和技术设计要点。

### 3.1. 核心 Hub 与客户端 (`client/`)

> **对应需求**: `FR1`, `FR4`, `FR8`, `FR10`, `NFR3`

`Hub` 是用户与 `notifyhub` 交互的唯一入口，负责协调内部所有模块。

#### 核心API设计
```go
type Hub interface {
    Send(ctx context.Context, message *Message) error
    SendSync(ctx context.Context, message *Message) (*Result, error)
    SendBatch(ctx context.Context, messages []*Message) error
    Stop(ctx context.Context) error
    Health() HealthStatus
}
```

#### 配置选项
- `WithNotifier()`: 注册通知器
- `WithQueue()`: 配置队列后端
- `WithLogger()`: 设置日志器
- `WithTemplate()`: 配置模板引擎
- `WithRetryConfig()`: 设置重试策略
- `WithMiddleware()`: 添加中间件

### 3.2. 队列系统 (`queue/`)

> **对应需求**: `NFR1`, `NFR2`, `NFR4`, `FR11`, `FR12`

队列系统是实现异步处理的核心。采用工厂模式支持多种队列后端。

#### 队列接口设计
```go
type Queue interface {
    Enqueue(ctx context.Context, job *Job) error
    EnqueueBatch(ctx context.Context, jobs []*Job) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() QueueHealth
}
```

#### 支持的队列类型
- **内存队列**: 基于Go channels的轻量级实现，适合开发和小规模部署
- **Redis Queue**: 基于Redis Streams的持久化队列
- **Kafka Queue**: 基于Kafka的高吞吐量队列
- **NSQ Queue**: 基于NSQ的分布式队列

> **[➡️ 详细设计参见队列系统需求文档](./queue-system-requirements.md)**

### 3.3. 通知器模块 (`notifiers/`)

> **对应需求**: `FR2`, `FR3`, `NFR4`, `NFR7`

`Notifier` 是所有通知渠道的统一抽象，采用开放的插件化架构。

#### 通知器接口
```go
type Notifier interface {
    Send(ctx context.Context, message *Message) (*Result, error)
    Name() string
    HealthCheck(ctx context.Context) error
}
```

#### 内置通知器
- **Email**: SMTP邮件发送
- **Feishu**: 飞书机器人通知
- **Slack**: Slack Webhook通知
- **DingTalk**: 钉钉机器人通知
- **SMS**: 短信发送
- **Webhook**: 通用HTTP Webhook

#### 通知器特性
- **状态管理**: 支持有状态（连接池）和无状态实现
- **速率限制**: 内置令牌桶算法的限流机制
- **健康检查**: 定期检测通知渠道可用性
- **指标采集**: 自动收集发送成功率、延迟等指标

### 3.4. 路由与消息 (`router/`, `types/`)

> **对应需求**: `FR9`

#### 路由策略
- **类型路由**: 根据消息类型路由到对应通知器
- **标签路由**: 基于消息标签的规则路由
- **负载均衡**: 支持轮询、随机、加权等负载均衡算法
- **故障转移**: 主通知器失败时自动切换到备用通知器

#### 消息处理流程
1. **消息验证**: 校验必填字段和格式
2. **路由决策**: 根据路由规则选择目标通知器
3. **消息转换**: 将通用消息格式转换为特定通知器格式
4. **发送执行**: 调用选定的通知器执行发送

### 3.5. 模板引擎 (`template/`)

> **对应需求**: `FR5`, `FR6`

支持动态消息内容生成的模板系统。

#### 模板引擎接口
```go
type TemplateEngine interface {
    Render(ctx context.Context, template string, data interface{}) (string, error)
    RegisterFunction(name string, fn interface{}) error
    LoadTemplate(name string, content string) error
}
```

#### 支持的模板引擎
- **Go Template**: 原生模板引擎，功能强大
- **Mustache**: 无逻辑模板，多语言兼容
- **Handlebars**: 支持助手函数的模板引擎

#### 模板特性
- **缓存机制**: 模板预编译和缓存
- **国际化**: 支持多语言消息模板
- **安全防护**: XSS防护和输入验证
- **变量注入**: 支持系统变量和用户自定义变量

### 3.6. 日志模块 (`logger/`)

> **对应需求**: `FR13`, `NFR5`

为整个系统提供统一、可插拔的日志记录能力。

#### 日志适配器
- **默认适配器**: 带彩色输出的控制台日志
- **文件适配器**: 支持日志轮转的文件输出
- **第三方适配器**: 支持Logrus、Zap、企业日志库等
- **自定义适配器**: 灵活的自定义日志框架

#### 日志特性
- **结构化日志**: 统一的键值对格式
- **性能追踪**: 自动记录操作耗时
- **分布式追踪**: 集成TraceID支持
- **敏感信息脱敏**: 自动脱敏敏感数据

---

## 4. 核心数据结构

### 4.1. Message 消息结构

```go
// Message 是系统中流转的核心数据对象
type Message struct {
    // 基础信息
    ID          string            `json:"id"`                    // 消息唯一标识
    Type        string            `json:"type"`                  // 消息类型: email, feishu, slack等
    Recipients  []string          `json:"recipients"`            // 收件人列表

    // 消息内容
    Subject     string            `json:"subject,omitempty"`     // 消息主题（邮件等需要）
    Content     string            `json:"content"`               // 消息正文
    Template    string            `json:"template,omitempty"`    // 模板名称
    Data        map[string]interface{} `json:"data,omitempty"`   // 模板渲染数据
    Attachments []Attachment      `json:"attachments,omitempty"` // 附件列表

    // 调度信息
    Priority    Priority          `json:"priority"`              // 消息优先级
    DelayUntil  *time.Time       `json:"delay_until,omitempty"` // 延迟发送时间
    ExpireAt    *time.Time       `json:"expire_at,omitempty"`   // 消息过期时间

    // 扩展信息
    Metadata    map[string]string `json:"metadata,omitempty"`    // 元数据
    TraceID     string            `json:"trace_id,omitempty"`    // 分布式追踪ID
    Tags        []string          `json:"tags,omitempty"`        // 消息标签

    // 内部字段
    CreatedAt   time.Time         `json:"created_at"`            // 创建时间
    Attempts    int               `json:"attempts"`              // 重试次数
}

// Priority 消息优先级
type Priority int

const (
    PriorityLow Priority = iota + 1
    PriorityNormal
    PriorityHigh
    PriorityCritical
)

// Attachment 附件结构
type Attachment struct {
    Name        string `json:"name"`         // 文件名
    ContentType string `json:"content_type"` // MIME类型
    Data        []byte `json:"data"`         // 文件内容
    Size        int64  `json:"size"`         // 文件大小
}
```

### 4.2. RetryConfig 重试配置

```go
// RetryConfig 定义了任务失败后的重试策略
type RetryConfig struct {
    MaxAttempts    int           `json:"max_attempts"`     // 最大重试次数
    InitialDelay   time.Duration `json:"initial_delay"`    // 初始延迟时间
    MaxDelay       time.Duration `json:"max_delay"`        // 最大延迟时间
    BackoffFactor  float64       `json:"backoff_factor"`   // 退避因子
    Jitter         bool          `json:"jitter"`           // 是否添加随机抖动
    RetryableErrors []string     `json:"retryable_errors"` // 可重试的错误类型
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
    MaxAttempts:   3,
    InitialDelay:  1 * time.Second,
    MaxDelay:      30 * time.Second,
    BackoffFactor: 2.0,
    Jitter:        true,
    RetryableErrors: []string{
        "timeout", "connection_error", "rate_limit", "server_error",
    },
}
```

### 4.3. Result 结果结构

```go
// Result 包含了异步任务的最终执行结果
type Result struct {
    MessageID    string        `json:"message_id"`    // 消息ID
    NotifierType string        `json:"notifier_type"` // 通知器类型
    Success      bool          `json:"success"`       // 是否成功
    Error        string        `json:"error,omitempty"` // 错误信息
    Attempts     int           `json:"attempts"`      // 重试次数
    Duration     time.Duration `json:"duration"`      // 处理耗时
    Timestamp    time.Time     `json:"timestamp"`     // 完成时间

    // 扩展信息
    ProviderID   string            `json:"provider_id,omitempty"`   // 第三方服务返回的ID
    Metadata     map[string]string `json:"metadata,omitempty"`      // 结果元数据
    Metrics      ResultMetrics     `json:"metrics,omitempty"`       // 性能指标
}

// ResultMetrics 结果性能指标
type ResultMetrics struct {
    QueueTime    time.Duration `json:"queue_time"`    // 队列等待时间
    ProcessTime  time.Duration `json:"process_time"`  // 处理时间
    NetworkTime  time.Duration `json:"network_time"`  // 网络传输时间
    PayloadSize  int64         `json:"payload_size"`  // 负载大小
}
```

### 4.4. Callback 回调接口

```go
// Callback 是单条消息完成回调的接口
type Callback interface {
    // OnSuccess 消息发送成功时调用
    OnSuccess(result *Result)

    // OnFailure 消息发送失败时调用
    OnFailure(result *Result)

    // OnRetry 消息重试时调用
    OnRetry(result *Result)
}

// BatchCallback 批量消息回调接口
type BatchCallback interface {
    // OnBatchComplete 批量消息处理完成时调用
    OnBatchComplete(results []*Result)

    // OnPartialSuccess 部分消息成功时调用
    OnPartialSuccess(successful, failed []*Result)
}

// CallbackFunc 函数式回调适配器
type CallbackFunc func(*Result)

func (f CallbackFunc) OnSuccess(result *Result) {
    if result.Success {
        f(result)
    }
}

func (f CallbackFunc) OnFailure(result *Result) {
    if !result.Success {
        f(result)
    }
}

func (f CallbackFunc) OnRetry(result *Result) {
    // 默认不处理重试事件
}
```

### 4.5. Job 任务结构

```go
// Job 队列中的任务单元
type Job struct {
    ID          string           `json:"id"`           // 任务ID
    Message     *Message         `json:"message"`      // 关联的消息
    RetryConfig *RetryConfig     `json:"retry_config"` // 重试配置
    Callback    Callback         `json:"-"`            // 回调函数（不序列化）
    Context     context.Context  `json:"-"`            // 上下文（不序列化）

    // 任务状态
    Status      JobStatus        `json:"status"`       // 任务状态
    CreatedAt   time.Time        `json:"created_at"`   // 创建时间
    UpdatedAt   time.Time        `json:"updated_at"`   // 更新时间
    ScheduledAt time.Time        `json:"scheduled_at"` // 计划执行时间

    // 执行信息
    Attempts    int              `json:"attempts"`     // 已尝试次数
    LastError   string           `json:"last_error,omitempty"` // 最后一次错误
    NextRetryAt *time.Time       `json:"next_retry_at,omitempty"` // 下次重试时间
}

// JobStatus 任务状态
type JobStatus string

const (
    JobStatusPending    JobStatus = "pending"    // 等待中
    JobStatusProcessing JobStatus = "processing" // 处理中
    JobStatusCompleted  JobStatus = "completed"  // 已完成
    JobStatusFailed     JobStatus = "failed"     // 已失败
    JobStatusDeadLetter JobStatus = "dead_letter" // 死信
)
```

---

## 5. 安全性考虑

> **对应需求**: `NFR6` (安全性)

### 5.1. 凭证管理
- **环境变量**: 敏感配置通过环境变量注入
- **配置文件加密**: 支持配置文件的AES加密
- **密钥轮转**: 支持API密钥的定期轮转
- **权限隔离**: 不同通知器使用独立的凭证

### 5.2. 数据安全
- **传输加密**: 所有外部通信使用TLS/HTTPS
- **日志脱敏**: 自动脱敏敏感信息（邮箱、手机号等）
- **数据验证**: 严格的输入验证和输出编码
- **访问控制**: 基于RBAC的细粒度权限控制

### 5.3. 安全审计
- **审计日志**: 记录所有关键操作和访问
- **异常检测**: 检测异常访问模式和行为
- **合规支持**: 支持SOC 2、GDPR等合规要求
- **安全扫描**: 集成安全扫描工具到CI/CD流程

---

## 6. 可观测性

> **对应需求**: `NFR5` (可观测性)

### 6.1. 日志系统
- **结构化日志**: 统一的JSON格式日志
- **日志级别**: Debug、Info、Warn、Error四级日志
- **上下文传递**: 通过Context传递TraceID和业务上下文
- **日志采集**: 支持ELK、Loki等日志采集系统

### 6.2. 指标监控
- **业务指标**:
  - 消息发送总数、成功率、失败率
  - 各通知器的发送量和成功率
  - 队列长度和处理延迟
  - 模板渲染耗时和成功率

- **技术指标**:
  - HTTP请求延迟分布
  - 内存和CPU使用率
  - Goroutine数量
  - GC停顿时间

### 6.3. 分布式追踪
- **OpenTelemetry集成**: 标准化的追踪实现
- **Span划分**: 细粒度的操作追踪
- **链路分析**: 端到端的请求链路可视化
- **性能瓶颈识别**: 自动识别慢查询和瓶颈点

### 6.4. 告警系统
- **阈值告警**: 基于指标阈值的自动告警
- **趋势告警**: 基于指标趋势变化的告警
- **异常检测**: 基于机器学习的异常检测
- **告警聚合**: 智能的告警聚合和降噪

---

## 7. 测试策略

### 7.1. 单元测试
- **测试覆盖率**: 目标90%以上的代码覆盖率
- **Mock框架**: 使用gomock进行依赖mock
- **边界测试**: 重点测试边界条件和异常情况
- **性能测试**: 关键路径的基准测试

### 7.2. 集成测试
- **Docker环境**: 使用Docker容器模拟外部依赖
- **端到端测试**: 完整的消息发送流程测试
- **故障注入**: 模拟网络故障、服务不可用等场景
- **数据一致性**: 验证异步处理的数据一致性

### 7.3. 性能测试
- **压力测试**: 高并发场景下的性能表现
- **负载测试**: 持续负载下的稳定性验证
- **容量测试**: 确定系统的容量上限
- **内存泄漏**: 长时间运行的内存泄漏检测

### 7.4. 持续集成
- **自动化测试**: 代码提交自动触发测试
- **静态分析**: golint、go vet、staticcheck
- **安全扫描**: gosec安全漏洞扫描
- **依赖检查**: 第三方依赖的安全性检查

---

## 8. 部署与运维

### 8.1. 部署模式
- **单机部署**: 适合小规模应用的简单部署
- **集群部署**: 高可用的多节点集群部署
- **容器化**: Docker镜像和Kubernetes部署
- **云原生**: 支持主流云平台的托管服务

### 8.2. 配置管理
- **配置分层**: 支持开发、测试、生产环境配置
- **动态配置**: 支持配置的热更新
- **配置验证**: 启动时的配置有效性验证
- **配置备份**: 配置变更的版本控制和回滚

### 8.3. 监控运维
- **健康检查**: HTTP健康检查端点
- **优雅关闭**: 处理SIGTERM信号的优雅关闭
- **资源限制**: 内存和CPU使用限制
- **自动恢复**: 服务异常时的自动重启

---

## 9. 发展路线图 (Roadmap)

### Phase 1: 核心功能强化 (3个月)
*   ✅ **任务1**: 实现 `Notifier` 和 `Queue` 的核心接口及默认的 Worker-Pool 实现
*   ✅ **任务2**: 实现 `Send/SendSync` API 及完整的错误处理/回调/重试机制
*   [ ] **任务3**: 引入 `viper` 实现灵活的配置加载
*   [ ] **任务4**: 建立基础的单元测试和集成测试框架，并配置 CI

### Phase 2: 生态与高级功能 (6个月)
*   [ ] **任务1**: 新增 `Slack`、`DingTalk`、`SMS` 等官方 `Notifier`
*   [ ] **任务2**: 实现 `Redis Streams` 队列适配器
*   [ ] **任务3**: 实现消息优先级和延迟消息功能
*   [ ] **任务4**: 在核心 `Notifier` 中集成速率限制功能

### Phase 3: 生产级强化 (9个月)
*   [ ] **任务1**: 实现 `Kafka` 队列适配器
*   [ ] **任务2**: 全面推行结构化日志，并集成 OpenTelemetry 实现分布式追踪
*   [ ] **任务3**: 完善用户文档和开发者文档
*   [ ] **任务4**: 提供完整的 `docker-compose` 示例项目

### Phase 4: 企业级特性 (12个月)
*   [ ] **任务1**: Web管理界面和监控大盘
*   [ ] **任务2**: 消息模板管理和A/B测试
*   [ ] **任务3**: 多租户支持和资源隔离
*   [ ] **任务4**: 高级路由策略和智能故障转移

---

## 10. 附录

### 10.1. 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 消息吞吐量 | 10,000 msg/s | 异步模式下的消息处理能力 |
| API响应时间 | < 100ms (P99) | 同步发送API的响应时间 |
| 队列延迟 | < 1s (P95) | 消息入队到处理的延迟 |
| 系统可用性 | 99.9% | 系统整体可用性目标 |
| 错误率 | < 0.1% | 系统内部错误率（不包括第三方服务错误） |

### 10.2. 兼容性说明

- **Go版本**: 支持Go 1.19及以上版本
- **操作系统**: Linux、macOS、Windows
- **架构**: amd64、arm64
- **第三方依赖**: 最小化外部依赖，核心功能零依赖

### 10.3. 许可证

本项目采用MIT许可证，支持商业和开源使用。

---

📝 **文档版本**: v3.2
🕐 **最后更新**: 2024-01-15
👥 **维护者**: NotifyHub Team