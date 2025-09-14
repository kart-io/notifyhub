# 🚀 NotifyHub 多平台通知支持

NotifyHub 原生支持同时向多个通知平台发送消息，包括飞书和邮件。系统会自动处理不同平台的格式要求和发送逻辑。

## ✨ 支持的平台

- **飞书 (Feishu)**: 支持群组和个人消息
- **邮件 (Email)**: 支持 SMTP 协议的邮件服务
- **扩展性**: 可轻松添加新的通知平台

## 🎯 多平台通知方式

### 1. 自动路由（推荐）

通过配置路由规则，系统会根据消息的优先级、类型等属性自动选择发送平台：

```go
hub, err := notifyhub.New(
    notifyhub.WithFeishu("webhook-url", "secret"),
    notifyhub.WithEmail("smtp.host.com", 587, "user", "pass", "from@email.com"),
    notifyhub.WithRouting(
        // 高优先级消息同时发送到飞书和邮件
        notifyhub.NewRoutingRule("high_priority_all").
            WithPriority(4, 5).
            RouteTo("feishu", "email").
            Build(),
        // 普通消息只发送到飞书
        notifyhub.NewRoutingRule("normal_feishu").
            WithPriority(1, 2, 3).
            RouteTo("feishu").
            Build(),
    ),
)

// 高优先级消息会自动发送到两个平台
message := notifyhub.NewAlert("紧急告警", "系统故障").
    Priority(5). // 触发高优先级路由规则
    Build()

results, err := hub.Send(ctx, message, nil)
// results 将包含两个平台的发送结果
```

### 2. 手动指定目标

直接在消息中指定要发送到的具体平台和目标：

```go
message := notifyhub.NewNotice("项目更新", "新版本发布").
    // 指定飞书目标
    FeishuGroup("dev-team").
    FeishuUser("zhang_san").
    // 指定邮件目标
    Email("team@company.com").
    Email("manager@company.com").
    Build()

results, err := hub.Send(ctx, message, nil)
// 消息会同时发送到所有指定的目标
```

### 3. 混合使用

结合路由规则和手动指定，实现灵活的发送策略：

```go
message := notifyhub.NewAlert("系统告警", "数据库连接失败").
    Priority(4).              // 触发路由规则
    Email("dba@company.com"). // 额外发送给DBA
    Metadata("type", "database"). // 可能触发特定路由规则
    Build()
```

## 📊 发送结果处理

每次发送会返回详细的结果信息：

```go
results, err := hub.Send(ctx, message, nil)
if err != nil {
    log.Printf("发送失败: %v", err)
} else {
    for _, result := range results {
        log.Printf("平台: %s, 成功: %v, 耗时: %v",
            result.Platform, result.Success, result.Duration)

        if !result.Success {
            log.Printf("错误: %s", result.Error)
        }
    }
}
```

## 🔄 异步多平台发送

支持异步发送到多个平台，提高性能：

```go
// 异步发送到多个平台
taskID, err := hub.SendAsync(ctx, message, notifyhub.NewAsyncOptions())
if err != nil {
    log.Printf("异步发送失败: %v", err)
} else {
    log.Printf("任务ID: %s", taskID)
}
```

## 📝 不同格式支持

不同平台会自动处理相应的消息格式：

```go
// Markdown 格式（飞书原生支持）
markdownMsg := notifyhub.NewMarkdown("标题", `
## 更新内容
- 新功能 A
- 优化 B
- 修复 C
`).FeishuGroup("team").Email("team@company.com").Build()

// HTML 格式（邮件支持）
htmlMsg := notifyhub.NewHTML("标题", `
<h2>更新内容</h2>
<ul>
  <li>新功能 A</li>
  <li>优化 B</li>
  <li>修复 C</li>
</ul>
`).Email("team@company.com").Build()
```

## ⚡ 性能特性

- **并行发送**: 多个平台同时发送，不会阻塞
- **独立超时**: 每个平台可以设置独立的超时时间
- **失败隔离**: 一个平台失败不影响其他平台
- **重试机制**: 支持失败重试，可针对不同平台配置不同策略

## 🛠️ 配置示例

完整的多平台配置示例：

```go
hub, err := notifyhub.New(
    // 飞书配置
    notifyhub.WithFeishu(
        "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook",
        "your-secret",
    ),

    // 邮件配置
    notifyhub.WithEmail(
        "smtp.gmail.com", // SMTP服务器
        587,              // 端口
        "user@gmail.com", // 用户名
        "app-password",   // 密码
        "from@gmail.com", // 发送方
    ),

    // 队列配置（支持异步）
    notifyhub.WithQueue("memory", 1000, 3),

    // 路由配置
    notifyhub.WithRouting(
        // 紧急告警 -> 所有平台
        notifyhub.NewRoutingRule("emergency").
            WithPriority(5).
            WithMessageType("alert").
            RouteTo("feishu", "email").
            Build(),

        // 普通通知 -> 仅飞书
        notifyhub.NewRoutingRule("normal").
            WithPriority(1, 2, 3).
            RouteTo("feishu").
            Build(),
    ),
)
```

## 🔍 监控和调试

系统提供详细的监控信息：

```go
// 获取发送统计
metrics := hub.GetMetrics()
log.Printf("总发送数: %v", metrics["total_sent"])
log.Printf("成功率: %.2f%%", metrics["success_rate"].(float64)*100)

// 健康检查
health := hub.GetHealth(ctx)
log.Printf("系统状态: %s", health["status"])
```

## 📚 更多示例

查看以下示例文件了解详细用法：

- `examples/multi-platform/main.go` - 基础多平台示例
- `examples/multi-platform-demo/main.go` - 完整演示
- `examples/config/main.go` - 配置示例
- `examples/advanced/main.go` - 高级用法

---

💡 **总结**: NotifyHub 提供了灵活且强大的多平台通知能力，支持自动路由、手动指定、异步发送等多种方式，满足各种业务场景的需求。