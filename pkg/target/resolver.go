// Package target provides target resolution functionality for NotifyHub
package target

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/kart-io/notifyhub/pkg/errors"
	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// Resolver resolves targets from various formats and sources
type Resolver interface {
	// Resolve resolves a target specification into concrete targets
	Resolve(ctx context.Context, spec TargetSpec) ([]Target, error)

	// ResolveString resolves a string representation into targets
	ResolveString(ctx context.Context, targetStr string) ([]Target, error)

	// AddHandler adds a resolution handler for a specific type
	AddHandler(targetType string, handler ResolutionHandler)

	// RemoveHandler removes a resolution handler
	RemoveHandler(targetType string)
}

// TargetSpec represents a target specification that needs resolution
type TargetSpec struct {
	Type     string                 `json:"type"`
	Value    string                 `json:"value"`
	Platform string                 `json:"platform,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ResolutionHandler handles resolution for specific target types
type ResolutionHandler interface {
	// Resolve resolves a target specification into concrete targets
	Resolve(ctx context.Context, spec TargetSpec) ([]Target, error)

	// CanResolve checks if this handler can resolve the given specification
	CanResolve(spec TargetSpec) bool
}

// DefaultResolver provides default target resolution functionality
type DefaultResolver struct {
	handlers map[string]ResolutionHandler
	logger   logger.Logger
}

// NewDefaultResolver creates a new default resolver
func NewDefaultResolver(logger logger.Logger) *DefaultResolver {
	resolver := &DefaultResolver{
		handlers: make(map[string]ResolutionHandler),
		logger:   logger,
	}

	// Register default handlers
	resolver.AddHandler("email", &EmailResolutionHandler{})
	resolver.AddHandler("phone", &PhoneResolutionHandler{})
	resolver.AddHandler("user", &UserResolutionHandler{})
	resolver.AddHandler("group", &GroupResolutionHandler{})
	resolver.AddHandler("channel", &ChannelResolutionHandler{})
	resolver.AddHandler("webhook", &WebhookResolutionHandler{})

	return resolver
}

// Resolve resolves a target specification into concrete targets
func (r *DefaultResolver) Resolve(ctx context.Context, spec TargetSpec) ([]Target, error) {
	r.logger.Debug("Resolving target", "type", spec.Type, "value", spec.Value)

	handler, exists := r.handlers[spec.Type]
	if !exists {
		return r.fallbackResolve(spec)
	}

	if !handler.CanResolve(spec) {
		return r.fallbackResolve(spec)
	}

	targets, err := handler.Resolve(ctx, spec)
	if err != nil {
		r.logger.Error("Target resolution failed", "type", spec.Type, "value", spec.Value, "error", err)
		return nil, err
	}

	r.logger.Debug("Target resolved", "type", spec.Type, "value", spec.Value, "count", len(targets))
	return targets, nil
}

// ResolveString resolves a string representation into targets
func (r *DefaultResolver) ResolveString(ctx context.Context, targetStr string) ([]Target, error) {
	spec, err := r.parseTargetString(targetStr)
	if err != nil {
		return nil, err
	}

	return r.Resolve(ctx, spec)
}

// AddHandler adds a resolution handler
func (r *DefaultResolver) AddHandler(targetType string, handler ResolutionHandler) {
	r.handlers[targetType] = handler
	r.logger.Debug("Resolution handler added", "type", targetType)
}

// RemoveHandler removes a resolution handler
func (r *DefaultResolver) RemoveHandler(targetType string) {
	delete(r.handlers, targetType)
	r.logger.Debug("Resolution handler removed", "type", targetType)
}

// fallbackResolve provides fallback resolution for unknown types
func (r *DefaultResolver) fallbackResolve(spec TargetSpec) ([]Target, error) {
	// Create a basic target without resolution
	target := Target{
		Type:     spec.Type,
		Value:    spec.Value,
		Platform: spec.Platform,
	}

	r.logger.Debug("Using fallback resolution", "type", spec.Type, "value", spec.Value)
	return []Target{target}, nil
}

// parseTargetString parses a string representation into a TargetSpec
func (r *DefaultResolver) parseTargetString(targetStr string) (TargetSpec, error) {
	// Support various formats:
	// email:user@example.com
	// phone:+1234567890
	// user:john.doe
	// group:developers
	// channel:general
	// webhook:https://example.com/webhook
	// plain email: user@example.com

	// Check for type:value format
	if strings.Contains(targetStr, ":") {
		parts := strings.SplitN(targetStr, ":", 2)
		if len(parts) == 2 {
			return TargetSpec{
				Type:  strings.TrimSpace(parts[0]),
				Value: strings.TrimSpace(parts[1]),
			}, nil
		}
	}

	// Auto-detect type based on value format
	targetType := r.detectTargetType(targetStr)
	return TargetSpec{
		Type:  targetType,
		Value: targetStr,
	}, nil
}

// detectTargetType auto-detects target type based on value format
func (r *DefaultResolver) detectTargetType(value string) string {
	// Email pattern
	emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if emailPattern.MatchString(value) {
		return "email"
	}

	// Phone pattern
	phonePattern := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	if phonePattern.MatchString(value) {
		return "phone"
	}

	// URL pattern (webhook)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return "webhook"
	}

	// Channel pattern (starts with #)
	if strings.HasPrefix(value, "#") {
		return "channel"
	}

	// Group pattern (starts with @)
	if strings.HasPrefix(value, "@") {
		return "group"
	}

	// Default to user
	return "user"
}

// Email Resolution Handler
type EmailResolutionHandler struct{}

func (h *EmailResolutionHandler) CanResolve(spec TargetSpec) bool {
	return spec.Type == "email"
}

func (h *EmailResolutionHandler) Resolve(ctx context.Context, spec TargetSpec) ([]Target, error) {
	// Validate email format
	emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailPattern.MatchString(spec.Value) {
		return nil, &errors.NotifyError{
			Code:    errors.ErrInvalidTarget,
			Message: fmt.Sprintf("invalid email format: %s", spec.Value),
		}
	}

	return []Target{{
		Type:     "email",
		Value:    spec.Value,
		Platform: "email",
	}}, nil
}

// Phone Resolution Handler
type PhoneResolutionHandler struct{}

func (h *PhoneResolutionHandler) CanResolve(spec TargetSpec) bool {
	return spec.Type == "phone"
}

func (h *PhoneResolutionHandler) Resolve(ctx context.Context, spec TargetSpec) ([]Target, error) {
	// Validate phone format
	phonePattern := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	if !phonePattern.MatchString(spec.Value) {
		return nil, &errors.NotifyError{
			Code:    errors.ErrInvalidTarget,
			Message: fmt.Sprintf("invalid phone format: %s", spec.Value),
		}
	}

	return []Target{{
		Type:     "phone",
		Value:    spec.Value,
		Platform: "sms",
	}}, nil
}

// User Resolution Handler
type UserResolutionHandler struct{}

func (h *UserResolutionHandler) CanResolve(spec TargetSpec) bool {
	return spec.Type == "user"
}

func (h *UserResolutionHandler) Resolve(ctx context.Context, spec TargetSpec) ([]Target, error) {
	// User resolution logic:
	// 1. Parse user identifier (could be username, user ID, email)
	// 2. Look up user's preferred notification channels
	// 3. Expand to multiple targets if user has multiple channels
	// 4. Apply user preferences (e.g., do not disturb, preferred times)

	// Current implementation: Basic user resolution
	// Future enhancement: Integrate with user management system

	userID := spec.Value
	platform := spec.Platform

	// If platform is not specified, use email as default
	if platform == "" {
		// Check if value looks like an email
		if strings.Contains(userID, "@") {
			platform = "email"
		} else {
			// Default to email, assuming userID needs to be resolved to email
			platform = "email"
			// In a real system, you would:
			// 1. Query user database by userID
			// 2. Get user's email/phone/other contact info
			// 3. Get user's notification preferences
			// For now, append a default domain if needed
			if !strings.Contains(userID, "@") {
				userID = userID + "@example.com" // Placeholder
			}
		}
	}

	// Return resolved target
	// In production, this might return multiple targets for one user
	// (email, SMS, push notification, etc.)
	return []Target{{
		Type:     "user", // Keep original type
		Value:    userID,
		Platform: platform,
	}}, nil
}

// Group Resolution Handler
type GroupResolutionHandler struct{}

func (h *GroupResolutionHandler) CanResolve(spec TargetSpec) bool {
	return spec.Type == "group"
}

func (h *GroupResolutionHandler) Resolve(ctx context.Context, spec TargetSpec) ([]Target, error) {
	// Group resolution logic:
	// 1. Parse group identifier (could be group name, group ID)
	// 2. Query group membership from database/directory service
	// 3. Expand group into individual user targets
	// 4. Apply group-level notification policies

	// Current implementation: Basic group expansion
	// Future enhancement: Integrate with directory service (LDAP, AD, etc.)

	groupID := spec.Value
	platform := spec.Platform

	// In a real implementation, you would:
	// 1. Query group membership system
	// 2. Get list of users in the group
	// 3. Resolve each user to their contact info
	// 4. Apply group notification policies

	// Simulated group expansion
	// In production, replace this with actual database/API calls
	targets := []Target{}

	// Example: Expand group to members
	// This is a placeholder - in reality, you'd query a database
	groupMembers := h.getGroupMembers(groupID) // Mock function

	for _, member := range groupMembers {
		// Determine platform for each member
		memberPlatform := platform
		if memberPlatform == "" {
			memberPlatform = "email" // Default platform
		}

		targets = append(targets, Target{
			Type:     "group", // Keep original type
			Value:    member,
			Platform: memberPlatform,
		})
	}

	// If no members found, return error
	if len(targets) == 0 {
		return nil, fmt.Errorf("group '%s' has no members or does not exist", groupID)
	}

	return targets, nil
}

// getGroupMembers is a mock function that returns group members
// In production, this should query a real user directory service
func (h *GroupResolutionHandler) getGroupMembers(groupID string) []string {
	// Mock implementation
	// In production, replace with actual directory service query
	mockGroups := map[string][]string{
		"admins":     {"admin@example.com", "sysadmin@example.com"},
		"developers": {"dev1@example.com", "dev2@example.com", "dev3@example.com"},
		"support":    {"support1@example.com", "support2@example.com"},
		"all":        {"user1@example.com", "user2@example.com", "user3@example.com"},
	}

	members, exists := mockGroups[groupID]
	if !exists {
		// Return empty slice if group not found
		return []string{}
	}

	return members
}

// Channel Resolution Handler
type ChannelResolutionHandler struct{}

func (h *ChannelResolutionHandler) CanResolve(spec TargetSpec) bool {
	return spec.Type == "channel"
}

func (h *ChannelResolutionHandler) Resolve(ctx context.Context, spec TargetSpec) ([]Target, error) {
	return []Target{{
		Type:     "channel",
		Value:    spec.Value,
		Platform: spec.Platform,
	}}, nil
}

// Webhook Resolution Handler
type WebhookResolutionHandler struct{}

func (h *WebhookResolutionHandler) CanResolve(spec TargetSpec) bool {
	return spec.Type == "webhook"
}

func (h *WebhookResolutionHandler) Resolve(ctx context.Context, spec TargetSpec) ([]Target, error) {
	// Validate URL format
	if !strings.HasPrefix(spec.Value, "http://") && !strings.HasPrefix(spec.Value, "https://") {
		return nil, &errors.NotifyError{
			Code:    errors.ErrInvalidTarget,
			Message: fmt.Sprintf("invalid webhook URL format: %s", spec.Value),
		}
	}

	return []Target{{
		Type:     "webhook",
		Value:    spec.Value,
		Platform: "webhook",
	}}, nil
}

// Convenience functions

// ResolveEmail creates an email target
func ResolveEmail(email string) Target {
	return Target{
		Type:     "email",
		Value:    email,
		Platform: "email",
	}
}

// ResolvePhone creates a phone target
func ResolvePhone(phone string) Target {
	return Target{
		Type:     "phone",
		Value:    phone,
		Platform: "sms",
	}
}

// ResolveWebhook creates a webhook target
func ResolveWebhook(url string) Target {
	return Target{
		Type:     "webhook",
		Value:    url,
		Platform: "webhook",
	}
}

// ResolveFeishuWebhook creates a Feishu webhook target
func ResolveFeishuWebhook(url string) Target {
	return Target{
		Type:     "webhook",
		Value:    url,
		Platform: "feishu",
	}
}
