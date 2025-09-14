# NotifyHub 路由与消息技术设计 (v1.1)

> **对应需求**: `FR9` (消息路由)

本部分定义了系统中最核心的数据结构 `Message`，以及系统如何根据 `Message` 的属性将其分发给正确的 `Notifier`。

## 1. 核心数据结构: `Message`

`Message` 是在 `notifyhub` 系统内部流转的核心“信封”，它是一个包含了所有发送所需信息的结构体。其完整定义请参阅主技术方案文档的“核心数据结构”章节。

### 关键字段详解

*   `Channel` (string): **路由关键字**。此字段是路由匹配的**核心依据**。它的值必须与目标 `Notifier` 的 `Type()` 方法返回值完全一致。例如，要使用邮件通知器，此字段应为 `"email"`。
*   `Recipients` ([]string): 收件人列表。可以是邮箱地址、手机号、Webhook URL 的一部分，或用户 ID 等，具体格式由对应的 `Notifier` 解释。
*   `Payload` (map[string]interface{}): 一个灵活的 map，用于存放除标准字段外的、特定 `Notifier` 可能需要的任何动态数据。例如，飞书的“富文本”消息结构就可以放在这里。
*   `TemplateName` (string): 如果本次发送使用模板，此字段用于指定模板的名称。
*   `TemplateData` (interface{}): 渲染模板所需的数据对象。
*   `Priority` (int): 消息优先级，用于队列调度。
*   `Delay` (time.Duration): 延迟发送时间，用于队列调度。
*   `Retry` (*RetryConfig): 针对此消息的特定重试策略，会覆盖全局默认策略。

## 2. 路由机制 (`Router`)

`Router` 是 `Hub` 的一个内部核心组件，它扮演着“交通警察”的角色，负责将消息精确地导向目标通知器。

### 2.1. 注册 (Registration)

*   `Router` 内部维护一个 `map[string]Notifier` 的映射表。
*   当用户在初始化 `Hub` 时调用 `WithNotifier(myNotifier)`，`Hub` 会调用 `myNotifier.Type()` 获取其类型字符串（如 `"dingtalk"`）。
*   然后，`Hub` 将 `myNotifier` 实例以其类型为键，注册到 `Router` 的映射表中。
*   如果注册了两个相同 `Type` 的 `Notifier`，后注册的会覆盖先注册的。

### 2.2. 路由 (Routing)

*   当 `Hub` 需要发送一条消息时（无论是同步还是异步），它会调用 `Router` 的查找方法。
*   `Router` 从 `Message` 对象中获取 `Channel` 字段的值。
*   使用该 `Channel` 值作为 `key`，在内部映射表中查找对应的 `Notifier` 实例。

### 2.3. 路由失败 (Routing Failure)

*   这是一个明确定义的失败路径。
*   如果在映射表中**找不到**与 `Message.Channel` 匹配的 `Notifier`，`Router` 会立即返回一个错误（例如 `ErrNotifierNotFound`）。
*   `Hub` 在收到这个错误后，会中断当前的发送流程，并将此错误返回给 `SendSync` 的调用者，或通过 `Callback`/`FailureHandler` 在异步流程中报告此错误。
