package routing

import (
	"fmt"
	"sort"
)

// Engine processes routing rules to determine message targets
// This implements the proposal's routing engine without platform-specific dependencies
type Engine struct {
	rules []Rule
}

// NewEngine creates a new routing engine
func NewEngine(rules []Rule) *Engine {
	var enabledRules []Rule
	for _, rule := range rules {
		if rule.Enabled {
			enabledRules = append(enabledRules, rule)
		}
	}

	// Sort rules by priority (descending - higher values first)
	sort.Slice(enabledRules, func(i, j int) bool {
		return enabledRules[i].Priority > enabledRules[j].Priority
	})

	return &Engine{rules: enabledRules}
}

// ProcessMessage processes a message through routing rules
// Using map[string]interface{} to avoid dependency on specific message types
func (e *Engine) ProcessMessage(message map[string]interface{}) (map[string]interface{}, error) {
	if message == nil {
		return nil, fmt.Errorf("message is nil")
	}

	// Create a copy to avoid modifying the original
	processed := make(map[string]interface{})
	for k, v := range message {
		processed[k] = v
	}

	// Process all matching rules in priority order
	for _, rule := range e.rules {
		if e.matchesRule(message, rule) {
			e.applyRule(processed, rule)
			// Continue processing other rules unless this rule is exclusive
			if rule.StopProcessing {
				break
			}
		}
	}

	return processed, nil
}

// matchesRule checks if a message matches a routing rule
func (e *Engine) matchesRule(message map[string]interface{}, rule Rule) bool {
	// Check priority match
	if len(rule.Conditions.Priorities) > 0 {
		priority, ok := message["priority"].(int)
		if !ok {
			return false
		}

		priorityMatch := false
		for _, p := range rule.Conditions.Priorities {
			if priority == p {
				priorityMatch = true
				break
			}
		}
		if !priorityMatch {
			return false
		}
	}

	// Check metadata match
	if len(rule.Conditions.Metadata) > 0 {
		metadata, ok := message["metadata"].(map[string]string)
		if !ok {
			return false
		}

		for key, expectedValue := range rule.Conditions.Metadata {
			if actualValue, exists := metadata[key]; !exists || actualValue != expectedValue {
				return false
			}
		}
	}

	// Check template match
	if rule.Conditions.Template != "" {
		template, ok := message["template"].(string)
		if !ok || template != rule.Conditions.Template {
			return false
		}
	}

	// Check platform match
	if rule.Conditions.Platform != "" {
		platform, ok := message["platform"].(string)
		if !ok || platform != rule.Conditions.Platform {
			return false
		}
	}

	return true
}

// applyRule applies a routing rule to a message
func (e *Engine) applyRule(message map[string]interface{}, rule Rule) {
	// Get or create targets array
	var targets []interface{}
	if existingTargets, ok := message["targets"].([]interface{}); ok {
		targets = existingTargets
	} else {
		targets = make([]interface{}, 0)
	}

	// Add rule targets to message
	for _, target := range rule.Actions.Targets {
		targetMap := map[string]interface{}{
			"type":     target.Type,
			"value":    target.Value,
			"platform": target.Platform,
		}
		if len(target.Metadata) > 0 {
			targetMap["metadata"] = target.Metadata
		}
		targets = append(targets, targetMap)
	}

	message["targets"] = targets

	// Apply metadata changes
	if len(rule.Actions.AddMetadata) > 0 {
		metadata, ok := message["metadata"].(map[string]string)
		if !ok {
			metadata = make(map[string]string)
		}

		for key, value := range rule.Actions.AddMetadata {
			metadata[key] = value
		}

		message["metadata"] = metadata
	}

	// Apply priority override
	if rule.Actions.SetPriority > 0 {
		message["priority"] = rule.Actions.SetPriority
	}

	// Apply platform override
	if rule.Actions.SetPlatform != "" {
		message["platform"] = rule.Actions.SetPlatform
	}
}

// GetActiveRules returns all active rules
func (e *Engine) GetActiveRules() []Rule {
	return e.rules
}

// FindRule finds a rule by name
func (e *Engine) FindRule(name string) (*Rule, bool) {
	for _, rule := range e.rules {
		if rule.Name == name {
			return &rule, true
		}
	}
	return nil, false
}

// Matcher provides rule matching functionality
type Matcher struct {
	engine *Engine
}

// NewMatcher creates a new rule matcher
func NewMatcher(engine *Engine) *Matcher {
	return &Matcher{engine: engine}
}

// Match returns all rules that match the given message
func (m *Matcher) Match(message map[string]interface{}) []Rule {
	var matches []Rule
	for _, rule := range m.engine.rules {
		if m.engine.matchesRule(message, rule) {
			matches = append(matches, rule)
		}
	}
	return matches
}

// MatchFirst returns the first rule that matches the given message
func (m *Matcher) MatchFirst(message map[string]interface{}) (*Rule, bool) {
	for _, rule := range m.engine.rules {
		if m.engine.matchesRule(message, rule) {
			return &rule, true
		}
	}
	return nil, false
}
