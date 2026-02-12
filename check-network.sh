#!/bin/bash

# 网络诊断脚本 - 检查公网访问问题

echo "=========================================="
echo "  网络诊断工具"
echo "=========================================="
echo ""

PORT=8085

# 1. 检查容器状态
echo "1️⃣ 检查容器状态..."
if docker ps | grep -q "texas-poker-server"; then
    echo "✅ 容器正在运行"
    docker ps | grep texas-poker-server
else
    echo "❌ 容器未运行"
    exit 1
fi

echo ""
echo "2️⃣ 检查端口映射..."
if docker port texas-poker-server 2>/dev/null | grep -q "$PORT"; then
    echo "✅ 端口映射正常"
    docker port texas-poker-server
else
    echo "⚠️  端口映射可能有问题"
    docker port texas-poker-server
fi

echo ""
echo "3️⃣ 检查容器内服务..."
if docker exec texas-poker-server wget -q -O- http://localhost:8080 >/dev/null 2>&1; then
    echo "✅ 容器内服务正常（端口 8080）"
else
    echo "❌ 容器内服务无响应"
fi

echo ""
echo "4️⃣ 检查主机端口监听..."
if netstat -tuln 2>/dev/null | grep -q ":$PORT " || ss -tuln 2>/dev/null | grep -q ":$PORT "; then
    echo "✅ 主机端口 $PORT 正在监听"
    netstat -tuln 2>/dev/null | grep ":$PORT " || ss -tuln 2>/dev/null | grep ":$PORT "
else
    echo "❌ 主机端口 $PORT 未监听"
fi

echo ""
echo "5️⃣ 检查防火墙状态..."

# 检查 ufw
if command -v ufw &> /dev/null; then
    UFW_STATUS=$(ufw status 2>/dev/null | head -1)
    echo "UFW 状态: $UFW_STATUS"
    if echo "$UFW_STATUS" | grep -q "active"; then
        if ufw status | grep -q "$PORT"; then
            echo "✅ UFW 已开放端口 $PORT"
        else
            echo "⚠️  UFW 未开放端口 $PORT"
            echo "   运行: sudo ufw allow $PORT/tcp"
        fi
    fi
fi

# 检查 iptables
if command -v iptables &> /dev/null && [ "$EUID" -eq 0 ]; then
    if iptables -L -n | grep -q "$PORT"; then
        echo "✅ iptables 规则包含端口 $PORT"
    else
        echo "⚠️  iptables 可能阻止了端口 $PORT"
    fi
fi

echo ""
echo "6️⃣ 检查云服务商安全组..."

# 获取公网 IP
PUBLIC_IP=$(curl -s ifconfig.me 2>/dev/null || curl -s ipinfo.io/ip 2>/dev/null || echo "无法获取")
LOCAL_IP=$(hostname -I | awk '{print $1}')

echo "   公网 IP: $PUBLIC_IP"
echo "   内网 IP: $LOCAL_IP"

echo ""
echo "7️⃣ 测试本地访问..."
if curl -s -o /dev/null -w "%{http_code}" http://localhost:$PORT | grep -q "200\|301\|302"; then
    echo "✅ 本地访问正常"
else
    echo "❌ 本地访问失败"
fi

echo ""
echo "=========================================="
echo "📋 诊断总结和建议"
echo "=========================================="
echo ""
echo "如果无法公网访问，请检查："
echo ""
echo "1. 云服务商安全组规则："
echo "   - 确保开放端口 $PORT (TCP)"
echo "   - 检查入站规则"
echo ""
echo "2. 服务器防火墙："
echo "   Ubuntu/Debian: sudo ufw allow $PORT/tcp"
echo "   CentOS/RHEL: sudo firewall-cmd --add-port=$PORT/tcp --permanent"
echo "                 sudo firewall-cmd --reload"
echo ""
echo "3. 测试访问："
echo "   本地: curl http://localhost:$PORT"
echo "   公网: curl http://$PUBLIC_IP:$PORT"
echo ""
echo "4. 查看容器日志："
echo "   docker logs texas-poker-server"
echo ""
echo "=========================================="
