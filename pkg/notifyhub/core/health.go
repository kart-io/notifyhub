// Package core provides health monitoring functionality
package core

import (
	"time"
)

// HealthStatus represents the overall health status
type HealthStatus struct {
	Healthy   bool                      `json:"healthy"`
	Status    string                    `json:"status"`
	Platforms map[string]PlatformHealth `json:"platforms"`
	Queue     QueueHealth               `json:"queue"`
	Timestamp time.Time                 `json:"timestamp"`
}

// PlatformHealth represents the health of a single platform
type PlatformHealth struct {
	Available bool              `json:"available"`
	Status    string            `json:"status"`
	Details   map[string]string `json:"details,omitempty"`
}

// QueueHealth represents the health of the queue system
type QueueHealth struct {
	Available bool `json:"available"`
}

// determineOverallStatus determines the overall health status
func (h *HealthStatus) determineOverallStatus() {
	h.Healthy = true
	h.Status = "healthy"

	for _, platform := range h.Platforms {
		if !platform.Available {
			h.Healthy = false
			h.Status = "unhealthy"
			break
		}
	}

	if !h.Queue.Available {
		h.Healthy = false
		h.Status = "unhealthy"
	}
}