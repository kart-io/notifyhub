// NotifyHub 优化建议示例代码

package main

import (
	"context"
	"net/http"
	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
)

// ==========================================
// 优化建议 1: 改善测试配置
// ==========================================

// 当前问题: WithTestDefaults() 不包含notifier
// hub, err := client.New(config.WithTestDefaults()) // 失败

// 建议: 让测试配置包含mock notifier
func Example_ImprovedTestConfig() {
	// 选项1: 改进WithTestDefaults，包含mock notifier
	hub, err := client.New(config.WithTestDefaults()) // ✅ 应该直接工作
	_ = hub
	_ = err

	// 选项2: 添加专用测试构造函数
	hub, err = client.NewForTesting() // ✅ 简单易用
	_ = hub
	_ = err

	// 选项3: 添加WithMockNotifier选项
	hub, err = client.New(
		config.WithTestDefaults(),
		config.WithMockNotifier(), // ✅ 专门的mock notifier
	)
	_ = hub
	_ = err
}

// ==========================================
// 优化建议 2: 简化生命周期管理
// ==========================================

// 当前问题: 需要手动Start/Stop
func Example_CurrentLifecycle() {
	hub, err := client.New(config.WithTestDefaults())
	if err != nil { /* handle */ }

	if err := hub.Start(context.Background()); err != nil { /* handle */ }
	defer hub.Stop() // 容易忘记

	_ = hub
}

// 建议: 提供自动管理选项
func Example_ImprovedLifecycle() {
	ctx := context.Background()

	// 选项1: 组合创建+启动
	hub, err := client.NewAndStart(ctx, config.WithTestDefaults())
	if err != nil { /* handle */ }
	defer hub.Close() // 更简洁的关闭方法
	_ = hub

	// 选项2: Context自动管理
	hub, err = client.NewWithContext(ctx, config.WithTestDefaults())
	if err != nil { /* handle */ }
	// ctx取消时自动清理，无需手动Stop
	_ = hub

	// 选项3: 建造者模式
	hub = client.NewBuilder().
		WithTestDefaults().
		AutoStart(ctx).
		Build()
	defer hub.Close()
}

// ==========================================
// 优化建议 3: 简化消息创建
// ==========================================

// 当前问题: 需要40+行转换代码
type NotificationRequest struct {
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Targets   []TargetRequest        `json:"targets"`
	Priority  int                    `json:"priority,omitempty"`
	Format    string                 `json:"format,omitempty"`
	Template  string                 `json:"template,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type TargetRequest struct {
	Type     string                 `json:"type"`
	Value    string                 `json:"value"`
	Platform string                 `json:"platform,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// 建议: 提供直接转换函数
func Example_ImprovedMessageCreation() {
	var httpRequest *http.Request

	// 选项1: 直接从HTTP请求创建
	message, err := notifiers.NewMessageFromHTTPRequest(httpRequest)
	if err != nil { /* handle */ }
	_ = message

	// 选项2: 从结构体创建
	var req NotificationRequest
	message, err = notifiers.NewMessageFromStruct(&req)
	if err != nil { /* handle */ }
	_ = message

	// 选项3: 改进的建造者模式
	message = notifiers.NewMessage().
		Title("Alert").
		Body("Something happened").
		ToEmail("admin@example.com").
		ToFeishu("@all").
		WithPriority(4).
		Build()
	_ = message
}

// ==========================================
// 优化建议 4: 改善错误反馈
// ==========================================

// 当前问题: "skipped - no matching targets" 信息不够详细
func Example_ImprovedErrorFeedback() {
	hub, _ := client.NewForTesting()
	ctx := context.Background()

	message := &notifiers.Message{
		Title: "Test",
		Body:  "Test message",
		Targets: []notifiers.Target{
			{Type: "email", Value: "test@example.com"},
		},
	}

	// 选项1: 更详细的发送结果
	type ImprovedSendResult struct {
		TotalTargets    int
		SuccessfulSends int
		FailedSends     int
		SkippedSends    int
		Details         []SendDetail
	}

	type SendDetail struct {
		Target      notifiers.Target
		Notifier    string
		Status      string // "success", "failed", "skipped"
		Reason      string // 详细原因
		Error       error
		Duration    string
	}

	// 选项2: 发送前验证
	validation := hub.ValidateMessage(message)
	if !validation.IsValid() {
		// validation.Errors 包含详细错误信息
		for _, err := range validation.Errors {
			// err.Target, err.Reason, err.Suggestion
			_ = err
		}
	}

	// 选项3: 智能匹配建议
	suggestions := hub.GetTargetSuggestions(message.Targets)
	for _, suggestion := range suggestions {
		// suggestion.OriginalTarget, suggestion.SuggestedNotifier, suggestion.Reason
		_ = suggestion
	}

	_, _ = hub.Send(ctx, message, nil)
}

// ==========================================
// 优化建议 5: 改善配置体验
// ==========================================

func Example_ImprovedConfig() {
	// 选项1: 链式配置
	hub := client.NewBuilder().
		WithFeishu("webhook-url", "secret").
		WithEmail("smtp.gmail.com", 587, "user", "pass").
		WithQueue("redis", "localhost:6379").
		WithRetryPolicy(3, "exponential").
		Build()
	defer hub.Close()

	// 选项2: 配置加载优先级
	config := client.LoadConfig().
		FromFile("config.yaml").      // 1. 配置文件
		FromEnv().                   // 2. 环境变量
		FromDefaults().              // 3. 默认值
		Build()

	hub, err := client.NewWithConfig(config)
	if err != nil { /* handle */ }
	defer hub.Close()

	// 选项3: 预设配置
	hub, err = client.NewProduction()    // 生产环境预设
	if err != nil { /* handle */ }
	defer hub.Close()

	hub, err = client.NewDevelopment()   // 开发环境预设
	if err != nil { /* handle */ }
	defer hub.Close()
}

// ==========================================
// 优化建议 6: 添加便捷方法
// ==========================================

func Example_ConvenienceMethods() {
	// 当前: 需要创建完整的Message结构
	ctx := context.Background()
	hub, _ := client.NewForTesting()

	// 建议: 添加快捷发送方法
	err := hub.SendQuick(ctx, "Alert", "Something happened", "admin@example.com")
	if err != nil { /* handle */ }

	// 批量快捷发送
	err = hub.SendBulkQuick(ctx, "Alert", "Bulk message", []string{
		"admin1@example.com",
		"admin2@example.com",
	})
	if err != nil { /* handle */ }

	// 模板快速发送
	err = hub.SendTemplate(ctx, "alert-template", map[string]interface{}{
		"title":    "System Alert",
		"severity": "high",
		"target":   "admin@example.com",
	})
	if err != nil { /* handle */ }
}