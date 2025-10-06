// Package email provides SMTP email sending functionality for NotifyHub
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/target"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// SMTPSender handles sending emails via SMTP
type SMTPSender struct {
	config      *Config
	authHandler *AuthHandler
	msgBuilder  *MessageBuilder
	logger      logger.Logger
}

// NewSMTPSender creates a new SMTP email sender
func NewSMTPSender(config *Config, logger logger.Logger) (*SMTPSender, error) {
	if config == nil {
		return nil, fmt.Errorf("email config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid email config: %w", err)
	}

	return &SMTPSender{
		config:      config,
		authHandler: NewAuthHandler(config),
		msgBuilder:  NewMessageBuilder(config),
		logger:      logger,
	}, nil
}

// SendMessage sends an email message using SMTP
func (s *SMTPSender) SendMessage(ctx context.Context, msg *message.Message, targets []target.Target) error {
	s.logger.Info("🚀 开始发送SMTP邮件", "targets", len(targets), "subject", msg.Title, "smtp_server", s.config.GetServerAddress())

	// Create error analyzer
	errorAnalyzer := NewErrorAnalyzer(s.detectProvider())

	// Build email message
	emailMsg, err := s.msgBuilder.BuildMessage(msg, targets)
	if err != nil {
		s.logger.Error("构建邮件消息失败", "error", err)
		enhancedErr := NewEmailError(ErrorTypeMessage, "邮件消息构建失败", err)
		return enhancedErr
	}

	// Validate email message
	if err := emailMsg.Validate(); err != nil {
		s.logger.Error("邮件消息验证失败", "error", err)
		enhancedErr := NewEmailError(ErrorTypeValidation, "邮件消息验证失败", err)
		return enhancedErr
	}

	s.logger.Debug("邮件消息构建成功",
		"from", emailMsg.From,
		"to", emailMsg.To,
		"subject", emailMsg.Subject,
		"text_size", len(emailMsg.TextBody),
		"html_size", len(emailMsg.HTMLBody))

	// Send email with enhanced error handling
	if err := s.sendSMTP(ctx, emailMsg); err != nil {
		s.logger.Error("SMTP邮件发送失败", "error", err, "smtp_server", s.config.GetServerAddress())

		// Analyze and enhance the error
		enhancedErr := errorAnalyzer.AnalyzeError(err)

		// Log detailed error information
		s.logger.Error("邮件发送错误详情",
			"error_type", enhancedErr.Type,
			"error_code", enhancedErr.Code,
			"retryable", enhancedErr.Retryable,
			"suggestions", enhancedErr.Suggestions)

		return enhancedErr
	}

	s.logger.Info("✅ SMTP邮件发送成功",
		"to", emailMsg.To,
		"smtp_server", s.config.GetServerAddress(),
		"message_id", fmt.Sprintf("smtp_%d", time.Now().UnixNano()))
	return nil
}

// sendSMTP sends the email via SMTP
func (s *SMTPSender) sendSMTP(ctx context.Context, emailMsg *Message) error {
	s.logger.Debug("连接SMTP服务器", "host", s.config.SMTPHost, "port", s.config.SMTPPort)

	// Get RFC2822 format message
	messageBytes, err := emailMsg.ToRFC2822()
	if err != nil {
		return fmt.Errorf("failed to convert message to RFC2822: %w", err)
	}

	// Get all recipients
	recipients := emailMsg.GetAllRecipients()
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	s.logger.Debug("邮件内容大小", "bytes", len(messageBytes), "recipients", len(recipients))

	// Setup SMTP connection with context
	if err := s.sendWithContext(ctx, emailMsg.From, recipients, messageBytes); err != nil {
		return err
	}

	return nil
}

// sendWithContext sends email with context support
func (s *SMTPSender) sendWithContext(ctx context.Context, from string, to []string, message []byte) error {
	serverAddr := s.config.GetServerAddress()
	s.logger.Debug("正在连接SMTP服务器", "server", serverAddr)

	// Create a channel to receive the result
	resultChan := make(chan error, 1)

	go func() {
		defer close(resultChan)

		// Connect to SMTP server
		client, err := s.connectSMTP()
		if err != nil {
			resultChan <- fmt.Errorf("failed to connect to SMTP server: %w", err)
			return
		}
		defer func() {
			if closeErr := client.Close(); closeErr != nil {
				s.logger.Warn("SMTP客户端关闭失败", "error", closeErr)
			}
		}()

		s.logger.Debug("✅ SMTP连接成功")

		// Set sender (extract email address from formatted string)
		senderAddress := s.extractEmailAddress(from)
		if err := client.Mail(senderAddress); err != nil {
			resultChan <- fmt.Errorf("failed to set sender: %w", err)
			return
		}

		s.logger.Debug("✅ 设置发件人成功", "from", from)

		// Set recipients
		for _, recipient := range to {
			if err := client.Rcpt(recipient); err != nil {
				resultChan <- fmt.Errorf("failed to set recipient %s: %w", recipient, err)
				return
			}
		}

		s.logger.Debug("✅ 设置收件人成功", "count", len(to))

		// Send message
		wc, err := client.Data()
		if err != nil {
			resultChan <- fmt.Errorf("failed to initiate data transfer: %w", err)
			return
		}

		if _, err := wc.Write(message); err != nil {
			_ = wc.Close() // Best effort close, original error is more important
			resultChan <- fmt.Errorf("failed to write message data: %w", err)
			return
		}

		if err := wc.Close(); err != nil {
			resultChan <- fmt.Errorf("failed to close data writer: %w", err)
			return
		}

		s.logger.Debug("✅ 邮件数据发送成功")
		resultChan <- nil
	}()

	// Wait for result or context cancellation
	select {
	case err := <-resultChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("email sending cancelled: %w", ctx.Err())
	}
}

// connectSMTP establishes an SMTP connection with authentication
func (s *SMTPSender) connectSMTP() (*smtp.Client, error) {
	serverAddr := s.config.GetServerAddress()

	// Connect to server
	var client *smtp.Client
	var err error

	if s.config.UseTLS {
		// Direct TLS connection (port 465)
		s.logger.Debug("使用直接TLS连接")
		tlsConfig := s.authHandler.GetTLSConfig()
		conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("TLS dial failed: %w", err)
		}

		client, err = smtp.NewClient(conn, s.config.SMTPHost)
		if err != nil {
			_ = conn.Close() // Best effort close, original error is more important
			return nil, fmt.Errorf("SMTP client creation failed: %w", err)
		}
	} else {
		// Plain connection (port 587 with STARTTLS)
		s.logger.Debug("使用普通连接")
		client, err = smtp.Dial(serverAddr)
		if err != nil {
			return nil, fmt.Errorf("SMTP dial failed: %w", err)
		}
	}

	// Set EHLO/HELO
	hostname := s.config.LocalName
	if hostname == "" {
		hostname = "localhost"
	}

	if err := client.Hello(hostname); err != nil {
		_ = client.Close() // Best effort close, original error is more important
		return nil, fmt.Errorf("EHLO/HELO failed: %w", err)
	}

	s.logger.Debug("✅ SMTP握手成功", "hostname", hostname)

	// Start TLS if using STARTTLS
	if s.config.UseStartTLS && !s.config.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			s.logger.Debug("启用STARTTLS")
			tlsConfig := s.authHandler.GetTLSConfig()
			if err := client.StartTLS(tlsConfig); err != nil {
				_ = client.Close() // Best effort close, original error is more important
				return nil, fmt.Errorf("STARTTLS failed: %w", err)
			}
			s.logger.Debug("✅ STARTTLS成功")
		} else {
			s.logger.Warn("服务器不支持STARTTLS")
		}
	}

	// Authenticate if credentials are provided
	if s.config.IsAuthRequired() {
		s.logger.Debug("开始SMTP认证", "username", s.config.Username)
		auth := s.authHandler.GetAuth()
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				_ = client.Close() // Best effort close, original error is more important
				return nil, fmt.Errorf("SMTP authentication failed: %w", err)
			}
			s.logger.Debug("✅ SMTP认证成功")
		}
	}

	return client, nil
}

// TestConnection tests the SMTP connection
func (s *SMTPSender) TestConnection(ctx context.Context) error {
	s.logger.Info("测试SMTP连接")

	resultChan := make(chan error, 1)

	go func() {
		defer close(resultChan)

		client, err := s.connectSMTP()
		if err != nil {
			resultChan <- err
			return
		}
		defer func() {
			if closeErr := client.Close(); closeErr != nil {
				s.logger.Warn("Failed to close SMTP client", "error", closeErr)
			}
		}()

		// Test by sending NOOP command
		if err := client.Noop(); err != nil {
			resultChan <- fmt.Errorf("NOOP command failed: %w", err)
			return
		}

		resultChan <- nil
	}()

	select {
	case err := <-resultChan:
		if err != nil {
			s.logger.Error("SMTP连接测试失败", "error", err)
			return err
		}
		s.logger.Info("✅ SMTP连接测试成功")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("connection test cancelled: %w", ctx.Err())
	}
}

// GetServerCapabilities returns SMTP server capabilities
func (s *SMTPSender) GetServerCapabilities(ctx context.Context) (map[string]string, error) {
	resultChan := make(chan map[string]string, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errorChan)

		client, err := s.connectSMTP()
		if err != nil {
			errorChan <- err
			return
		}
		defer func() {
			if closeErr := client.Close(); closeErr != nil {
				s.logger.Warn("Failed to close SMTP client", "error", closeErr)
			}
		}()

		capabilities := make(map[string]string)

		// Check AUTH methods
		if ok, param := client.Extension("AUTH"); ok {
			capabilities["AUTH"] = param
		}

		// Check SIZE limit
		if ok, param := client.Extension("SIZE"); ok {
			capabilities["SIZE"] = param
		}

		// Check STARTTLS support
		if ok, _ := client.Extension("STARTTLS"); ok {
			capabilities["STARTTLS"] = "supported"
		}

		// Check 8BITMIME support
		if ok, _ := client.Extension("8BITMIME"); ok {
			capabilities["8BITMIME"] = "supported"
		}

		resultChan <- capabilities
	}()

	select {
	case capabilities := <-resultChan:
		return capabilities, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("capabilities check cancelled: %w", ctx.Err())
	}
}

// Close closes the SMTP sender (placeholder for future connection pooling)
func (s *SMTPSender) Close() error {
	s.logger.Debug("关闭SMTP发送器")
	return nil
}

// ConvertNotifyHubConfig converts NotifyHub config to email config
func ConvertNotifyHubConfig(nhConfig interface{}) (*Config, error) {
	// Handle different config types that might be passed from NotifyHub
	switch cfg := nhConfig.(type) {
	case *Config:
		return cfg, nil
	case map[string]interface{}:
		// Convert from map (common in factory patterns)
		config := NewConfig()

		if host, ok := cfg["host"].(string); ok {
			config.SMTPHost = host
		}
		if port, ok := cfg["port"].(int); ok {
			config.SMTPPort = port
		}
		if username, ok := cfg["username"].(string); ok {
			config.Username = username
		}
		if password, ok := cfg["password"].(string); ok {
			config.Password = password
		}
		if from, ok := cfg["from"].(string); ok {
			config.From = from
		}
		if useTLS, ok := cfg["use_tls"].(bool); ok {
			config.UseTLS = useTLS
		}
		if useStartTLS, ok := cfg["use_starttls"].(bool); ok {
			config.UseStartTLS = useStartTLS
		}

		return config, nil
	default:
		return nil, fmt.Errorf("unsupported config type: %T", nhConfig)
	}
}

// detectProvider detects the email provider based on SMTP host
func (s *SMTPSender) detectProvider() string {
	host := strings.ToLower(s.config.SMTPHost)

	switch {
	case strings.Contains(host, "gmail.com"):
		return "Gmail"
	case strings.Contains(host, "163.com"):
		return "163邮箱"
	case strings.Contains(host, "126.com"):
		return "126邮箱"
	case strings.Contains(host, "yeah.net"):
		return "Yeah邮箱"
	case strings.Contains(host, "qq.com") && !strings.Contains(host, "exmail"):
		return "QQ邮箱"
	case strings.Contains(host, "exmail.qq.com"):
		return "腾讯企业邮箱"
	case strings.Contains(host, "outlook.com") || strings.Contains(host, "hotmail"):
		return "Outlook"
	case strings.Contains(host, "yahoo.com"):
		return "Yahoo"
	case strings.Contains(host, "yahoo.co.jp"):
		return "Yahoo Japan"
	case strings.Contains(host, "zoho.com"):
		return "Zoho"
	case strings.Contains(host, "sina.com"):
		return "新浪邮箱"
	case strings.Contains(host, "sohu.com"):
		return "搜狐邮箱"
	case strings.Contains(host, "mxhichina.com"):
		return "阿里云邮箱"
	case strings.Contains(host, "127.0.0.1") && s.config.SMTPPort == 1025:
		return "ProtonMail"
	default:
		return "通用SMTP"
	}
}

// extractEmailAddress extracts email address from formatted string
// Handles both "email@domain.com" and "Display Name <email@domain.com>" formats
func (s *SMTPSender) extractEmailAddress(address string) string {
	// Check if it's in "Display Name <email@domain.com>" format
	if strings.Contains(address, "<") && strings.Contains(address, ">") {
		start := strings.Index(address, "<")
		end := strings.Index(address, ">")
		if start != -1 && end != -1 && end > start {
			return address[start+1 : end]
		}
	}

	// Return as-is if it's already a plain email address
	return strings.TrimSpace(address)
}
