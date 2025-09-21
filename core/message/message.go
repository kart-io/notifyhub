package message

import (
	"github.com/kart-io/notifyhub/core"
)

// 使用core包中统一定义的类型 - 向后兼容
type Format = core.Format
type TargetType = core.TargetType
type Target = core.Target
type Priority = core.Priority
type Message struct {
	*core.Message
}

// NewTarget creates a new target - 向后兼容函数
func NewTarget(targetType TargetType, value, platform string) Target {
	return core.NewTarget(targetType, value, platform)
}

// NewMessage creates a new message with default values - 向后兼容函数
func NewMessage() *Message {
	return &Message{
		Message: core.NewMessage(),
	}
}

// 向后兼容的常量定义
const (
	PriorityLow      = int(core.PriorityLow)
	PriorityNormal   = int(core.PriorityNormal)
	PriorityMedium   = int(core.PriorityMedium)
	PriorityHigh     = int(core.PriorityHigh)
	PriorityCritical = int(core.PriorityCritical)
)

// 格式常量 - 向后兼容
const (
	FormatText     = core.FormatText
	FormatMarkdown = core.FormatMarkdown
	FormatHTML     = core.FormatHTML
	FormatCard     = core.FormatCard
)

// 目标类型常量 - 向后兼容
const (
	TargetTypeEmail   = core.TargetTypeEmail
	TargetTypeUser    = core.TargetTypeUser
	TargetTypeGroup   = core.TargetTypeGroup
	TargetTypeChannel = core.TargetTypeChannel
	TargetTypeSMS     = core.TargetTypeSMS
)

// 向后兼容的方法 - 添加便利方法到Message类型
func (m *Message) SetTitle(title string) *Message {
	m.Title = title
	return m
}

func (m *Message) SetBody(body string) *Message {
	m.Body = body
	return m
}

func (m *Message) SetFormat(format Format) *Message {
	m.Format = format
	return m
}

func (m *Message) SetPriority(priority Priority) *Message {
	m.Priority = priority
	return m
}

func (m *Message) SetTemplate(template string) *Message {
	m.Template = template
	return m
}

func (m *Message) AddVariable(key string, value interface{}) *Message {
	if m.Variables == nil {
		m.Variables = make(map[string]interface{})
	}
	m.Variables[key] = value
	return m
}

func (m *Message) AddMetadata(key, value string) *Message {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
	return m
}

// 为了向后兼容，添加缺少的getter方法
func (m *Message) GetTitle() string {
	return m.Title
}

func (m *Message) GetBody() string {
	return m.Body
}

func (m *Message) GetPriority() Priority {
	return m.Priority
}

func (m *Message) GetVariables() map[string]interface{} {
	return m.Variables
}

func (m *Message) SetVariables(vars map[string]interface{}) *Message {
	m.Variables = vars
	return m
}

func (m *Message) GetMetadata() map[string]string {
	return m.Metadata
}

func (m *Message) SetMetadataMap(metadata map[string]string) *Message {
	m.Metadata = metadata
	return m
}

func (m *Message) AddTarget(target Target) *Message {
	m.Targets = append(m.Targets, target)
	return m
}

func (m *Message) GetTargets() []Target {
	return m.Targets
}
