#!/bin/bash

# 强制修复 Docker 镜像加速器配置

echo "=========================================="
echo "  Docker 镜像加速器强制配置"
echo "=========================================="
echo ""

# 检查是否有 root 权限
if [ "$EUID" -ne 0 ]; then 
    echo "⚠️  需要 root 权限"
    echo "   请运行: sudo ./fix-docker-mirror.sh"
    exit 1
fi

DOCKER_DAEMON_JSON="/etc/docker/daemon.json"
DOCKER_DAEMON_JSON_DIR="/etc/docker"

# 创建配置目录
mkdir -p "$DOCKER_DAEMON_JSON_DIR"

# 备份现有配置
if [ -f "$DOCKER_DAEMON_JSON" ]; then
    cp "$DOCKER_DAEMON_JSON" "${DOCKER_DAEMON_JSON}.backup.$(date +%Y%m%d_%H%M%S)"
    echo "✅ 已备份现有配置"
fi

# 创建/更新配置文件
cat > "$DOCKER_DAEMON_JSON" << 'EOF'
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com",
    "https://dockerhub.azk8s.cn"
  ]
}
EOF

echo "✅ 配置文件已更新: $DOCKER_DAEMON_JSON"
echo ""
echo "📋 配置的镜像加速器："
echo "   - https://docker.mirrors.ustc.edu.cn (中科大)"
echo "   - https://hub-mirror.c.163.com (网易)"
echo "   - https://mirror.baidubce.com (百度云)"
echo "   - https://dockerhub.azk8s.cn (Azure)"
echo ""

# 重启 Docker
echo "🔄 正在重启 Docker 服务..."
systemctl daemon-reload
systemctl restart docker

if [ $? -eq 0 ]; then
    echo "✅ Docker 服务重启成功"
    echo ""
    sleep 2
    
    # 验证配置
    echo "🔍 验证配置..."
    if docker info 2>/dev/null | grep -q "Registry Mirrors"; then
        echo "✅ 镜像加速器配置成功！"
        docker info | grep -A 10 "Registry Mirrors"
    else
        echo "⚠️  配置可能未生效，请检查 Docker 服务状态"
    fi
    
    echo ""
    echo "🧪 测试拉取镜像..."
    echo "   正在拉取 alpine:latest（测试镜像加速器）..."
    if docker pull alpine:latest 2>&1 | head -5; then
        echo "✅ 镜像拉取成功，镜像加速器工作正常！"
    else
        echo "⚠️  镜像拉取可能仍有问题，请检查网络连接"
    fi
else
    echo "❌ Docker 服务重启失败"
    echo "   请手动运行: sudo systemctl restart docker"
fi

echo ""
echo "=========================================="
echo "✅ 配置完成！"
echo "=========================================="
echo ""
echo "现在可以重新运行构建命令："
echo "   docker-compose up -d --build"
echo "   或"
echo "   ./docker-build-local.sh"
echo ""
