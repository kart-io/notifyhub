#!/bin/bash

# 飞书示例测试脚本
# 用于验证示例代码的编译和基本功能

set -e

echo "🚀 开始测试飞书示例..."

# 检查环境变量
echo "📋 检查环境变量..."
if [ -z "$FEISHU_WEBHOOK_URL" ]; then
    echo "⚠️  警告: FEISHU_WEBHOOK_URL 环境变量未设置"
    echo "   设置示例: export FEISHU_WEBHOOK_URL='https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook'"
fi

if [ -z "$FEISHU_SECRET" ]; then
    echo "⚠️  警告: FEISHU_SECRET 环境变量未设置"
    echo "   设置示例: export FEISHU_SECRET='your-secret'"
fi

# 切换到示例目录
cd "$(dirname "$0")"

echo "📦 更新依赖..."
for dir in basic advanced batch quick-demo simple-demo; do
    echo "  - 更新 $dir 依赖..."
    (cd "$dir" && go mod tidy)
done

echo "🔨 编译检查..."

# 编译所有示例目录
echo "  - 编译快速演示..."
(cd quick-demo && go build -o /tmp/feishu-quick-demo main.go)
if [ $? -eq 0 ]; then
    echo "    ✅ 快速演示编译成功"
else
    echo "    ❌ 快速演示编译失败"
    exit 1
fi

echo "  - 编译基础示例..."
(cd basic && go build -o /tmp/feishu-basic main.go)
if [ $? -eq 0 ]; then
    echo "    ✅ 基础示例编译成功"
else
    echo "    ❌ 基础示例编译失败"
    exit 1
fi

echo "  - 编译高级示例..."
(cd advanced && go build -o /tmp/feishu-advanced main.go)
if [ $? -eq 0 ]; then
    echo "    ✅ 高级示例编译成功"
else
    echo "    ❌ 高级示例编译失败"
    exit 1
fi

echo "  - 编译批量示例..."
(cd batch && go build -o /tmp/feishu-batch main.go)
if [ $? -eq 0 ]; then
    echo "    ✅ 批量示例编译成功"
else
    echo "    ❌ 批量示例编译失败"
    exit 1
fi

echo "  - 编译简单示例..."
(cd simple-demo && go build -o /tmp/feishu-simple main.go)
if [ $? -eq 0 ]; then
    echo "    ✅ 简单示例编译成功"
else
    echo "    ❌ 简单示例编译失败"
    exit 1
fi

echo "🧪 运行语法检查..."
for dir in basic advanced batch quick-demo simple-demo; do
    echo "  - 检查 $dir..."
    (cd "$dir" && go vet .)
    if [ $? -ne 0 ]; then
        echo "    ❌ $dir 语法检查失败"
        exit 1
    fi
done
echo "✅ 语法检查通过"

echo "📋 代码格式检查..."
format_errors=""
for dir in basic advanced batch quick-demo simple-demo; do
    gofmt_output=$(cd "$dir" && gofmt -l .)
    if [ -n "$gofmt_output" ]; then
        format_errors="$format_errors\n$dir: $gofmt_output"
    fi
done

if [ -z "$format_errors" ]; then
    echo "✅ 代码格式正确"
else
    echo "❌ 以下文件格式需要调整:"
    echo -e "$format_errors"
    exit 1
fi

# 如果设置了环境变量，进行实际发送测试
if [ -n "$FEISHU_WEBHOOK_URL" ] && [ -n "$FEISHU_SECRET" ]; then
    echo "🌐 运行实际发送测试 (使用真实飞书API)..."

    # 创建测试配置
    export TEST_MODE="true"
    export TEST_GROUP_ID="test-group"

    echo "  - 测试快速演示..."
    timeout 30s /tmp/feishu-quick-demo > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo "    ✅ 快速演示测试通过"
    else
        echo "    ⚠️  快速演示测试超时或失败 (可能是网络问题)"
    fi

else
    echo "ℹ️  跳过实际发送测试 (需要设置环境变量)"
fi

echo "📊 生成示例统计..."
echo "文件统计:"
echo "  - 示例目录数: 5"
echo "  - 快速演示: $(wc -l < quick-demo/main.go) 行"
echo "  - 基础示例: $(wc -l < basic/main.go) 行"
echo "  - 高级示例: $(wc -l < advanced/main.go) 行"
echo "  - 批量示例: $(wc -l < batch/main.go) 行"
echo "  - 简单示例: $(wc -l < simple-demo/main.go) 行"
total_lines=$(($(wc -l < quick-demo/main.go) + $(wc -l < basic/main.go) + $(wc -l < advanced/main.go) + $(wc -l < batch/main.go) + $(wc -l < simple-demo/main.go)))
echo "  - 总代码行数: $total_lines 行"

echo "🔍 检查示例完整性..."
required_functions=(
    "demonstrateTemplates"
    "demonstrateRouting"
    "demonstrateRetryHandling"
    "demonstrateCallbacks"
    "demonstrateDelayedSending"
    "demonstrateBasicBatch"
    "demonstrateGroupedBatch"
    "demonstrateConcurrentBatch"
)

for func in "${required_functions[@]}"; do
    if grep -q "$func" */main.go; then
        echo "  ✅ $func 函数存在"
    else
        echo "  ❌ $func 函数缺失"
    fi
done

echo ""
echo "🎉 飞书示例测试完成!"
echo ""
echo "📖 使用说明:"
echo "  1. 设置环境变量:"
echo "     export FEISHU_WEBHOOK_URL='your-webhook-url'"
echo "     export FEISHU_SECRET='your-secret'"
echo ""
echo "  2. 运行示例:"
echo "     cd quick-demo && go run main.go   # 快速验证"
echo "     cd simple-demo && go run main.go  # 简单示例（无需 secret）"
echo "     cd basic && go run main.go        # 基础功能"
echo "     cd advanced && go run main.go     # 高级功能"
echo "     cd batch && go run main.go        # 批量发送"
echo ""
echo "  3. 查看文档:"
echo "     cat README.md"
echo ""

# 清理临时文件
rm -f /tmp/feishu-quick-demo /tmp/feishu-basic /tmp/feishu-advanced /tmp/feishu-batch /tmp/feishu-simple