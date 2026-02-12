# 打平逻辑测试指南

## 修复内容

### 1. determineWinner 函数修复
- ✅ 支持识别多个获胜者（打平情况）
- ✅ 多人打平时平分底池
- ✅ 余数分配给第一个获胜者
- ✅ 添加 `winners` 和 `isTie` 字段到 `gameEnded` 消息

### 2. 代码变更
```go
// 之前：只选择第一个获胜者
winner = activePlayers[0]

// 现在：识别所有获胜者
winners = []*Player{}
for _, p := range activePlayers {
    handRank := evaluateHand(p.Hand, room.CommunityCards)
    comparison := compareHandRanks(handRank, bestRank)
    if comparison > 0 {
        bestRank = handRank
        winners = []*Player{p}
    } else if comparison == 0 {
        winners = append(winners, p)  // 打平
    }
}

// 平分底池
if len(winners) > 1 {
    share := pot / len(winners)
    remainder := pot % len(winners)
    for i, w := range winners {
        w.Chips += share
        if i == 0 {
            w.Chips += remainder  // 余数给第一个玩家
        }
    }
}
```

## 测试方法

### 手动测试步骤

1. **启动服务器**
   ```bash
   ./start.sh
   ```

2. **创建测试场景：两个玩家打平**
   - 4个玩家加入房间
   - 开始游戏
   - 所有玩家跟注到河牌
   - 确保公共牌能让多个玩家组成相同的最佳牌型
   - 例如：公共牌是同花顺，所有玩家都使用公共牌

3. **验证结果**
   - 检查服务器日志，应该看到 "多人打平" 的日志
   - 检查玩家筹码，应该看到底池被平分
   - 检查前端收到的 `gameEnded` 消息，应该包含：
     - `isTie: true`
     - `winners: [玩家1, 玩家2, ...]`
     - `winningHand: "同花顺 (多人打平)"`

### 测试场景

#### 场景1：两个玩家打平（同花顺）
- **公共牌**：10♠ 9♠ 8♠ 7♠ 6♠
- **玩家1手牌**：任意两张牌
- **玩家2手牌**：任意两张牌
- **玩家3、4**：弃牌
- **预期**：玩家1和玩家2平分底池

#### 场景2：三个玩家打平（皇家同花顺）
- **公共牌**：A♠ K♠ Q♠ J♠ 10♠
- **所有玩家手牌**：任意两张牌
- **预期**：三个玩家平分底池

#### 场景3：单个获胜者
- **公共牌**：10♠ 9♠ 8♠ 7♥ 6♠
- **玩家1手牌**：A♠ K♠（同花）
- **玩家2手牌**：任意两张牌
- **预期**：玩家1获得全部底池

## 验证点

1. ✅ **底池分配正确**
   - 多人打平时，每人获得 `pot / winners_count`
   - 余数给第一个获胜者
   - 总分配 = 底池总额

2. ✅ **消息格式正确**
   - `gameEnded` 消息包含 `winners` 数组
   - `isTie` 字段正确标识打平情况
   - `winningHand` 显示 "(多人打平)"

3. ✅ **日志记录**
   - 服务器日志记录打平信息
   - 包含获胜者数量、底池、每人分得金额

## 边缘情况

### 已处理
- ✅ 两个玩家打平
- ✅ 三个或更多玩家打平
- ✅ 底池无法整除时的余数处理
- ✅ 单个获胜者（不打平）

### 待实现（未来）
- ⏳ 边池分配（全押情况）
- ⏳ 多个边池的复杂分配
- ⏳ 玩家筹码不足时的边池处理

## 测试命令

```bash
# 编译并启动服务器
go build -o poker_server main.go game.go
./poker_server

# 在另一个终端运行测试客户端
# 或使用浏览器打开 http://localhost:8080
```

## 预期日志输出

当发生打平时，服务器日志应该显示：
```
多人打平，房间 xxx，获胜者数: 2，底池: 100，每人分得: 50，余数: 0
```

或
```
多人打平，房间 xxx，获胜者数: 3，底池: 100，每人分得: 33，余数: 1
```
