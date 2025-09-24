package validation

import (
	"testing"
)

func TestRequiredRule(t *testing.T) {
	rule := RequiredRule{}

	// Test nil value
	if err := rule.Validate(nil); err == nil {
		t.Error("Should fail for nil value")
	}

	// Test empty string
	if err := rule.Validate(""); err == nil {
		t.Error("Should fail for empty string")
	}

	// Test whitespace string
	if err := rule.Validate("   "); err == nil {
		t.Error("Should fail for whitespace string")
	}

	// Test valid string
	if err := rule.Validate("valid"); err != nil {
		t.Errorf("Should pass for valid string: %v", err)
	}

	// Test empty slice
	if err := rule.Validate([]interface{}{}); err == nil {
		t.Error("Should fail for empty slice")
	}

	// Test non-empty slice
	if err := rule.Validate([]interface{}{1, 2, 3}); err != nil {
		t.Errorf("Should pass for non-empty slice: %v", err)
	}
}

func TestEmailRule(t *testing.T) {
	rule := EmailRule{}

	// Test valid emails
	validEmails := []string{
		"test@example.com",
		"user.name@domain.co.uk",
		"user+tag@example.org",
	}

	for _, email := range validEmails {
		if err := rule.Validate(email); err != nil {
			t.Errorf("Should pass for valid email %s: %v", email, err)
		}
	}

	// Test invalid emails
	invalidEmails := []string{
		"invalid",
		"@example.com",
		"user@",
		"user space@example.com",
	}

	for _, email := range invalidEmails {
		if err := rule.Validate(email); err == nil {
			t.Errorf("Should fail for invalid email: %s", email)
		}
	}

	// Test empty string (should pass for non-required fields)
	if err := rule.Validate(""); err != nil {
		t.Errorf("Should pass for empty string: %v", err)
	}

	// Test non-string value
	if err := rule.Validate(123); err == nil {
		t.Error("Should fail for non-string value")
	}
}

func TestURLRule(t *testing.T) {
	rule := URLRule{}

	// Test valid URLs
	validURLs := []string{
		"https://example.com",
		"http://localhost:8080",
		"ftp://files.example.com/path",
	}

	for _, url := range validURLs {
		if err := rule.Validate(url); err != nil {
			t.Errorf("Should pass for valid URL %s: %v", url, err)
		}
	}

	// Test empty string (should pass for non-required fields)
	if err := rule.Validate(""); err != nil {
		t.Errorf("Should pass for empty string: %v", err)
	}

	// Test non-string value
	if err := rule.Validate(123); err == nil {
		t.Error("Should fail for non-string value")
	}
}

func TestPhoneRule(t *testing.T) {
	rule := PhoneRule{}

	// Test valid phone numbers
	validPhones := []string{
		"+1234567890",
		"+86138000000000",
		"+442071234567",
	}

	for _, phone := range validPhones {
		if err := rule.Validate(phone); err != nil {
			t.Errorf("Should pass for valid phone %s: %v", phone, err)
		}
	}

	// Test invalid phone numbers
	invalidPhones := []string{
		"1234567890",  // missing +
		"+0123456789", // starts with 0
		"+1",          // too short (need at least 2 digits)
		"invalid",     // not numeric
	}

	for _, phone := range invalidPhones {
		if err := rule.Validate(phone); err == nil {
			t.Errorf("Should fail for invalid phone: %s", phone)
		}
	}
}

func TestMinLengthRule(t *testing.T) {
	rule := MinLengthRule{Min: 5}

	// Test string shorter than minimum
	if err := rule.Validate("abc"); err == nil {
		t.Error("Should fail for string shorter than minimum")
	}

	// Test string equal to minimum
	if err := rule.Validate("abcde"); err != nil {
		t.Errorf("Should pass for string equal to minimum: %v", err)
	}

	// Test string longer than minimum
	if err := rule.Validate("abcdef"); err != nil {
		t.Errorf("Should pass for string longer than minimum: %v", err)
	}

	// Test non-string value
	if err := rule.Validate(123); err == nil {
		t.Error("Should fail for non-string value")
	}
}

func TestMaxLengthRule(t *testing.T) {
	rule := MaxLengthRule{Max: 5}

	// Test string longer than maximum
	if err := rule.Validate("abcdef"); err == nil {
		t.Error("Should fail for string longer than maximum")
	}

	// Test string equal to maximum
	if err := rule.Validate("abcde"); err != nil {
		t.Errorf("Should pass for string equal to maximum: %v", err)
	}

	// Test string shorter than maximum
	if err := rule.Validate("abc"); err != nil {
		t.Errorf("Should pass for string shorter than maximum: %v", err)
	}
}

func TestRangeRule(t *testing.T) {
	rule := RangeRule{Min: 10, Max: 100}

	// Test valid integers
	if err := rule.Validate(50); err != nil {
		t.Errorf("Should pass for valid integer: %v", err)
	}

	// Test valid float
	if err := rule.Validate(75.5); err != nil {
		t.Errorf("Should pass for valid float: %v", err)
	}

	// Test valid string number
	if err := rule.Validate("25"); err != nil {
		t.Errorf("Should pass for valid string number: %v", err)
	}

	// Test below minimum
	if err := rule.Validate(5); err == nil {
		t.Error("Should fail for value below minimum")
	}

	// Test above maximum
	if err := rule.Validate(150); err == nil {
		t.Error("Should fail for value above maximum")
	}

	// Test invalid string
	if err := rule.Validate("not-a-number"); err == nil {
		t.Error("Should fail for invalid string")
	}
}

func TestInRule(t *testing.T) {
	rule := InRule{AllowedValues: []interface{}{"red", "green", "blue"}}

	// Test valid values
	if err := rule.Validate("red"); err != nil {
		t.Errorf("Should pass for valid value: %v", err)
	}

	// Test invalid value
	if err := rule.Validate("yellow"); err == nil {
		t.Error("Should fail for invalid value")
	}
}

func TestNotInRule(t *testing.T) {
	rule := NotInRule{ForbiddenValues: []interface{}{"admin", "root", "system"}}

	// Test allowed value
	if err := rule.Validate("user"); err != nil {
		t.Errorf("Should pass for allowed value: %v", err)
	}

	// Test forbidden value
	if err := rule.Validate("admin"); err == nil {
		t.Error("Should fail for forbidden value")
	}
}

func TestRegexRule(t *testing.T) {
	rule, err := NewRegexRule(`^[a-zA-Z]+$`, "only letters allowed")
	if err != nil {
		t.Fatalf("Failed to create regex rule: %v", err)
	}

	// Test valid string
	if err := rule.Validate("abcDEF"); err != nil {
		t.Errorf("Should pass for valid string: %v", err)
	}

	// Test invalid string
	if err := rule.Validate("abc123"); err == nil {
		t.Error("Should fail for invalid string")
	}

	// Test empty string (should pass for non-required fields)
	if err := rule.Validate(""); err != nil {
		t.Errorf("Should pass for empty string: %v", err)
	}
}

func TestValidator(t *testing.T) {
	validator := NewValidator()
	validator.AddRule("name", RequiredRule{})
	validator.AddRule("name", MinLengthRule{Min: 2})
	validator.AddRule("email", EmailRule{})
	validator.AddRule("age", RangeRule{Min: 18, Max: 120})

	// Test valid data
	validData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   25,
	}

	result := validator.Validate(validData)
	if !result.Valid {
		t.Errorf("Should be valid: %v", result.Errors)
	}

	// Test invalid data
	invalidData := map[string]interface{}{
		"name":  "J",             // too short
		"email": "invalid-email", // invalid email
		"age":   15,              // too young
	}

	result = validator.Validate(invalidData)
	if result.Valid {
		t.Error("Should be invalid")
	}

	if len(result.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(result.Errors))
	}

	// Test missing required field
	missingData := map[string]interface{}{
		"email": "john@example.com",
		"age":   25,
		// missing required "name"
	}

	result = validator.Validate(missingData)
	if result.Valid {
		t.Error("Should be invalid for missing required field")
	}

	if _, exists := result.Errors["name"]; !exists {
		t.Error("Should have error for missing name field")
	}
}

func TestConvenienceValidators(t *testing.T) {
	// Test email validator
	emailValidator := CreateEmailValidator(true)

	validResult := emailValidator.Validate(map[string]interface{}{
		"email": "test@example.com",
	})
	if !validResult.Valid {
		t.Errorf("Email validator should pass for valid email: %v", validResult.Errors)
	}

	invalidResult := emailValidator.Validate(map[string]interface{}{
		"email": "invalid",
	})
	if invalidResult.Valid {
		t.Error("Email validator should fail for invalid email")
	}

	// Test URL validator
	urlValidator := CreateURLValidator(false) // not required

	validResult = urlValidator.Validate(map[string]interface{}{
		"url": "https://example.com",
	})
	if !validResult.Valid {
		t.Errorf("URL validator should pass for valid URL: %v", validResult.Errors)
	}

	// Test phone validator
	phoneValidator := CreatePhoneValidator(true)

	validResult = phoneValidator.Validate(map[string]interface{}{
		"phone": "+1234567890",
	})
	if !validResult.Valid {
		t.Errorf("Phone validator should pass for valid phone: %v", validResult.Errors)
	}
}

func TestValidateMessage(t *testing.T) {
	// Test valid message
	validMessage := map[string]interface{}{
		"title": "Test Message",
		"body":  "This is a test message",
		"targets": []interface{}{
			map[string]interface{}{
				"type":     "email",
				"value":    "test@example.com",
				"platform": "email",
			},
		},
		"priority": 2,
	}

	result := ValidateMessage(validMessage)
	if !result.Valid {
		t.Errorf("Should be valid message: %v", result.Errors)
	}

	// Test message without title or body
	invalidMessage := map[string]interface{}{
		"targets": []interface{}{
			map[string]interface{}{
				"type":     "email",
				"value":    "test@example.com",
				"platform": "email",
			},
		},
	}

	result = ValidateMessage(invalidMessage)
	if result.Valid {
		t.Error("Should be invalid for message without title or body")
	}

	// Test message without targets
	noTargetsMessage := map[string]interface{}{
		"title": "Test Message",
		"body":  "This is a test message",
	}

	result = ValidateMessage(noTargetsMessage)
	if result.Valid {
		t.Error("Should be invalid for message without targets")
	}

	// Test message with invalid target
	invalidTargetMessage := map[string]interface{}{
		"title": "Test Message",
		"body":  "This is a test message",
		"targets": []interface{}{
			map[string]interface{}{
				"type":     "email",
				"value":    "invalid-email",
				"platform": "email",
			},
		},
	}

	result = ValidateMessage(invalidTargetMessage)
	if result.Valid {
		t.Error("Should be invalid for message with invalid target")
	}
}

func TestSanitizeString(t *testing.T) {
	// Test control character removal
	input := "Hello\x00World\x01Test"
	expected := "HelloWorldTest"
	result := SanitizeString(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test whitespace normalization
	input = "  Hello   World  "
	expected = "Hello   World"
	result = SanitizeString(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test that normal whitespace is preserved
	input = "Hello\nWorld\tTest"
	result = SanitizeString(input)
	if result != input {
		t.Errorf("Normal whitespace should be preserved: %q -> %q", input, result)
	}
}

func TestSanitizeMap(t *testing.T) {
	input := map[string]interface{}{
		"name":  "John\x00Doe",
		"email": "  john@example.com  ",
		"age":   25,
	}

	result := SanitizeMap(input)

	if result["name"] != "JohnDoe" {
		t.Errorf("Expected sanitized name, got %v", result["name"])
	}

	if result["email"] != "john@example.com" {
		t.Errorf("Expected trimmed email, got %v", result["email"])
	}

	if result["age"] != 25 {
		t.Errorf("Non-string values should be unchanged, got %v", result["age"])
	}
}

func BenchmarkValidator(b *testing.B) {
	validator := NewValidator()
	validator.AddRule("name", RequiredRule{})
	validator.AddRule("name", MinLengthRule{Min: 2})
	validator.AddRule("email", EmailRule{})

	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(data)
	}
}

func BenchmarkEmailRule(b *testing.B) {
	rule := EmailRule{}
	email := "test@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rule.Validate(email)
	}
}
