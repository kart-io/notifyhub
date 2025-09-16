#!/bin/bash

# 更完整的示例修复脚本

examples_dir="/Users/costalong/code/go/src/github.com/kart/notifyhub/examples"

echo "🔧 Fixing all NotifyHub examples (v2)..."

# 遍历所有示例目录
for dir in $(find $examples_dir -name "main.go" -type f | xargs dirname | sort); do
    example_name=$(basename $dir)
    main_file="$dir/main.go"

    echo "Processing: $example_name"

    if [ -f "$main_file" ]; then
        # 创建备份
        cp "$main_file" "$main_file.backup"

        # 第一步：修复导入
        sed -i '' 's|"github.com/kart-io/notifyhub"|"github.com/kart-io/notifyhub/client"|g' "$main_file"

        # 添加必要的导入（如果不存在）
        if ! grep -q 'github.com/kart-io/notifyhub/config' "$main_file"; then
            sed -i '' '/github.com\/kart-io\/notifyhub\/client/a\
	"github.com/kart-io/notifyhub/config"' "$main_file"
        fi

        if ! grep -q 'github.com/kart-io/notifyhub/notifiers' "$main_file" && grep -q 'Target\|TargetType' "$main_file"; then
            sed -i '' '/github.com\/kart-io\/notifyhub\/config/a\
	"github.com/kart-io/notifyhub/notifiers"' "$main_file"
        fi

        # 第二步：修复API调用
        sed -i '' 's|notifyhub\.New|client.New|g' "$main_file"
        sed -i '' 's|notifyhub\.NewWithDefaults|client.New(config.WithTestDefaults)|g' "$main_file"
        sed -i '' 's|notifyhub\.WithDefaults()|config.WithDefaults()|g' "$main_file"
        sed -i '' 's|notifyhub\.WithTestDefaults|config.WithTestDefaults|g' "$main_file"
        sed -i '' 's|notifyhub\.WithFeishu|config.WithFeishu|g' "$main_file"
        sed -i '' 's|notifyhub\.WithEmail|config.WithEmail|g' "$main_file"
        sed -i '' 's|notifyhub\.WithQueue|config.WithQueue|g' "$main_file"
        sed -i '' 's|notifyhub\.WithFeishuFromEnv|config.WithFeishuFromEnv|g' "$main_file"
        sed -i '' 's|notifyhub\.WithEmailFromEnv|config.WithEmailFromEnv|g' "$main_file"

        # 修复消息构建器
        sed -i '' 's|notifyhub\.NewAlert|client.NewAlert|g' "$main_file"
        sed -i '' 's|notifyhub\.NewNotice|client.NewNotice|g' "$main_file"
        sed -i '' 's|notifyhub\.NewReport|client.NewReport|g' "$main_file"
        sed -i '' 's|notifyhub\.NewMessage|client.NewMessage|g' "$main_file"

        # 修复Target相关
        sed -i '' 's|notifyhub\.Target|notifiers.Target|g' "$main_file"
        sed -i '' 's|notifyhub\.TargetType|notifiers.TargetType|g' "$main_file"

        # 修复选项
        sed -i '' 's|notifyhub\.NewRetryOptions|(\&client.Options{Retry: true, MaxRetries:|g' "$main_file"
        sed -i '' 's|notifyhub\.NewAsyncOptions|(\&client.Options{Async: true})|g' "$main_file"
        sed -i '' 's|client\.NewAsyncOptions|(\\&client.Options{Async: true})|g' "$main_file"

        # 处理残留的notifyhub引用
        sed -i '' 's|notifyhub\.WithRouting|config.WithRouting|g' "$main_file"
        sed -i '' 's|notifyhub\.WithDefaultRouting|config.WithDefaultRouting|g' "$main_file"
        sed -i '' 's|notifyhub\.NewRoutingRule|config.NewRoutingRule|g' "$main_file"
        sed -i '' 's|client\.NewRoutingRule|config.NewRoutingRule|g' "$main_file"
        sed -i '' 's|notifyhub\.ExponentialBackoffPolicy|queue.ExponentialBackoffPolicy|g' "$main_file"
        sed -i '' 's|notifyhub\.WithQueueRetryPolicy|config.WithQueueRetryPolicy|g' "$main_file"

        # 修复函数参数错误
        sed -i '' 's|\*notifyhub\.Hub|\*client.Hub|g' "$main_file"

        # 清理未使用的导入
        if ! grep -q 'notifiers\.' "$main_file"; then
            sed -i '' '/github.com\/kart-io\/notifyhub\/notifiers/d' "$main_file"
        fi

        echo "  ✅ Fixed $example_name"
    fi
done

echo "🎉 All examples fixed (v2)!"