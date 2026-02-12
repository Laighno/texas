#!/bin/bash

# 自动配置防火墙开放端口

echo "=========================================="
echo "  防火墙配置工具"
echo "=========================================="
echo ""

# 检查是否有 root 权限
if [ "$EUID" -ne 0 ]; then 
    echo "⚠️  需要 root 权限"
    echo "   请运行: sudo ./fix-firewall.sh"
    exit 1
fi

PORT=8085

echo "🔧 正在配置防火墙开放端口 $PORT..."
echo ""

# 检测系统类型并配置防火墙
if command -v ufw &> /dev/null; then
    echo "检测到 UFW 防火墙"
    ufw allow $PORT/tcp
    ufw reload
    echo "✅ UFW 已配置"
    
elif command -v firewall-cmd &> /dev/null; then
    echo "检测到 firewalld 防火墙"
    firewall-cmd --permanent --add-port=$PORT/tcp
    firewall-cmd --reload
    echo "✅ firewalld 已配置"
    
elif command -v iptables &> /dev/null; then
    echo "检测到 iptables 防火墙"
    iptables -I INPUT -p tcp --dport $PORT -j ACCEPT
    
    # 保存规则（根据系统不同）
    if command -v iptables-save &> /dev/null; then
        iptables-save > /etc/iptables/rules.v4 2>/dev/null || \
        iptables-save > /etc/sysconfig/iptables 2>/dev/null || \
        echo "⚠️  请手动保存 iptables 规则"
    fi
    echo "✅ iptables 已配置"
    
else
    echo "⚠️  未检测到常见防火墙，请手动配置"
    echo ""
    echo "常见命令："
    echo "  UFW: sudo ufw allow $PORT/tcp"
    echo "  firewalld: sudo firewall-cmd --add-port=$PORT/tcp --permanent"
    echo "  iptables: sudo iptables -I INPUT -p tcp --dport $PORT -j ACCEPT"
    exit 1
fi

echo ""
echo "🧪 验证配置..."
if netstat -tuln 2>/dev/null | grep -q ":$PORT " || ss -tuln 2>/dev/null | grep -q ":$PORT "; then
    echo "✅ 端口 $PORT 已开放"
else
    echo "⚠️  端口可能仍未开放，请检查云服务商安全组"
fi

echo ""
echo "=========================================="
echo "✅ 配置完成！"
echo "=========================================="
echo ""
echo "⚠️  重要提示："
echo "   如果仍然无法访问，请检查："
echo "   1. 云服务商安全组规则（必须开放端口 $PORT）"
echo "   2. 运行诊断: ./check-network.sh"
echo ""
