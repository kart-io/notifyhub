// Package crypto provides encryption utilities for NotifyHub
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// EncryptionAlgorithm represents different encryption algorithms
type EncryptionAlgorithm string

const (
	// AES256GCM - AES-256 with Galois/Counter Mode
	AES256GCM EncryptionAlgorithm = "aes-256-gcm"
	// AES192GCM - AES-192 with Galois/Counter Mode
	AES192GCM EncryptionAlgorithm = "aes-192-gcm"
	// AES128GCM - AES-128 with Galois/Counter Mode
	AES128GCM EncryptionAlgorithm = "aes-128-gcm"
)

// Encryptor provides encryption and decryption functionality
type Encryptor struct {
	algorithm EncryptionAlgorithm
	key       []byte
}

// NewEncryptor creates a new encryptor with the specified algorithm and key
func NewEncryptor(algorithm EncryptionAlgorithm, key []byte) (*Encryptor, error) {
	if err := validateKeySize(algorithm, len(key)); err != nil {
		return nil, err
	}

	return &Encryptor{
		algorithm: algorithm,
		key:       key,
	}, nil
}

// NewEncryptorFromHex creates a new encryptor with a hex-encoded key
func NewEncryptorFromHex(algorithm EncryptionAlgorithm, hexKey string) (*Encryptor, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}
	return NewEncryptor(algorithm, key)
}

// NewEncryptorFromBase64 creates a new encryptor with a base64-encoded key
func NewEncryptorFromBase64(algorithm EncryptionAlgorithm, base64Key string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 key: %w", err)
	}
	return NewEncryptor(algorithm, key)
}

// Encrypt encrypts the given plaintext
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	switch e.algorithm {
	case AES128GCM, AES192GCM, AES256GCM:
		return e.encryptAESGCM(plaintext)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", e.algorithm)
	}
}

// EncryptString encrypts a string and returns base64-encoded ciphertext
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts the given ciphertext
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	switch e.algorithm {
	case AES128GCM, AES192GCM, AES256GCM:
		return e.decryptAESGCM(ciphertext)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", e.algorithm)
	}
}

// DecryptString decrypts a base64-encoded ciphertext and returns the plaintext string
func (e *Encryptor) DecryptString(base64Ciphertext string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		return "", fmt.Errorf("invalid base64 ciphertext: %w", err)
	}

	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// encryptAESGCM encrypts data using AES-GCM
func (e *Encryptor) encryptAESGCM(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptAESGCM decrypts data using AES-GCM
func (e *Encryptor) decryptAESGCM(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// Key generation utilities

// KeyGenerator provides key generation functionality
type KeyGenerator struct{}

// NewKeyGenerator creates a new key generator
func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{}
}

// GenerateKey generates a random key of the specified size
func (kg *KeyGenerator) GenerateKey(size int) ([]byte, error) {
	key := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyForAlgorithm generates a key appropriate for the specified algorithm
func (kg *KeyGenerator) GenerateKeyForAlgorithm(algorithm EncryptionAlgorithm) ([]byte, error) {
	var keySize int
	switch algorithm {
	case AES128GCM:
		keySize = 16 // 128 bits
	case AES192GCM:
		keySize = 24 // 192 bits
	case AES256GCM:
		keySize = 32 // 256 bits
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	return kg.GenerateKey(keySize)
}

// GenerateKeyHex generates a key and returns it as hex string
func (kg *KeyGenerator) GenerateKeyHex(size int) (string, error) {
	key, err := kg.GenerateKey(size)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// GenerateKeyBase64 generates a key and returns it as base64 string
func (kg *KeyGenerator) GenerateKeyBase64(size int) (string, error) {
	key, err := kg.GenerateKey(size)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// Field-level encryption for sensitive data

// FieldEncryptor provides field-level encryption for sensitive configuration
type FieldEncryptor struct {
	encryptor *Encryptor
}

// NewFieldEncryptor creates a new field encryptor
func NewFieldEncryptor(encryptor *Encryptor) *FieldEncryptor {
	return &FieldEncryptor{
		encryptor: encryptor,
	}
}

// EncryptField encrypts a field value and adds a prefix to indicate it's encrypted
func (fe *FieldEncryptor) EncryptField(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	encrypted, err := fe.encryptor.EncryptString(value)
	if err != nil {
		return "", err
	}

	return "enc:" + encrypted, nil
}

// DecryptField decrypts a field value if it has the encrypted prefix
func (fe *FieldEncryptor) DecryptField(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	if !fe.IsEncrypted(value) {
		return value, nil // Not encrypted, return as-is
	}

	// Remove the "enc:" prefix
	encrypted := value[4:]
	return fe.encryptor.DecryptString(encrypted)
}

// IsEncrypted checks if a field value is encrypted (has the "enc:" prefix)
func (fe *FieldEncryptor) IsEncrypted(value string) bool {
	return len(value) > 4 && value[:4] == "enc:"
}

// Message encryption for sensitive notification content

// MessageEncryptor provides message-level encryption
type MessageEncryptor struct {
	encryptor *Encryptor
}

// NewMessageEncryptor creates a new message encryptor
func NewMessageEncryptor(encryptor *Encryptor) *MessageEncryptor {
	return &MessageEncryptor{
		encryptor: encryptor,
	}
}

// EncryptedMessage represents an encrypted message
type EncryptedMessage struct {
	Algorithm  EncryptionAlgorithm `json:"algorithm"`
	Ciphertext string              `json:"ciphertext"`
	Metadata   map[string]string   `json:"metadata,omitempty"`
}

// EncryptMessage encrypts a message and returns an EncryptedMessage
func (me *MessageEncryptor) EncryptMessage(plaintext string, metadata map[string]string) (*EncryptedMessage, error) {
	ciphertext, err := me.encryptor.EncryptString(plaintext)
	if err != nil {
		return nil, err
	}

	return &EncryptedMessage{
		Algorithm:  me.encryptor.algorithm,
		Ciphertext: ciphertext,
		Metadata:   metadata,
	}, nil
}

// DecryptMessage decrypts an EncryptedMessage and returns the plaintext
func (me *MessageEncryptor) DecryptMessage(msg *EncryptedMessage) (string, error) {
	if msg.Algorithm != me.encryptor.algorithm {
		return "", fmt.Errorf("algorithm mismatch: expected %s, got %s",
			me.encryptor.algorithm, msg.Algorithm)
	}

	return me.encryptor.DecryptString(msg.Ciphertext)
}

// Utility functions

// validateKeySize validates that the key size is appropriate for the algorithm
func validateKeySize(algorithm EncryptionAlgorithm, keySize int) error {
	var expectedSize int
	switch algorithm {
	case AES128GCM:
		expectedSize = 16
	case AES192GCM:
		expectedSize = 24
	case AES256GCM:
		expectedSize = 32
	default:
		return fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	if keySize != expectedSize {
		return fmt.Errorf("invalid key size for %s: expected %d bytes, got %d",
			algorithm, expectedSize, keySize)
	}

	return nil
}

// SecureCompare performs a constant-time comparison of two byte slices
func SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// SecureCompareString performs a constant-time comparison of two strings
func SecureCompareString(a, b string) bool {
	return SecureCompare([]byte(a), []byte(b))
}

// ZeroBytes securely zeros out a byte slice
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Error types
var (
	ErrInvalidKeySize       = fmt.Errorf("invalid key size")
	ErrUnsupportedAlgorithm = fmt.Errorf("unsupported encryption algorithm")
	ErrDecryptionFailed     = fmt.Errorf("decryption failed")
	ErrInvalidCiphertext    = fmt.Errorf("invalid ciphertext")
)

// Configuration encryption helpers

// ConfigEncryptor provides configuration encryption utilities
type ConfigEncryptor struct {
	fieldEncryptor *FieldEncryptor
}

// NewConfigEncryptor creates a new configuration encryptor
func NewConfigEncryptor(key []byte) (*ConfigEncryptor, error) {
	encryptor, err := NewEncryptor(AES256GCM, key)
	if err != nil {
		return nil, err
	}

	return &ConfigEncryptor{
		fieldEncryptor: NewFieldEncryptor(encryptor),
	}, nil
}

// EncryptSensitiveFields encrypts sensitive fields in a configuration map
func (ce *ConfigEncryptor) EncryptSensitiveFields(config map[string]interface{}, sensitiveFields []string) error {
	for _, field := range sensitiveFields {
		if value, exists := config[field]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				encrypted, err := ce.fieldEncryptor.EncryptField(strValue)
				if err != nil {
					return fmt.Errorf("failed to encrypt field %s: %w", field, err)
				}
				config[field] = encrypted
			}
		}
	}
	return nil
}

// DecryptSensitiveFields decrypts sensitive fields in a configuration map
func (ce *ConfigEncryptor) DecryptSensitiveFields(config map[string]interface{}, sensitiveFields []string) error {
	for _, field := range sensitiveFields {
		if value, exists := config[field]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				decrypted, err := ce.fieldEncryptor.DecryptField(strValue)
				if err != nil {
					return fmt.Errorf("failed to decrypt field %s: %w", field, err)
				}
				config[field] = decrypted
			}
		}
	}
	return nil
}
