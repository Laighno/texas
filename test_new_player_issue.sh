#!/bin/bash

# 测试新玩家加入后游戏无法开始的问题

echo "=== 测试新玩家加入后游戏无法开始的问题 ==="
echo "场景：4人开局 -> 第5人进入 -> 完成一局 -> 第二局无法开始"
echo ""

# 检查服务器是否在运行
if ! pgrep -f "go run main.go" > /dev/null && ! pgrep -f "poker-server" > /dev/null; then
    echo "启动服务器..."
    cd /home/laighno/go/src/awesomeProject
    timeout 180 bash start.sh > /tmp/poker_new_player_test.log 2>&1 &
    SERVER_PID=$!
    sleep 3
    echo "服务器已启动 (PID: $SERVER_PID)"
else
    echo "✅ 服务器正在运行"
fi

echo ""
echo "开始测试..."

# 运行测试
cd /home/laighno/go/src/awesomeProject
timeout 120 go run -tags test test_new_player_issue.go test_new_player_issue_runner.go 2>&1

echo ""
echo "=== 测试完成 ==="

# 清理
if [ ! -z "$SERVER_PID" ]; then
    kill $SERVER_PID 2>/dev/null
fi
