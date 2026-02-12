#!/bin/bash

# 直接运行方案（不使用 Docker，最简单）

echo "=========================================="
echo "  德州扑克服务器 - 直接运行"
echo "=========================================="
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到 Go，请先安装 Go 1.21+"
    exit 1
fi

echo "✅ 环境检查通过"
echo ""

# 检查是否已编译
if [ ! -f "poker-server" ] || [ "main.go" -nt "poker-server" ] || [ "game.go" -nt "poker-server" ]; then
    echo "📦 正在编译 Go 程序..."
    go build -o poker-server main.go game.go
    
    if [ $? -ne 0 ]; then
        echo "❌ 编译失败"
        exit 1
    fi
    echo "✅ 编译成功"
else
    echo "✅ 使用已存在的编译文件"
fi

echo ""
echo "🚀 启动服务器..."
echo "   服务器将在 http://localhost:8080 启动"
echo "   按 Ctrl+C 停止服务器"
echo ""

# 检查端口是否被占用
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo "⚠️  警告: 端口 8080 已被占用"
    echo "   正在尝试停止旧进程..."
    pkill -f "poker-server" 2>/dev/null
    sleep 1
fi

# 启动服务器
./poker-server
