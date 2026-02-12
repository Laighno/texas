#!/bin/bash

# 加载打包的 alpine 镜像

echo "=========================================="
echo "  加载 Alpine 镜像"
echo "=========================================="
echo ""

if [ ! -f "alpine-latest.tar" ]; then
    echo "❌ 错误: 未找到 alpine-latest.tar 文件"
    exit 1
fi

echo "📦 正在加载 alpine:latest 镜像..."
docker load -i alpine-latest.tar

if [ $? -eq 0 ]; then
    echo "✅ Alpine 镜像加载成功！"
    echo ""
    echo "验证镜像："
    docker images | grep alpine
else
    echo "❌ 镜像加载失败"
    exit 1
fi
