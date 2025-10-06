# 163邮箱SMTP配置指南

NotifyHub现在完全支持163邮箱发送邮件。本文档将指导您如何正确配置163邮箱的SMTP设置。

## 🔧 163邮箱SMTP配置

### 基本信息
- **SMTP服务器**: `smtp.163.com`
- **端口**: `25` (推荐) 或 `587`
- **加密方式**: STARTTLS
- **认证方式**: PLAIN

### 配置步骤

#### 1. 开启163邮箱SMTP服务

1. 登录163邮箱 (mail.163.com)
2. 点击右上角"设置" → "POP3/SMTP/IMAP"
3. 开启"SMTP服务"
4. 设置授权码（重要：不是登录密码！）

#### 2. NotifyHub代码配置

```go
// 163邮箱配置示例
config.Email.Host = "smtp.163.com"               // 163 SMTP服务器
config.Email.Port = 25                           // 推荐端口
config.Email.Username = "your_email@163.com"     // 您的163邮箱
config.Email.Password = "your_auth_code"         // 163邮箱授权码（不是登录密码）
config.Email.From = "your_email@163.com"        // 发件人
config.Email.To = "recipient@example.com"       // 收件人
```

#### 3. 使用预定义配置

```go
import "github.com/kart-io/notifyhub/pkg/platforms/email"

// 使用163邮箱预定义配置
emailConfig := email.NetEase163Config("your_email@163.com", "your_auth_code")

// 创建NotifyHub客户端
cfg := &config.Config{
    Email: &config.EmailConfig{
        Host:     emailConfig.SMTPHost,
        Port:     emailConfig.SMTPPort,
        Username: emailConfig.Username,
        Password: emailConfig.Password,
        From:     "your_email@163.com",
        UseTLS:   emailConfig.UseTLS,
    },
}

client, err := notifyhub.NewClient(cfg)
```

## 🚀 完整示例

```go
package main

import (
    "context"

    "github.com/kart-io/notifyhub/examples/common"
    "github.com/kart-io/notifyhub/pkg/notifyhub"
    "github.com/kart-io/notifyhub/pkg/target"
)

func main() {
    // 创建配置
    config := common.DefaultExampleConfig()

    // 163邮箱配置
    config.Email.Host = "smtp.163.com"
    config.Email.Port = 25
    config.Email.Username = "your_email@163.com"
    config.Email.Password = "your_auth_code"        // 授权码
    config.Email.From = "your_email@163.com"
    config.Email.To = "recipient@example.com"

    // 创建NotifyHub客户端
    cfg := config.CreateEmailConfig()
    client, err := notifyhub.NewClient(cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 创建邮件消息
    msg := common.CreateTestMessage("163 Email", "basic")
    msg.Title = "163邮箱测试邮件"
    msg.Body = "这是通过163邮箱SMTP发送的测试邮件。"
    msg.Targets = []target.Target{
        common.CreateEmailTarget(config.Email.To),
    }

    // 发送邮件
    ctx := context.Background()
    receipt, err := client.Send(ctx, msg)
    if err != nil {
        panic(err)
    }

    fmt.Printf("邮件发送成功: %+v\n", receipt)
}
```

## ⚠️ 重要注意事项

### 1. 授权码 vs 登录密码
- **必须使用授权码**，不是163邮箱的登录密码
- 授权码在163邮箱设置中生成，通常是16位字符
- 每个应用可以有不同的授权码

### 2. SMTP服务开启
- 必须在163邮箱设置中手动开启SMTP服务
- 开启过程可能需要手机验证

### 3. 端口选择
- **端口25**: 适用于STARTTLS，推荐使用
- **端口587**: 也支持STARTTLS
- **不推荐使用端口465** (SSL直连)

### 4. 安全设置
- 163邮箱会检测异常登录，建议配置常用IP
- 授权码泄露风险较低，但仍需妥善保管

## 🔍 故障排除

### 常见错误及解决方案

#### 1. `535 Error: authentication failed`
**原因**: 认证失败
**解决**:
- 确认使用的是授权码，不是登录密码
- 检查163邮箱是否已开启SMTP服务
- 验证用户名格式（需要包含@163.com）

#### 2. `Connection refused`
**原因**: 连接被拒绝
**解决**:
- 检查网络连接
- 确认端口号正确（25或587）
- 查看是否有防火墙阻挡

#### 3. `TLS handshake failed`
**原因**: TLS握手失败
**解决**:
- 使用STARTTLS而不是直接TLS
- 检查服务器地址是否正确

## 📧 支持的网易邮箱

NotifyHub支持所有网易邮箱服务：

| 邮箱类型 | SMTP服务器 | 端口 | 配置函数 |
|----------|------------|------|----------|
| 163邮箱 | smtp.163.com | 25 | `NetEase163Config()` |
| 126邮箱 | smtp.126.com | 25 | `NetEase126Config()` |
| Yeah邮箱 | smtp.yeah.net | 25 | `NetEaseYeahConfig()` |

## 📞 获取帮助

如果遇到问题，请：
1. 检查163邮箱SMTP设置是否正确开启
2. 确认授权码是否有效
3. 查看NotifyHub的详细日志输出
4. 参考provider-test示例进行配置验证

---

**注意**: 163邮箱的SMTP配置会因网易政策调整而变化，建议查看最新的163邮箱帮助文档确认配置信息。