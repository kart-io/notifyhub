package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/core/errors"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
)

// Transport implements the Transport interface for Email
type Transport struct {
	host     string
	port     int
	username string
	password string
	from     string
	useTLS   bool
	timeout  time.Duration
}

// Config holds the configuration for Email transport
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
	Timeout  time.Duration
}

// NewTransport creates a new Email transport
func NewTransport(config *Config) *Transport {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Port == 0 {
		config.Port = 587
	}

	return &Transport{
		host:     config.Host,
		port:     config.Port,
		username: config.Username,
		password: config.Password,
		from:     config.From,
		useTLS:   config.UseTLS,
		timeout:  config.Timeout,
	}
}

// Name returns the transport name
func (t *Transport) Name() string {
	return "email"
}

// Send sends a message through Email
func (t *Transport) Send(ctx context.Context, msg *message.Message, target sending.Target) (*sending.Result, error) {
	result := sending.NewResult(msg.ID, target)
	result.SetStatus(sending.StatusSending)

	// Validate target is email
	if target.Type != sending.TargetTypeEmail {
		err := errors.NewEmailError(errors.CodeInvalidTarget, fmt.Sprintf("invalid target type for email transport: %s", target.Type))
		result.SetError(err)
		return result, err
	}

	// Build email content
	emailMsg, err := t.buildEmailMessage(msg, target)
	if err != nil {
		wrappedErr := errors.WrapWithPlatform(errors.CodeProcessingFailed, errors.CategoryTransport, "failed to build email message", "email", err)
		result.SetError(wrappedErr)
		return result, wrappedErr
	}

	// Send email
	if err := t.sendEmail(ctx, target.Value, emailMsg); err != nil {
		// Use specialized SMTP error mapping
		smtpErr := errors.MapSMTPError(err)
		result.SetError(smtpErr)
		return result, smtpErr
	}

	result.SetStatus(sending.StatusSent)
	return result, nil
}

// buildEmailMessage builds the email message content
func (t *Transport) buildEmailMessage(msg *message.Message, target sending.Target) (string, error) {
	var emailMsg strings.Builder

	// Headers
	emailMsg.WriteString(fmt.Sprintf("From: %s\r\n", t.from))
	emailMsg.WriteString(fmt.Sprintf("To: %s\r\n", target.Value))

	// Subject
	subject := msg.Title
	if subject == "" {
		subject = "Notification"
	}
	emailMsg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	// Content-Type based on format
	switch msg.Format {
	case message.FormatHTML:
		emailMsg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	case message.FormatMarkdown:
		// Convert markdown to HTML if needed, for now treat as plain text
		emailMsg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	default:
		emailMsg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	emailMsg.WriteString("\r\n")

	// Body
	body := msg.Body
	if body == "" && msg.Title != "" {
		body = msg.Title
	}
	emailMsg.WriteString(body)

	return emailMsg.String(), nil
}

// sendEmail sends the email using SMTP
func (t *Transport) sendEmail(ctx context.Context, to, message string) error {
	// Create auth
	auth := smtp.PlainAuth("", t.username, t.password, t.host)

	// Server address
	addr := fmt.Sprintf("%s:%d", t.host, t.port)

	// Recipients
	recipients := []string{to}

	// Send email with context support
	done := make(chan error, 1)
	go func() {
		var err error
		if t.useTLS {
			err = t.sendWithTLS(addr, auth, t.from, recipients, []byte(message))
		} else {
			err = smtp.SendMail(addr, auth, t.from, recipients, []byte(message))
		}
		done <- err
	}()

	// Wait for completion or context cancellation
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return errors.WrapWithPlatform(errors.CodeTimeout, errors.CategoryNetwork, "context cancelled", "email", ctx.Err())
	case <-time.After(t.timeout):
		return errors.NewEmailError(errors.CodeTimeout, fmt.Sprintf("email send timeout after %v", t.timeout))
	}
}

// sendWithTLS sends email with explicit TLS
func (t *Transport) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Connect to server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Start TLS
	if err := client.StartTLS(&tls.Config{
		ServerName: t.host,
	}); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to add recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer func() { _ = writer.Close() }()

	if _, err := writer.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Shutdown shuts down the transport
func (t *Transport) Shutdown() error {
	// Email transport doesn't maintain persistent connections
	return nil
}
