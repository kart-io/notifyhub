// Package email provides Email platform integration for NotifyHub
// This package implements unified platform architecture for email notifications
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// EmailSender implements the ExternalSender interface for email notifications
type EmailSender struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	smtpFrom     string
	smtpTLS      bool
	smtpSSL      bool
	timeout      time.Duration
	auth         smtp.Auth
}

// NewEmailSender creates a new Email sender
func NewEmailSender(config map[string]interface{}) (platform.ExternalSender, error) {
	smtpHost, ok := config["smtp_host"].(string)
	if !ok || smtpHost == "" {
		return nil, fmt.Errorf("smtp_host is required for Email platform")
	}

	smtpPort, ok := config["smtp_port"].(int)
	if !ok {
		return nil, fmt.Errorf("smtp_port is required for Email platform")
	}

	smtpFrom, ok := config["smtp_from"].(string)
	if !ok || smtpFrom == "" {
		return nil, fmt.Errorf("smtp_from is required for Email platform")
	}

	sender := &EmailSender{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		smtpFrom: smtpFrom,
		smtpTLS:  true, // Default to TLS
		timeout:  30 * time.Second,
	}

	// Configure authentication
	if username, ok := config["smtp_username"].(string); ok {
		sender.smtpUsername = username
	}

	if password, ok := config["smtp_password"].(string); ok {
		sender.smtpPassword = password
	}

	// Setup SMTP authentication if both username and password are provided
	if sender.smtpUsername != "" && sender.smtpPassword != "" {
		sender.auth = smtp.PlainAuth("", sender.smtpUsername, sender.smtpPassword, sender.smtpHost)
	}

	// Configure TLS/SSL
	if tls, ok := config["smtp_tls"].(bool); ok {
		sender.smtpTLS = tls
	}

	if ssl, ok := config["smtp_ssl"].(bool); ok {
		sender.smtpSSL = ssl
	}

	// Configure timeout
	if timeout, ok := config["timeout"].(time.Duration); ok {
		sender.timeout = timeout
	}

	return sender, nil
}

// Name returns the platform name
func (e *EmailSender) Name() string {
	return "email"
}

// Send sends a message to Email
func (e *EmailSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := &platform.SendResult{
			Target:  target,
			Success: false,
		}

		// Validate target
		if err := e.ValidateTarget(target); err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		// Build email message
		emailMsg := e.buildEmailMessage(msg, target)

		// Send email
		if err := e.sendEmail(ctx, target.Value, emailMsg); err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.MessageID = fmt.Sprintf("email_%d", time.Now().UnixNano())
			result.Response = "Email sent successfully"
		}

		// Set metadata
		result.Metadata = map[string]interface{}{
			"duration":  time.Since(startTime).Milliseconds(),
			"smtp_host": e.smtpHost,
		}

		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates a target for Email
func (e *EmailSender) ValidateTarget(target platform.Target) error {
	switch target.Type {
	case "email":
		// Valid target type for Email
	default:
		return fmt.Errorf("email supports email targets, got %s", target.Type)
	}

	if target.Value == "" {
		return fmt.Errorf("email address cannot be empty")
	}

	// Basic email validation
	if !isValidEmail(target.Value) {
		return fmt.Errorf("invalid email address: %s", target.Value)
	}

	return nil
}

// GetCapabilities returns Email platform capabilities
func (e *EmailSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "email",
		SupportedTargetTypes: []string{"email"},
		SupportedFormats:     []string{"text", "html"},
		MaxMessageSize:       25 * 1024 * 1024, // 25MB (typical email limit)
		SupportsScheduling:   false,
		SupportsAttachments:  true,
		SupportsMentions:     false,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"smtp_host", "smtp_port", "smtp_from"},
	}
}

// IsHealthy checks if Email SMTP server is accessible
func (e *EmailSender) IsHealthy(ctx context.Context) error {
	if e.smtpHost == "" {
		return fmt.Errorf("SMTP host is not configured")
	}

	// Try to connect to SMTP server
	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)

	// Use context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Simple connection test
	done := make(chan error, 1)
	go func() {
		conn, err := smtp.Dial(addr)
		if err != nil {
			done <- err
			return
		}
		defer func() { _ = conn.Close() }()
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		return fmt.Errorf("SMTP connection timeout")
	}
}

// Close cleans up resources
func (e *EmailSender) Close() error {
	return nil
}

// buildEmailMessage builds an email message from the platform message
func (e *EmailSender) buildEmailMessage(msg *platform.Message, target platform.Target) string {
	var content string

	// Build headers
	content += fmt.Sprintf("From: %s\r\n", e.smtpFrom)
	content += fmt.Sprintf("To: %s\r\n", target.Value)

	// Add CC and BCC from platform data
	if ccList, ok := msg.PlatformData["email_cc"].([]string); ok && len(ccList) > 0 {
		content += fmt.Sprintf("Cc: %s\r\n", strings.Join(ccList, ", "))
	}

	// Subject
	subject := msg.Title
	if subject == "" {
		subject = "Notification"
	}
	content += fmt.Sprintf("Subject: %s\r\n", subject)

	// Priority
	if priority, ok := msg.PlatformData["email_priority"].(string); ok {
		switch priority {
		case "high":
			content += "X-Priority: 1\r\n"
		case "low":
			content += "X-Priority: 5\r\n"
		}
	}

	// Content-Type based on format
	if msg.Format == "html" {
		content += "MIME-Version: 1.0\r\n"
		content += "Content-Type: text/html; charset=UTF-8\r\n"
	} else {
		content += "MIME-Version: 1.0\r\n"
		content += "Content-Type: text/plain; charset=UTF-8\r\n"
	}

	content += "\r\n" // Empty line between headers and body

	// Body
	body := msg.Body
	if body == "" {
		body = subject // Use subject as body if body is empty
	}

	content += body

	return content
}

// sendEmail sends the email using SMTP
func (e *EmailSender) sendEmail(ctx context.Context, to, content string) error {
	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)
	recipients := []string{to}

	// Use context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Debug logging
	fmt.Printf("[SMTP DEBUG] Connecting to %s\n", addr)
	fmt.Printf("[SMTP DEBUG] Auth configured: %v\n", e.auth != nil)
	fmt.Printf("[SMTP DEBUG] From: %s, To: %s\n", e.smtpFrom, to)
	fmt.Printf("[SMTP DEBUG] Timeout: %v\n", e.timeout)

	// Send email with timeout
	done := make(chan error, 1)
	go func() {
		// Use the original smtp.SendMail - it worked before!
		fmt.Printf("[SMTP DEBUG] Calling smtp.SendMail (original implementation)...\n")
		var err error

		if e.smtpTLS {
			// Use STARTTLS (port 587)
			fmt.Printf("[SMTP DEBUG] Using STARTTLS connection...\n")
			err = e.sendWithSTARTTLS(addr, recipients, content)
		} else if e.smtpSSL {
			// Use SSL/TLS (port 465)
			fmt.Printf("[SMTP DEBUG] Using SSL/TLS connection...\n")
			err = e.sendWithSSL(addr, recipients, content)
		} else {
			// Plain connection (not recommended)
			fmt.Printf("[SMTP DEBUG] Using plain SMTP (no encryption)...\n")
			err = smtp.SendMail(addr, e.auth, e.smtpFrom, recipients, []byte(content))
		}
		if err != nil {
			fmt.Printf("[SMTP DEBUG] SendMail error: %v\n", err)
		} else {
			fmt.Printf("[SMTP DEBUG] SendMail success\n")
		}
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		fmt.Printf("[SMTP DEBUG] Send timeout after %v\n", e.timeout)
		return fmt.Errorf("email send timeout")
	}
}

// sendWithSTARTTLS sends email using STARTTLS (port 587)
func (e *EmailSender) sendWithSTARTTLS(addr string, recipients []string, content string) error {
	// Connect to SMTP server
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial error: %w", err)
	}
	defer func() { _ = conn.Close() }()

	c, err := smtp.NewClient(conn, e.smtpHost)
	if err != nil {
		return fmt.Errorf("SMTP client error: %w", err)
	}
	defer func() { _ = c.Close() }()

	// Send EHLO
	if err = c.Hello("localhost"); err != nil {
		return fmt.Errorf("EHLO error: %w", err)
	}

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: e.smtpHost,
		MinVersion: tls.VersionTLS12,
	}
	if err = c.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS error: %w", err)
	}

	// Authenticate
	if e.auth != nil {
		if err = c.Auth(e.auth); err != nil {
			return fmt.Errorf("auth error: %w", err)
		}
	}

	// Send email
	if err = c.Mail(e.smtpFrom); err != nil {
		return fmt.Errorf("MAIL FROM error: %w", err)
	}

	for _, rcpt := range recipients {
		if err = c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("RCPT TO error: %w", err)
		}
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA error: %w", err)
	}

	if _, err = fmt.Fprintf(wc, "%s", content); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	if err = wc.Close(); err != nil {
		return fmt.Errorf("close error: %w", err)
	}

	return c.Quit()
}

// sendWithSSL sends email using SSL/TLS (port 465)
func (e *EmailSender) sendWithSSL(addr string, recipients []string, content string) error {
	tlsConfig := &tls.Config{
		ServerName: e.smtpHost,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial error: %w", err)
	}
	defer func() { _ = conn.Close() }()

	c, err := smtp.NewClient(conn, e.smtpHost)
	if err != nil {
		return fmt.Errorf("SMTP client error: %w", err)
	}
	defer func() { _ = c.Close() }()

	// Authenticate
	if e.auth != nil {
		if err = c.Auth(e.auth); err != nil {
			return fmt.Errorf("auth error: %w", err)
		}
	}

	// Send email
	if err = c.Mail(e.smtpFrom); err != nil {
		return fmt.Errorf("MAIL FROM error: %w", err)
	}

	for _, rcpt := range recipients {
		if err = c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("RCPT TO error: %w", err)
		}
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA error: %w", err)
	}

	if _, err = fmt.Fprintf(wc, "%s", content); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	if err = wc.Close(); err != nil {
		return fmt.Errorf("close error: %w", err)
	}

	return c.Quit()
}
