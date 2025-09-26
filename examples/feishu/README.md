# 飞书推送示例（模板集成）

Feishu Push Example (Template Integration)

这是一个演示如何在 NotifyHub 中使用模板系统进行飞书消息推送的示例应用。

This is an example application demonstrating how to use the template system in NotifyHub for Feishu message pushing.

## 功能特性 / Features

- ✅ **多模板引擎支持**: Go template、Mustache、Handlebars
- ✅ **动态变量替换**: 支持复杂的数据结构和变量替换
- ✅ **条件渲染**: 基于数据内容的条件显示逻辑
- ✅ **循环遍历**: 数组和对象的循环渲染
- ✅ **模板缓存**: 提升渲染性能的智能缓存
- ✅ **语法验证**: 模板语法验证和错误提示
- ✅ **多种消息类型**: 告警、状态报告、部署通知、用户活动

- ✅ **Multiple Template Engines**: Go template, Mustache, Handlebars
- ✅ **Dynamic Variable Substitution**: Support for complex data structures
- ✅ **Conditional Rendering**: Conditional display logic based on data
- ✅ **Loops and Iteration**: Array and object iteration rendering
- ✅ **Template Caching**: Smart caching for improved rendering performance
- ✅ **Syntax Validation**: Template syntax validation and error reporting
- ✅ **Multiple Message Types**: Alerts, status reports, deployment notifications, user activities

## 快速开始 / Quick Start

### 1. 环境准备 / Environment Setup

```bash
# 克隆项目 / Clone project
git clone https://github.com/kart-io/notifyhub.git
cd notifyhub/examples/feishu

# 运行设置脚本 / Run setup script
./setup.sh

# 或手动构建 / Or build manually
go build -o feishu-example .
```

### 2. 配置飞书机器人 / Configure Feishu Robot

1. 在飞书群中点击 `群设置` → `群机器人`
2. 点击 `添加机器人` → `自定义机器人`
3. 设置机器人名称和描述
4. 选择安全设置（IP白名单、签名验证、关键词验证）
5. 复制生成的 Webhook URL

1. In Feishu group, click `Group Settings` → `Group Bots`
2. Click `Add Bot` → `Custom Bot`
3. Set bot name and description
4. Choose security settings (IP whitelist, signature verification, keyword verification)
5. Copy the generated Webhook URL

### 3. 环境变量配置 / Environment Variables

```bash
# 必需：飞书 Webhook URL / Required: Feishu Webhook URL
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your_hook_id"

# 可选：签名密钥（签名验证模式） / Optional: Signature secret (signature verification mode)
export FEISHU_SECRET="your_secret_key"

# 可选：关键词验证 / Optional: Keyword verification
export FEISHU_KEYWORDS="通知"
```

### 4. 运行示例 / Run Example

```bash
# 使用环境变量 / Using environment variables
./feishu-example

# 或直接运行 / Or run directly
go run main.go
```

## 模板系统 / Template System

### 模板引擎 / Template Engines

#### 1. Go Template 引擎

```go
// 条件判断 / Conditional
{{if eq .severity "critical"}}
⚠️ **紧急处理**: 请立即检查系统状态！
{{else if eq .severity "warning"}}
⚠️ **注意**: 请关注系统状态
{{else}}
ℹ️ **信息**: 系统状态正常
{{end}}

// 循环遍历 / Loop iteration
{{range .affected_services}}
- {{.}}
{{end}}

// 变量操作 / Variable manipulation
{{.severity | upper}}
```

#### 2. Mustache 模板引擎

```mustache
{{!-- 条件渲染 / Conditional rendering --}}
{{#security_alert}}
🚨 **安全提醒**
- 风险等级: {{risk_level}}
- 描述: {{description}}
{{/security_alert}}

{{!-- 反向条件 / Inverted condition --}}
{{^sensitive}}
普通数据访问
{{/sensitive}}

{{!-- 循环 / Loop --}}
{{#services}}
- {{name}}: {{status}}
{{/services}}
```

### 模板文件结构 / Template File Structure

```
templates/
├── alert.tmpl              # 告警消息模板（Go template）
├── system_status.tmpl      # 系统状态报告模板（Go template）
├── deployment.tmpl         # 部署通知模板（Go template）
└── user_activity.mustache  # 用户活动通知模板（Mustache）
```

### 模板变量 / Template Variables

所有模板变量定义在 `template_vars.json` 文件中，包含：

All template variables are defined in `template_vars.json`, including:

- **required_variables**: 必需变量 / Required variables
- **optional_variables**: 可选变量 / Optional variables
- **variable_types**: 变量类型定义 / Variable type definitions
- **example_data**: 示例数据 / Example data

## 消息类型示例 / Message Type Examples

### 1. 告警消息 / Alert Message

**模板**: `templates/alert.tmpl`
**引擎**: Go template

```yaml
variables:
  severity: "critical"
  service_name: "API Gateway"
  alert_type: "High CPU Usage"
  timestamp: "2024-09-26 15:30:00"
  description: "CPU 使用率持续超过 90%"
  affected_services:
    - "登录服务"
    - "订单服务"
  metrics:
    cpu_usage: "94"
    memory_usage: "78"
```

### 2. 系统状态报告 / System Status Report

**模板**: `templates/system_status.tmpl`
**引擎**: Go template

```yaml
variables:
  report_date: "2024-09-26"
  services:
    - name: "Web 前端"
      status: "healthy"
      response_time: 120
      error_rate: 0.1
  server:
    name: "prod-server-01"
    cpu_usage: 65
    memory_usage: 72
```

### 3. 部署通知 / Deployment Notification

**模板**: `templates/deployment.tmpl`
**引擎**: Go template

```yaml
variables:
  project_name: "NotifyHub"
  environment: "production"
  version: "v3.1.0"
  status: "success"
  changes:
    - type: "新功能"
      description: "添加飞书模板支持"
```

### 4. 用户活动通知 / User Activity Notification

**模板**: `templates/user_activity.mustache`
**引擎**: Mustache

```yaml
variables:
  user:
    name: "王五"
    email: "wang.wu@example.com"
  activity:
    type: "登录"
    login:
      ip_address: "192.168.1.100"
      device:
        type: "Desktop"
        name: "Windows PC"
```

## 配置选项 / Configuration Options

### 使用环境变量 / Using Environment Variables

```bash
export FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/your_hook_id"
export FEISHU_SECRET="your_secret"
export FEISHU_KEYWORDS="通知"
```

### 使用配置文件 / Using Configuration File

编辑 `config.yaml` 文件：

Edit the `config.yaml` file:

```yaml
platforms:
  feishu:
    webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/your_hook_id"
    secret: "your_secret"
    keywords:
      - "通知"
      - "告警"

templates:
  engines:
    go:
      enabled: true
      supports_functions: true
    mustache:
      enabled: true
      supports_partials: true

  cache:
    enabled: true
    ttl: "5m"
    max_size: 1000
```

## 自定义模板 / Custom Templates

### 创建新模板 / Creating New Templates

1. **在 templates/ 目录创建模板文件**

```bash
touch templates/my_template.tmpl
```

2. **编写模板内容**

```go
{{/* 自定义模板 */}}
# {{.title}}

**时间**: {{.timestamp}}
**内容**: {{.content}}

{{range .items}}
- {{.name}}: {{.value}}
{{end}}
```

3. **在代码中注册模板**

```go
templateContent, err := os.ReadFile("templates/my_template.tmpl")
if err != nil {
    return err
}

err = templateManager.RegisterTemplate("my_template", string(templateContent), template.EngineGo)
if err != nil {
    return err
}
```

4. **使用模板渲染**

```go
variables := map[string]interface{}{
    "title": "我的标题",
    "timestamp": time.Now().Format("2006-01-02 15:04:05"),
    "content": "这是内容",
    "items": []map[string]interface{}{
        {"name": "项目1", "value": "值1"},
        {"name": "项目2", "value": "值2"},
    },
}

content, err := templateManager.RenderTemplate(ctx, "my_template", variables)
```

### 模板最佳实践 / Template Best Practices

1. **变量命名**: 使用清晰、语义化的变量名
2. **错误处理**: 为可选变量提供默认值
3. **性能优化**: 启用模板缓存
4. **安全考虑**: 避免在模板中包含敏感信息
5. **文档完整**: 在 `template_vars.json` 中记录所有变量

## 高级功能 / Advanced Features

### 1. 模板缓存 / Template Caching

```go
// 启用缓存
options := []template.ManagerOption{
    template.WithCacheEnabled(true),
    template.WithCacheTTL(5 * time.Minute),
}

templateManager, err := template.NewManager(logger, options...)
```

### 2. 热重载 / Hot Reload

```go
// 启用热重载（开发环境）
options := []template.ManagerOption{
    template.WithHotReload(true),
    template.WithWatchDirs([]string{"./templates"}),
}
```

### 3. 自定义函数 / Custom Functions

```go
// Go template 支持自定义函数
functions := template.FuncMap{
    "formatTime": func(t time.Time) string {
        return t.Format("2006-01-02 15:04:05")
    },
    "status2emoji": func(status string) string {
        switch status {
        case "healthy":
            return "✅"
        case "warning":
            return "⚠️"
        case "error":
            return "❌"
        default:
            return "ℹ️"
        }
    },
}
```

## 错误处理 / Error Handling

### 常见错误和解决方案 / Common Errors and Solutions

1. **模板语法错误**

```
Error: template: alert:5:10: executing "alert" at <.unknown_var>: can't evaluate field unknown_var
```

解决方案：检查变量名拼写，确保在变量映射中提供了所有必需变量。

2. **模板文件不存在**

```
Error: 读取告警模板失败: open templates/alert.tmpl: no such file or directory
```

解决方案：确保模板文件存在于正确的路径。

3. **变量类型错误**

```
Error: template: range can't iterate over string
```

解决方案：确保用于 range 的变量是数组或切片类型。

## 性能优化 / Performance Optimization

### 1. 模板预编译 / Template Precompilation

```go
// 预编译模板以提高性能
err := templateManager.ValidateTemplate("alert")
if err != nil {
    log.Printf("模板验证失败: %v", err)
}
```

### 2. 缓存策略 / Caching Strategy

```go
// 配置缓存参数
options := []template.ManagerOption{
    template.WithCacheEnabled(true),
    template.WithCacheTTL(10 * time.Minute),    // 缓存时间
    template.WithMaxCacheSize(1000),            // 最大缓存项目数
}
```

### 3. 渲染性能监控 / Rendering Performance Monitoring

```go
start := time.Now()
content, err := templateManager.RenderTemplate(ctx, "alert", variables)
duration := time.Since(start)

log.Printf("模板渲染耗时: %v", duration)
```

## 测试和调试 / Testing and Debugging

### 运行完整测试 / Run Full Test

```bash
# 运行设置脚本验证环境
./setup.sh

# 运行示例应用
./feishu-example

# 检查输出日志
tail -f logs/feishu-example.log
```

### 调试模式 / Debug Mode

```bash
# 启用调试日志
export LOG_LEVEL=debug
go run main.go
```

### 模板语法验证 / Template Syntax Validation

```bash
# 使用设置脚本验证模板语法
./setup.sh | grep "模板语法检查"
```

## 参考资料 / References

- [飞书群机器人配置说明](https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN)
- [飞书机器人消息格式](https://open.feishu.cn/document/ukTMukTMukTM/uAjNwUjLwYDM14CM2ATN)
- [Go Template 语法](https://pkg.go.dev/text/template)
- [Mustache 模板语法](https://mustache.github.io/mustache.5.html)
- [NotifyHub 模板系统文档](../../pkg/notifyhub/template/README.md)

## 许可证 / License

本项目基于 MIT 许可证开源。

This project is open source under the MIT License.