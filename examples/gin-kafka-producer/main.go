package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"

	"github.com/kart-io/notifyhub/notifiers"
)

// ================================
// Configuration
// ================================

// Config holds the application configuration
type Config struct {
	HTTP struct {
		Port string `json:"port"`
	} `json:"http"`

	Kafka struct {
		Brokers []string `json:"brokers"`
		Topic   string   `json:"topic"`
	} `json:"kafka"`

	Service struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"service"`
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	config := &Config{}

	// HTTP configuration
	config.HTTP.Port = getEnvOrDefault("HTTP_PORT", "8080")

	// Kafka configuration
	brokersStr := getEnvOrDefault("KAFKA_BROKERS", "localhost:9092")
	config.Kafka.Brokers = strings.Split(brokersStr, ",")
	config.Kafka.Topic = getEnvOrDefault("KAFKA_TOPIC", "notifications")

	// Service configuration
	config.Service.Name = getEnvOrDefault("SERVICE_NAME", "gin-kafka-producer")
	config.Service.Version = getEnvOrDefault("SERVICE_VERSION", "1.0.0")

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ================================
// HTTP Request/Response Models
// ================================

// NotificationRequest represents the HTTP request payload
type NotificationRequest struct {
	Title     string                 `json:"title" binding:"required" example:"Alert"`
	Body      string                 `json:"body" binding:"required" example:"System alert message"`
	Priority  int                    `json:"priority,omitempty" example:"3"`
	Targets   []TargetRequest        `json:"targets" binding:"required,dive"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Template  string                 `json:"template,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`

	// Kafka-specific options
	KafkaOptions *KafkaOptions `json:"kafka_options,omitempty"`
}

// TargetRequest represents a notification target in the request
type TargetRequest struct {
	Type     string `json:"type" binding:"required,oneof=email user group channel" example:"email"`
	Value    string `json:"value" binding:"required" example:"admin@example.com"`
	Platform string `json:"platform,omitempty" example:""`
}

// KafkaOptions represents Kafka-specific sending options
type KafkaOptions struct {
	Partition int               `json:"partition,omitempty"`
	Key       string            `json:"key,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// NotificationResponse represents the HTTP response
type NotificationResponse struct {
	Success   bool              `json:"success"`
	Message   string            `json:"message"`
	Data      *NotificationData `json:"data,omitempty"`
	RequestID string            `json:"request_id"`
	Timestamp time.Time         `json:"timestamp"`
	Errors    []string          `json:"errors,omitempty"`
}

// NotificationData contains response data
type NotificationData struct {
	MessageID      string `json:"message_id"`
	KafkaOffset    int64  `json:"kafka_offset"`
	KafkaPartition int    `json:"kafka_partition"`
	TargetCount    int    `json:"target_count"`
}

// ================================
// Kafka Message Format
// ================================

// KafkaNotificationMessage represents the message sent to Kafka
type KafkaNotificationMessage struct {
	// Metadata
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"`
	Version   string    `json:"version"`

	// NotifyHub message (this is what the consumer will process)
	Message *notifiers.Message `json:"message"`

	// Optional processing hints for consumer
	ProcessingHints *ProcessingHints `json:"processing_hints,omitempty"`
}

// ProcessingHints provides guidance to the Kafka consumer
type ProcessingHints struct {
	Priority         int               `json:"priority,omitempty"`
	RetryPolicy      *RetryHints       `json:"retry_policy,omitempty"`
	DeliveryDeadline *time.Time        `json:"delivery_deadline,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// RetryHints suggests retry behavior to the consumer
type RetryHints struct {
	MaxRetries      int           `json:"max_retries,omitempty"`
	InitialInterval time.Duration `json:"initial_interval,omitempty"`
	BackoffFactor   float64       `json:"backoff_factor,omitempty"`
}

// ================================
// Fixed Message Builder
// ================================

// FixedMessageBuilder provides message building functionality without client package dependency
type FixedMessageBuilder struct {
	message *notifiers.Message
}

// NewFixedMessage creates a new message builder
func NewFixedMessage() *FixedMessageBuilder {
	return &FixedMessageBuilder{
		message: &notifiers.Message{
			Targets:   []notifiers.Target{},
			Variables: make(map[string]interface{}),
			Metadata:  make(map[string]string),
			Priority:  3, // Default priority
			CreatedAt: time.Now(),
		},
	}
}

// Title sets the message title
func (b *FixedMessageBuilder) Title(title string) *FixedMessageBuilder {
	b.message.Title = title
	return b
}

// Body sets the message body
func (b *FixedMessageBuilder) Body(body string) *FixedMessageBuilder {
	b.message.Body = body
	return b
}

// Priority sets the message priority
func (b *FixedMessageBuilder) Priority(priority int) *FixedMessageBuilder {
	b.message.Priority = priority
	return b
}

// Target adds a target to the message
func (b *FixedMessageBuilder) Target(target notifiers.Target) *FixedMessageBuilder {
	b.message.Targets = append(b.message.Targets, target)
	return b
}

// Variable adds a variable to the message
func (b *FixedMessageBuilder) Variable(key string, value interface{}) *FixedMessageBuilder {
	b.message.Variables[key] = value
	return b
}

// Template sets the message template
func (b *FixedMessageBuilder) Template(template string) *FixedMessageBuilder {
	b.message.Template = template
	return b
}

// Metadata adds metadata to the message
func (b *FixedMessageBuilder) Metadata(key, value string) *FixedMessageBuilder {
	b.message.Metadata[key] = value
	return b
}

// Build returns the built message
func (b *FixedMessageBuilder) Build() *notifiers.Message {
	// Generate ID if not set
	if b.message.ID == "" {
		b.message.ID = generateMessageID()
	}
	return b.message
}

// ================================
// Kafka Producer Service
// ================================

// KafkaProducer handles Kafka message production
type KafkaProducer struct {
	writer *kafka.Writer
	config *Config
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(config *Config) *KafkaProducer {
	// Configure Kafka writer
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Kafka.Brokers...),
		Topic:        config.Kafka.Topic,
		Balancer:     &kafka.LeastBytes{}, // Distribute messages evenly
		RequiredAcks: kafka.RequireOne,    // Wait for leader acknowledgment
		Async:        false,               // Synchronous writes for this example
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    100,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		ErrorLogger:  kafka.LoggerFunc(log.Printf),
	}

	return &KafkaProducer{
		writer: writer,
		config: config,
	}
}

// SendNotification sends a notification message to Kafka
func (kp *KafkaProducer) SendNotification(ctx context.Context, message *notifiers.Message, options *KafkaOptions) (*NotificationData, error) {
	// Create Kafka notification message
	kafkaMessage := &KafkaNotificationMessage{
		ID:        generateMessageID(),
		CreatedAt: time.Now(),
		Source:    kp.config.Service.Name,
		Version:   kp.config.Service.Version,
		Message:   message,
		ProcessingHints: &ProcessingHints{
			Priority: message.Priority,
			RetryPolicy: &RetryHints{
				MaxRetries:      3,
				InitialInterval: 30 * time.Second,
				BackoffFactor:   2.0,
			},
			Tags: []string{"http-api", "gin-producer"},
		},
	}

	// Serialize to JSON
	payload, err := json.Marshal(kafkaMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	// Prepare Kafka message
	kafkaMsg := kafka.Message{
		Value: payload,
		Time:  time.Now(),
	}

	// Add options if provided
	if options != nil {
		if options.Key != "" {
			kafkaMsg.Key = []byte(options.Key)
		}

		if options.Headers != nil {
			for key, value := range options.Headers {
				kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
					Key:   key,
					Value: []byte(value),
				})
			}
		}
	}

	// Add standard headers
	kafkaMsg.Headers = append(kafkaMsg.Headers,
		kafka.Header{Key: "source", Value: []byte(kp.config.Service.Name)},
		kafka.Header{Key: "message-id", Value: []byte(kafkaMessage.ID)},
		kafka.Header{Key: "content-type", Value: []byte("application/json")},
		kafka.Header{Key: "schema-version", Value: []byte("1.0")},
	)

	// Send to Kafka
	err = kp.writer.WriteMessages(ctx, kafkaMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to send message to Kafka: %w", err)
	}

	// Return response data
	return &NotificationData{
		MessageID:      kafkaMessage.ID,
		KafkaOffset:    -1, // Offset not available in synchronous mode
		KafkaPartition: -1, // Partition not available in synchronous mode
		TargetCount:    len(message.Targets),
	}, nil
}

// Close closes the Kafka producer
func (kp *KafkaProducer) Close() error {
	return kp.writer.Close()
}

// ================================
// HTTP Service
// ================================

// HTTPService handles HTTP requests
type HTTPService struct {
	config   *Config
	producer *KafkaProducer
	router   *gin.Engine
}

// NewHTTPService creates a new HTTP service
func NewHTTPService(config *Config, producer *KafkaProducer) *HTTPService {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(requestIDMiddleware())
	router.Use(corsMiddleware())
	router.Use(loggingMiddleware())

	service := &HTTPService{
		config:   config,
		producer: producer,
		router:   router,
	}

	// Setup routes
	service.setupRoutes()

	return service
}

// setupRoutes configures HTTP routes
func (hs *HTTPService) setupRoutes() {
	// Health check
	hs.router.GET("/health", hs.healthHandler)

	// API info
	hs.router.GET("/", hs.infoHandler)

	// Notification API
	api := hs.router.Group("/api/v1")
	{
		api.POST("/notifications", hs.sendNotificationHandler)
		api.POST("/send", hs.sendNotificationHandler) // Alternative endpoint
	}

	// Metrics and status
	hs.router.GET("/metrics", hs.metricsHandler)
	hs.router.GET("/status", hs.statusHandler)
}

// sendNotificationHandler handles notification sending requests
func (hs *HTTPService) sendNotificationHandler(c *gin.Context) {
	requestID := getRequestID(c)

	// Parse request
	var req NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, &NotificationResponse{
			Success:   false,
			Message:   "Invalid request format",
			RequestID: requestID,
			Timestamp: time.Now(),
			Errors:    []string{err.Error()},
		})
		return
	}

	// Convert to NotifyHub message using FixedMessageBuilder
	message, err := hs.buildNotificationMessage(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, &NotificationResponse{
			Success:   false,
			Message:   "Failed to build notification message",
			RequestID: requestID,
			Timestamp: time.Now(),
			Errors:    []string{err.Error()},
		})
		return
	}

	// Send to Kafka
	data, err := hs.producer.SendNotification(c.Request.Context(), message, req.KafkaOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, &NotificationResponse{
			Success:   false,
			Message:   "Failed to send notification to Kafka",
			RequestID: requestID,
			Timestamp: time.Now(),
			Errors:    []string{err.Error()},
		})
		return
	}

	// Success response
	c.JSON(http.StatusAccepted, &NotificationResponse{
		Success:   true,
		Message:   "Notification queued successfully",
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now(),
	})
}

// buildNotificationMessage converts HTTP request to NotifyHub message
func (hs *HTTPService) buildNotificationMessage(req *NotificationRequest) (*notifiers.Message, error) {
	// Start with FixedMessageBuilder
	builder := NewFixedMessage().
		Title(req.Title).
		Body(req.Body)

	// Set priority (default to 3 if not provided)
	priority := req.Priority
	if priority == 0 {
		priority = 3
	}
	builder = builder.Priority(priority)

	// Add targets
	for _, target := range req.Targets {
		notifyTarget := notifiers.Target{
			Type:     notifiers.TargetType(target.Type),
			Value:    target.Value,
			Platform: target.Platform,
		}
		builder = builder.Target(notifyTarget)
	}

	// Add variables if provided
	if req.Variables != nil {
		for key, value := range req.Variables {
			builder = builder.Variable(key, value)
		}
	}

	// Add template if provided
	if req.Template != "" {
		builder = builder.Template(req.Template)
	}

	// Add metadata if provided
	if req.Metadata != nil {
		for key, value := range req.Metadata {
			builder = builder.Metadata(key, value)
		}
	}

	// Build the message
	return builder.Build(), nil
}

// healthHandler returns service health status
func (hs *HTTPService) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   hs.config.Service.Name,
		"version":   hs.config.Service.Version,
		"timestamp": time.Now(),
	})
}

// infoHandler returns service information
func (hs *HTTPService) infoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": gin.H{
			"name":        hs.config.Service.Name,
			"version":     hs.config.Service.Version,
			"description": "Gin HTTP to Kafka notification producer service",
		},
		"endpoints": gin.H{
			"send_notification": "/api/v1/notifications",
			"health":            "/health",
			"metrics":           "/metrics",
			"status":            "/status",
		},
		"kafka": gin.H{
			"topic":   hs.config.Kafka.Topic,
			"brokers": hs.config.Kafka.Brokers,
		},
		"documentation": "https://github.com/kart-io/notifyhub/tree/main/examples/gin-kafka-producer",
	})
}

// metricsHandler returns basic metrics
func (hs *HTTPService) metricsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"metrics": gin.H{
			"requests_total":      "TODO: implement counter",
			"requests_success":    "TODO: implement counter",
			"requests_error":      "TODO: implement counter",
			"kafka_messages_sent": "TODO: implement counter",
		},
		"timestamp": time.Now(),
	})
}

// statusHandler returns detailed service status
func (hs *HTTPService) statusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": gin.H{
			"name":    hs.config.Service.Name,
			"version": hs.config.Service.Version,
			"status":  "running",
			"uptime":  "TODO: implement uptime tracking",
		},
		"kafka": gin.H{
			"topic":     hs.config.Kafka.Topic,
			"brokers":   hs.config.Kafka.Brokers,
			"connected": "TODO: implement connection check",
		},
		"timestamp": time.Now(),
	})
}

// Run starts the HTTP server
func (hs *HTTPService) Run() error {
	addr := ":" + hs.config.HTTP.Port
	log.Printf("Starting HTTP server on %s", addr)
	return hs.router.Run(addr)
}

// ================================
// Middleware
// ================================

// requestIDMiddleware adds request ID to each request
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
	})
}

// ================================
// Utility Functions
// ================================

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// getRequestID extracts request ID from Gin context
func getRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		return id.(string)
	}
	return "unknown"
}

// ================================
// Main Function
// ================================

func main() {
	fmt.Println("🚀 Gin Kafka Producer - NotifyHub HTTP to Kafka Bridge")
	fmt.Println("=" + strings.Repeat("=", 65))

	// Load configuration
	config := LoadConfig()
	fmt.Printf("📋 Configuration loaded:\n")
	fmt.Printf("  HTTP Port: %s\n", config.HTTP.Port)
	fmt.Printf("  Kafka Brokers: %v\n", config.Kafka.Brokers)
	fmt.Printf("  Kafka Topic: %s\n", config.Kafka.Topic)
	fmt.Printf("  Service: %s v%s\n", config.Service.Name, config.Service.Version)

	// Create Kafka producer
	fmt.Printf("\n📡 Initializing Kafka producer...\n")
	producer := NewKafkaProducer(config)
	defer func() {
		fmt.Printf("🔌 Closing Kafka producer...\n")
		if err := producer.Close(); err != nil {
			log.Printf("Error closing Kafka producer: %v", err)
		}
	}()

	// Create HTTP service
	fmt.Printf("🌐 Initializing HTTP service...\n")
	httpService := NewHTTPService(config, producer)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("\n🛑 Shutdown signal received, stopping service...\n")
		cancel()
	}()

	// Start HTTP service
	fmt.Printf("\n✅ Service ready! Endpoints:\n")
	fmt.Printf("  📍 Send Notification: POST http://localhost:%s/api/v1/notifications\n", config.HTTP.Port)
	fmt.Printf("  💚 Health Check: GET http://localhost:%s/health\n", config.HTTP.Port)
	fmt.Printf("  ℹ️  Service Info: GET http://localhost:%s/\n", config.HTTP.Port)
	fmt.Printf("  📊 Metrics: GET http://localhost:%s/metrics\n", config.HTTP.Port)
	fmt.Printf("\n🎯 Kafka Target: %s -> %s\n", strings.Join(config.Kafka.Brokers, ","), config.Kafka.Topic)
	fmt.Printf("\n📖 Example Usage:\n")
	fmt.Printf("  curl -X POST http://localhost:%s/api/v1/notifications \\\n", config.HTTP.Port)
	fmt.Printf("    -H \"Content-Type: application/json\" \\\n")
	fmt.Printf("    -d '{\n")
	fmt.Printf("      \"title\": \"Test Alert\",\n")
	fmt.Printf("      \"body\": \"This is a test notification\",\n")
	fmt.Printf("      \"priority\": 3,\n")
	fmt.Printf("      \"targets\": [\n")
	fmt.Printf("        {\"type\": \"email\", \"value\": \"admin@example.com\"}\n")
	fmt.Printf("      ]\n")
	fmt.Printf("    }'\n")
	fmt.Printf("\n🔥 Service starting...\n")

	// Run with graceful shutdown
	go func() {
		if err := httpService.Run(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	fmt.Printf("✅ Service stopped gracefully\n")
}