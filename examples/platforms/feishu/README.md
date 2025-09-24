# Feishu Platform Features

This example demonstrates the complete Feishu/Lark platform integration capabilities in NotifyHub's unified architecture.

## What You'll Learn

- Different Feishu authentication modes
- Rich content types (cards, posts, mentions)
- Various target types (webhook, group, user, channel)
- Platform-specific features and configuration
- Legacy compatibility patterns

## Authentication Modes

### 1. No Authentication (Simple Webhook)

For basic webhook-only integration:

```go
hub, err := notifyhub.NewHub(
    feishu.WithFeishu("https://example.com/feishu/webhook"),
)
```

### 2. Signature Authentication (HMAC-SHA256)

For secure webhook with signature verification:

```go
hub, err := notifyhub.NewHub(
    feishu.WithFeishu("https://example.com/feishu/webhook",
        feishu.WithFeishuSecret("your-webhook-secret"),
        feishu.WithFeishuAuthMode(feishu.AuthModeSignature),
    ),
)
```

### 3. Keywords Authentication

For keyword-based message filtering:

```go
hub, err := notifyhub.NewHub(
    feishu.WithFeishu("https://example.com/feishu/webhook",
        feishu.WithFeishuKeywords([]string{"alert", "notification"}),
    ),
)
```

## Message Types

### Basic Text Messages

Simple text notifications:

```go
msg := notifyhub.NewMessage("System Alert").
    WithBody("Database connection restored.").
    ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
    Build()
```

### Messages with Mentions

#### @All Mention
```go
msg := notifyhub.NewAlert("Critical Issue").
    WithBody("Immediate attention required!").
    WithPlatformData(map[string]interface{}{
        "feishu_mention_all": true,
    }).
    Build()
```

#### Specific User Mentions
```go
msg := notifyhub.NewMessage("Task Assignment").
    WithBody("Please review the deployment.").
    WithPlatformData(map[string]interface{}{
        "feishu_mentions": []map[string]interface{}{
            {"user_id": "ou_123456789"},
            {"user_id": "ou_987654321"},
        },
    }).
    Build()
```

### Interactive Cards

Rich interactive content with buttons:

```go
msg := notifyhub.NewMessage("System Status").
    WithPlatformData(map[string]interface{}{
        "feishu_card": map[string]interface{}{
            "header": map[string]interface{}{
                "title": map[string]interface{}{
                    "tag":     "plain_text",
                    "content": "🔔 System Notification",
                },
                "template": "blue",
            },
            "elements": []map[string]interface{}{
                {
                    "tag": "div",
                    "text": map[string]interface{}{
                        "tag":     "lark_md",
                        "content": "**Status**: ✅ Healthy\n**Uptime**: 99.9%",
                    },
                },
                {
                    "tag": "action",
                    "actions": []map[string]interface{}{
                        {
                            "tag": "button",
                            "text": map[string]interface{}{
                                "tag":     "plain_text",
                                "content": "View Details",
                            },
                            "url": "https://monitor.example.com",
                        },
                    },
                },
            },
        },
    }).
    Build()
```

### Rich Text Posts

Multi-paragraph formatted content:

```go
msg := notifyhub.NewMessage("Daily Report").
    WithPlatformData(map[string]interface{}{
        "feishu_post": map[string]interface{}{
            "zh_cn": map[string]interface{}{
                "title": "📈 Daily Report",
                "content": [][]map[string]interface{}{
                    {
                        {
                            "tag":  "text",
                            "text": "Traffic increased by ",
                        },
                        {
                            "tag":  "text",
                            "text": "25%",
                            "style": []string{"bold"},
                        },
                    },
                },
            },
        },
    }).
    Build()
```

## Target Types

### Webhook Target (Default)
Send to bot webhook endpoint:
```go
.ToTarget(notifyhub.NewTarget("webhook", "", "feishu"))
```

### Group Target
Send to specific group chat:
```go
.ToTarget(notifyhub.NewTarget("group", "oc_group123456789", "feishu"))
```

### User Target
Send private message to user:
```go
.ToTarget(notifyhub.NewTarget("user", "ou_user123456789", "feishu"))
```

### Channel Target
Send to specific channel:
```go
.ToTarget(notifyhub.NewTarget("channel", "oc_channel123456789", "feishu"))
```

## Advanced Configuration

### Custom Timeouts and Settings

```go
hub, err := notifyhub.NewHub(
    feishu.WithFeishu("https://example.com/feishu/webhook",
        feishu.WithFeishuSecret("webhook-secret"),
        feishu.WithFeishuTimeout(45*time.Second),
        feishu.WithFeishuAuthMode(feishu.AuthModeSignature),
    ),
)
```

### Configuration Options

- `WithFeishuSecret(secret)` - Add HMAC signature
- `WithFeishuKeywords(keywords)` - Set keyword filters
- `WithFeishuTimeout(duration)` - Custom timeout
- `WithFeishuAuthMode(mode)` - Explicit auth mode

## Legacy Compatibility

Deprecated functions still work for backward compatibility:

```go
// Deprecated but still functional
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishu("webhook-url", "secret"), // Old way
)

// Recommended new way
hub, err := notifyhub.NewHub(
    feishu.WithFeishu("webhook-url", feishu.WithFeishuSecret("secret")), // New way
)
```

## Use Cases

### Development Team Alerts
```go
msg := notifyhub.NewAlert("Build Failed").
    WithBody("CI/CD pipeline failed on branch: main").
    WithPlatformData(map[string]interface{}{
        "feishu_mentions": []map[string]interface{}{
            {"user_id": "ou_developer1"},
            {"user_id": "ou_developer2"},
        },
    }).
    Build()
```

### System Monitoring
```go
msg := notifyhub.NewMessage("System Health").
    WithPlatformData(map[string]interface{}{
        "feishu_card": systemHealthCard(), // Custom card function
    }).
    Build()
```

### Incident Response
```go
msg := notifyhub.NewUrgent("Production Incident").
    WithBody("Payment gateway is down!").
    WithPlatformData(map[string]interface{}{
        "feishu_mention_all": true,
    }).
    Build()
```

## Running the Example

```bash
cd examples/platforms/feishu
go run main.go
```

## Configuration Setup

1. Create a Feishu bot in your organization
2. Get the webhook URL from bot settings
3. (Optional) Configure webhook secret for security
4. Update the example with your webhook URL

## Platform Capabilities

The Feishu platform supports:

✅ **Rich Content** - Cards, posts, formatted text
✅ **Mentions** - @all and specific user mentions
✅ **Authentication** - Multiple security modes
✅ **Target Types** - Webhook, group, user, channel
✅ **Attachments** - File and image support
✅ **Interactive Elements** - Buttons and forms

## Next Steps

- [Email Platform](../email/) - SMTP email integration
- [SMS Platform](../sms/) - Multi-provider SMS support
- [Unified Demo](../unified-demo/) - All platforms together