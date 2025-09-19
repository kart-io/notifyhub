package targeting

import (
	"context"
	"fmt"
	"strings"

	"github.com/kart-io/notifyhub/core/sending"
)

// Resolver resolves target expressions into concrete targets
type Resolver struct {
	providers map[string]Provider
}

// Provider defines the interface for target providers
type Provider interface {
	ResolveTargets(ctx context.Context, expression string) ([]sending.Target, error)
	SupportedExpressions() []string
}

// NewResolver creates a new target resolver
func NewResolver() *Resolver {
	return &Resolver{
		providers: make(map[string]Provider),
	}
}

// RegisterProvider registers a target provider
func (r *Resolver) RegisterProvider(name string, provider Provider) {
	r.providers[name] = provider
}

// ResolveTargets resolves target expressions into concrete targets
func (r *Resolver) ResolveTargets(ctx context.Context, expressions []string) ([]sending.Target, error) {
	var allTargets []sending.Target

	for _, expression := range expressions {
		targets, err := r.resolveExpression(ctx, expression)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve expression '%s': %w", expression, err)
		}
		allTargets = append(allTargets, targets...)
	}

	// Deduplicate targets
	return r.deduplicateTargets(allTargets), nil
}

// resolveExpression resolves a single target expression
func (r *Resolver) resolveExpression(ctx context.Context, expression string) ([]sending.Target, error) {
	// Parse expression format: provider:expression
	parts := strings.SplitN(expression, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid expression format, expected 'provider:expression', got: %s", expression)
	}

	providerName := parts[0]
	providerExpression := parts[1]

	provider, exists := r.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	return provider.ResolveTargets(ctx, providerExpression)
}

// deduplicateTargets removes duplicate targets
func (r *Resolver) deduplicateTargets(targets []sending.Target) []sending.Target {
	seen := make(map[string]bool)
	var unique []sending.Target

	for _, target := range targets {
		key := target.String()
		if !seen[key] {
			seen[key] = true
			unique = append(unique, target)
		}
	}

	return unique
}

// StaticProvider provides static target resolution
type StaticProvider struct {
	targets map[string][]sending.Target
}

// NewStaticProvider creates a new static provider
func NewStaticProvider() *StaticProvider {
	return &StaticProvider{
		targets: make(map[string][]sending.Target),
	}
}

// AddTargetGroup adds a group of targets with a name
func (p *StaticProvider) AddTargetGroup(name string, targets []sending.Target) {
	p.targets[name] = targets
}

// ResolveTargets resolves targets from static configuration
func (p *StaticProvider) ResolveTargets(ctx context.Context, expression string) ([]sending.Target, error) {
	targets, exists := p.targets[expression]
	if !exists {
		return nil, fmt.Errorf("target group not found: %s", expression)
	}

	// Clone targets to avoid modification
	result := make([]sending.Target, len(targets))
	copy(result, targets)
	return result, nil
}

// SupportedExpressions returns supported expressions
func (p *StaticProvider) SupportedExpressions() []string {
	expressions := make([]string, 0, len(p.targets))
	for name := range p.targets {
		expressions = append(expressions, name)
	}
	return expressions
}

// DirectProvider provides direct target creation from simple formats
type DirectProvider struct{}

// NewDirectProvider creates a new direct provider
func NewDirectProvider() *DirectProvider {
	return &DirectProvider{}
}

// ResolveTargets resolves targets from direct expressions
// Supports formats like: email:user@example.com, feishu:group:abc123, feishu:user:xyz789
func (p *DirectProvider) ResolveTargets(ctx context.Context, expression string) ([]sending.Target, error) {
	parts := strings.Split(expression, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid direct expression format: %s", expression)
	}

	platform := parts[0]

	switch platform {
	case "email":
		if len(parts) != 2 {
			return nil, fmt.Errorf("email format should be 'email:address': %s", expression)
		}
		return []sending.Target{
			sending.NewTarget(sending.TargetTypeEmail, parts[1], "email"),
		}, nil

	case "feishu":
		if len(parts) != 3 {
			return nil, fmt.Errorf("feishu format should be 'feishu:type:value': %s", expression)
		}
		targetType := sending.TargetType(parts[1])
		value := parts[2]
		return []sending.Target{
			sending.NewTarget(targetType, value, "feishu"),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// SupportedExpressions returns supported expression patterns
func (p *DirectProvider) SupportedExpressions() []string {
	return []string{
		"email:<address>",
		"feishu:user:<user_id>",
		"feishu:group:<group_id>",
		"feishu:channel:<channel_id>",
	}
}
