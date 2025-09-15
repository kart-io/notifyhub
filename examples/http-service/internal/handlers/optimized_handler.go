package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/notifiers"
)

// OptimizedNotificationHandler 展示优化后的handler实现
type OptimizedNotificationHandler struct {
	hub      *client.Hub
	logger   logger.Interface
	response *ResponseHandler
	parser   *RequestParser
}

// NewOptimizedNotificationHandler 创建优化的通知handler
func NewOptimizedNotificationHandler(hub *client.Hub, logger logger.Interface) *OptimizedNotificationHandler {
	return &OptimizedNotificationHandler{
		hub:      hub,
		logger:   logger,
		response: NewResponseHandler(logger),
		parser:   NewRequestParser(),
	}
}

// SendNotification 发送单个通知 - 优化版本
func (h *OptimizedNotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	// 1. 统一的请求解析
	message, options, err := h.parser.ParseNotificationRequest(r)
	if err != nil {
		h.response.Error(w, err)
		return
	}

	// 2. 创建带超时的context
	ctx, cancel := h.parser.CreateTimeoutContext(r, options)
	defer cancel()

	// 3. 发送通知
	result, err := h.sendWithOptions(ctx, message, options)
	if err != nil {
		h.response.Error(w, err)
		return
	}

	// 4. 统一的成功响应
	h.logSuccess(message, result)
	if result.IsAsync {
		h.response.AsyncSuccess(w, result.TaskID)
	} else {
		h.response.Success(w, result.ToMap(), "Notification sent successfully")
	}
}

// SendQuickNotification 快速通知 - 优化版本
func (h *OptimizedNotificationHandler) SendQuickNotification(w http.ResponseWriter, r *http.Request) {
	// 1. 解析查询参数
	message, err := h.parser.ParseQuickNotification(r)
	if err != nil {
		h.response.Error(w, err)
		return
	}

	// 2. 发送通知
	ctx, cancel := h.parser.CreateTimeoutContext(r, nil)
	defer cancel()

	results, err := h.hub.Send(ctx, message, nil)
	if err != nil {
		h.response.InternalError(w, err)
		return
	}

	// 3. 返回结果
	result := &SendResult{
		MessageID: message.ID,
		Results:   results,
		IsAsync:   false,
	}

	h.logSuccess(message, result)
	h.response.Success(w, result.ToMap(), "Quick notification sent successfully")
}

// SendBulkNotifications 批量通知 - 优化版本
func (h *OptimizedNotificationHandler) SendBulkNotifications(w http.ResponseWriter, r *http.Request) {
	// 检查批量发送参数
	bulk := r.URL.Query().Get("bulk") == "true"
	if !bulk {
		h.response.BadRequest(w, "Missing bulk=true parameter")
		return
	}

	// 解析批量请求
	messages, options, err := h.parser.ParseBulkNotificationRequest(r)
	if err != nil {
		h.response.Error(w, err)
		return
	}

	// 发送批量通知
	ctx, cancel := h.parser.CreateTimeoutContext(r, options)
	defer cancel()

	results, err := h.hub.SendBatch(ctx, messages, options)
	if err != nil {
		h.response.InternalError(w, err)
		return
	}

	// 返回批量结果
	batchResult := &BulkSendResult{
		Total:      len(messages),
		Successful: countSuccessful(results),
		Results:    results,
	}

	h.response.Success(w, batchResult.ToMap(), "Bulk notifications processed")
}

// sendWithOptions 根据选项发送通知
func (h *OptimizedNotificationHandler) sendWithOptions(ctx context.Context, message *notifiers.Message, options *client.Options) (*SendResult, error) {
	if options != nil && options.Async {
		// 异步发送
		taskID, err := h.hub.SendAsync(ctx, message, options)
		if err != nil {
			return nil, err
		}
		return &SendResult{
			MessageID: message.ID,
			TaskID:    taskID,
			IsAsync:   true,
		}, nil
	}

	// 同步发送
	results, err := h.hub.Send(ctx, message, options)
	if err != nil {
		return nil, err
	}

	return &SendResult{
		MessageID: message.ID,
		Results:   results,
		IsAsync:   false,
	}, nil
}

// logSuccess 记录成功日志
func (h *OptimizedNotificationHandler) logSuccess(message *notifiers.Message, result *SendResult) {
	if result.IsAsync {
		h.logger.Info(nil, "Notification enqueued successfully: id=%s, task_id=%s",
			message.ID, result.TaskID)
	} else {
		h.logger.Info(nil, "Notification sent successfully: id=%s, targets=%d, results=%d",
			message.ID, len(message.Targets), len(result.Results))
	}
}

// SendResult 发送结果
type SendResult struct {
	MessageID string                        `json:"message_id"`
	TaskID    string                        `json:"task_id,omitempty"`
	Results   []*notifiers.SendResult      `json:"results,omitempty"`
	IsAsync   bool                         `json:"is_async"`
}

// ToMap 转换为map用于响应
func (sr *SendResult) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"message_id": sr.MessageID,
		"is_async":   sr.IsAsync,
	}

	if sr.IsAsync {
		result["task_id"] = sr.TaskID
	} else {
		result["results"] = sr.Results
		result["target_count"] = len(sr.Results)
	}

	return result
}

// BulkSendResult 批量发送结果
type BulkSendResult struct {
	Total      int                          `json:"total"`
	Successful int                          `json:"successful"`
	Results    []*notifiers.SendResult     `json:"results"`
}

// ToMap 转换为map用于响应
func (bsr *BulkSendResult) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"total":          bsr.Total,
		"successful":     bsr.Successful,
		"failed":         bsr.Total - bsr.Successful,
		"success_rate":   float64(bsr.Successful) / float64(bsr.Total),
		"results":        bsr.Results,
	}
}

// countSuccessful 计算成功数量
func countSuccessful(results []*notifiers.SendResult) int {
	count := 0
	for _, result := range results {
		if result.Success {
			count++
		}
	}
	return count
}