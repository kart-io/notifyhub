package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kart-io/notifyhub/api"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	hub *api.NotifyHub
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(hub *api.NotifyHub) *HealthHandler {
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
	metrics := h.hub.GetMetrics()
	transports := h.hub.GetTransports()

	// Build health response
	response := HealthResponse{
		Status:     "healthy",
		Timestamp:  "2024-01-01T00:00:00Z", // Should use actual timestamp
		Uptime:     metrics.UptimeSeconds,
		Transports: transports,
		Metrics: HealthMetrics{
			TotalSent:   metrics.TotalSent,
			TotalFailed: metrics.TotalFailed,
			SuccessRate: metrics.SuccessRate,
			FailureRate: metrics.FailureRate,
		},
		Details: map[string]interface{}{
			"platform_count": len(transports),
			"is_shutdown":    h.hub.IsShutdown(),
		},
	}

	// Check if hub is shutdown
	if h.hub.IsShutdown() {
		response.Status = "shutdown"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
