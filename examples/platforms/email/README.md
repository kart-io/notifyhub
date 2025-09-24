# Email Platform Features

This example demonstrates the complete Email SMTP integration capabilities in NotifyHub's unified architecture.

## What You'll Learn

- Different SMTP configuration options
- HTML and plain text email formats
- Email templates with variables
- Advanced features (CC, priority, multiple recipients)
- Popular SMTP provider configurations
- Local testing without network dependencies
- Troubleshooting network and authentication issues

## ⚠️ Network Requirements

**Important:** This demo requires network access to SMTP servers. If you encounter timeout errors:

1. **Use MailHog for local testing** (recommended for development):
   ```bash
   # Install MailHog
   brew install mailhog

   # Run MailHog
   mailhog

   # Run local test
   go run test_local.go

   # View emails at: http://localhost:8025
   ```

2. **Check network connectivity**:
   ```bash
   nc -zv smtp.gmail.com 587
   ```

3. **See [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)** for detailed solutions

## SMTP Configurations

### 1. Basic SMTP (No Authentication)

For simple SMTP servers that don't require authentication:

```go
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.example.com", 25, "notifications@company.com"),
)
```

### 2. Authenticated SMTP with TLS

Most modern email providers require authentication:

```go
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.gmail.com", 587, "notifications@company.com",
        email.WithEmailAuth("username", "password"),
        email.WithEmailTLS(true),
        email.WithEmailTimeout(45*time.Second),
    ),
)
```

### 3. SSL Configuration

For SMTP servers using SSL instead of TLS:

```go
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.secure.com", 465, "secure@company.com",
        email.WithEmailAuth("user", "pass"),
        email.WithEmailSSL(true),
        email.WithEmailTLS(false),
    ),
)
```

## Email Content Types

### Plain Text Emails

Simple text-based notifications:

```go
msg := notifyhub.NewMessage("System Alert").
    WithBody("Database connection restored.\nAll systems operational.").
    ToTarget(notifyhub.NewTarget("email", "admin@company.com", "email")).
    Build()
```

### HTML Emails

Rich formatted emails with CSS styling:

```go
msg := notifyhub.NewMessage("Daily Report").
    WithBody(`
<!DOCTYPE html>
<html>
<head>
    <style>
        .header { background-color: #4CAF50; color: white; padding: 20px; }
        .metric { background-color: #f1f1f1; padding: 10px; margin: 5px 0; }
        .success { color: #4CAF50; }
    </style>
</head>
<body>
    <div class="header">
        <h1>📊 System Report</h1>
    </div>
    <div class="metric">
        <strong>Uptime:</strong> <span class="success">99.9%</span>
    </div>
</body>
</html>
    `).
    WithFormat("html").
    Build()
```

### Template Emails with Variables

Dynamic content using variable substitution:

```go
msg := notifyhub.NewMessage("Welcome {{user_name}}!").
    WithBody(`Hello {{user_name}},

Welcome to our platform! Your account details:
• Username: {{username}}
• Email: {{email}}
• Registration: {{reg_date}}

Visit {{login_url}} to get started.

Best regards,
The {{company}} Team`).
    WithVariable("user_name", "Alice Smith").
    WithVariable("username", "alice.smith").
    WithVariable("email", "alice@example.com").
    WithVariable("reg_date", time.Now().Format("2006-01-02")).
    WithVariable("login_url", "https://app.company.com/login").
    WithVariable("company", "NotifyHub").
    Build()
```

## Advanced Features

### CC and Priority

Add CC recipients and set email priority:

```go
msg := notifyhub.NewAlert("Security Review").
    WithBody("Please review the monthly security report.").
    WithPlatformData(map[string]interface{}{
        "email_cc":       []string{"security@company.com", "manager@company.com"},
        "email_priority": "high", // high, normal, low
    }).
    Build()
```

### Multiple Recipients

Send to multiple email addresses:

```go
msg := notifyhub.NewMessage("Team Announcement").
    WithBody("Important team update...").
    ToTarget(notifyhub.NewTarget("email", "john@company.com", "email")).
    ToTarget(notifyhub.NewTarget("email", "jane@company.com", "email")).
    ToTarget(notifyhub.NewTarget("email", "bob@company.com", "email")).
    Build()
```

## Popular SMTP Providers

### Gmail

```go
email.WithEmail("smtp.gmail.com", 587, "your-email@gmail.com",
    email.WithEmailAuth("your-email@gmail.com", "app-password"),
    email.WithEmailTLS(true),
)
```

**Note:** Use App Passwords, not regular passwords for Gmail.

### Outlook/Hotmail

```go
email.WithEmail("smtp-mail.outlook.com", 587, "your-email@outlook.com",
    email.WithEmailAuth("your-email@outlook.com", "password"),
    email.WithEmailTLS(true),
)
```

### SendGrid

```go
email.WithEmail("smtp.sendgrid.net", 587, "noreply@yourcompany.com",
    email.WithEmailAuth("apikey", "your-sendgrid-api-key"),
    email.WithEmailTLS(true),
)
```

### Mailgun

```go
email.WithEmail("smtp.mailgun.org", 587, "noreply@mg.yourcompany.com",
    email.WithEmailAuth("postmaster@mg.yourcompany.com", "your-mailgun-password"),
    email.WithEmailTLS(true),
)
```

### Amazon SES

```go
email.WithEmail("email-smtp.us-east-1.amazonaws.com", 587, "noreply@yourcompany.com",
    email.WithEmailAuth("your-ses-username", "your-ses-password"),
    email.WithEmailTLS(true),
)
```

## Configuration Options

### Available Options

- `WithEmailAuth(username, password)` - SMTP authentication
- `WithEmailTLS(bool)` - Enable/disable TLS encryption
- `WithEmailSSL(bool)` - Enable/disable SSL encryption
- `WithEmailTimeout(duration)` - Custom timeout for SMTP operations

### Security Best Practices

1. **Use App Passwords**: For Gmail and other providers
2. **Enable TLS/SSL**: Always encrypt SMTP connections
3. **Store Credentials Securely**: Use environment variables or secret management
4. **Use Dedicated Email Accounts**: Don't use personal accounts for notifications

## Use Cases

### System Monitoring

```go
msg := notifyhub.NewAlert("System Alert").
    WithBody("Database connection pool at 90% capacity").
    WithPlatformData(map[string]interface{}{
        "email_priority": "high",
        "email_cc":       []string{"ops@company.com"},
    }).
    Build()
```

### User Notifications

```go
msg := notifyhub.NewMessage("Account Created").
    WithBody("Welcome! Your account has been created successfully.").
    WithFormat("html").
    Build()
```

### Reports and Analytics

```go
msg := notifyhub.NewMessage("Weekly Report").
    WithBody(generateWeeklyReport()). // Custom report generation
    WithFormat("html").
    WithPlatformData(map[string]interface{}{
        "email_cc": []string{"management@company.com"},
    }).
    Build()
```

### Marketing Campaigns

```go
msg := notifyhub.NewMessage("Newsletter").
    WithBody(generateNewsletterHTML()).
    WithFormat("html").
    Build()
```

## Error Handling

Common email errors and solutions:

### Authentication Errors
- **Cause**: Invalid username/password
- **Solution**: Check credentials, use app passwords for Gmail

### Connection Timeouts
- **Cause**: Network issues or slow SMTP server
- **Solution**: Increase timeout with `WithEmailTimeout()`

### TLS/SSL Errors
- **Cause**: Incorrect encryption settings
- **Solution**: Match provider requirements (TLS for port 587, SSL for port 465)

### Rate Limiting
- **Cause**: Too many emails sent too quickly
- **Solution**: Implement delays or use professional email services

## Running the Example

```bash
cd examples/platforms/email
go run main.go
```

## Configuration Setup

1. Choose an SMTP provider (Gmail, SendGrid, etc.)
2. Get SMTP credentials (username, password/API key)
3. Configure authentication and encryption
4. Update example with your credentials
5. Test with a real email address

## Legacy Compatibility

Deprecated functions still work:

```go
// Deprecated but functional
notifyhub.WithEmail("host", 587, "user", "pass", "from@example.com", true, 30*time.Second)

// Recommended new way
email.WithEmail("host", 587, "from@example.com",
    email.WithEmailAuth("user", "pass"),
    email.WithEmailTLS(true),
    email.WithEmailTimeout(30*time.Second),
)
```

## Platform Capabilities

The Email platform supports:

✅ **Multiple Formats** - Plain text and HTML
✅ **Authentication** - Username/password and API key
✅ **Encryption** - TLS and SSL support
✅ **Advanced Features** - CC, BCC, priority
✅ **Templates** - Variable substitution
✅ **Attachments** - File attachment support (planned)

## Next Steps

- [SMS Platform](../sms/) - Multi-provider SMS support
- [Feishu Platform](../feishu/) - Rich messaging features
- [Unified Demo](../unified-demo/) - All platforms together