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
- **🔄 Backward Compatible**: V1 API compatibility layer
- **🎨 Rich Formatting**: Platform-specific rich content (cards, blocks, HTML)

## 🚀 Quick Start

### Installation

```bash
go get github.com/kart-io/notifyhub
```

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/kart-io/notifyhub"
    "github.com/kart-io/notifyhub/api/v2/types"
)

func main() {
    // Create configuration
    cfg := notifyhub.NewConfig().
        WithEmail("smtp.gmail.com", 587, "user@gmail.com", "password", "notifications@company.com").
        WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/xxx", "secret").
        WithSlack("xoxb-123456789").
        LoadFromEnv()

    // Create client
    client, err := notifyhub.New(cfg.Config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Send notification with smart targets
    result, err := client.Send().
        Title("System Alert").
        Body("Database backup completed successfully").
        To("admin@company.com", "#alerts", "@john.doe").
        Priority(types.PriorityHigh).
        Execute(context.Background())

    if err != nil {
        log.Printf("Send failed: %v", err)
    } else {
        log.Printf("Message sent: %s", result.MessageID)
    }
}
```

## 📚 Documentation

- [Migration Guide](MIGRATION.md) - Migrate from v1/v2 APIs
- [API Reference](api/v2/README.md) - Complete API documentation
- [Examples](examples/) - Usage examples and demos

## 🎯 Smart Target Detection

NotifyHub automatically detects target types from string patterns:

```go
client.Send().
    To(
        "user@example.com",    // → Email target
        "@john",               // → User mention  
        "#alerts",             // → Channel target
        "+1234567890",         // → SMS target
        "https://webhook.com", // → Webhook target
    ).
    Execute(ctx)
```

## 🔧 Platform-Specific Features

### 📧 Email with Rich Features

```go
client.Email().
    Title("Monthly Report").
    Body("Please find the report attached").
    HTMLBody("<h1>Report</h1><p>Content here</p>").
    To("manager@company.com").
    CC("team@company.com").
    Priority(types.PriorityHigh).
    EnableTracking().
    Attach("report.pdf", pdfContent).
    Send(ctx)
```

### 🚀 Feishu Cards

```go
client.Feishu().
    AlertCard("Production Issue", "Database timeout detected", types.AlertLevelError).
    ToGroup("operations").
    AtUser("oncall", "admin").
    AddButton("View Dashboard", "https://dashboard.company.com").
    AddImage("chart.png", "Performance chart").
    Send(ctx)
```

### 💬 Slack Blocks

```go
client.Slack().
    HeaderBlock("Deployment Status").
    SimpleBlock("✅ Application v2.1.0 deployed successfully").
    DividerBlock().
    AddField("Environment", "Production", true).
    AddField("Duration", "2m 34s", true).
    AddButton("View Logs", "logs", "https://logs.company.com").
    ToChannel("deployments").
    Send(ctx)
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
