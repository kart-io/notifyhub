package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kart-io/notifyhub/api"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
)

// SendHandler handles individual message sending
type SendHandler struct {
	hub *api.NotifyHub
}

// NewSendHandler creates a new send handler
func NewSendHandler(hub *api.NotifyHub) *SendHandler {
	return &SendHandler{hub: hub}
}

// SendRequest represents a send request
type SendRequest struct {
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Format    string                 `json:"format,omitempty"`
	Priority  int                    `json:"priority,omitempty"`
	Template  string                 `json:"template,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	Targets   []TargetRequest        `json:"targets"`
}

// TargetRequest represents a target in the request
type TargetRequest struct {
	Type     string            `json:"type"`
	Value    string            `json:"value"`
	Platform string            `json:"platform"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SendResponse represents a send response
type SendResponse struct {
	MessageID string           `json:"message_id"`
	Status    string           `json:"status"`
	Results   []sending.Result `json:"results"`
	Summary   ResultSummary    `json:"summary"`
}

// ResultSummary provides a summary of sending results
type ResultSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Pending int `json:"pending"`
}

// Handle handles the send request
func (h *SendHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if err := h.validateRequest(&req); err != nil {
		http.Error(w, "Validation error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Build message
	msg := message.NewMessage()
	msg.SetTitle(req.Title).SetBody(req.Body)

	if req.Format != "" {
		msg.SetFormat(message.Format(req.Format))
	}
	if req.Priority > 0 {
		msg.SetPriority(req.Priority)
	}
	if req.Template != "" {
		msg.SetTemplate(req.Template)
	}

	// Add variables
	for k, v := range req.Variables {
		msg.AddVariable(k, v)
	}

	// Add metadata
	for k, v := range req.Metadata {
		msg.AddMetadata(k, v)
	}

	// Build targets
	targets := make([]sending.Target, len(req.Targets))
	for i, t := range req.Targets {
		targets[i] = sending.NewTarget(sending.TargetType(t.Type), t.Value, t.Platform)
		for k, v := range t.Metadata {
			targets[i].AddMetadata(k, v)
		}
	}

	// Send message
	ctx := r.Context()
	results, err := h.hub.Send(ctx, msg, targets)
	if err != nil {
		http.Error(w, "Send failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build response
	response := SendResponse{
		MessageID: msg.ID,
		Status:    "completed",
		Results:   make([]sending.Result, len(results.Results)),
		Summary: ResultSummary{
			Total:   results.Total,
			Success: results.Success,
			Failed:  results.Failed,
			Pending: results.Pending,
		},
	}

	// Copy results (dereference pointers)
	for i, result := range results.Results {
		response.Results[i] = *result
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// validateRequest validates the send request
func (h *SendHandler) validateRequest(req *SendRequest) error {
	if req.Title == "" && req.Body == "" {
		return ErrEmptyMessage
	}
	if len(req.Targets) == 0 {
		return ErrNoTargets
	}
	return nil
}
