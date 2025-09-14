package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kart-io/notifyhub"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/logger/adapters"
)

// ================================
// 示例1: 简单的控制台日志器
// ================================

type ConsoleLogger struct {
	prefix string
}

func NewConsoleLogger(prefix string) *ConsoleLogger {
	return &ConsoleLogger{prefix: prefix}
}

// 实现 adapters.CustomLogger 接口
func (c *ConsoleLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := level.String()

	output := fmt.Sprintf("[%s] [%s] %s%s", timestamp, levelStr, c.prefix, msg)

	if len(fields) > 0 {
		output += " fields:"
		for k, v := range fields {
			output += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	fmt.Println(output)
}

// ================================
// 示例2: JSON 格式日志器
// ================================

type JSONLogger struct {
	serviceName string
}

func NewJSONLogger(serviceName string) *JSONLogger {
	return &JSONLogger{serviceName: serviceName}
}

// 实现 adapters.CustomLogger 接口
func (j *JSONLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
	logEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level.String(),
		"service":   j.serviceName,
		"message":   msg,
	}

	// 合并字段
	for k, v := range fields {
		logEntry[k] = v
	}

	jsonData, _ := json.Marshal(logEntry)
	fmt.Println(string(jsonData))
}

// ================================
// 示例3: 文件日志器
// ================================

type FileLogger struct {
	file   *os.File
	prefix string
}

func NewFileLogger(filename, prefix string) (*FileLogger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &FileLogger{
		file:   file,
		prefix: prefix,
	}, nil
}

func (f *FileLogger) Close() error {
	return f.file.Close()
}

// 实现 adapters.CustomLogger 接口
func (f *FileLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := level.String()

	output := fmt.Sprintf("[%s] [%s] %s%s", timestamp, levelStr, f.prefix, msg)

	if len(fields) > 0 {
		output += " fields:"
		for k, v := range fields {
			output += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	output += "\n"
	f.file.WriteString(output)
}

// ================================
// 示例4: 多目标日志器（同时输出到控制台和文件）
// ================================

type MultiTargetLogger struct {
	targets []adapters.CustomLogger
}

func NewMultiTargetLogger(targets ...adapters.CustomLogger) *MultiTargetLogger {
	return &MultiTargetLogger{targets: targets}
}

// 实现 adapters.CustomLogger 接口
func (m *MultiTargetLogger) Log(level logger.LogLevel, msg string, fields map[string]interface{}) {
	for _, target := range m.targets {
		target.Log(level, msg, fields)
	}
}

// ================================
// 主函数演示
// ================================

func main() {
	log.Println("🚀 NotifyHub 自定义日志适配器演示")
	log.Println("=========================================")

	ctx := context.Background()

	// ========================================
	// 示例1：简单控制台日志器
	// ========================================
	log.Println("\n📺 示例1: 简单控制台日志器")
	log.Println("---------------------------------")

	consoleLogger := NewConsoleLogger("[NOTIFYHUB-CONSOLE] ")

	hub1, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithLogger(
			notifyhub.NewCustomAdapter(consoleLogger, notifyhub.LogLevelInfo),
		),
		notifyhub.WithQueue("memory", 50, 1),
	)
	if err != nil {
		log.Fatalf("Failed to create hub1: %v", err)
	}

	hub1.Start(ctx)

	message1 := notifyhub.NewAlert("控制台日志测试", "使用自定义控制台日志器").
		Variable("test_type", "console_logger").
		FeishuGroup("console-group").
		Build()

	_, err = hub1.Send(ctx, message1, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		log.Printf("控制台日志消息发送成功")
	}

	hub1.Stop()

	// ========================================
	// 示例2：JSON格式日志器
	// ========================================
	log.Println("\n📄 示例2: JSON格式日志器")
	log.Println("---------------------------------")

	jsonLogger := NewJSONLogger("notifyhub-service")

	hub2, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithLogger(
			notifyhub.NewCustomAdapter(jsonLogger, notifyhub.LogLevelDebug),
		),
		notifyhub.WithQueue("memory", 50, 1),
	)
	if err != nil {
		log.Fatalf("Failed to create hub2: %v", err)
	}

	hub2.Start(ctx)

	message2 := notifyhub.NewReport("JSON日志测试", "使用JSON格式的结构化日志").
		Variable("format", "json").
		Variable("structured", true).
		FeishuGroup("json-group").
		Build()

	_, err = hub2.Send(ctx, message2, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		log.Printf("JSON格式日志消息发送成功")
	}

	hub2.Stop()

	// ========================================
	// 示例3：文件日志器
	// ========================================
	log.Println("\n💾 示例3: 文件日志器")
	log.Println("---------------------------------")

	fileLogger, err := NewFileLogger("/tmp/notifyhub.log", "[NOTIFYHUB-FILE] ")
	if err != nil {
		log.Fatalf("Failed to create file logger: %v", err)
	}
	defer fileLogger.Close()

	hub3, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithLogger(
			notifyhub.NewCustomAdapter(fileLogger, notifyhub.LogLevelWarn),
		),
		notifyhub.WithQueue("memory", 50, 1),
	)
	if err != nil {
		log.Fatalf("Failed to create hub3: %v", err)
	}

	hub3.Start(ctx)

	message3 := notifyhub.NewNotice("文件日志测试", "日志将写入文件 /tmp/notifyhub.log").
		Variable("output", "file").
		Variable("path", "/tmp/notifyhub.log").
		FeishuGroup("file-group").
		Build()

	_, err = hub3.Send(ctx, message3, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		log.Printf("文件日志消息发送成功，检查文件: /tmp/notifyhub.log")
	}

	hub3.Stop()

	// ========================================
	// 示例4：多目标日志器
	// ========================================
	log.Println("\n🎯 示例4: 多目标日志器（控制台 + 文件 + JSON）")
	log.Println("---------------------------------")

	// 创建多个目标
	consoleTarget := NewConsoleLogger("[MULTI-CONSOLE] ")
	jsonTarget := NewJSONLogger("multi-target-service")
	fileTarget, _ := NewFileLogger("/tmp/notifyhub-multi.log", "[MULTI-FILE] ")
	defer fileTarget.Close()

	multiLogger := NewMultiTargetLogger(consoleTarget, jsonTarget, fileTarget)

	hub4, err := notifyhub.New(
		notifyhub.WithFeishu("https://httpbin.org/post", ""),
		notifyhub.WithLogger(
			notifyhub.NewCustomAdapter(multiLogger, notifyhub.LogLevelInfo),
		),
		notifyhub.WithQueue("memory", 50, 1),
	)
	if err != nil {
		log.Fatalf("Failed to create hub4: %v", err)
	}

	hub4.Start(ctx)

	message4 := notifyhub.NewAlert("多目标日志测试", "同时输出到控制台、JSON和文件").
		Variable("targets", []string{"console", "json", "file"}).
		Variable("multi_output", true).
		FeishuGroup("multi-group").
		Build()

	_, err = hub4.Send(ctx, message4, nil)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		log.Printf("多目标日志消息发送成功")
	}

	hub4.Stop()

	log.Println("\n🎉 自定义日志适配器演示完成!")
	log.Println("=========================================")
	log.Println("💡 自定义适配器特点:")
	log.Println("• 简单接口: 只需实现 CustomLogger 接口的 Log 方法")
	log.Println("• 高度灵活: 支持任意格式和输出目标")
	log.Println("• 结构化数据: 自动解析和传递结构化字段")
	log.Println("• 可组合: 支持多目标和复杂的日志策略")
	log.Println("• 高性能: 基于接口的设计，运行时开销最小")
	log.Println("")
	log.Println("📁 文件输出:")
	log.Println("• /tmp/notifyhub.log - 单目标文件日志")
	log.Println("• /tmp/notifyhub-multi.log - 多目标文件日志")
}