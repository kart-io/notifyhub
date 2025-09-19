package feishu

import (
	"github.com/kart-io/notifyhub/platforms/common"
)

// FeishuBuilder provides Feishu-specific functionality
// This implements the platform-specific builder pattern from the proposal
type FeishuBuilder struct {
	base     *common.MessageBuilder
	mentions []FeishuMention
}

// Platform returns the platform name
func (fb *FeishuBuilder) Platform() string {
	return "feishu"
}

// Builder returns the base message builder
func (fb *FeishuBuilder) Builder() *common.MessageBuilder {
	return fb.base
}

// AtUser adds a user mention (Feishu-specific functionality)
func (fb *FeishuBuilder) AtUser(userID, userName string) *FeishuBuilder {
	mention := FeishuMention{
		MentionType: common.MentionTypeUser,
		UserID:      userID,
		UserName:    userName,
	}
	fb.mentions = append(fb.mentions, mention)
	return fb
}

// AtUserStrict adds a strict user mention (must be valid user)
func (fb *FeishuBuilder) AtUserStrict(userID, userName string) *FeishuBuilder {
	mention := FeishuMention{
		MentionType: common.MentionTypeUser,
		UserID:      userID,
		UserName:    userName,
		Strict:      true,
	}
	fb.mentions = append(fb.mentions, mention)
	return fb
}

// AtAll mentions all users in the group
func (fb *FeishuBuilder) AtAll() *FeishuBuilder {
	mention := FeishuMention{
		MentionType: common.MentionTypeAll,
	}
	fb.mentions = append(fb.mentions, mention)
	return fb
}

// ToGroup adds a Feishu group target
func (fb *FeishuBuilder) ToGroup(groupID string) *FeishuBuilder {
	target := map[string]interface{}{
		"type":     "group",
		"value":    groupID,
		"platform": "feishu",
	}
	fb.base.AddTarget(target)
	return fb
}

// ToUser adds a Feishu user target
func (fb *FeishuBuilder) ToUser(userID string) *FeishuBuilder {
	target := map[string]interface{}{
		"type":     "user",
		"value":    userID,
		"platform": "feishu",
	}
	fb.base.AddTarget(target)
	return fb
}

// Build returns the final message with Feishu-specific processing
func (fb *FeishuBuilder) Build() *common.MessageBuilder {
	// Process mentions and add them to the message body
	if len(fb.mentions) > 0 {
		// Add mention processing logic here
		// This would render mentions into the message format
		// TODO: Implement mention handling
		_ = fb.mentions // Acknowledge usage
	}
	return fb.base
}
