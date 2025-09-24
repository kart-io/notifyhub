# Unified Platform Demo - All Platforms Together

This comprehensive demo showcases ALL supported platforms working together in the unified architecture, demonstrating the complete solution to the original extensibility problem.

## What This Demo Proves

- **True External Extensibility** - External platforms work exactly like built-in ones
- **Unified Developer Experience** - Consistent APIs across all platforms
- **Cross-Platform Broadcasting** - Single message to multiple platforms
- **Platform-Specific Features** - Rich content unique to each platform
- **Graceful Error Handling** - Robust error management across platforms
- **Performance Excellence** - Efficient multi-platform operations

## Platforms Demonstrated

### Built-in Platforms (Core Library)
- **Feishu** - Team communication with interactive cards
- **Email** - SMTP with HTML content and CC/BCC
- **SMS** - Multi-provider SMS with templates (Twilio)

### External Platforms (Community)
- **Discord** - Rich embeds and webhooks (completely external)

## Key Architectural Achievements

### 1. Consistent Configuration Patterns

All platforms follow identical patterns:

```go
// Built-in platforms
feishu.WithFeishu("webhook", feishu.WithFeishuSecret("secret"))
email.WithEmail("host", 587, "from", email.WithEmailAuth("user", "pass"))
sms.WithSMSTwilio("key", "from", sms.WithSMSTimeout(30*time.Second))

// External platforms - same API quality!
discord.WithDiscord("webhook", discord.WithDiscordUsername("bot"))
```

### 2. Cross-Platform Broadcasting

Send one message to all platforms simultaneously:

```go
message := notifyhub.NewMessage("Broadcast Alert").
    WithBody("Critical system update for all channels").
    ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
    ToTarget(notifyhub.NewTarget("email", "admin@company.com", "email")).
    ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
    ToTarget(notifyhub.NewTarget("webhook", "", "discord")).
    Build()

receipt, err := hub.Send(ctx, message)
// Sent to 4 platforms with one call!
```

### 3. Priority-Based Routing

Route messages to different platform combinations based on priority:

```go
// Normal: Team channels only (Feishu + Discord)
normal := notifyhub.NewMessage("Daily Update").
    ToTarget(feishuTarget).
    ToTarget(discordTarget)

// Alert: Team + stakeholders (Feishu + Discord + Email)
alert := notifyhub.NewAlert("System Warning").
    ToTarget(feishuTarget).
    ToTarget(discordTarget).
    ToTarget(emailTarget)

// Urgent: All channels including SMS
urgent := notifyhub.NewUrgent("CRITICAL ISSUE").
    ToTarget(feishuTarget).
    ToTarget(discordTarget).
    ToTarget(emailTarget).
    ToTarget(smsTarget)
```

### 4. Platform-Specific Rich Content

Each platform maintains its unique capabilities:

#### Feishu Interactive Cards
```go
msg.WithPlatformData(map[string]interface{}{
    "feishu_card": map[string]interface{}{
        "header": map[string]interface{}{
            "title": map[string]interface{}{
                "tag":     "plain_text",
                "content": "System Status",
            },
            "template": "green",
        },
        "elements": []map[string]interface{}{
            {
                "tag": "action",
                "actions": []map[string]interface{}{
                    {
                        "tag": "button",
                        "text": map[string]interface{}{
                            "tag":     "plain_text",
                            "content": "View Dashboard",
                        },
                        "url": "https://monitor.example.com",
                    },
                },
            },
        },
    },
})
```

#### Email HTML Reports
```go
msg.WithBody(`
<!DOCTYPE html>
<html>
<head>
    <style>
        .header { background: linear-gradient(90deg, #4CAF50, #2196F3); }
        .platform { background: #f1f1f1; padding: 15px; }
        .status-online { color: #4CAF50; font-weight: bold; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Platform Status Report</h1>
    </div>
    <div class="platform">
        <h3>Feishu</h3>
        <p>Status: <span class="status-online">ONLINE</span></p>
    </div>
</body>
</html>
`).WithFormat("html")
```

#### Discord Rich Embeds
```go
msg.WithPlatformData(map[string]interface{}{
    "discord_embeds": []map[string]interface{}{
        {
            "title":       "Platform Success!",
            "description": "All platforms operational",
            "color":       0x00ff00,
            "fields": []map[string]interface{}{
                {
                    "name":   "Architecture",
                    "value":  "Built-in + External platforms",
                    "inline": false,
                },
            },
            "timestamp": time.Now().Format(time.RFC3339),
        },
    },
})
```

#### SMS Template Variables
```go
msg.WithPlatformData(map[string]interface{}{
    "sms_template": "🌟 {{platforms}} platforms ONLINE. Success: {{rate}}%. {{url}}",
    "sms_variables": map[string]interface{}{
        "platforms": "4",
        "rate":      "99.7",
        "url":       "bit.ly/status",
    },
})
```

## Use Cases Demonstrated

### 1. System Monitoring
- **Normal updates** → Team channels (Feishu + Discord)
- **Warnings** → Team + Email to ops
- **Critical alerts** → All channels including SMS

### 2. Business Communication
- **Announcements** → Team channels
- **Reports** → Email with HTML formatting
- **Urgent updates** → SMS for immediate attention

### 3. Incident Response
- **Detection** → Automated alerts to all platforms
- **Updates** → Coordinated communication across channels
- **Resolution** → Confirmation to all stakeholders

## Error Handling Excellence

The demo shows robust error handling:

```go
receipt, err := hub.Send(ctx, message)
if err != nil {
    // Critical error - all platforms failed
    log.Fatalf("All platforms failed: %v", err)
} else if receipt.Failed > 0 {
    // Partial failure - some platforms succeeded
    log.Warnf("Partial failure: %d/%d failed", receipt.Failed, receipt.Total)

    for _, result := range receipt.Results {
        if !result.Success {
            log.Errorf("Platform %s failed: %s", result.Platform, result.Error)
        }
    }
}
```

## Performance Metrics

The demo measures performance across all platforms:

- **Total hub time** - End-to-end operation
- **Per-platform time** - Individual platform performance
- **Overhead** - Framework efficiency
- **Average response time** - Cross-platform performance

Example output:
```
Performance test completed in 450ms
Per-platform performance:
   📱 feishu: 120ms
   📱 email: 250ms
   📱 sms: 1200ms
   📱 discord: 180ms

Summary:
   Total hub time: 450ms
   Platform time: 1750ms
   Overhead: 15ms
   Avg per platform: 437ms
```

## Architecture Problem: SOLVED ✅

This demo proves the unified architecture successfully solved the original problems:

### ❌ Original Problems
- External developers couldn't add platforms without modifying core
- Hardcoded platform functions in core library
- Inconsistent APIs between built-in and external platforms
- Violated open/closed principle

### ✅ Solutions Delivered
- **True External Extensibility** - Discord platform created without core changes
- **Unified APIs** - All platforms use identical configuration patterns
- **Auto-Registration** - Platforms register themselves via imports
- **Open/Closed Principle** - Open for extension, closed for modification

## Running the Demo

```bash
cd examples/platforms/unified-demo
go run main.go
```

## Key Takeaways

1. **External platforms are first-class citizens** - Same API quality as built-in
2. **Consistent developer experience** - All platforms follow same patterns
3. **Rich platform capabilities preserved** - Each platform keeps unique features
4. **Robust error handling** - Graceful degradation with partial failures
5. **Performance excellence** - Efficient multi-platform operations
6. **Community ecosystem enabled** - External developers can contribute platforms

## Impact

This unified architecture:

- **Solves the extensibility problem** completely
- **Enables community ecosystem** growth
- **Maintains excellent DX** (developer experience)
- **Preserves platform uniqueness** and rich features
- **Provides production-ready** error handling and performance

The original architecture limitations are now **completely resolved**! 🎉

## Next Steps

- [External Platform Development](../../external/) - Create your own platforms
- [Advanced Examples](../../advanced/) - Production patterns
- [Platform-Specific Features](../) - Deep dive into individual platforms