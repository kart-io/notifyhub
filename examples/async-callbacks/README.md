# NotifyHub 异步回调功能演示

这个示例演示了 NotifyHub 的异步回调功能，包括成功回调、错误回调和进度回调。

## 支持的回调类型

### 1. 完成回调 (OnComplete)
当消息发送成功时触发：

```go
handle.OnComplete(func(receipt *receipt.Receipt) {
    log.Printf("消息发送成功! MessageID: %s", receipt.MessageID)
    log.Printf("成功发送到 %d 个目标", len(receipt.Results))
})
```

### 2. 错误回调 (OnError)
当消息发送失败时触发：

```go
handle.OnError(func(message *message.Message, err error) {
    log.Printf("消息发送失败! 消息ID: %s", message.ID)
    log.Printf("错误信息: %v", err)
})
```

### 3. 进度回调 (OnProgress)
用于批量操作的进度监控：

```go
handle.OnProgress(func(completed, total int) {
    progress := float64(completed) / float64(total) * 100
    log.Printf("进度: %d/%d (%.1f%%)", completed, total, progress)
})
```

## 使用方式

### 单条消息异步发送
```go
handle, err := client.SendAsync(ctx, msg)
if err != nil {
    return err
}

// 设置回调
handle.OnComplete(func(receipt *receipt.Receipt) {
    // 处理成功结果
}).OnError(func(message *message.Message, err error) {
    // 处理错误
})

// 等待完成
receipt, err := handle.Wait(ctx)
```

### 批量消息异步发送
```go
batchHandle, err := client.SendAsyncBatch(ctx, messages)
if err != nil {
    return err
}

// 监听进度
go func() {
    for progress := range batchHandle.Progress() {
        log.Printf("批量进度: %d/%d (%.1f%%)",
            progress.Completed, progress.Total, progress.Progress*100)
    }
}()

// 监听结果
go func() {
    for result := range batchHandle.Results() {
        if result.Error != nil {
            log.Printf("失败: %v", result.Error)
        } else {
            log.Printf("成功")
        }
    }
}()

// 等待全部完成
receipts, err := batchHandle.Wait(ctx)
```

## 回调优势

1. **实时反馈**: 无需主动轮询，即时获得发送状态
2. **错误处理**: 可以立即处理失败的消息
3. **进度监控**: 批量操作可以实时查看进度
4. **异步处理**: 不阻塞主线程，提高性能
5. **链式调用**: 支持流畅的链式设置多个回调

## 运行示例

```bash
cd examples/async-callbacks
go run main.go
```

注意：运行前请修改配置中的 Webhook URL。