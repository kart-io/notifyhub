package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/notifiers"
	"github.com/kart-io/notifyhub/template"
)

// TemplateManager provides enhanced template management capabilities
type TemplateManager struct {
	engine       *template.Engine
	hub          *Hub
	templates    map[string]*TemplateMetadata
	cache        map[string]*CachedTemplate
	validators   []TemplateValidator
	preprocessors []TemplatePreprocessor
	mu           sync.RWMutex
	cacheEnabled bool
	cacheTimeout time.Duration
}

// TemplateMetadata contains information about a template
type TemplateMetadata struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Version       string            `json:"version"`
	Author        string            `json:"author"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Tags          []string          `json:"tags"`
	Variables     []VariableInfo    `json:"variables"`
	Platforms     []string          `json:"platforms"`
	MessageTypes  []string          `json:"message_types"`
	UsageCount    int64             `json:"usage_count"`
	Metadata      map[string]string `json:"metadata"`
}

// VariableInfo describes a template variable
type VariableInfo struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // "string", "number", "boolean", "object", "array"
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Description  string      `json:"description"`
	Examples     []string    `json:"examples,omitempty"`
	Validation   string      `json:"validation,omitempty"` // validation rule/pattern
}

// CachedTemplate represents a cached compiled template
type CachedTemplate struct {
	CompiledTemplate interface{}
	Metadata         *TemplateMetadata
	CompiledAt       time.Time
	LastUsed         time.Time
	UsageCount       int64
}

// TemplateValidator validates template content and metadata
type TemplateValidator func(name string, content string, metadata *TemplateMetadata) error

// TemplatePreprocessor preprocesses template content before compilation
type TemplatePreprocessor func(name string, content string) (string, error)

// TemplateValidationResult represents template validation results
type TemplateValidationResult struct {
	Valid         bool                    `json:"valid"`
	Errors        []TemplateValidationError `json:"errors,omitempty"`
	Warnings      []TemplateValidationWarning `json:"warnings,omitempty"`
	MissingVars   []string                `json:"missing_variables,omitempty"`
	UnusedVars    []string                `json:"unused_variables,omitempty"`
	Suggestions   []string                `json:"suggestions,omitempty"`
}

// TemplateValidationError represents a template validation error
type TemplateValidationError struct {
	Type        string `json:"type"`    // "syntax", "variable", "metadata", "logic"
	Message     string `json:"message"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Context     string `json:"context,omitempty"`
	Severity    string `json:"severity"` // "error", "warning"
}

// TemplateValidationWarning represents a template validation warning
type TemplateValidationWarning struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	Context string `json:"context,omitempty"`
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(hub *Hub) *TemplateManager {
	tm := &TemplateManager{
		engine:        hub.templates,
		hub:           hub,
		templates:     make(map[string]*TemplateMetadata),
		cache:         make(map[string]*CachedTemplate),
		validators:    make([]TemplateValidator, 0),
		preprocessors: make([]TemplatePreprocessor, 0),
		cacheEnabled:  true,
		cacheTimeout:  time.Hour,
	}

	// Add default validators
	tm.AddValidator(tm.validateSyntax)
	tm.AddValidator(tm.validateVariables)
	tm.AddValidator(tm.validateMetadata)

	return tm
}

// RegisterTemplate registers a template with metadata
func (tm *TemplateManager) RegisterTemplate(name, content string, metadata *TemplateMetadata) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Set default metadata if not provided
	if metadata == nil {
		metadata = &TemplateMetadata{
			Name:      name,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   "1.0.0",
		}
	}

	// Update timestamps
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now()
	}
	metadata.UpdatedAt = time.Now()

	// Run preprocessors
	processedContent := content
	for _, preprocessor := range tm.preprocessors {
		var err error
		processedContent, err = preprocessor(name, processedContent)
		if err != nil {
			return fmt.Errorf("template preprocessing failed: %w", err)
		}
	}

	// Run validators
	for _, validator := range tm.validators {
		if err := validator(name, processedContent, metadata); err != nil {
			return fmt.Errorf("template validation failed: %w", err)
		}
	}

	// Register with engine
	if tm.engine != nil {
		if err := tm.engine.AddTextTemplate(name, processedContent); err != nil {
			return fmt.Errorf("failed to register template with engine: %w", err)
		}
	}

	// Store metadata
	tm.templates[name] = metadata

	// Clear cache for this template
	delete(tm.cache, name)

	return nil
}

// GetTemplate retrieves a template by name
func (tm *TemplateManager) GetTemplate(name string) (*TemplateMetadata, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	metadata, exists := tm.templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	return metadata, nil
}

// ListTemplates lists all registered templates
func (tm *TemplateManager) ListTemplates() []*TemplateMetadata {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	templates := make([]*TemplateMetadata, 0, len(tm.templates))
	for _, metadata := range tm.templates {
		templates = append(templates, metadata)
	}

	return templates
}

// SearchTemplates searches templates by tags, name, or description
func (tm *TemplateManager) SearchTemplates(query string) []*TemplateMetadata {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	query = strings.ToLower(query)
	results := make([]*TemplateMetadata, 0)

	for _, metadata := range tm.templates {
		// Search in name
		if strings.Contains(strings.ToLower(metadata.Name), query) {
			results = append(results, metadata)
			continue
		}

		// Search in description
		if strings.Contains(strings.ToLower(metadata.Description), query) {
			results = append(results, metadata)
			continue
		}

		// Search in tags
		for _, tag := range metadata.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, metadata)
				break
			}
		}
	}

	return results
}

// ValidateTemplate validates a template without registering it
func (tm *TemplateManager) ValidateTemplate(name, content string, metadata *TemplateMetadata) *TemplateValidationResult {
	result := &TemplateValidationResult{
		Valid:       true,
		Errors:      make([]TemplateValidationError, 0),
		Warnings:    make([]TemplateValidationWarning, 0),
		MissingVars: make([]string, 0),
		UnusedVars:  make([]string, 0),
		Suggestions: make([]string, 0),
	}

	// Run all validators
	for _, validator := range tm.validators {
		if err := validator(name, content, metadata); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, TemplateValidationError{
				Type:     "validation",
				Message:  err.Error(),
				Severity: "error",
			})
		}
	}

	return result
}

// RenderTemplate renders a template with variables
func (tm *TemplateManager) RenderTemplate(ctx context.Context, name string, variables map[string]interface{}) (string, error) {
	// Update usage statistics
	tm.updateUsageStats(name)

	// Check cache first
	if tm.cacheEnabled {
		if cached := tm.getCachedTemplate(name); cached != nil {
			cached.LastUsed = time.Now()
			cached.UsageCount++
		}
	}

	// Render using the engine - create a message and render it
	if tm.engine != nil {
		message := &notifiers.Message{
			Title:     name,
			Body:      "{{." + name + "}}",
			Variables: variables,
		}
		rendered, err := tm.engine.RenderMessage(message)
		if err != nil {
			return "", err
		}
		return rendered.Body, nil
	}

	return "", fmt.Errorf("template engine not available")
}

// DeleteTemplate removes a template
func (tm *TemplateManager) DeleteTemplate(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if template exists
	if _, exists := tm.templates[name]; !exists {
		return fmt.Errorf("template not found: %s", name)
	}

	// Remove from engine (template engine doesn't support removal, just clear from memory)
	// The template engine doesn't have a RemoveTemplate method, so we can't remove it

	// Remove from manager
	delete(tm.templates, name)
	delete(tm.cache, name)

	return nil
}

// UpdateTemplate updates an existing template
func (tm *TemplateManager) UpdateTemplate(name, content string, metadata *TemplateMetadata) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if template exists
	existingMetadata, exists := tm.templates[name]
	if !exists {
		return fmt.Errorf("template not found: %s", name)
	}

	// Merge metadata
	if metadata == nil {
		metadata = existingMetadata
	} else {
		// Preserve creation time and usage stats
		metadata.CreatedAt = existingMetadata.CreatedAt
		metadata.UsageCount = existingMetadata.UsageCount
	}

	metadata.UpdatedAt = time.Now()

	// Use the regular registration process (which includes validation)
	tm.mu.Unlock() // Unlock before calling RegisterTemplate
	return tm.RegisterTemplate(name, content, metadata)
}

// GetTemplateStats returns usage statistics for a template
func (tm *TemplateManager) GetTemplateStats(name string) (*TemplateStats, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	metadata, exists := tm.templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	stats := &TemplateStats{
		Name:         metadata.Name,
		UsageCount:   metadata.UsageCount,
		CreatedAt:    metadata.CreatedAt,
		UpdatedAt:    metadata.UpdatedAt,
		Version:      metadata.Version,
		CacheStatus:  "not_cached",
	}

	// Check cache status
	if cached, exists := tm.cache[name]; exists {
		stats.CacheStatus = "cached"
		stats.LastCacheTime = &cached.CompiledAt
		stats.CacheUsageCount = cached.UsageCount
	}

	return stats, nil
}

// ClearCache clears the template cache
func (tm *TemplateManager) ClearCache() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.cache = make(map[string]*CachedTemplate)
}

// SetCacheConfig configures template caching
func (tm *TemplateManager) SetCacheConfig(enabled bool, timeout time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.cacheEnabled = enabled
	tm.cacheTimeout = timeout

	if !enabled {
		tm.cache = make(map[string]*CachedTemplate)
	}
}

// AddValidator adds a template validator
func (tm *TemplateManager) AddValidator(validator TemplateValidator) {
	tm.validators = append(tm.validators, validator)
}

// AddPreprocessor adds a template preprocessor
func (tm *TemplateManager) AddPreprocessor(preprocessor TemplatePreprocessor) {
	tm.preprocessors = append(tm.preprocessors, preprocessor)
}

// Built-in validators

// validateSyntax validates template syntax
func (tm *TemplateManager) validateSyntax(name, content string, metadata *TemplateMetadata) error {
	if content == "" {
		return fmt.Errorf("template content cannot be empty")
	}

	// Basic syntax validation (this would be more comprehensive in a real implementation)
	if strings.Count(content, "{{") != strings.Count(content, "}}") {
		return fmt.Errorf("unmatched template braces in template")
	}

	return nil
}

// validateVariables validates template variables
func (tm *TemplateManager) validateVariables(name, content string, metadata *TemplateMetadata) error {
	// Extract variables from content (simple implementation)
	// In a real implementation, this would use proper template parsing
	// This is a simplified version for demonstration

	if metadata == nil {
		return nil
	}

	// Check if required variables are mentioned in metadata
	for _, varInfo := range metadata.Variables {
		if varInfo.Required {
			placeholder := fmt.Sprintf("{{.%s}}", varInfo.Name)
			if !strings.Contains(content, placeholder) && !strings.Contains(content, fmt.Sprintf("{{ .%s }}", varInfo.Name)) {
				return fmt.Errorf("required variable '%s' not found in template", varInfo.Name)
			}
		}
	}

	return nil
}

// validateMetadata validates template metadata
func (tm *TemplateManager) validateMetadata(name, content string, metadata *TemplateMetadata) error {
	if metadata == nil {
		return nil
	}

	if metadata.Name != name {
		return fmt.Errorf("metadata name '%s' does not match template name '%s'", metadata.Name, name)
	}

	if metadata.Version == "" {
		return fmt.Errorf("template version is required")
	}

	return nil
}

// Helper methods

// updateUsageStats updates usage statistics for a template
func (tm *TemplateManager) updateUsageStats(name string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if metadata, exists := tm.templates[name]; exists {
		metadata.UsageCount++
	}
}

// getCachedTemplate retrieves a template from cache
func (tm *TemplateManager) getCachedTemplate(name string) *CachedTemplate {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if !tm.cacheEnabled {
		return nil
	}

	cached, exists := tm.cache[name]
	if !exists {
		return nil
	}

	// Check if cache entry has expired
	if time.Since(cached.CompiledAt) > tm.cacheTimeout {
		delete(tm.cache, name)
		return nil
	}

	return cached
}

// TemplateStats contains statistics about a template
type TemplateStats struct {
	Name              string     `json:"name"`
	UsageCount        int64      `json:"usage_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Version           string     `json:"version"`
	CacheStatus       string     `json:"cache_status"`
	LastCacheTime     *time.Time `json:"last_cache_time,omitempty"`
	CacheUsageCount   int64      `json:"cache_usage_count,omitempty"`
}

// Hub integration methods

// Templates returns the template manager for the hub
func (h *Hub) Templates() *TemplateManager {
	if h.templateManager == nil {
		h.templateManager = NewTemplateManager(h)
	}
	return h.templateManager
}

// We need to add this field to the Hub struct
// This would go in hub.go, but we'll add the method that creates it lazily
var templateManagerField *TemplateManager

// RegisterTemplate registers a template with the hub's template manager
func (h *Hub) RegisterTemplate(name, content string, metadata *TemplateMetadata) error {
	return h.Templates().RegisterTemplate(name, content, metadata)
}

// RenderTemplate renders a template using the hub's template manager
func (h *Hub) RenderTemplate(ctx context.Context, name string, variables map[string]interface{}) (string, error) {
	return h.Templates().RenderTemplate(ctx, name, variables)
}

// SendWithTemplateManager sends a message using the enhanced template manager
func (h *Hub) SendWithTemplateManager(ctx context.Context, templateName string, variables map[string]interface{}, targets []notifiers.Target, options *Options) ([]*notifiers.SendResult, error) {
	// Render the template
	content, err := h.Templates().RenderTemplate(ctx, templateName, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Extract title and body (simple implementation - assumes template format)
	lines := strings.Split(content, "\n")
	title := ""
	body := content

	if len(lines) > 0 && strings.HasPrefix(lines[0], "TITLE:") {
		title = strings.TrimPrefix(lines[0], "TITLE:")
		title = strings.TrimSpace(title)
		if len(lines) > 1 {
			body = strings.Join(lines[1:], "\n")
		} else {
			body = ""
		}
	}

	// Create message
	messageBuilder := NewMessage().Title(title).Body(body)
	for _, target := range targets {
		messageBuilder.Target(target)
	}

	// Add original variables as message variables
	for k, v := range variables {
		messageBuilder.Variable(k, v)
	}

	message := messageBuilder.Build()
	return h.Send(ctx, message, options)
}