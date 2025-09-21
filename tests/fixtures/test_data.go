package fixtures

import (
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/config/routing"
	"github.com/kart-io/notifyhub/core"
)

// TestMessages 测试消息集合
var TestMessages = struct {
	Simple      *core.Message
	Complex     *core.Message
	Alert       *core.Message
	Templated   *core.Message
	MultiTarget *core.Message
}{
	Simple: func() *core.Message {
		msg := core.NewMessage()
		msg.Title = "Simple Test Message"
		msg.Body = "This is a simple test message"
		msg.Priority = core.Priority(3)
		return msg
	}(),

	Complex: func() *core.Message {
		msg := core.NewMessage()
		msg.Title = "Complex Test Message"
		msg.Body = "This is a complex message with multiple attributes"
		msg.Priority = core.Priority(4)
		msg.Format = core.FormatMarkdown
		if msg.Variables == nil {
			msg.Variables = make(map[string]interface{})
		}
		msg.Variables["user"] = "testuser"
		msg.Variables["action"] = "login"
		msg.Variables["timestamp"] = "2024-01-01T12:00:00Z"
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]string)
		}
		msg.Metadata["source"] = "api"
		msg.Metadata["type"] = "notification"
		msg.Metadata["env"] = "test"
		return msg
	}(),

	Alert: func() *core.Message {
		msg := core.NewMessage()
		msg.Title = "Critical System Alert"
		msg.Body = "Database connection lost"
		msg.Priority = core.Priority(5)
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]string)
		}
		msg.Metadata["severity"] = "critical"
		msg.Metadata["component"] = "database"
		if msg.Variables == nil {
			msg.Variables = make(map[string]interface{})
		}
		msg.Variables["error"] = "connection timeout"
		msg.Variables["retry_count"] = 3
		return msg
	}(),

	Templated: func() *core.Message {
		msg := core.NewMessage()
		msg.Template = "alert-template"
		msg.Title = "{{.service}} Alert: {{.status}}"
		msg.Body = "Service {{.service}} is {{.status}}. Details: {{.details}}"
		msg.Priority = core.Priority(3)
		if msg.Variables == nil {
			msg.Variables = make(map[string]interface{})
		}
		msg.Variables["service"] = "web-api"
		msg.Variables["status"] = "degraded"
		msg.Variables["details"] = "high latency detected"
		return msg
	}(),

	MultiTarget: func() *core.Message {
		msg := core.NewMessage()
		msg.Title = "Multi-Target Message"
		msg.Body = "This message will be sent to multiple targets"
		msg.Priority = core.Priority(3)
		msg.Targets = []core.Target{
			core.NewTarget(core.TargetTypeEmail, "user1@example.com", "email"),
			core.NewTarget(core.TargetTypeEmail, "user2@example.com", "email"),
			core.NewTarget(core.TargetTypeUser, "user123", "feishu"),
			core.NewTarget(core.TargetTypeGroup, "dev-team", "slack"),
			core.NewTarget(core.TargetTypeChannel, "general", "discord"),
		}
		return msg
	}(),
}

// TestTargets 测试目标集合
var TestTargets = struct {
	EmailTargets   []core.Target
	FeishuTargets  []core.Target
	SlackTargets   []core.Target
	DiscordTargets []core.Target
	MixedTargets   []core.Target
}{
	EmailTargets: []core.Target{
		core.NewTarget(core.TargetTypeEmail, "test@example.com", "email"),
		core.NewTarget(core.TargetTypeEmail, "admin@example.com", "email"),
		core.NewTarget(core.TargetTypeEmail, "support@example.com", "email"),
	},

	FeishuTargets: []core.Target{
		core.NewTarget(core.TargetTypeUser, "user123", "feishu"),
		core.NewTarget(core.TargetTypeUser, "user456", "feishu"),
		core.NewTarget(core.TargetTypeGroup, "dev-team", "feishu"),
		core.NewTarget(core.TargetTypeGroup, "ops-team", "feishu"),
	},

	SlackTargets: []core.Target{
		core.NewTarget(core.TargetTypeChannel, "general", "slack"),
		core.NewTarget(core.TargetTypeChannel, "alerts", "slack"),
		core.NewTarget(core.TargetTypeUser, "john.doe", "slack"),
		core.NewTarget(core.TargetTypeGroup, "engineering", "slack"),
	},

	DiscordTargets: []core.Target{
		core.NewTarget(core.TargetTypeChannel, "general", "discord"),
		core.NewTarget(core.TargetTypeChannel, "announcements", "discord"),
		core.NewTarget(core.TargetTypeUser, "user#1234", "discord"),
	},

	MixedTargets: []core.Target{
		core.NewTarget(core.TargetTypeEmail, "test@example.com", "email"),
		core.NewTarget(core.TargetTypeUser, "user123", "feishu"),
		core.NewTarget(core.TargetTypeChannel, "general", "slack"),
		core.NewTarget(core.TargetTypeChannel, "announcements", "discord"),
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
