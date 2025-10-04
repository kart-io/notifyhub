# NotifyHub 示例代码

本目录包含了NotifyHub各个平台的完整使用示例，按平台分类组织。

## 📁 目录结构

```
examples/
├── README.md                 # 本文档
├── common/                   # 公共配置和工具
│   ├── config.go            # 通用配置结构
│   └── utils.go             # 工具函数
├── email/                    # 邮件平台示例
│   ├── basic/               # 基础功能
│   │   └── main.go
│   └── advanced/            # 高级功能
│       └── main.go
├── feishu/                   # 飞书平台示例
│   ├── basic/               # 基础功能
│   │   └── main.go
│   └── advanced/            # 高级功能
│       └── main.go
└── webhook/                  # Webhook平台示例
    ├── basic/               # 基础功能
    │   └── main.go
    └── advanced/            # 高级功能
        └── main.go
```

## 🚀 快速开始

### 1. 运行邮件示例

```bash
# 基础邮件发送
cd examples/email/basic
go run main.go

# 高级邮件功能
cd examples/email/advanced
go run main.go
```

### 2. 运行飞书示例

```bash
# 基础飞书消息
cd examples/feishu/basic
go run main.go

# 高级飞书功能
cd examples/feishu/advanced
go run main.go
```

### 3. 运行Webhook示例

```bash
# 基础Webhook发送
cd examples/webhook/basic
go run main.go

# 高级Webhook功能
cd examples/webhook/advanced
go run main.go
```

## ⚙️ 配置说明

每个示例都包含详细的配置说明。在运行前，请修改代码中的配置信息：

### 邮件配置
- SMTP服务器地址和端口
- 邮箱用户名和密码
- 发件人和收件人地址

### 飞书配置
- Webhook URL
- 签名密钥（可选）
- 关键词设置（可选）

### Webhook配置
- 目标URL
- 认证信息
- 自定义头部

## 📋 功能对比

| 功能 | 邮件 | 飞书 | Webhook |
|------|------|------|---------|
| 文本消息 | ✅ | ✅ | ✅ |
| HTML格式 | ✅ | ❌ | ✅ |
| Markdown | ✅ | ✅ | ✅ |
| 卡片消息 | ❌ | ✅ | ✅ |
| 附件支持 | ✅ | ❌ | ✅ |
| 批量发送 | ✅ | ✅ | ✅ |
| 异步发送 | ✅ | ✅ | ✅ |
| 优先级 | ✅ | ✅ | ✅ |

## 🐛 故障排除

### 邮件发送问题
- 检查SMTP设置是否正确
- 确认防火墙未阻止SMTP端口
- 验证邮箱密码或应用专用密码

### 飞书发送问题
- 确认Webhook URL有效
- 检查网络连接
- 验证消息格式

### Webhook发送问题
- 确认目标服务器可访问
- 检查认证信息
- 验证请求格式

## 📚 更多资源

- [NotifyHub 文档](../README.md)
- [邮件平台配置](./email/README.md)
- [飞书平台配置](./feishu/README.md)
- [Webhook平台配置](./webhook/README.md)

---

如有问题，请查看各平台的详细README或提交Issue。