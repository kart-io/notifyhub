# Queue Worker 迁移指南

## 从旧版 Worker 迁移到 WorkerV2

### 迁移背景

原有的 `Worker` 类存在以下问题：
- 依赖过多（直接依赖 notifiers、queueCallbacks、queueCore、retry）
- 职责不单一（既处理队列管理，又处理发送、回调、重试）
- 高耦合度，难以测试和维护

`WorkerV2` 通过职责分离和接口抽象解决了这些问题。

### 迁移步骤

#### 1. 依赖更新

**旧版本使用：**
```go
import (
    "github.com/kart-io/notifyhub/queue/worker"
    "github.com/kart-io/notifyhub/queue/core"
    "github.com/kart-io/notifyhub/queue/retry"
    "github.com/kart-io/notifyhub/queue/callbacks"
)

// 旧版本创建方式
worker := worker.NewWorker(queue, sender, retryPolicy, concurrency)
```

**新版本使用：**
```go
import (
    "github.com/kart-io/notifyhub/queue/worker"
)

// 新版本创建方式 - 使用工厂
factory := worker.NewWorkerFactory()
workerV2 := factory.CreateWorker(queue, sender, config)
```

#### 2. 配置迁移

**旧版本配置：**
```go
retryPolicy := retry.DefaultRetryPolicy()
concurrency := 4

worker := worker.NewWorker(queue, sender, retryPolicy, concurrency)
```

**新版本配置：**
```go
config := &worker.WorkerConfig{
    Concurrency:    4,
    ProcessTimeout: 30 * time.Second,
    RetryPolicy:    retry.DefaultRetryPolicy(),
}

factory := worker.NewWorkerFactory()
workerV2 := factory.CreateWorker(queue, sender, config)
```

#### 3. 启动方式迁移

**旧版本：**
```go
ctx := context.Background()
err := worker.Start(ctx)
// 处理错误...

// 停止
worker.Stop()
```

**新版本：**
```go
ctx := context.Background()
err := workerV2.Start(ctx)
// 处理错误...

// 停止
workerV2.Stop()
```

### 高级迁移场景

#### 自定义组件替换

如果你之前扩展了 Worker 的功能，现在可以通过接口替换特定组件：

```go
// 自定义消息处理器
type CustomProcessor struct{}

func (p *CustomProcessor) ProcessMessage(ctx context.Context, msg *coreMessage.Message, targets []sending.Target) (*worker.ProcessResult, error) {
    // 你的自定义逻辑
    return &worker.ProcessResult{Success: true}, nil
}

// 使用自定义组件
queueManager := worker.NewDefaultQueueManager(queue)
customProcessor := &CustomProcessor{}
retryManager := worker.NewDefaultRetryManager(retryPolicy, queueManager)
callbackManager := worker.NewDefaultCallbackManager(nil)

factory := worker.NewWorkerFactory()
workerV2 := factory.CreateWorkerWithCustomComponents(
    queueManager, customProcessor, retryManager, callbackManager, 4)
```

### 主要 API 对比

| 功能 | 旧版本 | 新版本 |
|------|---------|---------|
| 创建 | `NewWorker(queue, sender, retry, concurrency)` | `factory.CreateWorker(queue, sender, config)` |
| 启动 | `worker.Start(ctx)` | `workerV2.Start(ctx)` |
| 停止 | `worker.Stop()` | `workerV2.Stop()` |
| 自定义 | 修改源码 | 实现接口并注入 |

### 兼容性说明

1. **向后兼容**：旧版本 Worker 仍然可用，但建议迁移到新版本
2. **配置兼容**：大部分配置可以直接迁移
3. **接口变更**：新版本提供了更清晰的接口抽象

### 迁移检查清单

- [ ] 更新导入路径
- [ ] 使用工厂模式创建 Worker
- [ ] 更新配置结构
- [ ] 测试启动和停止功能
- [ ] 验证消息处理逻辑
- [ ] 检查自定义扩展（如有）
- [ ] 运行集成测试

### 常见问题

**Q: 新版本性能如何？**
A: 新版本通过接口抽象可能有微小的性能开销，但通过更好的组件分离实际上可以提高整体性能。

**Q: 可以同时使用新旧版本吗？**
A: 可以，但建议统一迁移到新版本以获得更好的维护性。

**Q: 自定义扩展如何迁移？**
A: 通过实现相应的接口（MessageProcessor、RetryManager等）来替换默认实现。

### 迁移示例

完整的迁移示例：

```go
// 旧版本代码
func oldWorkerSetup() {
    queue := core.NewSimpleQueue(1000)
    sender := &MySender{}
    retryPolicy := retry.DefaultRetryPolicy()

    worker := worker.NewWorker(queue, sender, retryPolicy, 4)

    ctx := context.Background()
    worker.Start(ctx)
    defer worker.Stop()
}

// 新版本代码
func newWorkerSetup() {
    queue := core.NewSimpleQueue(1000)
    sender := &MySender{}

    config := &worker.WorkerConfig{
        Concurrency:    4,
        ProcessTimeout: 30 * time.Second,
        RetryPolicy:    retry.DefaultRetryPolicy(),
    }

    factory := worker.NewWorkerFactory()
    workerV2 := factory.CreateWorker(queue, sender, config)

    ctx := context.Background()
    workerV2.Start(ctx)
    defer workerV2.Stop()
}
```

通过这个迁移指南，你可以顺利地从旧版本迁移到新版本，享受更好的代码结构和维护性。