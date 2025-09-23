# NotifyHub Elegant API Examples

This directory demonstrates the new elegant API architecture for NotifyHub, featuring a fluent interface and simplified usage patterns.

## Architecture Overview

The new architecture follows a three-layer design:

### 1. Public API Layer (`pkg/notifyhub/`)

- **Clean Interface**: Single package import with fluent API
- **Message Builder**: Chain-based message construction
- **Type Safety**: Strong typing for all configurations
- **Platform Abstraction**: Unified interface across all platforms

### 2. Internal Core Layer (`internal/platform/`, `internal/dispatcher/`)

- **Platform Senders**: Dedicated implementations for each platform
- **Message Routing**: Automatic target-to-platform resolution
- **Health Management**: Built-in health checking and monitoring
- **Error Handling**: Comprehensive error classification

### 3. Base Services Layer (`internal/queue/`, `internal/logger/`, `internal/config/`)

- **Queue System**: Async processing with multiple backends
- **Configuration**: Environment-based and programmatic setup
- **Logging**: Structured logging with multiple levels

## Key Features

### 🚀 Elegant Fluent API

```go
// Before (old verbose API)
message := &Message{
    Title: "Task Complete",
    Body:  "Processing finished",
    Targets: []Target{
        {Type: "email", Value: "user@example.com"},
        {Type: "feishu_user", Value: "oc_xxxxxxxx"},
    },
}

// After (new elegant API)
receipt, err := hub.Send(ctx, notifyhub.NewMessage("Task Complete").
    WithText("Processing finished").
    ToEmail("user@example.com").
    ToFeishuUser("oc_xxxxxxxx"),
)
```

### 🎯 Platform-Specific Features

Each platform's unique features are accessible through the unified interface:

```go
// Feishu cards and mentions
hub.Send(ctx, notifyhub.NewAlert("System Alert").
    WithFeishuCard(cardContent).
    WithFeishuMentions("oc_user1", "oc_user2").
    ToFeishuGroup("oc_group"),
)

// Email with CC/BCC and HTML
hub.Send(ctx, notifyhub.NewMessage("Report").
    WithHTML(htmlContent).
    WithEmailCC("manager@company.com").
    WithEmailPriority("high").
    ToEmail("admin@company.com"),
)

// SMS with templates
hub.Send(ctx, notifyhub.NewMessage().
    WithSMSTemplate("order_confirmation").
    WithSMSVariables(map[string]interface{}{
        "order_id": "12345",
        "total": "$99.99",
    }).
    ToPhone("+1234567890"),
)
```

### 🔧 Simple Configuration

```go
hub, err := notifyhub.NewHub(
    // Configure platforms
    notifyhub.WithFeishu(map[string]interface{}{
        "webhook_url": "https://...",
        "secret": "...",
    }),
    notifyhub.WithEmail(map[string]interface{}{
        "smtp_host": "smtp.gmail.com",
        "smtp_port": 587,
        "smtp_username": "user@gmail.com",
        "smtp_password": "password",
        "smtp_from": "user@gmail.com",
    }),

    // Global settings
    notifyhub.WithTimeout(10*time.Second),
    notifyhub.WithRetryPolicy(3, time.Second),
)
```

### 📊 Rich Response Information

```go
receipt, err := hub.Send(ctx, message)
if err != nil {
    return err
}

fmt.Printf("Status: %s, Successful: %d, Failed: %d\n",
    receipt.Status, receipt.Successful, receipt.Failed)

// Detailed per-platform results
for _, result := range receipt.Results {
    fmt.Printf("Platform: %s, Success: %v, Error: %s\n",
        result.Platform, result.Success, result.Error)
}
```

### ⚡ Auto-Detection and Smart Routing

```go
// Automatically detects platform and target type
hub.Send(ctx, notifyhub.NewMessage("Hello").
    ToFeishu("oc_xxxxxxxx").           // Auto-detected as Feishu user
    ToFeishu("https://feishu.cn/..."). // Auto-detected as webhook
    ToEmail("user@example.com").       // Auto-detected as email
    ToPhone("+1234567890"),           // Auto-detected as SMS
)
```

### 🎯 Message Types and Priorities

```go
// Different message types with appropriate priorities
notifyhub.NewMessage("Info")           // Normal priority
notifyhub.NewAlert("Warning")          // High priority
notifyhub.NewUrgent("Critical Error")  // Urgent priority
```

### ⏰ Scheduling Support

```go
// Schedule for specific time
hub.Send(ctx, message.ScheduleAt(futureTime))

// Schedule relative to now
hub.Send(ctx, message.ScheduleIn(5*time.Minute))
```

### 🔄 Async Processing

```go
// Send asynchronously
asyncReceipt, err := hub.SendAsync(ctx, message)
fmt.Printf("Queued: %s, Status: %s\n",
    asyncReceipt.MessageID, asyncReceipt.Status)
```

### 💊 Health Monitoring

```go
health, err := hub.Health(ctx)
fmt.Printf("Overall: %s\n", health.Status)

for platform, status := range health.Platforms {
    fmt.Printf("%s: %s\n", platform, status.Status)
}
```

## Running the Examples

```bash
cd examples/elegant_api
go run main.go
```

## Key Architectural Benefits

1. **Single Import**: Only need `github.com/kart-io/notifyhub/pkg/notifyhub`
2. **Type Safety**: Compile-time validation of all configurations
3. **Platform Decoupling**: Easy to add new platforms without affecting existing code
4. **Error Handling**: Rich error information with proper classification
5. **Testing Friendly**: Interface-based design enables easy mocking
6. **Performance**: Parallel sending across platforms
7. **Observability**: Built-in health checks and metrics support

## Migration from Old API

The new API is designed to be intuitive for new users while providing migration paths for existing code. The old patterns can be gradually replaced with the new fluent interface.

## Configuration Options

All configuration can be done through:

- **Functional Options**: `notifyhub.WithFeishu()`, `notifyhub.WithEmail()`, etc.
- **Environment Variables**: Automatic detection and loading
- **Configuration Files**: JSON/YAML support
- **Runtime Updates**: Dynamic configuration changes

This new architecture provides a clean, powerful, and maintainable foundation for notification management across multiple platforms.
