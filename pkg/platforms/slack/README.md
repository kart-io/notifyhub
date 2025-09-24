# Slack Platform for NotifyHub

This package provides Slack integration for NotifyHub with automatic platform registration and convenient configuration options.

## Quick Start

```go
import (
    "github.com/kart-io/notifyhub/pkg/notifyhub"
    "github.com/kart-io/notifyhub/pkg/platforms/slack"
)

// Simple usage
hub, err := notifyhub.NewHub(
    slack.WithSlack("https://example.com/slack/webhook/your-id"),
)

// With options
hub, err := notifyhub.NewHub(
    slack.WithSlack(
        "https://example.com/slack/webhook/your-id",
        slack.WithSlackTimeout(45*time.Second),
        slack.WithSlackUsername("NotifyHub Bot"),
        slack.WithSlackIcon(":robot_face:"),
    ),
)
```

## Features

- **Automatic Registration**: No need to manually register the platform
- **Convenient Options**: Type-safe configuration functions
- **Rich Content Support**: Supports Slack blocks and attachments
- **Flexible Targeting**: Channel, user, and webhook targets
- **Health Checks**: Built-in platform health monitoring

## Configuration Options

### WithSlack(webhookURL, ...options)

Creates a Slack platform configuration with the specified webhook URL.

**Parameters:**

- `webhookURL`: Your Slack webhook URL (required)
- `options`: Additional configuration options (optional)

### Available Options

- `WithSlackTimeout(duration)`: Sets request timeout
- `WithSlackUsername(username)`: Sets default bot username
- `WithSlackIcon(emoji)`: Sets default bot icon emoji
- `WithSlackChannel(channel)`: Sets default channel

## Message Types

### Text Messages

```go
msg := notifyhub.NewMessage("Alert").
    Body("System is down").
    ToTarget(notifyhub.NewTarget("channel", "#alerts", "slack")).
    Build()
```

### Rich Blocks

```go
msg := notifyhub.NewMessage("Deployment").
    WithPlatformData(map[string]interface{}{
        "slack_blocks": []map[string]interface{}{
            {
                "type": "section",
                "text": map[string]interface{}{
                    "type": "mrkdwn",
                    "text": "*Deployment Complete* :white_check_mark:",
                },
            },
        },
    }).
    Build()
```

## Supported Targets

- **Channel**: `#channel-name` or `channel-id`
- **User**: `@username` or `user-id`
- **Webhook**: Direct webhook posting

```go
// Channel target
target := notifyhub.NewTarget("channel", "#alerts", "slack")

// User target
target := notifyhub.NewTarget("user", "@john.doe", "slack")

// Webhook target
target := notifyhub.NewTarget("webhook", "", "slack")
```

## Platform Capabilities

- **Message Size**: Up to 40KB
- **Formats**: Text, Markdown
- **Rich Content**: Blocks, attachments
- **Mentions**: User and channel mentions
- **Scheduling**: Not supported (use external scheduling)

## External Platform Example

This package demonstrates how external developers can create their own platform integrations:

1. **Implement ExternalSender Interface**
2. **Provide Convenient Options Functions**
3. **Handle Automatic Registration**
4. **Provide Complete Documentation**

External developers can use this as a template for creating their own platform integrations without modifying the core NotifyHub library.
