package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/api"
	"github.com/kart-io/notifyhub/monitoring"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	hub api.Client
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(hub api.Client) *HealthHandler {
	return &HealthHandler{hub: hub}
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status     string                 `json:"status"`
	Timestamp  string                 `json:"timestamp"`
	Version    string                 `json:"version,omitempty"`
	Uptime     int64                  `json:"uptime_seconds"`
	Transports []string               `json:"transports"`
	Metrics    HealthMetrics          `json:"metrics"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// HealthMetrics represents health metrics
type HealthMetrics struct {
	TotalSent   int64   `json:"total_sent"`
	TotalFailed int64   `json:"total_failed"`
	SuccessRate float64 `json:"success_rate"`
	FailureRate float64 `json:"failure_rate"`
}

// Handle handles the health check request
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get metrics from hub
	metricsInterface := h.hub.Metrics()
	// Note: Transport details are now internal to the client
	transports := []string{"unified"} // V2 API uses unified interface

	// Type assert to get actual metrics
	metrics, ok := metricsInterface.(*monitoring.Metrics)
	if !ok {
		// Fallback if metrics interface changes
		response := HealthResponse{
			Status:     "degraded",
			Timestamp:  time.Now().Format("2006-01-02T15:04:05Z"),
			Transports: transports,
			Metrics: HealthMetrics{
				TotalSent:   0,
				TotalFailed: 0,
				SuccessRate: 0,
				FailureRate: 0,
			},
			Details: map[string]interface{}{
				"error": "metrics unavailable",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	// Calculate success/failure rates
	totalMessages := metrics.TotalSent + metrics.TotalFailed
	var successRate, failureRate float64
	if totalMessages > 0 {
		successRate = float64(metrics.TotalSent) / float64(totalMessages)
		failureRate = float64(metrics.TotalFailed) / float64(totalMessages)
	}

	// Calculate uptime
	uptime := time.Since(metrics.StartTime)

	// Build health response
	response := HealthResponse{
		Status:     "healthy",
		Timestamp:  time.Now().Format("2006-01-02T15:04:05Z"),
		Uptime:     int64(uptime.Seconds()),
		Transports: transports,
		Metrics: HealthMetrics{
			TotalSent:   metrics.TotalSent,
			TotalFailed: metrics.TotalFailed,
			SuccessRate: successRate,
			FailureRate: failureRate,
		},
		Details: map[string]interface{}{
			"platform_count": len(transports),
			"is_shutdown":    false, // V2 API doesn't expose shutdown state
		},
	}

	// V2 API always returns healthy status when accessible
	w.WriteHeader(http.StatusOK)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
