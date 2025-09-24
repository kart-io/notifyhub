# SMS Platform Features

This example demonstrates the complete SMS multi-provider integration capabilities in NotifyHub's unified architecture.

## What You'll Learn

- Multiple SMS provider configurations (Twilio, Aliyun, Tencent, AWS SNS)
- Template-based SMS with variables
- Provider-specific features and optimizations
- International phone number handling
- SMS best practices and cost optimization

## SMS Providers

### 1. Twilio (Global)

Best for international reach and reliability:

```go
hub, err := notifyhub.NewHub(
    sms.WithSMSTwilio("your-twilio-api-key", "+1234567890",
        sms.WithSMSTimeout(30*time.Second),
    ),
)
```

**Features:**
- Global coverage
- E.164 phone format required
- Premium delivery rates
- Rich API features

### 2. Aliyun (阿里云) - China/APAC

Optimized for Chinese and Asia Pacific markets:

```go
hub, err := notifyhub.NewHub(
    sms.WithSMSAliyun("your-aliyun-api-key", "+8612345678901",
        sms.WithSMSAPISecret("api-secret"),
        sms.WithSMSSignName("阿里云"),
        sms.WithSMSTemplate("SMS_123456789"),
    ),
)
```

**Features:**
- Strong in China
- Template-based messaging
- Signature name required
- Cost-effective

### 3. Tencent (腾讯云) - China

Competitive option for Chinese market:

```go
hub, err := notifyhub.NewHub(
    sms.WithSMSTencent("your-tencent-api-key", "+8687654321098",
        sms.WithSMSAPISecret("api-secret"),
        sms.WithSMSRegion("ap-beijing"),
    ),
)
```

**Features:**
- Good domestic delivery
- WeChat ecosystem integration
- Template and signature support

### 4. AWS SNS

Part of AWS ecosystem:

```go
hub, err := notifyhub.NewHub(
    sms.WithSMSAWS("your-access-key", "+1987654321",
        sms.WithSMSAPISecret("secret-access-key"),
        sms.WithSMSRegion("us-east-1"),
    ),
)
```

**Features:**
- AWS ecosystem integration
- Pay-as-you-go pricing
- Global infrastructure

## SMS Message Types

### Basic Text Messages

Simple SMS notifications:

```go
msg := notifyhub.NewMessage("System Alert").
    WithBody("Database backup completed successfully.").
    ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
    Build()
```

### Template-Based Messages

Dynamic content with variable substitution:

```go
msg := notifyhub.NewMessage("Account Alert").
    WithPlatformData(map[string]interface{}{
        "sms_template": "ALERT: Login from {{location}} at {{time}}. If not you, contact support.",
        "sms_variables": map[string]interface{}{
            "location": "New York",
            "time":     time.Now().Format("15:04"),
        },
    }).
    Build()
```

### Verification Codes

Common pattern for 2FA and verification:

```go
msg := notifyhub.NewMessage("Verification Code").
    WithPlatformData(map[string]interface{}{
        "sms_template": "Your verification code is {{code}}. Valid for {{duration}} minutes.",
        "sms_variables": map[string]interface{}{
            "code":     "123456",
            "duration": "5",
        },
    }).
    Build()
```

## Common Use Cases

### Authentication (2FA)

```go
msg := notifyhub.NewMessage("2FA Code").
    WithBody("Your login code is 789012. Do not share this code.").
    Build()
```

### Transaction Alerts

```go
msg := notifyhub.NewAlert("Transaction Alert").
    WithBody("$250.00 charged to card ending in 1234. Not you? Call us.").
    Build()
```

### Delivery Updates

```go
msg := notifyhub.NewMessage("Package Update").
    WithBody("Your package #PKG123 is out for delivery. Expected: 2-4 PM.").
    Build()
```

### Emergency Alerts

```go
msg := notifyhub.NewUrgent("EMERGENCY").
    WithBody("Server room temperature critical. Immediate attention required!").
    Build()
```

### Appointment Reminders

```go
msg := notifyhub.NewMessage("Appointment Reminder").
    WithBody("Reminder: Doctor appointment tomorrow at 3:00 PM. Reply CONFIRM.").
    Build()
```

## Configuration Options

### Available Options

- `WithSMSAPISecret(secret)` - API secret for providers requiring it
- `WithSMSRegion(region)` - AWS region or provider-specific region
- `WithSMSTimeout(duration)` - Custom timeout for SMS operations
- `WithSMSTemplate(template)` - Template ID for template-based providers
- `WithSMSSignName(name)` - Signature name for Chinese providers

### Provider-Specific Configurations

#### Twilio
```go
sms.WithSMSTwilio("api-key", "+1234567890",
    sms.WithSMSTimeout(30*time.Second),
)
```

#### Aliyun
```go
sms.WithSMSAliyun("api-key", "+8612345678901",
    sms.WithSMSAPISecret("secret"),
    sms.WithSMSSignName("Company Name"),
    sms.WithSMSTemplate("SMS_123456"),
)
```

#### AWS SNS
```go
sms.WithSMSAWS("access-key", "+1987654321",
    sms.WithSMSAPISecret("secret-key"),
    sms.WithSMSRegion("us-east-1"),
)
```

## Phone Number Formats

### E.164 Format (Recommended)

Use international E.164 format for all providers:

```
+[country code][area code][local number]
```

**Examples:**
- US: `+1234567890`
- UK: `+447700900123`
- China: `+8613812345678`

### Validation

All providers validate phone numbers:

```go
// Valid formats
"+1234567890"    // US
"+447700900123"  // UK
"+8613812345678" // China

// Invalid formats
"1234567890"     // Missing country code
"+1-234-567-890" // Contains dashes
"invalid"        // Not a number
```

## Best Practices

### Message Length

- **Keep under 160 characters** for single SMS
- Longer messages are split into multiple parts (additional cost)
- Use templates to optimize length

### Security

```go
// ✅ Good: Use templates for sensitive data
"Your code is {{code}}"

// ❌ Bad: Include sensitive data directly
"Your credit card 4532-1234-5678-9012 was charged"
```

### Timing Considerations

- Respect recipient time zones
- Avoid sending during night hours (10 PM - 8 AM)
- Consider business vs personal numbers

### Cost Optimization

1. **Choose the right provider for your region**
2. **Use templates to reduce message length**
3. **Implement opt-out mechanisms**
4. **Monitor usage and costs**

### Error Handling

```go
receipt, err := hub.Send(ctx, message)
if err != nil {
    log.Printf("SMS send failed: %v", err)
    return
}

for _, result := range receipt.Results {
    if !result.Success {
        log.Printf("SMS to %s failed: %s", result.Target, result.Error)
    }
}
```

## Provider Selection Guide

### Choose Twilio When:
- Need global reach
- Require premium delivery rates
- Want rich API features
- Budget allows for higher costs

### Choose Aliyun When:
- Primary audience in China
- Need cost-effective solution
- Can work with template restrictions
- Want local Chinese support

### Choose Tencent When:
- Targeting Chinese market
- Need WeChat ecosystem integration
- Want competitive pricing
- Prefer local Chinese provider

### Choose AWS SNS When:
- Already using AWS services
- Need pay-as-you-go pricing
- Want infrastructure integration
- Require global AWS regions

## Running the Example

```bash
cd examples/platforms/sms
go run main.go
```

## Configuration Setup

1. **Choose SMS Provider**: Based on your target audience and requirements
2. **Get API Credentials**: Sign up and get API keys/secrets
3. **Configure Phone Numbers**: Use E.164 format
4. **Set Up Templates**: If using template-based providers
5. **Test with Real Numbers**: Verify delivery and formatting

## Common Errors and Solutions

### Authentication Errors
- **Cause**: Invalid API keys or secrets
- **Solution**: Verify credentials with provider

### Phone Number Format Errors
- **Cause**: Invalid number format
- **Solution**: Use E.164 format (+country code + number)

### Template Errors (Aliyun/Tencent)
- **Cause**: Invalid template ID or missing variables
- **Solution**: Verify template exists and provide all variables

### Rate Limiting
- **Cause**: Too many messages sent too quickly
- **Solution**: Implement delays or use rate limiting

## Platform Capabilities

The SMS platform supports:

✅ **Multi-Provider** - Twilio, Aliyun, Tencent, AWS SNS
✅ **Templates** - Variable substitution and reusable templates
✅ **Validation** - Phone number format validation
✅ **Regional Optimization** - Choose best provider per region
✅ **Cost Control** - Monitor usage and optimize costs
✅ **Error Handling** - Comprehensive error reporting

## Next Steps

- [Feishu Platform](../feishu/) - Rich messaging features
- [Email Platform](../email/) - SMTP email integration
- [Unified Demo](../unified-demo/) - All platforms together