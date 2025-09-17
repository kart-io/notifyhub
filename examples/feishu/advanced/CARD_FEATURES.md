# 飞书卡片功能指南

本文档介绍了NotifyHub中新增的飞书卡片功能，包括使用方法和高级示例。

## 功能概览

NotifyHub现在完全支持飞书的交互式卡片功能，包括：

- ✅ **默认卡片模板** - 基于标题、内容、元数据自动生成
- ✅ **完全自定义卡片** - 支持飞书官方卡片规范的所有元素
- ✅ **交互式元素** - 按钮、链接、分栏布局、图片等
- ✅ **多种卡片主题** - blue、green、red、orange、purple等
- ✅ **异步发送支持** - 支持同步和异步发送模式

## 快速开始

### 1. 简单卡片（推荐）

使用`NewCard()`创建简单卡片，系统会自动生成卡片布局：

```go
message := client.NewCard("📊 系统状态报告", "服务器运行状态良好").
    Metadata("服务器", "web-server-01").
    Metadata("CPU", "45%").
    Metadata("内存", "68%").
    Metadata("状态", "🟢 正常").
    Priority(3).
    FeishuGroup("default").
    Build()

results, err := hub.Send(ctx, message, nil)
```

### 2. 完全自定义卡片

使用`CardData()`设置完全自定义的卡片结构：

```go
customCardData := map[string]interface{}{
    "elements": []map[string]interface{}{
        {
            "tag": "div",
            "text": map[string]interface{}{
                "content": "**🚀 部署成功通知**",
                "tag":     "lark_md",
            },
        },
        {
            "tag": "action",
            "actions": []map[string]interface{}{
                {
                    "tag": "button",
                    "text": map[string]interface{}{
                        "content": "查看详情",
                        "tag":     "plain_text",
                    },
                    "type": "primary",
                    "url":  "https://example.com/details",
                },
            },
        },
    },
    "header": map[string]interface{}{
        "title": map[string]interface{}{
            "content": "部署通知",
            "tag":     "plain_text",
        },
        "template": "green",
    },
}

message := client.NewMessage().
    Format(notifiers.FormatCard).
    CardData(customCardData).
    FeishuGroup("default").
    Build()
```

## 高级示例

### 监控仪表板卡片

展示系统监控数据，包含多个指标和操作按钮：

```go
monitoringData := map[string]interface{}{
    "elements": []map[string]interface{}{
        {
            "tag": "div",
            "fields": []map[string]interface{}{
                {
                    "is_short": true,
                    "text": map[string]interface{}{
                        "content": "**CPU使用率**\n🟢 45%",
                        "tag":     "lark_md",
                    },
                },
                {
                    "is_short": true,
                    "text": map[string]interface{}{
                        "content": "**内存使用率**\n🟡 68%",
                        "tag":     "lark_md",
                    },
                },
            },
        },
        {
            "tag": "action",
            "actions": []map[string]interface{}{
                {
                    "tag": "button",
                    "text": map[string]interface{}{
                        "content": "查看详情",
                        "tag":     "plain_text",
                    },
                    "type": "primary",
                    "url":  "https://monitor.example.com/dashboard",
                },
            },
        },
    },
    "header": map[string]interface{}{
        "title": map[string]interface{}{
            "content": "系统监控",
            "tag":     "plain_text",
        },
        "template": "blue",
    },
}
```

### 事件处理卡片

用于紧急事件通知和处理流程：

```go
incidentData := map[string]interface{}{
    "elements": []map[string]interface{}{
        {
            "tag": "div",
            "text": map[string]interface{}{
                "content": "**🚨 紧急事件通知**",
                "tag":     "lark_md",
            },
        },
        {
            "tag": "div",
            "text": map[string]interface{}{
                "content": "**事件ID**: INC-2024-001\n**级别**: 🔴 P1 - 严重",
                "tag":     "lark_md",
            },
        },
        {
            "tag": "action",
            "actions": []map[string]interface{}{
                {
                    "tag": "button",
                    "text": map[string]interface{}{
                        "content": "认领处理",
                        "tag":     "plain_text",
                    },
                    "type": "primary",
                },
                {
                    "tag": "button",
                    "text": map[string]interface{}{
                        "content": "状态跟踪",
                        "tag":     "plain_text",
                    },
                    "type": "default",
                },
            },
        },
    },
    "header": map[string]interface{}{
        "template": "red",
    },
}
```

### 审批流程卡片

适用于各种审批场景：

```go
approvalData := map[string]interface{}{
    "elements": []map[string]interface{}{
        {
            "tag": "div",
            "text": map[string]interface{}{
                "content": "**📋 待审批申请**",
                "tag":     "lark_md",
            },
        },
        {
            "tag": "action",
            "actions": []map[string]interface{}{
                {
                    "tag": "button",
                    "text": map[string]interface{}{
                        "content": "✅ 批准",
                        "tag":     "plain_text",
                    },
                    "type": "primary",
                },
                {
                    "tag": "button",
                    "text": map[string]interface{}{
                        "content": "❌ 拒绝",
                        "tag":     "plain_text",
                    },
                    "type": "danger",
                },
            },
        },
    },
    "header": map[string]interface{}{
        "template": "orange",
    },
}
```

## API参考

### 新增方法

#### client包

- `NewCard(title, body string) *MessageBuilder` - 创建卡片消息构建器
- `CardData(cardData interface{}) *MessageBuilder` - 设置自定义卡片数据
- `AsCard() *MessageBuilder` - 将消息格式设置为卡片

#### notifiers包

- `FormatCard MessageFormat = "card"` - 卡片消息格式常量

### 卡片元素支持

支持飞书官方卡片规范的所有元素：

- **文本元素**: `div`, `markdown`, `plain_text`
- **布局元素**: `hr`, `fields`, `note`
- **交互元素**: `button`, `action`
- **媒体元素**: `img`
- **主题模板**: `blue`, `green`, `red`, `orange`, `purple`

## 运行示例

### 基础示例
```bash
cd examples/feishu/basic
go run main.go
```

### 高级示例（包含卡片功能）
```bash
cd examples/feishu/advanced
go run main.go
```

### 卡片专项测试
```bash
cd examples/feishu/advanced/card-demo
go run main.go
```

## 最佳实践

1. **优先使用简单卡片** - 对于大多数场景，使用`NewCard()`即可满足需求
2. **合理使用自定义卡片** - 仅在需要复杂布局或特殊交互时使用`CardData()`
3. **注意API限制** - 避免短时间内发送大量消息导致限流
4. **按钮数量控制** - 每行最多3个按钮，总数建议不超过6个
5. **内容长度控制** - 单个文本元素建议不超过1000字符

## 故障排除

### 常见问题

1. **卡片不显示** - 检查`CardData`结构是否符合飞书规范
2. **按钮无法点击** - 确认URL格式正确且可访问
3. **样式不生效** - 检查`template`字段是否使用支持的主题

### 错误码参考

- `9499` - 请求频率过高，建议增加发送间隔
- `1002` - 卡片格式错误，检查JSON结构
- `19021` - 签名验证失败，检查webhook配置

## 更多信息

- [飞书卡片官方文档](https://open.feishu.cn/document/ukTMukTMukTM/uczM3QjL3MzN04yNzMDN)
- [NotifyHub基础文档](../../../README.md)
- [高级功能示例](main.go)