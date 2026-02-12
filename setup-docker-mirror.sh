#!/bin/bash

# Docker 镜像加速器配置脚本

echo "=========================================="
echo "  Docker 镜像加速器配置"
echo "=========================================="
echo ""

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ 错误: 未找到 Docker，请先安装 Docker"
    exit 1
fi

# Docker 配置文件路径
DOCKER_DAEMON_JSON="/etc/docker/daemon.json"
DOCKER_DAEMON_JSON_DIR="/etc/docker"

# 检查是否有 root 权限
if [ "$EUID" -ne 0 ]; then 
    echo "⚠️  需要 root 权限来配置 Docker 镜像加速器"
    echo ""
    echo "请使用以下命令运行此脚本："
    echo "  sudo ./setup-docker-mirror.sh"
    echo ""
    echo "或者手动配置，编辑文件: $DOCKER_DAEMON_JSON"
    echo ""
    echo "添加以下内容："
    echo '{'
    echo '  "registry-mirrors": ['
    echo '    "https://docker.mirrors.ustc.edu.cn",'
    echo '    "https://hub-mirror.c.163.com",'
    echo '    "https://mirror.baidubce.com"'
    echo '  ]'
    echo '}'
    exit 1
fi

# 创建配置目录（如果不存在）
if [ ! -d "$DOCKER_DAEMON_JSON_DIR" ]; then
    mkdir -p "$DOCKER_DAEMON_JSON_DIR"
fi

# 备份现有配置
if [ -f "$DOCKER_DAEMON_JSON" ]; then
    cp "$DOCKER_DAEMON_JSON" "${DOCKER_DAEMON_JSON}.backup.$(date +%Y%m%d_%H%M%S)"
    echo "✅ 已备份现有配置"
fi

# 镜像加速器列表（国内常用）
MIRRORS=(
    "https://docker.mirrors.ustc.edu.cn"
    "https://hub-mirror.c.163.com"
    "https://mirror.baidubce.com"
    "https://dockerhub.azk8s.cn"
)

# 创建或更新配置文件
if [ -f "$DOCKER_DAEMON_JSON" ]; then
    # 如果文件存在，检查是否已有 registry-mirrors
    if grep -q "registry-mirrors" "$DOCKER_DAEMON_JSON"; then
        echo "⚠️  配置文件已存在 registry-mirrors，请手动检查配置"
        echo "配置文件位置: $DOCKER_DAEMON_JSON"
        exit 0
    else
        # 添加 registry-mirrors 到现有配置
        echo "📝 更新现有配置文件..."
        python3 << 'PYEOF'
import json
import sys

config_file = '/etc/docker/daemon.json'
mirrors = [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
]

try:
    with open(config_file, 'r') as f:
        config = json.load(f)
except:
    config = {}

config['registry-mirrors'] = mirrors

with open(config_file, 'w') as f:
    json.dump(config, f, indent=2, ensure_ascii=False)
PYEOF
    fi
else
    # 创建新配置文件
    echo "📝 创建新配置文件..."
    cat > "$DOCKER_DAEMON_JSON" << EOF
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ]
}
EOF
fi

echo "✅ 配置文件已更新: $DOCKER_DAEMON_JSON"
echo ""
echo "📋 配置的镜像加速器："
for mirror in "${MIRRORS[@]}"; do
    echo "   - $mirror"
done
echo ""
echo "🔄 正在重启 Docker 服务..."
systemctl daemon-reload
systemctl restart docker

if [ $? -eq 0 ]; then
    echo "✅ Docker 服务重启成功"
    echo ""
    echo "验证配置："
    docker info | grep -A 10 "Registry Mirrors"
else
    echo "❌ Docker 服务重启失败，请手动重启："
    echo "   sudo systemctl restart docker"
fi

echo ""
echo "=========================================="
echo "✅ 配置完成！"
echo "=========================================="
echo ""
echo "现在可以重新运行构建命令："
echo "   ./docker-start.sh"
echo "   或"
echo "   docker-compose up -d --build"
echo ""
