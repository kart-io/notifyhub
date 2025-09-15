package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/notifiers"
)

// ResponseHandler 统一处理HTTP响应格式
type ResponseHandler struct {
	logger logger.Interface
}

// NewResponseHandler 创建响应处理器
func NewResponseHandler(logger logger.Interface) *ResponseHandler {
	return &ResponseHandler{logger: logger}
}

// Success 发送成功响应
func (rh *ResponseHandler) Success(w http.ResponseWriter, data interface{}, message string) {
	response := client.CreateSuccessResponse(message, data)
	rh.writeJSON(w, http.StatusOK, response)
}

// AsyncSuccess 发送异步操作成功响应
func (rh *ResponseHandler) AsyncSuccess(w http.ResponseWriter, taskID string) {
	response := client.CreateAsyncSuccessResponse(taskID)
	rh.writeJSON(w, http.StatusAccepted, response)
}

// Error 发送错误响应
func (rh *ResponseHandler) Error(w http.ResponseWriter, err error) {
	var statusCode int
	var message string
	var errors []string

	switch e := err.(type) {
	case notifiers.ValidationErrors:
		statusCode = http.StatusBadRequest
		message = "Validation failed"
		for _, ve := range e {
			errors = append(errors, ve.Error())
		}
	case *notifiers.ValidationError:
		statusCode = http.StatusBadRequest
		message = "Validation failed"
		errors = []string{e.Error()}
	default:
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
		errors = []string{err.Error()}
		rh.logger.Error(nil, "Internal error: %v", err)
	}

	response := client.CreateErrorResponse(message, errors...)
	rh.writeJSON(w, statusCode, response)
}

// BadRequest 发送400错误
func (rh *ResponseHandler) BadRequest(w http.ResponseWriter, message string, errors ...string) {
	response := client.CreateErrorResponse(message, errors...)
	rh.writeJSON(w, http.StatusBadRequest, response)
}

// InternalError 发送500错误
func (rh *ResponseHandler) InternalError(w http.ResponseWriter, err error) {
	rh.logger.Error(nil, "Internal server error: %v", err)
	response := client.CreateErrorResponse("Internal server error", err.Error())
	rh.writeJSON(w, http.StatusInternalServerError, response)
}

// writeJSON 写入JSON响应
func (rh *ResponseHandler) writeJSON(w http.ResponseWriter, statusCode int, response *client.HTTPResponse) {
	if err := client.WriteJSONResponse(w, statusCode, response); err != nil {
		rh.logger.Error(nil, "Failed to write JSON response: %v", err)
	}
}