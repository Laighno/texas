#!/bin/bash

# 德州扑克 Docker 启动脚本

echo "=========================================="
echo "  德州扑克服务器 Docker 启动脚本"
echo "=========================================="
echo ""

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ 错误: 未找到 Docker，请先安装 Docker"
    echo "   安装指南: https://docs.docker.com/get-docker/"
    exit 1
fi

# 检查 Docker Compose 是否安装
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ 错误: 未找到 Docker Compose，请先安装 Docker Compose"
    echo "   安装指南: https://docs.docker.com/compose/install/"
    exit 1
fi

# 使用 docker compose（新版本）或 docker-compose（旧版本）
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

echo "✅ Docker 环境检查通过"
echo ""

# 构建并启动容器
echo "📦 正在构建 Docker 镜像..."
$DOCKER_COMPOSE build

if [ $? -ne 0 ]; then
    echo "❌ 构建失败，请检查错误信息"
    exit 1
fi

echo ""
echo "🚀 正在启动服务器..."
$DOCKER_COMPOSE up -d

if [ $? -ne 0 ]; then
    echo "❌ 启动失败，请检查错误信息"
    exit 1
fi

echo ""
echo "=========================================="
echo "✅ 服务器启动成功！"
echo "=========================================="
echo ""
echo "🌐 访问地址: http://localhost:8080"
echo ""
echo "📋 常用命令:"
echo "   查看日志: $DOCKER_COMPOSE logs -f"
echo "   停止服务: $DOCKER_COMPOSE down"
echo "   重启服务: $DOCKER_COMPOSE restart"
echo ""
echo "=========================================="
