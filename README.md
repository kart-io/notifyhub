# NotifyHub - Unified Notification System

A modern, type-safe, and unified notification system for Go applications with support for multiple platforms including Email, Feishu, Slack, and SMS.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)

## ✨ Features

- **🚀 Unified API**: Single entry point with fluent builders
- **🔒 Type Safety**: Platform-specific builders with compile-time validation
- **🎯 Smart Targets**: Automatic target type detection from string patterns
- **⚡ High Performance**: Asynchronous processing with worker pools
- **🔄 Rate Limiting**: Built-in token bucket rate limiting
- **📊 Monitoring**: Real-time health checks and metrics
- **🔧 Configuration**: Environment-based config with validation
- **🔄 Backward Compatible**: Full backward compatibility with existing APIs
- **🎨 Rich Formatting**: Platform-specific rich content (cards, blocks, HTML)
- **🛡️ Internal Encapsulation**: Core logic protected in internal packages

## 🏗️ Architecture

NotifyHub follows a clean, modular architecture:

```
notifyhub/
├── notifyhub/          # 🎯 Unified SDK Entry Point
├── platforms/          # 🔌 Platform Implementations
├── internal/           # 🔒 Protected Core Logic
│   ├── model/         # 📋 Core Data Types
│   ├── hub/           # ⚙️ Message Coordination
│   ├── queue/         # 📬 Async Processing
│   └── transport/     # 🚀 Delivery Layer
├── logger/            # 📝 Logging Interface
└── examples/          # 📚 Usage Examples
```

## 🚀 Quick Start

### Installation

```bash
go get github.com/kart-io/notifyhub/notifyhub
```

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/kart-io/notifyhub/notifyhub"
)

func main() {
    // Create client with the new unified API
    client, err := notifyhub.New(
        notifyhub.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/xxx", "secret"),
        notifyhub.WithEmailSimple("smtp.gmail.com", 587, "user@gmail.com", "password", "notifications@company.com"),
        notifyhub.WithMemoryQueue(1000, 4),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Send notification with the new chain builder API
    ctx := context.Background()
    err = client.Send(ctx).
        Title("System Alert").
        Body("Database backup completed successfully").
        ToAuto("admin@company.com").   // Smart target detection
        ToAuto("#alerts").             // Channel target
        ToAuto("@john.doe").           // User mention
        Execute()

    if err != nil {
        log.Printf("Send failed: %v", err)
    } else {
        log.Println("Message sent successfully")
    }
}
```

## 📚 Documentation

- [Migration Guide](MIGRATION_GUIDE.md) - Complete migration guide from old API
- [Architecture Refactor](ARCHITECTURE_REFACTOR.md) - Technical details of the refactor
- [Examples](examples/) - Usage examples and demos
- [Refactor Completion](REFACTOR_COMPLETION.md) - Refactor completion report

## 🎯 Smart Target Detection

NotifyHub automatically detects target types from string patterns:

```go
client.Send(ctx).
    Title("Multi-platform Notification").
    Body("This message will be sent to multiple platforms").
    ToAuto("user@example.com").      // → Email target
    ToAuto("@john").                 // → User mention
    ToAuto("#alerts").               // → Channel target
    ToAuto("+1234567890").           // → SMS target
    ToAuto("https://webhook.com").   // → Webhook target
    Execute()
```

## 🔧 Advanced Features

### 🚨 Alert Messages

```go
// High-priority alerts with automatic routing
err := client.Alert(ctx).
    Title("CRITICAL: Database Down").
    Body("Primary database cluster is unreachable").
    Severity("critical").
    Source("database-monitor").
    ToOnCall().                      // Routes to on-call team
    Execute()
```

### 📬 Notification Messages

```go
// Regular notifications
err := client.Notification(ctx).
    Title("Daily Report").
    Body("System processed 1,234 requests today").
    Category("daily-report").
    ToChannel("general").
    Execute()
```

### 🎨 Template Messages

```go
// Using templates with variables
err := client.Send(ctx).
    Template("deployment_template").
    Variable("service", "user-service").
    Variable("version", "v2.1.0").
    Variable("environment", "production").
    ToEmail("devops@company.com").
    Execute()
```

### ⏰ Scheduled Messages

```go
// Schedule messages for later delivery
err := client.Send(ctx).
    Title("Maintenance Notice").
    Body("Scheduled maintenance in 1 hour").
    Schedule(time.Now().Add(time.Hour)).
    ToAuto("#maintenance").
    Execute()
```

## ⚙️ Configuration

### Environment Variables

```bash
# Email
NOTIFYHUB_EMAIL_HOST=smtp.gmail.com
NOTIFYHUB_EMAIL_PORT=587
NOTIFYHUB_EMAIL_USERNAME=user@gmail.com
NOTIFYHUB_EMAIL_PASSWORD=password
NOTIFYHUB_EMAIL_FROM=notifications@company.com

# Feishu
NOTIFYHUB_FEISHU_WEBHOOK=https://open.feishu.cn/...
NOTIFYHUB_FEISHU_SECRET=your-secret

# Slack
NOTIFYHUB_SLACK_TOKEN=xoxb-123456789

# Global
NOTIFYHUB_DEBUG=true
NOTIFYHUB_TIMEOUT=30s
```

### Programmatic Configuration

```go
cfg := notifyhub.NewConfig().
    WithEmail("smtp.gmail.com", 587, "user", "pass", "from@example.com").
    WithFeishu("webhook-url", "secret").
    WithSlack("bot-token").
    WithQueue("memory", 1000, 4).
    WithRateLimit(100, 10).
    WithRetry(3, time.Second, 30*time.Second).
    WithTelemetry("notifyhub", "1.0.0", "production", "http://jaeger:14268").
    WithDebug().
    LoadFromEnv()
```

## 📊 Monitoring & Health

```go
// Check system health
health := client.Health()
fmt.Printf("Status: %s, Uptime: %v\n", health.Status, health.Uptime)

// Get metrics
metrics := client.Metrics()
fmt.Printf("Sent: %d, Failed: %d, Queued: %d\n",
    metrics.MessagesSent, metrics.MessagesFailed, metrics.MessagesQueued)
```

## 🔄 Migration from V1/V2

### V1 Compatibility Layer

For existing V1 code, use the compatibility layer:

```go
// Convert existing V1 config
v1Client, err := notifyhub.NewV1Compat(existingV1Config)

// Existing V1 code continues to work
msg := v1Client.NewMessage()
targets := []sending.Target{...}
results, err := v1Client.Send(ctx, msg, targets)
```

### Migration Benefits

| Feature | V1 | Unified API |
|---------|----|----|
| Configuration | Complex setup | Fluent builders |
| Type Safety | Runtime errors | Compile-time validation |
| Target Specification | Manual target creation | Smart detection |
| Platform Features | Limited access | Type-safe builders |
| Error Handling | Generic errors | Specific error types |
| Monitoring | Complex analysis | Built-in health/metrics |

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client API    │    │   Configuration  │    │   Monitoring    │
│                 │    │                  │    │                 │
│ • Send()        │────│ • Environment    │────│ • Health()      │
│ • Email()       │    │ • Fluent Builder │    │ • Metrics()     │
│ • Feishu()      │    │ • Validation     │    │ • Telemetry     │
│ • Slack()       │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Core Engine   │    │   Queue System   │    │   Transports    │
│                 │    │                  │    │                 │
│ • Smart Routing │────│ • Memory/Redis   │────│ • Email SMTP    │
│ • Rate Limiting │    │ • Worker Pools   │    │ • Feishu API    │
│ • Retry Logic   │    │ • Async Process  │    │ • Slack API     │
│ • Validation    │    │                  │    │ • SMS/Webhook   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## 🧪 Testing

```go
func TestNotification(t *testing.T) {
    cfg := notifyhub.NewConfig().
        WithEmail("smtp.test.com", 587, "test", "pass", "test@example.com")
    
    client, err := notifyhub.New(cfg.Config)
    require.NoError(t, err)
    defer client.Shutdown(context.Background())
    
    result, err := client.Send().
        Title("Test").
        Body("Test message").
        To("test@example.com").
        Execute(context.Background())
    
    require.NoError(t, err)
    assert.NotEmpty(t, result.MessageID)
}
```

## 📖 Examples

- [Basic Usage](examples/unified_api/main.go) - Complete unified API example
- [V2 Features](examples/v2_api_demo/main.go) - Advanced v2 API features
- [Migration Example](MIGRATION.md) - Step-by-step migration guide

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by modern notification systems and Go best practices
- Built with performance and type safety in mind
- Community-driven development and feedback

---

**NotifyHub** - Making notifications simple, type-safe, and powerful! 🚀
