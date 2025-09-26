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

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// EmailPlatform implements the unified Platform interface for email notifications
type EmailPlatform struct {
	config *config.EmailConfig
	auth   smtp.Auth
	logger logger.Logger

	// SMTP connection settings
	smtpHost     string
	smtpPort     int
	smtpFrom     string
	smtpUsername string
	smtpPassword string
	smtpTLS      bool
	smtpSSL      bool
	timeout      time.Duration
}

// NewEmailPlatform creates a new Email platform with strong-typed configuration
func NewEmailPlatform(emailConfig *config.EmailConfig, logger logger.Logger) (platform.Platform, error) {
	if emailConfig == nil {
		return nil, fmt.Errorf("email configuration cannot be nil")
	}

	// Validate required fields
	if emailConfig.SMTPHost == "" {
		return nil, fmt.Errorf("smtp_host is required for Email platform")
	}
	if emailConfig.SMTPPort == 0 {
		return nil, fmt.Errorf("smtp_port is required for Email platform")
	}
	if emailConfig.SMTPFrom == "" {
		return nil, fmt.Errorf("smtp_from is required for Email platform")
	}

	platform := &EmailPlatform{
		config:       emailConfig,
		logger:       logger,
		smtpHost:     emailConfig.SMTPHost,
		smtpPort:     emailConfig.SMTPPort,
		smtpFrom:     emailConfig.SMTPFrom,
		smtpUsername: emailConfig.SMTPUsername,
		smtpPassword: emailConfig.SMTPPassword,
		smtpTLS:      emailConfig.SMTPTLS,
		smtpSSL:      emailConfig.SMTPSSL,
		timeout:      emailConfig.Timeout,
	}

	// Use default timeout if not specified
	if platform.timeout == 0 {
		platform.timeout = 30 * time.Second
	}

	// Setup SMTP authentication if both username and password are provided
	if platform.smtpUsername != "" && platform.smtpPassword != "" {
		platform.auth = smtp.PlainAuth("", platform.smtpUsername, platform.smtpPassword, platform.smtpHost)
	}

	return platform, nil
}

// Name returns the platform name
func (e *EmailPlatform) Name() string {
	return "email"
}

// Send sends a message to Email
func (e *EmailPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, tgt := range targets {
		startTime := time.Now()
		result := &platform.SendResult{
			Target:  tgt,
			Success: false,
		}

		// Validate target
		if err := e.ValidateTarget(tgt); err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		// Build email message
		emailMsg := e.buildEmailMessage(msg, tgt)

		// Send email
		if err := e.sendEmail(ctx, tgt.Value, emailMsg); err != nil {
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
func (e *EmailPlatform) ValidateTarget(tgt target.Target) error {
	switch tgt.Type {
	case "email":
		// Valid target type for Email
	default:
		return fmt.Errorf("email supports email targets, got %s", tgt.Type)
	}

	if tgt.Value == "" {
		return fmt.Errorf("email address cannot be empty")
	}

	// Basic email validation
	if !isValidEmail(tgt.Value) {
		return fmt.Errorf("invalid email address: %s", tgt.Value)
	}

	return nil
}

// GetCapabilities returns Email platform capabilities
func (e *EmailPlatform) GetCapabilities() platform.Capabilities {
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
func (e *EmailPlatform) IsHealthy(ctx context.Context) error {
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
func (e *EmailPlatform) Close() error {
	return nil
}

// buildEmailMessage builds an email message from the platform message
func (e *EmailPlatform) buildEmailMessage(msg *message.Message, tgt target.Target) string {
	var content string

	// Build headers
	content += fmt.Sprintf("From: %s\r\n", e.smtpFrom)
	content += fmt.Sprintf("To: %s\r\n", tgt.Value)

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
func (e *EmailPlatform) sendEmail(ctx context.Context, to, content string) error {
	if e.logger == nil {
		e.logger = logger.Discard
	}
	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)
	recipients := []string{to}

	// Use context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Debug logging
	e.logger.Debug("Connecting to SMTP", "address", addr)
	e.logger.Debug("Authentication configured", "auth", e.auth != nil)
	e.logger.Debug("Email details", "from", e.smtpFrom, "to", to)
	e.logger.Debug("Connection timeout", "timeout", e.timeout)

	// Send email with timeout
	done := make(chan error, 1)
	go func() {
		var err error

		if e.smtpTLS {
			// Use STARTTLS (port 587)
			e.logger.Debug("Using STARTTLS connection...")
			err = e.sendWithSTARTTLS(addr, recipients, content)
		} else if e.smtpSSL {
			// Use SSL/TLS (port 465)
			e.logger.Debug("Using SSL/TLS connection...")
			err = e.sendWithSSL(addr, recipients, content)
		} else {
			// Plain connection (not recommended)
			e.logger.Warn("Using plain SMTP (no encryption)")
			err = smtp.SendMail(addr, e.auth, e.smtpFrom, recipients, []byte(content))
		}
		if err != nil {
			e.logger.Error("SendMail error", "error", err)
		} else {
			e.logger.Info("Email sent successfully", "to", to)
		}
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		e.logger.Error("Send timeout", "duration", e.timeout)
		return fmt.Errorf("email send timeout")
	}
}

// sendWithSTARTTLS sends email using STARTTLS (port 587)
func (e *EmailPlatform) sendWithSTARTTLS(addr string, recipients []string, content string) error {
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
func (e *EmailPlatform) sendWithSSL(addr string, recipients []string, content string) error {
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

// isValidEmail performs basic email address validation
func isValidEmail(email string) bool {
	// Basic email validation - contains @ and has parts before and after
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	local, domain := parts[0], parts[1]

	// Check local part (before @)
	if len(local) == 0 || len(local) > 64 {
		return false
	}

	// Check domain part (after @)
	if len(domain) == 0 || len(domain) > 255 {
		return false
	}

	// Domain should contain at least one dot
	if !strings.Contains(domain, ".") {
		return false
	}

	return true
}

// init registers the email platform automatically
func init() {
	// Register email platform creator with the global registry
	platform.RegisterPlatform("email", func(cfg map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
		// Convert config to EmailConfig
		emailConfig := &config.EmailConfig{}
		if host, ok := cfg["host"].(string); ok {
			emailConfig.SMTPHost = host
		}
		if port, ok := cfg["port"].(int); ok {
			emailConfig.SMTPPort = port
		}
		if from, ok := cfg["from"].(string); ok {
			emailConfig.SMTPFrom = from
		}

		return NewEmailPlatform(emailConfig, logger)
	})
}
