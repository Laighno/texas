# 德州扑克多人游戏

一个支持4-12人同时游戏的德州扑克Web应用，包含Go服务端和H5前端。

## 功能特性

- ✅ 支持4-12人同时游戏
- ✅ 实时WebSocket通信
- ✅ 完整的德州扑克游戏逻辑
- ✅ 房间系统（创建/加入房间）
- ✅ 完整的牌型判断（高牌、一对、两对、三条、顺子、同花、葫芦、四条、同花顺、皇家同花顺）
- ✅ 游戏阶段管理（翻牌前、翻牌、转牌、河牌、比牌）
- ✅ 下注系统（弃牌、过牌、跟注、加注、全押）
- ✅ 响应式设计，支持移动端

## 技术栈

### 服务端
- Go 1.21+
- Gorilla WebSocket
- 并发安全的房间管理

### 前端
- 原生HTML/CSS/JavaScript
- WebSocket API
- 响应式设计

## 安装和运行

### 方式一：使用 Docker（推荐，一行命令启动）

#### 前置要求

- Docker 和 Docker Compose
- 现代浏览器（支持WebSocket）

#### 启动步骤

**一行命令启动：**

```bash
./docker-start.sh
```

或者使用 docker-compose：

```bash
docker-compose up -d
```

服务器将在 `http://localhost:8080` 启动

**⚠️ 如果遇到网络超时问题**（无法拉取 Docker 镜像），请参考：
- [快速修复指南](./QUICK-FIX.md) - **推荐先看这个**
- [Docker 网络问题解决方案](./README-DOCKER-NETWORK.md)

**快速解决方案：**
1. **离线构建**（最可靠，推荐）：`./docker-build-offline.sh` 
   - 使用项目内打包的 alpine 镜像，无需网络
   - 自动检测并加载镜像
2. **DNS 问题**：`sudo ./fix-dns-and-mirror.sh` （修复 DNS + 镜像加速器）
3. **本地编译**：`./docker-build-local.sh` （需要拉取 alpine 镜像）

**注意**：项目内已包含 `alpine-latest.tar` 镜像文件，离线构建时会自动加载。

**常用 Docker 命令：**

```bash
# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 重新构建并启动
docker-compose up -d --build
```

### 方式二：本地运行

#### 前置要求

- Go 1.21 或更高版本
- 现代浏览器（支持WebSocket）

#### 启动步骤

1. **安装依赖**

```bash
cd /home/laighno/go/src/texas
go mod download
```

2. **启动服务器**

```bash
go run main.go game.go
```

服务器将在 `http://localhost:8080` 启动

3. **打开浏览器**

在浏览器中访问 `http://localhost:8080`

## 游戏规则

### 基本规则

1. **玩家数量**: 4-12人
2. **初始筹码**: 每个玩家1000筹码
3. **盲注**: 
   - 小盲注: 10筹码
   - 大盲注: 20筹码

### 游戏流程

1. **翻牌前 (Pre-flop)**
   - 发两张底牌给每个玩家
   - 下大小盲注
   - 从大盲注下一位开始行动

2. **翻牌 (Flop)**
   - 发3张公共牌
   - 玩家继续下注

3. **转牌 (Turn)**
   - 发第4张公共牌
   - 玩家继续下注

4. **河牌 (River)**
   - 发第5张公共牌
   - 最后一轮下注

5. **比牌 (Showdown)**
   - 所有未弃牌玩家亮牌
   - 比较最佳5张牌组合
   - 获胜者获得底池

### 牌型等级（从高到低）

1. 皇家同花顺 (Royal Flush)
2. 同花顺 (Straight Flush)
3. 四条 (Four of a Kind)
4. 葫芦 (Full House)
5. 同花 (Flush)
6. 顺子 (Straight)
7. 三条 (Three of a Kind)
8. 两对 (Two Pair)
9. 一对 (One Pair)
10. 高牌 (High Card)

## 使用说明

### 创建房间

1. 输入你的名字
2. 房间ID留空
3. 点击"加入游戏"
4. 系统会生成一个房间ID，分享给其他玩家

### 加入房间

1. 输入你的名字
2. 输入房间ID
3. 点击"加入游戏"

### 开始游戏

1. 等待至少4个玩家加入
2. 点击"开始游戏"按钮
3. 游戏开始后，按照提示进行下注操作

### 游戏操作

- **弃牌 (Fold)**: 放弃本局游戏
- **过牌 (Check)**: 不下注，但继续游戏（仅当无需跟注时）
- **跟注 (Call)**: 下注与当前最高下注相同的金额
- **加注 (Raise)**: 在跟注基础上额外增加下注

## 项目结构

```
awesomeProject/
├── main.go          # 主服务器文件（WebSocket、房间管理）
├── game.go          # 游戏逻辑（牌型判断、比牌）
├── go.mod           # Go模块依赖
├── index.html       # 前端HTML页面
├── style.css        # 前端样式
├── app.js           # 前端JavaScript逻辑
└── README.md        # 项目说明
```

## 开发说明

### 消息协议

#### 客户端 -> 服务端

```json
// 创建房间
{
  "type": "createRoom",
  "data": {
    "playerName": "玩家名字"
  }
}

// 加入房间
{
  "type": "joinRoom",
  "data": {
    "roomId": "房间ID",
    "playerName": "玩家名字"
  }
}

// 开始游戏
{
  "type": "startGame",
  "data": {}
}

// 玩家行动
{
  "type": "action",
  "data": {
    "action": "fold|check|call|raise",
    "amount": 0  // raise时使用
  }
}
```

#### 服务端 -> 客户端

```json
// 房间创建成功
{
  "type": "roomCreated",
  "data": {
    "roomId": "房间ID"
  }
}

// 玩家加入
{
  "type": "playerJoined",
  "data": {
    "player": {...},
    "room": {...}
  }
}

// 游戏开始/状态更新
{
  "type": "gameStarted|actionTaken",
  "data": {
    // 完整的房间状态
  }
}

// 游戏结束
{
  "type": "gameEnded",
  "data": {
    "winner": {...},
    "pot": 1000,
    "winningHand": "同花顺"
  }
}
```

## 注意事项

1. 当前版本为演示版本，部分功能可能需要进一步完善
2. 建议在局域网或本地环境测试
3. 如需部署到生产环境，请添加：
   - 身份验证
   - 房间密码
   - 断线重连
   - 更完善的错误处理
   - 日志记录

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！
