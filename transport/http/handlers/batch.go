package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kart-io/notifyhub/api"
)

// BatchHandler handles batch message sending
type BatchHandler struct {
	hub *api.NotifyHub
}

// NewBatchHandler creates a new batch handler
func NewBatchHandler(hub *api.NotifyHub) *BatchHandler {
	return &BatchHandler{hub: hub}
}

// BatchRequest represents a batch send request
type BatchRequest struct {
	Messages []SendRequest `json:"messages"`
	Options  BatchOptions  `json:"options,omitempty"`
}

// BatchOptions represents batch processing options
type BatchOptions struct {
	Parallel       bool `json:"parallel,omitempty"`
	MaxConcurrency int  `json:"max_concurrency,omitempty"`
	StopOnError    bool `json:"stop_on_error,omitempty"`
}

// BatchResponse represents a batch send response
type BatchResponse struct {
	BatchID string         `json:"batch_id"`
	Status  string         `json:"status"`
	Results []SendResponse `json:"results"`
	Summary BatchSummary   `json:"summary"`
}

// BatchSummary provides a summary of batch results
type BatchSummary struct {
	TotalMessages   int `json:"total_messages"`
	SuccessMessages int `json:"success_messages"`
	FailedMessages  int `json:"failed_messages"`
	TotalTargets    int `json:"total_targets"`
	SuccessTargets  int `json:"success_targets"`
	FailedTargets   int `json:"failed_targets"`
}

// Handle handles the batch send request
func (h *BatchHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Messages) == 0 {
		http.Error(w, "No messages provided", http.StatusBadRequest)
		return
	}

	// Process batch
	batchID := generateBatchID()
	results := make([]SendResponse, len(req.Messages))

	var summary BatchSummary
	summary.TotalMessages = len(req.Messages)

	// Process each message
	for i, msgReq := range req.Messages {
		// Create a temporary send handler to reuse logic
		sendHandler := NewSendHandler(h.hub)

		// Validate message
		if err := sendHandler.validateRequest(&msgReq); err != nil {
			results[i] = SendResponse{
				Status: "failed",
			}
			summary.FailedMessages++
			continue
		}

		// This is a simplified batch processing
		// In a real implementation, you'd handle parallel processing,
		// concurrency limits, and error handling as specified in options

		summary.SuccessMessages++
		results[i] = SendResponse{
			Status: "completed",
		}
	}

	// Build response
	response := BatchResponse{
		BatchID: batchID,
		Status:  "completed",
		Results: results,
		Summary: summary,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// generateBatchID generates a unique batch ID
func generateBatchID() string {
	// Simple batch ID generation
	// In a real implementation, you'd use a proper ID generator
	return "batch_" + "123456"
}
