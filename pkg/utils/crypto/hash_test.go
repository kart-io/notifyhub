package crypto

import (
	"strings"
	"testing"
)

func TestHasher(t *testing.T) {
	hasher := NewHasher(SHA256)

	// Test basic hashing
	data := []byte("hello world")
	hash, err := hasher.Hash(data)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if len(hash) != 32 { // SHA256 produces 32 bytes
		t.Errorf("Expected 32 bytes, got %d", len(hash))
	}

	// Test string hashing
	hashStr, err := hasher.HashString("hello world")
	if err != nil {
		t.Fatalf("HashString failed: %v", err)
	}

	if len(hashStr) != 64 { // 32 bytes * 2 hex chars per byte
		t.Errorf("Expected 64 hex characters, got %d", len(hashStr))
	}

	// Test consistency
	hash2, err := hasher.Hash(data)
	if err != nil {
		t.Fatalf("Second hash failed: %v", err)
	}

	if string(hash) != string(hash2) {
		t.Error("Hash results should be consistent")
	}
}

func TestHashAlgorithms(t *testing.T) {
	data := []byte("test data")

	algorithms := []HashAlgorithm{MD5, SHA1, SHA256, SHA384, SHA512}
	expectedLengths := []int{16, 20, 32, 48, 64} // bytes

	for i, algo := range algorithms {
		hasher := NewHasher(algo)
		hash, err := hasher.Hash(data)
		if err != nil {
			t.Errorf("Hash failed for %s: %v", algo, err)
			continue
		}

		if len(hash) != expectedLengths[i] {
			t.Errorf("Expected %d bytes for %s, got %d", expectedLengths[i], algo, len(hash))
		}
	}
}

func TestMultiHasher(t *testing.T) {
	multiHasher := NewMultiHasher(SHA256, SHA512)
	data := []byte("test data")

	results, err := multiHasher.Hash(data)
	if err != nil {
		t.Fatalf("MultiHash failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that each result has correct algorithm
	foundSHA256 := false
	foundSHA512 := false
	for _, result := range results {
		if result.Algorithm == SHA256 {
			foundSHA256 = true
			if len(result.Bytes) != 32 {
				t.Error("SHA256 should produce 32 bytes")
			}
		}
		if result.Algorithm == SHA512 {
			foundSHA512 = true
			if len(result.Bytes) != 64 {
				t.Error("SHA512 should produce 64 bytes")
			}
		}
	}

	if !foundSHA256 || !foundSHA512 {
		t.Error("Should have results for both SHA256 and SHA512")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	data := []byte("test data")
	dataStr := "test data"

	// Test MD5
	md5Hash := HashMD5(data)
	md5HashStr := HashMD5String(dataStr)
	if md5Hash != md5HashStr {
		t.Error("MD5 hash functions should produce same result")
	}
	if len(md5Hash) != 32 { // 16 bytes * 2 hex chars
		t.Errorf("Expected 32 hex chars for MD5, got %d", len(md5Hash))
	}

	// Test SHA1
	sha1Hash := HashSHA1(data)
	sha1HashStr := HashSHA1String(dataStr)
	if sha1Hash != sha1HashStr {
		t.Error("SHA1 hash functions should produce same result")
	}
	if len(sha1Hash) != 40 { // 20 bytes * 2 hex chars
		t.Errorf("Expected 40 hex chars for SHA1, got %d", len(sha1Hash))
	}

	// Test SHA256
	sha256Hash := HashSHA256(data)
	sha256HashStr := HashSHA256String(dataStr)
	if sha256Hash != sha256HashStr {
		t.Error("SHA256 hash functions should produce same result")
	}
	if len(sha256Hash) != 64 { // 32 bytes * 2 hex chars
		t.Errorf("Expected 64 hex chars for SHA256, got %d", len(sha256Hash))
	}

	// Test SHA512
	sha512Hash := HashSHA512(data)
	sha512HashStr := HashSHA512String(dataStr)
	if sha512Hash != sha512HashStr {
		t.Error("SHA512 hash functions should produce same result")
	}
	if len(sha512Hash) != 128 { // 64 bytes * 2 hex chars
		t.Errorf("Expected 128 hex chars for SHA512, got %d", len(sha512Hash))
	}
}

func TestFileIntegrityChecker(t *testing.T) {
	checker := NewFileIntegrityChecker(SHA256)
	data := []byte("file content")

	// Compute checksum
	checksum, err := checker.ComputeChecksum(data)
	if err != nil {
		t.Fatalf("ComputeChecksum failed: %v", err)
	}

	// Verify correct checksum
	valid, err := checker.VerifyChecksum(data, checksum)
	if err != nil {
		t.Fatalf("VerifyChecksum failed: %v", err)
	}
	if !valid {
		t.Error("Checksum verification should pass for correct data")
	}

	// Verify incorrect checksum
	valid, err = checker.VerifyChecksum([]byte("different content"), checksum)
	if err != nil {
		t.Fatalf("VerifyChecksum failed: %v", err)
	}
	if valid {
		t.Error("Checksum verification should fail for different data")
	}
}

func TestSimplePasswordHasher(t *testing.T) {
	hasher := NewSimplePasswordHasher()
	password := "mypassword"
	salt := hasher.GenerateSalt()

	// Test password hashing
	hash := hasher.HashPassword(password, salt)
	if len(hash) == 0 {
		t.Error("Password hash should not be empty")
	}

	// Test password verification
	if !hasher.VerifyPassword(password, salt, hash) {
		t.Error("Password verification should pass for correct password")
	}

	if hasher.VerifyPassword("wrongpassword", salt, hash) {
		t.Error("Password verification should fail for wrong password")
	}

	// Test salt generation
	salt2 := hasher.GenerateSalt()
	if salt == salt2 {
		t.Error("Generated salts should be different")
	}

	if len(salt) != 16 {
		t.Errorf("Expected salt length 16, got %d", len(salt))
	}
}

func TestContentHasher(t *testing.T) {
	hasher := NewContentHasher(SHA256)
	content := []byte("content data")

	// Test basic content hashing
	hash, err := hasher.HashContent(content)
	if err != nil {
		t.Fatalf("HashContent failed: %v", err)
	}
	if len(hash) == 0 {
		t.Error("Content hash should not be empty")
	}

	// Test content hashing with metadata
	metadata := map[string]string{
		"type":    "document",
		"version": "1.0",
	}
	hashWithMeta, err := hasher.HashContentWithMetadata(content, metadata)
	if err != nil {
		t.Fatalf("HashContentWithMetadata failed: %v", err)
	}

	// Hash with metadata should be different
	if hash == hashWithMeta {
		t.Error("Hash with metadata should be different from plain hash")
	}
}

func TestMessageIntegrityChecker(t *testing.T) {
	checker := NewMessageIntegrityChecker(SHA256)
	messageID := "msg_123"
	content := "message content"
	timestamp := int64(1640995200) // Fixed timestamp

	// Compute message hash
	hash, err := checker.ComputeMessageHash(messageID, content, timestamp)
	if err != nil {
		t.Fatalf("ComputeMessageHash failed: %v", err)
	}

	// Verify correct hash
	valid, err := checker.VerifyMessageHash(messageID, content, timestamp, hash)
	if err != nil {
		t.Fatalf("VerifyMessageHash failed: %v", err)
	}
	if !valid {
		t.Error("Message hash verification should pass for correct data")
	}

	// Verify with wrong content
	valid, err = checker.VerifyMessageHash(messageID, "different content", timestamp, hash)
	if err != nil {
		t.Fatalf("VerifyMessageHash failed: %v", err)
	}
	if valid {
		t.Error("Message hash verification should fail for different content")
	}
}

func TestUnsupportedAlgorithm(t *testing.T) {
	hasher := NewHasher("unsupported")
	_, err := hasher.Hash([]byte("test"))
	if err == nil {
		t.Error("Should return error for unsupported algorithm")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("Error should mention unsupported algorithm: %v", err)
	}
}

func BenchmarkHashSHA256(b *testing.B) {
	data := []byte("benchmark data for hashing performance test")
	hasher := NewHasher(SHA256)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hasher.Hash(data)
	}
}

func BenchmarkHashMD5(b *testing.B) {
	data := []byte("benchmark data for hashing performance test")
	hasher := NewHasher(MD5)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hasher.Hash(data)
	}
}

func BenchmarkHashSHA512(b *testing.B) {
	data := []byte("benchmark data for hashing performance test")
	hasher := NewHasher(SHA512)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hasher.Hash(data)
	}
}

func BenchmarkConvenienceSHA256(b *testing.B) {
	data := []byte("benchmark data for hashing performance test")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		HashSHA256(data)
	}
}