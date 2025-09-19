package notifyhub

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
)

// DebugClient 调试客户端
type DebugClient struct {
	client *Client
	traces []TraceEvent
	config DebugConfig
}

// DebugConfig 调试配置
type DebugConfig struct {
	EnableTrace    bool `json:"enable_trace"`
	EnableMetrics  bool `json:"enable_metrics"`
	VerboseLogging bool `json:"verbose_logging"`
	DryRunDefault  bool `json:"dry_run_default"`
}

// TraceEvent 追踪事件
type TraceEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      TraceEventType         `json:"type"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	MessageID string                 `json:"message_id,omitempty"`
	Target    string                 `json:"target,omitempty"`
	Platform  string                 `json:"platform,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// TraceEventType 追踪事件类型
type TraceEventType string

const (
	TraceEventStart     TraceEventType = "start"
	TraceEventValidate  TraceEventType = "validate"
	TraceEventRoute     TraceEventType = "route"
	TraceEventSend      TraceEventType = "send"
	TraceEventComplete  TraceEventType = "complete"
	TraceEventError     TraceEventType = "error"
	TraceEventRetry     TraceEventType = "retry"
	TraceEventRateLimit TraceEventType = "rate_limit"
)

// Debug 创建调试客户端
func (c *Client) Debug() *DebugClient {
	return &DebugClient{
		client: c,
		traces: make([]TraceEvent, 0),
		config: DebugConfig{
			EnableTrace:    true,
			EnableMetrics:  true,
			VerboseLogging: true,
		},
	}
}

// Send 创建调试发送构建器
func (dc *DebugClient) Send(ctx context.Context) *DebugSendBuilder {
	builder := &DebugSendBuilder{
		SendBuilder: dc.client.Send(ctx),
		debug:       dc,
		startTime:   time.Now(),
	}

	dc.addTrace(TraceEvent{
		Timestamp: time.Now(),
		Type:      TraceEventStart,
		Message:   "Starting message build",
	})

	return builder
}

// Alert 创建调试告警构建器
func (dc *DebugClient) Alert(ctx context.Context) *DebugAlertBuilder {
	return &DebugAlertBuilder{
		DebugSendBuilder: dc.Send(ctx),
	}
}

// Notification 创建调试通知构建器
func (dc *DebugClient) Notification(ctx context.Context) *DebugNotificationBuilder {
	return &DebugNotificationBuilder{
		DebugSendBuilder: dc.Send(ctx),
	}
}

// Trace 获取追踪信息
func (dc *DebugClient) Trace() []TraceEvent {
	return dc.traces
}

// ClearTrace 清空追踪信息
func (dc *DebugClient) ClearTrace() {
	dc.traces = make([]TraceEvent, 0)
}

// GetMetrics 获取性能指标
func (dc *DebugClient) GetMetrics() DebugMetrics {
	var totalDuration time.Duration
	var successCount, failureCount int
	platformCounts := make(map[string]int)

	for _, event := range dc.traces {
		if event.Type == TraceEventComplete {
			totalDuration += event.Duration
			if event.Error == "" {
				successCount++
			} else {
				failureCount++
			}
		}
		if event.Platform != "" {
			platformCounts[event.Platform]++
		}
	}

	return DebugMetrics{
		TotalMessages:   successCount + failureCount,
		SuccessCount:    successCount,
		FailureCount:    failureCount,
		AverageDuration: totalDuration / time.Duration(successCount+failureCount),
		PlatformCounts:  platformCounts,
		LastUpdate:      time.Now(),
	}
}

// addTrace 添加追踪事件
func (dc *DebugClient) addTrace(event TraceEvent) {
	if dc.config.EnableTrace {
		dc.traces = append(dc.traces, event)
	}
}

// DebugSendBuilder 调试发送构建器
type DebugSendBuilder struct {
	*SendBuilder
	debug     *DebugClient
	startTime time.Time
}

// Execute 执行发送（带调试）
func (dsb *DebugSendBuilder) Execute() (*Results, error) {
	// 记录验证阶段
	dsb.debug.addTrace(TraceEvent{
		Timestamp: time.Now(),
		Type:      TraceEventValidate,
		Message:   "Validating message",
		MessageID: dsb.message.ID,
		Data: map[string]interface{}{
			"targets_count": len(dsb.targets),
			"has_title":     dsb.message.Title != "",
			"has_body":      dsb.message.Body != "",
		},
	})

	// 记录路由阶段
	dsb.debug.addTrace(TraceEvent{
		Timestamp: time.Now(),
		Type:      TraceEventRoute,
		Message:   "Routing to platforms",
		MessageID: dsb.message.ID,
		Data: map[string]interface{}{
			"targets": dsb.formatTargets(),
		},
	})

	// 执行实际发送
	start := time.Now()
	result, err := dsb.SendBuilder.Execute()
	duration := time.Since(start)

	// 记录完成事件
	event := TraceEvent{
		Timestamp: time.Now(),
		Type:      TraceEventComplete,
		Message:   "Message sending completed",
		MessageID: dsb.message.ID,
		Duration:  duration,
		Data: map[string]interface{}{
			"total_duration": time.Since(dsb.startTime),
		},
	}

	if err != nil {
		event.Type = TraceEventError
		event.Error = err.Error()
		event.Message = "Message sending failed"
	} else if result != nil {
		event.Data["sent"] = result.Sent
		event.Data["failed"] = result.Failed
	}

	dsb.debug.addTrace(event)
	return result, err
}

// DryRun 模拟运行（带调试）
func (dsb *DebugSendBuilder) DryRun() (*DryRunResult, error) {
	dsb.debug.addTrace(TraceEvent{
		Timestamp: time.Now(),
		Type:      TraceEventStart,
		Message:   "Starting dry run",
		MessageID: dsb.message.ID,
	})

	result, err := dsb.SendBuilder.DryRun()

	dsb.debug.addTrace(TraceEvent{
		Timestamp: time.Now(),
		Type:      TraceEventComplete,
		Message:   "Dry run completed",
		MessageID: dsb.message.ID,
		Duration:  time.Since(dsb.startTime),
		Data: map[string]interface{}{
			"valid":         result != nil && result.Valid,
			"targets_count": len(result.Targets),
		},
	})

	return result, err
}

// Analyze 分析消息和目标配置
func (dsb *DebugSendBuilder) Analyze() *MessageAnalysis {
	analysis := &MessageAnalysis{
		Message:     dsb.message,
		Targets:     dsb.targets,
		Timestamp:   time.Now(),
		Issues:      make([]AnalysisIssue, 0),
		Suggestions: make([]string, 0),
	}

	// 分析消息内容
	if dsb.message.Title == "" {
		analysis.Issues = append(analysis.Issues, AnalysisIssue{
			Type:    IssueTypeWarning,
			Message: "Message title is empty",
			Field:   "title",
		})
		analysis.Suggestions = append(analysis.Suggestions, "Consider adding a descriptive title")
	}

	if dsb.message.Body == "" {
		analysis.Issues = append(analysis.Issues, AnalysisIssue{
			Type:    IssueTypeWarning,
			Message: "Message body is empty",
			Field:   "body",
		})
	}

	// 分析目标配置
	if len(dsb.targets) == 0 {
		analysis.Issues = append(analysis.Issues, AnalysisIssue{
			Type:    IssueTypeError,
			Message: "No targets specified",
			Field:   "targets",
		})
	}

	// 分析平台分布
	platforms := make(map[string]int)
	for _, target := range dsb.targets {
		platforms[target.Platform]++
	}
	analysis.PlatformDistribution = platforms

	if len(platforms) == 1 {
		analysis.Suggestions = append(analysis.Suggestions,
			"Consider adding multiple platforms for redundancy")
	}

	// 分析优先级
	if dsb.message.Priority == 0 {
		analysis.Suggestions = append(analysis.Suggestions,
			"Consider setting message priority for better routing")
	}

	return analysis
}

// formatTargets 格式化目标列表
func (dsb *DebugSendBuilder) formatTargets() []string {
	targets := make([]string, len(dsb.targets))
	for i, target := range dsb.targets {
		targets[i] = fmt.Sprintf("%s:%s", target.Platform, target.Value)
	}
	return targets
}

// DebugAlertBuilder 调试告警构建器
type DebugAlertBuilder struct {
	*DebugSendBuilder
}

// DebugNotificationBuilder 调试通知构建器
type DebugNotificationBuilder struct {
	*DebugSendBuilder
}

// 调试相关类型

// DebugMetrics 调试指标
type DebugMetrics struct {
	TotalMessages   int            `json:"total_messages"`
	SuccessCount    int            `json:"success_count"`
	FailureCount    int            `json:"failure_count"`
	AverageDuration time.Duration  `json:"average_duration"`
	PlatformCounts  map[string]int `json:"platform_counts"`
	LastUpdate      time.Time      `json:"last_update"`
}

// MessageAnalysis 消息分析结果
type MessageAnalysis struct {
	Message              *message.Message `json:"message"`
	Targets              []sending.Target `json:"targets"`
	Timestamp            time.Time        `json:"timestamp"`
	Issues               []AnalysisIssue  `json:"issues"`
	Suggestions          []string         `json:"suggestions"`
	PlatformDistribution map[string]int   `json:"platform_distribution"`
	EstimatedCost        float64          `json:"estimated_cost,omitempty"`
	EstimatedDuration    time.Duration    `json:"estimated_duration,omitempty"`
}

// AnalysisIssue 分析问题
type AnalysisIssue struct {
	Type    IssueType `json:"type"`
	Message string    `json:"message"`
	Field   string    `json:"field,omitempty"`
	Code    string    `json:"code,omitempty"`
}

// IssueType 问题类型
type IssueType string

const (
	IssueTypeError   IssueType = "error"
	IssueTypeWarning IssueType = "warning"
	IssueTypeInfo    IssueType = "info"
)

// 调试工具函数

// PrintTrace 打印追踪信息
func (dc *DebugClient) PrintTrace() {
	fmt.Println("=== Message Sending Trace ===")
	for i, event := range dc.traces {
		fmt.Printf("[%d] %s | %s | %s",
			i+1,
			event.Timestamp.Format("15:04:05.000"),
			event.Type,
			event.Message,
		)

		if event.MessageID != "" {
			fmt.Printf(" | ID:%s", event.MessageID)
		}

		if event.Duration > 0 {
			fmt.Printf(" | Duration:%v", event.Duration)
		}

		if event.Error != "" {
			fmt.Printf(" | Error:%s", event.Error)
		}

		fmt.Println()

		if event.Data != nil && dc.config.VerboseLogging {
			for k, v := range event.Data {
				fmt.Printf("    %s: %v\n", k, v)
			}
		}
	}
	fmt.Println("=============================")
}

// PrintMetrics 打印性能指标
func (dc *DebugClient) PrintMetrics() {
	metrics := dc.GetMetrics()
	fmt.Println("=== Performance Metrics ===")
	fmt.Printf("Total Messages: %d\n", metrics.TotalMessages)
	fmt.Printf("Success Rate: %.2f%%\n",
		float64(metrics.SuccessCount)/float64(metrics.TotalMessages)*100)
	fmt.Printf("Average Duration: %v\n", metrics.AverageDuration)
	fmt.Println("Platform Distribution:")
	for platform, count := range metrics.PlatformCounts {
		fmt.Printf("  %s: %d\n", platform, count)
	}
	fmt.Printf("Last Update: %s\n", metrics.LastUpdate.Format("2006-01-02 15:04:05"))
	fmt.Println("===========================")
}

// InspectMessage 检查消息配置
func InspectMessage(msg *message.Message) {
	fmt.Println("=== Message Inspection ===")
	fmt.Printf("ID: %s\n", msg.ID)
	fmt.Printf("Title: %s\n", msg.Title)
	fmt.Printf("Body: %s (%d chars)\n",
		truncateString(msg.Body, 50), len(msg.Body))
	fmt.Printf("Priority: %d\n", msg.Priority)
	fmt.Printf("Format: %s\n", msg.Format)
	fmt.Printf("Template: %s\n", msg.Template)
	fmt.Printf("Variables: %d\n", len(msg.Variables))
	fmt.Printf("Metadata: %d\n", len(msg.Metadata))
	fmt.Printf("Delay: %v\n", msg.Delay)
	fmt.Printf("Created: %s\n", msg.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println("==========================")
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
