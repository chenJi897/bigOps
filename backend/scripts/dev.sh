#!/bin/bash
# BigOps 后端常用操作脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# 强制将 Go 构建缓存和临时目录落到仓库所在的 /data 下，避免写入 /tmp。
source "$SCRIPT_DIR/go-env.sh"

CMD=${1:-help}

case "$CMD" in
  build)
    echo "编译中..."
    go build -o bin/bigops-core ./cmd/core
    echo "编译完成: bin/bigops-core"
    ;;
  run)
    go run ./cmd/core/main.go
    ;;
  test)
    go test -v ./...
    ;;
  clean)
    rm -rf bin/ coverage.out coverage.html
    echo "清理完成"
    ;;
  help|*)
    echo "用法: ./scripts/dev.sh [命令]"
    echo ""
    echo "  build   编译应用"
    echo "  run     运行应用"
    echo "  test    运行全部测试"
    echo "  clean   清理构建产物"
    ;;
esac
