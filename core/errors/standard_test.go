package errors

import (
	"errors"
	"testing"
)

func TestStandardErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotifyError
		wantCode ErrorCode
		wantCat  ErrorCategory
	}{
		{
			name:     "ErrInvalidConfig",
			err:      ErrInvalidConfig,
			wantCode: CodeInvalidConfig,
			wantCat:  CategoryConfig,
		},
		{
			name:     "ErrEmptyMessage",
			err:      ErrEmptyMessage,
			wantCode: CodeEmptyMessage,
			wantCat:  CategoryValidation,
		},
		{
			name:     "ErrInvalidTarget",
			err:      ErrInvalidTarget,
			wantCode: CodeInvalidTarget,
			wantCat:  CategoryValidation,
		},
		{
			name:     "ErrNetworkError",
			err:      ErrNetworkError,
			wantCode: CodeNetworkError,
			wantCat:  CategoryNetwork,
		},
		{
			name:     "ErrRateLimited",
			err:      ErrRateLimited,
			wantCode: CodeRateLimited,
			wantCat:  CategoryRateLimit,
		},
		{
			name:     "ErrInvalidCredentials",
			err:      ErrInvalidCredentials,
			wantCode: CodeInvalidCredentials,
			wantCat:  CategoryAuth,
		},
		{
			name:     "ErrSendingFailed",
			err:      ErrSendingFailed,
			wantCode: CodeSendingFailed,
			wantCat:  CategoryTransport,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("%s code = %v, want %v", tt.name, tt.err.Code, tt.wantCode)
			}
			if tt.err.Category != tt.wantCat {
				t.Errorf("%s category = %v, want %v", tt.name, tt.err.Category, tt.wantCat)
			}
		})
	}
}

func TestPlatformSpecificErrors(t *testing.T) {
	t.Run("NewFeishuError", func(t *testing.T) {
		err := NewFeishuError(CodeInvalidConfig, "test message")
		if err.Platform != "feishu" {
			t.Errorf("NewFeishuError() platform = %v, want feishu", err.Platform)
		}
		if err.Code != CodeInvalidConfig {
			t.Errorf("NewFeishuError() code = %v, want %v", err.Code, CodeInvalidConfig)
		}
	})

	t.Run("NewEmailError", func(t *testing.T) {
		err := NewEmailError(CodeRateLimited, "test message")
		if err.Platform != "email" {
			t.Errorf("NewEmailError() platform = %v, want email", err.Platform)
		}
		if err.Code != CodeRateLimited {
			t.Errorf("NewEmailError() code = %v, want %v", err.Code, CodeRateLimited)
		}
	})

	t.Run("NewSMSError", func(t *testing.T) {
		err := NewSMSError(CodeTimeout, "test message")
		if err.Platform != "sms" {
			t.Errorf("NewSMSError() platform = %v, want sms", err.Platform)
		}
		if err.Code != CodeTimeout {
			t.Errorf("NewSMSError() code = %v, want %v", err.Code, CodeTimeout)
		}
	})
}

func TestErrorCategoryCheckers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		checker func(error) bool
		want    bool
	}{
		{
			name:    "IsConfigurationError with config error",
			err:     ErrInvalidConfig,
			checker: IsConfigurationError,
			want:    true,
		},
		{
			name:    "IsConfigurationError with non-config error",
			err:     ErrNetworkError,
			checker: IsConfigurationError,
			want:    false,
		},
		{
			name:    "IsValidationError with validation error",
			err:     ErrEmptyMessage,
			checker: IsValidationError,
			want:    true,
		},
		{
			name:    "IsValidationError with non-validation error",
			err:     ErrNetworkError,
			checker: IsValidationError,
			want:    false,
		},
		{
			name:    "IsNetworkError with network error",
			err:     ErrNetworkError,
			checker: IsNetworkError,
			want:    true,
		},
		{
			name:    "IsNetworkError with non-network error",
			err:     ErrInvalidConfig,
			checker: IsNetworkError,
			want:    false,
		},
		{
			name:    "IsAuthError with auth error",
			err:     ErrInvalidCredentials,
			checker: IsAuthError,
			want:    true,
		},
		{
			name:    "IsAuthError with non-auth error",
			err:     ErrNetworkError,
			checker: IsAuthError,
			want:    false,
		},
		{
			name:    "IsRateLimitError with rate limit error",
			err:     ErrRateLimited,
			checker: IsRateLimitError,
			want:    true,
		},
		{
			name:    "IsRateLimitError with non-rate-limit error",
			err:     ErrNetworkError,
			checker: IsRateLimitError,
			want:    false,
		},
		{
			name:    "IsRetryableError with retryable error",
			err:     ErrNetworkError,
			checker: IsRetryableError,
			want:    true,
		},
		{
			name:    "IsRetryableError with non-retryable error",
			err:     ErrInvalidConfig,
			checker: IsRetryableError,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.checker(tt.err); got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestErrorCheckersWithStandardErrors(t *testing.T) {
	// Test with standard Go error
	stdErr := errors.New("standard error")

	checkers := []struct {
		name    string
		checker func(error) bool
	}{
		{"IsConfigurationError", IsConfigurationError},
		{"IsValidationError", IsValidationError},
		{"IsNetworkError", IsNetworkError},
		{"IsAuthError", IsAuthError},
		{"IsRateLimitError", IsRateLimitError},
		{"IsRetryableError", IsRetryableError},
		{"IsTemporaryError", IsTemporaryError},
	}

	for _, checker := range checkers {
		t.Run(checker.name+"_with_standard_error", func(t *testing.T) {
			if checker.checker(stdErr) {
				t.Errorf("%s should return false for standard errors", checker.name)
			}
		})
	}
}

func TestLegacyErrorCompatibility(t *testing.T) {
	// Test that legacy error variables map to new standard errors
	tests := []struct {
		name     string
		legacy   *NotifyError
		standard *NotifyError
	}{
		{
			name:     "ErrMissingTemplate maps to ErrTemplateError",
			legacy:   ErrMissingTemplate,
			standard: ErrTemplateError,
		},
		{
			name:     "ErrInvalidTargetType maps to ErrInvalidTarget",
			legacy:   ErrInvalidTargetType,
			standard: ErrInvalidTarget,
		},
		{
			name:     "ErrEmptyTargetValue maps to ErrEmptyTarget",
			legacy:   ErrEmptyTargetValue,
			standard: ErrEmptyTarget,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.legacy != tt.standard {
				t.Errorf("Legacy error %s should map to standard error", tt.name)
			}
		})
	}
}

func TestNew(t *testing.T) {
	err := New(CodeInvalidConfig, CategoryConfig, "test message")

	if err.Code != CodeInvalidConfig {
		t.Errorf("New() code = %v, want %v", err.Code, CodeInvalidConfig)
	}
	if err.Category != CategoryConfig {
		t.Errorf("New() category = %v, want %v", err.Category, CategoryConfig)
	}
	if err.Message != "test message" {
		t.Errorf("New() message = %v, want test message", err.Message)
	}
	if err.Platform != "" {
		t.Errorf("New() platform = %v, want empty", err.Platform)
	}
}

func TestNewWithPlatform(t *testing.T) {
	err := NewWithPlatform(CodeRateLimited, CategoryRateLimit, "test message", "feishu")

	if err.Code != CodeRateLimited {
		t.Errorf("NewWithPlatform() code = %v, want %v", err.Code, CodeRateLimited)
	}
	if err.Category != CategoryRateLimit {
		t.Errorf("NewWithPlatform() category = %v, want %v", err.Category, CategoryRateLimit)
	}
	if err.Message != "test message" {
		t.Errorf("NewWithPlatform() message = %v, want test message", err.Message)
	}
	if err.Platform != "feishu" {
		t.Errorf("NewWithPlatform() platform = %v, want feishu", err.Platform)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(CodeNetworkError, CategoryNetwork, "wrapper message", cause)

	if err.Code != CodeNetworkError {
		t.Errorf("Wrap() code = %v, want %v", err.Code, CodeNetworkError)
	}
	if err.Category != CategoryNetwork {
		t.Errorf("Wrap() category = %v, want %v", err.Category, CategoryNetwork)
	}
	if err.Message != "wrapper message" {
		t.Errorf("Wrap() message = %v, want wrapper message", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("Wrap() cause = %v, want %v", err.Cause, cause)
	}
}

func TestWrapWithPlatform(t *testing.T) {
	cause := errors.New("underlying error")
	err := WrapWithPlatform(CodeTimeout, CategoryNetwork, "wrapper message", "email", cause)

	if err.Code != CodeTimeout {
		t.Errorf("WrapWithPlatform() code = %v, want %v", err.Code, CodeTimeout)
	}
	if err.Category != CategoryNetwork {
		t.Errorf("WrapWithPlatform() category = %v, want %v", err.Category, CategoryNetwork)
	}
	if err.Message != "wrapper message" {
		t.Errorf("WrapWithPlatform() message = %v, want wrapper message", err.Message)
	}
	if err.Platform != "email" {
		t.Errorf("WrapWithPlatform() platform = %v, want email", err.Platform)
	}
	if err.Cause != cause {
		t.Errorf("WrapWithPlatform() cause = %v, want %v", err.Cause, cause)
	}
}
