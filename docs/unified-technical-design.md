# NotifyHub 统一技术方案文档 (最终版)

## 1. 概述

### 1.1. 项目目标

NotifyHub 旨在成为一个功能强大、高度可扩展且易于使用的统一通知发送解决方案。它致力于解决在不同业务场景下，向多种渠道（如邮件、即时通讯工具、短信等）发送通知的复杂性。

### 1.2. 设计原则

* **灵活性与可扩展性**: 系统的核心模块均采用基于接口的插件化设计，支持同步和异步两种发送模式，并允许开发者轻松添加新的通知渠道、队列后端、日志记录器等。
* **高可靠性**: 通过异步处理、消息重试和死信队列等机制，确保通知消息在各种异常情况下都能被可靠地投递。
* **出色的可观测性**: 提供详细的结构化日志、丰富的监控指标和分布式追踪能力，使系统在生产环境中的运行状态清晰可见，易于排错和维护。

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

### 2.2. 工作流程详解

> **对应需求**: `FR1`, `FR4`, `FR9`

1. **初始化**: 用户应用通过**建造者模式** (`notifyhub.New(...)`) 创建并配置一个 `Hub` 实例。在此阶段，用户可以注册自定义的 `Notifier`、配置 `Queue` 后端、设置全局 `FailureHandler` 等。

2. **选择发送模式**: 用户根据业务需求，选择调用 `Hub` 的不同方法：
    * **同步发送**: 调用 `hub.SendSync(message)`。
    * **异步发送**: 调用 `hub.Send(message)` 或 `hub.SendBatch(messages)`。

3. **执行路径**:
    * **同步路径**:
        1. `Hub` 接收到请求，调用内部的 `Router`。
        2. `Router` 根据 `Message` 的属性（如 `Channel` 字段）匹配到对应的 `Notifier`。
        3. `Hub` 调用 `Template Engine` 渲染消息内容（如果需要）。
        4. `Hub` 直接调用 `Notifier` 的 `Send` 方法，并**阻塞等待**其返回结果（`error`）。
        5. `SendSync` 方法将最终结果返回给调用方。

    * **异步路径**:
        1. `Hub` 接收到请求，将一个或多个 `Message` 封装成 `Job`(s)。
        2. `Hub` 调用 `Queue` 系统的 `Enqueue` 或 `EnqueueBatch` 方法，将 `Job`(s) 推入队列，然后**立即返回**。
        3. 后台的 `Worker` 从队列中获取 `Job`。
        4. `Worker` 执行与同步路径类似的**路由、渲染、发送**逻辑。
        5. `Worker` 在任务完成后，根据最终结果执行相应的**回调逻辑**。

---

## 3. 模块详解

### 3.1. 核心 Hub 与客户端 (`client/`)

> **对应需求**: `FR1`, `FR4`, `FR8`, `FR10`, `NFR3`

`Hub` 是用户与 `notifyhub` 交互的唯一入口，负责协调内部所有模块。

* **核心 API**:
  * `New(...)`: 使用建造者模式创建 `Hub`。
  * `Send(ctx, msg, opts...)`: 异步发送单条消息。
  * `SendBatch(ctx, msgs, opts...)`: 异步批量发送多条消息。
  * `SendSync(ctx, msg)`: 同步发送单条消息。
  * `Stop(ctx)`: 优雅地关闭 `Hub`。

* **核心配置选项 (`Options`)**:
  * `WithNotifier(Notifier)`: 注册自定义通知器。
  * `WithQueue(Queue)`: 配置队列系统实现。
  * `WithWorkerCount(int)`: 配置内置队列的 Worker 数量。
  * `WithQueueSize(int)`: 配置内置队列的缓冲区大小。
  * `WithFailureHandler(FailureHandler)`: 注册全局异步任务失败处理器。
  * `WithTemplateProvider(TemplateProvider)`: 配置模板提供者。

#### 3.1.1. 初始化示例

```go
func main() {
    // 创建自定义组件
    myNotifier := custom.NewMyNotifier(...)
    myQueue := custom.NewMyQueue(...)

    // 使用建造者模式和选项函数初始化 Hub
    hub, err := notifyhub.New(
        notifyhub.WithNotifier(myNotifier),
        notifyhub.WithQueue(myQueue), // 可选，默认使用内置队列
        notifyhub.WithWorkerCount(20), // 配置内置队列的 worker 数
        notifyhub.WithFailureHandler(func(msg *notifyhub.Message, err error) {
            log.Printf("Message finally failed: %s, err: %v", msg.ID, err)
        }),
    )
    if err != nil {
        panic(err)
    }

    // 优雅关闭
    defer hub.Stop(context.Background())

    // ... 使用 hub.Send() 或 hub.SendSync()
}
```

### 3.2. 队列系统 (`queue/`)

> **对应需求**: `NFR1`, `NFR2`, `NFR4`, `FR11`, `FR12`

队列系统是 `notifyhub` 实现异步处理的核心，被设计为可插拔模块。

#### 3.2.1. 内置 Worker-Pool 队列 (默认)

* **核心组件**: `Job`, `JobQueue`, `Worker`。
* **配置**: 可通过 `WithWorkerCount` 和 `WithQueueSize` 选项进行配置。
* **优雅停机**: `hub.Stop()` 会关闭 `JobQueue` 并使用 `sync.WaitGroup` 等待所有 `Worker` 完成当前任务。

#### 3.2.2. 消息优先级处理 (设计目标)

> **对应需求**: `FR11` (消息优先级)

* **目标**: 支持高优先级的消息（如验证码）能被优先处理。
* **实现思路**: 内置队列可采用多级 channel 方案（为不同优先级各设一个 channel），`Worker` 优先从高优先级 channel 中获取任务。第三方队列则利用其自身的优先级队列特性。

#### 3.2.3. 延迟消息处理 (设计目标)

> **对应需求**: `FR12` (延迟/计划消息)

* **目标**: 支持消息在指定时间后才被投递。
* **实现思路**: 内置队列可引入一个基于**最小堆（Min-Heap）**的调度器，管理所有延迟消息，在消息到期后才将其推入 `JobQueue`。第三方队列则利用其自身的延迟消息功能。

#### 3.2.4. 第三方消息队列适配器 (扩展)

* **目标**: 支持将任务持久化到专业的第三方消息中间件。
* **实施路径**: 定义 `Queue` 接口，并提供 `Redis Streams`、`Kafka` 等适配器。

### 3.3. 通知器模块 (`notifiers/`)

> **对应需求**: `FR2`, `FR3`, `NFR4`, `NFR7`

`Notifier` 是所有通知渠道的统一抽象，采用开放的插件化架构。

#### 3.3.1. `Notifier` 接口定义

```go
// Notifier 是所有通知渠道都必须实现的接口。
type Notifier interface {
    Send(ctx context.Context, payload map[string]interface{}) error
    Type() string
    Shutdown(ctx context.Context) error
}
```

#### 3.3.2. 实现模式

* **无状态通知器 (Stateless)**: 适用于独立的 HTTP 请求（如 Webhook）。`Shutdown` 方法返回 `nil`。
* **有状态通知器 (Stateful)**: 适用于需要维持长连接的渠道（如 gRPC）。在**构造函数**中建立连接，在 `Send` 中**复用**，在 `Shutdown` 中**关闭**。

#### 3.3.3. 速率限制 (设计目标)

> **对应需求**: `NFR7` (速率限制)

* **目标**: 防止因请求速率过快而超出第三方 API 的配额限制。
* **实现思路**: 在 `Notifier` 的实现内部集成速率限制逻辑（如令牌桶算法）。`Notifier` 的构造函数应接受速率参数，`Send` 方法在执行前需先获取令牌。

### 3.4. 路由与消息 (`config/`, `types.go`)

> **对应需求**: `FR9` (消息路由)

* **消息 (`Message`)**: 系统的核心数据对象，包含 `Channel` (用于路由), `Payload`, `TemplateName`, `TemplateData`, `Priority`, `Delay`, `Retry` 等字段。
* **路由 (`Router`)**: `Hub` 的内部组件，负责根据 `Message.Channel` 字段，将消息匹配到已注册的 `Notifier`。
* **路由失败**: 当 `Message.Channel` 无法匹配到任何已注册的 `Notifier` 时，发送操作（`Send` 或 `SendSync`）将立即失败并返回一个明确的错误。

### 3.5. 模板引擎 (`template/`)

> **对应需求**: `FR5` (消息模板), `FR6` (动态模板源)

该模块负责将模板和动态数据渲染成最终的消息内容，设计上将**加载** (`TemplateProvider`) 与**渲染** (`TemplateEngine`) 分离。

---

## 4. 错误处理与可靠性

> **对应需求**: `FR7` (异步重试), `FR8` (失败处理机制), `NFR2` (可靠性)

### 4.1. 同步模式 (`SendSync`)

* **机制**: 直接返回 `error` 对象，由调用方负责处理。

### 4.2. 异步模式 (`Send` / `SendBatch`)

* **机制**: 通过回调和全局处理器。
* **单消息回调 (`Callback`)**: 在任务**最终成功或失败**后被调用一次。
* **全局失败处理器 (`FailureHandler`)**: 在任何任务**最终失败**（耗尽重试）后被调用。

### 4.3. 重试策略配置与应用

* **机制**: `Hub` 拥有全局默认重试策略，同时允许在单条 `Message` 上附加 `RetryConfig` 来进行覆盖。
* **退避策略**: 为了避免在下游服务故障时发起“惊群”攻击，重试机制必须采用**“指数退避+随机抖动” (Exponential Backoff with Jitter)** 策略。

---

## 5. 核心数据结构

本章节统一列出系统中的核心公共结构体。

```go
// Message 是系统中流转的核心数据对象
type Message struct {
    ID           string                 `json:"id"`
    Channel      string                 `json:"channel"`
    Priority     int                    `json:"priority"`     // 消息优先级
    Delay        time.Duration          `json:"delay"`        // 延迟发送时间
    Payload      map[string]interface{} `json:"payload"`
    TemplateName string                 `json:"templateName"`
    TemplateData interface{}            `json:"templateData"`
    Retry        *RetryConfig           `json:"retry"`
}

// RetryConfig 定义了任务失败后的重试策略
type RetryConfig struct {
    MaxRetries      int           // 最大重试次数
    InitialInterval time.Duration // 初始重试间隔
    Multiplier      float64       // 退避倍数
}

// Result 包含了异步任务的最终执行结果
type Result struct {
    Success   bool      // 最终结果是否成功
    Error     error     // 如果失败，此字段包含错误信息
    Attempts  int       // 总共尝试的次数
}

// Callback 是单条消息完成回调的接口
type Callback interface {
    OnComplete(result *Result)
}
```

---

## 6. 安全性考虑

> **对应需求**: `NFR6` (安全性)

* **凭证管理**: 所有 `Notifier` 的敏感凭证（API Key/Secret）必须通过安全的配置方式（如环境变量、Secrets Manager）在初始化时注入，严禁硬编码和记入日志。
* **日志审查**: 确保在日志中脱敏处理可能包含在 `Payload` 中的个人身份信息（PII）。

---

## 7. 可观测性

> **对应需求**: `NFR5` (可观测性)

* **结构化日志**: 全面采用结构化日志，并在日志中添加 `message_id`, `trace_id` 等关键上下文信息。
* **分布式追踪**: 集成 OpenTelemetry，实现核心链路的分布式追踪。
* **监控指标**: 使用 Prometheus 客户端库，提供丰富的性能和业务指标。

---

## 8. 测试策略

为确保项目质量，必须建立全面的测试策略，包括**单元测试**、**集成测试**和**持续集成 (CI)**。

---

## 9. 发展路线图 (Roadmap)

### Phase 1: 核心功能强化

* [ ] **任务1**: 实现 `Notifier` 和 `Queue` 的核心接口及默认的 Worker-Pool 实现。
* [ ] **任务2**: 实现 `Send/SendSync` API 及完整的错误处理/回调/重试机制。
* [ ] **任务3**: 引入 `viper` 实现灵活的配置加载。
* [ ] **任务4**: 建立基础的单元测试和集成测试框架，并配置 CI。

### Phase 2: 生态与高级功能

* [ ] **任务1**: 新增 `Slack`、`DingTalk`、`SMS` 等官方 `Notifier`。
* [ ] **任务2**: 实现 `Redis Streams` 队列适配器。
* [ ] **任务3**: 实现消息优先级和延迟消息功能。
* [ ] **任务4**: 在核心 `Notifier` 中集成速率限制功能。

### Phase 3: 生产级强化

* [ ] **任务1**: 实现 `Kafka` 队列适配器。
* [ ] **任务2**: 全面推行结构化日志，并集成 OpenTelemetry 实现分布式追踪。
* [ ] **任务3**: 完善用户文档和开发者文档。
