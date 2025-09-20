# NotifyHub Examples

This directory contains comprehensive examples demonstrating various NotifyHub features and usage patterns, from basic platform integration to advanced enterprise scenarios.

## Quick Start

Choose an example based on your needs:

- **New to NotifyHub?** Start with [Unified API Example](unified_api/)
- **Platform-specific integration?** See platform examples below
- **Production deployment?** Check [Advanced Usage](advanced_usage/) and [Error Handling](error_handling/)
- **Custom middleware?** See [Middleware Usage](middleware_usage/)

## Examples Overview

### 📚 Basic Examples

#### [Unified API Example](unified_api/)
**What:** Complete introduction to NotifyHub with multiple platforms
**Use Case:** Getting started, understanding core concepts
**Features:** Basic sending, multiple platforms, simple configuration
**Complexity:** ⭐⭐☆☆☆

#### [Feishu Basic Example](feishu_basic/)
**What:** Feishu-specific notification examples
**Use Case:** Feishu bot integration, team notifications
**Features:** Text, markdown, cards, alerts
**Complexity:** ⭐⭐☆☆☆

#### [Email Basic Example](email_basic/)
**What:** Email notification examples with various formats
**Use Case:** Email campaigns, alerts, reports
**Features:** Plain text, HTML, SMTP configuration
**Complexity:** ⭐⭐☆☆☆

### 🔧 Intermediate Examples

#### [Error Handling Example](error_handling/)
**What:** Comprehensive error handling strategies
**Use Case:** Production systems, reliability
**Features:** Error categorization, retry logic, circuit breakers
**Complexity:** ⭐⭐⭐☆☆

#### [Middleware Usage Example](middleware_usage/)
**What:** Using and creating custom middleware
**Use Case:** Cross-cutting concerns, routing, rate limiting
**Features:** Built-in middleware, custom middleware, chains
**Complexity:** ⭐⭐⭐☆☆

### 🚀 Advanced Examples

#### [Advanced Usage Example](advanced_usage/)
**What:** Enterprise-grade patterns and features
**Use Case:** Large-scale deployments, complex requirements
**Features:** Templates, batch processing, scheduling, monitoring, multi-tenancy
**Complexity:** ⭐⭐⭐⭐⭐

#### [Unified Errors Example](unified_errors_example.go)
**What:** Standardized error handling across platforms
**Use Case:** Consistent error management
**Features:** Error mapping, categorization, retry decisions
**Complexity:** ⭐⭐⭐☆☆

#### [External Redis Queue Example](external_redis_queue.go)
**What:** Redis queue integration for scalability
**Use Case:** High-throughput scenarios
**Features:** Redis Streams, distributed processing
**Complexity:** ⭐⭐⭐⭐☆

## Example Matrix

| Example | Feishu | Email | SMS | Error Handling | Middleware | Templates | Monitoring |
|---------|--------|-------|-----|----------------|------------|-----------|------------|
| Unified API | ✅ | ✅ | ✅ | Basic | Basic | ❌ | ❌ |
| Feishu Basic | ✅ | ❌ | ❌ | Basic | ❌ | ❌ | ❌ |
| Email Basic | ❌ | ✅ | ❌ | Basic | ❌ | ❌ | ❌ |
| Error Handling | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Middleware Usage | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ |
| Advanced Usage | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

## Running Examples

### Prerequisites

1. **Go Environment**: Go 1.19 or later
2. **External Services**: Configure based on example needs

### Environment Setup

Each example may require different environment variables:

```bash
# Feishu (for Feishu examples)
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
export FEISHU_SECRET="your-secret"

# Email (for email examples)
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT="587"
export SMTP_USER="your-email@gmail.com"
export SMTP_PASS="your-app-password"
export SMTP_FROM="noreply@yourcompany.com"
export EMAIL_TO="recipient@example.com"

# SMS (for SMS examples)
export SMS_PROVIDER="twilio"
export SMS_ACCOUNT_SID="your-account-sid"
export SMS_AUTH_TOKEN="your-auth-token"
export SMS_FROM="+1234567890"
export SMS_TO="+0987654321"

# Redis (for Redis examples)
export REDIS_URL="redis://localhost:6379"
```

### Running Individual Examples

```bash
# Basic examples
cd examples/unified_api && go run main.go
cd examples/feishu_basic && go run main.go
cd examples/email_basic && go run main.go

# Intermediate examples
cd examples/error_handling && go run main.go
cd examples/middleware_usage && go run main.go

# Advanced examples
cd examples/advanced_usage && go run main.go

# Standalone examples
go run examples/unified_errors_example.go
go run examples/external_redis_queue.go
```

## Example Walkthrough

### 1. Start with Unified API

The [Unified API Example](unified_api/) is the best starting point:

```bash
cd examples/unified_api
go run main.go
```

This example demonstrates:
- Basic NotifyHub setup
- Multiple platform configuration
- Different message formats
- Simple error handling

### 2. Explore Platform-Specific Features

Choose your platform:

```bash
# For Feishu integration
cd examples/feishu_basic

# For Email integration
cd examples/email_basic
```

Learn platform-specific features like:
- Feishu cards and webhooks
- Email HTML templates and SMTP
- Platform-specific error handling

### 3. Add Production Reliability

Move to production-ready patterns:

```bash
cd examples/error_handling
go run main.go
```

Learn about:
- Error categorization and retry strategies
- Circuit breaker patterns
- Graceful degradation

### 4. Implement Middleware

Add cross-cutting concerns:

```bash
cd examples/middleware_usage
go run main.go
```

Discover:
- Built-in middleware (routing, rate limiting, retry)
- Custom middleware creation
- Middleware chain configuration

### 5. Scale to Enterprise

For large-scale deployments:

```bash
cd examples/advanced_usage
go run main.go
```

Explore:
- Template systems
- Batch processing
- Message scheduling
- Multi-tenancy
- A/B testing
- High availability

## Common Patterns

### Configuration Pattern

```go
// Environment-based configuration
cfg := config.New()
cfg.AddFeishu(os.Getenv("FEISHU_WEBHOOK_URL"), os.Getenv("FEISHU_SECRET"))
cfg.AddEmail(/* email config from env */)

// Create hub
hub, err := notifyhub.New(cfg, &notifyhub.Options{
    Logger: logger.New(),
})
```

### Error Handling Pattern

```go
results, err := hub.Send(ctx, msg, targets)
if err != nil {
    return fmt.Errorf("send failed: %w", err)
}

// Check individual results
for _, result := range results.Results {
    if result.Error != nil {
        log.Printf("Failed to send to %s: %v", result.Target.Value, result.Error)

        // Determine if retryable
        if errors.IsRetryableError(result.Error) {
            // Schedule retry
        }
    }
}
```

### Template Pattern

```go
// Use templates for consistent messaging
msg := hub.NewMessage().
    SetTitle("{{.AlertType}}: {{.Component}}").
    SetBody(alertTemplate).
    SetFormat(message.FormatMarkdown).
    AddMetadata("severity", "high")
```

## Testing Examples

Run example tests:

```bash
# Test specific example
cd examples/error_handling
go test -v

# Test all examples
find examples -name "*_test.go" -exec go test -v {} \;

# Integration tests (requires external services)
INTEGRATION=true go test -v ./examples/...
```

## Troubleshooting

### Common Issues

1. **Missing Environment Variables**
   ```
   Error: FEISHU_WEBHOOK_URL environment variable is required
   ```
   Solution: Set required environment variables for your example

2. **Network Connectivity**
   ```
   Error: connection refused
   ```
   Solution: Check firewall, VPN, or proxy settings

3. **Authentication Failures**
   ```
   Error: 401 Unauthorized
   ```
   Solution: Verify API keys, tokens, and credentials

4. **Rate Limiting**
   ```
   Error: 429 Too Many Requests
   ```
   Solution: Implement rate limiting or reduce request frequency

### Debug Mode

Enable debug logging:

```bash
export DEBUG=true
export LOG_LEVEL=debug
go run main.go
```

### Getting Help

1. Check example README files for specific setup instructions
2. Review the main project documentation
3. Check the troubleshooting section in individual examples
4. Open an issue if you find bugs or need clarification

## Contributing

Want to add an example? Please:

1. Follow the existing structure and naming conventions
2. Include comprehensive README with setup instructions
3. Add error handling and logging
4. Include troubleshooting section
5. Test with different configurations
6. Update this main examples README

## Example Roadmap

Planned examples:

- [ ] **SMS Integration Example** - SMS provider integration
- [ ] **Slack Integration Example** - Slack bot and webhook usage
- [ ] **Monitoring & Observability** - Metrics, tracing, logging
- [ ] **Performance Testing** - Load testing and benchmarks
- [ ] **Docker Deployment** - Containerized deployment examples
- [ ] **Kubernetes Integration** - K8s operators and helm charts
- [ ] **Webhook Handlers** - Receiving webhooks from platforms
- [ ] **Multi-Region Setup** - Geographic distribution patterns

Each example is designed to be self-contained and runnable, providing practical demonstrations of NotifyHub capabilities for real-world use cases.