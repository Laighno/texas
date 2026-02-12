#!/bin/bash

# 测试打平情况的脚本

echo "=== 测试德州扑克打平情况 ==="

# 编译测试程序
echo "编译测试程序..."
go build -tags test -o test_tie_breaker test_tie_breaker.go main.go game.go 2>&1
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"

# 启动服务器（后台运行）
echo "启动服务器..."
./test_tie_breaker &
SERVER_PID=$!

# 等待服务器启动
sleep 2

# 运行测试
echo "运行测试..."
./test_tie_breaker

# 等待测试完成
sleep 5

# 停止服务器
echo "停止服务器..."
kill $SERVER_PID 2>/dev/null

echo "=== 测试完成 ==="
