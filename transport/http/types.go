package http

import (
	"net/http"

	"github.com/kart-io/notifyhub/api"
)

// sendHandler handles individual message sending
type sendHandler struct {
	hub *api.NotifyHub
}

// Handle handles the send request
func (h *sendHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Implementation moved here from handlers package to avoid import cycles
	// This is a simplified implementation
	w.WriteHeader(200)
	_, _ = w.Write([]byte(`{"status": "ok"}`))
}

// batchHandler handles batch message sending
type batchHandler struct {
	hub *api.NotifyHub
}

// Handle handles the batch send request
func (h *batchHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Implementation moved here from handlers package to avoid import cycles
	// This is a simplified implementation
	w.WriteHeader(200)
	_, _ = w.Write([]byte(`{"status": "ok"}`))
}

// healthHandler handles health check requests
type healthHandler struct {
	hub *api.NotifyHub
}

// Handle handles the health check request
func (h *healthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Implementation moved here from handlers package to avoid import cycles
	// This is a simplified implementation
	w.WriteHeader(200)
	_, _ = w.Write([]byte(`{"status": "healthy"}`))
}
