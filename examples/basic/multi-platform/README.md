# Multi-Platform Integration

This example demonstrates how to use multiple notification platforms together with the unified architecture.

## What You'll Learn

- Configuring multiple platforms in a single hub
- Broadcasting messages to all platforms
- Platform-specific messaging features
- Priority-based routing strategies

## Key Features

### 1. Unified Configuration

All platforms follow the same configuration pattern:

```go
hub, err := notifyhub.NewHub(
    feishu.WithFeishu("webhook", feishu.WithFeishuSecret("secret")),
    email.WithEmail("host", 587, "from", email.WithEmailAuth("user", "pass")),
    sms.WithSMSTwilio("key", "from", sms.WithSMSTimeout(30*time.Second)),
)
```

### 2. Cross-Platform Broadcasting

Send the same message to multiple platforms with a single call:

```go
message := notifyhub.NewMessage("Broadcast").
    WithBody("Message for all platforms").
    ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
    ToTarget(notifyhub.NewTarget("email", "user@example.com", "email")).
    ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
    Build()
```

### 3. Platform-Specific Features

Each platform can use its unique features while maintaining unified APIs:

- **Feishu**: Rich formatting, mentions, cards
- **Email**: HTML content, CC/BCC, priorities
- **SMS**: Template variables, provider-specific features

### 4. Priority-Based Routing

Route messages to different platforms based on priority:

- **Normal**: Internal team only (Feishu)
- **Alert**: Team + stakeholders (Feishu + Email)
- **Urgent**: All channels (Feishu + Email + SMS)

## Running the Example

```bash
cd examples/basic/multi-platform
go run main.go
```

## Configuration

Update the configuration with your actual credentials:

```go
// Feishu webhook
feishu.WithFeishu("https://your-feishu-webhook", feishu.WithFeishuSecret("secret"))

// SMTP email
email.WithEmail("smtp.your-provider.com", 587, "notifications@yourcompany.com",
    email.WithEmailAuth("username", "password"))

// SMS provider (Twilio example)
sms.WithSMSTwilio("your-api-key", "+your-phone-number")
```

## Use Cases

### 1. System Monitoring

- Normal logs → Feishu team channel
- Warnings → Feishu + Email to on-call
- Critical alerts → All platforms including SMS

### 2. Business Updates

- Daily reports → Email to stakeholders
- Urgent issues → Feishu + SMS to team leads
- Customer alerts → Email + SMS to support team

### 3. DevOps Pipeline

- Build success → Feishu developer channel
- Build failures → Feishu + Email to dev team
- Production issues → All platforms to incident response

## Architecture Benefits

The unified architecture provides:

✅ **Consistent APIs** - Same patterns across all platforms
✅ **Easy Scaling** - Add new platforms without refactoring
✅ **Platform Independence** - Change platforms without code changes
✅ **Rich Features** - Access to platform-specific capabilities
✅ **Unified Error Handling** - Consistent response format

## Next Steps

- [Message Types Example](../message-types/) - Advanced message configuration
- [Platform Examples](../../platforms/) - Deep dive into platform features
- [Error Handling Example](../error-handling/) - Robust error handling patterns