package config

import (
	"sort"

	"github.com/kart-io/notifyhub/notifiers"
)

// ================================
// 路由引擎
// ================================

// RoutingEngine processes routing rules
type RoutingEngine struct {
	rules []RoutingRule
}

// NewRoutingEngine creates a new routing engine
func NewRoutingEngine(rules []RoutingRule) *RoutingEngine {
	var enabledRules []RoutingRule
	for _, rule := range rules {
		if rule.Enabled {
			enabledRules = append(enabledRules, rule)
		}
	}

	// Sort rules by priority (descending - higher values first)
	sort.Slice(enabledRules, func(i, j int) bool {
		return enabledRules[i].Priority > enabledRules[j].Priority
	})

	return &RoutingEngine{rules: enabledRules}
}

// ProcessMessage processes a message through routing rules
func (r *RoutingEngine) ProcessMessage(message *notifiers.Message) *notifiers.Message {
	processed := *message // Copy

	// Process all matching rules in priority order
	// Rules are already sorted by priority (highest first)
	for _, rule := range r.rules {
		if r.matchesRule(message, rule) {
			r.applyRule(&processed, rule)
			// Continue processing other rules unless this rule is exclusive
			// For now, we still break after first match for backward compatibility
			break
		}
	}

	return &processed
}

// matchesRule checks if a message matches a routing rule
func (r *RoutingEngine) matchesRule(message *notifiers.Message, rule RoutingRule) bool {
	// Check priority
	if len(rule.Conditions.Priority) > 0 {
		found := false
		for _, p := range rule.Conditions.Priority {
			if message.Priority == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check metadata
	for key, expectedValue := range rule.Conditions.Metadata {
		if actualValue, exists := message.Metadata[key]; !exists || actualValue != expectedValue {
			return false
		}
	}

	// Check message type (could be extended)
	if len(rule.Conditions.MessageType) > 0 {
		messageType, exists := message.Metadata["type"]
		if !exists {
			return false
		}
		found := false
		for _, t := range rule.Conditions.MessageType {
			if messageType == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// applyRule applies a routing rule to a message
func (r *RoutingEngine) applyRule(message *notifiers.Message, rule RoutingRule) {
	for _, action := range rule.Actions {
		if action.Type == "route" && len(action.Platforms) > 0 {
			// Filter targets based on platform restrictions
			var filteredTargets []notifiers.Target
			for _, target := range message.Targets {
				for _, platform := range action.Platforms {
					if target.Platform == "" || target.Platform == platform {
						if target.Platform == "" {
							target.Platform = platform
						}
						filteredTargets = append(filteredTargets, target)
						break
					}
				}
			}
			message.Targets = filteredTargets
		}
	}
}

// AddRule adds a new routing rule and maintains priority order
func (r *RoutingEngine) AddRule(rule RoutingRule) {
	if rule.Enabled {
		r.rules = append(r.rules, rule)
		// Re-sort rules by priority
		sort.Slice(r.rules, func(i, j int) bool {
			return r.rules[i].Priority > r.rules[j].Priority
		})
	}
}

// RemoveRule removes a routing rule by name
func (r *RoutingEngine) RemoveRule(name string) {
	for i, rule := range r.rules {
		if rule.Name == name {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			break
		}
	}
}

// GetRules returns all active routing rules
func (r *RoutingEngine) GetRules() []RoutingRule {
	return r.rules
}

// ================================
// Rule Builder Helpers
// ================================

// NewRoutingRule creates a new routing rule builder
func NewRoutingRule(name string) *RoutingRuleBuilder {
	return &RoutingRuleBuilder{
		rule: RoutingRule{
			Name:       name,
			Priority:   10, // Default priority
			Enabled:    true,
			Conditions: RuleConditions{},
			Actions:    []RuleAction{},
		},
	}
}

// RoutingRuleBuilder provides a fluent interface for building routing rules
type RoutingRuleBuilder struct {
	rule RoutingRule
}

// Priority sets the rule priority (higher values = higher priority)
func (b *RoutingRuleBuilder) Priority(priority int) *RoutingRuleBuilder {
	b.rule.Priority = priority
	return b
}

// Enabled sets whether the rule is enabled
func (b *RoutingRuleBuilder) Enabled(enabled bool) *RoutingRuleBuilder {
	b.rule.Enabled = enabled
	return b
}

// WithPriority adds priority conditions
func (b *RoutingRuleBuilder) WithPriority(priorities ...int) *RoutingRuleBuilder {
	b.rule.Conditions.Priority = append(b.rule.Conditions.Priority, priorities...)
	return b
}

// WithMessageType adds message type conditions
func (b *RoutingRuleBuilder) WithMessageType(types ...string) *RoutingRuleBuilder {
	b.rule.Conditions.MessageType = append(b.rule.Conditions.MessageType, types...)
	return b
}

// WithMetadata adds metadata conditions
func (b *RoutingRuleBuilder) WithMetadata(key, value string) *RoutingRuleBuilder {
	if b.rule.Conditions.Metadata == nil {
		b.rule.Conditions.Metadata = make(map[string]string)
	}
	b.rule.Conditions.Metadata[key] = value
	return b
}

// RouteTo adds a routing action to specified platforms
func (b *RoutingRuleBuilder) RouteTo(platforms ...string) *RoutingRuleBuilder {
	b.rule.Actions = append(b.rule.Actions, RuleAction{
		Type:      "route",
		Platforms: platforms,
	})
	return b
}

// Build returns the constructed routing rule
func (b *RoutingRuleBuilder) Build() RoutingRule {
	return b.rule
}
