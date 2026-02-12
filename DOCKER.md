# Docker 部署指南

## 快速开始

### 一行命令启动

```bash
./docker-start.sh
```

或者直接使用 docker-compose：

```bash
docker-compose up -d
```

### 访问应用

启动成功后，在浏览器中访问：**http://localhost:8080**

## Docker 命令参考

### 启动服务

```bash
# 后台启动
docker-compose up -d

# 前台启动（查看日志）
docker-compose up
```

### 查看日志

```bash
# 查看所有日志
docker-compose logs

# 实时查看日志
docker-compose logs -f

# 查看最近100行日志
docker-compose logs --tail=100
```

### 停止服务

```bash
# 停止服务（保留容器）
docker-compose stop

# 停止并删除容器
docker-compose down

# 停止并删除容器和镜像
docker-compose down --rmi local
```

### 重启服务

```bash
# 重启服务
docker-compose restart

# 重新构建并启动
docker-compose up -d --build
```

### 查看状态

```bash
# 查看容器状态
docker-compose ps

# 查看容器详细信息
docker-compose ps -a
```

## 构建说明

### 手动构建镜像

```bash
# 构建镜像
docker build -t texas-poker:latest .

# 运行容器
docker run -d -p 8080:8080 --name texas-poker texas-poker:latest
```

### 多阶段构建

Dockerfile 使用多阶段构建，分为两个阶段：

1. **构建阶段**：使用 `golang:1.21-alpine` 镜像编译 Go 应用
2. **运行阶段**：使用 `alpine:latest` 镜像运行编译后的二进制文件

这样可以减小最终镜像大小。

## 端口配置

默认端口：**8080**

如需修改端口，编辑 `docker-compose.yml`：

```yaml
ports:
  - "8080:8080"  # 修改左侧端口号即可
```

## 故障排查

### Docker 镜像拉取失败 / 网络超时

如果遇到以下错误：
```
failed to solve: golang:1.21-alpine: failed to resolve source metadata
dial tcp: i/o timeout
```

**解决方案 1：配置 Docker 镜像加速器（推荐）**

运行配置脚本：
```bash
sudo ./setup-docker-mirror.sh
```

或手动配置，编辑 `/etc/docker/daemon.json`：
```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ]
}
```

然后重启 Docker：
```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

**解决方案 2：使用国内镜像源版本**

使用专门为国内网络优化的 Dockerfile：
```bash
docker-compose -f docker-compose.mirror.yml up -d --build
```

**解决方案 3：手动拉取镜像**

```bash
# 使用镜像加速器拉取
docker pull docker.mirrors.ustc.edu.cn/library/golang:1.21-alpine
docker tag docker.mirrors.ustc.edu.cn/library/golang:1.21-alpine golang:1.21-alpine

docker pull docker.mirrors.ustc.edu.cn/library/alpine:latest
docker tag docker.mirrors.ustc.edu.cn/library/alpine:latest alpine:latest
```

### 端口被占用

如果 8080 端口被占用，可以：

1. 修改 `docker-compose.yml` 中的端口映射
2. 或者停止占用端口的服务

```bash
# 查看端口占用
sudo lsof -i :8080
# 或
sudo netstat -tulpn | grep 8080
```

### 查看容器日志

```bash
docker-compose logs texas-poker
```

### 进入容器调试

```bash
# 进入容器
docker-compose exec texas-poker sh

# 查看文件
ls -la /app
```

### 重新构建

如果代码有更新，需要重新构建：

```bash
docker-compose up -d --build
```

## 环境要求

- Docker 20.10+
- Docker Compose 1.29+ (或 Docker Compose V2)

## 注意事项

1. 确保 Docker 服务正在运行
2. 确保 8080 端口未被占用
3. 首次构建可能需要几分钟时间下载依赖
4. 静态文件（HTML/CSS/JS）会自动包含在镜像中
