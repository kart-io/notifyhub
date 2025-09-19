# Queue Worker 重构说明

## 重构目标

解决原有 `queueWorker` 包依赖过多、职责不单一的问题，实现：
- **职责分离**：将复杂的功能拆分为独立的组件
- **降低耦合**：通过接口依赖而非具体实现
- **提高可测试性**：每个组件都可以独立测试
- **增强可重用性**：组件可以在不同场景下组合使用

## 重构后的架构

### 核心接口层 (`interfaces.go`)

定义了六个核心接口，实现职责分离：

- **MessageProcessor** - 专注消息发送逻辑
- **RetryManager** - 专注重试策略管理
- **CallbackManager** - 专注回调执行
- **QueueManager** - 专注队列操作抽象
- **WorkerCoordinator** - 协调组件交互

### 实现层

1. **DefaultMessageProcessor** (`processor.go`) - 消息发送处理
2. **DefaultRetryManager** (`retry_manager.go`) - 重试策略管理
3. **DefaultCallbackManager** (`callback_manager.go`) - 回调执行
4. **DefaultQueueManager** (`queue_manager.go`) - 队列操作抽象
5. **DefaultWorkerCoordinator** (`coordinator.go`) - 组件协调器
6. **WorkerV2** (`worker_v2.go`) - 工作池管理

### 工厂层 (`factory.go`)

提供便捷的组件组装方法，封装复杂的对象创建过程。

## 依赖关系对比

### 重构前 - 高耦合
```
queueWorker 直接依赖:
├── notifiers (具体实现)
├── queueCallbacks (具体实现)
├── queueCore (具体实现)
└── retry (具体实现)
```

### 重构后 - 低耦合
```
WorkerV2 只依赖:
└── WorkerCoordinator (接口)

各组件通过接口解耦:
├── MessageProcessor (接口)
├── RetryManager (接口)
├── CallbackManager (接口)
└── QueueManager (接口)
```

## 主要改进

1. **单一职责原则** - 每个组件只负责一个职责
2. **依赖倒置原则** - 依赖接口而非具体实现
3. **开闭原则** - 易于扩展新功能而无需修改现有代码
4. **接口隔离原则** - 接口职责明确，避免冗余依赖

## 使用示例

```go
// 基本使用
factory := NewWorkerFactory()
worker := factory.CreateWorker(queue, sender, DefaultWorkerConfig())

// 自定义组件
worker := factory.CreateWorkerWithCustomComponents(
    queueManager, customProcessor, retryManager, callbackManager, 4)

// 最小化设置
worker := factory.CreateMinimalWorker(queue, sender, 2)
```

这种重构显著降低了复杂性，提高了代码的可维护性和可测试性。