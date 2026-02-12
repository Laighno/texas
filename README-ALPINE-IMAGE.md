# Alpine 镜像打包说明

## 概述

项目内包含了 `alpine-latest.tar` 文件，这是打包好的 Alpine Linux Docker 镜像。这样在云端部署时就不需要从网络拉取镜像了。

## 文件说明

- `alpine-latest.tar` - 打包的 Alpine Linux 镜像文件（约 5-10MB）
- `load-alpine-image.sh` - 加载镜像的脚本

## 使用方法

### 方法 1：自动加载（推荐）

运行离线构建脚本，会自动检测并加载镜像：

```bash
./docker-build-offline.sh
```

### 方法 2：手动加载

如果需要单独加载镜像：

```bash
./load-alpine-image.sh
```

或直接使用 Docker 命令：

```bash
docker load -i alpine-latest.tar
```

## 更新镜像

如果需要更新 Alpine 镜像：

```bash
# 1. 拉取最新镜像（如果有网络）
docker pull alpine:latest

# 2. 导出镜像
docker save alpine:latest -o alpine-latest.tar

# 3. 提交到项目
git add alpine-latest.tar
git commit -m "Update alpine image"
```

## 注意事项

1. **文件大小**：alpine-latest.tar 文件约 5-10MB，已包含在项目中
2. **版本控制**：建议将镜像文件提交到 Git（如果仓库允许）
3. **压缩**：如果需要减小文件大小，可以使用 gzip 压缩：
   ```bash
   docker save alpine:latest | gzip > alpine-latest.tar.gz
   # 加载时：
   gunzip -c alpine-latest.tar.gz | docker load
   ```

## 故障排查

### 镜像加载失败

如果加载失败，检查：
1. 文件是否完整：`ls -lh alpine-latest.tar`
2. Docker 是否运行：`docker ps`
3. 磁盘空间是否足够：`df -h`

### 构建时找不到镜像

确保：
1. 已运行 `./load-alpine-image.sh` 或 `./docker-build-offline.sh`
2. 镜像已加载：`docker images | grep alpine`
