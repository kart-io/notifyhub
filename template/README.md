# Template Package

## 功能概述

Template包实现了NotifyHub的模板引擎系统，提供强大的消息模板化功能。支持变量替换、内置函数、格式转换和多种模板类型，使消息内容的管理更加灵活和规范。

## 核心功能

### 1. 变量替换
- 占位符语法：`{{variable}}`
- 支持嵌套对象访问：`{{user.name}}`
- 默认值语法：`{{variable | default "默认值"}}`

### 2. 内置函数
- **文本处理**：`upper`, `lower`, `title`, `trim`
- **时间函数**：`now`, `formatTime`, `timeAgo`
- **数值函数**：`add`, `sub`, `mul`, `div`
- **逻辑函数**：`default`, `if`, `eq`, `ne`

### 3. 格式转换
- 自动检测消息格式
- Markdown到文本的转换
- HTML到文本的转换
- 平台特定格式适配

## 核心结构

### Engine结构体
```go
type Engine struct {
    templates map[string]*template.Template
    funcMap   template.FuncMap
}
```

### 内置模板
```go
var builtinTemplates = map[string]string{
    "alert":  "🚨 ALERT: {{.title}}\n\n{{.body}}\n\n...",
    "notice": "📢 NOTICE: {{.title}}\n\n{{.body}}\n\n...",
    "report": "📊 REPORT: {{.title}}\n\n{{.body}}\n\n...",
}
```

## 使用示例

### 基本变量替换

```go
// 创建模板引擎
engine := template.NewEngine()

// 原始消息
message := &notifiers.Message{
    Title: "服务器告警",
    Body:  "服务器 {{server}} 在环境 {{environment}} 中CPU使用率达到 {{cpu_usage}}%",
    Variables: map[string]interface{}{
        "server":      "web-01",
        "environment": "production",
        "cpu_usage":   95.7,
    },
}

// 渲染模板
rendered, err := engine.RenderMessage(message)
// 结果: "服务器 web-01 在环境 production 中CPU使用率达到 95.7%"
```

### 使用内置模板

```go
// 使用alert模板
message := &notifiers.Message{
    Template: "alert",
    Variables: map[string]interface{}{
        "title":       "系统异常",
        "body":        "数据库连接失败",
        "server":      "db-01",
        "environment": "PRODUCTION",
    },
}

rendered, err := engine.RenderMessage(message)
```

渲染结果：
```
🚨 ALERT: 系统异常

数据库连接失败

Server: db-01
Environment: PRODUCTION

Time: 2024-09-14 15:30:25

---
This is an automated alert from NotifyHub
```

### 内置函数使用

```go
message := &notifiers.Message{
    Body: `
用户: {{user.name | upper}}
时间: {{now | formatTime "2006-01-02 15:04:05"}}
状态: {{status | default "未知"}}
优先级: {{if eq priority 5}}紧急{{else}}普通{{end}}
`,
    Variables: map[string]interface{}{
        "user": map[string]string{"name": "张三"},
        "priority": 5,
        "status": "",
    },
}

rendered, err := engine.RenderMessage(message)
```

## 内置函数详解

### 文本处理函数

```go
// 大小写转换
{{name | upper}}     // "ZHANG SAN"
{{name | lower}}     // "zhang san"
{{name | title}}     // "Zhang San"

// 字符串处理
{{text | trim}}      // 去除首尾空白
{{text | truncate 50}} // 截断到50字符
```

### 时间函数

```go
// 当前时间
{{now}}                                    // 2024-09-14T15:30:25Z
{{now | formatTime "2006-01-02 15:04:05"}} // 2024-09-14 15:30:25

// 时间格式化
{{timestamp | formatTime "Jan 2, 2006"}}   // Sep 14, 2024

// 相对时间
{{created_at | timeAgo}}                   // "2小时前"
```

### 数值函数

```go
// 数学运算
{{add value 10}}      // value + 10
{{sub value 5}}       // value - 5
{{mul value 2}}       // value * 2
{{div value 3}}       // value / 3

// 格式化
{{cpu_usage | printf "%.1f%%"}}  // "95.7%"
```

### 逻辑函数

```go
// 默认值
{{name | default "匿名用户"}}

// 条件判断
{{if gt priority 3}}
高优先级消息
{{else}}
普通消息
{{end}}

// 比较函数
{{if eq status "error"}}错误{{end}}
{{if ne status "ok"}}异常{{end}}
{{if gt count 100}}超量{{end}}
```

## 高级特性

### 自定义模板

```go
// 注册自定义模板
customTemplate := `
📋 **{{.title | upper}}**

**描述**: {{.body}}
**项目**: {{.project | default "未指定"}}
**负责人**: {{.owner | default "待分配"}}
**截止时间**: {{.deadline | formatTime "2006-01-02"}}

---
状态: {{if eq .status "urgent"}}🔴 紧急{{else}}🟢 正常{{end}}
`

engine.RegisterTemplate("task", customTemplate)

// 使用自定义模板
message := &notifiers.Message{
    Template: "task",
    Variables: map[string]interface{}{
        "title":    "完成API文档",
        "body":     "需要更新用户认证相关的API文档",
        "project":  "NotifyHub",
        "owner":    "张三",
        "deadline": time.Now().AddDate(0, 0, 7),
        "status":   "urgent",
    },
}
```

### 模板继承

```go
// 基础模板
baseTemplate := `
{{define "base"}}
📢 {{.service_name | upper}}

{{template "content" .}}

---
Time: {{now | formatTime "2006-01-02 15:04:05"}}
Environment: {{.environment | upper}}
{{end}}
`

// 继承模板
alertTemplate := `
{{template "base" .}}
{{define "content"}}
🚨 ALERT: {{.title}}

{{.body}}

Severity: {{.severity | default "medium"}}
{{end}}
`
```

### 条件模板

```go
template := `
{{.title}}

{{.body}}

{{if .attachments}}
📎 附件:
{{range .attachments}}
- {{.}}
{{end}}
{{end}}

{{if gt (len .tags) 0}}
🏷️ 标签: {{join .tags ", "}}
{{end}}
`
```

## 格式转换

### Markdown转换

```go
// Markdown格式消息
message := &notifiers.Message{
    Format: notifiers.FormatMarkdown,
    Body: `
# 系统告警

服务器 **{{server}}** 出现异常：

- CPU使用率：{{cpu_usage}}%
- 内存使用率：{{memory_usage}}%
- 磁盘使用率：{{disk_usage}}%

## 处理建议

1. 检查系统进程
2. 清理临时文件
3. 重启相关服务
`,
    Variables: map[string]interface{}{
        "server": "web-01",
        "cpu_usage": 95,
        "memory_usage": 80,
        "disk_usage": 75,
    },
}

// 引擎会根据目标平台自动转换格式
rendered, err := engine.RenderMessage(message)
```

### HTML转换

```go
// HTML格式消息
message := &notifiers.Message{
    Format: notifiers.FormatHTML,
    Body: `
<h2>{{.title}}</h2>
<p><strong>服务器:</strong> {{.server}}</p>
<p><strong>状态:</strong> <span style="color: red;">{{.status}}</span></p>
<ul>
{{range .issues}}
<li>{{.}}</li>
{{end}}
</ul>
`,
}

// 对于不支持HTML的平台（如邮件），会自动转换为纯文本
```

## 扩展功能

### 自定义函数

```go
// 注册自定义函数
customFuncs := template.FuncMap{
    "maskEmail": func(email string) string {
        parts := strings.Split(email, "@")
        if len(parts) != 2 {
            return email
        }
        name := parts[0]
        if len(name) > 2 {
            name = name[:2] + "***"
        }
        return name + "@" + parts[1]
    },
    "formatBytes": func(bytes int64) string {
        if bytes < 1024 {
            return fmt.Sprintf("%d B", bytes)
        }
        if bytes < 1024*1024 {
            return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
        }
        return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
    },
}

engine.RegisterFunctions(customFuncs)

// 使用自定义函数
template := `
用户: {{.email | maskEmail}}
文件大小: {{.file_size | formatBytes}}
`
```

### 模板缓存

```go
// 启用模板缓存
engine := template.NewEngine()
engine.EnableCache(true)

// 预编译常用模板
templates := map[string]string{
    "daily_report": dailyReportTemplate,
    "alert":        alertTemplate,
    "notice":       noticeTemplate,
}

for name, tmpl := range templates {
    engine.RegisterTemplate(name, tmpl)
}
```

## 错误处理

### 常见错误类型

1. **模板语法错误**: 占位符格式不正确
2. **变量不存在**: 引用了未定义的变量
3. **函数调用错误**: 函数参数类型或数量不匹配
4. **模板不存在**: 引用了未注册的模板

### 错误处理示例

```go
rendered, err := engine.RenderMessage(message)
if err != nil {
    switch e := err.(type) {
    case *template.SyntaxError:
        log.Printf("模板语法错误: %v", e)
    case *template.VariableError:
        log.Printf("变量错误: %v", e)
    case *template.FunctionError:
        log.Printf("函数调用错误: %v", e)
    default:
        log.Printf("渲染错误: %v", e)
    }

    // 使用原始消息作为降级
    rendered = message
}
```

## 性能优化

### 模板预编译

```go
// 启动时预编译所有模板
func (e *Engine) PrecompileTemplates() error {
    for name, tmpl := range builtinTemplates {
        if err := e.RegisterTemplate(name, tmpl); err != nil {
            return fmt.Errorf("预编译模板 %s 失败: %w", name, err)
        }
    }
    return nil
}
```

### 缓存策略

```go
// LRU缓存
type TemplateCache struct {
    cache *lru.Cache
    mutex sync.RWMutex
}

func (c *TemplateCache) Get(key string) (*template.Template, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    return c.cache.Get(key)
}
```

## 最佳实践

### 1. 模板组织

```go
// 按类型组织模板
const (
    AlertTemplate    = "alert"
    NoticeTemplate   = "notice"
    ReportTemplate   = "report"
    CustomTemplate   = "custom"
)
```

### 2. 变量命名

```go
// 使用一致的命名规范
variables := map[string]interface{}{
    "service_name":  "NotifyHub",
    "server_name":   "web-01",
    "user_name":     "张三",
    "created_at":    time.Now(),
    "is_critical":   true,
}
```

### 3. 错误容错

```go
// 提供默认值避免渲染失败
template := `
标题: {{.title | default "无标题"}}
内容: {{.body | default "无内容"}}
时间: {{.created_at | default now | formatTime "2006-01-02 15:04:05"}}
`
```

## HTTP远程模板加载 🌐

模板引擎支持从远程HTTP源加载模板：

### 核心功能

```go
// 创建模板引擎
engine := NewEngine()

// 从HTTP URL加载模板
err := engine.LoadTemplateFromURL(
    "template_name",
    "https://example.com/templates/alert.tmpl",
    notifiers.FormatText,
    map[string]string{
        "Authorization": "Bearer your-token",
        "User-Agent":    "NotifyHub/1.0",
    },
)
```

### 支持的模板源

#### 1. GitHub公共仓库
```go
err := engine.LoadTemplateFromURL(
    "github_public",
    "https://raw.githubusercontent.com/company/templates/main/alert.tmpl",
    notifiers.FormatText,
    nil, // 公共仓库不需要认证
)
```

#### 2. GitHub私有仓库
```go
err := engine.LoadTemplateFromURL(
    "github_private",
    "https://raw.githubusercontent.com/company/private-templates/main/alert.tmpl",
    notifiers.FormatText,
    map[string]string{
        "Authorization": "token ghp_xxxxxxxxxxxx",
        "User-Agent":    "NotifyHub/1.0",
    },
)
```

#### 3. 企业内部API
```go
err := engine.LoadTemplateFromURL(
    "internal_template",
    "https://templates.company.internal/api/v1/templates/alert",
    notifiers.FormatHTML,
    map[string]string{
        "X-API-Key": "internal-api-key-123",
        "Accept":    "text/plain",
    },
)
```

#### 4. CDN模板服务
```go
err := engine.LoadTemplateFromURL(
    "cdn_template",
    "https://cdn.company.com/templates/notification.tmpl",
    notifiers.FormatText,
    map[string]string{
        "Cache-Control": "no-cache",
    },
)
```

### 使用示例

#### 基本远程加载
```go
package main

import (
    "log"
    "github.com/kart-io/notifyhub/template"
    "github.com/kart-io/notifyhub/notifiers"
)

func main() {
    engine := template.NewEngine()

    // 加载GitHub上的模板
    err := engine.LoadTemplateFromURL(
        "github_alert",
        "https://raw.githubusercontent.com/example/templates/main/alert.tmpl",
        notifiers.FormatText,
        map[string]string{
            "Authorization": "token YOUR_GITHUB_TOKEN",
        },
    )
    if err != nil {
        log.Printf("Failed to load template: %v", err)
        return
    }

    // 使用加载的模板
    message := &notifiers.Message{
        Template: "github_alert",
        Variables: map[string]interface{}{
            "title": "System Alert",
            "service": "Database",
            "status": "Critical",
        },
    }

    rendered, err := engine.RenderMessage(message)
    if err != nil {
        log.Printf("Render error: %v", err)
        return
    }

    log.Printf("Rendered: %s", rendered.Body)
}
```

## 高级模板示例 🎨

### Slack Webhook模板
```json
{
    "text": "{{.title}}",
    "blocks": [
        {
            "type": "header",
            "text": {
                "type": "plain_text",
                "text": "{{.title | upper}}"
            }
        },
        {
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": "{{.body}}"
            }
        }
    ]
}
```

### HTML邮件模板
```html
<!DOCTYPE html>
<html>
<head>
    <style>
        .alert { background: #fff3cd; padding: 15px; border-radius: 5px; }
        .info-table { width: 100%; border-collapse: collapse; }
    </style>
</head>
<body>
    <div class="alert">
        <h2>{{.title}}</h2>
        <p>{{.body}}</p>

        {{if .details}}
        <table class="info-table">
            {{range $key, $value := .details}}
            <tr>
                <td>{{$key | title}}</td>
                <td>{{$value}}</td>
            </tr>
            {{end}}
        </table>
        {{end}}
    </div>
</body>
</html>
```

### 指标报告模板
```text
📊 **{{.report_name | upper}}**
Generated: {{.timestamp | formatTime "2006-01-02 15:04:05"}}

## 📈 Metrics
{{range .metrics}}
**{{.name}}**: {{.value}}{{if .unit}} {{.unit}}{{end}}
{{end}}

## ⚠️ Alerts
{{range .alerts}}
- **{{.level | upper}}**: {{.message}}
{{end}}
```

## 集成指南 🔧

### 错误处理策略

```go
func LoadTemplatesSafely(engine *template.Engine, sources map[string]string) {
    for name, url := range sources {
        err := engine.LoadTemplateFromURL(name, url, notifiers.FormatText, nil)
        if err != nil {
            log.Printf("Failed to load template %s from %s: %v", name, url, err)
            // 继续加载其他模板，不中断程序
            continue
        }
        log.Printf("Successfully loaded template: %s", name)
    }
}

// 使用示例
templateSources := map[string]string{
    "alert":      "https://github.com/company/templates/raw/main/alert.tmpl",
    "notice":     "https://github.com/company/templates/raw/main/notice.tmpl",
    "report":     "https://github.com/company/templates/raw/main/report.tmpl",
}

LoadTemplatesSafely(engine, templateSources)
```

### 与Hub集成
```go
// 创建模板引擎并加载远程模板
engine := template.NewEngine()

// 加载远程模板
engine.LoadTemplateFromURL(
    "custom_alert",
    "https://your-company.com/templates/alert.tmpl",
    notifiers.FormatText,
    map[string]string{"Authorization": "Bearer your-token"},
)

// 在Hub中使用（需要扩展Hub支持）
hub.SetTemplateEngine(engine)

// 使用自定义模板发送消息
message := &notifiers.Message{
    Template: "custom_alert",
    Variables: map[string]interface{}{
        "title": "Critical Issue",
        "service": "Payment Gateway",
        "status": "DOWN",
    },
}

results, err := hub.Send(ctx, message, nil)
```

## 安全考虑 🔒

### 远程模板安全
```go
// 配置可信域名
allowedDomains := []string{
    "github.com",
    "gitlab.com",
    "raw.githubusercontent.com",
    "internal.company.com",
}

// URL验证函数
func validateRemoteURL(url string) error {
    u, err := url.Parse(url)
    if err != nil {
        return err
    }

    // 只允许HTTPS协议
    if u.Scheme != "https" {
        return fmt.Errorf("only HTTPS URLs are allowed")
    }

    // 检查域名白名单
    for _, domain := range allowedDomains {
        if u.Host == domain || strings.HasSuffix(u.Host, "."+domain) {
            return nil
        }
    }

    return fmt.Errorf("domain %s not in allowlist", u.Host)
}
```

### 模板验证
- 自动验证模板语法
- 防止无限循环和递归
- 限制模板复杂度
- 超时控制（默认30秒）

## 文件说明

- `engine.go` - 核心模板引擎实现，包含HTTP远程加载功能
- `example_usage.go` - HTTP远程模板加载使用示例