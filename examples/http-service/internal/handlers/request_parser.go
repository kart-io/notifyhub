package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/notifiers"
)

// RequestParser 统一解析HTTP请求
type RequestParser struct {
	defaultTimeout time.Duration
}

// NewRequestParser 创建请求解析器
func NewRequestParser() *RequestParser {
	return &RequestParser{
		defaultTimeout: 30 * time.Second,
	}
}

// ParseNotificationRequest 解析通知请求
func (rp *RequestParser) ParseNotificationRequest(r *http.Request) (*notifiers.Message, *client.Options, error) {
	// 解析HTTP请求体
	httpReq, err := client.ParseHTTPRequest(r)
	if err != nil {
		return nil, nil, &ParseError{Type: "request_body", Err: err}
	}

	// 解析HTTP选项
	httpOptions, err := client.ParseHTTPOptions(r)
	if err != nil {
		return nil, nil, &ParseError{Type: "request_options", Err: err}
	}

	// 转换为NotifyHub消息
	message, err := client.ConvertHTTPToMessage(httpReq)
	if err != nil {
		return nil, nil, &ParseError{Type: "message_conversion", Err: err}
	}

	// 转换为NotifyHub选项
	options, err := client.ConvertHTTPToOptions(httpOptions)
	if err != nil {
		return nil, nil, &ParseError{Type: "options_conversion", Err: err}
	}

	return message, options, nil
}

// ParseQuickNotification 解析快速通知（来自查询参数）
func (rp *RequestParser) ParseQuickNotification(r *http.Request) (*notifiers.Message, error) {
	title := r.URL.Query().Get("title")
	body := r.URL.Query().Get("body")
	target := r.URL.Query().Get("target")
	priority := r.URL.Query().Get("priority")

	if title == "" || body == "" || target == "" {
		return nil, &ParseError{
			Type: "missing_parameters",
			Err:  fmt.Errorf("title, body, and target parameters are required"),
		}
	}

	// 使用Builder API构建消息
	builder := client.NewMessage().
		Title(title).
		Body(body)

	// 智能目标检测
	if err := rp.addTarget(builder, target); err != nil {
		return nil, err
	}

	// 设置优先级
	if err := rp.setPriority(builder, priority); err != nil {
		return nil, err
	}

	return builder.BuildAndValidate()
}

// ParseBulkNotificationRequest 解析批量通知请求
func (rp *RequestParser) ParseBulkNotificationRequest(r *http.Request) ([]*notifiers.Message, *client.Options, error) {
	// 解析HTTP选项
	httpOptions, err := client.ParseHTTPOptions(r)
	if err != nil {
		return nil, nil, &ParseError{Type: "request_options", Err: err}
	}

	// 解析批量请求体
	var bulkReq struct {
		Notifications []client.HTTPMessageRequest `json:"notifications"`
	}

	if err := json.NewDecoder(r.Body).Decode(&bulkReq); err != nil {
		return nil, nil, &ParseError{Type: "request_body", Err: err}
	}

	if len(bulkReq.Notifications) == 0 {
		return nil, nil, &ParseError{
			Type: "empty_bulk",
			Err:  fmt.Errorf("no notifications provided in bulk request"),
		}
	}

	if len(bulkReq.Notifications) > 100 {
		return nil, nil, &ParseError{
			Type: "bulk_too_large",
			Err:  fmt.Errorf("too many notifications (max 100, got %d)", len(bulkReq.Notifications)),
		}
	}

	// 转换每个消息
	messages := make([]*notifiers.Message, 0, len(bulkReq.Notifications))
	for i, httpReq := range bulkReq.Notifications {
		message, err := client.ConvertHTTPToMessage(&httpReq)
		if err != nil {
			return nil, nil, &ParseError{
				Type: "message_conversion",
				Err:  fmt.Errorf("notification[%d]: %v", i, err),
			}
		}
		messages = append(messages, message)
	}

	// 转换选项
	options, err := client.ConvertHTTPToOptions(httpOptions)
	if err != nil {
		return nil, nil, &ParseError{Type: "options_conversion", Err: err}
	}

	return messages, options, nil
}

// CreateTimeoutContext 创建带超时的context
func (rp *RequestParser) CreateTimeoutContext(r *http.Request, options *client.Options) (context.Context, context.CancelFunc) {
	timeout := rp.defaultTimeout
	if options != nil && options.Timeout > 0 {
		timeout = options.Timeout
	}
	return context.WithTimeout(r.Context(), timeout)
}

// addTarget 智能添加目标
func (rp *RequestParser) addTarget(builder *client.MessageBuilder, target string) error {
	if strings.Contains(target, "@") {
		builder.Email(target)
	} else {
		// 支持多种格式: "platform:userid" 或者 "userid"
		parts := strings.SplitN(target, ":", 2)
		if len(parts) == 2 {
			builder.User(parts[1], parts[0])
		} else {
			builder.User(target, "") // 平台通过路由确定
		}
	}
	return nil
}

// setPriority 设置优先级
func (rp *RequestParser) setPriority(builder *client.MessageBuilder, priority string) error {
	switch priority {
	case "urgent", "5":
		builder.Urgent()
	case "high", "4":
		builder.High()
	case "low", "2":
		builder.Low()
	case "minimal", "1":
		builder.Minimal()
	case "", "normal", "3":
		builder.Normal()
	default:
		return &ParseError{
			Type: "invalid_priority",
			Err:  fmt.Errorf("invalid priority: %s", priority),
		}
	}
	return nil
}

// ParseError 解析错误
type ParseError struct {
	Type string
	Err  error
}

func (pe *ParseError) Error() string {
	return fmt.Sprintf("parse error (%s): %v", pe.Type, pe.Err)
}

func (pe *ParseError) Unwrap() error {
	return pe.Err
}