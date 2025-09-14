# NotifyHub 模板引擎模块技术设计 (v1.1)

> **对应需求**: `FR5` (消息模板), `FR6` (动态模板源)

该模块负责将模板和动态数据渲染成最终的消息内容。其核心设计思想是将**模板的加载（Loading）**与**模板的渲染（Rendering）**彻底分离，以实现最大程度的灵活性和可扩展性。

## 1. 设计思想

系统包含两个核心接口：`TemplateProvider` 和 `TemplateEngine`。

*   `TemplateProvider`: 负责**加载**。它的任务是根据一个模板名称，从任意来源获取模板原始内容。
*   `TemplateEngine`: 负责**渲染**。它的任务是接收模板字符串和数据，生成最终文本。

## 2. 模板提供者 (`TemplateProvider`)

### 2.1. 接口定义

```go
package templates

// TemplateProvider defines the interface for retrieving template content.
type TemplateProvider interface {
    Get(ctx context.Context, templateName string) (string, error)
}
```

### 2.2. 内置实现

*   **`FileSystemProvider`**: 从本地文件系统加载模板。构造时接收一个根目录路径，`Get` 方法会根据 `templateName` 在该目录下查找对应的文件（如 `user/welcome.html`）。
*   **`MemoryProvider`**: 从内存 `map[string]string` 加载模板。构造时接收一个 map，`Get` 方法直接从中查找。

### 2.3. 扩展实现：远程提供者

用户可以轻松实现此接口，以便从数据库或远程 API 加载模板。

```go
// 示例：从远程 HTTP 服务加载模板
type RemoteTemplateProvider struct {
    BaseURL    string
    HTTPClient *http.Client
}

func (p *RemoteTemplateProvider) Get(ctx context.Context, templateName string) (string, error) {
    url := fmt.Sprintf("%s/%s", p.BaseURL, templateName)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    // ... (发起 HTTP 请求并返回 body 的逻辑) ...
}
```

## 3. 模板引擎 (`TemplateEngine`)

### 3.1. 核心职责

`TemplateEngine` 的核心职责是执行渲染。默认实现将基于 Go 标准库的 `text/template` 和 `html/template` 进行封装。

### 3.2. 自定义函数

> **对应需求**: `FR5` (丰富的渲染能力)

引擎必须提供机制，允许开发者将自定义的 Go 函数注册到模板引擎中，以在模板内实现复杂的数据格式化。

*   **实现思路**: 提供一个 `WithTemplateFuncs(funcs map[string]interface{})` 的 `Hub` 配置选项，用于在初始化时注册函数。

*   **使用示例**:
    ```go
    // 1. 定义函数
    func formatPrice(price float64) string {
        return fmt.Sprintf("¥%.2f", price)
    }

    // 2. 初始化时注册
    hub := notifyhub.New(
        notifyhub.WithTemplateFuncs(map[string]interface{}{
            "formatPrice": formatPrice,
        }),
    )

    // 3. 在模板中使用
    // {{ .OrderAmount | formatPrice }}
    ```

## 4. 整体工作流程

1.  用户调用 `hub.Send` 并提供 `Message`，其中包含 `TemplateName` 和 `TemplateData`。
2.  `Hub` 调用已注册的 `TemplateProvider` 的 `Get(TemplateName)` 方法，获取模板字符串。
3.  `Hub` 调用 `TemplateEngine` 的 `Render(templateString, TemplateData)` 方法，生成最终内容。
4.  `Hub` 将渲染后的内容交给 `Notifier` 进行发送。
