package errors

import (
	"testing"
)

func TestGetErrorInfo(t *testing.T) {
	tests := []struct {
		name           string
		code           Code
		expectedCode   Code
		expectedCat    string
		expectedSev    string
		expectedRetry  bool
		expectedDesc   string
	}{
		{
			name:          "configuration error",
			code:          ErrInvalidConfig,
			expectedCode:  ErrInvalidConfig,
			expectedCat:   ConfigurationCategory,
			expectedSev:   "ERROR",
			expectedRetry: false,
			expectedDesc:  "Invalid configuration provided",
		},
		{
			name:          "platform unavailable",
			code:          ErrPlatformUnavailable,
			expectedCode:  ErrPlatformUnavailable,
			expectedCat:   PlatformCategory,
			expectedSev:   "ERROR",
			expectedRetry: true,
			expectedDesc:  "Platform currently unavailable",
		},
		{
			name:          "rate limit exceeded",
			code:          ErrPlatformRateLimit,
			expectedCode:  ErrPlatformRateLimit,
			expectedCat:   PlatformCategory,
			expectedSev:   "WARN",
			expectedRetry: true,
			expectedDesc:  "Platform rate limit exceeded",
		},
		{
			name:          "message too large",
			code:          ErrMessageTooLarge,
			expectedCode:  ErrMessageTooLarge,
			expectedCat:   MessageCategory,
			expectedSev:   "ERROR",
			expectedRetry: false,
			expectedDesc:  "Message size exceeds limit",
		},
		{
			name:          "network timeout",
			code:          ErrNetworkTimeout,
			expectedCode:  ErrNetworkTimeout,
			expectedCat:   NetworkCategory,
			expectedSev:   "WARN",
			expectedRetry: true,
			expectedDesc:  "Network timeout",
		},
		{
			name:          "system unavailable",
			code:          ErrSystemUnavailable,
			expectedCode:  ErrSystemUnavailable,
			expectedCat:   SystemCategory,
			expectedSev:   "CRITICAL",
			expectedRetry: true,
			expectedDesc:  "System unavailable",
		},
		{
			name:          "unknown error code",
			code:          Code("UNKNOWN999"),
			expectedCode:  Code("UNKNOWN999"),
			expectedCat:   "UNKNOWN",
			expectedSev:   "ERROR",
			expectedRetry: false,
			expectedDesc:  "Unknown error code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := GetErrorInfo(tt.code)

			if info.Code != tt.expectedCode {
				t.Errorf("Expected code %v, got %v", tt.expectedCode, info.Code)
			}

			if info.Category != tt.expectedCat {
				t.Errorf("Expected category %v, got %v", tt.expectedCat, info.Category)
			}

			if info.Severity != tt.expectedSev {
				t.Errorf("Expected severity %v, got %v", tt.expectedSev, info.Severity)
			}

			if info.Retryable != tt.expectedRetry {
				t.Errorf("Expected retryable %v, got %v", tt.expectedRetry, info.Retryable)
			}

			if info.Description != tt.expectedDesc {
				t.Errorf("Expected description %v, got %v", tt.expectedDesc, info.Description)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		code     Code
		expected bool
	}{
		{"platform unavailable should be retryable", ErrPlatformUnavailable, true},
		{"platform rate limit should be retryable", ErrPlatformRateLimit, true},
		{"platform timeout should be retryable", ErrPlatformTimeout, true},
		{"network timeout should be retryable", ErrNetworkTimeout, true},
		{"network connection should be retryable", ErrNetworkConnection, true},
		{"queue full should be retryable", ErrQueueFull, true},
		{"system unavailable should be retryable", ErrSystemUnavailable, true},
		{"resource exhausted should be retryable", ErrResourceExhausted, true},

		{"invalid config should not be retryable", ErrInvalidConfig, false},
		{"invalid message should not be retryable", ErrInvalidMessage, false},
		{"message too large should not be retryable", ErrMessageTooLarge, false},
		{"platform auth should not be retryable", ErrPlatformAuth, false},
		{"template invalid should not be retryable", ErrTemplateInvalid, false},
		{"validation failed should not be retryable", ErrValidationFailed, false},
		{"permission denied should not be retryable", ErrPermissionDenied, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.code); got != tt.expected {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestGetCategory(t *testing.T) {
	tests := []struct {
		name     string
		code     Code
		expected string
	}{
		{"config error should return config category", ErrInvalidConfig, ConfigurationCategory},
		{"platform error should return platform category", ErrPlatformUnavailable, PlatformCategory},
		{"message error should return message category", ErrInvalidMessage, MessageCategory},
		{"template error should return template category", ErrTemplateNotFound, TemplateCategory},
		{"queue error should return queue category", ErrQueueFull, QueueCategory},
		{"network error should return network category", ErrNetworkTimeout, NetworkCategory},
		{"validation error should return validation category", ErrValidationFailed, ValidationCategory},
		{"system error should return system category", ErrSystemUnavailable, SystemCategory},
		{"unknown error should return unknown category", Code("UNKNOWN999"), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCategory(tt.code); got != tt.expected {
				t.Errorf("GetCategory(%v) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestGetSeverity(t *testing.T) {
	tests := []struct {
		name     string
		code     Code
		expected string
	}{
		{"config error should be ERROR", ErrInvalidConfig, "ERROR"},
		{"platform unavailable should be ERROR", ErrPlatformUnavailable, "ERROR"},
		{"platform rate limit should be WARN", ErrPlatformRateLimit, "WARN"},
		{"platform timeout should be WARN", ErrPlatformTimeout, "WARN"},
		{"network timeout should be WARN", ErrNetworkTimeout, "WARN"},
		{"queue full should be WARN", ErrQueueFull, "WARN"},
		{"queue empty should be INFO", ErrQueueEmpty, "INFO"},
		{"system unavailable should be CRITICAL", ErrSystemUnavailable, "CRITICAL"},
		{"resource exhausted should be CRITICAL", ErrResourceExhausted, "CRITICAL"},
		{"system overload should be CRITICAL", ErrSystemOverload, "CRITICAL"},
		{"unknown error should be ERROR", Code("UNKNOWN999"), "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSeverity(tt.code); got != tt.expected {
				t.Errorf("GetSeverity(%v) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Test that all error codes have the expected format
	tests := []struct {
		name     string
		code     Code
		category string
	}{
		// Configuration errors
		{"ErrInvalidConfig", ErrInvalidConfig, "CON"},
		{"ErrMissingConfig", ErrMissingConfig, "CON"},
		{"ErrConfigValidation", ErrConfigValidation, "CON"},
		{"ErrUnsupportedConfig", ErrUnsupportedConfig, "CON"},
		{"ErrConfigLoadFailed", ErrConfigLoadFailed, "CON"},

		// Platform errors
		{"ErrPlatformNotFound", ErrPlatformNotFound, "PLT"},
		{"ErrPlatformUnavailable", ErrPlatformUnavailable, "PLT"},
		{"ErrPlatformAuth", ErrPlatformAuth, "PLT"},
		{"ErrPlatformRateLimit", ErrPlatformRateLimit, "PLT"},
		{"ErrPlatformTimeout", ErrPlatformTimeout, "PLT"},
		{"ErrPlatformInternal", ErrPlatformInternal, "PLT"},
		{"ErrPlatformMaintenance", ErrPlatformMaintenance, "PLT"},

		// Message errors
		{"ErrInvalidMessage", ErrInvalidMessage, "MSG"},
		{"ErrMessageTooLarge", ErrMessageTooLarge, "MSG"},
		{"ErrInvalidTarget", ErrInvalidTarget, "MSG"},
		{"ErrMessageEncoding", ErrMessageEncoding, "MSG"},
		{"ErrMessageSendFailed", ErrMessageSendFailed, "MSG"},
		{"ErrMessageTimeout", ErrMessageTimeout, "MSG"},

		// Template errors
		{"ErrTemplateNotFound", ErrTemplateNotFound, "TPL"},
		{"ErrTemplateInvalid", ErrTemplateInvalid, "TPL"},
		{"ErrTemplateRender", ErrTemplateRender, "TPL"},
		{"ErrTemplateEngine", ErrTemplateEngine, "TPL"},
		{"ErrTemplateVariables", ErrTemplateVariables, "TPL"},
		{"ErrTemplateCacheError", ErrTemplateCacheError, "TPL"},

		// Queue errors
		{"ErrQueueFull", ErrQueueFull, "QUE"},
		{"ErrQueueEmpty", ErrQueueEmpty, "QUE"},
		{"ErrQueueTimeout", ErrQueueTimeout, "QUE"},
		{"ErrQueueConnection", ErrQueueConnection, "QUE"},
		{"ErrQueueSerialization", ErrQueueSerialization, "QUE"},
		{"ErrQueueWorkerFailed", ErrQueueWorkerFailed, "QUE"},

		// Network errors
		{"ErrNetworkTimeout", ErrNetworkTimeout, "NET"},
		{"ErrNetworkConnection", ErrNetworkConnection, "NET"},
		{"ErrNetworkDNS", ErrNetworkDNS, "NET"},
		{"ErrNetworkSSL", ErrNetworkSSL, "NET"},
		{"ErrNetworkProtocol", ErrNetworkProtocol, "NET"},

		// Validation errors
		{"ErrValidationFailed", ErrValidationFailed, "VAL"},
		{"ErrInvalidFormat", ErrInvalidFormat, "VAL"},
		{"ErrMissingRequired", ErrMissingRequired, "VAL"},
		{"ErrValueOutOfRange", ErrValueOutOfRange, "VAL"},
		{"ErrInvalidType", ErrInvalidType, "VAL"},

		// System errors
		{"ErrSystemUnavailable", ErrSystemUnavailable, "SYS"},
		{"ErrInternalError", ErrInternalError, "SYS"},
		{"ErrResourceExhausted", ErrResourceExhausted, "SYS"},
		{"ErrPermissionDenied", ErrPermissionDenied, "SYS"},
		{"ErrSystemTimeout", ErrSystemTimeout, "SYS"},
		{"ErrSystemOverload", ErrSystemOverload, "SYS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codeStr := string(tt.code)

			// Check that code starts with expected category prefix
			if len(codeStr) < 6 {
				t.Errorf("Error code %v should be at least 6 characters long", tt.code)
				return
			}

			prefix := codeStr[:3]
			if prefix != tt.category {
				t.Errorf("Error code %v should start with %v, got %v", tt.code, tt.category, prefix)
			}

			// Check that it has a numeric suffix
			suffix := codeStr[3:]
			if len(suffix) != 3 {
				t.Errorf("Error code %v should have 3-digit numeric suffix, got %v", tt.code, suffix)
			}

			// Verify the error code is registered in the error info map
			info := GetErrorInfo(tt.code)
			if info.Code != tt.code {
				t.Errorf("Error code %v not properly registered in error info map", tt.code)
			}
		})
	}
}

func TestFactoryFunctions(t *testing.T) {
	t.Run("NewConfigError", func(t *testing.T) {
		err := NewConfigError(ErrInvalidConfig, "invalid config provided")

		if err.Code != ErrInvalidConfig {
			t.Errorf("Expected code %v, got %v", ErrInvalidConfig, err.Code)
		}

		if err.Message != "invalid config provided" {
			t.Errorf("Expected message 'invalid config provided', got %v", err.Message)
		}

		if err.Context["category"] != ConfigurationCategory {
			t.Errorf("Expected category %v, got %v", ConfigurationCategory, err.Context["category"])
		}
	})

	t.Run("NewPlatformError", func(t *testing.T) {
		err := NewPlatformError(ErrPlatformUnavailable, "feishu", "platform is down")

		if err.Code != ErrPlatformUnavailable {
			t.Errorf("Expected code %v, got %v", ErrPlatformUnavailable, err.Code)
		}

		if err.Message != "platform is down" {
			t.Errorf("Expected message 'platform is down', got %v", err.Message)
		}

		if err.Context["category"] != PlatformCategory {
			t.Errorf("Expected category %v, got %v", PlatformCategory, err.Context["category"])
		}

		if err.Context["platform"] != "feishu" {
			t.Errorf("Expected platform 'feishu', got %v", err.Context["platform"])
		}
	})

	t.Run("NewMessageError", func(t *testing.T) {
		err := NewMessageError(ErrMessageTooLarge, "msg-123", "message exceeds limit")

		if err.Code != ErrMessageTooLarge {
			t.Errorf("Expected code %v, got %v", ErrMessageTooLarge, err.Code)
		}

		if err.Message != "message exceeds limit" {
			t.Errorf("Expected message 'message exceeds limit', got %v", err.Message)
		}

		if err.Context["category"] != MessageCategory {
			t.Errorf("Expected category %v, got %v", MessageCategory, err.Context["category"])
		}

		if err.Context["message_id"] != "msg-123" {
			t.Errorf("Expected message_id 'msg-123', got %v", err.Context["message_id"])
		}
	})

	t.Run("NewTemplateError", func(t *testing.T) {
		err := NewTemplateError(ErrTemplateNotFound, "welcome.tmpl", "template not found")

		if err.Code != ErrTemplateNotFound {
			t.Errorf("Expected code %v, got %v", ErrTemplateNotFound, err.Code)
		}

		if err.Context["template"] != "welcome.tmpl" {
			t.Errorf("Expected template 'welcome.tmpl', got %v", err.Context["template"])
		}
	})

	t.Run("NewQueueError", func(t *testing.T) {
		err := NewQueueError(ErrQueueFull, "notifications", "queue is full")

		if err.Code != ErrQueueFull {
			t.Errorf("Expected code %v, got %v", ErrQueueFull, err.Code)
		}

		if err.Context["queue"] != "notifications" {
			t.Errorf("Expected queue 'notifications', got %v", err.Context["queue"])
		}
	})

	t.Run("NewNetworkError", func(t *testing.T) {
		err := NewNetworkError(ErrNetworkTimeout, "https://api.feishu.cn", "request timeout")

		if err.Code != ErrNetworkTimeout {
			t.Errorf("Expected code %v, got %v", ErrNetworkTimeout, err.Code)
		}

		if err.Context["endpoint"] != "https://api.feishu.cn" {
			t.Errorf("Expected endpoint 'https://api.feishu.cn', got %v", err.Context["endpoint"])
		}
	})

	t.Run("NewValidationError", func(t *testing.T) {
		err := NewValidationError(ErrMissingRequired, "webhook_url", "field is required")

		if err.Code != ErrMissingRequired {
			t.Errorf("Expected code %v, got %v", ErrMissingRequired, err.Code)
		}

		if err.Context["field"] != "webhook_url" {
			t.Errorf("Expected field 'webhook_url', got %v", err.Context["field"])
		}
	})

	t.Run("NewSystemError", func(t *testing.T) {
		err := NewSystemError(ErrSystemUnavailable, "dispatcher", "system overloaded")

		if err.Code != ErrSystemUnavailable {
			t.Errorf("Expected code %v, got %v", ErrSystemUnavailable, err.Code)
		}

		if err.Context["component"] != "dispatcher" {
			t.Errorf("Expected component 'dispatcher', got %v", err.Context["component"])
		}
	})
}

func TestErrorCodeCoverage(t *testing.T) {
	// Ensure all defined error codes are tested and have metadata
	allCodes := []Code{
		// Configuration errors
		ErrInvalidConfig, ErrMissingConfig, ErrConfigValidation, ErrUnsupportedConfig, ErrConfigLoadFailed,
		// Platform errors
		ErrPlatformNotFound, ErrPlatformUnavailable, ErrPlatformAuth, ErrPlatformRateLimit,
		ErrPlatformTimeout, ErrPlatformInternal, ErrPlatformMaintenance,
		// Message errors
		ErrInvalidMessage, ErrMessageTooLarge, ErrInvalidTarget, ErrMessageEncoding,
		ErrMessageSendFailed, ErrMessageTimeout,
		// Template errors
		ErrTemplateNotFound, ErrTemplateInvalid, ErrTemplateRender, ErrTemplateEngine,
		ErrTemplateVariables, ErrTemplateCacheError,
		// Queue errors
		ErrQueueFull, ErrQueueEmpty, ErrQueueTimeout, ErrQueueConnection,
		ErrQueueSerialization, ErrQueueWorkerFailed,
		// Network errors
		ErrNetworkTimeout, ErrNetworkConnection, ErrNetworkDNS, ErrNetworkSSL, ErrNetworkProtocol,
		// Validation errors
		ErrValidationFailed, ErrInvalidFormat, ErrMissingRequired, ErrValueOutOfRange, ErrInvalidType,
		// System errors
		ErrSystemUnavailable, ErrInternalError, ErrResourceExhausted, ErrPermissionDenied,
		ErrSystemTimeout, ErrSystemOverload,
	}

	for _, code := range allCodes {
		t.Run(string(code), func(t *testing.T) {
			info := GetErrorInfo(code)

			// Every error code should have proper metadata
			if info.Code != code {
				t.Errorf("Error code %v not found in metadata", code)
			}

			if info.Category == "" {
				t.Errorf("Error code %v missing category", code)
			}

			if info.Severity == "" {
				t.Errorf("Error code %v missing severity", code)
			}

			if info.Description == "" {
				t.Errorf("Error code %v missing description", code)
			}

			// Verify category mapping is correct
			codeStr := string(code)
			expectedCategory := ""
			switch codeStr[:3] {
			case "CON":
				expectedCategory = ConfigurationCategory
			case "PLT":
				expectedCategory = PlatformCategory
			case "MSG":
				expectedCategory = MessageCategory
			case "TPL":
				expectedCategory = TemplateCategory
			case "QUE":
				expectedCategory = QueueCategory
			case "NET":
				expectedCategory = NetworkCategory
			case "VAL":
				expectedCategory = ValidationCategory
			case "SYS":
				expectedCategory = SystemCategory
			}

			if info.Category != expectedCategory {
				t.Errorf("Error code %v has incorrect category %v, expected %v", code, info.Category, expectedCategory)
			}
		})
	}
}

func BenchmarkGetErrorInfo(b *testing.B) {
	codes := []Code{
		ErrInvalidConfig,
		ErrPlatformUnavailable,
		ErrMessageTooLarge,
		ErrNetworkTimeout,
		ErrSystemUnavailable,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code := codes[i%len(codes)]
		_ = GetErrorInfo(code)
	}
}

func BenchmarkIsRetryable(b *testing.B) {
	codes := []Code{
		ErrInvalidConfig,
		ErrPlatformUnavailable,
		ErrMessageTooLarge,
		ErrNetworkTimeout,
		ErrSystemUnavailable,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code := codes[i%len(codes)]
		_ = IsRetryable(code)
	}
}