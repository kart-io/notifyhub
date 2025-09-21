# Feishu Demo

本示例演示如何使用NotifyHub发送Feishu（飞书）通知。

## ⚠️ 重要说明

示例中的默认webhook URL和secret是**测试凭证，已失效**。要成功运行示例，必须使用**真实有效的飞书机器人凭证**。

## 🔧 配置步骤

### 第一步：创建飞书机器人

1. 在飞书群组中，点击右上角设置 → 群机器人
2. 点击"添加机器人" → "自定义机器人"
3. 设置机器人名称和描述
4. **开启"签名校验"**（重要！）
5. 复制生成的**Webhook URL**和**签名密钥**

### 第二步：配置凭证

#### 方法1：使用环境变量（推荐）

```bash
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/YOUR_WEBHOOK_ID"
export FEISHU_SECRET="your-actual-secret"
go run main.go
```

#### 方法2：直接修改代码

编辑 `main.go` 文件第35-38行，替换默认的webhook URL和secret。

## 🚀 运行示例

```bash
go run main.go
```

## ✅ 预期结果

**成功时：**

```
✅ 消息发送成功! 结果: ...
```

**失败时：**

```
❌ 发送失败: [PLATFORM:SENDING_FAILED] Feishu API error (code=19021): sign match fail...

📋 故障排除:
1. 确认webhook URL是否有效
2. 确认secret是否正确
3. 确认飞书机器人是否已启用
4. 尝试设置环境变量...
```

## 🔍 常见问题

### Q: 为什么总是提示"sign match fail"？

A: 这表示webhook凭证无效或已过期。请确保：

- 使用的是**真实创建的机器人凭证**
- 机器人已在群组中启用
- 签名校验功能已开启

### Q: 如何验证凭证是否正确？

A: 在飞书群组中查看机器人是否在线，并测试发送简单消息。

### Q: 可以不使用签名校验吗？

A: 为了安全，建议启用签名校验。NotifyHub已实现完整的签名算法。

## 📚 参考文档

- [飞书机器人开发指南](https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot)
- [飞书Webhook签名验证](https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN)
