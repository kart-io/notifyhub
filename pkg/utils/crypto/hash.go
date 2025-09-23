// Package crypto provides cryptographic utilities for NotifyHub
package crypto

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"time"
)

// HashAlgorithm represents different hashing algorithms
type HashAlgorithm string

const (
	// MD5 algorithm (deprecated for security-critical uses)
	MD5 HashAlgorithm = "md5"
	// SHA1 algorithm (deprecated for security-critical uses)
	SHA1 HashAlgorithm = "sha1"
	// SHA256 algorithm (recommended)
	SHA256 HashAlgorithm = "sha256"
	// SHA384 algorithm
	SHA384 HashAlgorithm = "sha384"
	// SHA512 algorithm
	SHA512 HashAlgorithm = "sha512"
)

// Hasher provides cryptographic hashing functionality
type Hasher struct {
	algorithm HashAlgorithm
}

// NewHasher creates a new hasher with the specified algorithm
func NewHasher(algorithm HashAlgorithm) *Hasher {
	return &Hasher{
		algorithm: algorithm,
	}
}

// Hash computes the hash of the input data
func (h *Hasher) Hash(data []byte) ([]byte, error) {
	hasher, err := h.newHashFunc()
	if err != nil {
		return nil, err
	}

	hasher.Write(data)
	return hasher.Sum(nil), nil
}

// HashString computes the hash of the input string and returns hex encoding
func (h *Hasher) HashString(data string) (string, error) {
	hash, err := h.Hash([]byte(data))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// HashReader computes the hash of data from a reader
func (h *Hasher) HashReader(reader io.Reader) ([]byte, error) {
	hasher, err := h.newHashFunc()
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(hasher, reader); err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	return hasher.Sum(nil), nil
}

// HashReaderString computes the hash of data from a reader and returns hex encoding
func (h *Hasher) HashReaderString(reader io.Reader) (string, error) {
	hash, err := h.HashReader(reader)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// newHashFunc creates a new hash function based on the algorithm
func (h *Hasher) newHashFunc() (hash.Hash, error) {
	switch h.algorithm {
	case MD5:
		return md5.New(), nil
	case SHA1:
		return sha1.New(), nil
	case SHA256:
		return sha256.New(), nil
	case SHA384:
		return sha512.New384(), nil
	case SHA512:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", h.algorithm)
	}
}

// MultiHasher computes multiple hashes simultaneously
type MultiHasher struct {
	algorithms []HashAlgorithm
}

// NewMultiHasher creates a new multi-hasher with the specified algorithms
func NewMultiHasher(algorithms ...HashAlgorithm) *MultiHasher {
	return &MultiHasher{
		algorithms: algorithms,
	}
}

// HashResult represents the result of multi-hashing
type HashResult struct {
	Algorithm HashAlgorithm `json:"algorithm"`
	Hash      string        `json:"hash"`
	Bytes     []byte        `json:"-"`
}

// Hash computes multiple hashes of the input data
func (mh *MultiHasher) Hash(data []byte) ([]HashResult, error) {
	results := make([]HashResult, len(mh.algorithms))

	for i, algo := range mh.algorithms {
		hasher := NewHasher(algo)
		hashBytes, err := hasher.Hash(data)
		if err != nil {
			return nil, fmt.Errorf("failed to hash with %s: %w", algo, err)
		}

		results[i] = HashResult{
			Algorithm: algo,
			Hash:      hex.EncodeToString(hashBytes),
			Bytes:     hashBytes,
		}
	}

	return results, nil
}

// HashString computes multiple hashes of the input string
func (mh *MultiHasher) HashString(data string) ([]HashResult, error) {
	return mh.Hash([]byte(data))
}

// Convenience functions for common hash operations

// HashMD5 computes MD5 hash and returns hex string
func HashMD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// HashMD5String computes MD5 hash of string and returns hex string
func HashMD5String(data string) string {
	return HashMD5([]byte(data))
}

// HashSHA1 computes SHA1 hash and returns hex string
func HashSHA1(data []byte) string {
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:])
}

// HashSHA1String computes SHA1 hash of string and returns hex string
func HashSHA1String(data string) string {
	return HashSHA1([]byte(data))
}

// HashSHA256 computes SHA256 hash and returns hex string
func HashSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// HashSHA256String computes SHA256 hash of string and returns hex string
func HashSHA256String(data string) string {
	return HashSHA256([]byte(data))
}

// HashSHA512 computes SHA512 hash and returns hex string
func HashSHA512(data []byte) string {
	hash := sha512.Sum512(data)
	return hex.EncodeToString(hash[:])
}

// HashSHA512String computes SHA512 hash of string and returns hex string
func HashSHA512String(data string) string {
	return HashSHA512([]byte(data))
}

// File integrity checking

// FileIntegrityChecker provides file integrity verification
type FileIntegrityChecker struct {
	algorithm HashAlgorithm
}

// NewFileIntegrityChecker creates a new file integrity checker
func NewFileIntegrityChecker(algorithm HashAlgorithm) *FileIntegrityChecker {
	return &FileIntegrityChecker{
		algorithm: algorithm,
	}
}

// ComputeChecksum computes checksum for the given data
func (fic *FileIntegrityChecker) ComputeChecksum(data []byte) (string, error) {
	hasher := NewHasher(fic.algorithm)
	return hasher.HashString(string(data))
}

// VerifyChecksum verifies that data matches the expected checksum
func (fic *FileIntegrityChecker) VerifyChecksum(data []byte, expectedChecksum string) (bool, error) {
	actualChecksum, err := fic.ComputeChecksum(data)
	if err != nil {
		return false, err
	}
	return actualChecksum == expectedChecksum, nil
}

// ComputeChecksumReader computes checksum for data from a reader
func (fic *FileIntegrityChecker) ComputeChecksumReader(reader io.Reader) (string, error) {
	hasher := NewHasher(fic.algorithm)
	return hasher.HashReaderString(reader)
}

// Password hashing utilities (for configuration security)

// SimplePasswordHasher provides basic password hashing (not for production auth)
type SimplePasswordHasher struct{}

// NewSimplePasswordHasher creates a new simple password hasher
func NewSimplePasswordHasher() *SimplePasswordHasher {
	return &SimplePasswordHasher{}
}

// HashPassword hashes a password with salt (simplified implementation)
func (sph *SimplePasswordHasher) HashPassword(password, salt string) string {
	combined := password + salt
	return HashSHA256String(combined)
}

// VerifyPassword verifies a password against a hash
func (sph *SimplePasswordHasher) VerifyPassword(password, salt, hash string) bool {
	computedHash := sph.HashPassword(password, salt)
	return computedHash == hash
}

// GenerateSalt generates a simple salt for password hashing
func (sph *SimplePasswordHasher) GenerateSalt() string {
	// Simple salt generation - in production, use crypto/rand
	data := fmt.Sprintf("%d", time.Now().UnixNano())
	return HashSHA256String(data)[:16]
}

// Content hash utilities for deduplication

// ContentHasher provides content-based hashing for deduplication
type ContentHasher struct {
	algorithm HashAlgorithm
}

// NewContentHasher creates a new content hasher
func NewContentHasher(algorithm HashAlgorithm) *ContentHasher {
	return &ContentHasher{
		algorithm: algorithm,
	}
}

// HashContent computes a content hash for the given data
func (ch *ContentHasher) HashContent(data []byte) (string, error) {
	hasher := NewHasher(ch.algorithm)
	return hasher.HashString(string(data))
}

// HashContentWithMetadata computes hash including metadata
func (ch *ContentHasher) HashContentWithMetadata(data []byte, metadata map[string]string) (string, error) {
	hasher := NewHasher(ch.algorithm)

	// Hash the data first
	dataHash, err := hasher.Hash(data)
	if err != nil {
		return "", err
	}

	// Create a combined hash with metadata
	combined := hex.EncodeToString(dataHash)
	for key, value := range metadata {
		combined += fmt.Sprintf("%s=%s;", key, value)
	}

	return hasher.HashString(combined)
}

// Checksum utilities for message integrity

// MessageIntegrityChecker provides message integrity verification
type MessageIntegrityChecker struct {
	algorithm HashAlgorithm
}

// NewMessageIntegrityChecker creates a new message integrity checker
func NewMessageIntegrityChecker(algorithm HashAlgorithm) *MessageIntegrityChecker {
	return &MessageIntegrityChecker{
		algorithm: algorithm,
	}
}

// ComputeMessageHash computes hash for a message
func (mic *MessageIntegrityChecker) ComputeMessageHash(messageID, content string, timestamp int64) (string, error) {
	combined := fmt.Sprintf("%s|%s|%d", messageID, content, timestamp)
	hasher := NewHasher(mic.algorithm)
	return hasher.HashString(combined)
}

// VerifyMessageHash verifies message integrity
func (mic *MessageIntegrityChecker) VerifyMessageHash(messageID, content string, timestamp int64, expectedHash string) (bool, error) {
	actualHash, err := mic.ComputeMessageHash(messageID, content, timestamp)
	if err != nil {
		return false, err
	}
	return actualHash == expectedHash, nil
}

// Error types
var (
	ErrHashMismatch = fmt.Errorf("hash mismatch")
	ErrInvalidInput = fmt.Errorf("invalid input")
)