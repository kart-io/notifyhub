# 飞书通知示例

这个示例展示了如何使用 NotifyHub 向飞书发送各种类型的通知消息。

## 功能特性

✨ **完整功能演示**:

- 📝 简单文本消息
- 📋 Markdown 格式消息
- 🎴 飞书卡片消息
- 📤 批量发送
- ⚡ 异步发送
- 🔍 健康检查

## 快速开始

### 1. 配置环境变量

```bash
# 必需配置
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-token"

# 可选配置（用于签名验证）
export FEISHU_SECRET="your-webhook-secret"
```

### 2. 运行示例

```bash
cd examples/feishu
go run main.go
```

## 飞书 Webhook 配置

### 创建自定义机器人

1. 在飞书群聊中，点击右上角设置图标
2. 选择"群机器人" -> "添加机器人" -> "自定义机器人"
3. 设置机器人名称和描述
4. 复制生成的 Webhook URL
5. 可选：设置安全校验的签名密钥

### Webhook URL 格式

```bash
https://open.feishu.cn/open-apis/bot/v2/hook/[TOKEN]
```

## 消息类型说明

### 1. 文本消息

发送简单的纯文本消息到飞书群聊：

```go
message := notifyhub.NewMessage("通知标题").
    WithText("消息内容").
    ToFeishuGroup("oc_group_id").
    Build()
```

### 2. Markdown 消息

支持 Markdown 格式的富文本消息：

```go
message := notifyhub.NewAlert("状态报告").
    WithMarkdown("## 标题\n**粗体文本**\n- 列表项").
    ToFeishu("ou_user_id").
    Build()
```

### 3. 卡片消息

发送交互式卡片消息，支持按钮、链接等元素：

```go
cardContent := map[string]interface{}{
    "config": map[string]interface{}{
        "wide_screen_mode": true,
    },
    "elements": []map[string]interface{}{
        // 卡片元素配置
    },
}

message := notifyhub.NewMessage("卡片通知").
    WithFeishuCard(cardContent).
    ToFeishuGroup("oc_group_id").
    Build()
```

## 目标类型

### 用户 ID (ou_)

向特定用户发送私聊消息：

```go
.ToFeishu("ou_1234567890abcdef")
```

### 群组 ID (oc_)

向群聊发送消息：

```go
.ToFeishuGroup("oc_group1234567890")
```

### 提及用户

在消息中 @特定用户：

```go
.WithFeishuMentions("ou_user1", "ou_user2")
.WithFeishuMentionAll() // @所有人
```

### 自定义关键词

飞书机器人支持设置自定义关键词来触发消息发送。NotifyHub 提供了多种方式来配置关键词：

#### 使用 WithFeishuKeywords

```go
// 单个关键词
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuKeywords(webhookURL, secret, "告警"),
)

// 多个关键词（使用第一个作为触发词）
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuKeywords(webhookURL, secret, "通知", "系统", "监控"),
)
```

#### 使用配置映射

```go
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuFromMap(map[string]interface{}{
        "webhook_url": webhookURL,
        "secret":      secret,
        "keywords":    []string{"部署", "发布", "更新"},
    }),
)

// 或者单个关键词
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuFromMap(map[string]interface{}{
        "webhook_url": webhookURL,
        "secret":      secret,
        "keywords":    "告警", // 字符串形式
    }),
)
```

#### 关键词行为

- 关键词会自动添加到消息文本的开头
- 如果设置了多个关键词，默认使用第一个作为触发词
- 关键词必须在飞书机器人设置中预先配置
- 不设置关键词时，消息不会添加任何前缀

#### ⚠️ 重要：签名验证与关键词互斥

**签名验证和自定义关键词是两种互斥的安全机制，不能同时使用：**

```go
// ✅ 正确：只使用签名验证
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishu(webhookURL, "your-secret"),
)

// ✅ 正确：只使用关键词
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuKeywords(webhookURL, "", "告警"),
)

// ❌ 错误：同时使用两种机制会报错
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuKeywords(webhookURL, "secret", "告警"), // 会报错
)
```

## 三种安全模式

飞书机器人支持三种互斥的安全模式：

### 🔒 模式1: 签名验证模式（高安全性）

```go
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishu(webhookURL, "your-secret"),
)
```

- **特点**: 使用 HMAC-SHA256 签名验证
- **适用场景**: 生产环境、高安全性要求
- **消息处理**: 自动添加时间戳和签名字段
- **安全级别**: ⭐⭐⭐⭐⭐

### 🔑 模式2: 关键词触发模式（内容过滤）

```go
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuKeywords(webhookURL, "", "告警"),
)
```

- **特点**: 消息开头自动添加关键词
- **适用场景**: 需要关键词过滤的场景
- **消息处理**: 在消息文本开头添加指定关键词
- **安全级别**: ⭐⭐⭐

### 🌐 模式3: 普通模式（无安全验证）

```go
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishu(webhookURL, ""),
)
```

- **特点**: 消息直接发送，无额外处理
- **适用场景**: 内网环境、测试场景
- **消息处理**: 消息内容不做任何修改
- **安全级别**: ⭐

### 模式选择建议

| 场景 | 推荐模式 | 原因 |
|------|----------|------|
| 生产环境 | 签名验证模式 | 最高安全性，防止恶意请求 |
| 关键词过滤 | 关键词触发模式 | 确保消息包含必要的触发词 |
| 内网测试 | 普通模式 | 简单直接，无额外配置 |
| 开发调试 | 普通模式 | 快速测试，无安全限制 |

**示例效果:**

```
// 设置关键词 "告警"
消息: "数据库连接失败"
实际发送: "告警 数据库连接失败"

// 不设置关键词
消息: "数据库连接失败"
实际发送: "数据库连接失败"
```

## 示例输出

运行示例时，你将看到类似以下的输出：

```
🚀 飞书通知示例
===============

📝 示例1: 简单文本消息
------------------
✅ 消息发送成功 - ID: msg_1234567890, 状态: success

📋 示例2: Markdown 格式消息
------------------------
✅ Markdown消息发送成功 - ID: msg_1234567891

🎴 示例3: 飞书卡片消息
--------------------
✅ 卡片消息发送成功 - ID: msg_1234567892

📤 示例4: 批量发送
----------------
✅ 批量发送完成 - 总计: 4, 成功: 4, 失败: 0
  ✅ ou_user1 -> feishu (45ms)
  ✅ ou_user2 -> feishu (52ms)
  ✅ oc_group1 -> feishu (38ms)
  ✅ oc_group2 -> feishu (41ms)

⚡ 示例5: 异步发送
----------------
✅ 异步发送已提交 - ID: msg_1234567893, 状态: queued
  队列时间: 14:30:25

🔍 示例6: 系统健康检查
--------------------
✅ 系统状态: healthy
  检查时间: 2024-01-20 14:30:25
  feishu: 🟢 健康

✨ 飞书通知示例运行完成!
========================
```

## 错误处理

示例包含了完善的错误处理机制：

- **配置验证**: 检查必需的环境变量
- **发送失败**: 显示详细的错误信息
- **连接超时**: 10秒超时保护
- **资源清理**: 程序结束时正确关闭连接

## 高级功能

### 消息优先级

```go
// 普通消息
notifyhub.NewMessage("标题")

// 警告消息
notifyhub.NewAlert("警告标题")

// 紧急消息
notifyhub.NewUrgent("紧急标题")
```

### 元数据和变量

```go
message := notifyhub.NewMessage("通知").
    WithMetadata("source", "monitoring_system").
    WithVariable("alert_level", "warning").
    Build()
```

### 批量目标

```go
message := notifyhub.NewMessage("批量通知").
    ToFeishu("ou_user1").
    ToFeishu("ou_user2").
    ToFeishuGroup("oc_group1").
    ToFeishuGroup("oc_group2").
    Build()
```

## 最佳实践

1. **环境隔离**: 不同环境使用不同的 Webhook URL
2. **错误重试**: 对网络错误实施重试机制
3. **速率限制**: 避免频繁发送消息被限流
4. **消息去重**: 使用元数据标识避免重复发送
5. **监控告警**: 监控发送成功率和响应时间

## 故障排除

### 常见问题

1. **Webhook URL 无效**

   ```
   FEISHU_WEBHOOK_URL 格式错误或已过期
   ```

2. **签名验证失败**

   ```
   检查 FEISHU_SECRET 配置是否正确
   ```

3. **发送超时**

   ```
   网络连接问题或飞书服务不可用
   ```

4. **卡片格式错误**

   ```
   检查卡片 JSON 结构是否符合飞书规范
   ```

### 调试模式

设置详细日志输出：

```bash
export LOG_LEVEL=debug
go run main.go
```

## 相关链接

- [飞书开放平台文档](https://open.feishu.cn/document/)
- [自定义机器人指南](https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN)
- [消息卡片搭建工具](https://open.feishu.cn/tool/cardbuilder)
- [NotifyHub 文档](../../README.md)
