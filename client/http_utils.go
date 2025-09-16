package client

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/notifiers"
)

// HTTPMessageRequest represents an incoming HTTP request for sending messages
type HTTPMessageRequest struct {
	Title     string                 `json:"title" validate:"required,min=1,max=200"`
	Body      string                 `json:"body" validate:"required,min=1,max=5000"`
	Priority  int                    `json:"priority" validate:"min=1,max=5"`
	Targets   []HTTPTarget           `json:"targets" validate:"required,dive"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Template  string                 `json:"template,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// HTTPTarget represents a notification target from HTTP request
type HTTPTarget struct {
	Type     string `json:"type" validate:"required,oneof=email user group channel"`
	Value    string `json:"value" validate:"required,min=1,max=255"`
	Platform string `json:"platform,omitempty"`
}

// HTTPSendOptions represents sending options from HTTP request
type HTTPSendOptions struct {
	Async     bool `json:"async,omitempty"`
	Timeout   int  `json:"timeout,omitempty"` // in seconds
	Retry     bool `json:"retry,omitempty"`
	MaxRetry  int  `json:"max_retry,omitempty"`
	DelayMs   int  `json:"delay_ms,omitempty"`
	BatchSize int  `json:"batch_size,omitempty"`
}

// HTTPResponse represents the standard HTTP API response
type HTTPResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	TaskID  string      `json:"task_id,omitempty"`
	Errors  []string    `json:"errors,omitempty"`
}

// ConvertHTTPToMessage converts HTTP request to NotifyHub message
func ConvertHTTPToMessage(req *HTTPMessageRequest) (*notifiers.Message, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	// Validate required fields
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title is required and cannot be empty")
	}
	if strings.TrimSpace(req.Body) == "" {
		return nil, fmt.Errorf("body is required and cannot be empty")
	}
	if len(req.Targets) == 0 {
		return nil, fmt.Errorf("at least one target is required")
	}

	// Validate title and body length
	if len(req.Title) > 200 {
		return nil, fmt.Errorf("title exceeds maximum length of 200 characters")
	}
	if len(req.Body) > 5000 {
		return nil, fmt.Errorf("body exceeds maximum length of 5000 characters")
	}

	// Validate priority range
	priority := req.Priority
	if priority < 1 {
		priority = 3 // default priority
	}
	if priority > 5 {
		return nil, fmt.Errorf("priority must be between 1 and 5")
	}

	// Convert targets
	targets := make([]notifiers.Target, 0, len(req.Targets))
	for i, httpTarget := range req.Targets {
		target, err := convertHTTPTarget(httpTarget, i)
		if err != nil {
			return nil, fmt.Errorf("invalid target at index %d: %v", i, err)
		}
		targets = append(targets, target)
	}

	// Create message
	message := &notifiers.Message{
		ID:        generateMessageID(),
		Title:     strings.TrimSpace(req.Title),
		Body:      strings.TrimSpace(req.Body),
		Priority:  priority,
		Targets:   targets,
		Variables: req.Variables,
		Template:  req.Template,
		Metadata:  req.Metadata,
	}

	return message, nil
}

// ConvertHTTPToOptions converts HTTP options to NotifyHub options
func ConvertHTTPToOptions(req *HTTPSendOptions) (*Options, error) {
	if req == nil {
		return nil, nil
	}

	options := &Options{
		Async: req.Async,
		Retry: req.Retry,
	}

	// Convert timeout from seconds to duration
	if req.Timeout > 0 {
		if req.Timeout > 300 { // max 5 minutes
			return nil, fmt.Errorf("timeout exceeds maximum of 300 seconds")
		}
		options.Timeout = time.Duration(req.Timeout) * time.Second
	}

	// Validate retry settings
	if req.MaxRetry > 0 {
		if req.MaxRetry > 10 {
			return nil, fmt.Errorf("max_retry exceeds maximum of 10")
		}
		options.MaxRetry = req.MaxRetry
	}

	// Convert delay from milliseconds to duration
	if req.DelayMs > 0 {
		if req.DelayMs > 60000 { // max 1 minute
			return nil, fmt.Errorf("delay_ms exceeds maximum of 60000ms")
		}
		options.DelayMs = req.DelayMs
	}

	// Validate batch size
	if req.BatchSize > 0 {
		if req.BatchSize > 1000 {
			return nil, fmt.Errorf("batch_size exceeds maximum of 1000")
		}
		options.BatchSize = req.BatchSize
	}

	return options, nil
}

// convertHTTPTarget converts a single HTTP target to NotifyHub target
func convertHTTPTarget(httpTarget HTTPTarget, index int) (notifiers.Target, error) {
	// Validate type
	validTypes := []string{"email", "user", "group", "channel"}
	typeValid := false
	for _, validType := range validTypes {
		if httpTarget.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return notifiers.Target{}, fmt.Errorf("invalid target type '%s', must be one of: %s",
			httpTarget.Type, strings.Join(validTypes, ", "))
	}

	// Validate value
	value := strings.TrimSpace(httpTarget.Value)
	if value == "" {
		return notifiers.Target{}, fmt.Errorf("target value cannot be empty")
	}
	if len(value) > 255 {
		return notifiers.Target{}, fmt.Errorf("target value exceeds maximum length of 255 characters")
	}

	// Additional validation based on type
	switch httpTarget.Type {
	case "email":
		if !isValidEmail(value) {
			return notifiers.Target{}, fmt.Errorf("invalid email format: %s", value)
		}
	case "user", "group", "channel":
		if len(value) < 1 {
			return notifiers.Target{}, fmt.Errorf("%s identifier cannot be empty", httpTarget.Type)
		}
	}

	return notifiers.Target{
		Type:     notifiers.TargetType(httpTarget.Type),
		Value:    value,
		Platform: httpTarget.Platform,
	}, nil
}

// CreateSuccessResponse creates a standard success HTTP response
func CreateSuccessResponse(message string, data interface{}) *HTTPResponse {
	return &HTTPResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// CreateAsyncSuccessResponse creates a success response for async operations
func CreateAsyncSuccessResponse(taskID string) *HTTPResponse {
	return &HTTPResponse{
		Success: true,
		Message: "Message queued successfully",
		TaskID:  taskID,
	}
}

// CreateErrorResponse creates a standard error HTTP response
func CreateErrorResponse(message string, errors ...string) *HTTPResponse {
	response := &HTTPResponse{
		Success: false,
		Message: message,
	}
	if len(errors) > 0 {
		response.Errors = errors
	}
	return response
}

// CreateValidationErrorResponse creates a validation error response
func CreateValidationErrorResponse(errors []string) *HTTPResponse {
	return &HTTPResponse{
		Success: false,
		Message: "Validation failed",
		Errors:  errors,
	}
}

// ParseHTTPRequest parses JSON request body to HTTPMessageRequest
func ParseHTTPRequest(r *http.Request) (*HTTPMessageRequest, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()

	var req HTTPMessageRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Reject unknown fields

	if err := decoder.Decode(&req); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %v", err)
	}

	return &req, nil
}

// ParseHTTPOptions parses HTTP headers and query parameters to options
func ParseHTTPOptions(r *http.Request) (*HTTPSendOptions, error) {
	options := &HTTPSendOptions{}

	// Parse query parameters
	query := r.URL.Query()

	// Parse async flag
	if asyncStr := query.Get("async"); asyncStr != "" {
		async, err := strconv.ParseBool(asyncStr)
		if err != nil {
			return nil, fmt.Errorf("invalid async parameter: %v", err)
		}
		options.Async = async
	}

	// Parse timeout
	if timeoutStr := query.Get("timeout"); timeoutStr != "" {
		timeout, err := strconv.Atoi(timeoutStr)
		if err != nil || timeout < 0 {
			return nil, fmt.Errorf("invalid timeout parameter: must be a positive integer")
		}
		options.Timeout = timeout
	}

	// Parse retry flag
	if retryStr := query.Get("retry"); retryStr != "" {
		retry, err := strconv.ParseBool(retryStr)
		if err != nil {
			return nil, fmt.Errorf("invalid retry parameter: %v", err)
		}
		options.Retry = retry
	}

	// Parse max retry
	if maxRetryStr := query.Get("max_retry"); maxRetryStr != "" {
		maxRetry, err := strconv.Atoi(maxRetryStr)
		if err != nil || maxRetry < 0 {
			return nil, fmt.Errorf("invalid max_retry parameter: must be a positive integer")
		}
		options.MaxRetry = maxRetry
	}

	// Parse delay
	if delayStr := query.Get("delay_ms"); delayStr != "" {
		delay, err := strconv.Atoi(delayStr)
		if err != nil || delay < 0 {
			return nil, fmt.Errorf("invalid delay_ms parameter: must be a positive integer")
		}
		options.DelayMs = delay
	}

	// Parse batch size
	if batchStr := query.Get("batch_size"); batchStr != "" {
		batchSize, err := strconv.Atoi(batchStr)
		if err != nil || batchSize < 1 {
			return nil, fmt.Errorf("invalid batch_size parameter: must be a positive integer")
		}
		options.BatchSize = batchSize
	}

	return options, nil
}

// WriteJSONResponse writes a JSON response to HTTP response writer
func WriteJSONResponse(w http.ResponseWriter, statusCode int, response *HTTPResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to encode JSON response: %v", err)
	}

	return nil
}

// Helper functions

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) == 0 || len(email) > 254 {
		return false
	}

	// Basic email format validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	local, domain := parts[0], parts[1]
	if len(local) == 0 || len(local) > 64 || len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// Check for at least one dot in domain
	if !strings.Contains(domain, ".") {
		return false
	}

	return true
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

// ================================
// HTTP Middleware and Handler Wrappers
// ================================

// HTTPHandler represents a NotifyHub HTTP handler function
type HTTPHandler func(w http.ResponseWriter, r *http.Request, hub *Hub)

// NotifyHubMiddleware wraps HTTP handlers with NotifyHub functionality
func NotifyHubMiddleware(hub *Hub, handler HTTPHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set common headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Powered-By", "NotifyHub")

		// Add request timeout if not present
		if r.Context().Err() == nil {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			r = r.WithContext(ctx)
		}

		// Call the handler
		handler(w, r, hub)
	}
}

// ParseAndSendMiddleware handles common request parsing and sending logic
func ParseAndSendMiddleware(hub *Hub, handler func(w http.ResponseWriter, r *http.Request, message *notifiers.Message, options *Options)) http.HandlerFunc {
	return NotifyHubMiddleware(hub, func(w http.ResponseWriter, r *http.Request, h *Hub) {
		// Parse HTTP request
		httpReq, err := ParseHTTPRequest(r)
		if err != nil {
			response := CreateErrorResponse("Invalid request format", err.Error())
			WriteJSONResponse(w, http.StatusBadRequest, response)
			return
		}

		// Parse HTTP options
		httpOptions, err := ParseHTTPOptions(r)
		if err != nil {
			response := CreateErrorResponse("Invalid request options", err.Error())
			WriteJSONResponse(w, http.StatusBadRequest, response)
			return
		}

		// Convert to NotifyHub types
		message, err := ConvertHTTPToMessage(httpReq)
		if err != nil {
			response := CreateValidationErrorResponse([]string{err.Error()})
			WriteJSONResponse(w, http.StatusBadRequest, response)
			return
		}

		options, err := ConvertHTTPToOptions(httpOptions)
		if err != nil {
			response := CreateErrorResponse("Invalid options", err.Error())
			WriteJSONResponse(w, http.StatusBadRequest, response)
			return
		}

		// Call the actual handler
		handler(w, r, message, options)
	})
}

// ================================
// Quick Handler Builders
// ================================

// QuickSendHandler creates a simple send message handler
func QuickSendHandler(hub *Hub) http.HandlerFunc {
	return ParseAndSendMiddleware(hub, func(w http.ResponseWriter, r *http.Request, message *notifiers.Message, options *Options) {
		ctx := r.Context()

		if options != nil && options.Async {
			// Async sending
			taskID, err := hub.SendAsync(ctx, message, options)
			if err != nil {
				response := CreateErrorResponse("Failed to enqueue message", err.Error())
				WriteJSONResponse(w, http.StatusInternalServerError, response)
				return
			}

			response := CreateAsyncSuccessResponse(taskID)
			WriteJSONResponse(w, http.StatusAccepted, response)
		} else {
			// Sync sending
			results, err := hub.Send(ctx, message, options)
			if err != nil {
				response := CreateErrorResponse("Failed to send message", err.Error())
				WriteJSONResponse(w, http.StatusInternalServerError, response)
				return
			}

			response := CreateSuccessResponse("Message sent successfully", map[string]interface{}{
				"message_id": message.ID,
				"targets":    len(message.Targets),
				"results":    len(results),
			})
			WriteJSONResponse(w, http.StatusOK, response)
		}
	})
}

// QuickBatchHandler creates a batch send message handler
func QuickBatchHandler(hub *Hub) http.HandlerFunc {
	return NotifyHubMiddleware(hub, func(w http.ResponseWriter, r *http.Request, h *Hub) {
		var batchReq struct {
			Messages []HTTPMessageRequest `json:"messages" validate:"required,dive"`
			Options  *HTTPSendOptions     `json:"options,omitempty"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&batchReq); err != nil {
			response := CreateErrorResponse("Invalid JSON format", err.Error())
			WriteJSONResponse(w, http.StatusBadRequest, response)
			return
		}

		// Convert all messages
		messages := make([]*notifiers.Message, 0, len(batchReq.Messages))
		for i, httpMsg := range batchReq.Messages {
			message, err := ConvertHTTPToMessage(&httpMsg)
			if err != nil {
				response := CreateErrorResponse(fmt.Sprintf("Invalid message at index %d", i), err.Error())
				WriteJSONResponse(w, http.StatusBadRequest, response)
				return
			}
			messages = append(messages, message)
		}

		// Convert options
		options, err := ConvertHTTPToOptions(batchReq.Options)
		if err != nil {
			response := CreateErrorResponse("Invalid options", err.Error())
			WriteJSONResponse(w, http.StatusBadRequest, response)
			return
		}

		// Send batch
		ctx := r.Context()
		results, err := h.SendBatch(ctx, messages, options)
		if err != nil {
			response := CreateErrorResponse("Failed to send batch", err.Error())
			WriteJSONResponse(w, http.StatusInternalServerError, response)
			return
		}

		response := CreateSuccessResponse("Batch sent successfully", map[string]interface{}{
			"batch_size": len(messages),
			"results":    len(results),
		})
		WriteJSONResponse(w, http.StatusOK, response)
	})
}

// ================================
// Convenience HTTP Handlers for Common Scenarios
// ================================

// QuickTextHandler creates a handler for simple text messages
func QuickTextHandler(hub *Hub) http.HandlerFunc {
	return NotifyHubMiddleware(hub, func(w http.ResponseWriter, r *http.Request, h *Hub) {
		var req struct {
			Title   string       `json:"title" validate:"required"`
			Body    string       `json:"body" validate:"required"`
			Targets []HTTPTarget `json:"targets" validate:"required,dive"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response := CreateErrorResponse("Invalid JSON format", err.Error())
			WriteJSONResponse(w, http.StatusBadRequest, response)
			return
		}

		// Convert targets
		targets := make([]notifiers.Target, 0, len(req.Targets))
		for _, httpTarget := range req.Targets {
			target, err := convertHTTPTarget(httpTarget, 0)
			if err != nil {
				response := CreateErrorResponse("Invalid target", err.Error())
				WriteJSONResponse(w, http.StatusBadRequest, response)
				return
			}
			targets = append(targets, target)
		}

		// Send text message
		ctx := r.Context()
		err := h.SendText(ctx, req.Title, req.Body, targets...)
		if err != nil {
			response := CreateErrorResponse("Failed to send text message", err.Error())
			WriteJSONResponse(w, http.StatusInternalServerError, response)
			return
		}

		response := CreateSuccessResponse("Text message sent", nil)
		WriteJSONResponse(w, http.StatusOK, response)
	})
}

// QuickAlertHandler creates a handler for alert messages
func QuickAlertHandler(hub *Hub) http.HandlerFunc {
	return NotifyHubMiddleware(hub, func(w http.ResponseWriter, r *http.Request, h *Hub) {
		var req struct {
			Title   string       `json:"title" validate:"required"`
			Body    string       `json:"body" validate:"required"`
			Targets []HTTPTarget `json:"targets" validate:"required,dive"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response := CreateErrorResponse("Invalid JSON format", err.Error())
			WriteJSONResponse(w, http.StatusBadRequest, response)
			return
		}

		// Convert targets
		targets := make([]notifiers.Target, 0, len(req.Targets))
		for _, httpTarget := range req.Targets {
			target, err := convertHTTPTarget(httpTarget, 0)
			if err != nil {
				response := CreateErrorResponse("Invalid target", err.Error())
				WriteJSONResponse(w, http.StatusBadRequest, response)
				return
			}
			targets = append(targets, target)
		}

		// Send alert message
		ctx := r.Context()
		err := h.SendAlert(ctx, req.Title, req.Body, targets...)
		if err != nil {
			response := CreateErrorResponse("Failed to send alert", err.Error())
			WriteJSONResponse(w, http.StatusInternalServerError, response)
			return
		}

		response := CreateSuccessResponse("Alert sent", nil)
		WriteJSONResponse(w, http.StatusOK, response)
	})
}

// QuickTemplateHandler creates a handler for template-based messages
func QuickTemplateHandler(hub *Hub) http.HandlerFunc {
	return NotifyHubMiddleware(hub, func(w http.ResponseWriter, r *http.Request, h *Hub) {
		var req struct {
			Template  string                 `json:"template" validate:"required"`
			Variables map[string]interface{} `json:"variables,omitempty"`
			Targets   []HTTPTarget           `json:"targets" validate:"required,dive"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response := CreateErrorResponse("Invalid JSON format", err.Error())
			WriteJSONResponse(w, http.StatusBadRequest, response)
			return
		}

		// Convert targets
		targets := make([]notifiers.Target, 0, len(req.Targets))
		for _, httpTarget := range req.Targets {
			target, err := convertHTTPTarget(httpTarget, 0)
			if err != nil {
				response := CreateErrorResponse("Invalid target", err.Error())
				WriteJSONResponse(w, http.StatusBadRequest, response)
				return
			}
			targets = append(targets, target)
		}

		// Send template message
		ctx := r.Context()
		err := h.SendWithTemplate(ctx, req.Template, req.Variables, targets...)
		if err != nil {
			response := CreateErrorResponse("Failed to send template message", err.Error())
			WriteJSONResponse(w, http.StatusInternalServerError, response)
			return
		}

		response := CreateSuccessResponse("Template message sent", nil)
		WriteJSONResponse(w, http.StatusOK, response)
	})
}

// ================================
// HTTP Server Utilities
// ================================

// SetupNotifyHubRoutes sets up standard NotifyHub HTTP routes on a mux
func SetupNotifyHubRoutes(mux *http.ServeMux, hub *Hub, basePath string) {
	if basePath == "" {
		basePath = "/notify"
	}

	// Ensure basePath doesn't end with /
	if strings.HasSuffix(basePath, "/") {
		basePath = strings.TrimSuffix(basePath, "/")
	}

	// Standard routes
	mux.HandleFunc(basePath+"/send", QuickSendHandler(hub))
	mux.HandleFunc(basePath+"/batch", QuickBatchHandler(hub))
	mux.HandleFunc(basePath+"/text", QuickTextHandler(hub))
	mux.HandleFunc(basePath+"/alert", QuickAlertHandler(hub))
	mux.HandleFunc(basePath+"/template", QuickTemplateHandler(hub))

	// Health check
	mux.HandleFunc(basePath+"/health", func(w http.ResponseWriter, r *http.Request) {
		health := hub.GetHealth(r.Context())
		WriteJSONResponse(w, http.StatusOK, &HTTPResponse{
			Success: true,
			Data:    health,
		})
	})

	// Metrics
	mux.HandleFunc(basePath+"/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics := hub.GetMetrics()
		WriteJSONResponse(w, http.StatusOK, &HTTPResponse{
			Success: true,
			Data:    metrics,
		})
	})
}

// ================================
// HTTP Server Configuration Types
// ================================

// HTTPServerOptions defines configuration options for QuickHTTPServer
type HTTPServerOptions struct {
	// Basic server configuration
	Addr         string
	BasePath     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Middleware injection points
	GlobalMiddleware []func(http.Handler) http.Handler // Applied to entire server
	RouteMiddleware  []func(http.Handler) http.Handler // Applied only to NotifyHub routes
	CustomHandlers   map[string]http.Handler           // Custom route handlers

	// CORS and security settings
	EnableCORS     bool
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string

	// Additional server settings
	MaxHeaderBytes   int
	EnableKeepAlives bool

	// Custom mux (if provided, will use instead of creating new one)
	Mux *http.ServeMux
}

// HTTPServerOption represents a functional option for configuring HTTP server
type HTTPServerOption func(*HTTPServerOptions)

// WithAddress sets the server address
func WithAddress(addr string) HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		opts.Addr = addr
	}
}

// WithBasePath sets the base path for NotifyHub routes
func WithBasePath(basePath string) HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		opts.BasePath = basePath
	}
}

// WithTimeouts sets server timeout configurations
func WithTimeouts(read, write, idle time.Duration) HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		if read > 0 {
			opts.ReadTimeout = read
		}
		if write > 0 {
			opts.WriteTimeout = write
		}
		if idle > 0 {
			opts.IdleTimeout = idle
		}
	}
}

// WithGlobalMiddleware adds middleware to be applied to all routes
func WithGlobalMiddleware(middleware ...func(http.Handler) http.Handler) HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		opts.GlobalMiddleware = append(opts.GlobalMiddleware, middleware...)
	}
}

// WithRouteMiddleware adds middleware to be applied only to NotifyHub routes
func WithRouteMiddleware(middleware ...func(http.Handler) http.Handler) HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		opts.RouteMiddleware = append(opts.RouteMiddleware, middleware...)
	}
}

// WithCustomHandler adds a custom handler for a specific route
func WithCustomHandler(pattern string, handler http.Handler) HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		if opts.CustomHandlers == nil {
			opts.CustomHandlers = make(map[string]http.Handler)
		}
		opts.CustomHandlers[pattern] = handler
	}
}

// WithCORS enables CORS with specified configuration
func WithCORS(origins, methods, headers []string) HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		opts.EnableCORS = true
		opts.AllowedOrigins = origins
		opts.AllowedMethods = methods
		opts.AllowedHeaders = headers
	}
}

// WithDefaultCORS enables CORS with permissive defaults
func WithDefaultCORS() HTTPServerOption {
	return WithCORS(
		[]string{"*"},
		[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		[]string{"Content-Type", "Authorization", "X-Requested-With"},
	)
}

// WithCustomMux provides a custom ServeMux to use instead of creating a new one
func WithCustomMux(mux *http.ServeMux) HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		opts.Mux = mux
	}
}

// WithServerConfig sets additional server configuration
func WithServerConfig(maxHeaderBytes int, enableKeepAlives bool) HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		opts.MaxHeaderBytes = maxHeaderBytes
		opts.EnableKeepAlives = enableKeepAlives
	}
}

// ================================
// Enhanced HTTP Server Builder
// ================================

// applyMiddleware applies a chain of middleware to a handler
func applyMiddleware(handler http.Handler, middleware []func(http.Handler) http.Handler) http.Handler {
	// Apply middleware in reverse order so the first middleware wraps the outermost layer
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

// corsMiddleware creates a CORS middleware
func corsMiddleware(origins, methods, headers []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			if len(origins) > 0 {
				if origins[0] == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					origin := r.Header.Get("Origin")
					for _, allowedOrigin := range origins {
						if origin == allowedOrigin {
							w.Header().Set("Access-Control-Allow-Origin", origin)
							break
						}
					}
				}
			}

			if len(methods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
			}

			if len(headers) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
			}

			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// QuickHTTPServer creates a complete HTTP server with NotifyHub routes
// Deprecated: Use QuickHTTPServerWithOptions for more flexibility
func QuickHTTPServer(hub *Hub, addr string, basePath string) *http.Server {
	return QuickHTTPServerWithOptions(hub,
		WithAddress(addr),
		WithBasePath(basePath),
	)
}

// QuickHTTPServerWithOptions creates a complete HTTP server with enhanced configuration options
func QuickHTTPServerWithOptions(hub *Hub, options ...HTTPServerOption) *http.Server {
	// Default options
	opts := &HTTPServerOptions{
		Addr:             ":8080",
		BasePath:         "/notify",
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     30 * time.Second,
		IdleTimeout:      120 * time.Second,
		MaxHeaderBytes:   1 << 20, // 1 MB
		EnableKeepAlives: true,
	}

	// Apply options
	for _, opt := range options {
		opt(opts)
	}

	// Create or use provided mux
	var mux *http.ServeMux
	if opts.Mux != nil {
		mux = opts.Mux
	} else {
		mux = http.NewServeMux()
	}

	// Setup NotifyHub routes with route-specific middleware
	if len(opts.RouteMiddleware) > 0 {
		// Wrap NotifyHub routes with route middleware
		setupRoutesWithMiddleware(mux, hub, opts.BasePath, opts.RouteMiddleware)
	} else {
		// Setup routes normally
		SetupNotifyHubRoutes(mux, hub, opts.BasePath)
	}

	// Add custom handlers
	for pattern, handler := range opts.CustomHandlers {
		mux.Handle(pattern, handler)
	}

	// Create base handler from mux
	var handler http.Handler = mux

	// Apply CORS middleware if enabled
	if opts.EnableCORS {
		handler = corsMiddleware(opts.AllowedOrigins, opts.AllowedMethods, opts.AllowedHeaders)(handler)
	}

	// Apply global middleware
	if len(opts.GlobalMiddleware) > 0 {
		handler = applyMiddleware(handler, opts.GlobalMiddleware)
	}

	// Create server
	server := &http.Server{
		Addr:           opts.Addr,
		Handler:        handler,
		ReadTimeout:    opts.ReadTimeout,
		WriteTimeout:   opts.WriteTimeout,
		IdleTimeout:    opts.IdleTimeout,
		MaxHeaderBytes: opts.MaxHeaderBytes,
	}

	// Configure keep-alives
	if !opts.EnableKeepAlives {
		server.SetKeepAlivesEnabled(false)
	}

	return server
}

// setupRoutesWithMiddleware sets up NotifyHub routes with middleware applied
func setupRoutesWithMiddleware(mux *http.ServeMux, hub *Hub, basePath string, middleware []func(http.Handler) http.Handler) {
	if basePath == "" {
		basePath = "/notify"
	}

	// Ensure basePath doesn't end with /
	if strings.HasSuffix(basePath, "/") {
		basePath = strings.TrimSuffix(basePath, "/")
	}

	// Create handlers and apply middleware
	routes := map[string]http.HandlerFunc{
		basePath + "/send":     QuickSendHandler(hub),
		basePath + "/batch":    QuickBatchHandler(hub),
		basePath + "/text":     QuickTextHandler(hub),
		basePath + "/alert":    QuickAlertHandler(hub),
		basePath + "/template": QuickTemplateHandler(hub),
		basePath + "/health": func(w http.ResponseWriter, r *http.Request) {
			health := hub.GetHealth(r.Context())
			WriteJSONResponse(w, http.StatusOK, &HTTPResponse{
				Success: true,
				Data:    health,
			})
		},
		basePath + "/metrics": func(w http.ResponseWriter, r *http.Request) {
			metrics := hub.GetMetrics()
			WriteJSONResponse(w, http.StatusOK, &HTTPResponse{
				Success: true,
				Data:    metrics,
			})
		},
	}

	// Apply middleware to each route and register
	for pattern, handler := range routes {
		wrappedHandler := applyMiddleware(handler, middleware)
		mux.Handle(pattern, wrappedHandler)
	}
}

// ================================
// Common Middleware Builders
// ================================

// LoggingMiddleware creates a middleware that logs HTTP requests
func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer that captures status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			fmt.Printf("[%s] %s %s %d %v\n",
				start.Format("2006-01-02 15:04:05"),
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				duration)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// AuthenticationMiddleware creates a middleware that validates API keys
func AuthenticationMiddleware(validAPIKeys map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for API key in header or query parameter
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			if apiKey == "" {
				response := CreateErrorResponse("API key required", "Missing X-API-Key header or api_key parameter")
				WriteJSONResponse(w, http.StatusUnauthorized, response)
				return
			}

			if !validAPIKeys[apiKey] {
				response := CreateErrorResponse("Invalid API key", "The provided API key is not valid")
				WriteJSONResponse(w, http.StatusUnauthorized, response)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware creates a basic rate limiting middleware
func RateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
	type client struct {
		count int
		reset time.Time
	}

	clients := make(map[string]*client)
	mu := sync.RWMutex{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as client identifier
			clientIP := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = strings.Split(forwarded, ",")[0]
			}

			mu.Lock()
			defer mu.Unlock()

			now := time.Now()

			// Get or create client record
			c, exists := clients[clientIP]
			if !exists || now.After(c.reset) {
				c = &client{
					count: 0,
					reset: now.Add(time.Minute),
				}
				clients[clientIP] = c
			}

			// Check rate limit
			if c.count >= requestsPerMinute {
				response := CreateErrorResponse("Rate limit exceeded",
					fmt.Sprintf("Maximum %d requests per minute allowed", requestsPerMinute))
				WriteJSONResponse(w, http.StatusTooManyRequests, response)
				return
			}

			c.count++
			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware creates a middleware that recovers from panics
func RecoveryMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					fmt.Printf("Panic recovered: %v\n", err)
					response := CreateErrorResponse("Internal server error", "An unexpected error occurred")
					WriteJSONResponse(w, http.StatusInternalServerError, response)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// MetricsMiddleware creates a middleware that collects basic metrics
func MetricsMiddleware() func(http.Handler) http.Handler {
	var (
		requestCount    int64
		responseTimeSum int64
		errorCount      int64
		mu              sync.RWMutex
	)

	// Expose metrics endpoint
	http.HandleFunc("/internal/metrics", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		avgResponseTime := float64(0)
		if requestCount > 0 {
			avgResponseTime = float64(responseTimeSum) / float64(requestCount) / float64(time.Millisecond)
		}

		metrics := map[string]interface{}{
			"request_count":        requestCount,
			"error_count":          errorCount,
			"avg_response_time_ms": avgResponseTime,
			"error_rate":           float64(errorCount) / float64(requestCount),
		}

		WriteJSONResponse(w, http.StatusOK, &HTTPResponse{
			Success: true,
			Data:    metrics,
		})
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			mu.Lock()
			requestCount++
			responseTimeSum += int64(duration)
			if wrapped.statusCode >= 400 {
				errorCount++
			}
			mu.Unlock()
		})
	}
}

// CompressionMiddleware creates a middleware that compresses responses
func CompressionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts gzip encoding
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Set compression headers
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")

			// Create gzip writer
			gzipWriter := gzip.NewWriter(w)
			defer gzipWriter.Close()

			// Wrap response writer
			gzipResponseWriter := &gzipResponseWriter{
				ResponseWriter: w,
				Writer:         gzipWriter,
			}

			next.ServeHTTP(gzipResponseWriter, r)
		})
	}
}

// gzipResponseWriter wraps http.ResponseWriter with gzip compression
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (grw *gzipResponseWriter) Write(data []byte) (int, error) {
	return grw.Writer.Write(data)
}

// ================================
// Convenience Functions for Common Setups
// ================================

// WithLogging adds logging middleware
func WithLogging() HTTPServerOption {
	return WithGlobalMiddleware(LoggingMiddleware())
}

// WithAuth adds authentication middleware with API keys
func WithAuth(apiKeys ...string) HTTPServerOption {
	validKeys := make(map[string]bool)
	for _, key := range apiKeys {
		validKeys[key] = true
	}
	return WithRouteMiddleware(AuthenticationMiddleware(validKeys))
}

// WithRateLimit adds rate limiting middleware
func WithRateLimit(requestsPerMinute int) HTTPServerOption {
	return WithGlobalMiddleware(RateLimitMiddleware(requestsPerMinute))
}

// WithRecovery adds panic recovery middleware
func WithRecovery() HTTPServerOption {
	return WithGlobalMiddleware(RecoveryMiddleware())
}

// WithMetrics adds metrics collection middleware
func WithMetrics() HTTPServerOption {
	return WithGlobalMiddleware(MetricsMiddleware())
}

// WithCompression adds gzip compression middleware
func WithCompression() HTTPServerOption {
	return WithGlobalMiddleware(CompressionMiddleware())
}

// WithProductionDefaults applies common production middleware
func WithProductionDefaults() HTTPServerOption {
	return func(opts *HTTPServerOptions) {
		WithRecovery()(opts)
		WithLogging()(opts)
		WithMetrics()(opts)
		WithCompression()(opts)
		WithDefaultCORS()(opts)
	}
}
