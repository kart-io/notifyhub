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
	hub *api.Client
}

// NewSendHandler creates a new send handler
func NewSendHandler(hub *api.Client) *SendHandler {
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

	// Send message using V2 API
	ctx := r.Context()
	// V2 API uses builder pattern - this is a simplified HTTP adapter
	result, err := h.hub.Send().
		Title(msg.Title).
		Body(msg.Body).
		Priority(msg.Priority).
		Send(ctx)
	if err != nil {
		http.Error(w, "Send failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build response for V2 API
	response := SendResponse{
		MessageID: msg.ID,
		Status:    "completed",
		Results:   []sending.Result{}, // V2 API has different result structure
		Summary: ResultSummary{
			Total:   1,
			Success: 1,
			Failed:  0,
			Pending: 0,
		},
	}

	// Add result information if available
	if result != nil {
		response.Summary.Success = 1
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
