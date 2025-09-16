package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/logger"
)

// CustomLogger 实现了自定义日志器
type CustomLogger struct {
	prefix string
	level  logger.LogLevel
}

// NewCustomLogger 创建自定义日志器
func NewCustomLogger(prefix string) logger.Interface {
	return &CustomLogger{
		prefix: prefix,
		level:  logger.Info,
	}
}

// LogMode 设置日志级别
func (cl *CustomLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &CustomLogger{
		prefix: cl.prefix,
		level:  level,
	}
}

// Debug 输出调试日志
func (cl *CustomLogger) Debug(ctx context.Context, msg string, data ...interface{}) {
	if cl.level >= logger.Debug {
		log.Printf("[DEBUG %s] %s %v", cl.prefix, msg, data)
	}
}

// Info 输出信息日志
func (cl *CustomLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if cl.level >= logger.Info {
		log.Printf("[INFO %s] %s %v", cl.prefix, msg, data)
	}
}

// Warn 输出警告日志
func (cl *CustomLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if cl.level >= logger.Warn {
		log.Printf("[WARN %s] %s %v", cl.prefix, msg, data)
	}
}

// Error 输出错误日志
func (cl *CustomLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if cl.level >= logger.Error {
		log.Printf("[ERROR %s] %s %v", cl.prefix, msg, data)
	}
}

// Trace 记录操作跟踪
func (cl *CustomLogger) Trace(ctx context.Context, begin time.Time, fc func() (operation string, targets int64), err error) {
	if cl.level >= logger.Debug {
		operation, targets := fc()
		elapsed := time.Since(begin)
		if err != nil {
			log.Printf("[TRACE %s] %s failed in %v (targets: %d) error: %v", cl.prefix, operation, elapsed, targets, err)
		} else {
			log.Printf("[TRACE %s] %s completed in %v (targets: %d)", cl.prefix, operation, elapsed, targets)
		}
	}
}

func main() {
	// 创建自定义日志器
	customLogger := NewCustomLogger("NotifyHub")

	// 使用自定义日志器创建NotifyHub实例
	hub, err := client.New(
		config.WithTestDefaults(),
		config.WithLogger(customLogger),
	)
	if err != nil {
		log.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("Failed to start NotifyHub: %v", err)
	}
	defer hub.Stop()

	// 发送测试消息
	message := client.NewNotice("自定义日志器测试", "测试自定义日志器功能").
		Email("test@example.com").
		Variable("timestamp", time.Now().Format(time.RFC3339)).
		Build()

	results, err := hub.Send(ctx, message, nil)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		log.Printf("发送成功，结果数量: %d", len(results))
		for _, result := range results {
			log.Printf("  平台: %s, 成功: %v, 耗时: %v",
				result.Platform, result.Success, result.Duration)
		}
	}

	log.Println("自定义日志器示例执行完成")
}