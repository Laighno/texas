# 云端部署指南

## 公网访问配置

### 问题：无法通过公网访问

如果部署后无法通过公网访问，通常有以下原因：

1. **云服务商安全组未开放端口**（最常见）
2. **服务器防火墙阻止端口**
3. **端口映射配置错误**
4. **容器未正常启动**

## 快速诊断

运行诊断脚本：

```bash
./check-network.sh
```

这个脚本会检查：
- 容器运行状态
- 端口映射
- 防火墙配置
- 网络连接

## 解决方案

### 1. 配置云服务商安全组

**阿里云/腾讯云/华为云等：**

1. 登录云控制台
2. 找到"安全组"或"防火墙"设置
3. 添加入站规则：
   - 协议：TCP
   - 端口：8085（或你配置的端口）
   - 源：0.0.0.0/0（允许所有IP，或指定IP段）

**AWS：**

1. EC2 控制台 → 安全组
2. 添加入站规则：
   - Type: Custom TCP
   - Port: 8085
   - Source: 0.0.0.0/0

### 2. 配置服务器防火墙

**自动配置（推荐）：**

```bash
sudo ./fix-firewall.sh
```

**手动配置：**

**Ubuntu/Debian (UFW):**
```bash
sudo ufw allow 8085/tcp
sudo ufw reload
```

**CentOS/RHEL (firewalld):**
```bash
sudo firewall-cmd --permanent --add-port=8085/tcp
sudo firewall-cmd --reload
```

**iptables:**
```bash
sudo iptables -I INPUT -p tcp --dport 8085 -j ACCEPT
# 保存规则（根据系统不同）
sudo iptables-save > /etc/iptables/rules.v4
# 或
sudo service iptables save
```

### 3. 验证配置

**检查端口监听：**
```bash
netstat -tuln | grep 8085
# 或
ss -tuln | grep 8085
```

**测试本地访问：**
```bash
curl http://localhost:8085
```

**测试公网访问：**
```bash
# 获取公网 IP
curl ifconfig.me

# 测试访问（替换为你的公网 IP）
curl http://YOUR_PUBLIC_IP:8085
```

## 端口配置

当前配置：
- **容器内端口**: 8080
- **主机端口**: 8085
- **访问地址**: `http://YOUR_PUBLIC_IP:8085`

如需修改端口，编辑 `docker-build-offline.sh`：

```bash
# 修改这一行
docker run -d -p YOUR_PORT:8080 --name texas-poker-server ...
```

然后：
1. 更新安全组规则
2. 更新防火墙规则
3. 重启容器

## 常见问题

### Q: 本地可以访问，公网无法访问

**A:** 通常是安全组或防火墙问题
1. 检查云服务商安全组
2. 运行 `./check-network.sh` 诊断
3. 运行 `sudo ./fix-firewall.sh` 配置防火墙

### Q: 容器运行正常，但端口未监听

**A:** 检查端口映射
```bash
docker port texas-poker-server
```

应该显示：`8080/tcp -> 0.0.0.0:8085`

### Q: 如何查看容器日志

```bash
docker logs -f texas-poker-server
```

### Q: 如何重启服务

```bash
docker restart texas-poker-server
```

## 完整部署流程

1. **部署应用**
   ```bash
   ./docker-build-offline.sh
   ```

2. **配置防火墙**
   ```bash
   sudo ./fix-firewall.sh
   ```

3. **配置云服务商安全组**
   - 开放端口 8085 (TCP)

4. **验证访问**
   ```bash
   ./check-network.sh
   curl http://YOUR_PUBLIC_IP:8085
   ```

5. **访问应用**
   ```
   http://YOUR_PUBLIC_IP:8085
   ```

## 安全建议

1. **限制访问源**：在安全组中只允许特定 IP 访问
2. **使用 HTTPS**：配置反向代理（Nginx）并启用 SSL
3. **定期更新**：保持系统和镜像更新
4. **监控日志**：定期检查容器日志

## 获取帮助

如果问题仍未解决：

1. 运行诊断：`./check-network.sh`
2. 查看日志：`docker logs texas-poker-server`
3. 检查容器状态：`docker ps -a`
