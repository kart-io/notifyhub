# 飞书通知示例 Makefile

.PHONY: help build-all build-example build-debug build-test run example debug test demo deps clean fmt vet check

# 默认目标
help:
	@echo "飞书通知示例 - 可用命令:"
	@echo "  make build-all    - 构建所有程序"
	@echo "  make build-example- 构建示例程序"
	@echo "  make build-debug  - 构建调试工具"
	@echo "  make build-test   - 构建测试工具"
	@echo ""
	@echo "  make example      - 运行示例程序（需要设置环境变量）"
	@echo "  make debug        - 运行调试工具（直接HTTP请求）"
	@echo "  make test         - 运行真实环境测试"
	@echo "  make demo         - 运行演示（使用模拟数据）"
	@echo ""
	@echo "  make deps         - 下载所有依赖"
	@echo "  make clean        - 清理构建文件"
	@echo "  make fmt          - 格式化代码"
	@echo "  make vet          - 代码检查"
	@echo "  make check        - 完整检查"
	@echo "  make help         - 显示此帮助信息"
	@echo ""
	@echo "环境变量配置:"
	@echo "  FEISHU_WEBHOOK_URL  - 飞书 Webhook URL（必需）"
	@echo "  FEISHU_SECRET       - 飞书签名密钥（可选）"
	@echo ""
	@echo "项目结构:"
	@echo "  cmd/example/      - 主要示例程序（6种场景演示）"
	@echo "  cmd/debug/        - 调试工具（直接HTTP测试）"
	@echo "  cmd/test/         - 测试工具（NotifyHub集成测试）"

# 构建所有程序
build-all: build-example build-debug build-test

# 构建示例程序
build-example:
	@echo "🔨 构建示例程序..."
	cd ../cmd/example && go build -o feishu-example main.go
	@echo "✅ 构建完成: cmd/example/feishu-example"

# 构建调试工具
build-debug:
	@echo "🔨 构建调试工具..."
	cd ../cmd/debug && go build -o debug-sender main.go
	@echo "✅ 构建完成: cmd/debug/debug-sender"

# 构建测试工具
build-test:
	@echo "🔨 构建测试工具..."
	cd ../cmd/test && go build -o test-real main.go
	@echo "✅ 构建完成: cmd/test/test-real"

# 运行示例程序（需要环境变量）
example:
	@echo "🚀 运行飞书通知示例..."
	@if [ -z "$(FEISHU_WEBHOOK_URL)" ]; then \
		echo "❌ 错误: 请设置环境变量 FEISHU_WEBHOOK_URL"; \
		echo "   export FEISHU_WEBHOOK_URL=\"https://open.feishu.cn/open-apis/bot/v2/hook/your-token\""; \
		exit 1; \
	fi
	cd ../cmd/example && go run main.go

# 运行调试工具（直接 HTTP 请求）
debug:
	@echo "🧪 运行飞书调试工具..."
	@if [ -z "$(FEISHU_WEBHOOK_URL)" ]; then \
		echo "❌ 错误: 请设置环境变量 FEISHU_WEBHOOK_URL"; \
		echo "   export FEISHU_WEBHOOK_URL=\"https://open.feishu.cn/open-apis/bot/v2/hook/your-token\""; \
		exit 1; \
	fi
	cd ../cmd/debug && go run main.go

# 运行真实环境测试（NotifyHub 集成）
test:
	@echo "🧪 运行 NotifyHub 真实环境测试..."
	@if [ -z "$(FEISHU_WEBHOOK_URL)" ]; then \
		echo "❌ 错误: 请设置环境变量 FEISHU_WEBHOOK_URL"; \
		echo "   export FEISHU_WEBHOOK_URL=\"https://open.feishu.cn/open-apis/bot/v2/hook/your-token\""; \
		exit 1; \
	fi
	cd ../cmd/test && go run main.go

# 运行演示（使用模拟数据）
demo:
	@echo "🎭 运行飞书通知演示..."
	./demo.sh

# 下载依赖
deps:
	@echo "📦 下载依赖..."
	cd ../cmd/example && go mod tidy && go mod download
	cd ../cmd/debug && go mod tidy && go mod download
	cd ../cmd/test && go mod tidy && go mod download
	@echo "✅ 依赖下载完成"

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	rm -f ../cmd/example/feishu-example
	rm -f ../cmd/debug/debug-sender
	rm -f ../cmd/test/test-real
	cd ../cmd/example && go clean
	cd ../cmd/debug && go clean
	cd ../cmd/test && go clean
	@echo "✅ 清理完成"

# 验证代码
vet:
	@echo "🔍 代码检查..."
	cd ../cmd/example && go vet ./...
	cd ../cmd/debug && go vet ./...
	cd ../cmd/test && go vet ./...
	@echo "✅ 代码检查通过"

# 格式化代码
fmt:
	@echo "📝 格式化代码..."
	cd ../cmd/example && go fmt ./...
	cd ../cmd/debug && go fmt ./...
	cd ../cmd/test && go fmt ./...
	@echo "✅ 代码格式化完成"

# 完整检查
check: fmt vet build-all
	@echo "✅ 所有检查通过"

# 兼容性别名
run: example
build: build-example
test-real: test