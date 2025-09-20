# NotifyHub API Migration Guide

This guide helps you migrate from the old API versions to the new unified NotifyHub API.

## Overview

The unified API combines the best features from v1 and v2 APIs into a single, cohesive interface that provides:

- **Unified Entry Point**: Single client with fluent builders
- **Type Safety**: Platform-specific builders with compile-time validation
- **Smart Target Detection**: Automatic target type detection from strings
- **Enhanced Error Handling**: Specific error types with detailed context
- **Modern Configuration**: Environment-based config with fluent builders
- **Backward Compatibility**: V1 compatibility layer for existing code

## Migration Paths

### Option 1: Full Migration to Unified API (Recommended)

This is the recommended approach for new projects and major refactors.

#### Before (V1 API)

```go
// V1 Configuration
cfg := config.New()
cfg.AddFeishu("webhook-url", "secret")
cfg.AddEmail("smtp.example.com", 587, "user", "pass")

// V1 Client Creation
hub, err := api.New(cfg, &api.Options{Logger: logger.New()})
if err != nil {
    log.Fatal(err)
}
defer hub.Shutdown(context.Background())

// V1 Message Sending
msg := hub.NewMessage().
    SetTitle("Alert").
    SetBody("System is down")

targets := []sending.Target{
    sending.NewTarget(sending.TargetTypeEmail, "admin@example.com", "email"),
}

results, err := hub.Send(context.Background(), msg, targets)
```

#### After (Unified API)

```go
// Unified Configuration
cfg := notifyhub.NewConfig().
    WithFeishu("webhook-url", "secret").
    WithEmail("smtp.example.com", 587, "user", "pass", "from@example.com").
    LoadFromEnv()

// Unified Client Creation
client, err := notifyhub.New(cfg.Config)
if err != nil {
    log.Fatal(err)
}
defer client.Shutdown(context.Background())

// Unified Message Sending
result, err := client.Send().
    Title("Alert").
    Body("System is down").
    To("admin@example.com").
    Execute(context.Background())
```

### Option 2: Gradual Migration with V1 Compatibility

For existing codebases that need gradual migration.

```go
// Use V1 compatibility layer with existing config
v1Client, err := notifyhub.NewV1Compat(existingV1Config)
if err != nil {
    log.Fatal(err)
}
defer v1Client.Shutdown(context.Background())

// Existing V1 code continues to work
msg := v1Client.NewMessage()
msg.SetTitle("V1 Compatible Message")
msg.SetBody("This uses v1 API style")

targets := []sending.Target{
    sending.NewTarget(sending.TargetTypeEmail, "admin@example.com", "email"),
}

results, err := v1Client.Send(context.Background(), msg, targets)
```

## Key API Changes

### 1. Configuration

**Before:**

```go
cfg := config.New()
cfg.AddFeishu("webhook", "secret")
cfg.AddEmail("host", 587, "user", "pass")
```

**After:**

```go
cfg := notifyhub.NewConfig().
    WithFeishu("webhook", "secret").
    WithEmail("host", 587, "user", "pass", "from@example.com").
    WithQueue("memory", 1000, 4).
    WithRateLimit(100, 10).
    LoadFromEnv()
```

### 2. Client Creation

**Before:**

```go
hub, err := api.New(cfg, &api.Options{Logger: logger.New()})
```

**After:**

```go
client, err := notifyhub.New(cfg.Config)
```

### 3. Message Sending

**Before:**

```go
msg := hub.NewMessage().SetTitle("Alert").SetBody("System down")
targets := []sending.Target{
    sending.NewTarget(sending.TargetTypeEmail, "admin@example.com", "email"),
}
results, err := hub.Send(ctx, msg, targets)
```

**After:**

```go
// Simple approach with smart targets
result, err := client.Send().
    Title("Alert").
    Body("System down").
    To("admin@example.com").
    Execute(ctx)

// Type-safe approach for specific platforms
result, err := client.Email().
    Title("Alert").
    Body("System down").
    To("admin@example.com").
    Send(ctx)
```

### 4. Platform-Specific Features

**Before:**

```go
// Platform-specific features were harder to access
platformBuilder, err := messageBuilder.Platform("feishu")
// Complex type assertions and platform-specific code
```

**After:**

```go
// Direct access to type-safe platform builders
result, err := client.Feishu().
    AlertCard("Critical Error", "Database offline", types.AlertLevelError).
    ToGroup("ops-team").
    AtUser("oncall").
    AddButton("View Dashboard", "https://dashboard.com").
    Send(ctx)
```

### 5. Error Handling

**Before:**

```go
if err != nil {
    log.Printf("Send failed: %v", err)
}
```

**After:**

```go
if err != nil {
    switch e := err.(type) {
    case *types.ValidationError:
        log.Printf("Validation error in %s: %s", e.Field, e.Message)
    case *types.SendError:
        log.Printf("Send error for message %s: %v", e.MessageID, e.Cause)
    case *types.RateLimitError:
        log.Printf("Rate limited on %s, retry after %d seconds", e.Platform, e.RetryAfter)
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

## Feature Mapping

| V1 Feature | Unified API Equivalent | Notes |
|------------|------------------------|-------|
| `api.New()` | `notifyhub.New()` | Simplified creation |
| `hub.NewMessage()` | `client.Send()` | Fluent builder pattern |
| Target expressions | `client.Send().To()` | Smart target detection |
| Platform adapters | `client.Email()`, `client.Feishu()`, etc. | Type-safe builders |
| Middleware | Built-in (rate limiting, retry) | Configured via config |
| Analysis | `client.Metrics()`, `client.Health()` | Simplified monitoring |

## Smart Target Detection

The unified API can automatically detect target types:

```go
client.Send().
    To(
        "user@example.com",    // → Email
        "@john",               // → User mention
        "#alerts",             // → Channel
        "+1234567890",         // → SMS
        "https://webhook.com", // → Webhook
    ).
    Execute(ctx)
```

## Type-Safe Platform Features

### Email

```go
client.Email().
    Title("Report").
    HTMLBody("<h1>Monthly Report</h1>").
    To("manager@company.com").
    CC("team@company.com").
    EnableTracking().
    Attach("report.pdf", pdfData).
    Send(ctx)
```

### Feishu

```go
client.Feishu().
    AlertCard("System Alert", "CPU usage high", types.AlertLevelWarning).
    ToGroup("ops").
    AtUser("oncall").
    AddButton("Dashboard", "https://monitor.com").
    Send(ctx)
```

### Slack

```go
client.Slack().
    HeaderBlock("Deployment").
    SimpleBlock("✅ v2.1.0 deployed").
    AddField("Environment", "Production", true).
    AddButton("Logs", "logs", "https://logs.com").
    ToChannel("deploys").
    Send(ctx)
```

## Environment Variables

The unified API supports environment-based configuration:

```bash
# Email configuration
NOTIFYHUB_EMAIL_HOST=smtp.gmail.com
NOTIFYHUB_EMAIL_PORT=587
NOTIFYHUB_EMAIL_USERNAME=user@gmail.com
NOTIFYHUB_EMAIL_PASSWORD=password
NOTIFYHUB_EMAIL_FROM=notifications@company.com

# Feishu configuration
NOTIFYHUB_FEISHU_WEBHOOK=https://open.feishu.cn/...
NOTIFYHUB_FEISHU_SECRET=your-secret

# Slack configuration
NOTIFYHUB_SLACK_TOKEN=xoxb-123456789

# Global settings
NOTIFYHUB_DEBUG=true
NOTIFYHUB_TIMEOUT=30s
```

Load with:

```go
cfg := notifyhub.NewConfig().LoadFromEnv()
```

## Testing

### V1 Compatibility Testing

```go
func TestV1Compatibility(t *testing.T) {
    // Create mock config for testing
    cfg := &config.Config{}
    // ... configure mock settings

    client, err := notifyhub.NewV1Compat(cfg)
    require.NoError(t, err)
    defer client.Shutdown(context.Background())

    // Test v1-style API
    msg := client.NewMessage()
    // ... rest of v1 test code
}
```

### Unified API Testing

```go
func TestUnifiedAPI(t *testing.T) {
    cfg := notifyhub.NewConfig().
        WithEmail("smtp.test.com", 587, "test", "pass", "test@example.com")

    client, err := notifyhub.New(cfg.Config)
    require.NoError(t, err)
    defer client.Shutdown(context.Background())

    // Test unified API
    result, err := client.Send().
        Title("Test").
        Body("Test message").
        To("test@example.com").
        Execute(context.Background())

    require.NoError(t, err)
    assert.NotEmpty(t, result.MessageID)
}
```

## Best Practices

1. **Use Type-Safe Builders**: Prefer `client.Email()`, `client.Feishu()` for platform-specific features
2. **Smart Targets for Simple Cases**: Use `client.Send().To()` for basic notifications
3. **Environment Configuration**: Use `LoadFromEnv()` for deployment flexibility
4. **Specific Error Handling**: Check for specific error types for better error handling
5. **Health Monitoring**: Use `client.Health()` and `client.Metrics()` for observability
6. **Graceful Shutdown**: Always call `client.Shutdown(ctx)` for cleanup

## Migration Checklist

- [ ] Update imports to use `github.com/kart-io/notifyhub`
- [ ] Replace config creation with `notifyhub.NewConfig()`
- [ ] Update client creation to use `notifyhub.New()`
- [ ] Convert message sending to use fluent builders
- [ ] Update error handling for specific error types
- [ ] Add environment variable support
- [ ] Update tests to use new API patterns
- [ ] Review and update platform-specific code
- [ ] Test health and metrics endpoints
- [ ] Verify graceful shutdown behavior

## Support

For migration questions or issues:

1. Check the [examples](examples/) directory for complete usage examples
2. Review the [API documentation](api/v2/README.md)
3. Run the unified API demo: `go run examples/unified_api/main.go`
