#!/bin/bash

echo "启动德州扑克服务器..."
echo "服务器将在 http://localhost:8080 启动"
echo "按 Ctrl+C 停止服务器"
echo ""

# 设置Go代理（如果网络有问题可以尝试 direct）
export GOPROXY=${GOPROXY:-direct}

# 确保依赖已下载
echo "检查依赖..."
go mod download 2>/dev/null || true
go mod tidy 2>/dev/null || true

echo "启动服务器..."
go run main.go game.go
