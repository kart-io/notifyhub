# Error Handling Patterns

This example demonstrates robust error handling patterns in NotifyHub, showing how to build resilient notification systems that handle failures gracefully.

## What You'll Learn

- Common error types and causes
- Error handling at different stages
- Partial failure management
- Retry strategies for critical messages
- Context-based timeout handling
- Error classification and monitoring

## Error Categories

### Configuration Errors

These occur during hub creation due to invalid configuration:

```go
// This will fail - empty webhook URL
hub, err := notifyhub.NewHub(
    feishu.WithFeishu("", feishu.WithFeishuSecret("secret")),
)
if err != nil {
    log.Fatalf("Configuration error: %v", err)
}
```

**Common causes:**
- Empty or malformed URLs
- Missing required credentials
- Invalid configuration parameters

### Validation Errors

These occur when message or target validation fails:

```go
// Invalid target type
msg := notifyhub.NewMessage("Test").
    ToTarget(notifyhub.NewTarget("invalid-type", "value", "feishu")).
    Build()

receipt, err := hub.Send(ctx, msg)
// Check receipt.Results for specific validation errors
```

**Common causes:**
- Invalid target types
- Malformed email addresses
- Invalid phone number formats

### Network Errors

These occur during message sending due to network issues:

```go
// Very short timeout to simulate network errors
hub, err := notifyhub.NewHub(
    feishu.WithFeishu("https://example.com/webhook",
        feishu.WithFeishuTimeout(1*time.Millisecond),
    ),
)
```

**Common causes:**
- Connection timeouts
- DNS resolution failures
- Network connectivity issues

## Best Practices

### 1. Always Check Errors

Check errors at every stage of the process:

```go
// Hub creation
hub, err := notifyhub.NewHub(/* config */)
if err != nil {
    return fmt.Errorf("failed to create hub: %w", err)
}
defer hub.Close(ctx)

// Message sending
receipt, err := hub.Send(ctx, message)
if err != nil {
    return fmt.Errorf("failed to send message: %w", err)
}
```

### 2. Handle Partial Failures

NotifyHub can succeed on some platforms while failing on others:

```go
receipt, err := hub.Send(ctx, message)
if err != nil {
    // Critical error - all platforms failed
    return fmt.Errorf("all platforms failed: %w", err)
} else if receipt.Failed > 0 {
    // Partial failure - some platforms succeeded
    log.Warnf("Partial failure: %d/%d platforms failed", receipt.Failed, receipt.Total)

    // Log details of failed platforms
    for _, result := range receipt.Results {
        if !result.Success {
            log.Errorf("Platform %s failed: %s", result.Platform, result.Error)
        }
    }
}
```

### 3. Implement Retry Logic

For critical messages, implement exponential backoff retry:

```go
func sendWithRetry(hub notifyhub.Hub, ctx context.Context, msg *notifyhub.Message, maxRetries int) error {
    for attempt := 1; attempt <= maxRetries; attempt++ {
        receipt, err := hub.Send(ctx, msg)

        if err != nil {
            if attempt == maxRetries {
                return fmt.Errorf("max retries exceeded: %w", err)
            }

            backoff := time.Duration(attempt) * time.Second
            time.Sleep(backoff)
            continue
        }

        if receipt.Failed == 0 {
            return nil // Success
        }

        // Partial failure - retry failed platforms only
        if attempt < maxRetries {
            time.Sleep(time.Duration(attempt) * time.Second)
        }
    }

    return fmt.Errorf("retry limit exceeded with partial failures")
}
```

### 4. Use Context for Timeouts

Always use context for timeout and cancellation control:

```go
// Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

receipt, err := hub.Send(ctx, message)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("operation timed out: %w", err)
    }
    return err
}
```

### 5. Error Classification

Classify errors for appropriate response strategies:

```go
func classifyError(err error) string {
    errStr := err.Error()

    switch {
    case strings.Contains(errStr, "timeout"):
        return "NETWORK_ERROR"
    case strings.Contains(errStr, "invalid"):
        return "VALIDATION_ERROR"
    case strings.Contains(errStr, "unauthorized"):
        return "AUTH_ERROR"
    case strings.Contains(errStr, "rate limit"):
        return "RATE_LIMIT_ERROR"
    default:
        return "UNKNOWN_ERROR"
    }
}
```

## Production Patterns

### Circuit Breaker Pattern

Implement circuit breaker to prevent cascade failures:

```go
type CircuitBreaker struct {
    threshold int
    failures  int
    lastFailure time.Time
    state     string // CLOSED, OPEN, HALF_OPEN
}

func (cb *CircuitBreaker) Send(hub notifyhub.Hub, ctx context.Context, msg *notifyhub.Message) error {
    if cb.state == "OPEN" {
        if time.Since(cb.lastFailure) > 5*time.Minute {
            cb.state = "HALF_OPEN"
        } else {
            return errors.New("circuit breaker is open")
        }
    }

    receipt, err := hub.Send(ctx, msg)

    if err != nil || receipt.Failed > 0 {
        cb.failures++
        cb.lastFailure = time.Now()

        if cb.failures >= cb.threshold {
            cb.state = "OPEN"
        }
        return err
    }

    // Success - reset circuit breaker
    cb.failures = 0
    cb.state = "CLOSED"
    return nil
}
```

### Monitoring Integration

Integrate with monitoring systems:

```go
func sendWithMonitoring(hub notifyhub.Hub, ctx context.Context, msg *notifyhub.Message) error {
    start := time.Now()
    defer metrics.RecordDuration("notifyhub.send.duration", time.Since(start))

    metrics.IncrementCounter("notifyhub.messages.total")

    receipt, err := hub.Send(ctx, msg)

    if err != nil {
        metrics.IncrementCounter("notifyhub.errors.total")
        metrics.IncrementCounterWithTags("notifyhub.errors.by_type", 1,
            map[string]string{"type": classifyError(err)})
        return err
    }

    metrics.IncrementCounter("notifyhub.messages.success", receipt.Successful)
    metrics.IncrementCounter("notifyhub.messages.failed", receipt.Failed)

    // Record per-platform metrics
    for _, result := range receipt.Results {
        tags := map[string]string{"platform": result.Platform}
        if result.Success {
            metrics.IncrementCounterWithTags("notifyhub.platform.success", 1, tags)
        } else {
            metrics.IncrementCounterWithTags("notifyhub.platform.failure", 1, tags)
        }
        metrics.RecordDurationWithTags("notifyhub.platform.duration",
            result.Duration, tags)
    }

    return nil
}
```

## Running the Example

```bash
cd examples/basic/error-handling
go run main.go
```

## Key Takeaways

✅ **Always Check Errors** - At every stage of the process
✅ **Handle Partial Failures** - Some platforms may succeed while others fail
✅ **Implement Retries** - For critical messages with exponential backoff
✅ **Use Context** - For timeout and cancellation control
✅ **Classify Errors** - Different error types need different responses
✅ **Monitor Everything** - Track success rates, errors, and performance
✅ **Circuit Breakers** - Prevent cascade failures in production

## Next Steps

- [Platform Examples](../../platforms/) - Platform-specific error handling
- [Advanced Examples](../../advanced/) - Production-ready patterns
- [Monitoring Example](../../advanced/monitoring/) - Comprehensive monitoring setup