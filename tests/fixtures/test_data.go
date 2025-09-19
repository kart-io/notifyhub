package fixtures

import (
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/config/routing"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
)

// TestMessages 测试消息集合
var TestMessages = struct {
	Simple      *message.Message
	Complex     *message.Message
	Alert       *message.Message
	Templated   *message.Message
	MultiTarget *message.Message
}{
	Simple: func() *message.Message {
		msg := message.NewMessage()
		msg.SetTitle("Simple Test Message")
		msg.SetBody("This is a simple test message")
		msg.SetPriority(3)
		return msg
	}(),

	Complex: func() *message.Message {
		msg := message.NewMessage()
		msg.SetTitle("Complex Test Message")
		msg.SetBody("This is a complex message with multiple attributes")
		msg.SetPriority(4)
		msg.SetFormat("markdown")
		msg.AddVariable("user", "testuser")
		msg.AddVariable("action", "login")
		msg.AddVariable("timestamp", "2024-01-01T12:00:00Z")
		msg.AddMetadata("source", "api")
		msg.AddMetadata("type", "notification")
		msg.AddMetadata("env", "test")
		return msg
	}(),

	Alert: func() *message.Message {
		msg := message.NewMessage()
		msg.SetTitle("Critical System Alert")
		msg.SetBody("Database connection lost")
		msg.SetPriority(5)
		msg.AddMetadata("severity", "critical")
		msg.AddMetadata("component", "database")
		msg.AddVariable("error", "connection timeout")
		msg.AddVariable("retry_count", 3)
		return msg
	}(),

	Templated: func() *message.Message {
		msg := message.NewMessage()
		msg.SetTemplate("alert-template")
		msg.SetTitle("{{.service}} Alert: {{.status}}")
		msg.SetBody("Service {{.service}} is {{.status}}. Details: {{.details}}")
		msg.SetPriority(3)
		msg.AddVariable("service", "web-api")
		msg.AddVariable("status", "degraded")
		msg.AddVariable("details", "high latency detected")
		return msg
	}(),

	MultiTarget: func() *message.Message {
		msg := message.NewMessage()
		msg.SetTitle("Multi-Target Message")
		msg.SetBody("This message will be sent to multiple targets")
		msg.SetPriority(3)
		msg.AddTarget(message.NewTarget(message.TargetTypeEmail, "user1@example.com", "email"))
		msg.AddTarget(message.NewTarget(message.TargetTypeEmail, "user2@example.com", "email"))
		msg.AddTarget(message.NewTarget(message.TargetTypeUser, "user123", "feishu"))
		msg.AddTarget(message.NewTarget(message.TargetTypeGroup, "dev-team", "slack"))
		msg.AddTarget(message.NewTarget(message.TargetTypeChannel, "general", "discord"))
		return msg
	}(),
}

// TestTargets 测试目标集合
var TestTargets = struct {
	EmailTargets   []sending.Target
	FeishuTargets  []sending.Target
	SlackTargets   []sending.Target
	DiscordTargets []sending.Target
	MixedTargets   []sending.Target
}{
	EmailTargets: []sending.Target{
		sending.NewTarget(sending.TargetTypeEmail, "test@example.com", "email"),
		sending.NewTarget(sending.TargetTypeEmail, "admin@example.com", "email"),
		sending.NewTarget(sending.TargetTypeEmail, "support@example.com", "email"),
	},

	FeishuTargets: []sending.Target{
		sending.NewTarget(sending.TargetTypeUser, "user123", "feishu"),
		sending.NewTarget(sending.TargetTypeUser, "user456", "feishu"),
		sending.NewTarget(sending.TargetTypeGroup, "dev-team", "feishu"),
		sending.NewTarget(sending.TargetTypeGroup, "ops-team", "feishu"),
	},

	SlackTargets: []sending.Target{
		sending.NewTarget(sending.TargetTypeChannel, "general", "slack"),
		sending.NewTarget(sending.TargetTypeChannel, "alerts", "slack"),
		sending.NewTarget(sending.TargetTypeUser, "john.doe", "slack"),
		sending.NewTarget(sending.TargetTypeGroup, "engineering", "slack"),
	},

	DiscordTargets: []sending.Target{
		sending.NewTarget(sending.TargetTypeChannel, "general", "discord"),
		sending.NewTarget(sending.TargetTypeChannel, "announcements", "discord"),
		sending.NewTarget(sending.TargetTypeUser, "user#1234", "discord"),
	},

	MixedTargets: []sending.Target{
		sending.NewTarget(sending.TargetTypeEmail, "test@example.com", "email"),
		sending.NewTarget(sending.TargetTypeUser, "user123", "feishu"),
		sending.NewTarget(sending.TargetTypeChannel, "general", "slack"),
		sending.NewTarget(sending.TargetTypeChannel, "announcements", "discord"),
	},
}

// TestRoutingRules 测试路由规则集合
var TestRoutingRules = []routing.Rule{
	{
		Name:     "critical_alerts",
		Priority: 100,
		Enabled:  true,
		Conditions: routing.Conditions{
			Priorities: []int{5},
			Metadata: map[string]string{
				"severity": "critical",
			},
		},
		Actions: routing.Actions{
			Targets: []routing.Target{
				{Type: "email", Value: "oncall@company.com", Platform: "email"},
				{Type: "group", Value: "critical-alerts", Platform: "feishu"},
				{Type: "channel", Value: "incidents", Platform: "slack"},
			},
			AddMetadata: map[string]string{
				"routed": "true",
				"rule":   "critical_alerts",
			},
		},
	},
	{
		Name:     "high_priority",
		Priority: 80,
		Enabled:  true,
		Conditions: routing.Conditions{
			Priorities: []int{4},
		},
		Actions: routing.Actions{
			Targets: []routing.Target{
				{Type: "email", Value: "team@company.com", Platform: "email"},
				{Type: "group", Value: "high-priority", Platform: "feishu"},
			},
			SetPriority: 5, // 升级优先级
		},
	},
	{
		Name:     "business_hours",
		Priority: 60,
		Enabled:  true,
		Conditions: routing.Conditions{
			Priorities: []int{2, 3},
			Metadata: map[string]string{
				"env": "production",
			},
		},
		Actions: routing.Actions{
			Targets: []routing.Target{
				{Type: "channel", Value: "notifications", Platform: "slack"},
			},
			AddMetadata: map[string]string{
				"time_based": "business_hours",
			},
		},
	},
	{
		Name:     "dev_environment",
		Priority: 40,
		Enabled:  true,
		Conditions: routing.Conditions{
			Metadata: map[string]string{
				"env": "development",
			},
		},
		Actions: routing.Actions{
			Targets: []routing.Target{
				{Type: "channel", Value: "dev-notifications", Platform: "slack"},
			},
		},
	},
	{
		Name:     "default_rule",
		Priority: 1,
		Enabled:  true,
		Conditions: routing.Conditions{
			Priorities: []int{1, 2, 3, 4, 5}, // 匹配所有优先级
		},
		Actions: routing.Actions{
			Targets: []routing.Target{
				{Type: "channel", Value: "general", Platform: "slack"},
			},
			AddMetadata: map[string]string{
				"routed_by": "default",
			},
		},
	},
}

// TestConfigs 测试配置集合
var TestConfigs = struct {
	Simple        *config.Config
	MultiPlatform *config.Config
	WithQueue     *config.Config
	WithRouting   *config.Config
	Full          *config.Config
}{
	Simple: config.New(
		config.WithMockNotifier("test"),
		config.WithSilentLogger(),
	),

	MultiPlatform: config.New(
		config.WithFeishu("https://example.com/webhook", "secret"),
		config.WithEmail("smtp.example.com", 587, "user", "pass", "from@example.com"),
		config.WithMockNotifier("test"),
		config.WithSilentLogger(),
	),

	WithQueue: config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 1000, 4),
		config.WithSilentLogger(),
	),

	WithRouting: config.New(
		config.WithMockNotifier("test"),
		config.WithRouting(TestRoutingRules...),
		config.WithSilentLogger(),
	),

	Full: config.New(
		config.WithFeishu("https://example.com/webhook", "secret"),
		config.WithEmail("smtp.example.com", 587, "user", "pass", "from@example.com"),
		config.WithQueue("memory", 5000, 8),
		config.WithRouting(TestRoutingRules...),
		config.WithSilentLogger(),
	),
}

// ptr 辅助函数，用于创建指针 (工具函数，保留供未来使用)
func ptr[T any](v T) *T { //nolint:unused
	return &v
}
