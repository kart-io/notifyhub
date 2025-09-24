# Email Platform Troubleshooting Guide

## Common Issues and Solutions

### 1. Network Connection Timeout

**Error:**
```
dial error: dial tcp 74.125.204.109:587: i/o timeout
```

**Cause:** Unable to connect to Gmail SMTP server. This can happen due to:
- Firewall blocking outbound SMTP connections
- Corporate network restrictions
- ISP blocking port 587
- Network requiring proxy configuration

**Solutions:**

#### Option A: Use a Different SMTP Server

If you have access to another SMTP server, configure it instead:

```go
// Example: Using a local or corporate SMTP server
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.yourcompany.com", 587, "noreply@yourcompany.com",
        email.WithEmailAuth("username", "password"),
        email.WithEmailTLS(true),
    ),
)
```

#### Option B: Use Mock SMTP for Testing

Install and use a local SMTP testing server:

```bash
# Install MailHog (local SMTP server for testing)
brew install mailhog  # macOS
# or download from https://github.com/mailhog/MailHog

# Run MailHog
mailhog

# Configure NotifyHub to use MailHog
hub, err := notifyhub.NewHub(
    email.WithEmail("localhost", 1025, "test@example.com",
        email.WithEmailTLS(false),  // No TLS for local testing
    ),
)
```

#### Option C: Check Network Connectivity

Test SMTP connectivity:

```bash
# Test port 587 (STARTTLS)
nc -zv smtp.gmail.com 587

# Test port 465 (SSL)
nc -zv smtp.gmail.com 465

# Test port 25 (Plain SMTP, often blocked)
nc -zv smtp.gmail.com 25

# Check if proxy is required
curl -v telnet://smtp.gmail.com:587
```

#### Option D: Use Gmail via SSL (Port 465)

If port 587 is blocked but 465 is open:

```go
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.gmail.com", 465, "your-email@gmail.com",
        email.WithEmailAuth("your-email@gmail.com", "your-app-password"),
        email.WithEmailSSL(true),   // Use SSL instead of TLS
        email.WithEmailTLS(false),
    ),
)
```

### 2. Authentication Failures

**Error:**
```
auth error: 535 5.7.8 Username and Password not accepted
```

**Cause:** Invalid credentials or Gmail security settings.

**Solutions:**

1. **Use App Password (Recommended for Gmail):**
   - Go to https://myaccount.google.com/apppasswords
   - Generate an app-specific password
   - Use this password instead of your regular password

2. **Enable "Less secure app access" (Not recommended):**
   - Go to https://myaccount.google.com/lesssecureapps
   - Turn on "Allow less secure apps"

### 3. TLS/SSL Configuration Issues

**Error:**
```
STARTTLS error: x509: certificate signed by unknown authority
```

**Solution:** Update TLS configuration to skip verification (for testing only):

```go
// In sender.go, modify tlsConfig:
tlsConfig := &tls.Config{
    ServerName:         e.smtpHost,
    MinVersion:         tls.VersionTLS12,
    InsecureSkipVerify: true,  // Only for testing!
}
```

### 4. Timeout Issues

**Current timeout:** 45 seconds (configurable)

**Adjust timeout:**

```go
hub, err := notifyhub.NewHub(
    email.WithEmail("smtp.gmail.com", 587, "your-email@gmail.com",
        email.WithEmailAuth("your-email@gmail.com", "password"),
        email.WithEmailTLS(true),
        email.WithEmailTimeout(60*time.Second),  // Increase to 60s
    ),
)
```

## Testing Email Without SMTP

### Use a Mock Email Function

Create a test configuration that logs emails instead of sending:

```go
// Create a mock sender for testing
type MockEmailSender struct{}

func (m *MockEmailSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
    results := make([]*platform.SendResult, len(targets))
    for i, target := range targets {
        fmt.Printf("[MOCK EMAIL] To: %s\n", target.Value)
        fmt.Printf("[MOCK EMAIL] Subject: %s\n", msg.Title)
        fmt.Printf("[MOCK EMAIL] Body: %s\n", msg.Body)

        results[i] = &platform.SendResult{
            Target:    target,
            Success:   true,
            MessageID: fmt.Sprintf("mock_%d", time.Now().UnixNano()),
            Response:  "Mock email logged",
        }
    }
    return results, nil
}
```

## Recommended Testing Setup

For development and testing without network dependencies:

1. **Install MailHog:**
   ```bash
   # macOS
   brew install mailhog

   # Linux
   wget https://github.com/mailhog/MailHog/releases/download/v1.0.1/MailHog_linux_amd64
   chmod +x MailHog_linux_amd64
   ./MailHog_linux_amd64
   ```

2. **Configure NotifyHub:**
   ```go
   hub, err := notifyhub.NewHub(
       email.WithEmail("localhost", 1025, "test@example.com"),
   )
   ```

3. **View emails:**
   - Open browser: http://localhost:8025
   - All sent emails will appear in the MailHog UI

## Popular SMTP Providers Configuration

### Gmail (Port 587 - TLS)
```go
email.WithEmail("smtp.gmail.com", 587, "your-email@gmail.com",
    email.WithEmailAuth("your-email@gmail.com", "app-password"),
    email.WithEmailTLS(true),
)
```

### Gmail (Port 465 - SSL)
```go
email.WithEmail("smtp.gmail.com", 465, "your-email@gmail.com",
    email.WithEmailAuth("your-email@gmail.com", "app-password"),
    email.WithEmailSSL(true),
    email.WithEmailTLS(false),
)
```

### Outlook/Office 365
```go
email.WithEmail("smtp.office365.com", 587, "your-email@outlook.com",
    email.WithEmailAuth("your-email@outlook.com", "password"),
    email.WithEmailTLS(true),
)
```

### SendGrid
```go
email.WithEmail("smtp.sendgrid.net", 587, "noreply@example.com",
    email.WithEmailAuth("apikey", "YOUR_SENDGRID_API_KEY"),
    email.WithEmailTLS(true),
)
```

### Amazon SES
```go
email.WithEmail("email-smtp.us-east-1.amazonaws.com", 587, "verified@example.com",
    email.WithEmailAuth("SMTP_USERNAME", "SMTP_PASSWORD"),
    email.WithEmailTLS(true),
)
```

## Debug Mode

Enable detailed logging to troubleshoot issues:

The current implementation includes debug logging. To disable it in production:

1. Remove `fmt.Printf` statements from `pkg/platforms/email/sender.go`
2. Or wrap them in a debug flag:

```go
if os.Getenv("NOTIFYHUB_DEBUG") == "true" {
    fmt.Printf("[SMTP DEBUG] ...")
}
```

## Contact Support

If issues persist:
1. Check the error message details
2. Verify network connectivity
3. Confirm SMTP credentials
4. Test with a local SMTP server
5. Review firewall/proxy settings