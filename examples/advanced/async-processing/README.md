# Async Processing Example

This example demonstrates how to use NotifyHub's async message processing capabilities with queue support.

## Features Demonstrated

1. **Async Hub Creation**: Setting up an async hub with memory or Redis queue
2. **Message Queueing**: Enqueueing messages for background processing
3. **Priority Handling**: High-priority messages processed first
4. **Scheduled Messages**: Delay message delivery to specific times
5. **Retry Policies**: Automatic retry with exponential backoff
6. **Dead Letter Queue**: Failed message isolation
7. **Worker Pools**: Concurrent message processing
8. **Queue Monitoring**: Real-time statistics and health checks
9. **Graceful Shutdown**: Proper cleanup on termination

## Running the Example

### Prerequisites

1. Set environment variables for notification platforms:
```bash
# Feishu configuration
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
export FEISHU_SECRET="your-secret"

# Email configuration
export SMTP_HOST="smtp.gmail.com"
export SMTP_USERNAME="your-email@gmail.com"
export SMTP_PASSWORD="your-password"
export SMTP_FROM="notifications@example.com"

# Optional: Redis configuration for persistent queue
export REDIS_ADDR="localhost:6379"
```

2. Run the example:
```bash
go run main.go
```

## Queue Configuration Options

### Memory Queue
```go
// In-memory queue with specified capacity and workers
notifyhub.WithMemoryQueue(1000, 4)  // 1000 capacity, 4 workers
```

### Redis Queue
```go
// Redis-backed queue for persistence
notifyhub.WithRedisQueue("localhost:6379", 10000, 8)  // Redis addr, capacity, workers
```

### Retry Policy
```go
// Configure retry with max attempts and initial interval
notifyhub.WithQueueRetry(3, 1*time.Second)  // 3 retries, 1s initial delay
```

## Message Priority Levels

Messages are processed based on priority:
- `PriorityUrgent`: Processed immediately
- `PriorityHigh`: High priority queue
- `PriorityNormal`: Standard processing
- `PriorityLow`: Processed when queue is idle

## Monitoring Queue Health

The async hub provides real-time statistics:
```go
stats := asyncHub.GetQueueStats()
// Returns:
// - queue_size: Current number of messages
// - is_empty: Whether queue is empty
// - processing: Whether workers are active
// - workers: Worker pool statistics
```

## Error Handling

Failed messages are automatically retried based on the retry policy. After maximum retries are exceeded, messages are moved to the dead letter queue for manual intervention.

## Best Practices

1. **Set Appropriate Queue Capacity**: Based on expected message volume
2. **Configure Worker Count**: Balance between throughput and resource usage
3. **Monitor Dead Letter Queue**: Regularly check for failed messages
4. **Use Priority Wisely**: Reserve urgent priority for critical alerts
5. **Implement Graceful Shutdown**: Allow workers to finish processing

## Architecture

```
┌─────────────────┐
│   Application   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Async Hub     │
├─────────────────┤
│  Message Queue  │ ← Memory/Redis Backend
├─────────────────┤
│  Worker Pool    │ ← Auto-scaling Workers
├─────────────────┤
│  Retry Logic    │ ← Exponential Backoff
├─────────────────┤
│  Dead Letter Q  │ ← Failed Messages
└─────────────────┘
         │
         ▼
┌─────────────────┐
│ Notification    │
│ Platforms       │
└─────────────────┘
```

## Performance Considerations

- **Memory Queue**: Fast, but messages lost on restart
- **Redis Queue**: Persistent, slightly higher latency
- **Worker Scaling**: Auto-scales based on queue depth
- **Batch Processing**: Send multiple messages efficiently

## Troubleshooting

1. **Messages not processing**: Check if `ProcessQueuedMessages()` was called
2. **High memory usage**: Reduce queue capacity or use Redis
3. **Slow processing**: Increase worker count
4. **Lost messages**: Use Redis queue for persistence
5. **Failed messages**: Check dead letter queue for errors