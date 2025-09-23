# External Platform Extensions Example

This directory demonstrates how to extend NotifyHub with external platforms using the new plugin-style architecture.

## Overview

NotifyHub v2 introduces a powerful extension system that allows you to add support for any external platform without modifying the core library. This example shows how to:

1. Implement a custom platform sender (Slack)
2. Register the platform extension
3. Use the extended platform alongside built-in platforms
4. Configure platforms using the new extension API

## Directory Structure

```
examples/external_platform/
├── README.md              # This documentation
├── main.go               # Complete demonstration example
└── slack/               # Slack platform implementation
    └── slack_sender.go  # Slack sender implementation
```

## Key Features Demonstrated

### 1. Platform Extension Registration

```go
err := notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
    Name:    "slack",
    Creator: slack.NewSlackSender,
    DefaultOpts: func() map[string]interface{} {
        return map[string]interface{}{
            "timeout": 30 * time.Second,
        }
    },
    Validator: func(config map[string]interface{}) error {
        // Configuration validation logic
        return nil
    },
})
```

### 2. Multiple Configuration Methods

```go
// Method 1: Using convenience function
hub, err := notifyhub.NewHub(
    notifyhub.WithSlack("https://example.com/slack/webhook/your-webhook-id"),
)

// Method 2: Using generic platform config
hub, err := notifyhub.NewHub(
    notifyhub.WithPlatformConfig("slack", map[string]interface{}{
        "webhook_url": "https://example.com/slack/webhook/your-webhook-id",
        "timeout":     45 * time.Second,
    }),
)

// Method 3: Using custom platform function
hub, err := notifyhub.NewHub(
    notifyhub.WithCustomPlatform("slack", config),
)
```

### 3. Mixed Platform Messaging

```go
message := notifyhub.NewMessage("Multi-Platform Alert").
    Body("Alert message").
    // Internal platform
    ToFeishuGroup("feishu_group").
    // External platform
    AddTarget(notifyhub.NewTarget("channel", "#alerts", "slack")).
    Build()
```

### 4. Platform-Specific Features

The example shows how to use platform-specific features like Slack blocks:

```go
slackMessage := notifyhub.NewAlert("Rich Message").
    WithPlatformData(map[string]interface{}{
        "slack_blocks": []map[string]interface{}{
            // Slack-specific block formatting
        },
    }).
    Build()
```

## Implementation Details

### Slack Platform Implementation

The `slack/slack_sender.go` file demonstrates:

- **Interface Compliance**: Implements `platform.ExternalSender` interface
- **Configuration Handling**: Processes webhook URLs, timeouts, and other settings
- **Message Transformation**: Converts generic messages to Slack-specific format
- **Error Handling**: Proper error reporting and validation
- **Rich Content Support**: Slack blocks, attachments, and formatting
- **Health Checks**: Platform availability monitoring

### Key Interface Methods

```go
type ExternalSender interface {
    Name() string
    Send(ctx context.Context, msg *Message, targets []Target) ([]*SendResult, error)
    ValidateTarget(target Target) error
    GetCapabilities() Capabilities
    IsHealthy(ctx context.Context) error
    Close() error
}
```

## Running the Example

1. **Set up Slack webhook** (optional for demo):
   ```bash
   export SLACK_WEBHOOK_URL="https://example.com/slack/webhook/your-webhook-id"
   ```

2. **Run the example**:
   ```bash
   cd examples/external_platform
   go run .
   ```

The example will:
- Register the Slack platform extension
- Create a hub with mixed internal/external platforms
- Send messages to multiple platforms
- Demonstrate platform-specific features
- Perform health checks on all platforms

## Extension Architecture Benefits

### 1. **Zero Core Modification**
Add new platforms without touching NotifyHub core code.

### 2. **Type Safety**
Full compile-time type checking for all configurations.

### 3. **Validation Support**
Built-in configuration validation with custom error handling.

### 4. **Consistent API**
All platforms use the same interface, ensuring consistent behavior.

### 5. **Hot Registration**
Register platforms at runtime before hub creation.

### 6. **Configuration Flexibility**
Multiple configuration methods for different use cases.

## Creating Your Own Platform Extension

To create a custom platform extension:

1. **Implement ExternalSender interface**:
   ```go
   type MyPlatformSender struct {
       // Platform-specific fields
   }

   func (m *MyPlatformSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
       // Implementation
   }
   // ... other interface methods
   ```

2. **Create platform factory function**:
   ```go
   func NewMyPlatformSender(config map[string]interface{}) (platform.ExternalSender, error) {
       // Create and configure your sender
   }
   ```

3. **Register the extension**:
   ```go
   notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
       Name:    "myplatform",
       Creator: NewMyPlatformSender,
       // Optional: default options and validator
   })
   ```

4. **Add convenience functions** (optional):
   ```go
   func WithMyPlatform(apiKey string) notifyhub.HubOption {
       return notifyhub.WithCustomPlatform("myplatform", map[string]interface{}{
           "api_key": apiKey,
       })
   }
   ```

## Best Practices

1. **Validation**: Always implement configuration validation
2. **Error Handling**: Provide detailed error messages
3. **Health Checks**: Implement meaningful health checks
4. **Resource Cleanup**: Properly implement `Close()` method
5. **Documentation**: Document your platform-specific features
6. **Testing**: Write comprehensive tests for your platform

## Integration with Existing Systems

This extension system allows seamless integration with:
- Existing NotifyHub deployments
- Third-party platforms and services
- Internal company messaging systems
- Custom notification channels

The architecture ensures that external platforms work identically to built-in platforms, providing a consistent developer experience.