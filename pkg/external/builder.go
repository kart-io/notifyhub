// Package external provides simplified platform building for external NotifyHub extensions
package external

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/platform"
	"github.com/kart-io/notifyhub/pkg/target"
)

// SimpleSender is the minimal interface external platforms need to implement
// 外部平台只需要实现这一个方法
type SimpleSender interface {
	// Send sends a message to a target and returns an error if failed
	Send(ctx context.Context, message string, target string) error
}

// SendResult contains the result of a send operation
type SendResult struct {
	MessageID string            `json:"message_id"`
	Cost      float64           `json:"cost"`
	Metadata  map[string]string `json:"metadata"`
}

// AdvancedSender is an optional interface for more control
type AdvancedSender interface {
	SimpleSender
	SendWithResult(ctx context.Context, message string, target string) (*SendResult, error)
	ValidateTarget(target string) error
	GetQuota() (remaining, total int)
	Close() error
}

// PlatformBuilder provides a fluent interface for building external platforms
type PlatformBuilder struct {
	name                string
	sender              SimpleSender
	supportedTypes      []string
	supportedFormats    []string
	maxMessageSize      int
	supportsScheduling  bool
	supportsAttachments bool
	requiredSettings    []string

	// Optional components
	rateLimiter      *RateLimiter
	templateEngine   *TemplateEngine
	targetValidator  func(string) error
	messageFormatter func(*message.Message) string
}

// NewPlatform starts building a new external platform
func NewPlatform(name string, sender SimpleSender) *PlatformBuilder {
	return &PlatformBuilder{
		name:                name,
		sender:              sender,
		supportedTypes:      []string{"default"},
		supportedFormats:    []string{"text"},
		maxMessageSize:      1000,
		supportsScheduling:  false,
		supportsAttachments: false,
		requiredSettings:    []string{},
	}
}

// WithTargetTypes sets supported target types
func (b *PlatformBuilder) WithTargetTypes(types ...string) *PlatformBuilder {
	b.supportedTypes = types
	return b
}

// WithFormats sets supported message formats
func (b *PlatformBuilder) WithFormats(formats ...string) *PlatformBuilder {
	b.supportedFormats = formats
	return b
}

// WithMaxMessageSize sets the maximum message size
func (b *PlatformBuilder) WithMaxMessageSize(size int) *PlatformBuilder {
	b.maxMessageSize = size
	return b
}

// WithScheduling enables scheduling support
func (b *PlatformBuilder) WithScheduling() *PlatformBuilder {
	b.supportsScheduling = true
	return b
}

// WithAttachments enables attachment support
func (b *PlatformBuilder) WithAttachments() *PlatformBuilder {
	b.supportsAttachments = true
	return b
}

// WithRequiredSettings sets required configuration settings
func (b *PlatformBuilder) WithRequiredSettings(settings ...string) *PlatformBuilder {
	b.requiredSettings = settings
	return b
}

// WithRateLimit adds rate limiting (per hour, per day)
func (b *PlatformBuilder) WithRateLimit(maxPerHour, maxPerDay int) *PlatformBuilder {
	b.rateLimiter = NewRateLimiter(maxPerHour, maxPerDay)
	return b
}

// WithTemplates adds template support
func (b *PlatformBuilder) WithTemplates(templates map[string]string) *PlatformBuilder {
	b.templateEngine = NewTemplateEngine(templates)
	return b
}

// WithTargetValidator adds custom target validation
func (b *PlatformBuilder) WithTargetValidator(validator func(string) error) *PlatformBuilder {
	b.targetValidator = validator
	return b
}

// WithMessageFormatter adds custom message formatting
func (b *PlatformBuilder) WithMessageFormatter(formatter func(*message.Message) string) *PlatformBuilder {
	b.messageFormatter = formatter
	return b
}

// Build creates the final platform instance
func (b *PlatformBuilder) Build() platform.Platform {
	return &builtPlatform{
		name:                b.name,
		sender:              b.sender,
		supportedTypes:      b.supportedTypes,
		supportedFormats:    b.supportedFormats,
		maxMessageSize:      b.maxMessageSize,
		supportsScheduling:  b.supportsScheduling,
		supportsAttachments: b.supportsAttachments,
		requiredSettings:    b.requiredSettings,
		rateLimiter:         b.rateLimiter,
		templateEngine:      b.templateEngine,
		targetValidator:     b.targetValidator,
		messageFormatter:    b.messageFormatter,
	}
}

// builtPlatform implements the platform.Platform interface
type builtPlatform struct {
	name                string
	sender              SimpleSender
	supportedTypes      []string
	supportedFormats    []string
	maxMessageSize      int
	supportsScheduling  bool
	supportsAttachments bool
	requiredSettings    []string

	rateLimiter      *RateLimiter
	templateEngine   *TemplateEngine
	targetValidator  func(string) error
	messageFormatter func(*message.Message) string
}

func (p *builtPlatform) Name() string {
	return p.name
}

func (p *builtPlatform) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 p.name,
		SupportedTargetTypes: p.supportedTypes,
		SupportedFormats:     p.supportedFormats,
		MaxMessageSize:       p.maxMessageSize,
		SupportsScheduling:   p.supportsScheduling,
		SupportsAttachments:  p.supportsAttachments,
		RequiredSettings:     p.requiredSettings,
	}
}

func (p *builtPlatform) ValidateTarget(target target.Target) error {
	// Check supported types
	supported := false
	for _, t := range p.supportedTypes {
		if t == target.Type || t == "default" {
			supported = true
			break
		}
	}
	if !supported {
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}

	// Custom validation
	if p.targetValidator != nil {
		return p.targetValidator(target.Value)
	}

	return nil
}

func (p *builtPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		result := &platform.SendResult{Target: target}

		// Validate target
		if err := p.ValidateTarget(target); err != nil {
			result.Error = err
			results[i] = result
			continue
		}

		// Check rate limit
		if p.rateLimiter != nil && !p.rateLimiter.Allow(target.Value) {
			result.Error = fmt.Errorf("rate limit exceeded for %s", target.Value)
			results[i] = result
			continue
		}

		// Prepare message content
		content := p.prepareMessage(msg)

		// Check message size
		if len([]rune(content)) > p.maxMessageSize {
			result.Error = fmt.Errorf("message too long: %d characters (max %d)",
				len([]rune(content)), p.maxMessageSize)
			results[i] = result
			continue
		}

		// Send message
		if advancedSender, ok := p.sender.(AdvancedSender); ok {
			// Use advanced sender if available
			sendResult, err := advancedSender.SendWithResult(ctx, content, target.Value)
			if err != nil {
				result.Error = err
				result.Response = err.Error()
			} else {
				result.Success = true
				result.MessageID = sendResult.MessageID
				result.Response = fmt.Sprintf("Cost: %.4f", sendResult.Cost)
			}
		} else {
			// Use simple sender
			err := p.sender.Send(ctx, content, target.Value)
			if err != nil {
				result.Error = err
				result.Response = err.Error()
			} else {
				result.Success = true
				result.MessageID = fmt.Sprintf("%s_%d", p.name, time.Now().Unix())
				result.Response = "Message sent successfully"
			}
		}

		results[i] = result
	}

	return results, nil
}

func (p *builtPlatform) prepareMessage(msg *message.Message) string {
	// Use custom formatter if provided
	if p.messageFormatter != nil {
		return p.messageFormatter(msg)
	}

	// Use template engine if available and template specified
	if p.templateEngine != nil {
		if templateName, exists := msg.Metadata["template"]; exists {
			if content, ok := p.templateEngine.Render(templateName.(string), msg.Variables); ok {
				return content
			}
		}
	}

	// Default formatting
	content := msg.Body
	if msg.Title != "" {
		content = fmt.Sprintf("%s: %s", msg.Title, msg.Body)
	}

	return content
}

func (p *builtPlatform) IsHealthy(ctx context.Context) error {
	// Check quota if advanced sender
	if advancedSender, ok := p.sender.(AdvancedSender); ok {
		remaining, total := advancedSender.GetQuota()
		if total > 0 && remaining <= 0 {
			return fmt.Errorf("quota exhausted: %d/%d remaining", remaining, total)
		}
	}
	return nil
}

func (p *builtPlatform) Close() error {
	if advancedSender, ok := p.sender.(AdvancedSender); ok {
		return advancedSender.Close()
	}
	return nil
}

// Helper components

// RateLimiter provides simple rate limiting
type RateLimiter struct {
	maxPerHour int
	maxPerDay  int
	counters   map[string]*counter
	mu         sync.RWMutex
}

type counter struct {
	hourlyCount int
	dailyCount  int
	hourlyReset time.Time
	dailyReset  time.Time
	mu          sync.Mutex
}

func NewRateLimiter(maxPerHour, maxPerDay int) *RateLimiter {
	return &RateLimiter{
		maxPerHour: maxPerHour,
		maxPerDay:  maxPerDay,
		counters:   make(map[string]*counter),
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	c, exists := rl.counters[key]
	if !exists {
		c = &counter{
			hourlyReset: time.Now().Add(time.Hour),
			dailyReset:  time.Now().Add(24 * time.Hour),
		}
		rl.counters[key] = c
	}
	rl.mu.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Reset counters if needed
	if now.After(c.hourlyReset) {
		c.hourlyCount = 0
		c.hourlyReset = now.Add(time.Hour)
	}
	if now.After(c.dailyReset) {
		c.dailyCount = 0
		c.dailyReset = now.Add(24 * time.Hour)
	}

	// Check limits
	if rl.maxPerHour > 0 && c.hourlyCount >= rl.maxPerHour {
		return false
	}
	if rl.maxPerDay > 0 && c.dailyCount >= rl.maxPerDay {
		return false
	}

	// Increment counters
	c.hourlyCount++
	c.dailyCount++
	return true
}

// TemplateEngine provides simple template rendering
type TemplateEngine struct {
	templates map[string]string
}

func NewTemplateEngine(templates map[string]string) *TemplateEngine {
	return &TemplateEngine{
		templates: templates,
	}
}

func (te *TemplateEngine) Render(templateName string, variables map[string]interface{}) (string, bool) {
	template, exists := te.templates[templateName]
	if !exists {
		return "", false
	}

	content := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, fmt.Sprintf("%v", value))
	}

	return content, true
}

// Utility functions for creating targets
func CreateTarget(targetType, value string) target.Target {
	return target.Target{
		Type:  targetType,
		Value: value,
	}
}

func CreateDefaultTarget(value string) target.Target {
	return target.Target{
		Type:  "default",
		Value: value,
	}
}
