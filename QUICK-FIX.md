# 快速修复指南

## 问题 1：镜像拉取超时

如果遇到以下错误：
```
dial tcp: i/o timeout
failed to resolve source metadata
```

## 问题 2：镜像源不存在

如果遇到以下错误：
```
registry.cn-hangzhou.aliyuncs.com/acs/golang:1.21-alpine: not found
```

## 解决方案

### 方案 1：配置 Docker 镜像加速器（推荐）

这是最简单可靠的方案：

```bash
# 1. 配置镜像加速器
sudo ./setup-docker-mirror.sh

# 2. 使用标准 Dockerfile 构建
docker-compose up -d --build
```

配置后，Docker 会自动通过镜像加速器拉取官方镜像。

### 方案 2：使用标准 Dockerfile

如果已经配置了镜像加速器，直接使用标准配置：

```bash
docker-compose up -d --build
```

### 方案 3：手动拉取镜像

如果镜像加速器配置失败，可以手动拉取：

```bash
# 使用中科大镜像源拉取
docker pull docker.mirrors.ustc.edu.cn/library/golang:1.21-alpine
docker tag docker.mirrors.ustc.edu.cn/library/golang:1.21-alpine golang:1.21-alpine

docker pull docker.mirrors.ustc.edu.cn/library/alpine:latest
docker tag docker.mirrors.ustc.edu.cn/library/alpine:latest alpine:latest

# 然后构建
docker-compose up -d --build
```

### 方案 4：使用本地编译

如果 Docker 镜像拉取一直失败，可以在本地编译后打包：

```bash
# 1. 本地编译
go build -o poker-server main.go game.go

# 2. 使用轻量级 Dockerfile
cat > Dockerfile.local << 'EOF'
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY poker-server index.html style.css app.js ./
EXPOSE 8080
CMD ["./poker-server"]
EOF

# 3. 构建
docker build -f Dockerfile.local -t texas-poker:local .

# 4. 运行
docker run -d -p 8080:8080 --name texas-poker texas-poker:local
```

## 推荐流程

### 方案 A：配置镜像加速器（推荐）

1. **强制配置镜像加速器**：
   ```bash
   sudo ./fix-docker-mirror.sh
   ```

2. **然后使用标准构建**：
   ```bash
   docker-compose up -d --build
   ```

3. **如果还是失败，检查网络**：
   ```bash
   # 测试镜像加速器
   docker pull alpine:latest
   
   # 如果成功，说明配置生效
   ```

### 方案 B：本地编译方案（最可靠）

如果镜像加速器配置后仍然失败，使用本地编译：

```bash
./docker-build-local.sh
```

这个方案：
- ✅ 在本地编译 Go 程序（无需 Docker 构建镜像）
- ✅ 只拉取 alpine:latest（很小的镜像）
- ✅ 即使 alpine 拉取失败，也可以手动拉取后重试

### 方案 C：完全离线方案

如果完全无法访问 Docker Hub：

```bash
# 1. 本地编译
go build -o poker-server main.go game.go

# 2. 手动拉取 alpine（通过其他方式获取）
# 或使用已有镜像

# 3. 构建
docker build -f Dockerfile.local -t texas-poker:local .
docker run -d -p 8080:8080 --name texas-poker-server texas-poker:local
```

## 验证配置

配置镜像加速器后，验证是否生效：

```bash
docker info | grep -A 10 "Registry Mirrors"
```

应该看到配置的镜像加速器地址。
