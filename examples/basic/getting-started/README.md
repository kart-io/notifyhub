# Getting Started with NotifyHub

This example demonstrates the basic concepts of NotifyHub using the new unified platform management system.

## What You'll Learn

- How to create a NotifyHub instance
- Basic message creation and sending
- Different message types (normal, alert, urgent)
- Understanding the new unified platform architecture

## Key Concepts

### 1. Unified Platform Packages

Instead of using hardcoded functions in the core library, each platform now lives in its own package:

```go
import "github.com/kart-io/notifyhub/pkg/platforms/feishu"

hub, err := notifyhub.NewHub(
    feishu.WithFeishu("webhook-url", feishu.WithFeishuSecret("secret")),
)
```

### 2. Auto-Registration

Platforms automatically register themselves when their packages are imported. No manual configuration needed!

### 3. Message Types

NotifyHub provides convenient message builders for different scenarios:

- `NewMessage()` - Normal priority messages
- `NewAlert()` - High priority messages
- `NewUrgent()` - Highest priority messages

### 4. Fluent API

Build messages using a fluent, chainable API:

```go
message := notifyhub.NewMessage("Title").
    WithBody("Message content").
    ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
    Build()
```

## Running the Example

```bash
cd examples/basic/getting-started
go run main.go
```

## Configuration

Replace the example webhook URL with your actual Feishu webhook:

```go
feishu.WithFeishu("https://your-actual-webhook-url", /* options */)
```

## Next Steps

- [Multi-Platform Example](../multi-platform/) - Using multiple platforms together
- [Message Types Example](../message-types/) - Advanced message configuration
- [Platform Examples](../../platforms/) - Platform-specific features

## Architecture Benefits

This example showcases the new unified architecture:

✅ **Clean Separation** - Platform code isolated in packages
✅ **External Extensibility** - Easy to add new platforms
✅ **Consistent APIs** - Same patterns across all platforms
✅ **Backward Compatibility** - Existing code continues to work