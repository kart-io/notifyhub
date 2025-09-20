package api

import (
	"strings"

	"github.com/kart-io/notifyhub/core/message"
)

// TargetDetectionStrategy defines how to detect target types from string input
type TargetDetectionStrategy interface {
	Detect(input string) (message.TargetType, string, string) // returns type, value, platform
	CanHandle(input string) bool
}

// EmailDetectionStrategy detects email targets
type EmailDetectionStrategy struct{}

func (s *EmailDetectionStrategy) CanHandle(input string) bool {
	// Must contain @ and . but should not start with @ (that's a mention)
	return strings.Contains(input, "@") && strings.Contains(input, ".") && !strings.HasPrefix(input, "@")
}

func (s *EmailDetectionStrategy) Detect(input string) (message.TargetType, string, string) {
	return message.TargetTypeEmail, input, "email"
}

// UserMentionDetectionStrategy detects user mentions (@user)
type UserMentionDetectionStrategy struct{}

func (s *UserMentionDetectionStrategy) CanHandle(input string) bool {
	return strings.HasPrefix(input, "@")
}

func (s *UserMentionDetectionStrategy) Detect(input string) (message.TargetType, string, string) {
	value := strings.TrimPrefix(input, "@")
	return message.TargetTypeUser, value, ""
}

// ChannelDetectionStrategy detects channel references (#channel)
type ChannelDetectionStrategy struct{}

func (s *ChannelDetectionStrategy) CanHandle(input string) bool {
	return strings.HasPrefix(input, "#")
}

func (s *ChannelDetectionStrategy) Detect(input string) (message.TargetType, string, string) {
	value := strings.TrimPrefix(input, "#")
	return message.TargetTypeChannel, value, ""
}

// SlackTargetDetectionStrategy detects Slack-specific targets
type SlackTargetDetectionStrategy struct{}

func (s *SlackTargetDetectionStrategy) CanHandle(input string) bool {
	return len(input) > 0
}

func (s *SlackTargetDetectionStrategy) Detect(input string) (message.TargetType, string, string) {
	switch {
	case strings.HasPrefix(input, "#"):
		return message.TargetTypeChannel, input[1:], "slack"
	case strings.HasPrefix(input, "@"):
		return message.TargetTypeUser, input[1:], "slack"
	default:
		return message.TargetTypeChannel, input, "slack"
	}
}

// FeishuTargetDetectionStrategy detects Feishu-specific targets
type FeishuTargetDetectionStrategy struct{}

func (s *FeishuTargetDetectionStrategy) CanHandle(input string) bool {
	return len(input) > 0
}

func (s *FeishuTargetDetectionStrategy) Detect(input string) (message.TargetType, string, string) {
	switch {
	case strings.HasPrefix(input, "#"):
		return message.TargetTypeChannel, input[1:], "feishu"
	case strings.HasPrefix(input, "@"):
		return message.TargetTypeUser, input[1:], "feishu"
	default:
		return message.TargetTypeGroup, input, "feishu"
	}
}

// DefaultDetectionStrategy fallback strategy for plain text
type DefaultDetectionStrategy struct{}

func (s *DefaultDetectionStrategy) CanHandle(input string) bool {
	return true // Always can handle as fallback
}

func (s *DefaultDetectionStrategy) Detect(input string) (message.TargetType, string, string) {
	return message.TargetTypeUser, input, ""
}

// TargetDetector provides intelligent target detection capabilities
type TargetDetector struct {
	strategies []TargetDetectionStrategy
}

// NewTargetDetector creates a new target detector with default strategies
func NewTargetDetector() *TargetDetector {
	return &TargetDetector{
		strategies: []TargetDetectionStrategy{
			&EmailDetectionStrategy{},
			&UserMentionDetectionStrategy{},
			&ChannelDetectionStrategy{},
			&DefaultDetectionStrategy{}, // Must be last as fallback
		},
	}
}

// NewSlackTargetDetector creates a detector optimized for Slack
func NewSlackTargetDetector() *TargetDetector {
	return &TargetDetector{
		strategies: []TargetDetectionStrategy{
			&SlackTargetDetectionStrategy{},
			&DefaultDetectionStrategy{},
		},
	}
}

// NewFeishuTargetDetector creates a detector optimized for Feishu
func NewFeishuTargetDetector() *TargetDetector {
	return &TargetDetector{
		strategies: []TargetDetectionStrategy{
			&FeishuTargetDetectionStrategy{},
			&DefaultDetectionStrategy{},
		},
	}
}

// DetectTarget analyzes input and returns appropriate target
func (d *TargetDetector) DetectTarget(input string) message.Target {
	for _, strategy := range d.strategies {
		if strategy.CanHandle(input) {
			targetType, value, platform := strategy.Detect(input)
			return message.NewTarget(targetType, value, platform)
		}
	}

	// Fallback (should never reach here with DefaultDetectionStrategy)
	return message.NewTarget(message.TargetTypeUser, input, "")
}

// DetectTargets processes multiple inputs and returns targets
func (d *TargetDetector) DetectTargets(inputs ...string) []message.Target {
	targets := make([]message.Target, 0, len(inputs))
	for _, input := range inputs {
		targets = append(targets, d.DetectTarget(input))
	}
	return targets
}

// AddStrategy adds a custom detection strategy
func (d *TargetDetector) AddStrategy(strategy TargetDetectionStrategy) {
	// Insert before the default strategy (which should be last)
	if len(d.strategies) > 0 {
		d.strategies = append(d.strategies[:len(d.strategies)-1], strategy, d.strategies[len(d.strategies)-1])
	} else {
		d.strategies = append(d.strategies, strategy)
	}
}

// SetPlatformForTargets updates platform for targets that don't have one
func (d *TargetDetector) SetPlatformForTargets(targets []message.Target, platforms ...string) []message.Target {
	if len(platforms) == 0 {
		return targets
	}

	defaultPlatform := platforms[0]
	for i := range targets {
		if targets[i].Platform == "" {
			targets[i].Platform = defaultPlatform
		}
	}
	return targets
}