// Package webhook provides webhook authentication functionality for NotifyHub
package webhook

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"strings"
	"time"
)

// AuthHandler handles webhook authentication
type AuthHandler struct {
	config *Config
}

// NewAuthHandler creates a new webhook authentication handler
func NewAuthHandler(config *Config) *AuthHandler {
	return &AuthHandler{
		config: config,
	}
}

// AuthType represents different authentication types
type AuthType string

const (
	AuthTypeNone      AuthType = "none"
	AuthTypeBasic     AuthType = "basic"
	AuthTypeBearer    AuthType = "bearer"
	AuthTypeAPIKey    AuthType = "api_key"
	AuthTypeSignature AuthType = "signature"
)

// AddAuthHeaders adds authentication headers to the HTTP request
func (a *AuthHandler) AddAuthHeaders(req *http.Request, payload []byte) error {
	if !a.config.IsAuthRequired() {
		return nil
	}

	switch AuthType(a.config.AuthType) {
	case AuthTypeBasic:
		return a.addBasicAuth(req)
	case AuthTypeBearer:
		return a.addBearerAuth(req)
	case AuthTypeAPIKey:
		return a.addAPIKeyAuth(req)
	case AuthTypeSignature:
		return a.addSignatureAuth(req, payload)
	default:
		return fmt.Errorf("unsupported auth type: %s", a.config.AuthType)
	}
}

// addBasicAuth adds basic authentication headers
func (a *AuthHandler) addBasicAuth(req *http.Request) error {
	if a.config.Username == "" || a.config.Password == "" {
		return fmt.Errorf("username and password required for basic auth")
	}

	credentials := base64.StdEncoding.EncodeToString(
		[]byte(a.config.Username + ":" + a.config.Password))
	req.Header.Set("Authorization", "Basic "+credentials)

	return nil
}

// addBearerAuth adds bearer token authentication
func (a *AuthHandler) addBearerAuth(req *http.Request) error {
	if a.config.BearerToken == "" {
		return fmt.Errorf("bearer token required for bearer auth")
	}

	req.Header.Set("Authorization", "Bearer "+a.config.BearerToken)
	return nil
}

// addAPIKeyAuth adds API key authentication
func (a *AuthHandler) addAPIKeyAuth(req *http.Request) error {
	if a.config.APIKey == "" {
		return fmt.Errorf("API key required for API key auth")
	}

	headerName := a.config.APIKeyHeader
	if headerName == "" {
		headerName = "X-API-Key" // default
	}

	req.Header.Set(headerName, a.config.APIKey)
	return nil
}

// addSignatureAuth adds signature-based authentication
func (a *AuthHandler) addSignatureAuth(req *http.Request, payload []byte) error {
	if a.config.Secret == "" {
		return fmt.Errorf("secret required for signature auth")
	}

	signature, err := a.generateSignature(payload)
	if err != nil {
		return fmt.Errorf("failed to generate signature: %w", err)
	}

	headerName := a.config.SignatureHeader
	if headerName == "" {
		headerName = "X-Signature"
	}

	req.Header.Set(headerName, signature)
	return nil
}

// generateSignature generates HMAC signature for the payload
func (a *AuthHandler) generateSignature(payload []byte) (string, error) {
	var hasher hash.Hash

	switch strings.ToLower(a.config.SignatureAlgo) {
	case "sha1":
		hasher = hmac.New(sha1.New, []byte(a.config.Secret))
	case "sha256":
		hasher = hmac.New(sha256.New, []byte(a.config.Secret))
	case "md5":
		hasher = hmac.New(md5.New, []byte(a.config.Secret))
	default:
		return "", fmt.Errorf("unsupported signature algorithm: %s", a.config.SignatureAlgo)
	}

	hasher.Write(payload)
	signature := hex.EncodeToString(hasher.Sum(nil))

	// Add prefix if configured
	if a.config.SignaturePrefix != "" {
		signature = a.config.SignaturePrefix + signature
	}

	return signature, nil
}

// ValidateAuth validates the authentication configuration
func (a *AuthHandler) ValidateAuth() error {
	if !a.config.IsAuthRequired() {
		return nil
	}

	switch AuthType(a.config.AuthType) {
	case AuthTypeBasic:
		if a.config.Username == "" {
			return fmt.Errorf("username is required for basic auth")
		}
		if a.config.Password == "" {
			return fmt.Errorf("password is required for basic auth")
		}

	case AuthTypeBearer:
		if a.config.BearerToken == "" {
			return fmt.Errorf("bearer_token is required for bearer auth")
		}

	case AuthTypeAPIKey:
		if a.config.APIKey == "" {
			return fmt.Errorf("api_key is required for API key auth")
		}
		// APIKeyHeader is optional, will default to "X-API-Key"

	case AuthTypeSignature:
		if a.config.Secret == "" {
			return fmt.Errorf("secret is required for signature auth")
		}
		if a.config.SignatureAlgo == "" {
			return fmt.Errorf("signature_algo is required for signature auth")
		}
		// Validate signature algorithm
		validAlgos := []string{"sha1", "sha256", "md5"}
		algoValid := false
		for _, algo := range validAlgos {
			if strings.EqualFold(a.config.SignatureAlgo, algo) {
				algoValid = true
				break
			}
		}
		if !algoValid {
			return fmt.Errorf("unsupported signature algorithm: %s (supported: %s)",
				a.config.SignatureAlgo, strings.Join(validAlgos, ", "))
		}

	default:
		return fmt.Errorf("unsupported auth type: %s", a.config.AuthType)
	}

	return nil
}

// TestAuth tests the authentication by making a test request
func (a *AuthHandler) TestAuth(testURL string) error {
	if testURL == "" {
		testURL = a.config.WebhookURL
	}

	if testURL == "" {
		return fmt.Errorf("no URL provided for auth test")
	}

	// Create a simple test request
	req, err := http.NewRequest("HEAD", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	// Add auth headers with empty payload for HEAD request
	if err := a.AddAuthHeaders(req, nil); err != nil {
		return fmt.Errorf("failed to add auth headers: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Send test request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("auth test request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check if the response indicates successful authentication
	// Most APIs return 2xx for authenticated requests, even for HEAD
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil // Authentication successful
	}

	// 405 Method Not Allowed might indicate auth is working but HEAD isn't supported
	if resp.StatusCode == 405 {
		return nil // Consider this success for auth test
	}

	// 401 or 403 typically indicate auth failure
	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed: unauthorized")
	}
	if resp.StatusCode == 403 {
		return fmt.Errorf("authentication failed: forbidden")
	}

	// Other status codes might indicate server issues rather than auth problems
	return fmt.Errorf("auth test returned status %d", resp.StatusCode)
}

// GetAuthInfo returns information about the current authentication setup
func (a *AuthHandler) GetAuthInfo() map[string]interface{} {
	info := map[string]interface{}{
		"auth_type":     a.config.AuthType,
		"auth_required": a.config.IsAuthRequired(),
	}

	switch AuthType(a.config.AuthType) {
	case AuthTypeBasic:
		info["username"] = a.config.Username
		info["has_password"] = a.config.Password != ""

	case AuthTypeBearer:
		info["has_bearer_token"] = a.config.BearerToken != ""

	case AuthTypeAPIKey:
		info["api_key_header"] = a.config.APIKeyHeader
		info["has_api_key"] = a.config.APIKey != ""

	case AuthTypeSignature:
		info["signature_algo"] = a.config.SignatureAlgo
		info["signature_header"] = a.config.SignatureHeader
		info["signature_prefix"] = a.config.SignaturePrefix
		info["has_secret"] = a.config.Secret != ""
	}

	return info
}

// VerifySignature verifies an incoming webhook signature
func (a *AuthHandler) VerifySignature(payload []byte, receivedSignature string) error {
	if !a.config.IsSignatureRequired() {
		return fmt.Errorf("signature verification not configured")
	}

	expectedSignature, err := a.generateSignature(payload)
	if err != nil {
		return fmt.Errorf("failed to generate expected signature: %w", err)
	}

	// Remove prefix from received signature if present
	if a.config.SignaturePrefix != "" && strings.HasPrefix(receivedSignature, a.config.SignaturePrefix) {
		receivedSignature = strings.TrimPrefix(receivedSignature, a.config.SignaturePrefix)
		expectedSignature = strings.TrimPrefix(expectedSignature, a.config.SignaturePrefix)
	}

	// Constant time comparison to prevent timing attacks
	if !hmac.Equal([]byte(receivedSignature), []byte(expectedSignature)) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// GetSupportedAuthTypes returns the list of supported authentication types
func GetSupportedAuthTypes() []string {
	return []string{
		string(AuthTypeNone),
		string(AuthTypeBasic),
		string(AuthTypeBearer),
		string(AuthTypeAPIKey),
		string(AuthTypeSignature),
	}
}

// GetSupportedSignatureAlgorithms returns supported signature algorithms
func GetSupportedSignatureAlgorithms() []string {
	return []string{"sha1", "sha256", "md5"}
}

// IsAuthTypeSupported checks if an auth type is supported
func IsAuthTypeSupported(authType string) bool {
	for _, supported := range GetSupportedAuthTypes() {
		if strings.EqualFold(authType, supported) {
			return true
		}
	}
	return false
}

// IsSignatureAlgorithmSupported checks if a signature algorithm is supported
func IsSignatureAlgorithmSupported(algo string) bool {
	for _, supported := range GetSupportedSignatureAlgorithms() {
		if strings.EqualFold(algo, supported) {
			return true
		}
	}
	return false
}

// AuthMetrics represents authentication metrics
type AuthMetrics struct {
	TotalRequests         int   `json:"total_requests"`
	AuthenticatedRequests int   `json:"authenticated_requests"`
	FailedAuth            int   `json:"failed_auth"`
	LastAuthTime          int64 `json:"last_auth_time,omitempty"`
}

// GetMetrics returns authentication metrics
func (a *AuthHandler) GetMetrics() *AuthMetrics {
	// This would integrate with the metrics system
	return &AuthMetrics{
		TotalRequests:         0,
		AuthenticatedRequests: 0,
		FailedAuth:            0,
	}
}
