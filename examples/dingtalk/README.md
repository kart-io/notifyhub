# 钉钉推送示例

这个示例展示了如何使用 NotifyHub 向钉钉群组发送消息。

## 功能特性

- 支持钉钉自定义机器人 Webhook
- 支持三种安全模式：无安全验证、签名验证、自定义关键词验证
- 支持多种消息格式：文本、Markdown、ActionCard
- 自动关键词添加（当配置了自定义关键词时）
- 完整的错误处理和重试机制

## 快速开始

### 1. 获取钉钉 Webhook URL

1. 在钉钉群中添加自定义机器人
2. 选择安全设置（推荐使用加签验证）
3. 复制生成的 Webhook URL

### 2. 设置环境变量

```bash
# 必需：钉钉 Webhook URL
export DINGTALK_WEBHOOK_URL="https://oapi.dingtalk.com/robot/send?access_token=your_token"

# 可选：签名密钥（如果启用了加签验证）
export DINGTALK_SECRET="your_secret_key"

# 可选：自定义关键词（如果启用了关键词验证）
export DINGTALK_KEYWORDS="通知"
```

### 3. 运行示例

```bash
cd examples/dingtalk
go run main.go
```

## 钉钉机器人安全配置

钉钉自定义机器人支持三种安全设置：

### 1. 无安全验证

最简单的配置，但安全性较低：

```go
dingTalkConfig := &config.DingTalkConfig{
    WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=your_token",
    Timeout:    30 * time.Second,
}
```

### 2. 加签验证（推荐）

使用 HMAC-SHA256 加签验证：

```go
dingTalkConfig := &config.DingTalkConfig{
    WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=your_token",
    Secret:     "your_secret_key", // 从钉钉机器人配置中获取
    Timeout:    30 * time.Second,
}
```

### 3. 自定义关键词

消息必须包含指定关键词：

```go
dingTalkConfig := &config.DingTalkConfig{
    WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=your_token",
    Keywords:   []string{"通知", "告警", "监控"},
    Timeout:    30 * time.Second,
}
```

**注意**: 如果消息不包含配置的关键词，系统会自动在消息开头添加第一个关键词。

## 支持的消息格式

### 文本消息

```go
msg := &message.Message{
    Title:  "通知标题",
    Body:   "这是消息内容",
    Format: message.FormatText,
    Targets: []target.Target{
        {Type: "dingtalk", Value: "default"},
    },
}
```

### Markdown 消息

```go
msg := &message.Message{
    Title: "📊 系统报告",
    Body:  "## 状态\n\n**CPU**: 65%\n**内存**: 78%",
    Format: message.FormatMarkdown,
    Targets: []target.Target{
        {Type: "dingtalk", Value: "report"},
    },
}
```

### ActionCard 消息

```go
msg := &message.Message{
    Title: "操作确认",
    PlatformData: map[string]interface{}{
        "dingtalk_actionCard": map[string]interface{}{
            "title":       "部署确认",
            "text":        "## 部署信息\n\n请确认是否继续？",
            "singleTitle": "确认",
            "singleURL":   "https://example.com/confirm",
        },
    },
    Targets: []target.Target{
        {Type: "dingtalk", Value: "deployment"},
    },
}
```

## 错误处理

钉钉 API 可能返回以下错误码：

- `310000`: 关键词不匹配
- `310001`: 签名验证失败
- `400001`: 参数错误
- `400002`: 请求体格式错误
- `430101`: 发送消息失败

所有错误都会在 `SendResult.Error` 中返回详细信息。

## 最佳实践

1. **安全性**: 推荐使用加签验证，避免使用无安全验证模式
2. **频率限制**: 钉钉对消息发送频率有限制，建议控制发送频率
3. **消息格式**: 根据内容选择合适的消息格式
4. **错误处理**: 始终检查发送结果，妥善处理失败情况
5. **关键词管理**: 如果使用关键词验证，确保关键词与消息内容相关

## 参考文档

- [钉钉自定义机器人开发文档](https://developers.dingtalk.com/document/app/custom-robot-access)
- [NotifyHub 架构文档](../../docs/ARCHITECTURE.md)
- [配置说明文档](../../docs/CONFIGURATION.md)