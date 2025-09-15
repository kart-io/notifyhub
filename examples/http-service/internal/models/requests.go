package models

import (
	"time"
)

// NotificationRequest represents a notification request
type NotificationRequest struct {
	Title     string                 `json:"title" binding:"required"`
	Body      string                 `json:"body" binding:"required"`
	Format    string                 `json:"format,omitempty"`
	Targets   []TargetRequest        `json:"targets" binding:"required"`
	Template  string                 `json:"template,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	Priority  int                    `json:"priority,omitempty"`
	Delay     *time.Duration         `json:"delay,omitempty"`
}

// TargetRequest represents a notification target
type TargetRequest struct {
	Type     string            `json:"type" binding:"required"`
	Value    string            `json:"value" binding:"required"`
	Platform string            `json:"platform,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// BulkNotificationRequest represents a bulk notification request
type BulkNotificationRequest struct {
	Notifications []NotificationRequest `json:"notifications" binding:"required"`
	Options       *SendOptionsRequest   `json:"options,omitempty"`
}

// SendOptionsRequest represents send options
type SendOptionsRequest struct {
	Async      bool           `json:"async,omitempty"`
	Retry      bool           `json:"retry,omitempty"`
	MaxRetries int            `json:"max_retries,omitempty"`
	Timeout    *time.Duration `json:"timeout,omitempty"`
}

// HealthCheckResponse represents health check response
type HealthCheckResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Uptime    time.Duration     `json:"uptime"`
}

// MetricsResponse represents metrics response
type MetricsResponse struct {
	TotalSent     int64             `json:"total_sent"`
	SuccessRate   float64           `json:"success_rate"`
	AvgDuration   time.Duration     `json:"avg_duration"`
	SendsByPlatform map[string]int64 `json:"sends_by_platform"`
	LastUpdated   time.Time         `json:"last_updated"`
}

// NotificationResponse represents a notification response
type NotificationResponse struct {
	ID      string                 `json:"id"`
	Status  string                 `json:"status"`
	Results []SendResultResponse   `json:"results,omitempty"`
	Message string                 `json:"message,omitempty"`
}

// SendResultResponse represents a send result
type SendResultResponse struct {
	Target    TargetRequest `json:"target"`
	Platform  string        `json:"platform"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	SentAt    time.Time     `json:"sent_at"`
	Attempts  int           `json:"attempts"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}