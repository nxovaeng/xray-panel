#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo "=================================="
echo "  Xray Panel 编译脚本"
echo "=================================="
echo ""

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    print_error "Go 未安装"
    exit 1
fi

print_info "Go 版本: $(go version)"

# 清理缓存
print_info "清理构建缓存..."
go clean -cache -modcache -i -r

# 删除旧文件
print_info "删除旧的二进制文件..."
rm -f xray-panel

# 下载依赖
print_info "下载依赖..."
go mod download
go mod tidy

# 获取版本信息
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.2.2")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

print_info "版本: $VERSION"
print_info "构建时间: $BUILD_TIME"
print_info "Git Commit: $GIT_COMMIT"

# 编译
print_info "开始编译..."
go build -ldflags "\
    -X main.Version=${VERSION} \
    -X main.BuildTime=${BUILD_TIME} \
    -X main.GitCommit=${GIT_COMMIT} \
    -s -w" \
    -o xray-panel \
    cmd/panel/main.go

# 验证
if [ -f xray-panel ]; then
    print_info "✓ 编译成功"
    ls -lh xray-panel
    echo ""
    print_info "二进制文件: ./xray-panel"
else
    print_error "编译失败"
    exit 1
fi

echo ""
print_info "编译完成！"
echo ""
echo "运行方式:"
echo "  开发: ./xray-panel"
echo "  安装: sudo cp xray-panel /usr/local/bin/"
echo ""
