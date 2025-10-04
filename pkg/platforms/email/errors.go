// Package email provides enhanced error handling for email platform
package email

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// EmailErrorType represents different types of email errors
type EmailErrorType string

const (
	// Connection errors
	ErrorTypeConnection        EmailErrorType = "connection"
	ErrorTypeTimeout          EmailErrorType = "timeout"
	ErrorTypeDNS              EmailErrorType = "dns"
	ErrorTypeNetwork          EmailErrorType = "network"

	// Authentication errors
	ErrorTypeAuth             EmailErrorType = "authentication"
	ErrorTypeCredentials      EmailErrorType = "credentials"
	ErrorTypePermission       EmailErrorType = "permission"

	// SMTP protocol errors
	ErrorTypeSMTP             EmailErrorType = "smtp"
	ErrorTypeProtocol         EmailErrorType = "protocol"
	ErrorTypeTLS              EmailErrorType = "tls"

	// Message errors
	ErrorTypeMessage          EmailErrorType = "message"
	ErrorTypeRecipient        EmailErrorType = "recipient"
	ErrorTypeSize             EmailErrorType = "size"
	ErrorTypeFormat           EmailErrorType = "format"

	// Rate limiting errors
	ErrorTypeRateLimit        EmailErrorType = "rate_limit"
	ErrorTypeQuota            EmailErrorType = "quota"

	// Server errors
	ErrorTypeServerUnavailable EmailErrorType = "server_unavailable"
	ErrorTypeServerError      EmailErrorType = "server_error"

	// Configuration errors
	ErrorTypeConfig           EmailErrorType = "configuration"
	ErrorTypeValidation       EmailErrorType = "validation"

	// Generic errors
	ErrorTypeUnknown          EmailErrorType = "unknown"
)

// EmailError represents a detailed email error with context and suggestions
type EmailError struct {
	Type        EmailErrorType `json:"type"`
	Code        string         `json:"code,omitempty"`
	Message     string         `json:"message"`
	Details     string         `json:"details,omitempty"`
	Suggestions []string       `json:"suggestions,omitempty"`
	Provider    string         `json:"provider,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
	Retryable   bool           `json:"retryable"`
	OriginalErr error          `json:"-"`
}

// Error implements the error interface
func (e *EmailError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s (%s): %s", e.Type, e.Code, e.Provider, e.Message)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Provider, e.Message)
}

// Unwrap returns the original error
func (e *EmailError) Unwrap() error {
	return e.OriginalErr
}

// IsRetryable returns true if the error might be resolved by retrying
func (e *EmailError) IsRetryable() bool {
	return e.Retryable
}

// GetSuggestions returns troubleshooting suggestions for the error
func (e *EmailError) GetSuggestions() []string {
	return e.Suggestions
}

// NewEmailError creates a new EmailError with enhanced context
func NewEmailError(errorType EmailErrorType, message string, originalErr error) *EmailError {
	emailErr := &EmailError{
		Type:        errorType,
		Message:     message,
		Timestamp:   time.Now(),
		OriginalErr: originalErr,
	}

	// Analyze the original error to provide better context
	if originalErr != nil {
		emailErr.analyzeError(originalErr)
	}

	return emailErr
}

// analyzeError analyzes the original error to provide enhanced context
func (e *EmailError) analyzeError(err error) {
	errStr := strings.ToLower(err.Error())

	// Authentication errors
	if strings.Contains(errStr, "authentication failed") ||
		strings.Contains(errStr, "535") ||
		strings.Contains(errStr, "auth") {
		e.Type = ErrorTypeAuth
		e.Code = "535"
		e.Retryable = false
		e.Suggestions = []string{
			"检查用户名和密码是否正确",
			"确认使用授权码而不是登录密码 (163, QQ等)",
			"确认使用应用专用密码 (Gmail等)",
			"检查邮箱是否开启了SMTP服务",
			"验证认证方式是否正确 (PLAIN, LOGIN等)",
		}
	}

	// Connection errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connect") {
		e.Type = ErrorTypeConnection
		e.Retryable = true
		e.Suggestions = []string{
			"检查SMTP服务器地址和端口是否正确",
			"检查网络连接",
			"检查防火墙设置",
			"确认SMTP服务是否正常运行",
			"尝试使用其他网络环境",
		}
	}

	// Timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline") {
		e.Type = ErrorTypeTimeout
		e.Retryable = true
		e.Suggestions = []string{
			"增加连接超时时间",
			"检查网络连接质量",
			"尝试在网络条件更好的环境下重试",
			"检查服务器是否过载",
		}
	}

	// TLS/SSL errors
	if strings.Contains(errStr, "tls") ||
		strings.Contains(errStr, "ssl") ||
		strings.Contains(errStr, "certificate") {
		e.Type = ErrorTypeTLS
		e.Retryable = false
		e.Suggestions = []string{
			"检查TLS/SSL配置 (UseTLS vs UseStartTLS)",
			"尝试不同的加密端口 (25, 587, 465)",
			"检查服务器证书是否有效",
			"尝试跳过证书验证 (仅测试环境)",
			"确认服务器支持的TLS版本",
		}
	}

	// DNS errors
	if strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "dns") {
		e.Type = ErrorTypeDNS
		e.Retryable = true
		e.Suggestions = []string{
			"检查SMTP服务器地址拼写",
			"检查DNS设置",
			"尝试使用IP地址而不是域名",
			"检查网络DNS配置",
		}
	}

	// Rate limiting
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many") ||
		strings.Contains(errStr, "429") {
		e.Type = ErrorTypeRateLimit
		e.Code = "429"
		e.Retryable = true
		e.Suggestions = []string{
			"降低发送频率",
			"等待一段时间后重试",
			"检查邮箱服务商的发送限制",
			"考虑分批发送邮件",
			"联系邮箱服务商提高发送配额",
		}
	}

	// Recipient errors
	if strings.Contains(errStr, "recipient") ||
		strings.Contains(errStr, "550") ||
		strings.Contains(errStr, "invalid") {
		e.Type = ErrorTypeRecipient
		e.Code = "550"
		e.Retryable = false
		e.Suggestions = []string{
			"检查收件人邮箱地址格式",
			"确认收件人邮箱存在",
			"检查收件人邮箱是否被禁用",
			"验证收件人邮箱服务商设置",
		}
	}

	// Message size errors
	if strings.Contains(errStr, "size") ||
		strings.Contains(errStr, "too large") ||
		strings.Contains(errStr, "552") {
		e.Type = ErrorTypeSize
		e.Code = "552"
		e.Retryable = false
		e.Suggestions = []string{
			"减小邮件大小",
			"压缩附件",
			"分多封邮件发送",
			"检查邮箱服务商的大小限制",
		}
	}

	// Server unavailable
	if strings.Contains(errStr, "421") ||
		strings.Contains(errStr, "service not available") {
		e.Type = ErrorTypeServerUnavailable
		e.Code = "421"
		e.Retryable = true
		e.Suggestions = []string{
			"稍后重试",
			"检查服务器状态",
			"联系邮箱服务商",
			"尝试使用备用SMTP服务器",
		}
	}

	// Analyze by error type
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			e.Type = ErrorTypeTimeout
			e.Retryable = true
		}
	}
}

// ErrorAnalyzer provides error analysis and suggestions
type ErrorAnalyzer struct {
	provider string
}

// NewErrorAnalyzer creates a new error analyzer for a specific provider
func NewErrorAnalyzer(provider string) *ErrorAnalyzer {
	return &ErrorAnalyzer{
		provider: provider,
	}
}

// AnalyzeError analyzes an error and returns an enhanced EmailError
func (ea *ErrorAnalyzer) AnalyzeError(err error) *EmailError {
	if err == nil {
		return nil
	}

	// Check if it's already an EmailError
	if emailErr, ok := err.(*EmailError); ok {
		if emailErr.Provider == "" {
			emailErr.Provider = ea.provider
		}
		return emailErr
	}

	// Create new EmailError
	emailErr := NewEmailError(ErrorTypeUnknown, err.Error(), err)
	emailErr.Provider = ea.provider

	// Add provider-specific suggestions
	ea.addProviderSpecificSuggestions(emailErr)

	return emailErr
}

// addProviderSpecificSuggestions adds provider-specific troubleshooting suggestions
func (ea *ErrorAnalyzer) addProviderSpecificSuggestions(emailErr *EmailError) {
	provider := strings.ToLower(ea.provider)

	switch emailErr.Type {
	case ErrorTypeAuth:
		switch provider {
		case "gmail":
			emailErr.Suggestions = append(emailErr.Suggestions,
				"Gmail: 确保开启了两步验证",
				"Gmail: 使用应用专用密码而不是账户密码",
				"Gmail: 检查Google账户安全设置",
			)
		case "163", "126", "yeah":
			emailErr.Suggestions = append(emailErr.Suggestions,
				fmt.Sprintf("%s: 在邮箱设置中开启SMTP服务", provider),
				fmt.Sprintf("%s: 生成并使用授权码", provider),
				fmt.Sprintf("%s: 确认授权码输入正确", provider),
			)
		case "qq":
			emailErr.Suggestions = append(emailErr.Suggestions,
				"QQ邮箱: 在邮箱设置中开启SMTP/POP3服务",
				"QQ邮箱: 生成并使用授权码",
				"QQ邮箱: 检查QQ安全中心设置",
			)
		}

	case ErrorTypeConnection:
		switch provider {
		case "gmail":
			emailErr.Suggestions = append(emailErr.Suggestions,
				"Gmail: 确认使用smtp.gmail.com:587",
				"Gmail: 检查网络是否能访问Google服务",
			)
		case "163":
			emailErr.Suggestions = append(emailErr.Suggestions,
				"163: 确认使用smtp.163.com:25或587",
				"163: 检查运营商是否屏蔽端口25",
			)
		case "qq":
			emailErr.Suggestions = append(emailErr.Suggestions,
				"QQ: 确认使用smtp.qq.com:587或465",
				"QQ: 检查企业网络防火墙设置",
			)
		}

	case ErrorTypeTLS:
		switch provider {
		case "gmail":
			emailErr.Suggestions = append(emailErr.Suggestions,
				"Gmail: 使用STARTTLS而不是直接TLS",
				"Gmail: 端口587 + STARTTLS是推荐配置",
			)
		case "163", "126":
			emailErr.Suggestions = append(emailErr.Suggestions,
				fmt.Sprintf("%s: 端口25通常使用STARTTLS", provider),
				fmt.Sprintf("%s: 避免使用端口465 (SSL)", provider),
			)
		}
	}
}

// GetCommonSolutions returns common solutions for email issues
func GetCommonSolutions(errorType EmailErrorType) []string {
	solutions := map[EmailErrorType][]string{
		ErrorTypeAuth: {
			"确认邮箱账号和密码正确",
			"检查是否需要使用授权码而不是登录密码",
			"确认邮箱已开启SMTP服务",
			"检查认证方式配置 (PLAIN, LOGIN, CRAM-MD5)",
		},
		ErrorTypeConnection: {
			"检查网络连接",
			"确认SMTP服务器地址和端口",
			"检查防火墙设置",
			"尝试使用不同的网络环境",
		},
		ErrorTypeTLS: {
			"检查TLS/SSL配置",
			"尝试STARTTLS而不是直接TLS",
			"确认端口和加密方式匹配",
			"检查服务器证书",
		},
		ErrorTypeTimeout: {
			"增加连接超时时间",
			"检查网络质量",
			"稍后重试",
			"检查服务器负载",
		},
		ErrorTypeRateLimit: {
			"降低发送频率",
			"等待后重试",
			"分批发送邮件",
			"联系服务商提高配额",
		},
	}

	if sols, exists := solutions[errorType]; exists {
		return sols
	}
	return []string{"检查配置和网络连接", "查看详细错误日志", "联系技术支持"}
}

// FormatErrorForUser formats an error message for end-user display
func FormatErrorForUser(err error) string {
	if emailErr, ok := err.(*EmailError); ok {
		msg := fmt.Sprintf("邮件发送失败: %s", emailErr.Message)

		if len(emailErr.Suggestions) > 0 {
			msg += "\n\n建议解决方案:"
			for i, suggestion := range emailErr.Suggestions {
				if i < 3 { // Show only top 3 suggestions
					msg += fmt.Sprintf("\n• %s", suggestion)
				}
			}
		}

		if emailErr.Retryable {
			msg += "\n\n此错误可能是临时的，建议稍后重试。"
		}

		return msg
	}

	return fmt.Sprintf("邮件发送失败: %s", err.Error())
}

// IsTemporaryError checks if an error is temporary and retryable
func IsTemporaryError(err error) bool {
	if emailErr, ok := err.(*EmailError); ok {
		return emailErr.Retryable
	}

	// Check for known temporary error patterns
	errStr := strings.ToLower(err.Error())
	temporaryPatterns := []string{
		"timeout",
		"connection refused",
		"temporary failure",
		"try again",
		"rate limit",
		"421", // Service not available
		"450", // Requested mail action not taken: mailbox unavailable
		"451", // Requested action aborted: local error
	}

	for _, pattern := range temporaryPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// GetRetryDelay calculates appropriate retry delay based on error type
func GetRetryDelay(err error, attempt int) time.Duration {
	baseDelay := time.Second * 30 // Base delay of 30 seconds

	if emailErr, ok := err.(*EmailError); ok {
		switch emailErr.Type {
		case ErrorTypeRateLimit:
			// Longer delay for rate limits
			baseDelay = time.Minute * 5
		case ErrorTypeTimeout, ErrorTypeConnection:
			// Medium delay for connection issues
			baseDelay = time.Minute * 2
		case ErrorTypeServerUnavailable:
			// Variable delay based on server issues
			baseDelay = time.Minute * 3
		default:
			// Standard delay for other errors
			baseDelay = time.Minute
		}
	}

	// Exponential backoff with jitter
	delay := baseDelay * time.Duration(1<<uint(attempt))
	if delay > time.Hour {
		delay = time.Hour // Cap at 1 hour
	}

	return delay
}