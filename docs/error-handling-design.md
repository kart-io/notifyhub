# NotifyHub 错误处理与可靠性技术设计 (v1.1)

> **对应需求**: `FR7` (异步重试), `FR8` (失败处理机制), `NFR2` (可靠性)

一个健壮的通知系统必须拥有完善的错误处理和重试机制，以应对网络波动、下游服务暂时不可用等异常情况。

## 1. 同步模式 (`SendSync`) 的错误处理

*   **机制**: 直接返回 `error` 对象。
*   **详解**: `SendSync` 是一个阻塞操作，调用方在原地等待结果。如果发送过程中的任何一步（路由、渲染、`Notifier` 发送）失败，函数将立即中断并返回一个非 `nil` 的 `error` 对象。错误处理的责任完全由调用方承担。

    ```go
    err := hub.SendSync(ctx, message)
    if err != nil {
        // 调用方立即感知失败，并负责处理
        log.Errorf("Sync send failed: %v", err)
    }
    ```

## 2. 异步模式 (`Send` / `SendBatch`) 的错误处理

*   **机制**: 通过“单消息完成回调”和“全局失败处理器”两种方式通知。

### 2.1. 单消息完成回调 (`Callback`)

*   **用途**: 为特定消息指定精细化的后处理逻辑，**无论成功或最终失败**。
*   **接口定义**:
    ```go
    type Result struct {
        Success   bool
        Error     error
        Attempts  int
    }

    type Callback interface {
        OnComplete(result *Result)
    }
    ```
*   **触发时机**: `OnComplete` 方法只在任务达到**最终状态**后被调用一次，而不是在每次重试后都调用。
*   **使用**: 用户在 `Send` 时通过 `WithCallback` 选项附加回调。在回调实现中，通过检查 `Result.Success` 字段来区分成功与失败。

### 2.2. 全局失败处理器 (`FailureHandler`)

*   **用途**: 统一处理所有**最终失败**（耗尽重试后）的消息，适用于全局日志、监控和告警。
*   **触发时机**: 仅在任务“最终失败”时触发。
*   **使用**: 在 `New` Hub 时通过 `WithFailureHandler` 选项进行全局注册。

## 3. 重试机制 (Retry)

### 3.1. 重试策略配置

*   **机制**: `Hub` 拥有一个可配置的全局默认重试策略。同时，允许在单条 `Message` 上附加一个 `RetryConfig` 对象来覆盖默认策略。
*   **数据结构**:
    ```go
    type RetryConfig struct {
        MaxRetries      int           // 最大重试次数, 0 表示不重试
        InitialInterval time.Duration // 初始重试间隔
        Multiplier      float64       // 退避倍数, > 1.0
    }
    ```

### 3.2. 退避算法

*   **策略**: 为了避免在下游服务故障时发起“惊群”攻击，重试机制必须采用**“指数退避+随机抖动” (Exponential Backoff with Jitter)** 策略。
*   **计算公式**: `下次重试间隔 = InitialInterval * (Multiplier ^ (Attempts - 1)) + RandomJitter`
