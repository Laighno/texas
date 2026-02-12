# Docker 网络问题解决方案

## 问题描述

如果遇到以下错误：
```
failed to solve: golang:1.21-alpine: failed to resolve source metadata 
for docker.io/library/golang:1.21-alpine: failed to do request: 
Head "https://registry-1.docker.io/v2/library/golang/manifests/1.21-alpine": 
dial tcp: i/o timeout
```

这通常是因为无法访问 Docker Hub（在中国大陆地区常见）。

## 解决方案

### 方案 1：配置 Docker 镜像加速器（推荐）

**自动配置：**
```bash
sudo ./setup-docker-mirror.sh
```

**手动配置：**

1. 编辑或创建 `/etc/docker/daemon.json`：
```bash
sudo nano /etc/docker/daemon.json
```

2. 添加以下内容：
```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ]
}
```

3. 重启 Docker 服务：
```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

4. 验证配置：
```bash
docker info | grep -A 10 "Registry Mirrors"
```

### 方案 2：配置镜像加速器后使用标准构建（推荐）

**步骤：**
```bash
# 1. 配置镜像加速器（只需一次）
sudo ./setup-docker-mirror.sh

# 2. 使用标准构建
docker-compose up -d --build
```

配置后，Docker 会自动通过镜像加速器拉取官方镜像，无需修改 Dockerfile。

### 方案 3：使用镜像加速器版本（如果方案2失败）

**一键启动：**
```bash
./docker-start-mirror.sh
```

**或手动使用：**
```bash
docker-compose -f docker-compose.mirror.yml up -d --build
```

注意：此方案也需要先配置镜像加速器才能正常工作。

### 方案 3：手动拉取并重命名镜像

```bash
# 拉取镜像（使用镜像加速器）
docker pull docker.mirrors.ustc.edu.cn/library/golang:1.21-alpine
docker tag docker.mirrors.ustc.edu.cn/library/golang:1.21-alpine golang:1.21-alpine

docker pull docker.mirrors.ustc.edu.cn/library/alpine:latest
docker tag docker.mirrors.ustc.edu.cn/library/alpine:latest alpine:latest

# 然后正常构建
docker-compose up -d --build
```

### 方案 4：使用代理

如果有代理服务器，可以配置 Docker 使用代理：

1. 创建或编辑 `/etc/systemd/system/docker.service.d/http-proxy.conf`：
```ini
[Service]
Environment="HTTP_PROXY=http://proxy.example.com:8080"
Environment="HTTPS_PROXY=http://proxy.example.com:8080"
Environment="NO_PROXY=localhost,127.0.0.1"
```

2. 重启 Docker：
```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

## 常用国内镜像源

- **中科大镜像**: https://docker.mirrors.ustc.edu.cn
- **网易镜像**: https://hub-mirror.c.163.com
- **百度云镜像**: https://mirror.baidubce.com
- **阿里云镜像**: https://registry.cn-hangzhou.aliyuncs.com
- **腾讯云镜像**: https://mirror.ccs.tencentyun.com

## 验证配置

配置完成后，测试拉取镜像：
```bash
docker pull golang:1.21-alpine
```

如果成功，说明配置生效。

## 其他网络问题

### Go 模块下载慢

如果 Go 模块下载也慢，可以在 Dockerfile 中设置 Go 代理：

```dockerfile
ENV GOPROXY=https://goproxy.cn,direct
```

### 构建超时

如果构建过程超时，可以增加超时时间：
```bash
docker-compose build --progress=plain --no-cache
```

## 联系支持

如果以上方案都无法解决问题，请检查：
1. 网络连接是否正常
2. 防火墙设置
3. DNS 配置
4. Docker 版本是否最新
