package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kart-io/notifyhub/api"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	hub *api.Client
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(hub *api.Client) *HealthHandler {
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
	metrics := h.hub.Metrics()
	// Note: Transport details are now internal to the client
	transports := []string{"unified"} // V2 API uses unified interface

	// Calculate success/failure rates
	totalMessages := metrics.MessagesSent + metrics.MessagesFailed
	var successRate, failureRate float64
	if totalMessages > 0 {
		successRate = float64(metrics.MessagesSent) / float64(totalMessages)
		failureRate = float64(metrics.MessagesFailed) / float64(totalMessages)
	}

	// Build health response
	response := HealthResponse{
		Status:     "healthy",
		Timestamp:  metrics.Timestamp.Format("2006-01-02T15:04:05Z"),
		Uptime:     0, // This would need to be calculated from start time
		Transports: transports,
		Metrics: HealthMetrics{
			TotalSent:   metrics.MessagesSent,
			TotalFailed: metrics.MessagesFailed,
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
