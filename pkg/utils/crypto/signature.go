// Package crypto provides digital signature utilities for NotifyHub
package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"
)

// SignatureAlgorithm represents different signature algorithms
type SignatureAlgorithm string

const (
	// HMAC-MD5 (not recommended for security-critical applications)
	HMACMD5 SignatureAlgorithm = "hmac-md5"
	// HMAC-SHA1 (not recommended for new applications)
	HMACSHA1 SignatureAlgorithm = "hmac-sha1"
	// HMAC-SHA256 (recommended)
	HMACSHA256 SignatureAlgorithm = "hmac-sha256"
	// HMAC-SHA384
	HMACSHA384 SignatureAlgorithm = "hmac-sha384"
	// HMAC-SHA512
	HMACSHA512 SignatureAlgorithm = "hmac-sha512"
)

// Signer provides digital signature functionality
type Signer struct {
	algorithm SignatureAlgorithm
	key       []byte
}

// NewSigner creates a new signer with the specified algorithm and key
func NewSigner(algorithm SignatureAlgorithm, key []byte) *Signer {
	return &Signer{
		algorithm: algorithm,
		key:       key,
	}
}

// NewSignerFromString creates a new signer with a string key
func NewSignerFromString(algorithm SignatureAlgorithm, key string) *Signer {
	return NewSigner(algorithm, []byte(key))
}

// Sign creates a signature for the given data
func (s *Signer) Sign(data []byte) ([]byte, error) {
	hasher, err := s.newHMACFunc()
	if err != nil {
		return nil, err
	}

	hasher.Write(data)
	return hasher.Sum(nil), nil
}

// SignString creates a signature for the given string and returns base64 encoding
func (s *Signer) SignString(data string) (string, error) {
	signature, err := s.Sign([]byte(data))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// SignHex creates a signature for the given data and returns hex encoding
func (s *Signer) SignHex(data []byte) (string, error) {
	signature, err := s.Sign(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(signature), nil
}

// SignStringHex creates a signature for the given string and returns hex encoding
func (s *Signer) SignStringHex(data string) (string, error) {
	return s.SignHex([]byte(data))
}

// Verify verifies a signature against the given data
func (s *Signer) Verify(data, signature []byte) (bool, error) {
	expectedSignature, err := s.Sign(data)
	if err != nil {
		return false, err
	}
	return hmac.Equal(signature, expectedSignature), nil
}

// VerifyString verifies a base64-encoded signature against the given string
func (s *Signer) VerifyString(data, base64Signature string) (bool, error) {
	signature, err := base64.StdEncoding.DecodeString(base64Signature)
	if err != nil {
		return false, fmt.Errorf("invalid base64 signature: %w", err)
	}
	return s.Verify([]byte(data), signature)
}

// VerifyHex verifies a hex-encoded signature against the given data
func (s *Signer) VerifyHex(data []byte, hexSignature string) (bool, error) {
	signature, err := hex.DecodeString(hexSignature)
	if err != nil {
		return false, fmt.Errorf("invalid hex signature: %w", err)
	}
	return s.Verify(data, signature)
}

// VerifyStringHex verifies a hex-encoded signature against the given string
func (s *Signer) VerifyStringHex(data, hexSignature string) (bool, error) {
	return s.VerifyHex([]byte(data), hexSignature)
}

// newHMACFunc creates a new HMAC function based on the algorithm
func (s *Signer) newHMACFunc() (hash.Hash, error) {
	switch s.algorithm {
	case HMACMD5:
		return hmac.New(md5.New, s.key), nil
	case HMACSHA1:
		return hmac.New(sha1.New, s.key), nil
	case HMACSHA256:
		return hmac.New(sha256.New, s.key), nil
	case HMACSHA384:
		return hmac.New(sha512.New384, s.key), nil
	case HMACSHA512:
		return hmac.New(sha512.New, s.key), nil
	default:
		return nil, fmt.Errorf("unsupported signature algorithm: %s", s.algorithm)
	}
}

// Webhook signature verification

// WebhookSigner provides webhook signature functionality
type WebhookSigner struct {
	signer *Signer
}

// NewWebhookSigner creates a new webhook signer
func NewWebhookSigner(algorithm SignatureAlgorithm, secret string) *WebhookSigner {
	return &WebhookSigner{
		signer: NewSignerFromString(algorithm, secret),
	}
}

// SignWebhookPayload signs a webhook payload with timestamp
func (ws *WebhookSigner) SignWebhookPayload(payload []byte, timestamp int64) (string, error) {
	// Create the signature string: timestamp + payload
	signatureString := strconv.FormatInt(timestamp, 10) + string(payload)
	return ws.signer.SignString(signatureString)
}

// VerifyWebhookSignature verifies a webhook signature
func (ws *WebhookSigner) VerifyWebhookSignature(payload []byte, timestamp int64, signature string) (bool, error) {
	expectedSignature, err := ws.SignWebhookPayload(payload, timestamp)
	if err != nil {
		return false, err
	}
	return SecureCompareString(signature, expectedSignature), nil
}

// Feishu-specific signature implementation

// FeishuSigner implements Feishu webhook signature verification
type FeishuSigner struct {
	secret string
}

// NewFeishuSigner creates a new Feishu signer
func NewFeishuSigner(secret string) *FeishuSigner {
	return &FeishuSigner{
		secret: secret,
	}
}

// GenerateSignature generates a Feishu webhook signature
func (fs *FeishuSigner) GenerateSignature(timestamp string) string {
	if fs.secret == "" {
		return ""
	}

	// Feishu signature algorithm: HMAC-SHA256 with special string format
	stringToSign := timestamp + "\n" + fs.secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies a Feishu webhook signature
func (fs *FeishuSigner) VerifySignature(timestamp, signature string) bool {
	if fs.secret == "" {
		return true // No secret means no verification
	}

	expectedSignature := fs.GenerateSignature(timestamp)
	return SecureCompareString(signature, expectedSignature)
}

// Request signature for API authentication

// RequestSigner provides API request signature functionality
type RequestSigner struct {
	signer *Signer
}

// NewRequestSigner creates a new request signer
func NewRequestSigner(algorithm SignatureAlgorithm, secret string) *RequestSigner {
	return &RequestSigner{
		signer: NewSignerFromString(algorithm, secret),
	}
}

// SignRequest signs an API request
func (rs *RequestSigner) SignRequest(method, path, body string, timestamp int64) (string, error) {
	// Create canonical request string
	canonicalRequest := fmt.Sprintf("%s\n%s\n%d\n%s", method, path, timestamp, body)
	return rs.signer.SignString(canonicalRequest)
}

// VerifyRequest verifies an API request signature
func (rs *RequestSigner) VerifyRequest(method, path, body string, timestamp int64, signature string) (bool, error) {
	expectedSignature, err := rs.SignRequest(method, path, body, timestamp)
	if err != nil {
		return false, err
	}
	return SecureCompareString(signature, expectedSignature), nil
}

// Message signature for integrity

// MessageSigner provides message signature functionality for integrity verification
type MessageSigner struct {
	signer *Signer
}

// NewMessageSigner creates a new message signer
func NewMessageSigner(algorithm SignatureAlgorithm, secret string) *MessageSigner {
	return &MessageSigner{
		signer: NewSignerFromString(algorithm, secret),
	}
}

// SignMessage signs a message with metadata
func (ms *MessageSigner) SignMessage(messageID, content string, timestamp int64, metadata map[string]string) (string, error) {
	// Create signature payload
	var parts []string
	parts = append(parts, messageID, content, strconv.FormatInt(timestamp, 10))

	// Add sorted metadata
	if metadata != nil {
		for key, value := range metadata {
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		}
	}

	payload := strings.Join(parts, "|")
	return ms.signer.SignString(payload)
}

// VerifyMessage verifies a message signature
func (ms *MessageSigner) VerifyMessage(messageID, content string, timestamp int64, metadata map[string]string, signature string) (bool, error) {
	expectedSignature, err := ms.SignMessage(messageID, content, timestamp, metadata)
	if err != nil {
		return false, err
	}
	return SecureCompareString(signature, expectedSignature), nil
}

// Token signature for authentication

// TokenSigner provides token signature functionality
type TokenSigner struct {
	signer *Signer
}

// NewTokenSigner creates a new token signer
func NewTokenSigner(algorithm SignatureAlgorithm, secret string) *TokenSigner {
	return &TokenSigner{
		signer: NewSignerFromString(algorithm, secret),
	}
}

// SignedToken represents a signed token
type SignedToken struct {
	Token     string    `json:"token"`
	Signature string    `json:"signature"`
	Timestamp int64     `json:"timestamp"`
	ExpiresAt int64     `json:"expires_at"`
	Algorithm string    `json:"algorithm"`
}

// CreateSignedToken creates a new signed token
func (ts *TokenSigner) CreateSignedToken(tokenData string, expiresIn time.Duration) (*SignedToken, error) {
	now := time.Now().Unix()
	expiresAt := now + int64(expiresIn.Seconds())

	// Create signature payload: token + timestamp + expires_at
	payload := fmt.Sprintf("%s|%d|%d", tokenData, now, expiresAt)
	signature, err := ts.signer.SignString(payload)
	if err != nil {
		return nil, err
	}

	return &SignedToken{
		Token:     tokenData,
		Signature: signature,
		Timestamp: now,
		ExpiresAt: expiresAt,
		Algorithm: string(ts.signer.algorithm),
	}, nil
}

// VerifySignedToken verifies a signed token
func (ts *TokenSigner) VerifySignedToken(token *SignedToken) (bool, error) {
	// Check expiration
	if time.Now().Unix() > token.ExpiresAt {
		return false, fmt.Errorf("token expired")
	}

	// Verify signature
	payload := fmt.Sprintf("%s|%d|%d", token.Token, token.Timestamp, token.ExpiresAt)
	return ts.signer.VerifyString(payload, token.Signature)
}

// Convenience functions for common signature operations

// SignHMACSHA256 signs data with HMAC-SHA256 and returns base64 encoding
func SignHMACSHA256(data []byte, key string) string {
	signer := NewSignerFromString(HMACSHA256, key)
	signature, _ := signer.SignString(string(data)) // Ignore error for convenience
	return signature
}

// SignHMACSHA256String signs string with HMAC-SHA256 and returns base64 encoding
func SignHMACSHA256String(data, key string) string {
	return SignHMACSHA256([]byte(data), key)
}

// VerifyHMACSHA256 verifies HMAC-SHA256 signature
func VerifyHMACSHA256(data []byte, signature, key string) bool {
	signer := NewSignerFromString(HMACSHA256, key)
	valid, _ := signer.VerifyString(string(data), signature) // Ignore error for convenience
	return valid
}

// VerifyHMACSHA256String verifies HMAC-SHA256 signature for string
func VerifyHMACSHA256String(data, signature, key string) bool {
	return VerifyHMACSHA256([]byte(data), signature, key)
}

// Error types
var (
	ErrInvalidSignature     = fmt.Errorf("invalid signature")
	ErrSignatureExpired     = fmt.Errorf("signature expired")
	ErrUnsupportedSignature = fmt.Errorf("unsupported signature algorithm")
)

// Timestamp utilities for signature verification

// TimestampValidator provides timestamp validation for signatures
type TimestampValidator struct {
	tolerance time.Duration
}

// NewTimestampValidator creates a new timestamp validator
func NewTimestampValidator(tolerance time.Duration) *TimestampValidator {
	return &TimestampValidator{
		tolerance: tolerance,
	}
}

// ValidateTimestamp validates that a timestamp is within the tolerance
func (tv *TimestampValidator) ValidateTimestamp(timestamp int64) error {
	now := time.Now().Unix()
	diff := now - timestamp

	if diff < 0 {
		diff = -diff
	}

	if time.Duration(diff)*time.Second > tv.tolerance {
		return fmt.Errorf("timestamp outside tolerance: %d seconds", diff)
	}

	return nil
}

// ValidateTimestampString validates a string timestamp
func (tv *TimestampValidator) ValidateTimestampString(timestampStr string) error {
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}
	return tv.ValidateTimestamp(timestamp)
}