package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub"
)

func main() {
	// 创建NotifyHub实例
	hub, err := notifyhub.NewWithDefaults()
	if err != nil {
		log.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("Failed to start NotifyHub: %v", err)
	}
	defer hub.Stop()

	// 创建回调配置
	callbacks := &notifyhub.CallbackOptions{
		CallbackTimeout: 10 * time.Second,
		// Webhook回调示例（可选）
		// WebhookURL: "https://your-webhook-endpoint.com/callback",
		// WebhookSecret: "your-secret",
	}

	// 添加成功发送回调
	callbacks.AddCallback(notifyhub.CallbackEventSent, notifyhub.NewCallbackFunc("success-handler", func(ctx context.Context, callbackCtx *notifyhub.CallbackContext) error {
		log.Printf("✅ 消息发送成功! ID: %s, 尝试次数: %d, 耗时: %v",
			callbackCtx.MessageID, callbackCtx.Attempts, callbackCtx.Duration)
		return nil
	}))

	// 添加失败回调
	callbacks.AddCallback(notifyhub.CallbackEventFailed, notifyhub.NewCallbackFunc("failure-handler", func(ctx context.Context, callbackCtx *notifyhub.CallbackContext) error {
		log.Printf("❌ 消息发送失败! ID: %s, 错误: %v, 尝试次数: %d",
			callbackCtx.MessageID, callbackCtx.Error, callbackCtx.Attempts)
		return nil
	}))

	// 添加重试回调
	callbacks.AddCallback(notifyhub.CallbackEventRetry, notifyhub.NewCallbackFunc("retry-handler", func(ctx context.Context, callbackCtx *notifyhub.CallbackContext) error {
		log.Printf("🔄 消息正在重试! ID: %s, 尝试次数: %d",
			callbackCtx.MessageID, callbackCtx.Attempts)
		return nil
	}))

	// 添加达到最大重试次数回调
	callbacks.AddCallback(notifyhub.CallbackEventMaxRetries, notifyhub.NewCallbackFunc("max-retries-handler", func(ctx context.Context, callbackCtx *notifyhub.CallbackContext) error {
		log.Printf("🚫 消息达到最大重试次数! ID: %s, 总尝试: %d",
			callbackCtx.MessageID, callbackCtx.Attempts)
		return nil
	}))

	// 添加日志回调
	callbacks.AddCallback(notifyhub.CallbackEventSent, notifyhub.NewLoggingCallback("audit-logger", nil))

	// 创建发送选项
	options := notifyhub.NewAsyncOptions().WithCallbacks(callbacks)

	// 创建消息
	message := notifyhub.NewAlert("回调测试", "这是一条用于测试回调功能的消息").
		Email("test@example.com").
		Variable("test_time", time.Now().Format(time.RFC3339)).
		Build()

	// 异步发送消息（这样可以观察到回调执行）
	taskID, err := hub.SendAsync(ctx, message, options)
	if err != nil {
		log.Printf("发送消息失败: %v", err)
	} else {
		log.Printf("消息已加入队列，任务ID: %s", taskID)
	}

	// 再发送一条同步消息进行对比
	syncMessage := notifyhub.NewNotice("同步测试", "这是同步发送的消息，不会触发回调").
		Email("test@example.com").
		Build()

	results, err := hub.Send(ctx, syncMessage, nil)
	if err != nil {
		log.Printf("同步发送失败: %v", err)
	} else {
		log.Printf("同步发送完成，结果数量: %d", len(results))
	}

	// 等待异步消息处理完成
	log.Println("等待回调执行...")
	time.Sleep(5 * time.Second)

	// 显示指标
	metrics := hub.GetMetrics()
	log.Printf("📊 最终指标: %+v", metrics)
}
