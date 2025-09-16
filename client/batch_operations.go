package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/kart-io/notifyhub/notifiers"
)

// TargetList provides a fluent API for building target lists
type TargetList struct {
	targets []notifiers.Target
}

// NewTargetList creates a new target list builder
func NewTargetList() *TargetList {
	return &TargetList{
		targets: make([]notifiers.Target, 0),
	}
}

// AddEmails adds one or more email targets
func (tl *TargetList) AddEmails(emails ...string) *TargetList {
	for _, email := range emails {
		tl.targets = append(tl.targets, notifiers.Target{
			Type:  notifiers.TargetTypeEmail,
			Value: email,
		})
	}
	return tl
}

// AddFeishuGroups adds one or more Feishu group targets
func (tl *TargetList) AddFeishuGroups(groups ...string) *TargetList {
	for _, group := range groups {
		tl.targets = append(tl.targets, notifiers.Target{
			Type:     notifiers.TargetTypeGroup,
			Value:    group,
			Platform: "feishu",
		})
	}
	return tl
}

// AddFeishuUsers adds one or more Feishu user targets
func (tl *TargetList) AddFeishuUsers(users ...string) *TargetList {
	for _, user := range users {
		tl.targets = append(tl.targets, notifiers.Target{
			Type:     notifiers.TargetTypeUser,
			Value:    user,
			Platform: "feishu",
		})
	}
	return tl
}

// AddTargets adds custom targets
func (tl *TargetList) AddTargets(targets ...notifiers.Target) *TargetList {
	tl.targets = append(tl.targets, targets...)
	return tl
}

// AddFromStrings adds targets from string format (type:value@platform)
func (tl *TargetList) AddFromStrings(targets ...string) *TargetList {
	for _, target := range targets {
		if parsed, err := parseTargetString(target); err == nil {
			tl.targets = append(tl.targets, parsed)
		}
	}
	return tl
}

// Build returns the list of targets
func (tl *TargetList) Build() []notifiers.Target {
	return tl.targets
}

// Count returns the number of targets
func (tl *TargetList) Count() int {
	return len(tl.targets)
}

// String returns a string representation of the target list
func (tl *TargetList) String() string {
	if len(tl.targets) == 0 {
		return "empty target list"
	}

	var parts []string
	for _, target := range tl.targets {
		if target.Platform != "" {
			parts = append(parts, fmt.Sprintf("%s:%s@%s", target.Type, target.Value, target.Platform))
		} else {
			parts = append(parts, fmt.Sprintf("%s:%s", target.Type, target.Value))
		}
	}
	return fmt.Sprintf("targets[%d]: %s", len(parts), strings.Join(parts, ", "))
}

// BatchMessage represents a message in a batch
type BatchMessage struct {
	Message   *notifiers.Message
	Targets   []notifiers.Target
	Options   *Options
}

// EnhancedBatchBuilder provides advanced batch operations
type EnhancedBatchBuilder struct {
	hub      *Hub
	messages []BatchMessage
	defaultTargets []notifiers.Target
	defaultOptions *Options
}

// NewEnhancedBatch creates a new enhanced batch builder
func (h *Hub) NewEnhancedBatch() *EnhancedBatchBuilder {
	return &EnhancedBatchBuilder{
		hub:      h,
		messages: make([]BatchMessage, 0),
	}
}

// WithDefaultTargets sets default targets for all messages in the batch
func (bb *EnhancedBatchBuilder) WithDefaultTargets(targets ...notifiers.Target) *EnhancedBatchBuilder {
	bb.defaultTargets = targets
	return bb
}

// WithDefaultTargetList sets default targets using TargetList
func (bb *EnhancedBatchBuilder) WithDefaultTargetList(targetList *TargetList) *EnhancedBatchBuilder {
	bb.defaultTargets = targetList.Build()
	return bb
}

// WithDefaultOptions sets default options for all messages in the batch
func (bb *EnhancedBatchBuilder) WithDefaultOptions(options *Options) *EnhancedBatchBuilder {
	bb.defaultOptions = options
	return bb
}

// AddMessage adds a message to the batch
func (bb *EnhancedBatchBuilder) AddMessage(message *notifiers.Message, targets []notifiers.Target, options *Options) *EnhancedBatchBuilder {
	bb.messages = append(bb.messages, BatchMessage{
		Message: message,
		Targets: targets,
		Options: options,
	})
	return bb
}

// AddText adds a text message to the batch
func (bb *EnhancedBatchBuilder) AddText(title, body string, targets ...notifiers.Target) *EnhancedBatchBuilder {
	message := NewMessage().Title(title).Body(body).Build()
	bb.messages = append(bb.messages, BatchMessage{
		Message: message,
		Targets: targets,
	})
	return bb
}

// AddAlert adds an alert message to the batch
func (bb *EnhancedBatchBuilder) AddAlert(title, body string, targets ...notifiers.Target) *EnhancedBatchBuilder {
	message := NewAlert(title, body).Build()
	bb.messages = append(bb.messages, BatchMessage{
		Message: message,
		Targets: targets,
	})
	return bb
}

// AddToTargetList adds a message that will be sent to a target list
func (bb *EnhancedBatchBuilder) AddToTargetList(message *notifiers.Message, targetList *TargetList) *EnhancedBatchBuilder {
	bb.messages = append(bb.messages, BatchMessage{
		Message: message,
		Targets: targetList.Build(),
	})
	return bb
}

// SendAll sends all messages in the batch
func (bb *EnhancedBatchBuilder) SendAll(ctx context.Context) ([]*notifiers.SendResult, error) {
	var allResults []*notifiers.SendResult

	for _, batchMsg := range bb.messages {
		// Use message-specific targets or default targets
		targets := batchMsg.Targets
		if len(targets) == 0 {
			targets = bb.defaultTargets
		}

		// Create a new message with targets
		messageBuilder := NewMessage().
			Title(batchMsg.Message.Title).
			Body(batchMsg.Message.Body).
			Priority(batchMsg.Message.Priority).
			Format(batchMsg.Message.Format)

		// Add variables
		for k, v := range batchMsg.Message.Variables {
			messageBuilder.Variable(k, v)
		}

		// Add metadata
		for k, v := range batchMsg.Message.Metadata {
			messageBuilder.Metadata(k, v)
		}

		// Add targets
		for _, target := range targets {
			messageBuilder.Target(target)
		}

		message := messageBuilder.Build()

		// Use message-specific options or default options
		options := batchMsg.Options
		if options == nil {
			options = bb.defaultOptions
		}

		// Send the message
		results, err := bb.hub.Send(ctx, message, options)
		if err != nil {
			return allResults, fmt.Errorf("failed to send batch message '%s': %w", message.Title, err)
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// Count returns the number of messages in the batch
func (bb *EnhancedBatchBuilder) Count() int {
	return len(bb.messages)
}

// Enhanced Hub methods for batch operations
func (h *Hub) SendBatchMessages(ctx context.Context, messages []BatchMessage) ([]*notifiers.SendResult, error) {
	batch := h.NewEnhancedBatch()
	for _, msg := range messages {
		batch.AddMessage(msg.Message, msg.Targets, msg.Options)
	}
	return batch.SendAll(ctx)
}

// SendToTargetList sends a single message to a target list
func (h *Hub) SendToTargetList(ctx context.Context, message *notifiers.Message, targetList *TargetList, options *Options) ([]*notifiers.SendResult, error) {
	// Create a new message with targets
	messageBuilder := NewMessage().
		Title(message.Title).
		Body(message.Body).
		Priority(message.Priority).
		Format(message.Format)

	// Add variables
	for k, v := range message.Variables {
		messageBuilder.Variable(k, v)
	}

	// Add metadata
	for k, v := range message.Metadata {
		messageBuilder.Metadata(k, v)
	}

	// Add targets from target list
	for _, target := range targetList.Build() {
		messageBuilder.Target(target)
	}

	return h.Send(ctx, messageBuilder.Build(), options)
}

// SendBulkEmails sends the same message to multiple email addresses efficiently
func (h *Hub) SendBulkEmails(ctx context.Context, title, body string, emails []string, options *Options) ([]*notifiers.SendResult, error) {
	targetList := NewTargetList().AddEmails(emails...)
	message := NewMessage().Title(title).Body(body).Build()
	return h.SendToTargetList(ctx, message, targetList, options)
}

// SendBulkFeishuGroups sends the same message to multiple Feishu groups efficiently
func (h *Hub) SendBulkFeishuGroups(ctx context.Context, title, body string, groups []string, options *Options) ([]*notifiers.SendResult, error) {
	targetList := NewTargetList().AddFeishuGroups(groups...)
	message := NewMessage().Title(title).Body(body).Build()
	return h.SendToTargetList(ctx, message, targetList, options)
}