package notifyhub

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
)

// SendBuilder 统一的发送构建器
type SendBuilder struct {
	client   *Client
	ctx      context.Context
	message  *message.Message
	targets  []sending.Target
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
	b.message.Priority = priority
	return b
}

func (b *SendBuilder) Format(format message.Format) *SendBuilder {
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

// 元数据
func (b *SendBuilder) Metadata(key, value string) *SendBuilder {
	if b.metadata == nil {
		b.metadata = make(map[string]string)
	}
	b.metadata[key] = value
	return b
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
		target := sending.NewTarget(sending.TargetTypeEmail, addr, "email")
		b.targets = append(b.targets, target)
	}
	return b
}

func (b *SendBuilder) ToFeishu(groups ...string) *SendBuilder {
	for _, group := range groups {
		target := sending.NewTarget(sending.TargetTypeGroup, group, "feishu")
		b.targets = append(b.targets, target)
	}
	return b
}

func (b *SendBuilder) ToSMS(numbers ...string) *SendBuilder {
	for _, number := range numbers {
		target := sending.NewTarget(sending.TargetTypeSMS, number, "sms")
		b.targets = append(b.targets, target)
	}
	return b
}

func (b *SendBuilder) ToSlack(channels ...string) *SendBuilder {
	for _, channel := range channels {
		target := sending.NewTarget(sending.TargetTypeGroup, channel, "slack")
		b.targets = append(b.targets, target)
	}
	return b
}

// 通用目标添加
func (b *SendBuilder) To(platform PlatformType, values ...string) *SendBuilder {
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

// 执行发送
func (b *SendBuilder) Execute() (*Results, error) {
	// 设置元数据
	if b.metadata != nil {
		if b.message.Metadata == nil {
			b.message.Metadata = make(map[string]string)
		}
		for k, v := range b.metadata {
			b.message.Metadata[k] = v
		}
	}

	// 发送消息
	results, err := b.client.hub.Send(b.ctx, b.message, b.targets)
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
	ab.Priority(message.PriorityCritical)
	return ab
}

func (ab *AlertBuilder) High() *AlertBuilder {
	ab.Priority(message.PriorityHigh)
	return ab
}

func (ab *AlertBuilder) Medium() *AlertBuilder {
	ab.Priority(message.PriorityMedium)
	return ab
}

func (ab *AlertBuilder) Low() *AlertBuilder {
	ab.Priority(message.PriorityLow)
	return ab
}

// NotificationBuilder 通知构建器
type NotificationBuilder struct {
	*SendBuilder
	priority int
}

func (nb *NotificationBuilder) Important() *NotificationBuilder {
	nb.Priority(message.PriorityHigh)
	return nb
}

func (nb *NotificationBuilder) Normal() *NotificationBuilder {
	nb.Priority(message.PriorityNormal)
	return nb
}

func (nb *NotificationBuilder) Info() *NotificationBuilder {
	nb.Priority(message.PriorityLow)
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
	Target    sending.Target     `json:"target"`
	Status    DeliveryStatus     `json:"status"`
	Error     *NotificationError `json:"error,omitempty"`
	Timestamp time.Time          `json:"timestamp"`
	Duration  time.Duration      `json:"duration"`
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

// NotificationError 统一错误类型
type NotificationError struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Platform  string    `json:"platform"`
	Target    string    `json:"target"`
	Retryable bool      `json:"retryable"`
}

// ErrorCode 错误代码
type ErrorCode int

const (
	ErrInvalidTarget ErrorCode = iota
	ErrRateLimited
	ErrNetworkFailure
	ErrAuthenticationFailed
	ErrInvalidMessage
	ErrPlatformUnavailable
	ErrTimeout
)

// DryRunResult 模拟运行结果
type DryRunResult struct {
	Message *message.Message `json:"message"`
	Targets []sending.Target `json:"targets"`
	Valid   bool             `json:"valid"`
	Issues  []string         `json:"issues,omitempty"`
}

// convertToTargetResults 转换发送结果
func convertToTargetResults(results []*sending.Result) []TargetResult {
	targetResults := make([]TargetResult, len(results))
	for i, result := range results {
		targetResults[i] = TargetResult{
			Target:    result.Target,
			Status:    convertStatus(result.Status),
			Timestamp: result.Timestamp,
			Duration:  result.Duration,
		}
		if result.Error != nil {
			targetResults[i].Error = &NotificationError{
				Code:      ErrNetworkFailure, // TODO: 更精确的错误码映射
				Message:   result.Error.Error(),
				Platform:  result.Target.Platform,
				Target:    result.Target.String(),
				Retryable: true, // TODO: 判断是否可重试
			}
		}
	}
	return targetResults
}

// convertStatus 转换状态
func convertStatus(status sending.Status) DeliveryStatus {
	switch status {
	case sending.StatusPending:
		return StatusPending
	case sending.StatusSending:
		return StatusSending
	case sending.StatusSent:
		return StatusSent
	case sending.StatusFailed:
		return StatusFailed
	default:
		return StatusFailed
	}
}
