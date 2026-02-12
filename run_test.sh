#!/bin/bash

# 测试脚本：测试两局游戏流程

echo "=== 德州扑克游戏测试脚本 ==="
echo ""

# 检查服务器是否在运行
if ! pgrep -f "poker-server" > /dev/null; then
    echo "❌ 服务器未运行，请先启动服务器: ./start.sh"
    exit 1
fi

echo "✅ 服务器正在运行"
echo ""

# 等待服务器完全启动
sleep 1

# 运行测试（使用build tag test）
echo "开始运行测试..."
echo ""

go run -tags test test_game.go test_runner.go

echo ""
echo "=== 测试完成 ==="
