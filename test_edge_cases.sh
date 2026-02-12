#!/bin/bash

# 边缘情况测试脚本

echo "=== 德州扑克边缘情况测试 ==="
echo ""

# 检查服务器是否在运行
if ! pgrep -f "go run main.go" > /dev/null && ! pgrep -f "poker-server" > /dev/null; then
    echo "启动服务器..."
    cd /home/laighno/go/src/awesomeProject
    timeout 30 bash start.sh > /tmp/poker_test.log 2>&1 &
    SERVER_PID=$!
    sleep 3
    echo "服务器已启动 (PID: $SERVER_PID)"
else
    echo "✅ 服务器正在运行"
fi

echo ""
echo "测试场景："
echo "1. 所有玩家全押"
echo "2. 只剩一个玩家"
echo "3. 玩家超时"
echo "4. 多局游戏"
echo ""

# 运行测试
cd /home/laighno/go/src/awesomeProject
timeout 60 go run -tags test test_game.go test_runner.go 2>&1

echo ""
echo "=== 测试完成 ==="

# 清理
if [ ! -z "$SERVER_PID" ]; then
    kill $SERVER_PID 2>/dev/null
fi
