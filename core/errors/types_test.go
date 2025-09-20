package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestNotifyError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotifyError
		expected string
	}{
		{
			name: "basic error",
			err: &NotifyError{
				Code:     CodeInvalidConfig,
				Category: CategoryConfig,
				Message:  "invalid configuration",
			},
			expected: "[CONFIG:INVALID_CONFIG] invalid configuration",
		},
		{
			name: "error with platform",
			err: &NotifyError{
				Code:     CodeRateLimited,
				Category: CategoryRateLimit,
				Message:  "rate limit exceeded",
				Platform: "feishu",
			},
			expected: "[RATE_LIMIT:RATE_LIMITED] rate limit exceeded (platform: feishu)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("NotifyError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNotifyError_IsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  *NotifyError
		want bool
	}{
		{
			name: "network error is retryable",
			err:  &NotifyError{Code: CodeNetworkError},
			want: true,
		},
		{
			name: "timeout is retryable",
			err:  &NotifyError{Code: CodeTimeout},
			want: true,
		},
		{
			name: "rate limited is retryable",
			err:  &NotifyError{Code: CodeRateLimited},
			want: true,
		},
		{
			name: "server error is retryable",
			err:  &NotifyError{Code: CodeServerError},
			want: true,
		},
		{
			name: "invalid config is not retryable",
			err:  &NotifyError{Code: CodeInvalidConfig},
			want: false,
		},
		{
			name: "unauthorized is not retryable",
			err:  &NotifyError{Code: CodeUnauthorized},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsRetryable(); got != tt.want {
				t.Errorf("NotifyError.IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotifyError_HTTPStatusCode(t *testing.T) {
	tests := []struct {
		name string
		err  *NotifyError
		want int
	}{
		{
			name: "invalid config returns 400",
			err:  &NotifyError{Code: CodeInvalidConfig},
			want: http.StatusBadRequest,
		},
		{
			name: "unauthorized returns 401",
			err:  &NotifyError{Code: CodeUnauthorized},
			want: http.StatusUnauthorized,
		},
		{
			name: "forbidden returns 403",
			err:  &NotifyError{Code: CodeForbidden},
			want: http.StatusForbidden,
		},
		{
			name: "not found returns 404",
			err:  &NotifyError{Code: CodeNotFound},
			want: http.StatusNotFound,
		},
		{
			name: "rate limited returns 429",
			err:  &NotifyError{Code: CodeRateLimited},
			want: http.StatusTooManyRequests,
		},
		{
			name: "server error returns 500",
			err:  &NotifyError{Code: CodeServerError},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.HTTPStatusCode(); got != tt.want {
				t.Errorf("NotifyError.HTTPStatusCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotifyError_Is(t *testing.T) {
	err1 := &NotifyError{Code: CodeInvalidConfig, Category: CategoryConfig}
	err2 := &NotifyError{Code: CodeInvalidConfig, Category: CategoryConfig}
	err3 := &NotifyError{Code: CodeRateLimited, Category: CategoryRateLimit}
	stdErr := errors.New("standard error")

	tests := []struct {
		name   string
		err    *NotifyError
		target error
		want   bool
	}{
		{
			name:   "same error codes match",
			err:    err1,
			target: err2,
			want:   true,
		},
		{
			name:   "different error codes don't match",
			err:    err1,
			target: err3,
			want:   false,
		},
		{
			name:   "standard error doesn't match",
			err:    err1,
			target: stdErr,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Is(tt.target); got != tt.want {
				t.Errorf("NotifyError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotifyError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &NotifyError{
		Code:    CodeNetworkError,
		Message: "wrapped error",
		Cause:   cause,
	}

	if got := err.Unwrap(); got != cause {
		t.Errorf("NotifyError.Unwrap() = %v, want %v", got, cause)
	}
}

func TestMapHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		platform   string
		wantCode   ErrorCode
		wantCat    ErrorCategory
	}{
		{
			name:       "401 unauthorized",
			statusCode: 401,
			body:       "Authentication required",
			platform:   "feishu",
			wantCode:   CodeUnauthorized,
			wantCat:    CategoryAuth,
		},
		{
			name:       "403 forbidden",
			statusCode: 403,
			body:       "Access denied",
			platform:   "email",
			wantCode:   CodeForbidden,
			wantCat:    CategoryAuth,
		},
		{
			name:       "429 rate limited",
			statusCode: 429,
			body:       "Too many requests",
			platform:   "feishu",
			wantCode:   CodeRateLimited,
			wantCat:    CategoryRateLimit,
		},
		{
			name:       "500 server error",
			statusCode: 500,
			body:       "Internal server error",
			platform:   "email",
			wantCode:   CodeServerError,
			wantCat:    CategoryNetwork,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MapHTTPError(tt.statusCode, tt.body, tt.platform)
			if err.Code != tt.wantCode {
				t.Errorf("MapHTTPError() code = %v, want %v", err.Code, tt.wantCode)
			}
			if err.Category != tt.wantCat {
				t.Errorf("MapHTTPError() category = %v, want %v", err.Category, tt.wantCat)
			}
			if err.Platform != tt.platform {
				t.Errorf("MapHTTPError() platform = %v, want %v", err.Platform, tt.platform)
			}
		})
	}
}

func TestMapNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		platform string
		wantCode ErrorCode
	}{
		{
			name:     "timeout error",
			err:      errors.New("connection timeout"),
			platform: "feishu",
			wantCode: CodeTimeout,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			platform: "email",
			wantCode: CodeNetworkError,
		},
		{
			name:     "generic network error",
			err:      errors.New("network unreachable"),
			platform: "sms",
			wantCode: CodeNetworkError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapNetworkError(tt.err, tt.platform)
			if result.Code != tt.wantCode {
				t.Errorf("MapNetworkError() code = %v, want %v", result.Code, tt.wantCode)
			}
			if result.Platform != tt.platform {
				t.Errorf("MapNetworkError() platform = %v, want %v", result.Platform, tt.platform)
			}
		})
	}
}

func TestMapSMTPError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode ErrorCode
	}{
		{
			name:     "authentication failed",
			err:      errors.New("535 authentication failed"),
			wantCode: CodeInvalidCredentials,
		},
		{
			name:     "rate limit",
			err:      errors.New("421 too many connections"),
			wantCode: CodeRateLimited,
		},
		{
			name:     "timeout",
			err:      errors.New("smtp timeout"),
			wantCode: CodeTimeout,
		},
		{
			name:     "invalid recipient",
			err:      errors.New("550 invalid recipient"),
			wantCode: CodeInvalidTarget,
		},
		{
			name:     "generic SMTP error",
			err:      errors.New("smtp send failed"),
			wantCode: CodeSendingFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapSMTPError(tt.err)
			if result.Code != tt.wantCode {
				t.Errorf("MapSMTPError() code = %v, want %v", result.Code, tt.wantCode)
			}
			if result.Platform != "email" {
				t.Errorf("MapSMTPError() platform = %v, want email", result.Platform)
			}
		})
	}
}

// Test helper functions for timeout and connection error detection
func Test_isTimeoutError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "timeout error",
			err:  errors.New("connection timeout"),
			want: true,
		},
		{
			name: "deadline exceeded",
			err:  errors.New("context deadline exceeded"),
			want: true,
		},
		{
			name: "i/o timeout",
			err:  errors.New("i/o timeout"),
			want: true,
		},
		{
			name: "not a timeout error",
			err:  errors.New("connection refused"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTimeoutError(tt.err); got != tt.want {
				t.Errorf("isTimeoutError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection refused",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "connection reset",
			err:  errors.New("connection reset by peer"),
			want: true,
		},
		{
			name: "network unreachable",
			err:  errors.New("network unreachable"),
			want: true,
		},
		{
			name: "not a connection error",
			err:  errors.New("authentication failed"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isConnectionError(tt.err); got != tt.want {
				t.Errorf("isConnectionError() = %v, want %v", got, tt.want)
			}
		})
	}
}
