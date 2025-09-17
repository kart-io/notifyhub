# 飞书@人功能使用指南

本文档介绍了NotifyHub中新增的飞书@人功能，包括使用方法和实用示例。

## 功能概览

NotifyHub现在完全支持飞书的@人功能，包括：

- ✅ **@单个用户** - 支持文本、富文本和卡片消息
- ✅ **@多个用户** - 一次@多个用户ID
- ✅ **@所有人** - 发送@all通知
- ✅ **自定义显示名称** - 可以设置@人时显示的用户名
- ✅ **多种消息格式支持** - 文本、富文本、卡片消息均支持@人

## 快速开始

### 1. @单个用户

```go
message := client.NewMessage().
    Title("📢 重要通知").
    Body("系统将于今晚22:00进行维护，预计耗时30分钟。").
    AtUser("ou_123456789", "张三").  // 用户ID和显示名称
    Priority(4).
    FeishuGroup("default").
    Build()

results, err := hub.Send(ctx, message, nil)
```

### 2. @多个用户

```go
message := client.NewMessage().
    Title("🚨 紧急事件").
    Body("生产环境出现异常，请相关人员立即处理！").
    AtUsers("ou_111111111", "ou_222222222", "ou_333333333").
    Priority(5).
    FeishuGroup("default").
    Build()
```

### 3. @所有人

```go
message := client.NewCard("🎉 发版通知", "新版本v2.1.0已成功发布！").
    AtAll().
    Priority(3).
    FeishuGroup("default").
    Build()
```

### 4. 卡片消息@人

```go
cardMessage := client.NewCard("📋 任务分配", "请及时处理").
    Metadata("优先级", "高").
    Metadata("截止时间", "2024-12-31").
    AtUser("ou_444444444", "李四").
    Priority(4).
    FeishuGroup("default").
    Build()
```

## API参考

### MessageBuilder新增方法

#### AtUser(userID string, userName ...string)
@指定用户
- `userID`: 用户的飞书ID（必填）
- `userName`: 显示名称（可选，如果不提供则显示为"用户"）

```go
builder.AtUser("ou_123456789")                // 只指定用户ID
builder.AtUser("ou_123456789", "张三")        // 指定用户ID和显示名称
```

#### AtUsers(userIDs ...string)
@多个用户（只传用户ID）

```go
builder.AtUsers("ou_111111111", "ou_222222222", "ou_333333333")
```

#### AtAll()
@所有人

```go
builder.AtAll()
```

#### AtMentions(mentions []notifiers.AtMention)
直接设置@人列表

```go
mentions := []notifiers.AtMention{
    {UserID: "ou_123456789", UserName: "张三", IsAll: false},
    {UserID: "all", IsAll: true},
}
builder.AtMentions(mentions)
```

#### ClearAtMentions()
清除所有@人设置

```go
builder.ClearAtMentions()
```

### AtMention结构体

```go
type AtMention struct {
    UserID   string `json:"user_id"`   // 用户ID，@所有人时为"all"
    UserName string `json:"user_name"` // 用户显示名称（可选）
    IsAll    bool   `json:"is_all"`    // 是否@所有人
}
```

## 消息格式支持

### 文本消息（FormatText）
@人会添加到消息内容的开头：
```
<at user_id="ou_123456789">张三</at> 您好，这是一条通知消息。
```

### 富文本消息（FormatMarkdown）
@人会作为富文本元素插入到第一行：
```json
{
  "msg_type": "post",
  "content": {
    "post": {
      "zh_cn": {
        "content": [
          [
            {"tag": "at", "user_id": "ou_123456789"},
            {"tag": "text", "text": " "},
            {"tag": "text", "text": "原消息内容..."}
          ]
        ]
      }
    }
  }
}
```

### 卡片消息（FormatCard）
@人会作为独立的div元素插入到卡片的第一个位置：
```json
{
  "msg_type": "interactive",
  "card": {
    "elements": [
      {
        "tag": "div",
        "text": {
          "content": "<at user_id=\"ou_123456789\">张三</at>",
          "tag": "lark_md"
        }
      },
      // 原有卡片元素...
    ]
  }
}
```

## 用户ID获取方法

### 方法1: 通过飞书管理后台
1. 登录飞书管理后台
2. 进入「通讯录」
3. 点击用户名称
4. 在URL中找到用户ID（ou_开头）

### 方法2: 通过API查询
```bash
curl -X GET \
  'https://open.feishu.cn/open-apis/contact/v3/users' \
  -H 'Authorization: Bearer <access_token>'
```

### 方法3: 在群聊中@用户
1. 在群聊中@用户
2. 查看消息的JSON格式
3. 提取user_id字段

## 最佳实践

### 1. 合理使用@人功能
- **重要通知**: 使用@所有人
- **任务分配**: @相关负责人
- **紧急事件**: @多个关键人员
- **日常提醒**: @单个用户

### 2. 避免过度@人
- 避免频繁@所有人，影响用户体验
- 根据消息重要性选择@人范围
- 在非工作时间谨慎使用@功能

### 3. 用户ID管理
- 建立用户ID映射表
- 定期更新用户信息
- 处理用户离职等情况

### 4. 错误处理
```go
message := client.NewMessage().
    Title("通知").
    Body("内容").
    AtUser("invalid_user_id").  // 无效的用户ID
    FeishuGroup("default").
    Build()

results, err := hub.Send(ctx, message, nil)
if err != nil {
    log.Printf("发送失败: %v", err)
}

// 检查发送结果
for _, result := range results {
    if !result.Success {
        log.Printf("@人消息发送失败: %s", result.Error)
    }
}
```

## 故障排除

### 常见问题

1. **@人不生效**
   - 检查用户ID格式是否正确（应以ou_开头）
   - 确认用户在目标群组中
   - 验证机器人权限

2. **消息格式异常**
   - 检查@人标签格式
   - 验证JSON结构
   - 确认平台支持@人功能

3. **用户收不到通知**
   - 检查用户通知设置
   - 确认用户状态（在线/离线）
   - 验证群组权限

### 调试技巧

```go
// 启用调试模式查看详细信息
message := client.NewMessage().
    Title("调试测试").
    Body("测试内容").
    AtUser("ou_123456789", "测试用户").
    Debug().  // 启用调试输出
    FeishuGroup("default").
    Build()
```

## 示例代码

完整的@人功能示例代码位于：
- `examples/feishu/advanced/card-demo/main.go` - testAtMentionFeatures函数

运行示例：
```bash
cd examples/feishu/advanced/card-demo
go run main.go
```

## 相关文档

- [卡片功能详细文档](CARD_FEATURES.md)
- [NotifyHub基础使用](../basic/)
- [飞书开放平台文档](https://open.feishu.cn/document/)