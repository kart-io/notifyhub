# 多邮件服务商测试工具

这个工具用于测试 NotifyHub 支持的所有邮件服务商配置和连接能力。

## 🎯 功能特性

- ✅ **全面的服务商支持**: 支持14个主流邮件服务商
- 🔍 **自动检测**: 根据邮箱地址自动识别服务商
- 🛠️ **配置验证**: 验证每个服务商的配置参数
- 📧 **集成测试**: 测试 NotifyHub 客户端创建和配置
- 🔒 **安全指南**: 提供安全配置建议

## 📋 支持的邮件服务商

### 国际服务商
| 服务商 | SMTP服务器 | 端口 | 加密方式 | 特殊要求 |
|--------|------------|------|----------|----------|
| Gmail | smtp.gmail.com | 587 | STARTTLS | 需要应用专用密码 |
| Outlook/Hotmail | smtp-mail.outlook.com | 587 | STARTTLS | 支持登录密码 |
| Yahoo Mail | smtp.mail.yahoo.com | 587 | STARTTLS | 需要应用专用密码 |
| Yahoo Japan | smtp.mail.yahoo.co.jp | 587 | STARTTLS | 需要应用专用密码 |
| Zoho Mail | smtp.zoho.com | 587 | STARTTLS | 企业邮箱服务 |
| ProtonMail | 127.0.0.1 | 1025 | STARTTLS | 需要Bridge软件 |

### 国内服务商
| 服务商 | SMTP服务器 | 端口 | 加密方式 | 特殊要求 |
|--------|------------|------|----------|----------|
| 163邮箱 | smtp.163.com | 25 | STARTTLS | 需要授权码 |
| 126邮箱 | smtp.126.com | 25 | STARTTLS | 需要授权码 |
| Yeah邮箱 | smtp.yeah.net | 25 | STARTTLS | 需要授权码 |
| QQ邮箱 | smtp.qq.com | 587 | STARTTLS | 需要授权码 |
| 新浪邮箱 | smtp.sina.com | 25 | STARTTLS | 支持登录密码 |
| 搜狐邮箱 | smtp.sohu.com | 25 | STARTTLS | 支持登录密码 |

### 企业邮箱
| 服务商 | SMTP服务器 | 端口 | 加密方式 | 特殊要求 |
|--------|------------|------|----------|----------|
| 腾讯企业邮箱 | smtp.exmail.qq.com | 587 | STARTTLS | 企业账号密码 |
| 阿里云邮箱 | smtp.mxhichina.com | 587 | STARTTLS | 企业账号密码 |

## 🚀 使用方法

### 基础测试
```bash
cd examples/email/multi-provider-test
go run main.go
```

### 功能说明

#### 1. 配置展示
显示所有支持的邮件服务商的详细配置信息：
- SMTP服务器地址和端口
- 加密方式和认证方法
- 配置说明和特殊要求
- 预定义配置函数测试

#### 2. 服务商检测
根据邮箱地址自动识别对应的邮件服务商：
```
user@gmail.com -> Gmail
user@163.com -> 163邮箱
user@qq.com -> QQ邮箱
```

#### 3. 配置验证
验证每个服务商的预定义配置：
- 参数完整性检查
- 配置格式验证
- NotifyHub客户端创建测试

#### 4. 连接测试指南
提供真实邮件发送测试的代码示例和安全建议。

## 🔧 真实邮件测试

要进行真实的邮件发送测试，请按以下步骤操作：

### 1. 创建测试配置文件
```bash
cp main.go test_real_sending.go
```

### 2. 修改配置
在 `test_real_sending.go` 中：
```go
// 取消注释真实发送测试函数
// 修改为真实的邮箱配置
testRealSending(logger, "Gmail", "your_real_gmail@gmail.com", "your_app_password", "recipient@example.com")
```

### 3. 设置环境变量（推荐）
```bash
export EMAIL_USERNAME="your_email@gmail.com"
export EMAIL_PASSWORD="your_app_password"
export EMAIL_RECIPIENT="recipient@example.com"
```

### 4. 运行测试
```bash
go run test_real_sending.go
```

## ⚠️ 安全提醒

### Gmail 配置
1. 开启两步验证
2. 生成应用专用密码
3. 使用应用专用密码，不是账户密码

### 163/126/QQ 邮箱配置
1. 在邮箱设置中开启 SMTP 服务
2. 生成授权码
3. 使用授权码，不是登录密码

### 企业邮箱配置
1. 联系管理员确认 SMTP 权限
2. 使用企业邮箱账号和密码
3. 确认防火墙和网络策略

### 通用安全建议
- 🔒 不要在代码中硬编码密码
- 🌍 使用环境变量存储敏感信息
- 🔑 定期更换邮箱授权码
- 📱 监控邮箱登录活动

## 🐛 故障排除

### 常见错误

#### 1. `535 Error: authentication failed`
**原因**: 认证失败
**解决方案**:
- 检查用户名和密码是否正确
- 确认使用的是授权码而不是登录密码
- 验证邮箱是否开启了 SMTP 服务

#### 2. `Connection refused`
**原因**: 连接被拒绝
**解决方案**:
- 检查网络连接
- 确认 SMTP 服务器地址和端口
- 检查防火墙设置

#### 3. `TLS handshake failed`
**原因**: TLS 握手失败
**解决方案**:
- 确认使用正确的加密方式（STARTTLS vs 直接TLS）
- 检查服务器证书
- 尝试不同的端口配置

### 调试技巧

1. **启用详细日志**:
```go
logger := common.NewLogger(true) // 启用调试模式
```

2. **检查配置**:
```go
logger.Debug("SMTP配置: Host=%s, Port=%d, TLS=%v", cfg.Host, cfg.Port, cfg.UseTLS)
```

3. **逐步测试**:
- 先测试配置验证
- 再测试客户端创建
- 最后测试邮件发送

## 📞 获取支持

如果遇到问题：
1. 查看详细的错误日志
2. 确认邮箱服务商的最新设置要求
3. 参考各邮箱服务商的官方SMTP文档
4. 检查 NotifyHub 的配置示例

---

**注意**: 各邮箱服务商的SMTP配置可能会随政策变化而调整，建议定期查看官方文档获取最新信息。