package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ================================
// Optimized NotifyHubMiddleware Usage
// ================================

// MiddlewareEngine provides a unified interface for different framework middleware patterns
type MiddlewareEngine struct {
	hub           *Hub
	framework     string
	config        *MiddlewareConfig
	metrics       *MiddlewareMetrics
	errorHandlers map[string]ErrorHandler
	interceptors  []RequestInterceptor
	mu            sync.RWMutex
}

// MiddlewareConfig holds middleware configuration
type MiddlewareConfig struct {
	// Auto-handling options
	AutoParseRequest   bool          // Automatically parse request body
	AutoHandleResponse bool          // Automatically format responses
	AutoErrorRecovery  bool          // Automatically recover from panics
	AutoContextTimeout time.Duration // Default context timeout

	// Request/Response options
	MaxRequestSize    int64  // Maximum request body size
	ResponseFormat    string // "json", "xml", "text"
	EnableCompression bool   // Enable response compression
	EnableCORS        bool   // Enable CORS headers

	// Error handling options
	DetailedErrors   bool           // Include stack traces in errors
	CustomErrorCodes map[string]int // Custom HTTP status codes for errors
	ErrorLogging     bool           // Log errors automatically

	// Performance options
	EnableMetrics      bool // Collect middleware metrics
	EnableTracing      bool // Enable distributed tracing
	PoolRequestParsers bool // Pool request parsers for performance

	// Security options
	ValidateContentType bool // Validate request content type
	RequireAPIKey       bool // Require API key authentication
	RateLimitEnabled    bool // Enable rate limiting

	// Framework-specific options
	FrameworkIntegration map[string]interface{} // Framework-specific settings
}

// MiddlewareMetrics collects middleware performance metrics
type MiddlewareMetrics struct {
	TotalRequests    int64
	SuccessRequests  int64
	ErrorRequests    int64
	AverageLatency   time.Duration
	MaxLatency       time.Duration
	MinLatency       time.Duration
	ParsingErrors    int64
	ValidationErrors int64
	mu               sync.RWMutex
}

// ErrorHandler defines custom error handling function
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error, context *ErrorContext)

// ErrorContext provides error handling context
type ErrorContext struct {
	ErrorType   string // "parse", "validate", "send", "internal"
	OriginalURL string
	Method      string
	UserAgent   string
	RequestID   string
	Timestamp   time.Time
	StackTrace  string
	Suggestions []string
}

// RequestInterceptor allows custom request processing
type RequestInterceptor func(r *http.Request) (*http.Request, error)

// ResponseInterceptor allows custom response processing
type ResponseInterceptor func(w http.ResponseWriter, response interface{}) error

// ================================
// Middleware Engine Creation
// ================================

// NewMiddlewareEngine creates a new optimized middleware engine
func NewMiddlewareEngine(hub *Hub, framework string) *MiddlewareEngine {
	return &MiddlewareEngine{
		hub:           hub,
		framework:     strings.ToLower(framework),
		config:        DefaultMiddlewareConfig(),
		metrics:       &MiddlewareMetrics{MinLatency: time.Hour}, // Initialize with high value
		errorHandlers: make(map[string]ErrorHandler),
		interceptors:  []RequestInterceptor{},
	}
}

// DefaultMiddlewareConfig returns sensible middleware defaults
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		AutoParseRequest:     true,
		AutoHandleResponse:   true,
		AutoErrorRecovery:    true,
		AutoContextTimeout:   30 * time.Second,
		MaxRequestSize:       1 << 20, // 1MB
		ResponseFormat:       "json",
		EnableCompression:    true,
		EnableCORS:           true,
		DetailedErrors:       false,
		CustomErrorCodes:     make(map[string]int),
		ErrorLogging:         true,
		EnableMetrics:        true,
		EnableTracing:        false,
		PoolRequestParsers:   true,
		ValidateContentType:  true,
		RequireAPIKey:        false,
		RateLimitEnabled:     false,
		FrameworkIntegration: make(map[string]interface{}),
	}
}

// ================================
// Configuration Methods
// ================================

// Configure applies middleware configuration options
func (me *MiddlewareEngine) Configure(options ...MiddlewareOption) *MiddlewareEngine {
	for _, opt := range options {
		opt(me.config)
	}
	return me
}

// MiddlewareOption defines middleware configuration option
type MiddlewareOption func(*MiddlewareConfig)

// WithAutoHandling enables/disables auto request/response handling
func WithAutoHandling(enabled bool) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.AutoParseRequest = enabled
		cfg.AutoHandleResponse = enabled
	}
}

// WithErrorRecovery enables/disables automatic error recovery
func WithErrorRecovery(enabled bool) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.AutoErrorRecovery = enabled
	}
}

// WithDetailedErrors enables/disables detailed error reporting
func WithDetailedErrors(enabled bool) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.DetailedErrors = enabled
	}
}

// WithMiddlewareMetrics enables/disables metrics collection
func WithMiddlewareMetrics(enabled bool) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.EnableMetrics = enabled
	}
}

// WithSecurity enables security features
func WithSecurity(apiKey, rateLimit bool) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.RequireAPIKey = apiKey
		cfg.RateLimitEnabled = rateLimit
	}
}

// WithMaxRequestSize sets maximum request size
func WithMaxRequestSize(size int64) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.MaxRequestSize = size
	}
}

// WithTimeout sets default context timeout
func WithTimeout(timeout time.Duration) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.AutoContextTimeout = timeout
	}
}

// ================================
// Framework-Agnostic Middleware
// ================================

// Middleware returns a framework-agnostic middleware function
func (me *MiddlewareEngine) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			me.handleRequest(w, r, next)
		})
	}
}

// MiddlewareFunc returns a middleware function that can be used directly
func (me *MiddlewareEngine) MiddlewareFunc() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		me.handleRequest(w, r, nil)
	})
}

// handleRequest is the core middleware request handler
func (me *MiddlewareEngine) handleRequest(w http.ResponseWriter, r *http.Request, next http.Handler) {
	startTime := time.Now()
	requestID := generateRequestID()

	// Update metrics
	if me.config.EnableMetrics {
		me.updateMetrics(func(m *MiddlewareMetrics) {
			m.TotalRequests++
		})
	}

	// Set request ID in context
	ctx := context.WithValue(r.Context(), "request_id", requestID)
	r = r.WithContext(ctx)

	// Apply context timeout if configured
	if me.config.AutoContextTimeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, me.config.AutoContextTimeout)
		defer cancel()
		r = r.WithContext(timeoutCtx)
	}

	// Recovery wrapper
	if me.config.AutoErrorRecovery {
		defer me.recoverFromPanic(w, r)
	}

	// Apply request interceptors
	var err error
	for _, interceptor := range me.interceptors {
		r, err = interceptor(r)
		if err != nil {
			me.handleError(w, r, err, "interceptor")
			return
		}
	}

	// Validate request size
	if me.config.MaxRequestSize > 0 && r.ContentLength > me.config.MaxRequestSize {
		me.handleError(w, r, fmt.Errorf("request body too large: %d bytes", r.ContentLength), "validation")
		return
	}

	// Validate content type for POST/PUT requests
	if me.config.ValidateContentType && (r.Method == "POST" || r.Method == "PUT") {
		if !me.isValidContentType(r) {
			me.handleError(w, r, fmt.Errorf("invalid content type: %s", r.Header.Get("Content-Type")), "validation")
			return
		}
	}

	// Set common response headers
	me.setResponseHeaders(w, r)

	// If this is a CORS preflight request, handle it
	if r.Method == "OPTIONS" && me.config.EnableCORS {
		w.WriteHeader(http.StatusNoContent)
		me.recordLatency(startTime)
		return
	}

	// Auto-parse and handle request if enabled
	if me.config.AutoParseRequest && me.isNotifyHubRoute(r) {
		me.handleNotifyHubRequest(w, r)
	} else if next != nil {
		// Pass to next handler in chain
		next.ServeHTTP(w, r)
	}

	me.recordLatency(startTime)
}

// ================================
// Auto Request/Response Handling
// ================================

// handleNotifyHubRequest automatically handles NotifyHub-specific requests
func (me *MiddlewareEngine) handleNotifyHubRequest(w http.ResponseWriter, r *http.Request) {
	// Parse request
	httpReq, err := ParseHTTPRequest(r)
	if err != nil {
		me.handleError(w, r, err, "parse")
		return
	}

	// Parse options
	httpOptions, err := ParseHTTPOptions(r)
	if err != nil {
		me.handleError(w, r, err, "parse")
		return
	}

	// Convert to NotifyHub types
	message, err := ConvertHTTPToMessage(httpReq)
	if err != nil {
		me.handleError(w, r, err, "validation")
		return
	}

	options, err := ConvertHTTPToOptions(httpOptions)
	if err != nil {
		me.handleError(w, r, err, "validation")
		return
	}

	// Send message
	ctx := r.Context()
	if options != nil && options.Async {
		taskID, err := me.hub.SendAsync(ctx, message, options)
		if err != nil {
			me.handleError(w, r, err, "send")
			return
		}

		response := CreateAsyncSuccessResponse(taskID)
		me.writeResponse(w, http.StatusAccepted, response)
	} else {
		results, err := me.hub.Send(ctx, message, options)
		if err != nil {
			me.handleError(w, r, err, "send")
			return
		}

		response := CreateSuccessResponse("Message sent successfully", map[string]interface{}{
			"message_id": message.ID,
			"targets":    len(message.Targets),
			"results":    len(results),
		})
		me.writeResponse(w, http.StatusOK, response)
	}

	// Update success metrics
	if me.config.EnableMetrics {
		me.updateMetrics(func(m *MiddlewareMetrics) {
			m.SuccessRequests++
		})
	}
}

// ================================
// Enhanced Error Management
// ================================

// handleError provides enhanced error handling with context
func (me *MiddlewareEngine) handleError(w http.ResponseWriter, r *http.Request, err error, errorType string) {
	if me.config.EnableMetrics {
		me.updateMetrics(func(m *MiddlewareMetrics) {
			m.ErrorRequests++
			switch errorType {
			case "parse":
				m.ParsingErrors++
			case "validation":
				m.ValidationErrors++
			}
		})
	}

	// Create error context
	errorCtx := &ErrorContext{
		ErrorType:   errorType,
		OriginalURL: r.URL.String(),
		Method:      r.Method,
		UserAgent:   r.Header.Get("User-Agent"),
		RequestID:   me.getRequestID(r),
		Timestamp:   time.Now(),
		Suggestions: me.generateErrorSuggestions(err, errorType),
	}

	if me.config.DetailedErrors {
		errorCtx.StackTrace = me.getStackTrace()
	}

	// Check for custom error handler
	if handler, exists := me.errorHandlers[errorType]; exists {
		handler(w, r, err, errorCtx)
		return
	}

	// Default error handling
	me.defaultErrorHandler(w, r, err, errorCtx)
}

// defaultErrorHandler provides default error handling
func (me *MiddlewareEngine) defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error, ctx *ErrorContext) {
	// Determine status code
	statusCode := me.getStatusCodeForError(err, ctx.ErrorType)

	// Create error response
	errorResponse := &HTTPResponse{
		Success: false,
		Message: me.getErrorMessage(err, ctx.ErrorType),
		Errors:  []string{err.Error()},
	}

	// Add suggestions if available
	if len(ctx.Suggestions) > 0 {
		errorResponse.Data = map[string]interface{}{
			"suggestions": ctx.Suggestions,
			"request_id":  ctx.RequestID,
		}
	}

	// Add debug information if enabled
	if me.config.DetailedErrors {
		if errorResponse.Data == nil {
			errorResponse.Data = make(map[string]interface{})
		}
		debugInfo := errorResponse.Data.(map[string]interface{})
		debugInfo["error_type"] = ctx.ErrorType
		debugInfo["timestamp"] = ctx.Timestamp
		debugInfo["url"] = ctx.OriginalURL
		if ctx.StackTrace != "" {
			debugInfo["stack_trace"] = ctx.StackTrace
		}
	}

	// Log error if enabled
	if me.config.ErrorLogging {
		me.logError(err, ctx)
	}

	me.writeResponse(w, statusCode, errorResponse)
}

// recoverFromPanic recovers from panics and converts them to errors
func (me *MiddlewareEngine) recoverFromPanic(w http.ResponseWriter, r *http.Request) {
	if recovered := recover(); recovered != nil {
		var err error
		switch v := recovered.(type) {
		case error:
			err = v
		case string:
			err = fmt.Errorf("panic: %s", v)
		default:
			err = fmt.Errorf("panic: %v", v)
		}

		me.handleError(w, r, err, "internal")
	}
}

// ================================
// Performance Monitoring
// ================================

// recordLatency records request latency for metrics
func (me *MiddlewareEngine) recordLatency(startTime time.Time) {
	if !me.config.EnableMetrics {
		return
	}

	latency := time.Since(startTime)
	me.updateMetrics(func(m *MiddlewareMetrics) {
		// Update average latency (simple moving average)
		if m.TotalRequests > 0 {
			m.AverageLatency = time.Duration((int64(m.AverageLatency)*m.TotalRequests + int64(latency)) / (m.TotalRequests + 1))
		} else {
			m.AverageLatency = latency
		}

		// Update min/max latency
		if latency > m.MaxLatency {
			m.MaxLatency = latency
		}
		if latency < m.MinLatency {
			m.MinLatency = latency
		}
	})
}

// updateMetrics safely updates metrics
func (me *MiddlewareEngine) updateMetrics(updateFn func(*MiddlewareMetrics)) {
	me.metrics.mu.Lock()
	defer me.metrics.mu.Unlock()
	updateFn(me.metrics)
}

// GetMetrics returns current middleware metrics
func (me *MiddlewareEngine) GetMetrics() *MiddlewareMetrics {
	me.metrics.mu.RLock()
	defer me.metrics.mu.RUnlock()

	// Return a copy to avoid data races
	return &MiddlewareMetrics{
		TotalRequests:    me.metrics.TotalRequests,
		SuccessRequests:  me.metrics.SuccessRequests,
		ErrorRequests:    me.metrics.ErrorRequests,
		AverageLatency:   me.metrics.AverageLatency,
		MaxLatency:       me.metrics.MaxLatency,
		MinLatency:       me.metrics.MinLatency,
		ParsingErrors:    me.metrics.ParsingErrors,
		ValidationErrors: me.metrics.ValidationErrors,
	}
}

// ================================
// Custom Error Handlers
// ================================

// SetErrorHandler sets a custom error handler for a specific error type
func (me *MiddlewareEngine) SetErrorHandler(errorType string, handler ErrorHandler) {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.errorHandlers[errorType] = handler
}

// AddRequestInterceptor adds a request interceptor
func (me *MiddlewareEngine) AddRequestInterceptor(interceptor RequestInterceptor) {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.interceptors = append(me.interceptors, interceptor)
}

// ================================
// Framework-Specific Adapters
// ================================

// ForGin returns a Gin-compatible middleware function
func (me *MiddlewareEngine) ForGin() interface{} {
	return func(c interface{}) {
		// In a real implementation, this would extract http.ResponseWriter and *http.Request from Gin context
		// For now, we return a placeholder that demonstrates the concept
		fmt.Println("Gin middleware adapter - not fully implemented")
	}
}

// ForEcho returns an Echo-compatible middleware function
func (me *MiddlewareEngine) ForEcho() interface{} {
	return func(next interface{}) interface{} {
		// In a real implementation, this would adapt to Echo's middleware pattern
		fmt.Println("Echo middleware adapter - not fully implemented")
		return next
	}
}

// ForChi returns a Chi-compatible middleware function
func (me *MiddlewareEngine) ForChi() func(http.Handler) http.Handler {
	return me.Middleware() // Chi uses standard http.Handler pattern
}

// ForNetHTTP returns a standard net/http middleware function
func (me *MiddlewareEngine) ForNetHTTP() func(http.Handler) http.Handler {
	return me.Middleware()
}

// ================================
// Utility Methods
// ================================

// isNotifyHubRoute checks if the request is for a NotifyHub route
func (me *MiddlewareEngine) isNotifyHubRoute(r *http.Request) bool {
	path := r.URL.Path
	return strings.Contains(path, "/notify") ||
		strings.HasSuffix(path, "/send") ||
		strings.HasSuffix(path, "/batch") ||
		strings.HasSuffix(path, "/alert") ||
		strings.HasSuffix(path, "/text") ||
		strings.HasSuffix(path, "/template")
}

// isValidContentType validates the request content type
func (me *MiddlewareEngine) isValidContentType(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	validTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	}

	for _, validType := range validTypes {
		if strings.Contains(contentType, validType) {
			return true
		}
	}
	return false
}

// setResponseHeaders sets common response headers
func (me *MiddlewareEngine) setResponseHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Powered-By", "NotifyHub")
	w.Header().Set("X-Request-ID", me.getRequestID(r))

	if me.config.EnableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	}

	if me.config.EnableCompression {
		w.Header().Set("Content-Encoding", "gzip")
	}
}

// writeResponse writes a response with proper formatting
func (me *MiddlewareEngine) writeResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		// Fallback error response
		fmt.Fprintf(w, `{"success":false,"message":"Failed to encode response"}`)
	}
}

// generateErrorSuggestions generates helpful error suggestions
func (me *MiddlewareEngine) generateErrorSuggestions(err error, errorType string) []string {
	suggestions := []string{}
	errMsg := strings.ToLower(err.Error())

	switch errorType {
	case "parse":
		if strings.Contains(errMsg, "json") {
			suggestions = append(suggestions, "Check that the request body contains valid JSON")
			suggestions = append(suggestions, "Verify Content-Type header is set to 'application/json'")
		}
		if strings.Contains(errMsg, "body") {
			suggestions = append(suggestions, "Ensure the request body is not empty")
		}
	case "validation":
		if strings.Contains(errMsg, "required") {
			suggestions = append(suggestions, "Check that all required fields are provided")
		}
		if strings.Contains(errMsg, "email") {
			suggestions = append(suggestions, "Verify email address format")
		}
		if strings.Contains(errMsg, "target") {
			suggestions = append(suggestions, "Ensure targets array is not empty")
			suggestions = append(suggestions, "Check target type and value formatting")
		}
	case "send":
		suggestions = append(suggestions, "Check NotifyHub configuration")
		suggestions = append(suggestions, "Verify network connectivity")
		suggestions = append(suggestions, "Check platform-specific settings")
	}

	return suggestions
}

// getStatusCodeForError returns appropriate HTTP status code for error
func (me *MiddlewareEngine) getStatusCodeForError(err error, errorType string) int {
	// Check custom error codes first
	if code, exists := me.config.CustomErrorCodes[errorType]; exists {
		return code
	}

	switch errorType {
	case "parse":
		return http.StatusBadRequest
	case "validation":
		return http.StatusBadRequest
	case "send":
		return http.StatusInternalServerError
	case "interceptor":
		return http.StatusBadRequest
	case "internal":
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// getErrorMessage returns user-friendly error message
func (me *MiddlewareEngine) getErrorMessage(err error, errorType string) string {
	switch errorType {
	case "parse":
		return "Failed to parse request"
	case "validation":
		return "Request validation failed"
	case "send":
		return "Failed to send notification"
	case "interceptor":
		return "Request processing failed"
	case "internal":
		return "Internal server error"
	default:
		return "An error occurred"
	}
}

// getRequestID extracts request ID from context
func (me *MiddlewareEngine) getRequestID(r *http.Request) string {
	if id := r.Context().Value("request_id"); id != nil {
		return id.(string)
	}
	return "unknown"
}

// getStackTrace captures current stack trace
func (me *MiddlewareEngine) getStackTrace() string {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

// logError logs error information
func (me *MiddlewareEngine) logError(err error, ctx *ErrorContext) {
	// In a real implementation, this would use the configured logger
	fmt.Printf("[ERROR] %s - %s: %v (Request ID: %s)\n",
		ctx.Timestamp.Format(time.RFC3339),
		ctx.ErrorType,
		err,
		ctx.RequestID)
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// ================================
// Convenience Functions
// ================================

// QuickMiddleware creates a middleware engine with sensible defaults
func QuickMiddleware(hub *Hub) *MiddlewareEngine {
	return NewMiddlewareEngine(hub, "auto").
		Configure(
			WithAutoHandling(true),
			WithErrorRecovery(true),
			WithMiddlewareMetrics(true),
		)
}

// ProductionMiddleware creates a production-ready middleware engine
func ProductionMiddleware(hub *Hub) *MiddlewareEngine {
	return NewMiddlewareEngine(hub, "auto").
		Configure(
			WithAutoHandling(true),
			WithErrorRecovery(true),
			WithDetailedErrors(false), // Don't expose internals in production
			WithMiddlewareMetrics(true),
			WithSecurity(false, true), // Enable rate limiting but not API key by default
			WithTimeout(30*time.Second),
		)
}

// DevelopmentMiddleware creates a development-friendly middleware engine
func DevelopmentMiddleware(hub *Hub) *MiddlewareEngine {
	return NewMiddlewareEngine(hub, "auto").
		Configure(
			WithAutoHandling(true),
			WithErrorRecovery(true),
			WithDetailedErrors(true), // Include debug info in development
			WithMiddlewareMetrics(true),
			WithTimeout(60*time.Second), // Longer timeout for debugging
		)
}

// TestMiddleware creates a test-friendly middleware engine
func TestMiddleware(hub *Hub) *MiddlewareEngine {
	return NewMiddlewareEngine(hub, "auto").
		Configure(
			WithAutoHandling(true),
			WithErrorRecovery(false), // Let tests see panics
			WithDetailedErrors(true),
			WithMiddlewareMetrics(false),         // Don't collect metrics in tests
			WithTimeout(5*time.Second), // Fast timeouts for tests
		)
}
