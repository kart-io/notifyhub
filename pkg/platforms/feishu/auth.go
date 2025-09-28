// Package feishu provides authentication functionality for Feishu platform
// This file handles signature generation and keyword verification for Feishu webhooks
package feishu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

// SecurityMode represents different Feishu security configurations
type SecurityMode string

const (
	SecurityModeNone              SecurityMode = "no_security"
	SecurityModeSignatureOnly     SecurityMode = "signature_only"
	SecurityModeKeywordsOnly      SecurityMode = "keywords_only"
	SecurityModeSignatureKeywords SecurityMode = "signature_and_keywords"
)

// AuthError represents authentication-related errors with detailed diagnostics
type AuthError struct {
	Code      string
	Message   string
	Details   map[string]interface{}
	Timestamp time.Time
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// AuthHandler handles authentication logic for Feishu platform
type AuthHandler struct {
	secret   string
	keywords []string
	mode     SecurityMode
	timeoutWindow time.Duration
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(secret string, keywords []string) *AuthHandler {
	handler := &AuthHandler{
		secret:   secret,
		keywords: keywords,
		timeoutWindow: 5 * time.Minute, // Default 5-minute window for timestamp validation
	}
	handler.mode = handler.determineSecurityMode()
	return handler
}

// NewAuthHandlerWithTimeout creates a new authentication handler with custom timeout
func NewAuthHandlerWithTimeout(secret string, keywords []string, timeout time.Duration) *AuthHandler {
	handler := &AuthHandler{
		secret:   secret,
		keywords: keywords,
		timeoutWindow: timeout,
	}
	handler.mode = handler.determineSecurityMode()
	return handler
}

// AddAuth adds authentication (signature and keyword processing) to the message
func (a *AuthHandler) AddAuth(feishuMsg *FeishuMessage) error {
	switch a.mode {
	case SecurityModeNone:
		return nil
	case SecurityModeSignatureOnly:
		a.addSignature(feishuMsg)
		return nil
	case SecurityModeKeywordsOnly:
		return a.processKeywords(feishuMsg)
	case SecurityModeSignatureKeywords:
		if err := a.processKeywords(feishuMsg); err != nil {
			return a.newAuthError("KEYWORD_PROCESSING_FAILED", fmt.Sprintf("Keyword processing failed: %v", err), nil)
		}
		a.addSignature(feishuMsg)
		return nil
	default:
		return a.newAuthError("UNKNOWN_SECURITY_MODE", fmt.Sprintf("Unknown mode: %s", a.mode), nil)
	}
}

// GetSecurityMode returns the current security mode
func (a *AuthHandler) GetSecurityMode() SecurityMode {
	return a.mode
}

// determineSecurityMode determines which security mode is configured
func (a *AuthHandler) determineSecurityMode() SecurityMode {
	hasSignature := a.secret != ""
	hasKeywords := len(a.keywords) > 0

	switch {
	case hasSignature && hasKeywords:
		return SecurityModeSignatureKeywords
	case hasSignature && !hasKeywords:
		return SecurityModeSignatureOnly
	case !hasSignature && hasKeywords:
		return SecurityModeKeywordsOnly
	default:
		return SecurityModeNone
	}
}

// addSignature adds signature fields to the message
func (a *AuthHandler) addSignature(feishuMsg *FeishuMessage) {
	if a.secret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		sign := a.generateSign(timestamp)
		feishuMsg.Sign = sign
		feishuMsg.Timestamp = timestamp
	}
}

// generateSign generates HMAC-SHA256 signature for Feishu webhook
// According to Feishu official documentation:
// 1. stringToSign = timestamp + "\n" + secret
// 2. signature = base64(hmac_sha256(stringToSign, ""))
func (a *AuthHandler) generateSign(timestamp string) string {
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, a.secret)
	hash := hmac.New(sha256.New, []byte(stringToSign))
	hash.Write([]byte("")) // Feishu uses empty string as data
	signature := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return signature
}

// VerifySignature verifies the signature for incoming webhook requests
func (a *AuthHandler) VerifySignature(timestamp, signature string) error {
	if a.secret == "" {
		return a.newAuthError("NO_SECRET_CONFIGURED", "No secret configured", nil)
	}

	// Validate timestamp format and freshness
	if err := a.validateTimestamp(timestamp); err != nil {
		return err
	}

	// Generate expected signature and verify using secure comparison
	expectedSignature := a.generateSign(timestamp)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return a.newAuthError("SIGNATURE_VERIFICATION_FAILED", "Signature mismatch",
			map[string]interface{}{"timestamp": timestamp, "sig_len": len(signature)})
	}

	return nil
}

// validateTimestamp validates timestamp format and checks for replay attacks
func (a *AuthHandler) validateTimestamp(timestampStr string) error {
	if timestampStr == "" {
		return a.newAuthError("EMPTY_TIMESTAMP", "Timestamp required", nil)
	}

	// Parse timestamp
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return a.newAuthError("INVALID_TIMESTAMP_FORMAT", fmt.Sprintf("Invalid timestamp: %v", err), nil)
	}

	// Check timestamp freshness for replay attack prevention
	messageTime := time.Unix(timestamp, 0)
	now := time.Now()
	timeDiff := now.Sub(messageTime)

	// Allow some tolerance for clock skew (both past and future)
	if timeDiff < -time.Minute {
		return a.newAuthError("TIMESTAMP_TOO_FUTURE", "Timestamp too far in future",
			map[string]interface{}{"diff_seconds": int(timeDiff.Seconds())})
	}

	if timeDiff > a.timeoutWindow {
		return a.newAuthError("TIMESTAMP_EXPIRED", "Timestamp too old (replay attack?)",
			map[string]interface{}{"diff_seconds": int(timeDiff.Seconds())})
	}

	return nil
}

// processKeywords processes keyword requirements for the message
func (a *AuthHandler) processKeywords(feishuMsg *FeishuMessage) error {
	if len(a.keywords) == 0 {
		return nil
	}

	// Note: This method requires access to MessageBuilder to extract and modify text
	// For now, we'll return nil as the keyword processing is handled in the original message building
	// This is a placeholder for the keyword processing logic that should be moved here
	// from the original sender.go file

	return nil
}

// ContainsRequiredKeyword checks if the message text contains any of the required keywords
func (a *AuthHandler) ContainsRequiredKeyword(messageText string) bool {
	if messageText == "" || len(a.keywords) == 0 {
		return false
	}

	messageTextLower := strings.ToLower(messageText)
	for _, keyword := range a.keywords {
		keywordLower := strings.ToLower(strings.TrimSpace(keyword))
		if keywordLower != "" && strings.Contains(messageTextLower, keywordLower) {
			return true
		}
	}

	return false
}

// GetFirstKeyword returns the first configured keyword for automatic addition
func (a *AuthHandler) GetFirstKeyword() string {
	if len(a.keywords) > 0 {
		return a.keywords[0]
	}
	return ""
}

// ProcessKeywordRequirement processes keyword requirement for the message
// This method works with MessageBuilder to handle keyword verification and addition
func (a *AuthHandler) ProcessKeywordRequirement(feishuMsg *FeishuMessage, msg *message.Message, builder *MessageBuilder) error {
	if len(a.keywords) == 0 {
		return nil
	}

	// Extract message text for keyword checking
	messageText := builder.ExtractMessageText(feishuMsg, msg)

	// Check if message already contains required keyword
	if a.ContainsRequiredKeyword(messageText) {
		return nil
	}

	// Message doesn't contain required keyword, add the first one
	keywordToAdd := a.GetFirstKeyword()
	if keywordToAdd == "" {
		return a.newAuthError("NO_KEYWORDS_CONFIGURED", "No keywords configured", nil)
	}

	if err := builder.AddKeywordToMessage(feishuMsg, keywordToAdd); err != nil {
		return a.newAuthError("KEYWORD_ADDITION_FAILED", fmt.Sprintf("Failed to add keyword: %v", err), nil)
	}

	return nil
}

// ValidateKeywordRequirement validates keyword requirement without modifying the message
func (a *AuthHandler) ValidateKeywordRequirement(messageText string) error {
	if len(a.keywords) == 0 {
		return nil // No keyword requirement
	}

	if messageText == "" {
		return a.newAuthError("EMPTY_MESSAGE_TEXT", "Message text empty but keywords required",
			map[string]interface{}{"keywords": a.keywords})
	}

	if !a.ContainsRequiredKeyword(messageText) {
		return a.newAuthError("KEYWORD_REQUIREMENT_NOT_MET", "Message missing required keywords",
			map[string]interface{}{"keywords": a.keywords, "preview": a.getTextPreview(messageText, 100)})
	}

	return nil
}

// newAuthError creates a new AuthError with diagnostic information
func (a *AuthHandler) newAuthError(code, message string, details map[string]interface{}) *AuthError {
	if details == nil {
		details = make(map[string]interface{})
	}
	// Always include common diagnostic information
	details["mode"] = a.mode
	details["has_secret"] = a.secret != ""
	details["keywords_count"] = len(a.keywords)
	return &AuthError{Code: code, Message: message, Details: details, Timestamp: time.Now()}
}

// getTextPreview returns a preview of text for diagnostic purposes
func (a *AuthHandler) getTextPreview(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

// GetDiagnosticInfo returns diagnostic information about the auth handler
func (a *AuthHandler) GetDiagnosticInfo() map[string]interface{} {
	return map[string]interface{}{
		"security_mode":         a.mode,
		"has_secret":            a.secret != "",
		"secret_length":         len(a.secret),
		"keywords_configured":   a.keywords,
		"keywords_count":        len(a.keywords),
		"timeout_window_seconds": int(a.timeoutWindow.Seconds()),
		"supported_modes":       []SecurityMode{SecurityModeNone, SecurityModeSignatureOnly, SecurityModeKeywordsOnly, SecurityModeSignatureKeywords},
	}
}