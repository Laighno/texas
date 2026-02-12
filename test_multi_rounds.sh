#!/bin/bash

# 多轮游戏测试脚本（包含新玩家加入和退出）

echo "=== 多轮游戏测试（包含新玩家加入和退出） ==="
echo ""

# 检查服务器是否在运行
if ! pgrep -f "go run main.go" > /dev/null && ! pgrep -f "poker-server" > /dev/null; then
    echo "启动服务器..."
    cd /home/laighno/go/src/awesomeProject
    timeout 300 bash start.sh > /tmp/poker_multi_rounds.log 2>&1 &
    SERVER_PID=$!
    sleep 3
    echo "服务器已启动 (PID: $SERVER_PID)"
else
    echo "✅ 服务器正在运行"
fi

echo ""
echo "测试场景："
echo "1. 第一轮：4个玩家正常游戏"
echo "2. 第二轮：添加新玩家，新玩家自动加入"
echo "3. 第三轮：玩家退出，新玩家加入"
echo "4. 第四轮：再次添加新玩家"
echo ""

# 运行测试
cd /home/laighno/go/src/awesomeProject
timeout 180 go run -tags test test_multi_rounds.go test_multi_rounds_runner.go 2>&1

echo ""
echo "=== 测试完成 ==="

# 清理
if [ ! -z "$SERVER_PID" ]; then
    kill $SERVER_PID 2>/dev/null
fi
