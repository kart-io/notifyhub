package notifyhub

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/core/errors"
)

// SendBuilder 统一的发送构建器
type SendBuilder struct {
	client   *Client
	ctx      context.Context
	message  *core.Message
	targets  []core.Target
	metadata map[string]string
}

// 消息内容设置
func (b *SendBuilder) Title(title string) *SendBuilder {
	b.message.Title = title
	return b
}

func (b *SendBuilder) Body(body string) *SendBuilder {
	b.message.Body = body
	return b
}

func (b *SendBuilder) Priority(priority int) *SendBuilder {
	b.message.Priority = core.Priority(priority)
	return b
}

func (b *SendBuilder) Format(format core.Format) *SendBuilder {
	b.message.Format = format
	return b
}

// 模板和变量
func (b *SendBuilder) Template(template string) *SendBuilder {
	b.message.Template = template
	return b
}

func (b *SendBuilder) Variable(key string, value interface{}) *SendBuilder {
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	b.message.Variables[key] = value
	return b
}

func (b *SendBuilder) Variables(vars map[string]interface{}) *SendBuilder {
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	for k, v := range vars {
		b.message.Variables[k] = v
	}
	return b
}

// Vars is an alias for Variables for convenience
func (b *SendBuilder) Vars(vars map[string]interface{}) *SendBuilder {
	return b.Variables(vars)
}

// 元数据
func (b *SendBuilder) Metadata(key, value string) *SendBuilder {
	if b.metadata == nil {
		b.metadata = make(map[string]string)
	}
	b.metadata[key] = value
	return b
}

// Meta is an alias for Metadata for convenience
func (b *SendBuilder) Meta(key, value string) *SendBuilder {
	return b.Metadata(key, value)
}

func (b *SendBuilder) MetadataMap(metadata map[string]string) *SendBuilder {
	if b.metadata == nil {
		b.metadata = make(map[string]string)
	}
	for k, v := range metadata {
		b.metadata[k] = v
	}
	return b
}

// 目标设置 - 类型安全的目标添加
func (b *SendBuilder) ToEmail(addresses ...string) *SendBuilder {
	for _, addr := range addresses {
		target := core.NewTarget(core.TargetTypeEmail, addr, "email")
		b.targets = append(b.targets, target)
	}
	return b
}

func (b *SendBuilder) ToFeishu(groups ...string) *SendBuilder {
	for _, group := range groups {
		target := core.NewTarget(core.TargetTypeGroup, group, "feishu")
		b.targets = append(b.targets, target)
	}
	return b
}

func (b *SendBuilder) ToSMS(numbers ...string) *SendBuilder {
	for _, number := range numbers {
		target := core.NewTarget(core.TargetTypeSMS, number, "sms")
		b.targets = append(b.targets, target)
	}
	return b
}

func (b *SendBuilder) ToSlack(channels ...string) *SendBuilder {
	for _, channel := range channels {
		target := core.NewTarget(core.TargetTypeGroup, channel, "slack")
		b.targets = append(b.targets, target)
	}
	return b
}

// ToPlatform allows explicitly specifying the platform
func (b *SendBuilder) ToPlatform(platform PlatformType, values ...string) *SendBuilder {
	switch platform {
	case PlatformEmail:
		return b.ToEmail(values...)
	case PlatformFeishu:
		return b.ToFeishu(values...)
	case PlatformSMS:
		return b.ToSMS(values...)
	case PlatformSlack:
		return b.ToSlack(values...)
	}
	return b
}

// To with automatic platform detection
func (b *SendBuilder) To(values ...string) *SendBuilder {
	for _, value := range values {
		// Auto-detect platform based on value format
		if isEmail(value) {
			b.ToEmail(value)
		} else if isPhoneNumber(value) {
			b.ToSMS(value)
		} else {
			// Default to treating as email for compatibility
			b.ToEmail(value)
		}
	}
	return b
}

// 目标表达式 - 支持更灵活的目标定义
func (b *SendBuilder) ToExpression(expressions ...string) *SendBuilder {
	// TODO: 解析目标表达式，如 "email:admin@company.com", "feishu:alerts-group"
	return b
}

// 延迟发送
func (b *SendBuilder) DelayBy(duration time.Duration) *SendBuilder {
	b.message.Delay = duration
	return b
}

func (b *SendBuilder) DelayUntil(t time.Time) *SendBuilder {
	b.message.Delay = time.Until(t)
	return b
}

// Send executes the message sending with given context
func (b *SendBuilder) Send(ctx context.Context) (*Results, error) {
	// 设置元数据
	if b.metadata != nil {
		if b.message.Metadata == nil {
			b.message.Metadata = make(map[string]string)
		}
		for k, v := range b.metadata {
			b.message.Metadata[k] = v
		}
	}

	// 将 builder targets 同步到消息中，以便验证通过
	b.message.Targets = b.targets

	// 发送消息 - use the provided context
	results, err := b.client.hub.Send(ctx, b.message, b.targets)
	if err != nil {
		return nil, err
	}

	return &Results{
		MessageID: b.message.ID,
		Sent:      results.Success,
		Failed:    results.Failed,
		Results:   convertToTargetResults(results.Results),
	}, nil
}

// Execute executes the message sending with the builder's context (for backward compatibility)
func (b *SendBuilder) Execute() (*Results, error) {
	return b.Send(b.ctx)
}

// GetMessage returns the current message being built (for debugging)
func (b *SendBuilder) GetMessage() *core.Message {
	return b.message
}

// DryRun 模拟发送，不实际执行
func (b *SendBuilder) DryRun() (*DryRunResult, error) {
	return &DryRunResult{
		Message: b.message,
		Targets: b.targets,
		Valid:   b.message.Validate() == nil,
	}, nil
}

// AlertBuilder 告警构建器
type AlertBuilder struct {
	*SendBuilder
	priority int
}

func (ab *AlertBuilder) Critical() *AlertBuilder {
	ab.Priority(int(core.PriorityCritical))
	return ab
}

func (ab *AlertBuilder) High() *AlertBuilder {
	ab.Priority(int(core.PriorityHigh))
	return ab
}

func (ab *AlertBuilder) Medium() *AlertBuilder {
	ab.Priority(int(core.PriorityMedium))
	return ab
}

func (ab *AlertBuilder) Low() *AlertBuilder {
	ab.Priority(int(core.PriorityLow))
	return ab
}

// NotificationBuilder 通知构建器
type NotificationBuilder struct {
	*SendBuilder
	priority int
}

func (nb *NotificationBuilder) Important() *NotificationBuilder {
	nb.Priority(int(core.PriorityHigh))
	return nb
}

func (nb *NotificationBuilder) Normal() *NotificationBuilder {
	nb.Priority(int(core.PriorityNormal))
	return nb
}

func (nb *NotificationBuilder) Info() *NotificationBuilder {
	nb.Priority(int(core.PriorityLow))
	return nb
}

// Results 发送结果
type Results struct {
	MessageID string         `json:"message_id"`
	Sent      int            `json:"sent"`
	Failed    int            `json:"failed"`
	Results   []TargetResult `json:"results"`
}

// TargetResult 单个目标的发送结果
type TargetResult struct {
	Target    core.Target         `json:"target"`
	Status    DeliveryStatus      `json:"status"`
	Error     *errors.NotifyError `json:"error,omitempty"`
	Timestamp time.Time           `json:"timestamp"`
	Duration  time.Duration       `json:"duration"`
}

// DeliveryStatus 发送状态
type DeliveryStatus string

const (
	StatusPending  DeliveryStatus = "pending"
	StatusSending  DeliveryStatus = "sending"
	StatusSent     DeliveryStatus = "sent"
	StatusFailed   DeliveryStatus = "failed"
	StatusRetrying DeliveryStatus = "retrying"
)

// DryRunResult 模拟运行结果
type DryRunResult struct {
	Message *core.Message `json:"message"`
	Targets []core.Target `json:"targets"`
	Valid   bool          `json:"valid"`
	Issues  []string      `json:"issues,omitempty"`
}

// convertToTargetResults 转换发送结果
func convertToTargetResults(results []*core.Result) []TargetResult {
	targetResults := make([]TargetResult, len(results))
	for i, result := range results {
		targetResults[i] = TargetResult{
			Target:    result.Target,
			Status:    convertStatus(result.Status),
			Timestamp: result.Timestamp,
			Duration:  result.Duration,
		}
		if result.Error != nil {
			// 尝试将错误转换为统一的错误类型
			if notifyErr, ok := result.Error.(*errors.NotifyError); ok {
				targetResults[i].Error = notifyErr
			} else {
				// 将其他错误映射为网络错误
				targetResults[i].Error = errors.WrapWithPlatform(
					errors.CodeSendingFailed,
					errors.CategoryTransport,
					result.Error.Error(),
					result.Target.Platform,
					result.Error,
				)
			}
		}
	}
	return targetResults
}

// convertStatus 转换状态
func convertStatus(status core.Status) DeliveryStatus {
	switch status {
	case core.StatusPending:
		return StatusPending
	case core.StatusSending:
		return StatusSending
	case core.StatusSent:
		return StatusSent
	case core.StatusFailed:
		return StatusFailed
	default:
		return StatusFailed
	}
}

// Helper functions for platform detection
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)

// isEmail checks if a string looks like an email address
func isEmail(value string) bool {
	return emailRegex.MatchString(strings.TrimSpace(value))
}

// isPhoneNumber checks if a string looks like a phone number
func isPhoneNumber(value string) bool {
	return phoneRegex.MatchString(strings.TrimSpace(value))
}
