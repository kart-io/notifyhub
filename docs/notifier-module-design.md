# NotifyHub 通知器模块技术设计 (v1.1)

> **对应需求**: `FR2`, `FR3`, `NFR4`, `NFR7`

`Notifier` 是所有通知渠道的统一抽象，采用开放的插件化架构，是 `notifyhub` 实现多渠道通知能力的核心。

## 1. `Notifier` 接口定义

系统定义了所有通知器都必须遵守的接口，它包含了生命周期管理，以支持有状态的连接。

```go
package notifiers

import "context"

// Notifier 是所有通知渠道都必须实现的接口。
type Notifier interface {
    // Send 方法负责执行实际的发送逻辑。
    // payload 是一个 map，包含了发送所需的所有动态数据，例如收件人、内容等。
    Send(ctx context.Context, payload map[string]interface{}) error

    // Type 方法返回该通知器的唯一名称标识，用于路由。
    Type() string

    // Shutdown 用于在系统关闭时，优雅地释放 Notifier 可能持有的资源（如长连接）。
    // 对于无状态的 Notifier，直接返回 nil 即可。
    Shutdown(ctx context.Context) error
}
```

## 2. 实现模式

### 2.1. 无状态通知器 (Stateless Notifier)

*   **场景**: 适用于每次发送都是独立 HTTP 请求的渠道（如 Webhook、钉钉、飞书）。这类通知器不需要在多次 `Send` 调用之间维持状态。
*   **实现要点**:
    1.  在结构体中通常持有一个可复用的 `*http.Client` 或 `*resty.Client`。
    2.  `Send` 方法根据 `payload` 构建请求体并发起 HTTP 请求。
    3.  `Shutdown` 方法因为没有需要释放的连接资源，直接返回 `nil`。

### 2.2. 有状态通知器 (Stateful Notifier)

*   **场景**: 适用于需要维持长连接或有状态客户端的渠道（如 gRPC、自定义 IM 协议）。
*   **实现要点**: 采用“构造时连接，发送时复用，关闭时释放”的模式。
    1.  在**构造函数** (`New...`) 中建立长连接（如 `grpc.Dial`）并创建可复用的客户端。
    2.  将连接对象和客户端实例保存在结构体的字段中。
    3.  在 `Send` 方法中**复用**已有的客户端实例来发送数据。
    4.  在 `Shutdown` 方法中**关闭**连接，释放资源。

## 3. 速率限制 (Rate Limiting)

> **对应需求**: `NFR7` (速率限制)

*   **目标**: 防止因请求速率过快而超出第三方 API 的配额限制。
*   **实现思路**: 速率限制逻辑应在**具体的 `Notifier` 实现内部**完成。
    *   在 `Notifier` 结构体中增加一个 `*rate.Limiter` 字段（来自 `golang.org/x/time/rate` 包）。
    *   `Notifier` 的构造函数应接受速率参数（如每秒请求数 `limit` 和并发数 `burst`），并用它们来初始化 `rate.Limiter`。
    *   `Send` 方法在执行外部调用前，需先调用 `limiter.Wait(ctx)` 来确保不超过速率限制。

    ```go
    // 在 Send 方法内部
    func (n *MyNotifier) Send(ctx context.Context, payload map[string]interface{}) error {
        if err := n.limiter.Wait(ctx); err != nil {
            return err // context aancelled
        }
        // ... 执行实际的发送逻辑 ...
    }
    ```

## 4. 注册与使用

任何实现了 `Notifier` 接口的结构体，都可以通过 `notifyhub.New(notifyhub.WithNotifier(myNotifier))` 的方式注册到 `Hub` 中，并通过 `Message.Channel` 字段进行路由调用。
