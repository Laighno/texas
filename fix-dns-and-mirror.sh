#!/bin/bash

# 修复 DNS 并配置镜像加速器

echo "=========================================="
echo "  DNS 和 Docker 镜像加速器修复"
echo "=========================================="
echo ""

# 检查是否有 root 权限
if [ "$EUID" -ne 0 ]; then 
    echo "⚠️  需要 root 权限"
    echo "   请运行: sudo ./fix-dns-and-mirror.sh"
    exit 1
fi

echo "🔍 检查 DNS 配置..."
CURRENT_DNS=$(grep nameserver /etc/resolv.conf | head -1 | awk '{print $2}')
echo "   当前 DNS: $CURRENT_DNS"

# 测试 DNS
echo ""
echo "🧪 测试 DNS 解析..."
if nslookup docker.mirrors.ustc.edu.cn >/dev/null 2>&1; then
    echo "✅ DNS 解析正常"
else
    echo "⚠️  DNS 解析失败，尝试修复..."
    
    # 备份 resolv.conf
    cp /etc/resolv.conf /etc/resolv.conf.backup.$(date +%Y%m%d_%H%M%S)
    
    # 添加备用 DNS
    if ! grep -q "8.8.8.8" /etc/resolv.conf; then
        echo "nameserver 8.8.8.8" >> /etc/resolv.conf
        echo "nameserver 114.114.114.114" >> /etc/resolv.conf
        echo "✅ 已添加备用 DNS (8.8.8.8, 114.114.114.114)"
    fi
    
    # 测试新 DNS
    sleep 1
    if nslookup docker.mirrors.ustc.edu.cn >/dev/null 2>&1; then
        echo "✅ DNS 修复成功"
    else
        echo "⚠️  DNS 仍然无法解析，可能需要："
        echo "   1. 检查网络连接"
        echo "   2. 检查防火墙设置"
        echo "   3. 使用其他镜像源"
    fi
fi

echo ""
echo "🐳 配置 Docker 镜像加速器..."

DOCKER_DAEMON_JSON="/etc/docker/daemon.json"
mkdir -p /etc/docker

# 备份现有配置
if [ -f "$DOCKER_DAEMON_JSON" ]; then
    cp "$DOCKER_DAEMON_JSON" "${DOCKER_DAEMON_JSON}.backup.$(date +%Y%m%d_%H%M%S)"
fi

# 使用多个镜像源（包括可用的）
cat > "$DOCKER_DAEMON_JSON" << 'EOF'
{
  "registry-mirrors": [
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com",
    "https://dockerhub.azk8s.cn",
    "https://docker.mirrors.ustc.edu.cn"
  ],
  "dns": ["8.8.8.8", "114.114.114.114", "223.5.5.5"]
}
EOF

echo "✅ Docker 配置已更新"
echo "   - 镜像加速器: 网易、百度云、Azure、中科大"
echo "   - DNS: 8.8.8.8, 114.114.114.114, 223.5.5.5"

echo ""
echo "🔄 正在重启 Docker 服务..."
systemctl daemon-reload
systemctl restart docker

if [ $? -eq 0 ]; then
    echo "✅ Docker 服务重启成功"
    sleep 2
    
    echo ""
    echo "🧪 测试镜像拉取..."
    echo "   尝试拉取 alpine:latest..."
    if timeout 30 docker pull alpine:latest 2>&1 | head -10; then
        echo ""
        echo "✅ 镜像拉取成功！"
    else
        echo ""
        echo "⚠️  镜像拉取可能仍有问题"
        echo "   建议使用离线构建方案: ./docker-build-offline.sh"
    fi
else
    echo "❌ Docker 服务重启失败"
fi

echo ""
echo "=========================================="
echo "✅ 配置完成！"
echo "=========================================="
echo ""
echo "如果镜像拉取仍然失败，建议使用："
echo "   ./docker-build-offline.sh  (离线构建方案)"
echo ""
