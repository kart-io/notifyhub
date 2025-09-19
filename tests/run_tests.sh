#!/bin/bash

# NotifyHub 测试运行脚本
# 用于运行各种测试套件

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试根目录
TEST_ROOT="tests"

# 帮助信息
usage() {
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  unit          运行单元测试"
    echo "  integration   运行集成测试"
    echo "  performance   运行性能测试"
    echo "  benchmark     运行基准测试"
    echo "  all           运行所有测试"
    echo "  coverage      运行测试并生成覆盖率报告"
    echo "  clean         清理测试缓存和临时文件"
    echo "  help          显示此帮助信息"
    echo ""
    echo "Options:"
    echo "  -v, --verbose     显示详细输出"
    echo "  -f, --filter      过滤测试用例 (e.g., -f TestHub)"
    echo "  -t, --timeout     设置测试超时时间 (默认: 10m)"
    echo "  -p, --parallel    并行运行测试"
    echo ""
    echo "Examples:"
    echo "  $0 unit                    # 运行所有单元测试"
    echo "  $0 unit -v                 # 运行单元测试并显示详细信息"
    echo "  $0 integration -f TestE2E  # 运行特定的集成测试"
    echo "  $0 coverage                # 生成测试覆盖率报告"
    echo "  $0 benchmark -f Queue      # 运行队列相关的基准测试"
}

# 打印带颜色的信息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 运行单元测试
run_unit_tests() {
    print_info "运行单元测试..."

    local verbose=""
    local filter=""
    local timeout="10m"

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                verbose="-v"
                shift
                ;;
            -f|--filter)
                filter="-run $2"
                shift 2
                ;;
            -t|--timeout)
                timeout="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    go test $verbose $filter -timeout=$timeout ./${TEST_ROOT}/unit/... || {
        print_error "单元测试失败"
        exit 1
    }

    print_success "单元测试完成"
}

# 运行集成测试
run_integration_tests() {
    print_info "运行集成测试..."

    local verbose=""
    local filter=""
    local timeout="15m"

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                verbose="-v"
                shift
                ;;
            -f|--filter)
                filter="-run $2"
                shift 2
                ;;
            -t|--timeout)
                timeout="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    go test $verbose $filter -timeout=$timeout ./${TEST_ROOT}/integration/... || {
        print_error "集成测试失败"
        exit 1
    }

    print_success "集成测试完成"
}

# 运行性能测试
run_performance_tests() {
    print_info "运行性能测试..."

    local verbose=""
    local filter=""
    local timeout="30m"

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                verbose="-v"
                shift
                ;;
            -f|--filter)
                filter="-run $2"
                shift 2
                ;;
            -t|--timeout)
                timeout="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    go test $verbose $filter -timeout=$timeout ./${TEST_ROOT}/performance/... || {
        print_error "性能测试失败"
        exit 1
    }

    print_success "性能测试完成"
}

# 运行基准测试
run_benchmark_tests() {
    print_info "运行基准测试..."

    local filter=""
    local count="1"
    local time="10s"

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -f|--filter)
                filter="$2"
                shift 2
                ;;
            -c|--count)
                count="$2"
                shift 2
                ;;
            -t|--time)
                time="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    if [ -n "$filter" ]; then
        go test -bench="$filter" -benchmem -benchtime=$time -count=$count ./${TEST_ROOT}/performance/... || {
            print_error "基准测试失败"
            exit 1
        }
    else
        go test -bench=. -benchmem -benchtime=$time -count=$count ./${TEST_ROOT}/performance/... || {
            print_error "基准测试失败"
            exit 1
        }
    fi

    print_success "基准测试完成"
}

# 运行所有测试
run_all_tests() {
    print_info "运行所有测试套件..."

    print_info "1/3 运行单元测试"
    run_unit_tests "$@"

    print_info "2/3 运行集成测试"
    run_integration_tests "$@"

    print_info "3/3 运行性能测试"
    run_performance_tests "$@"

    print_success "所有测试完成"
}

# 生成测试覆盖率报告
run_coverage() {
    print_info "生成测试覆盖率报告..."

    # 创建覆盖率目录
    mkdir -p coverage

    # 运行测试并生成覆盖率数据
    go test -coverprofile=coverage/unit.out ./${TEST_ROOT}/unit/... || {
        print_error "单元测试覆盖率生成失败"
        exit 1
    }

    go test -coverprofile=coverage/integration.out ./${TEST_ROOT}/integration/... || {
        print_error "集成测试覆盖率生成失败"
        exit 1
    }

    # 合并覆盖率文件
    echo "mode: set" > coverage/coverage.out
    tail -q -n +2 coverage/*.out >> coverage/coverage.out

    # 生成HTML报告
    go tool cover -html=coverage/coverage.out -o coverage/coverage.html

    # 显示覆盖率摘要
    go tool cover -func=coverage/coverage.out

    print_success "覆盖率报告已生成: coverage/coverage.html"

    # 如果可能，自动打开报告
    if command -v open &> /dev/null; then
        open coverage/coverage.html
    elif command -v xdg-open &> /dev/null; then
        xdg-open coverage/coverage.html
    fi
}

# 清理测试缓存
clean_test_cache() {
    print_info "清理测试缓存和临时文件..."

    # 清理Go测试缓存
    go clean -testcache

    # 清理覆盖率文件
    rm -rf coverage/

    # 清理临时文件
    find . -name "*.test" -type f -delete
    find . -name "*.out" -type f -delete

    print_success "清理完成"
}

# 检查依赖
check_dependencies() {
    # 检查Go是否安装
    if ! command -v go &> /dev/null; then
        print_error "Go未安装，请先安装Go"
        exit 1
    fi

    # 检查Go版本
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    required_version="1.25"

    if [ "$(printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1)" != "$required_version" ]; then
        print_warning "Go版本过低，建议升级到 $required_version 或更高版本"
    fi
}

# 主函数
main() {
    # 检查依赖
    check_dependencies

    # 如果没有参数，显示帮助
    if [ $# -eq 0 ]; then
        usage
        exit 0
    fi

    # 解析命令
    command=$1
    shift

    case $command in
        unit)
            run_unit_tests "$@"
            ;;
        integration)
            run_integration_tests "$@"
            ;;
        performance)
            run_performance_tests "$@"
            ;;
        benchmark)
            run_benchmark_tests "$@"
            ;;
        all)
            run_all_tests "$@"
            ;;
        coverage)
            run_coverage "$@"
            ;;
        clean)
            clean_test_cache
            ;;
        help)
            usage
            ;;
        *)
            print_error "未知命令: $command"
            usage
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"