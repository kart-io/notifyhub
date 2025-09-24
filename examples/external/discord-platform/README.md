# External Platform Development

This example demonstrates how to create external platform packages that integrate seamlessly with NotifyHub's unified architecture without modifying the core library.

## What This Example Shows

- Complete Discord platform implementation as external package
- Same API quality as built-in platforms
- Auto-registration mechanism
- True external extensibility
- Consistent developer experience

## Key Architecture Benefits

### 1. No Core Library Changes

External developers can create platform packages independently:

- No need to fork NotifyHub
- No pull requests to core repository
- Independent development and deployment
- Faster iteration cycles

### 2. First-Class Citizen Status

External platforms have the same capabilities as built-in ones:

```go
// Built-in platform
feishu.WithFeishu("webhook", feishu.WithFeishuSecret("secret"))

// External platform - identical API quality
discord.WithDiscord("webhook", discord.WithDiscordUsername("bot"))
```

### 3. Auto-Registration

Platforms register themselves when imported:

```go
var registerOnce sync.Once

func ensureRegistered() {
    registerOnce.Do(func() {
        notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
            Name:    "discord",
            Creator: NewDiscordSender,
            // ... configuration
        })
    })
}
```

## External Package Structure

To create your own external platform package:

### 1. Package Layout

```
yourorg/notifyhub-myplatform/
├── sender.go      # ExternalSender implementation
├── options.go     # Convenience functions
├── README.md      # Documentation
└── go.mod         # Go module
```

### 2. Implement ExternalSender Interface

```go
type MyPlatformSender struct {
    // Platform-specific fields
}

// Required methods
func (m *MyPlatformSender) Name() string { return "myplatform" }
func (m *MyPlatformSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error)
func (m *MyPlatformSender) ValidateTarget(target platform.Target) error
func (m *MyPlatformSender) GetCapabilities() platform.Capabilities
func (m *MyPlatformSender) IsHealthy(ctx context.Context) error
func (m *MyPlatformSender) Close() error
```

### 3. Create Convenience Functions

```go
func WithMyPlatform(apiKey string, options ...func(map[string]interface{})) notifyhub.HubOption {
    ensureRegistered()

    config := map[string]interface{}{
        "api_key": apiKey,
        "timeout": 30 * time.Second,
    }

    for _, opt := range options {
        opt(config)
    }

    return notifyhub.WithCustomPlatform("myplatform", config)
}

func WithMyPlatformTimeout(timeout time.Duration) func(map[string]interface{}) {
    return func(config map[string]interface{}) {
        config["timeout"] = timeout
    }
}
```

### 4. Auto-Registration

```go
var registerOnce sync.Once

func ensureRegistered() {
    registerOnce.Do(func() {
        notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
            Name:    "myplatform",
            Creator: NewMyPlatformSender,
            DefaultOpts: func() map[string]interface{} {
                return map[string]interface{}{
                    "timeout": 30 * time.Second,
                }
            },
            Validator: func(config map[string]interface{}) error {
                if _, ok := config["api_key"].(string); !ok {
                    return fmt.Errorf("api_key is required")
                }
                return nil
            },
        })
    })
}
```

## Discord Platform Features

The Discord platform implementation showcases:

### Rich Embeds

```go
message := notifyhub.NewMessage("Rich Content").
    WithPlatformData(map[string]interface{}{
        "discord_embeds": []map[string]interface{}{
            {
                "title":       "System Alert",
                "description": "Status update",
                "color":       0x7289da,
                "fields": []map[string]interface{}{
                    {
                        "name":   "Status",
                        "value":  "Online",
                        "inline": true,
                    },
                },
                "timestamp": time.Now().Format(time.RFC3339),
            },
        },
    }).
    Build()
```

### User Mentions

```go
message := notifyhub.NewMessage("Mention Users").
    WithPlatformData(map[string]interface{}{
        "discord_mentions": []string{"123456789", "987654321"},
    }).
    Build()
```

### Custom Bot Identity

```go
hub, err := notifyhub.NewHub(
    discord.WithDiscord("webhook-url",
        discord.WithDiscordUsername("Custom Bot Name"),
        discord.WithDiscordAvatar("https://example.com/avatar.png"),
    ),
)
```

## Running the Example

```bash
cd examples/external/discord-platform
go run main.go
```

## Use Cases for External Platforms

### Popular Platforms
- Slack, Microsoft Teams, Telegram
- Twitter, LinkedIn, Reddit
- PagerDuty, Opsgenie, VictorOps

### Specialized Platforms
- Custom webhook services
- Internal corporate systems
- IoT device notifications
- Mobile push notification services

### Industry-Specific
- Healthcare: HIPAA-compliant messaging
- Finance: Encrypted communication
- DevOps: Custom monitoring integrations

## Development Best Practices

### 1. Follow Naming Conventions

```go
// Package name matches platform
package myplatform

// Main function follows pattern
func WithMyPlatform(required string, options ...func(map[string]interface{})) notifyhub.HubOption

// Option functions are prefixed
func WithMyPlatformTimeout(timeout time.Duration) func(map[string]interface{})
```

### 2. Provide Comprehensive Configuration

```go
func WithMyPlatform(apiKey string, options ...func(map[string]interface{})) notifyhub.HubOption {
    // Set sensible defaults
    config := map[string]interface{}{
        "api_key": apiKey,
        "timeout": 30 * time.Second,
        "retry_count": 3,
    }

    // Apply user options
    for _, opt := range options {
        opt(config)
    }

    return notifyhub.WithCustomPlatform("myplatform", config)
}
```

### 3. Implement Robust Error Handling

```go
func (m *MyPlatformSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
    results := make([]*platform.SendResult, len(targets))

    for i, target := range targets {
        result := &platform.SendResult{Target: target}

        // Validate target
        if err := m.ValidateTarget(target); err != nil {
            result.Error = err.Error()
            results[i] = result
            continue
        }

        // Send with timeout and error handling
        // ... implementation

        results[i] = result
    }

    return results, nil
}
```

### 4. Document Platform Capabilities

```go
func (m *MyPlatformSender) GetCapabilities() platform.Capabilities {
    return platform.Capabilities{
        Name:                 "myplatform",
        SupportedTargetTypes: []string{"user", "channel", "webhook"},
        SupportedFormats:     []string{"text", "markdown", "html"},
        MaxMessageSize:       4096,
        SupportsScheduling:   false,
        SupportsAttachments:  true,
        SupportsMentions:     true,
        SupportsRichContent:  true,
        RequiredSettings:     []string{"api_key"},
    }
}
```

## Community Ecosystem

The external platform system enables:

- **Community Contributions** - Anyone can create platform packages
- **Specialized Solutions** - Niche platforms for specific use cases
- **Independent Maintenance** - Platform packages have their own maintainers
- **Rich Ecosystem** - Growing collection of community platforms

## Next Steps

- Study the Discord implementation in this example
- Create your own platform package
- Publish to Go modules for community use
- Contribute to the NotifyHub ecosystem

This architecture solves the original extensibility problem while maintaining excellent developer experience!