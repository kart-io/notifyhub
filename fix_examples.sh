#!/bin/bash

# 批量修复所有示例文件的脚本

examples_dir="/Users/costalong/code/go/src/github.com/kart/notifyhub/examples"

# 遍历所有示例目录
for dir in $(find $examples_dir -name "*.go" -type f | xargs dirname | sort | uniq); do
  echo "Processing directory: $dir"

  # 检查每个main.go文件
  if [ -f "$dir/main.go" ]; then
    echo "Fixing $dir/main.go"

    # 执行替换操作
    sed -i '' 's|"github.com/kart-io/notifyhub"|"github.com/kart-io/notifyhub/client"\
	"github.com/kart-io/notifyhub/config"\
	"github.com/kart-io/notifyhub/notifiers"|g' "$dir/main.go"

    # 替换常见的函数调用
    sed -i '' 's|notifyhub\.New|client.New|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.NewWithDefaults|client.New(config.WithTestDefaults)|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.WithFeishu|config.WithFeishu|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.WithEmail|config.WithEmail|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.WithQueue|config.WithQueue|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.WithDefaults|config.WithDefaults|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.WithTestDefaults|config.WithTestDefaults|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.NewAlert|client.NewAlert|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.NewNotice|client.NewNotice|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.NewReport|client.NewReport|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.Target|notifiers.Target|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.TargetType|notifiers.TargetType|g' "$dir/main.go"

    # 修复选项相关
    sed -i '' 's|notifyhub\.NewRetryOptions|client.Options|g' "$dir/main.go"
    sed -i '' 's|notifyhub\.NewAsyncOptions|client.Options|g' "$dir/main.go"

    echo "Fixed $dir/main.go"
  fi
done

echo "All examples fixed!"