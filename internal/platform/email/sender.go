// Package email implements the Email Sender for SMTP-based email notifications
// This integrates all Email-specific logic from the former adapter.go, platform.go, and builder.go
package email

import (
	"context"
	"fmt"
	"net/smtp"
	"time"

	"github.com/kart-io/notifyhub/internal/platform"
)

// EmailSender implements the platform.Sender interface for email notifications
type EmailSender struct {
	name   string
	config EmailConfig
	auth   smtp.Auth
}

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost     string        `json:"smtp_host"`
	SMTPPort     int           `json:"smtp_port"`
	SMTPUsername string        `json:"smtp_username"`
	SMTPPassword string        `json:"smtp_password"`
	SMTPFrom     string        `json:"smtp_from"`
	SMTPTLS      bool          `json:"smtp_tls"`
	SMTPSSL      bool          `json:"smtp_ssl"`
	Timeout      time.Duration `json:"timeout"`
}

// NewEmailSender creates a new email sender
func NewEmailSender(config map[string]interface{}) (platform.Sender, error) {
	emailConfig, err := parseEmailConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid email configuration: %w", err)
	}

	// Setup SMTP authentication
	var auth smtp.Auth
	if emailConfig.SMTPUsername != "" && emailConfig.SMTPPassword != "" {
		auth = smtp.PlainAuth("", emailConfig.SMTPUsername, emailConfig.SMTPPassword, emailConfig.SMTPHost)
	}

	return &EmailSender{
		name:   "email",
		config: emailConfig,
		auth:   auth,
	}, nil
}

// Name returns the platform name
func (e *EmailSender) Name() string {
	return e.name
}

// Send sends an email message to the specified targets
func (e *EmailSender) Send(ctx context.Context, msg *platform.InternalMessage, targets []platform.InternalTarget) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := platform.NewSendResult(target, false)

		// Validate target
		if err := e.ValidateTarget(target); err != nil {
			result.Error = err.Error()
			result.SetDuration(time.Since(startTime))
			results[i] = result
			continue
		}

		// Build email content
		emailContent, err := e.buildEmailContent(msg, target)
		if err != nil {
			result.Error = fmt.Sprintf("failed to build email content: %v", err)
			result.SetDuration(time.Since(startTime))
			results[i] = result
			continue
		}

		// Send email
		err = e.sendEmail(ctx, target.Value, emailContent)
		if err != nil {
			result.Error = fmt.Sprintf("failed to send email: %v", err)
		} else {
			result.Success = true
			result.MessageID = fmt.Sprintf("email_%d", time.Now().UnixNano())
		}

		result.SetDuration(time.Since(startTime))
		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates an email target
func (e *EmailSender) ValidateTarget(target platform.InternalTarget) error {
	if target.Type != "email" {
		return fmt.Errorf("email sender only supports email targets, got %s", target.Type)
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

// GetCapabilities returns the email platform capabilities
func (e *EmailSender) GetCapabilities() platform.PlatformCapabilities {
	return platform.PlatformCapabilities{
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

// IsHealthy checks if the email sender is healthy
func (e *EmailSender) IsHealthy(ctx context.Context) error {
	// Try to connect to SMTP server
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	// Use context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// Simple connection test
	done := make(chan error, 1)
	go func() {
		conn, err := smtp.Dial(addr)
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		return fmt.Errorf("smtp connection timeout")
	}
}

// Close cleans up the email sender resources
func (e *EmailSender) Close() error {
	// Email sender doesn't hold persistent connections
	return nil
}

// buildEmailContent builds the email content from the internal message
func (e *EmailSender) buildEmailContent(msg *platform.InternalMessage, target platform.InternalTarget) (string, error) {
	var content string

	// Build headers
	content += fmt.Sprintf("From: %s\r\n", e.config.SMTPFrom)
	content += fmt.Sprintf("To: %s\r\n", target.Value)

	// Add CC and BCC from platform data
	if ccList, ok := msg.PlatformData["email_cc"].([]string); ok && len(ccList) > 0 {
		content += fmt.Sprintf("Cc: %s\r\n", joinEmails(ccList))
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

	// TODO: Handle attachments from msg.PlatformData["email_attachments"]

	return content, nil
}

// sendEmail sends the email using SMTP
func (e *EmailSender) sendEmail(ctx context.Context, to, content string) error {
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	// Extract recipients (to, cc, bcc from content)
	recipients := []string{to}
	// TODO: Parse CC and BCC from content and add to recipients

	// Use context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// Send email with timeout
	done := make(chan error, 1)
	go func() {
		err := smtp.SendMail(addr, e.auth, e.config.SMTPFrom, recipients, []byte(content))
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		return fmt.Errorf("email send timeout")
	}
}

// parseEmailConfig parses email configuration from map
func parseEmailConfig(config map[string]interface{}) (EmailConfig, error) {
	emailConfig := EmailConfig{
		Timeout: 30 * time.Second, // Default timeout
		SMTPTLS: true,             // Default to TLS
	}

	// Required fields
	if host, ok := config["smtp_host"].(string); ok {
		emailConfig.SMTPHost = host
	} else {
		return emailConfig, fmt.Errorf("smtp_host is required")
	}

	if port, ok := config["smtp_port"].(int); ok {
		emailConfig.SMTPPort = port
	} else {
		return emailConfig, fmt.Errorf("smtp_port is required")
	}

	if from, ok := config["smtp_from"].(string); ok {
		emailConfig.SMTPFrom = from
	} else {
		return emailConfig, fmt.Errorf("smtp_from is required")
	}

	// Optional fields
	if username, ok := config["smtp_username"].(string); ok {
		emailConfig.SMTPUsername = username
	}

	if password, ok := config["smtp_password"].(string); ok {
		emailConfig.SMTPPassword = password
	}

	if tls, ok := config["smtp_tls"].(bool); ok {
		emailConfig.SMTPTLS = tls
	}

	if ssl, ok := config["smtp_ssl"].(bool); ok {
		emailConfig.SMTPSSL = ssl
	}

	if timeout, ok := config["timeout"].(time.Duration); ok {
		emailConfig.Timeout = timeout
	}

	return emailConfig, nil
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	// Basic validation - contains @ and .
	atCount := 0
	dotAfterAt := false
	atPos := -1

	for i, c := range email {
		if c == '@' {
			atCount++
			atPos = i
		} else if c == '.' && atPos > 0 && i > atPos {
			dotAfterAt = true
		}
	}

	return atCount == 1 && dotAfterAt && len(email) > 5
}

// joinEmails joins email addresses with commas
func joinEmails(emails []string) string {
	if len(emails) == 0 {
		return ""
	}

	result := emails[0]
	for i := 1; i < len(emails); i++ {
		result += ", " + emails[i]
	}
	return result
}
