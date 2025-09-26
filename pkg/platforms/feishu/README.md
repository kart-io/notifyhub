# Feishu Platform Integration

这个包为 NotifyHub 提供飞书（Lark）平台集成支持。

## 功能特性

- 支持飞书 webhook 消息发送
- 支持多种消息格式：文本、Markdown、卡片、富文本
- 自动平台注册机制
- 完整的错误处理和重试支持
- 健康检查功能

## 三种安全配置模式

飞书自定义机器人支持三种安全配置模式，本实现完全支持飞书官方的安全策略：

### 情况1: 无需配置签名与自定义关键词

最简单的配置方式，直接发送消息无需额外验证：

```go
feishuConfig := &config.FeishuConfig{
    WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-id",
    Timeout:    30 * time.Second,
}
sender, err := feishu.NewFeishuPlatform(feishuConfig, logger)
```

### 情况2: 配置签名验证

使用 HMAC-SHA256 签名验证确保消息安全：

```go
feishuConfig := &config.FeishuConfig{
    WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-id",
    Secret:     "your-webhook-secret", // 飞书机器人安全设置中的签名密钥
    Timeout:    30 * time.Second,
}
sender, err := feishu.NewFeishuPlatform(feishuConfig, logger)
```

### 情况3: 自定义关键词验证

配置关键词后，消息必须包含指定关键词才能发送成功：

```go
feishuConfig := &config.FeishuConfig{
    WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-id",
    Keywords:   []string{"报警", "通知", "测试"}, // 自定义关键词列表
    Timeout:    30 * time.Second,
}
sender, err := feishu.NewFeishuPlatform(feishuConfig, logger)
```

**注意**: 如果消息内容不包含任何配置的关键词，系统会自动在消息开头添加第一个关键词。

## 配置

### 使用强类型配置（推荐）

```go
import (
    "github.com/kart-io/notifyhub/pkg/notifyhub/config"
    "github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

// 完整配置示例（同时启用签名和关键词验证）
feishuConfig := &config.FeishuConfig{
    WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-id",
    Secret:     "your-webhook-secret", // 可选：签名验证密钥
    Keywords:   []string{"报警", "通知"}, // 可选：自定义关键词
    Timeout:    30 * time.Second,
}

sender, err := feishu.NewFeishuPlatform(feishuConfig, logger)
```

### 使用映射配置（向下兼容）

```go
// 基础配置
config := map[string]interface{}{
    "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-id",
    "secret":      "your-webhook-secret", // 可选：签名验证
    "keywords":    []string{"报警", "通知"}, // 可选：自定义关键词
    "timeout":     "30s",
}

// 也支持逗号分隔的关键词字符串格式
config2 := map[string]interface{}{
    "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-id",
    "keywords":    "报警,通知,测试", // 逗号分隔的关键词
    "timeout":     "30s",
}

sender, err := feishu.NewFeishuSender(config, logger)
```

## 消息格式

### 文本消息

```go
msg := &message.Message{
    Title:  "通知标题",
    Body:   "这是消息内容",
    Format: message.FormatText,
}
```

### Markdown 消息

```go
msg := &message.Message{
    Title:  "Markdown 通知",
    Body:   "**粗体文本** 和 *斜体文本*\n\n- 列表项 1\n- 列表项 2",
    Format: message.FormatMarkdown,
}
```

### 卡片消息

```go
msg := &message.Message{
    Title: "卡片消息",
    PlatformData: map[string]interface{}{
        "feishu_card": map[string]interface{}{
            "elements": []interface{}{
                map[string]interface{}{
                    "tag": "div",
                    "text": map[string]interface{}{
                        "tag":     "lark_md",
                        "content": "🎉 **卡片标题**\n\n这是卡片内容",
                    },
                },
            },
        },
    },
}
```

## 目标类型

支持以下目标类型：
- `feishu` - 飞书目标
- `webhook` - 通用 webhook 目标

```go
targets := []target.Target{
    {Type: "feishu", Value: "group-chat-id"},
    {Type: "webhook", Value: "webhook-identifier"},
}
```

## 测试

### 单元测试

```bash
go test ./pkg/platforms/feishu/...
```

### 集成测试

如需测试真实的飞书 webhook 发送，请设置环境变量：

```bash
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-id"
export FEISHU_SECRET="your-webhook-secret"  # 可选

go test -tags=integration ./pkg/platforms/feishu/...
```

## 获取飞书 Webhook URL

1. 在飞书中创建一个群组
2. 进入群组设置 → 群机器人 → 添加机器人
3. 选择"自定义机器人"
4. 设置机器人名称和描述
5. 复制生成的 Webhook URL
6. 如需签名验证，请记录安全设置中的密钥

## 限制和注意事项

### 消息限制
- 单条消息最大 4000 字符
- 支持的消息格式：text, markdown, card, rich_text
- 网络超时默认为 30 秒

### 安全功能
- ✅ **签名验证**: 使用 HMAC-SHA256 签名，完全兼容飞书官方标准
- ✅ **自定义关键词**: 自动检测和添加关键词，支持多种消息格式
- ✅ **组合安全**: 支持同时启用签名验证和自定义关键词

### 关键词处理机制
- 系统会自动检查消息内容是否包含配置的关键词（不区分大小写）
- 如果不包含，会在消息开头自动添加第一个关键词
- 对于卡片消息，关键词添加功能尚未完全实现（会显示警告）

## 错误处理

平台会自动处理以下错误情况：
- 网络连接超时
- HTTP 状态码错误
- 消息格式验证错误
- 目标类型不匹配

所有错误都会在 `SendResult` 中返回详细信息。