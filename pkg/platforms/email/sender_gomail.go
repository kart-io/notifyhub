// Package email provides Email platform integration using go-mail library
// This replaces the deprecated net/smtp with modern go-mail library
package email

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wneessen/go-mail"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// EmailSenderGoMail implements the ExternalSender interface using go-mail library
type EmailSenderGoMail struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	smtpFrom     string
	smtpTLS      bool
	smtpSSL      bool
	timeout      time.Duration
	logger       logger.Logger
}

// NewEmailSenderGoMail creates a new Email sender using go-mail
func NewEmailSenderGoMail(config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
	logger.Debug("Creating EmailSenderGoMail", "config_keys", getConfigKeys(config))

	smtpHost, ok := config["smtp_host"].(string)
	if !ok || smtpHost == "" {
		logger.Error("Missing or invalid smtp_host")
		return nil, fmt.Errorf("smtp_host is required for Email platform")
	}

	smtpPort, ok := config["smtp_port"].(int)
	if !ok {
		logger.Error("Missing or invalid smtp_port")
		return nil, fmt.Errorf("smtp_port is required for Email platform")
	}

	smtpFrom, ok := config["smtp_from"].(string)
	if !ok || smtpFrom == "" {
		logger.Error("Missing or invalid smtp_from")
		return nil, fmt.Errorf("smtp_from is required for Email platform")
	}

	sender := &EmailSenderGoMail{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		smtpFrom: smtpFrom,
		smtpTLS:  true, // Default to TLS
		timeout:  30 * time.Second,
		logger:   logger, // Assign logger directly
	}

	// Configure authentication
	if username, ok := config["smtp_username"].(string); ok {
		sender.smtpUsername = username
		logger.Debug("SMTP authentication configured", "username", username)
	}

	if password, ok := config["smtp_password"].(string); ok {
		sender.smtpPassword = password
		logger.Debug("SMTP password configured")
	}

	// Configure TLS/SSL
	if tls, ok := config["smtp_tls"].(bool); ok {
		sender.smtpTLS = tls
		logger.Debug("SMTP TLS configured", "tls", tls)
	}

	if ssl, ok := config["smtp_ssl"].(bool); ok {
		sender.smtpSSL = ssl
		logger.Debug("SMTP SSL configured", "ssl", ssl)
	}

	// Configure timeout
	if timeout, ok := config["timeout"].(time.Duration); ok {
		sender.timeout = timeout
		logger.Debug("SMTP timeout configured", "timeout", timeout)
	}

	logger.Info("EmailSenderGoMail created successfully", "host", smtpHost, "port", smtpPort, "from", smtpFrom)
	return sender, nil
}

// getConfigKeys returns the keys from a config map for logging
func getConfigKeys(config map[string]interface{}) []string {
	keys := make([]string, 0, len(config))
	for k := range config {
		keys = append(keys, k)
	}
	return keys
}

// Name returns the platform name
func (e *EmailSenderGoMail) Name() string {
	return "email"
}

// Send sends a message to Email using go-mail library
func (e *EmailSenderGoMail) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
	e.logger.Debug("Starting email send", "messageID", msg.ID, "targetCount", len(targets))
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := &platform.SendResult{
			Target:  target,
			Success: false,
		}

		// Validate target
		if err := e.ValidateTarget(target); err != nil {
			e.logger.Error("Invalid email target", "target", target.Value, "error", err)
			result.Error = err.Error()
			results[i] = result
			continue
		}

		// Send email using go-mail
		if err := e.sendEmailGoMail(ctx, target.Value, msg); err != nil {
			e.logger.Error("Failed to send email", "to", target.Value, "error", err)
			result.Error = err.Error()
		} else {
			e.logger.Info("Email sent successfully", "to", target.Value, "messageID", msg.ID)
			result.Success = true
			result.MessageID = fmt.Sprintf("email_%d", time.Now().UnixNano())
			result.Response = "Email sent successfully via go-mail"
		}

		// Set metadata
		result.Metadata = map[string]interface{}{
			"duration":  time.Since(startTime).Milliseconds(),
			"smtp_host": e.smtpHost,
			"library":   "go-mail",
		}

		results[i] = result
		e.logger.Debug("Email send attempt completed", "to", target.Value, "success", result.Success, "duration_ms", time.Since(startTime).Milliseconds())
	}

	e.logger.Debug("Email batch send completed", "messageID", msg.ID, "totalTargets", len(targets))
	return results, nil
}

// sendEmailGoMail sends email using the modern go-mail library
func (e *EmailSenderGoMail) sendEmailGoMail(ctx context.Context, to string, msg *platform.Message) error {
	if e.logger == nil {
		e.logger = logger.Discard
	}
	e.logger.Debug("Preparing email", "to", to, "subject", msg.Title)

	// Create new email message
	m := mail.NewMsg()

	// Set sender
	if err := m.From(e.smtpFrom); err != nil {
		e.logger.Error("Failed to set From address", "from", e.smtpFrom, "error", err)
		return fmt.Errorf("failed to set From address: %w", err)
	}

	// Set recipient
	if err := m.To(to); err != nil {
		e.logger.Error("Failed to set To address", "to", to, "error", err)
		return fmt.Errorf("failed to set To address: %w", err)
	}

	// Set subject
	subject := msg.Title
	if subject == "" {
		subject = "Notification"
	}
	m.Subject(subject)

	// Set body based on format
	if msg.Format == "html" {
		m.SetBodyString(mail.TypeTextHTML, msg.Body)
	} else {
		m.SetBodyString(mail.TypeTextPlain, msg.Body)
	}

	// Handle CC recipients from platform data
	if ccList, ok := msg.PlatformData["email_cc"].([]string); ok && len(ccList) > 0 {
		e.logger.Debug("Adding CC recipients", "count", len(ccList))
		if err := m.Cc(ccList...); err != nil {
			e.logger.Error("Failed to set CC addresses", "error", err)
			return fmt.Errorf("failed to set CC addresses: %w", err)
		}
	}

	// Handle BCC recipients
	if bccList, ok := msg.PlatformData["email_bcc"].([]string); ok && len(bccList) > 0 {
		e.logger.Debug("Adding BCC recipients", "count", len(bccList))
		if err := m.Bcc(bccList...); err != nil {
			e.logger.Error("Failed to set BCC addresses", "error", err)
			return fmt.Errorf("failed to set BCC addresses: %w", err)
		}
	}

	// Set priority
	if priority, ok := msg.PlatformData["email_priority"].(string); ok {
		switch strings.ToLower(priority) {
		case "high", "urgent":
			m.SetImportance(mail.ImportanceHigh)
		case "low":
			m.SetImportance(mail.ImportanceLow)
		default:
			m.SetImportance(mail.ImportanceNormal)
		}
	}

	// Create client with timeout
	clientOpts := []mail.Option{
		mail.WithTimeout(e.timeout),
		mail.WithPort(e.smtpPort),
	}

	// Configure TLS/SSL
	if e.smtpSSL {
		// Use implicit SSL/TLS (port 465)
		e.logger.Debug("Using SSL connection", "port", e.smtpPort)
		clientOpts = append(clientOpts, mail.WithSSLPort(true))
	} else if e.smtpTLS {
		// Use STARTTLS (port 587)
		e.logger.Debug("Using STARTTLS connection", "port", e.smtpPort)
		clientOpts = append(clientOpts, mail.WithTLSPolicy(mail.TLSMandatory))
	} else {
		// No encryption (not recommended)
		e.logger.Warn("Using plain SMTP without encryption")
		clientOpts = append(clientOpts, mail.WithTLSPolicy(mail.NoTLS))
	}

	// Add authentication if provided
	if e.smtpUsername != "" && e.smtpPassword != "" {
		clientOpts = append(clientOpts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(e.smtpUsername),
			mail.WithPassword(e.smtpPassword),
		)
	}

	// Create client
	client, err := mail.NewClient(e.smtpHost, clientOpts...)
	if err != nil {
		e.logger.Error("Failed to create mail client", "host", e.smtpHost, "port", e.smtpPort, "error", err)
		return fmt.Errorf("failed to create mail client: %w", err)
	}

	// Send with context
	e.logger.Debug("Sending email", "host", e.smtpHost, "port", e.smtpPort)
	if err := client.DialAndSendWithContext(ctx, m); err != nil {
		e.logger.Error("Failed to send email", "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	e.logger.Info("Email sent successfully", "to", to)
	return nil
}

// ValidateTarget validates a target for Email
func (e *EmailSenderGoMail) ValidateTarget(target platform.Target) error {
	switch target.Type {
	case "email":
		// Valid target type for Email
	default:
		e.logger.Debug("Invalid target type for email", "type", target.Type)
		return fmt.Errorf("email supports email targets, got %s", target.Type)
	}

	if target.Value == "" {
		e.logger.Debug("Empty email address")
		return fmt.Errorf("email address cannot be empty")
	}

	// Basic email validation
	if !isValidEmail(target.Value) {
		e.logger.Debug("Invalid email format", "email", target.Value)
		return fmt.Errorf("invalid email address: %s", target.Value)
	}

	return nil
}

// GetCapabilities returns Email platform capabilities
func (e *EmailSenderGoMail) GetCapabilities() platform.Capabilities {
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

// IsHealthy checks if Email SMTP server is accessible using go-mail
func (e *EmailSenderGoMail) IsHealthy(ctx context.Context) error {
	if e.smtpHost == "" {
		return fmt.Errorf("SMTP host is not configured")
	}

	// Create a test client
	clientOpts := []mail.Option{
		mail.WithTimeout(e.timeout),
		mail.WithPort(e.smtpPort),
	}

	if e.smtpSSL {
		clientOpts = append(clientOpts, mail.WithSSLPort(true))
	} else if e.smtpTLS {
		clientOpts = append(clientOpts, mail.WithTLSPolicy(mail.TLSMandatory))
	}

	if e.smtpUsername != "" && e.smtpPassword != "" {
		clientOpts = append(clientOpts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(e.smtpUsername),
			mail.WithPassword(e.smtpPassword),
		)
	}

	client, err := mail.NewClient(e.smtpHost, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Test connection
	if err := client.DialWithContext(ctx); err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	_ = client.Close()
	return nil
}

// Close cleans up resources
func (e *EmailSenderGoMail) Close() error {
	e.logger.Debug("Closing email sender", "host", e.smtpHost)
	// No persistent connections to close with go-mail
	return nil
}
