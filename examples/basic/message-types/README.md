# Message Types and Builder Patterns

This example demonstrates the flexible message creation system in NotifyHub, including different message types, builder patterns, and best practices.

## What You'll Learn

- Different message priority levels
- Fluent API builder patterns
- Message formats and content types
- Metadata and variables usage
- Platform-specific data handling

## Message Types

### Priority Levels

NotifyHub provides three built-in priority levels:

```go
// Normal priority (default)
normal := notifyhub.NewMessage("Regular Update").
    WithBody("System maintenance completed.").
    Build()

// High priority
alert := notifyhub.NewAlert("Database Warning").
    WithBody("Connection pool running low.").
    Build()

// Highest priority
urgent := notifyhub.NewUrgent("CRITICAL FAILURE").
    WithBody("Payment system down!").
    Build()
```

### Builder Pattern Features

#### 1. Basic Structure
```go
message := notifyhub.NewMessage("Title").
    WithBody("Content").
    ToTarget(target).
    Build()
```

#### 2. Metadata
Add structured data for processing and routing:
```go
message := notifyhub.NewMessage("Deployment").
    WithMetadata("version", "2.1.0").
    WithMetadata("environment", "production").
    Build()
```

#### 3. Variables
Template variables for dynamic content:
```go
message := notifyhub.NewMessage("User Action").
    WithBody("User {{user}} performed {{action}}").
    WithVariable("user", "Alice").
    WithVariable("action", "login").
    Build()
```

#### 4. Platform-Specific Data
Special features for individual platforms:
```go
message := notifyhub.NewMessage("Rich Content").
    WithPlatformData(map[string]interface{}{
        "feishu_mention_all": true,
        "email_priority": "high",
    }).
    Build()
```

## Message Formats

### Supported Formats

- **text** (default) - Plain text content
- **markdown** - Markdown formatting
- **html** - HTML content (primarily for email)

```go
// Markdown message
markdown := notifyhub.NewMessage("Formatted").
    WithBody("**Bold** and *italic* text").
    WithFormat("markdown").
    Build()

// HTML message
html := notifyhub.NewMessage("HTML Email").
    WithBody("<h1>Title</h1><p>Content</p>").
    WithFormat("html").
    Build()
```

## Advanced Patterns

### Multi-Target Messages

Send to multiple targets with a single message:

```go
message := notifyhub.NewMessage("Broadcast").
    ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
    ToTarget(notifyhub.NewTarget("group", "alerts", "feishu")).
    ToTarget(notifyhub.NewTarget("user", "user123", "feishu")).
    Build()
```

### Conditional Building

Build messages dynamically based on conditions:

```go
builder := notifyhub.NewMessage("Dynamic Message").
    WithBody("Base content")

if isUrgent {
    builder = builder.WithMetadata("priority", "urgent")
}

if includeUser {
    builder = builder.WithVariable("user", currentUser)
}

message := builder.Build()
```

### Message Templates

Create reusable message templates:

```go
func createIncidentMessage(incidentID, service, severity string) *notifyhub.Message {
    return notifyhub.NewAlert("Production Incident").
        WithBody("Incident {{id}} in {{service}} - Severity: {{severity}}").
        WithVariable("id", incidentID).
        WithVariable("service", service).
        WithVariable("severity", severity).
        WithMetadata("incident_id", incidentID).
        WithMetadata("service", service).
        Build()
}
```

## Best Practices

### 1. Choose Appropriate Priorities

- **Normal**: Regular updates, logs, routine notifications
- **Alert**: Issues requiring attention, warnings, important updates
- **Urgent**: Critical problems, system failures, emergency alerts

### 2. Use Metadata Effectively

Include relevant context for processing:
```go
message := notifyhub.NewMessage("Error").
    WithMetadata("service", "payment-gateway").
    WithMetadata("environment", "production").
    WithMetadata("error_code", "E001").
    WithMetadata("timestamp", time.Now().Unix()).
    Build()
```

### 3. Leverage Variables for Reusability

Create templates that work across different contexts:
```go
template := notifyhub.NewMessage("User Alert").
    WithBody("{{action}} performed by {{user}} at {{time}}").
    WithVariable("action", action).
    WithVariable("user", username).
    WithVariable("time", timestamp).
    Build()
```

### 4. Platform-Specific Enhancements

Use platform data for rich features:
```go
// Feishu with mentions
feishuMsg := notifyhub.NewMessage("Team Alert").
    WithPlatformData(map[string]interface{}{
        "feishu_mention_all": true,
        "feishu_mentions": mentions,
    }).
    Build()

// Email with formatting
emailMsg := notifyhub.NewMessage("Report").
    WithFormat("html").
    WithPlatformData(map[string]interface{}{
        "email_cc": []string{"manager@company.com"},
        "email_priority": "high",
    }).
    Build()
```

## Running the Example

```bash
cd examples/basic/message-types
go run main.go
```

## Builder Pattern Benefits

✅ **Fluent API** - Readable, chainable method calls
✅ **Type Safety** - Compile-time validation
✅ **Flexibility** - Add complexity as needed
✅ **Reusability** - Template-based message creation
✅ **Discoverability** - IDE autocomplete support

## Next Steps

- [Error Handling Example](../error-handling/) - Robust error handling patterns
- [Multi-Platform Example](../multi-platform/) - Using multiple platforms
- [Platform Examples](../../platforms/) - Platform-specific features