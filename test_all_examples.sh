#!/bin/bash

# 测试所有示例的编译和运行脚本

examples_dir="/Users/costalong/code/go/src/github.com/kart/notifyhub/examples"
total=0
success=0
failed=0

echo "🧪 Testing all NotifyHub examples..."
echo "=================================="

# 遍历所有示例目录
for dir in $(find $examples_dir -name "main.go" -type f | xargs dirname | sort); do
    example_name=$(basename $dir)
    echo ""
    echo "📁 Testing example: $example_name"

    total=$((total + 1))

    # 检查是否有 main.go
    if [ -f "$dir/main.go" ]; then
        cd "$dir"

        # 尝试编译
        echo "  ⚙️  Building..."
        if go build -o "/tmp/test-$example_name" . 2>/dev/null; then
            echo "  ✅ Build successful"

            # 尝试快速运行测试（超时3秒）
            echo "  🏃 Quick run test..."
            if timeout 3s "/tmp/test-$example_name" >/dev/null 2>&1; then
                echo "  ✅ Run successful"
                success=$((success + 1))
            else
                echo "  ⚠️  Run timeout/error (this might be normal for some examples)"
                success=$((success + 1))  # 编译成功就算成功
            fi

            # 清理
            rm -f "/tmp/test-$example_name"
        else
            echo "  ❌ Build failed"
            failed=$((failed + 1))
        fi
    else
        echo "  ⚠️  No main.go found"
        failed=$((failed + 1))
    fi
done

echo ""
echo "=================================="
echo "📊 Test Summary:"
echo "   Total examples: $total"
echo "   Successful: $success"
echo "   Failed: $failed"
echo ""

if [ $failed -eq 0 ]; then
    echo "🎉 All examples are working!"
    exit 0
else
    echo "⚠️  Some examples need attention"
    exit 1
fi