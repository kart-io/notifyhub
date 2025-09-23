# 飞书通知故障排除指南

## 🔍 常见问题和解决方案

### 1. 显示"发送成功"但飞书群聊没有收到消息

**症状**: 程序显示 `✅ 消息发送成功` 但飞书群聊中没有看到消息

**原因分析**:
- Webhook URL 无效或已过期
- 机器人未添加到目标群聊
- 签名验证失败
- 目标 ID (用户/群组) 不正确

**解决步骤**:

1. **验证 Webhook URL**:
   ```bash
   # 运行直接测试工具
   go run debug-sender.go
   ```

2. **检查签名验证**:
   ```bash
   # 查看详细的网络请求和响应
   FEISHU_WEBHOOK_URL="your-url" FEISHU_SECRET="your-secret" go run debug-sender.go
   ```

3. **确认机器人配置**:
   - 在飞书群聊设置中确认机器人已添加
   - 检查机器人权限设置
   - 确认 Webhook URL 对应正确的群聊

### 2. 签名验证失败

**错误信息**:
```json
{"code":19021,"msg":"sign match fail or timestamp is not within one hour from current time"}
```

**解决方案**:

1. **检查签名密钥**:
   ```bash
   # 确保密钥正确设置
   export FEISHU_SECRET="your-correct-secret"
   ```

2. **检查系统时间**:
   ```bash
   # 确保系统时间准确（误差不超过1小时）
   date
   ```

3. **验证签名算法**:
   - 飞书使用 SHA256 哈希算法
   - 签名字符串格式: `timestamp + "\n" + secret`

### 3. 网络连接问题

**症状**: 超时或连接失败

**解决步骤**:

1. **测试网络连通性**:
   ```bash
   curl -I https://open.feishu.cn/open-apis/bot/v2/hook/your-token
   ```

2. **检查防火墙设置**:
   - 确保出站 HTTPS (443端口) 连接被允许
   - 检查企业网络代理设置

3. **增加超时时间**:
   ```go
   hub, err := notifyhub.NewHub(
       notifyhub.WithFeishuFromMap(config),
       notifyhub.WithTimeout(30000), // 30秒
   )
   ```

### 4. 目标 ID 不正确

**症状**: 发送成功但特定用户/群组没收到

**解决方案**:

1. **获取正确的用户 ID**:
   - 用户 ID 格式: `ou_xxxxxxxxxxxxxxxx`
   - 在飞书管理后台或通过API获取

2. **获取正确的群组 ID**:
   - 群组 ID 格式: `oc_xxxxxxxxxxxxxxxx`
   - 通过群聊设置或API获取

3. **使用 AutoDetectTarget**:
   ```go
   target := notifyhub.AutoDetectTarget("ou_user_id")
   target := notifyhub.AutoDetectTarget("oc_group_id")
   ```

## 🧪 调试工具

### 直接发送测试

使用 `debug-sender.go` 进行底层网络请求测试:

```bash
FEISHU_WEBHOOK_URL="your-webhook-url" \
FEISHU_SECRET="your-secret" \
go run debug-sender.go
```

输出示例:
```
🔐 添加签名验证:
  时间戳: 1758532126
  签名: lVXwEMH7af1ted62ghUE7VZTPxK7BkkFewZyD4l0WbA=

📤 发送的消息内容:
{"msg_type":"text","content":{"text":"测试消息"},"timestamp":"1758532126","sign":"..."}

📥 响应状态: 200 OK (耗时 549ms)
📄 响应内容:
{"code":0,"msg":"success"}
```

### NotifyHub 集成测试

使用 `test-real.go` 测试完整的 NotifyHub 集成:

```bash
FEISHU_WEBHOOK_URL="your-webhook-url" \
FEISHU_SECRET="your-secret" \
go run test-real.go
```

## 📋 检查清单

在报告问题前，请确认以下事项:

- [ ] Webhook URL 格式正确: `https://open.feishu.cn/open-apis/bot/v2/hook/xxxxx`
- [ ] 签名密钥正确设置
- [ ] 系统时间准确（与标准时间误差 < 1小时）
- [ ] 机器人已添加到目标群聊
- [ ] 网络连接正常，可访问飞书服务
- [ ] 目标用户/群组 ID 正确
- [ ] 使用了正确的消息格式

## 🔧 高级调试

### 查看详细日志

修改代码以添加调试输出:

```go
import "log"

// 在发送前添加
log.Printf("发送消息到: %+v", message.Targets)
log.Printf("消息内容: %s", message.Body)

receipt, err := hub.Send(ctx, message)

// 在发送后添加
log.Printf("发送结果: %+v", receipt)
if err != nil {
    log.Printf("发送错误: %v", err)
}
```

### 网络抓包分析

使用网络抓包工具查看实际的HTTP请求:

```bash
# Linux/Mac
sudo tcpdump -i any -s 0 -w feishu.pcap host open.feishu.cn

# 分析抓包文件
wireshark feishu.pcap
```

### API 响应码说明

| 响应码 | 说明 | 解决方案 |
|--------|------|----------|
| 0 | 成功 | 消息已发送 |
| 19021 | 签名验证失败 | 检查签名和时间戳 |
| 19001 | 参数错误 | 检查消息格式 |
| 19002 | 机器人不存在 | 检查 Webhook URL |
| 19003 | 机器人未激活 | 激活群聊机器人 |

## 📞 获取帮助

如果问题仍未解决，请提供以下信息:

1. **环境信息**:
   - 操作系统和版本
   - Go 版本
   - NotifyHub 版本

2. **配置信息**:
   - Webhook URL 格式（隐藏敏感部分）
   - 是否使用签名验证
   - 目标 ID 格式

3. **错误日志**:
   - 完整的错误消息
   - debug-sender.go 的输出
   - 网络响应内容

4. **重现步骤**:
   - 详细的操作步骤
   - 预期结果 vs 实际结果

**联系方式**:
- 提交 GitHub Issue: [NotifyHub Issues](https://github.com/kart-io/notifyhub/issues)
- 附上调试输出和错误日志