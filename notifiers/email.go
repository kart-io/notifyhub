package notifiers

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/internal"
)

// ================================
// Email平台适配器实现
// ================================

// EmailNotifier implements email notifications via SMTP
type EmailNotifier struct {
	host        string
	port        int
	username    string
	password    string
	from        string
	useTLS      bool
	timeout     time.Duration
	rateLimiter internal.RateLimiter
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(host string, port int, username, password, from string, useTLS bool, timeout time.Duration) *EmailNotifier {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &EmailNotifier{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		from:        from,
		useTLS:      useTLS,
		timeout:     timeout,
		rateLimiter: &internal.NoOpRateLimiter{}, // Default: no rate limiting
	}
}

// NewEmailNotifierWithRateLimit creates a new email notifier with rate limiting
func NewEmailNotifierWithRateLimit(host string, port int, username, password, from string, useTLS bool, timeout time.Duration, rateLimitConfig *internal.RateLimitConfig) *EmailNotifier {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &EmailNotifier{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		from:        from,
		useTLS:      useTLS,
		timeout:     timeout,
		rateLimiter: internal.NewRateLimiter(rateLimitConfig),
	}
}

// Name returns the notifier name
func (e *EmailNotifier) Name() string {
	return "email"
}

// SupportsTarget checks if this notifier supports the given target
func (e *EmailNotifier) SupportsTarget(target Target) bool {
	if target.Platform != "" && target.Platform != "email" {
		return false
	}
	return target.Type == TargetTypeEmail || strings.Contains(target.Value, "@")
}

// Send sends a message via email
func (e *EmailNotifier) Send(ctx context.Context, message *Message) ([]*SendResult, error) {
	var results []*SendResult

	for _, target := range message.Targets {
		if e.SupportsTarget(target) {
			result := e.sendEmail(ctx, message, target)
			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no supported email targets found")
	}

	return results, nil
}

// sendEmail sends an email to a specific target
func (e *EmailNotifier) sendEmail(ctx context.Context, message *Message, target Target) *SendResult {
	start := time.Now()
	result := &SendResult{
		Target:   target,
		Platform: e.Name(),
		SentAt:   start,
		Attempts: 1,
	}

	// Apply rate limiting
	if err := e.rateLimiter.ConsumeWithTimeout(ctx, e.timeout); err != nil {
		result.Error = fmt.Sprintf("rate limit exceeded: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	subject := message.Title
	if subject == "" {
		subject = "Notification from NotifyHub"
	}

	body := e.buildEmailBody(message)
	emailMsg := e.buildEmailMessage(e.from, target.Value, subject, body)

	// Send email
	var auth smtp.Auth
	if e.username != "" && e.password != "" {
		auth = smtp.PlainAuth("", e.username, e.password, e.host)
	}

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	err := smtp.SendMail(addr, auth, e.from, []string{target.Value}, []byte(emailMsg))

	result.Duration = time.Since(start)
	result.Success = err == nil
	if err != nil {
		result.Error = fmt.Sprintf("send email: %v", err)
	}

	return result
}

// buildEmailBody builds the email body based on message format
func (e *EmailNotifier) buildEmailBody(message *Message) string {
	body := message.Body

	// Convert formats to plain text for email compatibility
	switch message.Format {
	case FormatMarkdown:
		body = e.markdownToText(body)
	case FormatHTML:
		body = e.htmlToText(body)
	}

	// Add metadata if present
	if len(message.Metadata) > 0 {
		body += "\n\n---\nAdditional Information:\n"
		for key, value := range message.Metadata {
			body += fmt.Sprintf("%s: %s\n", key, value)
		}
	}

	// Add timestamp
	body += fmt.Sprintf("\n\nSent at: %s", message.CreatedAt.Format("2006-01-02 15:04:05"))

	return body
}

// buildEmailMessage builds the complete email message with headers
func (e *EmailNotifier) buildEmailMessage(from, to, subject, body string) string {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)
	return msg
}

// markdownToText converts simple markdown to plain text
func (e *EmailNotifier) markdownToText(markdown string) string {
	// Simple markdown to text conversion
	text := strings.ReplaceAll(markdown, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ReplaceAll(text, "#", "")
	return strings.TrimSpace(text)
}

// htmlToText converts simple HTML to plain text
func (e *EmailNotifier) htmlToText(html string) string {
	// Simple HTML to text conversion
	text := strings.ReplaceAll(html, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<p>", "")
	text = strings.ReplaceAll(text, "</p>", "\n")
	return strings.TrimSpace(text)
}

// Health checks if email notifier is healthy
func (e *EmailNotifier) Health(ctx context.Context) error {
	if e.host == "" || e.from == "" {
		return fmt.Errorf("email configuration incomplete")
	}

	// Try to connect to SMTP server
	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %v", err)
	}
	defer client.Close()

	// Test authentication if configured
	if e.username != "" && e.password != "" {
		auth := smtp.PlainAuth("", e.username, e.password, e.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}
	}

	return nil
}

// Shutdown gracefully shuts down the email notifier
func (e *EmailNotifier) Shutdown(ctx context.Context) error {
	// Email notifier doesn't maintain persistent connections
	// No cleanup needed, but method required by interface
	return nil
}