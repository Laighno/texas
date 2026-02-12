#!/bin/bash

# 本地编译 + Docker 打包方案（无需拉取构建镜像）

echo "=========================================="
echo "  德州扑克服务器 - 本地编译方案"
echo "=========================================="
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到 Go，请先安装 Go 1.21+"
    echo "   安装指南: https://golang.org/doc/install"
    exit 1
fi

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ 错误: 未找到 Docker，请先安装 Docker"
    exit 1
fi

echo "✅ 环境检查通过"
echo ""

# 检查是否已编译
if [ ! -f "poker-server" ] || [ "main.go" -nt "poker-server" ] || [ "game.go" -nt "poker-server" ]; then
    echo "📦 正在本地编译 Go 程序..."
    export CGO_ENABLED=0
    export GOOS=linux
    export GOARCH=amd64
    go build -o poker-server main.go game.go
    
    if [ $? -ne 0 ]; then
        echo "❌ 编译失败"
        exit 1
    fi
    echo "✅ 编译成功"
else
    echo "✅ 使用已存在的编译文件"
fi

echo ""
echo "🐳 正在构建 Docker 镜像（使用本地编译的二进制文件）..."

# 创建临时 Dockerfile
cat > Dockerfile.local << 'EOF'
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY poker-server index.html style.css app.js ./
EXPOSE 8080
CMD ["./poker-server"]
EOF

# 构建镜像
docker build -f Dockerfile.local -t texas-poker:local .

if [ $? -ne 0 ]; then
    echo "❌ Docker 构建失败"
    echo ""
    echo "💡 提示：如果 alpine:latest 拉取失败，尝试："
    echo "   1. 配置镜像加速器: sudo ./setup-docker-mirror.sh"
    echo "   2. 或手动拉取: docker pull docker.mirrors.ustc.edu.cn/library/alpine:latest"
    echo "     然后: docker tag docker.mirrors.ustc.edu.cn/library/alpine:latest alpine:latest"
    exit 1
fi

echo "✅ Docker 镜像构建成功"
echo ""

# 停止并删除旧容器
echo "🛑 停止旧容器（如果存在）..."
docker stop texas-poker-server 2>/dev/null
docker rm texas-poker-server 2>/dev/null

# 启动容器
echo "🚀 启动容器..."
docker run -d -p 8080:8080 --name texas-poker-server --restart unless-stopped texas-poker:local

if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "✅ 服务器启动成功！"
    echo "=========================================="
    echo ""
    echo "🌐 访问地址: http://localhost:8080"
    echo ""
    echo "📋 常用命令:"
    echo "   查看日志: docker logs -f texas-poker-server"
    echo "   停止服务: docker stop texas-poker-server"
    echo "   重启服务: docker restart texas-poker-server"
    echo ""
    echo "=========================================="
else
    echo "❌ 启动失败"
    exit 1
fi
