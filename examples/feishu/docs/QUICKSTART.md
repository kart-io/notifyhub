# 飞书通知快速启动指南

## 🚀 一分钟快速体验

### 方法1: 演示模式（推荐新手）

```bash
# 1. 进入示例目录
cd examples/feishu

# 2. 运行演示
make demo
```

演示将使用模拟数据展示所有功能，无需真实的飞书配置。

### 方法2: 真实环境

```bash
# 1. 配置飞书 Webhook
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your-token"
export FEISHU_SECRET="your-secret"  # 可选

# 2. 运行示例
make run
```

## 📋 完整设置步骤

### 1. 创建飞书机器人

1. **进入飞书群聊**
   - 打开目标群聊
   - 点击右上角设置图标 ⚙️

2. **添加机器人**
   - 选择"群机器人"
   - 点击"添加机器人"
   - 选择"自定义机器人"

3. **配置机器人**
   - 设置机器人名称：`NotifyHub Bot`
   - 设置描述：`系统通知机器人`
   - 选择头像（可选）

4. **获取配置信息**
   - 复制 Webhook URL
   - 设置签名密钥（建议开启）
   - 保存配置

### 2. 配置环境变量

```bash
# 必需配置
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxxxx"

# 可选配置（推荐使用）
export FEISHU_SECRET="your-signature-secret"
```

### 3. 运行示例

```bash
# 方法1: 使用 Makefile
make run

# 方法2: 直接运行
go run main.go

# 方法3: 构建后运行
make build
./feishu-example
```

## 🎯 核心功能预览

### 文本消息

```go
message := notifyhub.NewMessage("通知标题").
    WithText("消息内容").
    ToFeishuGroup("oc_group_id").
    Build()
```

### Markdown 消息

```go
message := notifyhub.NewAlert("状态报告").
    WithMarkdown("## 系统状态\n**运行正常** ✅").
    ToFeishu("ou_user_id").
    Build()
```

### 卡片消息

```go
cardContent := map[string]interface{}{
    "config": map[string]interface{}{
        "wide_screen_mode": true,
    },
    "elements": []map[string]interface{}{
        // 卡片内容
    },
}

message := notifyhub.NewMessage("部署通知").
    WithFeishuCard(cardContent).
    ToFeishuGroup("oc_devops").
    Build()
```

## 📊 预期输出

运行成功后，你将看到：

```
🚀 飞书通知示例
===============

📝 示例1: 简单文本消息
------------------
✅ 消息发送成功 - ID: msg_xxxx, 状态: success

📋 示例2: Markdown 格式消息
------------------------
✅ Markdown消息发送成功 - ID: msg_xxxx

🎴 示例3: 飞书卡片消息
--------------------
✅ 卡片消息发送成功 - ID: msg_xxxx

📤 示例4: 批量发送
----------------
✅ 批量发送完成 - 总计: 4, 成功: 4, 失败: 0

⚡ 示例5: 异步发送
----------------
✅ 异步发送已提交 - ID: msg_xxxx, 状态: queued

🔍 示例6: 系统健康检查
--------------------
✅ 系统状态: healthy
```

## 🛠️ 可用命令

| 命令 | 功能 | 说明 |
|------|------|------|
| `make help` | 显示帮助 | 查看所有可用命令 |
| `make demo` | 运行演示 | 使用模拟数据，无需真实配置 |
| `make run` | 运行示例 | 需要设置环境变量 |
| `make build` | 构建程序 | 生成可执行文件 |
| `make clean` | 清理文件 | 删除构建产物 |
| `make check` | 完整检查 | 格式化、检查、构建 |

## ❌ 常见问题

### 1. 环境变量未设置

```
❌ 错误: 请设置环境变量 FEISHU_WEBHOOK_URL
```

**解决方案**:

```bash
export FEISHU_WEBHOOK_URL="your-webhook-url"
```

### 2. Webhook URL 格式错误

```
❌ 发送失败: invalid webhook URL format
```

**解决方案**: 确保 URL 格式正确

```
https://open.feishu.cn/open-apis/bot/v2/hook/[TOKEN]
```

### 3. 签名验证失败

```
❌ 发送失败: signature verification failed
```

**解决方案**: 检查签名密钥配置

```bash
export FEISHU_SECRET="your-correct-secret"
```

### 4. 网络连接超时

```
❌ 发送失败: context deadline exceeded
```

**解决方案**: 检查网络连接或增加超时时间

## 🔧 自定义配置

### 修改超时时间

```go
hub, err := notifyhub.NewHub(
    notifyhub.WithFeishuFromMap(config),
    notifyhub.WithTimeout(30000), // 30秒
)
```

### 修改目标用户/群组

编辑 `main.go` 中的 ID：

```go
// 用户 ID（以 ou_ 开头）
.ToFeishu("ou_your_user_id")

// 群组 ID（以 oc_ 开头）
.ToFeishuGroup("oc_your_group_id")
```

### 添加消息内容

```go
message := notifyhub.NewMessage("自定义标题").
    WithText("自定义消息内容").
    WithMetadata("source", "my_system").
    // 添加更多配置...
    Build()
```

## 📚 进阶使用

- [完整文档](./README.md)
- [API 参考](../../README.md)
- [飞书开放平台文档](https://open.feishu.cn/document/)
- [消息卡片搭建工具](https://open.feishu.cn/tool/cardbuilder)

## 🆘 获取帮助

如果遇到问题：

1. 查看 [README.md](./README.md) 详细文档
2. 检查飞书机器人配置
3. 验证网络连接
4. 查看程序输出的错误信息
