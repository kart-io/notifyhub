# NotifyHub Examples

Welcome to NotifyHub examples! This directory contains comprehensive examples demonstrating the unified platform management system and various NotifyHub features.

## 🚀 Quick Start

Choose an example based on your needs:

- **New to NotifyHub?** Start with [Basic Examples](basic/)
- **Platform-specific features?** See [Platform Examples](platforms/)
- **Advanced use cases?** Check [Advanced Examples](advanced/)
- **Building external platforms?** See [External Platform Example](external/)

## 📁 Examples Structure

### 📚 [Basic Examples](basic/)
Core NotifyHub functionality and getting started guides.

- **[getting-started](basic/getting-started/)** - Your first NotifyHub application
- **[multi-platform](basic/multi-platform/)** - Sending to multiple platforms
- **[message-types](basic/message-types/)** - Different message types (alerts, urgent, etc.)
- **[error-handling](basic/error-handling/)** - Proper error handling patterns

### 🔧 [Platform Examples](platforms/)
Platform-specific features and integrations using the new unified architecture.

- **[feishu](platforms/feishu/)** - Feishu/Lark platform features
- **[email](platforms/email/)** - Email SMTP configurations and features
- **[sms](platforms/sms/)** - SMS with multiple providers
- **[unified-demo](platforms/unified-demo/)** - All platforms working together

### 🚀 [Advanced Examples](advanced/)
Complex scenarios and production-ready patterns.

- **[middleware](advanced/middleware/)** - Custom middleware development
- **[configuration](advanced/configuration/)** - Advanced configuration patterns
- **[monitoring](advanced/monitoring/)** - Health checks and monitoring
- **[enterprise](advanced/enterprise/)** - Enterprise deployment patterns

### 🌟 [External Platform Example](external/)
Complete example of creating external platform packages.

- **[discord-platform](external/discord-platform/)** - Full Discord platform implementation
- **[custom-webhook](external/custom-webhook/)** - Generic webhook platform example

## ⚡ Running Examples

Each example includes:
- `main.go` - Runnable demo code
- `README.md` - Detailed explanation and usage
- Configuration examples and best practices

```bash
# Run any example
cd examples/basic/getting-started
go run main.go

# Or with specific configuration
cd examples/platforms/feishu
go run main.go
```

## 🏗️ Architecture Highlights

### Unified Platform Management
All examples use the new unified platform architecture:

```go
import (
    "github.com/kart-io/notifyhub/pkg/notifyhub"
    "github.com/kart-io/notifyhub/pkg/platforms/feishu"
    "github.com/kart-io/notifyhub/pkg/platforms/email"
)

hub, err := notifyhub.NewHub(
    feishu.WithFeishu("webhook-url", feishu.WithFeishuSecret("secret")),
    email.WithEmail("smtp.host.com", 587, "from@example.com"),
)
```

### External Extensibility
External platforms integrate seamlessly:

```go
import "github.com/yourorg/notifyhub-discord"

hub, err := notifyhub.NewHub(
    discord.WithDiscord("webhook-url"),  // Same API as built-in platforms
)
```

### Backward Compatibility
Existing code continues to work:

```go
// Still works, but deprecated
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishu("webhook", "secret"),
)
```

## 📖 Learning Path

### Beginner
1. [Getting Started](basic/getting-started/) - Basic concepts
2. [Multi-Platform](basic/multi-platform/) - Multiple platforms
3. [Message Types](basic/message-types/) - Different message types

### Intermediate
4. [Platform Features](platforms/) - Platform-specific features
5. [Error Handling](basic/error-handling/) - Robust error handling
6. [Configuration](advanced/configuration/) - Advanced config patterns

### Advanced
7. [Middleware](advanced/middleware/) - Custom middleware
8. [External Platforms](external/) - Building platform packages
9. [Enterprise](advanced/enterprise/) - Production deployment

## 🔗 Related Documentation

- [Platform Architecture V2](../PLATFORM_ARCHITECTURE_V2.md) - New architecture design
- [API Documentation](../pkg/notifyhub/) - Core API reference
- [Platform Packages](../pkg/platforms/) - Official platform implementations

## 🆘 Need Help?

- Check the README in each example directory
- Review the architecture documentation
- Look at similar examples for reference
- Each example is self-contained and well-documented

Happy coding with NotifyHub! 🎉