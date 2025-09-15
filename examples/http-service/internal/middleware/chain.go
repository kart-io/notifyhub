package middleware

import (
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/logger"
)

// MiddlewareChain 中间件链管理器
type MiddlewareChain struct {
	middlewares []func(http.Handler) http.Handler
}

// NewChain 创建新的中间件链
func NewChain() *MiddlewareChain {
	return &MiddlewareChain{}
}

// Add 添加中间件到链中
func (mc *MiddlewareChain) Add(middleware func(http.Handler) http.Handler) *MiddlewareChain {
	mc.middlewares = append(mc.middlewares, middleware)
	return mc
}

// Then 将中间件链应用到最终的handler上
func (mc *MiddlewareChain) Then(handler http.Handler) http.Handler {
	// 从后往前应用中间件，确保正确的执行顺序
	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		handler = mc.middlewares[i](handler)
	}
	return handler
}

// ChainBuilder 中间件链构建器，提供预定义的链配置
type ChainBuilder struct {
	logger logger.Interface
}

// NewChainBuilder 创建中间件链构建器
func NewChainBuilder(logger logger.Interface) *ChainBuilder {
	return &ChainBuilder{logger: logger}
}

// PublicChain 创建公共API的中间件链
func (cb *ChainBuilder) PublicChain() *MiddlewareChain {
	return NewChain().
		Add(SecurityHeaders).
		Add(JSONMiddleware).
		Add(CORSMiddleware).
		Add(RecoveryMiddleware(cb.logger))
}

// ProtectedChain 创建受保护API的中间件链
func (cb *ChainBuilder) ProtectedChain(apiKey string, rateLimit int, maxRequestSize int64) *MiddlewareChain {
	return cb.PublicChain().
		Add(AuthMiddleware(apiKey)).
		Add(RateLimitMiddleware(rateLimit)).
		Add(RequestSizeLimit(maxRequestSize)).
		Add(ValidateContentType).
		Add(LoggingMiddleware(cb.logger))
}

// DebugChain 创建调试模式的中间件链（包含详细日志）
func (cb *ChainBuilder) DebugChain(apiKey string) *MiddlewareChain {
	return NewChain().
		Add(SecurityHeaders).
		Add(JSONMiddleware).
		Add(CORSMiddleware).
		Add(DebugLoggingMiddleware(cb.logger)). // 详细的debug日志
		Add(AuthMiddleware(apiKey)).
		Add(RecoveryMiddleware(cb.logger))
}

// MonitoringChain 创建监控端点的中间件链
func (cb *ChainBuilder) MonitoringChain() *MiddlewareChain {
	return NewChain().
		Add(SecurityHeaders).
		Add(JSONMiddleware).
		Add(MetricsMiddleware). // 收集指标
		Add(RecoveryMiddleware(cb.logger))
}

// DebugLoggingMiddleware 详细的debug日志中间件
func DebugLoggingMiddleware(logger logger.Interface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debug(r.Context(), "Request: %s %s, Headers: %v, RemoteAddr: %s",
				r.Method, r.RequestURI, r.Header, r.RemoteAddr)

			// 继续处理请求
			next.ServeHTTP(w, r)

			logger.Debug(r.Context(), "Response completed for: %s %s", r.Method, r.RequestURI)
		})
	}
}

// MetricsMiddleware 指标收集中间件
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 这里可以收集请求指标
		// 例如：请求计数、响应时间、状态码分布等
		start := time.Now()

		// 创建response writer包装器来捕获状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		// 记录指标
		duration := time.Since(start)
		// 这里可以发送到监控系统，比如 Prometheus
		_ = duration
		_ = wrapped.statusCode
	})
}

// responseWriter 包装器用于捕获响应状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}