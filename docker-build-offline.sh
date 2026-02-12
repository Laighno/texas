#!/bin/bash

# 完全离线/本地构建方案（不依赖外部镜像源）

echo "=========================================="
echo "  德州扑克服务器 - 离线构建方案"
echo "=========================================="
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到 Go，请先安装 Go 1.21+"
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
echo "🐳 检查 Docker 镜像..."

# 检查是否有 alpine 镜像
if docker images | grep -q "alpine.*latest"; then
    echo "✅ 找到本地 alpine 镜像"
else
    echo "⚠️  未找到 alpine 镜像，尝试从项目内加载..."
    
    # 检查是否有打包的镜像文件
    if [ -f "alpine-latest.tar" ]; then
        echo "📦 发现打包的镜像文件，正在加载..."
        docker load -i alpine-latest.tar
        
        if [ $? -eq 0 ] && docker images | grep -q "alpine.*latest"; then
            echo "✅ Alpine 镜像加载成功！"
        else
            echo "❌ 镜像加载失败"
            exit 1
        fi
    else
        echo "⚠️  未找到打包的镜像文件，尝试使用 busybox..."
        if docker images | grep -q "busybox"; then
            echo "✅ 找到 busybox 镜像，将使用它"
            ALPINE_IMAGE="busybox:latest"
        else
            echo "❌ 未找到任何可用的基础镜像"
            echo ""
            echo "💡 解决方案："
            echo "   1. 运行: ./load-alpine-image.sh 加载打包的镜像"
            echo "   2. 或检查是否有其他基础镜像: docker images"
            echo "   3. 或使用系统包管理器安装 Docker 镜像"
            exit 1
        fi
    fi
fi

ALPINE_IMAGE=${ALPINE_IMAGE:-"alpine:latest"}

echo ""
echo "🐳 正在构建 Docker 镜像（使用本地镜像）..."

# 创建临时 Dockerfile
cat > Dockerfile.offline << EOF
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates 2>/dev/null || true
WORKDIR /app
COPY poker-server index.html style.css app.js ./
EXPOSE 8080
CMD ["./poker-server"]
EOF

# 确保文件存在
if [ ! -f "poker-server" ]; then
    echo "❌ 错误: poker-server 文件不存在"
    exit 1
fi

# 构建镜像（忽略 .dockerignore 中的 poker-server）
# 方法：临时修改 .dockerignore
if [ -f ".dockerignore" ]; then
    # 创建临时 .dockerignore，排除 poker-server 的排除规则
    grep -v "^poker-server$" .dockerignore > .dockerignore.temp 2>/dev/null || echo "" > .dockerignore.temp
    mv .dockerignore .dockerignore.backup
    mv .dockerignore.temp .dockerignore
fi

# 构建镜像
docker build -f Dockerfile.offline -t texas-poker:offline .

# 恢复 .dockerignore
if [ -f ".dockerignore.backup" ]; then
    mv .dockerignore.backup .dockerignore
    rm -f .dockerignore.temp
fi

if [ $? -ne 0 ]; then
    echo "❌ Docker 构建失败"
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
docker run -d -p 8085:8080 --name texas-poker-server --restart unless-stopped texas-poker:offline

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
