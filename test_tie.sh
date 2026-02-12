#!/bin/bash

# 测试打平逻辑的脚本

echo "=== 测试德州扑克打平逻辑 ==="

# 编译测试程序（使用build tag排除main.go的main函数）
echo "编译测试程序..."
go build -tags tie_test -o test_tie_standalone test_tie_standalone.go main.go game.go 2>&1
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"

# 运行测试
echo "运行测试..."
./test_tie_standalone

echo ""
echo "=== 测试完成 ==="
