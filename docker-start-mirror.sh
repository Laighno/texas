#!/bin/bash

# 使用国内镜像源的 Docker 启动脚本

echo "=========================================="
echo "  德州扑克服务器 Docker 启动脚本（国内镜像源版）"
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
echo "📦 使用国内镜像源构建（适合网络受限环境）"
echo ""

# 使用国内镜像源版本的 docker-compose 文件
if [ ! -f "docker-compose.mirror.yml" ]; then
    echo "❌ 错误: 未找到 docker-compose.mirror.yml 文件"
    exit 1
fi

# 构建并启动容器
echo "📦 正在构建 Docker 镜像（使用国内镜像源）..."
$DOCKER_COMPOSE -f docker-compose.mirror.yml build

if [ $? -ne 0 ]; then
    echo "❌ 构建失败，请检查错误信息"
    echo ""
    echo "💡 提示：如果仍然失败，可以尝试："
    echo "   1. 配置 Docker 镜像加速器: sudo ./setup-docker-mirror.sh"
    echo "   2. 检查网络连接"
    exit 1
fi

echo ""
echo "🚀 正在启动服务器..."
$DOCKER_COMPOSE -f docker-compose.mirror.yml up -d

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
echo "   查看日志: $DOCKER_COMPOSE -f docker-compose.mirror.yml logs -f"
echo "   停止服务: $DOCKER_COMPOSE -f docker-compose.mirror.yml down"
echo "   重启服务: $DOCKER_COMPOSE -f docker-compose.mirror.yml restart"
echo ""
echo "=========================================="
