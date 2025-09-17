package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
)

// 飞书批量发送示例
func main() {
	// 创建配置了批量处理的 NotifyHub
	hub, err := client.New(
		config.WithFeishuFromEnv(),
		config.WithQueue("memory", 5000, 16), // 大容量队列，16个工作协程
		config.WithTelemetry("feishu-batch", "1.0.0", "production", ""),
	)
	if err != nil {
		log.Fatalf("创建 NotifyHub 失败: %v", err)
	}

	ctx := context.Background()
	if err := hub.Start(ctx); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
	defer hub.Stop()

	// 示例1: 基础批量发送
	demonstrateBasicBatch(hub, ctx)

	// 示例2: 分组批量发送
	demonstrateGroupedBatch(hub, ctx)

	// 示例3: 并发批量发送
	demonstrateConcurrentBatch(hub, ctx)

	// 示例4: 混合类型批量发送
	demonstrateMixedTypeBatch(hub, ctx)

	// 示例5: 大规模批量发送
	demonstrateLargeScaleBatch(hub, ctx)

	// 示例6: 智能批量发送（根据优先级分批）
	demonstrateSmartBatch(hub, ctx)

	// 等待所有批量任务完成
	time.Sleep(10 * time.Second)

	// 显示批量发送统计
	showBatchStats(hub)
}

// 基础批量发送
func demonstrateBasicBatch(hub *client.Hub, ctx context.Context) {
	fmt.Println("=== 基础批量发送示例 ===")

	// 创建批量构建器
	batch := hub.NewEnhancedBatch()

	// 添加多个消息到同一个群
	groupID := "team-notifications"
	messages := []string{
		"今日站会将在30分钟后开始",
		"请大家准备好工作汇报",
		"会议链接已发送到邮箱",
		"预计会议时长30分钟",
		"感谢大家的配合",
	}

	for i, content := range messages {
		message := client.NewMessage().
			Title(fmt.Sprintf("站会通知 %d/5", i+1)).
			Body(content).
			FeishuGroup(groupID).
			Priority(3). // 3=normal
			Metadata("batch_type", "basic").
			Metadata("sequence", fmt.Sprintf("%d", i+1)).
			Build()

		target := notifiers.Target{Type: notifiers.TargetTypeGroup, Value: groupID, Platform: "feishu"}
		batch.AddMessage(message, []notifiers.Target{target}, &client.Options{
			Retry:      true,
			MaxRetries: 2,
		})
	}

	// 执行批量发送
	start := time.Now()
	results, err := batch.SendAll(ctx)
	duration := time.Since(start)

	if err != nil {
		log.Printf("基础批量发送失败: %v", err)
	} else {
		successCount := countSuccessResults(results)
		fmt.Printf("✅ 基础批量发送完成: 成功 %d/%d, 耗时: %v\n",
			successCount, len(results), duration)
	}
}

// 分组批量发送
func demonstrateGroupedBatch(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 分组批量发送示例 ===")

	// 定义不同团队
	teams := map[string][]string{
		"backend-team":  {"后端服务部署完成", "数据库优化已上线", "API性能提升15%"},
		"frontend-team": {"UI界面更新完成", "用户体验优化上线", "移动端适配完成"},
		"ops-team":      {"监控系统升级", "自动化部署配置", "安全扫描通过"},
		"qa-team":       {"测试用例更新", "自动化测试通过", "性能测试完成"},
	}

	var allBatches []*client.EnhancedBatchBuilder

	// 为每个团队创建独立的批次
	for teamID, notifications := range teams {
		teamBatch := hub.NewEnhancedBatch()

		for i, notification := range notifications {
			message := client.NewNotice(
				fmt.Sprintf("%s - 进展更新", getTeamName(teamID)),
				notification,
			).
				FeishuGroup(teamID).
				Variable("team", getTeamName(teamID)).
				Variable("update_number", i+1).
				Priority(3). // 3=normal
				Metadata("team_id", teamID).
				Build()

			target := notifiers.Target{Type: notifiers.TargetTypeGroup, Value: teamID, Platform: "feishu"}
			teamBatch.AddMessage(message, []notifiers.Target{target}, nil)
		}

		allBatches = append(allBatches, teamBatch)
	}

	// 并行发送所有团队的批次
	var wg sync.WaitGroup
	results := make(chan batchResult, len(allBatches))

	for i, batch := range allBatches {
		wg.Add(1)
		go func(batchIndex int, b *client.EnhancedBatchBuilder) {
			defer wg.Done()

			start := time.Now()
			batchResults, err := b.SendAll(ctx)
			duration := time.Since(start)

			results <- batchResult{
				Index:    batchIndex,
				Results:  batchResults,
				Duration: duration,
				Error:    err,
			}
		}(i, batch)
	}

	wg.Wait()
	close(results)

	// 收集结果
	totalSuccess := 0
	totalMessages := 0

	for result := range results {
		if result.Error != nil {
			log.Printf("团队批次 %d 发送失败: %v", result.Index, result.Error)
			continue
		}

		successCount := countSuccessResults(result.Results)
		totalSuccess += successCount
		totalMessages += len(result.Results)

		fmt.Printf("✅ 团队批次 %d: 成功 %d/%d, 耗时: %v\n",
			result.Index+1, successCount, len(result.Results), result.Duration)
	}

	fmt.Printf("📊 分组批量汇总: 总成功 %d/%d, 成功率: %.1f%%\n",
		totalSuccess, totalMessages, float64(totalSuccess)/float64(totalMessages)*100)
}

// 并发批量发送
func demonstrateConcurrentBatch(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 并发批量发送示例 ===")

	// 模拟高频率通知场景
	const (
		numBatches       = 5
		messagesPerBatch = 10
	)

	var wg sync.WaitGroup
	results := make(chan batchResult, numBatches)

	// 创建多个并发批次
	for batchNum := 0; batchNum < numBatches; batchNum++ {
		wg.Add(1)

		go func(batchIndex int) {
			defer wg.Done()

			batch := hub.NewEnhancedBatch()

			// 为每个批次添加消息
			for msgNum := 0; msgNum < messagesPerBatch; msgNum++ {
				groupID := fmt.Sprintf("concurrent-group-%d", (batchIndex%3)+1)
				message := client.NewMessage().
					Title(fmt.Sprintf("并发批次 %d - 消息 %d", batchIndex+1, msgNum+1)).
					Body(fmt.Sprintf("这是来自批次 %d 的第 %d 条消息", batchIndex+1, msgNum+1)).
					Priority(3). // 3=normal
					Metadata("batch_index", fmt.Sprintf("%d", batchIndex)).
					Metadata("message_index", fmt.Sprintf("%d", msgNum)).
					Build()

				target := notifiers.Target{Type: notifiers.TargetTypeGroup, Value: groupID, Platform: "feishu"}
				batch.AddMessage(message, []notifiers.Target{target}, &client.Options{
					Retry:      true,
					MaxRetries: 1,
					Timeout:    5 * time.Second,
				})
			}

			// 发送批次
			start := time.Now()
			batchResults, err := batch.SendAll(ctx)
			duration := time.Since(start)

			results <- batchResult{
				Index:    batchIndex,
				Results:  batchResults,
				Duration: duration,
				Error:    err,
			}

		}(batchNum)
	}

	// 等待所有批次完成
	wg.Wait()
	close(results)

	// 统计结果
	var totalDuration time.Duration
	totalSuccess := 0
	totalMessages := 0

	fmt.Println("并发批次结果:")
	for result := range results {
		if result.Error != nil {
			log.Printf("❌ 批次 %d 失败: %v", result.Index+1, result.Error)
			continue
		}

		successCount := countSuccessResults(result.Results)
		totalSuccess += successCount
		totalMessages += len(result.Results)
		totalDuration += result.Duration

		fmt.Printf("✅ 批次 %d: %d/%d 成功, 耗时: %v\n",
			result.Index+1, successCount, len(result.Results), result.Duration)
	}

	avgDuration := totalDuration / time.Duration(numBatches)
	fmt.Printf("📊 并发批量汇总: 总成功 %d/%d, 平均耗时: %v\n",
		totalSuccess, totalMessages, avgDuration)
}

// 混合类型批量发送
func demonstrateMixedTypeBatch(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 混合类型批量发送示例 ===")

	batch := hub.NewEnhancedBatch()

	// 添加不同类型的消息
	messageTypes := []struct {
		msgType  string
		priority int
		title    string
		content  string
		group    string
	}{
		{"alert", 5, "🚨 紧急告警", "服务器CPU使用率超过90%", "ops-alerts"},      // 5=urgent
		{"notice", 3, "📢 部署通知", "新版本将在1小时后部署", "dev-notifications"}, // 3=normal
		{"report", 1, "📊 日报", "今日系统运行正常", "daily-reports"},          // 1=low
		{"alert", 4, "⚠️ 性能警告", "数据库查询响应时间较慢", "dba-alerts"},        // 4=high
		{"notice", 3, "🎉 功能上线", "新的用户界面已发布", "product-updates"},     // 3=normal
	}

	for i, msgType := range messageTypes {
		var message *notifiers.Message

		switch msgType.msgType {
		case "alert":
			message = client.NewAlert(msgType.title, msgType.content).
				Variable("alert_id", fmt.Sprintf("ALT-%d", i+1)).
				Variable("timestamp", time.Now().Format("15:04:05")).
				Priority(msgType.priority).
				Build()

		case "notice":
			message = client.NewNotice(msgType.title, msgType.content).
				Variable("notice_id", fmt.Sprintf("NTC-%d", i+1)).
				Variable("department", "技术部").
				Priority(msgType.priority).
				Build()

		case "report":
			message = client.NewReport(msgType.title, msgType.content).
				Variable("report_id", fmt.Sprintf("RPT-%d", i+1)).
				Variable("date", time.Now().Format("2006-01-02")).
				Priority(msgType.priority).
				Build()
		}

		target := notifiers.Target{Type: notifiers.TargetTypeGroup, Value: msgType.group, Platform: "feishu"}
		batch.AddMessage(message, []notifiers.Target{target}, &client.Options{
			Retry:      true,
			MaxRetries: 3,
			Timeout:    15 * time.Second,
		})

		fmt.Printf("📝 添加 %s 消息: %s (优先级: %s)\n",
			msgType.msgType, msgType.title, getPriorityName(msgType.priority))
	}

	// 发送混合批次
	start := time.Now()
	results, err := batch.SendAll(ctx)
	duration := time.Since(start)

	if err != nil {
		log.Printf("混合类型批量发送失败: %v", err)
	} else {
		successCount := countSuccessResults(results)
		fmt.Printf("✅ 混合类型批量完成: 成功 %d/%d, 耗时: %v\n",
			successCount, len(results), duration)

		// 按优先级统计
		priorityStats := make(map[int]int)
		for _, msgType := range messageTypes {
			priorityStats[msgType.priority]++
		}

		fmt.Println("优先级分布:")
		for priority, count := range priorityStats {
			fmt.Printf("  %s: %d 条\n", getPriorityName(priority), count)
		}
	}
}

// 大规模批量发送
func demonstrateLargeScaleBatch(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 大规模批量发送示例 ===")

	const (
		totalMessages = 100
		batchSize     = 20 // 每批20条消息
	)

	var allResults [][]*notifiers.SendResult
	totalStart := time.Now()

	// 分批发送
	for i := 0; i < totalMessages; i += batchSize {
		end := i + batchSize
		if end > totalMessages {
			end = totalMessages
		}

		batch := hub.NewEnhancedBatch()

		// 创建当前批次的消息
		for j := i; j < end; j++ {
			groupID := fmt.Sprintf("large-scale-group-%d", (j%5)+1)
			message := client.NewMessage().
				Title(fmt.Sprintf("大规模消息 #%d", j+1)).
				Body(fmt.Sprintf("这是第 %d 条大规模批量消息", j+1)).
				Priority(3). // 3=normal
				Metadata("message_number", fmt.Sprintf("%d", j+1)).
				Metadata("batch_number", fmt.Sprintf("%d", (i/batchSize)+1)).
				Build()

			target := notifiers.Target{Type: notifiers.TargetTypeGroup, Value: groupID, Platform: "feishu"}
			batch.AddMessage(message, []notifiers.Target{target}, &client.Options{
				Retry:   false, // 大规模发送时禁用重试以提高速度
				Timeout: 3 * time.Second,
			})
		}

		// 发送当前批次
		batchStart := time.Now()
		results, err := batch.SendAll(ctx)
		batchDuration := time.Since(batchStart)

		if err != nil {
			log.Printf("❌ 批次 %d 发送失败: %v", (i/batchSize)+1, err)
			continue
		}

		allResults = append(allResults, results)
		successCount := countSuccessResults(results)

		fmt.Printf("✅ 批次 %d/%d: %d/%d 成功, 耗时: %v\n",
			(i/batchSize)+1, (totalMessages+batchSize-1)/batchSize,
			successCount, len(results), batchDuration)

		// 批次间稍作延迟以避免压垮系统
		time.Sleep(100 * time.Millisecond)
	}

	totalDuration := time.Since(totalStart)

	// 统计总体结果
	totalSuccess := 0
	totalSent := 0
	for _, results := range allResults {
		totalSuccess += countSuccessResults(results)
		totalSent += len(results)
	}

	fmt.Printf("📊 大规模发送汇总:\n")
	fmt.Printf("  总消息数: %d\n", totalSent)
	fmt.Printf("  成功数: %d\n", totalSuccess)
	fmt.Printf("  成功率: %.1f%%\n", float64(totalSuccess)/float64(totalSent)*100)
	fmt.Printf("  总耗时: %v\n", totalDuration)
	fmt.Printf("  平均速度: %.1f 消息/秒\n", float64(totalSent)/totalDuration.Seconds())
}

// 智能批量发送（根据优先级分批）
func demonstrateSmartBatch(hub *client.Hub, ctx context.Context) {
	fmt.Println("\n=== 智能批量发送示例 ===")

	// 按优先级分组的消息
	priorityGroups := map[int][]messageData{
		5: { // 5=urgent
			{"🚨 系统故障", "主数据库连接中断", "incident-response"},
			{"🔥 安全警报", "检测到异常登录", "security-team"},
		},
		4: { // 4=high
			{"⚠️ 性能警告", "API响应时间超过阈值", "ops-team"},
			{"🔔 容量告警", "磁盘使用率达到85%", "ops-team"},
		},
		3: { // 3=normal
			{"📢 版本发布", "v2.1.0 已发布到生产环境", "dev-team"},
			{"📊 周报", "本周开发进度汇总", "management"},
		},
		1: { // 1=low
			{"📝 日常维护", "定期数据库备份已完成", "dba-team"},
			{"🔄 定时任务", "日志清理任务执行完成", "ops-team"},
		},
	}

	// 按优先级顺序发送（紧急优先）
	priorities := []int{5, 4, 3, 1}

	for _, priority := range priorities {
		messages := priorityGroups[priority]
		if len(messages) == 0 {
			continue
		}

		fmt.Printf("🎯 发送 %s 优先级消息 (%d 条)\n", getPriorityName(priority), len(messages))

		batch := hub.NewEnhancedBatch()

		for i, msg := range messages {
			message := client.NewAlert(msg.title, msg.content).
				Variable("priority", getPriorityName(priority)).
				Variable("sequence", i+1).
				Priority(priority).
				Build()

			// 根据优先级调整发送选项
			var options *client.Options
			switch priority {
			case 5: // urgent
				options = &client.Options{
					Retry:      true,
					MaxRetries: 5,
					Timeout:    60 * time.Second,
				}
			case 4: // high
				options = &client.Options{
					Retry:      true,
					MaxRetries: 3,
					Timeout:    30 * time.Second,
				}
			default:
				options = &client.Options{
					Retry:      true,
					MaxRetries: 1,
					Timeout:    10 * time.Second,
				}
			}

			target := notifiers.Target{Type: notifiers.TargetTypeGroup, Value: msg.group, Platform: "feishu"}
			batch.AddMessage(message, []notifiers.Target{target}, options)
		}

		// 发送当前优先级的批次
		start := time.Now()
		results, err := batch.SendAll(ctx)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ %s 优先级批次发送失败: %v", getPriorityName(priority), err)
		} else {
			successCount := countSuccessResults(results)
			fmt.Printf("✅ %s 优先级完成: %d/%d 成功, 耗时: %v\n",
				getPriorityName(priority), successCount, len(results), duration)
		}

		// 优先级间的延迟（紧急消息立即发送，其他延迟处理）
		if priority != 5 { // 5=urgent
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// 显示批量发送统计
func showBatchStats(hub *client.Hub) {
	fmt.Println("\n=== 批量发送总体统计 ===")

	metrics := hub.GetMetrics()

	if totalSent, ok := metrics["total_sent"].(int64); ok {
		fmt.Printf("📤 总发送消息数: %d\n", totalSent)
	}

	if totalFailed, ok := metrics["total_failed"].(int64); ok {
		fmt.Printf("❌ 总失败消息数: %d\n", totalFailed)
	}

	if successRate, ok := metrics["success_rate"].(float64); ok {
		fmt.Printf("📊 总体成功率: %.1f%%\n", successRate*100)
	}

	if avgDuration, ok := metrics["avg_duration"].(string); ok {
		fmt.Printf("⏱️  平均发送耗时: %s\n", avgDuration)
	}

	if maxDuration, ok := metrics["max_duration"].(string); ok {
		fmt.Printf("⏰ 最大发送耗时: %s\n", maxDuration)
	}

	// 显示飞书平台特定统计
	if sendsByPlatform, ok := metrics["sends_by_platform"].(map[string]int64); ok {
		if feishuSends, exists := sendsByPlatform["feishu"]; exists {
			fmt.Printf("📱 飞书平台发送数: %d\n", feishuSends)
		}
	}

	if failsByPlatform, ok := metrics["fails_by_platform"].(map[string]int64); ok {
		if feishuFails, exists := failsByPlatform["feishu"]; exists {
			fmt.Printf("⚠️  飞书平台失败数: %d\n", feishuFails)
		}
	}
}

// 辅助结构和函数

type messageData struct {
	title   string
	content string
	group   string
}

type batchResult struct {
	Index    int
	Results  []*notifiers.SendResult
	Duration time.Duration
	Error    error
}

func countSuccessResults(results []*notifiers.SendResult) int {
	count := 0
	for _, result := range results {
		if result.Success {
			count++
		}
	}
	return count
}

func getTeamName(teamID string) string {
	names := map[string]string{
		"backend-team":  "后端团队",
		"frontend-team": "前端团队",
		"ops-team":      "运维团队",
		"qa-team":       "测试团队",
	}
	if name, exists := names[teamID]; exists {
		return name
	}
	return teamID
}

func getPriorityName(priority int) string {
	switch priority {
	case 1:
		return "低"
	case 2:
		return "较低"
	case 3:
		return "普通"
	case 4:
		return "高"
	case 5:
		return "紧急"
	default:
		return "未知"
	}
}
