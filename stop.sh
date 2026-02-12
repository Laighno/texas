#!/bin/bash

echo "正在关闭所有服务..."

# 关闭Go进程
pkill -f "go run main.go" 2>/dev/null
pkill -f "poker-server" 2>/dev/null
pkill -f "main.go" 2>/dev/null

# 关闭占用8080端口的进程
lsof -ti:8080 2>/dev/null | xargs kill -9 2>/dev/null

sleep 1

# 检查是否还有进程
if ps aux | grep -E "[g]o run|[p]oker-server" > /dev/null; then
    echo "仍有进程在运行，强制关闭..."
    pkill -9 -f "go run"
    pkill -9 -f "main.go"
fi

# 检查端口
if netstat -tlnp 2>/dev/null | grep 8080 > /dev/null || ss -tlnp 2>/dev/null | grep 8080 > /dev/null; then
    echo "警告: 端口8080仍被占用"
else
    echo "✅ 所有服务已关闭"
fi
