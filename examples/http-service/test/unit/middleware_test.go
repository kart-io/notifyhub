package unit

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/examples/http-service/internal/middleware"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/logger/adapters"
)

func TestJSONMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := middleware.JSONMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type to be application/json, got %s", contentType)
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsMiddleware := middleware.CORSMiddleware(handler)

	// Test GET request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	corsMiddleware.ServeHTTP(rr, req)

	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	for header, expected := range expectedHeaders {
		if actual := rr.Header().Get(header); actual != expected {
			t.Errorf("Expected %s to be %s, got %s", header, expected, actual)
		}
	}

	// Test OPTIONS request
	req = httptest.NewRequest(http.MethodOptions, "/test", nil)
	rr = httptest.NewRecorder()

	corsMiddleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS request, got %d", rr.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Info)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	loggingMiddleware := middleware.LoggingMiddleware(testLogger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	start := time.Now()
	loggingMiddleware.ServeHTTP(rr, req)
	duration := time.Since(start)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Duration should be reasonable (less than 100ms for this simple test)
	if duration > 100*time.Millisecond {
		t.Errorf("Request took too long: %v", duration)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Set a very low rate limit for testing
	rateLimitMiddleware := middleware.RateLimitMiddleware(2)(handler)

	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	rr1 := httptest.NewRecorder()

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "127.0.0.1:12345"
	rr2 := httptest.NewRecorder()

	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "127.0.0.1:12345"
	rr3 := httptest.NewRecorder()

	// First two requests should succeed
	rateLimitMiddleware.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("First request should succeed, got status %d", rr1.Code)
	}

	rateLimitMiddleware.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("Second request should succeed, got status %d", rr2.Code)
	}

	// Third request should be rate limited
	rateLimitMiddleware.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusTooManyRequests {
		t.Errorf("Third request should be rate limited, got status %d", rr3.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		apiKey         string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "no auth required",
			apiKey:         "",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid auth",
			apiKey:         "test-key",
			authHeader:     "Bearer test-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing auth header",
			apiKey:         "test-key",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid auth format",
			apiKey:         "test-key",
			authHeader:     "Basic test-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid api key",
			apiKey:         "test-key",
			authHeader:     "Bearer wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMiddleware := middleware.AuthMiddleware(tt.apiKey)(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			authMiddleware.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Error)

	// Handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	recoveryMiddleware := middleware.RecoveryMiddleware(testLogger)(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic
	recoveryMiddleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic recovery, got %d", rr.Code)
	}
}

func TestValidateContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	validateMiddleware := middleware.ValidateContentType(handler)

	tests := []struct {
		name           string
		method         string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "GET request - no validation",
			method:         http.MethodGet,
			contentType:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST with valid content type",
			method:         http.MethodPost,
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST with invalid content type",
			method:         http.MethodPost,
			contentType:    "text/plain",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "PUT with valid content type",
			method:         http.MethodPut,
			contentType:    "application/json; charset=utf-8",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT with invalid content type",
			method:         http.MethodPut,
			contentType:    "application/xml",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()

			validateMiddleware.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestRequestSizeLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	sizeLimit := int64(100) // 100 bytes limit
	sizeLimitMiddleware := middleware.RequestSizeLimit(sizeLimit)(handler)

	tests := []struct {
		name           string
		bodySize       int
		expectedStatus int
	}{
		{
			name:           "small request",
			bodySize:       50,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "large request",
			bodySize:       200,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(strings.Repeat("a", tt.bodySize))
			req := httptest.NewRequest(http.MethodPost, "/test", body)
			req.ContentLength = int64(tt.bodySize)
			rr := httptest.NewRecorder()

			sizeLimitMiddleware.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	securityMiddleware := middleware.SecurityHeaders(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	securityMiddleware.ServeHTTP(rr, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":   "nosniff",
		"X-Frame-Options":          "DENY",
		"X-XSS-Protection":         "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}

	for header, expected := range expectedHeaders {
		if actual := rr.Header().Get(header); actual != expected {
			t.Errorf("Expected %s to be %s, got %s", header, expected, actual)
		}
	}
}

// Benchmark tests for middleware
func BenchmarkJSONMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := middleware.JSONMiddleware(handler)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)
		}
	})
}

func BenchmarkLoggingMiddleware(b *testing.B) {
	testLogger := adapters.NewStdLogAdapter(log.New(os.Stderr, "", log.LstdFlags), logger.Error) // Reduce log level for benchmarking
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	loggingMiddleware := middleware.LoggingMiddleware(testLogger)(handler)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()
			loggingMiddleware.ServeHTTP(rr, req)
		}
	})
}