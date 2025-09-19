package notifyhub

import (
	"fmt"
	"strings"

	"github.com/kart-io/notifyhub/core/sending"
)

// TypedTarget 类型安全的目标定义
type TypedTarget[T PlatformType] struct {
	Type     T                 `json:"type"`
	Value    string            `json:"value"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// String 返回目标的字符串表示
func (t TypedTarget[T]) String() string {
	return fmt.Sprintf("%s:%s", string(t.Type), t.Value)
}

// ToSendingTarget 转换为发送目标
func (t TypedTarget[T]) ToSendingTarget() sending.Target {
	var targetType sending.TargetType
	switch string(t.Type) {
	case string(PlatformEmail):
		targetType = sending.TargetTypeEmail
	case string(PlatformFeishu):
		targetType = sending.TargetTypeGroup
	case string(PlatformSMS):
		targetType = sending.TargetTypeSMS
	case string(PlatformSlack):
		targetType = sending.TargetTypeGroup
	default:
		targetType = sending.TargetTypeOther
	}

	target := sending.NewTarget(targetType, t.Value, string(t.Type))
	if t.Metadata != nil {
		for k, v := range t.Metadata {
			target.Metadata[k] = v
		}
	}
	return target
}

// 类型安全的目标创建函数

// Email 创建邮件目标
func Email(address string) TypedTarget[PlatformType] {
	return TypedTarget[PlatformType]{
		Type:  PlatformEmail,
		Value: address,
	}
}

// EmailWithName 创建带名称的邮件目标
func EmailWithName(address, name string) TypedTarget[PlatformType] {
	return TypedTarget[PlatformType]{
		Type:  PlatformEmail,
		Value: fmt.Sprintf("%s <%s>", name, address),
	}
}

// Feishu 创建飞书目标
func Feishu(group string) TypedTarget[PlatformType] {
	return TypedTarget[PlatformType]{
		Type:  PlatformFeishu,
		Value: group,
	}
}

// FeishuUser 创建飞书用户目标
func FeishuUser(userID string) TypedTarget[PlatformType] {
	return TypedTarget[PlatformType]{
		Type:  PlatformFeishu,
		Value: userID,
		Metadata: map[string]string{
			"target_type": "user",
		},
	}
}

// SMS 创建短信目标
func SMS(number string) TypedTarget[PlatformType] {
	return TypedTarget[PlatformType]{
		Type:  PlatformSMS,
		Value: number,
	}
}

// Slack 创建Slack目标
func Slack(channel string) TypedTarget[PlatformType] {
	return TypedTarget[PlatformType]{
		Type:  PlatformSlack,
		Value: channel,
	}
}

// SlackUser 创建Slack用户目标
func SlackUser(userID string) TypedTarget[PlatformType] {
	return TypedTarget[PlatformType]{
		Type:  PlatformSlack,
		Value: userID,
		Metadata: map[string]string{
			"target_type": "user",
		},
	}
}

// 目标表达式解析

// TargetExpression 目标表达式
type TargetExpression string

// ParseTargetExpression 解析目标表达式
// 支持格式: "email:admin@company.com", "feishu:alerts-group", "sms:+8613800138000"
func ParseTargetExpression(expr TargetExpression) (TypedTarget[PlatformType], error) {
	str := string(expr)
	parts := strings.SplitN(str, ":", 2)
	if len(parts) != 2 {
		return TypedTarget[PlatformType]{}, fmt.Errorf("invalid target expression: %s", str)
	}

	platform := PlatformType(parts[0])
	value := parts[1]

	switch platform {
	case PlatformEmail:
		return Email(value), nil
	case PlatformFeishu:
		return Feishu(value), nil
	case PlatformSMS:
		return SMS(value), nil
	case PlatformSlack:
		return Slack(value), nil
	default:
		return TypedTarget[PlatformType]{}, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// ParseTargetExpressions 批量解析目标表达式
func ParseTargetExpressions(expressions ...TargetExpression) ([]TypedTarget[PlatformType], error) {
	targets := make([]TypedTarget[PlatformType], 0, len(expressions))
	for _, expr := range expressions {
		target, err := ParseTargetExpression(expr)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}

// 预定义目标组

// TargetGroup 目标组
type TargetGroup struct {
	Name    string                      `json:"name"`
	Targets []TypedTarget[PlatformType] `json:"targets"`
}

// TargetRegistry 目标注册表
type TargetRegistry struct {
	groups map[string]TargetGroup
}

// NewTargetRegistry 创建目标注册表
func NewTargetRegistry() *TargetRegistry {
	return &TargetRegistry{
		groups: make(map[string]TargetGroup),
	}
}

// RegisterGroup 注册目标组
func (tr *TargetRegistry) RegisterGroup(group TargetGroup) {
	tr.groups[group.Name] = group
}

// GetGroup 获取目标组
func (tr *TargetRegistry) GetGroup(name string) (TargetGroup, bool) {
	group, exists := tr.groups[name]
	return group, exists
}

// ListGroups 列出所有目标组
func (tr *TargetRegistry) ListGroups() []string {
	names := make([]string, 0, len(tr.groups))
	for name := range tr.groups {
		names = append(names, name)
	}
	return names
}

// 预定义的常用目标组

// PredefinedGroups 预定义目标组
var PredefinedGroups = map[string]TargetGroup{
	"admins": {
		Name: "admins",
		Targets: []TypedTarget[PlatformType]{
			Email("admin@company.com"),
			Feishu("admin-alerts"),
		},
	},
	"oncall": {
		Name: "oncall",
		Targets: []TypedTarget[PlatformType]{
			Email("oncall@company.com"),
			SMS("+8613800138000"),
			Feishu("oncall-alerts"),
		},
	},
	"team": {
		Name: "team",
		Targets: []TypedTarget[PlatformType]{
			Email("team@company.com"),
			Feishu("team-notifications"),
		},
	},
	"critical": {
		Name: "critical",
		Targets: []TypedTarget[PlatformType]{
			Email("critical@company.com"),
			SMS("+8613800138000"),
			Feishu("critical-alerts"),
			Slack("#critical-alerts"),
		},
	},
}

// 目标构建器扩展

// ToTargets 添加类型安全目标
func (b *SendBuilder) ToTargets(targets ...TypedTarget[PlatformType]) *SendBuilder {
	for _, target := range targets {
		b.targets = append(b.targets, target.ToSendingTarget())
	}
	return b
}

// ToGroup 发送到预定义目标组
func (b *SendBuilder) ToGroup(groupName string) *SendBuilder {
	if group, exists := PredefinedGroups[groupName]; exists {
		return b.ToTargets(group.Targets...)
	}
	return b
}

// ToExpressions 使用目标表达式
func (b *SendBuilder) ToExpressions(expressions ...string) *SendBuilder {
	for _, expr := range expressions {
		if target, err := ParseTargetExpression(TargetExpression(expr)); err == nil {
			b.targets = append(b.targets, target.ToSendingTarget())
		}
	}
	return b
}

// 智能目标解析

// SmartTarget 智能目标解析
func SmartTarget(input string) TypedTarget[PlatformType] {
	// 自动检测目标类型
	if strings.Contains(input, "@") && strings.Contains(input, ".") {
		// 看起来像邮箱
		return Email(input)
	}
	if strings.HasPrefix(input, "+") || (len(input) >= 10 && len(input) <= 15) {
		// 看起来像手机号
		return SMS(input)
	}
	if strings.HasPrefix(input, "#") {
		// 看起来像Slack频道
		return Slack(strings.TrimPrefix(input, "#"))
	}
	// 默认当作飞书群组
	return Feishu(input)
}

// ToSmart 智能目标添加
func (b *SendBuilder) ToSmart(inputs ...string) *SendBuilder {
	for _, input := range inputs {
		target := SmartTarget(input)
		b.targets = append(b.targets, target.ToSendingTarget())
	}
	return b
}
