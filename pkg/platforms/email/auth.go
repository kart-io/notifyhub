// Package email provides email authentication functionality for NotifyHub
package email

import (
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// AuthHandler handles email authentication
type AuthHandler struct {
	config *Config
}

// NewAuthHandler creates a new email authentication handler
func NewAuthHandler(config *Config) *AuthHandler {
	return &AuthHandler{
		config: config,
	}
}

// GetAuth returns the appropriate smtp.Auth based on configuration
func (a *AuthHandler) GetAuth() smtp.Auth {
	if !a.config.IsAuthRequired() {
		return nil
	}

	switch strings.ToLower(a.config.AuthMethod) {
	case "plain":
		return smtp.PlainAuth("", a.config.Username, a.config.Password, a.config.SMTPHost)

	case "login":
		return &LoginAuth{
			username: a.config.Username,
			password: a.config.Password,
		}

	case "cram-md5":
		return smtp.CRAMMD5Auth(a.config.Username, a.config.Password)

	default:
		// Default to PLAIN auth
		return smtp.PlainAuth("", a.config.Username, a.config.Password, a.config.SMTPHost)
	}
}

// GetTLSConfig returns the TLS configuration
func (a *AuthHandler) GetTLSConfig() *tls.Config {
	return &tls.Config{
		ServerName:         a.config.SMTPHost,
		InsecureSkipVerify: a.config.SkipCertVerify,
	}
}

// ValidateAuth validates the authentication configuration
func (a *AuthHandler) ValidateAuth() error {
	if !a.config.IsAuthRequired() {
		return nil
	}

	// Check if username and password are provided
	if a.config.Username == "" {
		return fmt.Errorf("username is required for authentication")
	}

	if a.config.Password == "" {
		return fmt.Errorf("password is required for authentication")
	}

	// Validate auth method
	validMethods := []string{"plain", "login", "cram-md5"}
	method := strings.ToLower(a.config.AuthMethod)

	for _, validMethod := range validMethods {
		if method == validMethod {
			return nil
		}
	}

	return fmt.Errorf("invalid auth method: %s (supported: %s)",
		a.config.AuthMethod, strings.Join(validMethods, ", "))
}

// TestAuth tests the authentication by connecting to the SMTP server
func (a *AuthHandler) TestAuth() error {
	// Connect to the SMTP server
	client, err := a.connectSMTP()
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Test authentication
	auth := a.GetAuth()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	return nil
}

// connectSMTP establishes a connection to the SMTP server
func (a *AuthHandler) connectSMTP() (*smtp.Client, error) {
	// Connect to server
	client, err := smtp.Dial(a.config.GetServerAddress())
	if err != nil {
		return nil, err
	}

	// Set HELO/EHLO
	hostname := a.config.LocalName
	if hostname == "" {
		hostname = "localhost"
	}

	if a.config.Helo != "" {
		if err := client.Hello(a.config.Helo); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("HELO failed: %w", err)
		}
	} else {
		if err := client.Hello(hostname); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("EHLO/HELO failed: %w", err)
		}
	}

	// Start TLS if required
	if a.config.UseTLS || a.config.UseStartTLS {
		tlsConfig := a.GetTLSConfig()

		if a.config.UseStartTLS {
			// Check if STARTTLS is supported
			if ok, _ := client.Extension("STARTTLS"); ok {
				if err := client.StartTLS(tlsConfig); err != nil {
					_ = client.Close()
					return nil, fmt.Errorf("STARTTLS failed: %w", err)
				}
			}
		}
	}

	return client, nil
}

// LoginAuth implements the LOGIN authentication mechanism
type LoginAuth struct {
	username string
	password string
}

// Start implements smtp.Auth interface
func (a *LoginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

// Next implements smtp.Auth interface
func (a *LoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, fmt.Errorf("unknown LOGIN challenge: %s", fromServer)
		}
	}
	return nil, nil
}

// CRAMMD5Auth creates a CRAM-MD5 auth
func CRAMMD5Auth(username, secret string) smtp.Auth {
	return &cramMD5Auth{username, secret}
}

type cramMD5Auth struct {
	username string
	secret   string
}

func (a *cramMD5Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "CRAM-MD5", nil, nil
}

func (a *cramMD5Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// Compute HMAC-MD5
		mac := md5.New()
		mac.Write([]byte(a.secret))
		digest := mac.Sum(nil)

		// Convert to hex
		hexDigest := fmt.Sprintf("%x", digest)

		// Create response
		response := fmt.Sprintf("%s %s", a.username, hexDigest)
		return []byte(response), nil
	}
	return nil, nil
}

// AuthMethod represents supported authentication methods
type AuthMethod string

const (
	AuthMethodPlain   AuthMethod = "plain"
	AuthMethodLogin   AuthMethod = "login"
	AuthMethodCRAMMD5 AuthMethod = "cram-md5"
)

// GetSupportedAuthMethods returns list of supported authentication methods
func GetSupportedAuthMethods() []AuthMethod {
	return []AuthMethod{
		AuthMethodPlain,
		AuthMethodLogin,
		AuthMethodCRAMMD5,
	}
}

// IsAuthMethodSupported checks if an authentication method is supported
func IsAuthMethodSupported(method string) bool {
	for _, supported := range GetSupportedAuthMethods() {
		if strings.EqualFold(string(supported), method) {
			return true
		}
	}
	return false
}

// DetectAuthMethods detects supported authentication methods from SMTP server
func (a *AuthHandler) DetectAuthMethods() ([]string, error) {
	client, err := a.connectSMTP()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Get AUTH extension
	if ok, param := client.Extension("AUTH"); ok {
		methods := strings.Fields(param)
		return methods, nil
	}

	return nil, fmt.Errorf("AUTH extension not supported by server")
}

// AuthInfo contains authentication information
type AuthInfo struct {
	Method             string   `json:"method"`
	Username           string   `json:"username"`
	IsAuthenticated    bool     `json:"is_authenticated"`
	ServerCapabilities []string `json:"server_capabilities,omitempty"`
}

// GetAuthInfo returns current authentication information
func (a *AuthHandler) GetAuthInfo() *AuthInfo {
	info := &AuthInfo{
		Method:          a.config.AuthMethod,
		Username:        a.config.Username,
		IsAuthenticated: a.config.IsAuthRequired(),
	}

	// Try to get server capabilities
	if capabilities, err := a.DetectAuthMethods(); err == nil {
		info.ServerCapabilities = capabilities
	}

	return info
}

// EncryptionInfo contains encryption information
type EncryptionInfo struct {
	UseTLS      bool   `json:"use_tls"`
	UseStartTLS bool   `json:"use_starttls"`
	TLSVersion  string `json:"tls_version,omitempty"`
	CipherSuite string `json:"cipher_suite,omitempty"`
}

// GetEncryptionInfo returns current encryption information
func (a *AuthHandler) GetEncryptionInfo() *EncryptionInfo {
	return &EncryptionInfo{
		UseTLS:      a.config.UseTLS,
		UseStartTLS: a.config.UseStartTLS,
	}
}

// SecurityLevel represents the security level of the connection
type SecurityLevel int

const (
	SecurityLevelNone SecurityLevel = iota
	SecurityLevelTLS
	SecurityLevelStartTLS
	SecurityLevelBoth
)

// GetSecurityLevel returns the configured security level
func (a *AuthHandler) GetSecurityLevel() SecurityLevel {
	if a.config.UseTLS && a.config.UseStartTLS {
		return SecurityLevelBoth
	} else if a.config.UseTLS {
		return SecurityLevelTLS
	} else if a.config.UseStartTLS {
		return SecurityLevelStartTLS
	}
	return SecurityLevelNone
}

// String returns string representation of security level
func (s SecurityLevel) String() string {
	switch s {
	case SecurityLevelNone:
		return "none"
	case SecurityLevelTLS:
		return "tls"
	case SecurityLevelStartTLS:
		return "starttls"
	case SecurityLevelBoth:
		return "tls+starttls"
	default:
		return "unknown"
	}
}

// ValidateSecurityLevel validates if the security level is appropriate
func (a *AuthHandler) ValidateSecurityLevel() error {
	level := a.GetSecurityLevel()

	// Warn about insecure configurations
	if level == SecurityLevelNone && a.config.IsAuthRequired() {
		return fmt.Errorf("authentication is configured but no encryption is enabled - this is insecure")
	}

	// Check for common secure ports
	if a.config.SMTPPort == 465 && !a.config.UseTLS {
		return fmt.Errorf("port 465 typically requires TLS encryption")
	}

	if a.config.SMTPPort == 587 && level == SecurityLevelNone {
		return fmt.Errorf("port 587 typically requires STARTTLS encryption")
	}

	return nil
}

// GetRecommendedSecurityLevel returns recommended security settings for common providers
func GetRecommendedSecurityLevel(smtpHost string) (bool, bool) {
	host := strings.ToLower(smtpHost)

	// Gmail
	if strings.Contains(host, "gmail.com") {
		return false, true // STARTTLS
	}

	// Outlook/Hotmail
	if strings.Contains(host, "outlook") || strings.Contains(host, "hotmail") {
		return false, true // STARTTLS
	}

	// Yahoo
	if strings.Contains(host, "yahoo") {
		return false, true // STARTTLS
	}

	// SendGrid
	if strings.Contains(host, "sendgrid") {
		return false, true // STARTTLS
	}

	// NetEase (163.com, 126.com, yeah.net)
	if strings.Contains(host, "163.com") || strings.Contains(host, "126.com") || strings.Contains(host, "yeah.net") {
		return false, true // STARTTLS
	}

	// QQ Mail
	if strings.Contains(host, "qq.com") {
		return false, true // STARTTLS
	}

	// Sina Mail
	if strings.Contains(host, "sina.com") || strings.Contains(host, "sina.cn") {
		return false, true // STARTTLS
	}

	// Default: STARTTLS
	return false, true
}
