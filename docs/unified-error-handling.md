# 统一错误处理系统 (Unified Error Handling System)

## 概述 (Overview)

本文档描述了 NotifyHub 的统一错误处理系统，该系统解决了不同平台 API 返回错误千差万别的问题，通过标准化错误处理确保接口抽象的一致性。

## 问题背景 (Problem Background)

### 原有问题

- **错误定义分散**: 不同包中定义了重复的错误类型（如多个 `ErrEmptyMessage`）
- **缺乏标准化**: 平台特定的 HTTP/SMTP 错误没有被统一映射
- **错误抽象不足**: 底层错误直接暴露给调用者，破坏了接口抽象
- **错误分类不明确**: 缺少明确的错误分类和错误代码

### 解决方案

实现统一的错误类型系统，无论底层是 HTTP 429 还是 SMTP 速率限制，都统一返回一个可识别的 `ErrRateLimited` 错误，使调用方可以编写统一的错误处理逻辑。

## 核心设计 (Core Design)

### 1. 错误代码系统 (Error Code System)

```go
type ErrorCode string

const (
    // Configuration errors
    CodeInvalidConfig    ErrorCode = "INVALID_CONFIG"
    CodeMissingConfig    ErrorCode = "MISSING_CONFIG"

    // Network and transport errors
    CodeNetworkError     ErrorCode = "NETWORK_ERROR"
    CodeTimeout          ErrorCode = "TIMEOUT"
    CodeRateLimited      ErrorCode = "RATE_LIMITED"
    CodeUnauthorized     ErrorCode = "UNAUTHORIZED"

    // ... 更多错误代码
)
```

### 2. 错误分类系统 (Error Category System)

```go
type ErrorCategory string

const (
    CategoryConfig       ErrorCategory = "CONFIG"
    CategoryValidation   ErrorCategory = "VALIDATION"
    CategoryNetwork      ErrorCategory = "NETWORK"
    CategoryAuth         ErrorCategory = "AUTH"
    CategoryRateLimit    ErrorCategory = "RATE_LIMIT"
    CategoryPlatform     ErrorCategory = "PLATFORM"
    CategoryTransport    ErrorCategory = "TRANSPORT"
    CategoryInternal     ErrorCategory = "INTERNAL"
)
```

### 3. 统一错误结构 (Unified Error Structure)

```go
type NotifyError struct {
    Code     ErrorCode     `json:"code"`
    Category ErrorCategory `json:"category"`
    Message  string        `json:"message"`
    Platform string        `json:"platform,omitempty"`
    Cause    error         `json:"-"`
}
```

## 核心功能 (Core Features)

### 1. 错误映射 (Error Mapping)

#### HTTP 错误映射

```go
func MapHTTPError(statusCode int, body string, platform string) *NotifyError
```

**映射规则:**

- `401` → `CodeUnauthorized` (AUTH category)
- `403` → `CodeForbidden` (AUTH category)
- `429` → `CodeRateLimited` (RATE_LIMIT category)
- `5xx` → `CodeServerError` (NETWORK category)

#### SMTP 错误映射

```go
func MapSMTPError(err error) *NotifyError
```

**映射规则:**

- `535 authentication failed` → `CodeInvalidCredentials`
- `421 rate limit` → `CodeRateLimited`
- `timeout` → `CodeTimeout`
- `550 invalid recipient` → `CodeInvalidTarget`

### 2. 平台特定错误 (Platform-Specific Errors)

```go
// Feishu 错误
func NewFeishuError(code ErrorCode, message string) *NotifyError

// Email 错误
func NewEmailError(code ErrorCode, message string) *NotifyError

// SMS 错误
func NewSMSError(code ErrorCode, message string) *NotifyError
```

### 3. 错误检查工具 (Error Checking Utilities)

```go
// 分类检查
func IsConfigurationError(err error) bool
func IsValidationError(err error) bool
func IsNetworkError(err error) bool
func IsAuthError(err error) bool
func IsRateLimitError(err error) bool

// 重试逻辑检查
func IsRetryableError(err error) bool
func IsTemporaryError(err error) bool
```

### 4. 错误属性 (Error Properties)

```go
// 判断是否可重试
func (e *NotifyError) IsRetryable() bool

// 获取对应的 HTTP 状态码
func (e *NotifyError) HTTPStatusCode() int

// 错误比较
func (e *NotifyError) Is(target error) bool

// 错误包装
func (e *NotifyError) Unwrap() error
```

## 使用示例 (Usage Examples)

### 1. 在 Transport 层使用统一错误

#### Feishu Transport

```go
// 网络错误映射
resp, err := t.client.Do(req)
if err != nil {
    return nil, errors.MapNetworkError(err, "feishu")
}

// HTTP 状态码映射
if resp.StatusCode != http.StatusOK {
    return nil, errors.MapHTTPError(resp.StatusCode, string(body), "feishu")
}

// Feishu API 错误映射
if response.Code != 0 {
    switch response.Code {
    case 19003: // Request too frequent
        return errors.NewFeishuError(errors.CodeRateLimited, response.Msg)
    case 19001: // Invalid app_id
        return errors.NewFeishuError(errors.CodeInvalidCredentials, response.Msg)
    default:
        return errors.NewFeishuError(errors.CodeSendingFailed, response.Msg)
    }
}
```

#### Email Transport

```go
// SMTP 错误映射
if err := t.sendEmail(ctx, target.Value, emailMsg); err != nil {
    smtpErr := errors.MapSMTPError(err)
    result.SetError(smtpErr)
    return result, smtpErr
}

// 超时错误处理
case <-time.After(t.timeout):
    return errors.NewEmailError(errors.CodeTimeout,
        fmt.Sprintf("email send timeout after %v", t.timeout))
```

### 2. 在业务逻辑中使用错误检查

```go
func handleSendingError(err error) (shouldRetry bool, waitTime time.Duration) {
    // 检查是否为速率限制错误
    if errors.IsRateLimitError(err) {
        return true, time.Minute * 5  // 等待 5 分钟后重试
    }

    // 检查是否为网络错误
    if errors.IsNetworkError(err) {
        return true, time.Second * 30  // 等待 30 秒后重试
    }

    // 检查是否为认证错误
    if errors.IsAuthError(err) {
        return false, 0  // 认证错误不重试
    }

    // 使用通用的重试检查
    return errors.IsRetryableError(err), time.Second * 10
}
```

### 3. 错误信息标准化输出

```go
func logError(err error) {
    if notifyErr, ok := err.(*errors.NotifyError); ok {
        log.WithFields(logrus.Fields{
            "error_code":     notifyErr.Code,
            "error_category": notifyErr.Category,
            "platform":       notifyErr.Platform,
            "retryable":      notifyErr.IsRetryable(),
            "http_status":    notifyErr.HTTPStatusCode(),
        }).Error(notifyErr.Message)
    } else {
        log.Error(err)
    }
}
```

## 向后兼容 (Backward Compatibility)

为保证向后兼容，所有现有的错误变量都映射到新的标准错误：

```go
// core/sending/errors.go
var (
    ErrInvalidTargetType = errors.ErrInvalidTarget
    ErrEmptyTargetValue  = errors.ErrEmptyTarget
    ErrSendingFailed     = errors.ErrSendingFailed
    // ... 其他映射
)

// core/message/errors.go
var (
    ErrEmptyMessage      = errors.ErrEmptyMessage
    ErrInvalidPriority   = errors.ErrInvalidPriority
    ErrInvalidFormat     = errors.ErrInvalidFormat
    // ... 其他映射
)
```

## 好处与优势 (Benefits)

### 1. 统一的错误处理

- 所有平台的错误都使用相同的错误代码和分类
- 调用方无需了解底层平台的具体错误格式
- 简化了错误处理逻辑的编写

### 2. 更好的可观测性

- 标准化的错误代码便于监控和告警
- 错误分类帮助快速定位问题领域
- 统一的 HTTP 状态码映射便于 API 响应

### 3. 智能重试机制

- 基于错误类型的智能重试决策
- 避免对不可重试错误的无效重试
- 提高系统的可靠性和用户体验

### 4. 平台抽象

- 隐藏底层平台 API 的差异性
- 保持接口抽象的一致性
- 便于添加新平台而不影响现有逻辑

## 测试覆盖 (Test Coverage)

完整的测试套件确保错误处理系统的正确性：

- **错误映射测试**: 验证 HTTP 和 SMTP 错误的正确映射
- **错误属性测试**: 验证 `IsRetryable()`、`HTTPStatusCode()` 等方法
- **错误比较测试**: 验证 `Is()` 和 `Unwrap()` 方法
- **分类检查测试**: 验证各种错误分类检查函数
- **向后兼容测试**: 验证旧错误变量的正确映射

## 最佳实践 (Best Practices)

### 1. Transport 层实现

- 使用 `MapHTTPError()` 处理 HTTP 响应错误
- 使用 `MapSMTPError()` 处理 SMTP 错误
- 使用 `MapNetworkError()` 处理网络连接错误
- 为平台特定错误使用对应的 `New*Error()` 函数

### 2. 业务逻辑层处理

- 使用错误分类检查函数进行错误类型判断
- 使用 `IsRetryableError()` 决定是否重试
- 记录错误时包含错误代码和分类信息

### 3. API 层响应

- 使用 `HTTPStatusCode()` 方法设置 HTTP 响应状态码
- 在响应体中包含标准化的错误代码和消息
- 保持错误响应格式的一致性

通过这个统一错误处理系统，NotifyHub 实现了跨平台的一致错误处理体验，提高了系统的可维护性和用户体验。
