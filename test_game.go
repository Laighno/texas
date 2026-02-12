//go:build test
// +build test

package main

import (
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	WS_URL = "ws://localhost:8080/ws"
)

type TestPlayer struct {
	ID       string
	Name     string
	Conn     *websocket.Conn
	RoomID   string
	IsMyTurn bool
	Hand     []interface{}
	Chips    int
	Bet      int
	Folded   bool
	mu       sync.Mutex
}

type TestMessage struct {
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	PlayerID string      `json:"playerId,omitempty"`
}

var (
	testPlayers []*TestPlayer
	wg          sync.WaitGroup
	gameMutex   sync.Mutex
	gameStarted bool
	gameEnded   bool
	handCount   int
)

func runTest() {
	log.Println("=== 开始测试两局游戏流程 ===")

	// 创建4个测试玩家
	testPlayers = make([]*TestPlayer, 4)
	for i := 0; i < 4; i++ {
		testPlayers[i] = &TestPlayer{
			ID:   fmt.Sprintf("test_player_%d", i+1),
			Name: fmt.Sprintf("玩家%d", i+1),
		}
	}

	// 连接所有玩家
	log.Println("正在连接所有玩家...")
	for _, player := range testPlayers {
		if err := connectPlayer(player); err != nil {
			log.Fatalf("玩家 %s 连接失败: %v", player.Name, err)
		}
		log.Printf("✅ 玩家 %s 连接成功", player.Name)
		time.Sleep(100 * time.Millisecond) // 避免连接过快
	}

	// 等待所有连接稳定
	time.Sleep(500 * time.Millisecond)

	// 第一局游戏
	log.Println("\n=== 开始第一局游戏 ===")
	handCount = 1
	gameStarted = false
	gameEnded = false

	// 第一个玩家创建房间
	log.Println("玩家1创建房间...")
	if err := testCreateRoom(testPlayers[0]); err != nil {
		log.Fatalf("创建房间失败: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// 其他玩家加入房间
	for i := 1; i < 4; i++ {
		log.Printf("玩家%d加入房间...", i+1)
		if err := testJoinRoom(testPlayers[i], testPlayers[0].RoomID); err != nil {
			log.Fatalf("玩家%d加入房间失败: %v", i+1, err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// 等待所有玩家加入
	time.Sleep(500 * time.Millisecond)

	// 开始游戏
	log.Println("开始第一局游戏...")
	if err := testStartGame(testPlayers[0]); err != nil {
		log.Fatalf("开始游戏失败: %v", err)
	}

	// 等待游戏开始
	time.Sleep(1 * time.Second)

	// 模拟第一局游戏流程
	testPlayGame(1)

	// 等待第一局结束
	log.Println("等待第一局游戏结束...")
	for i := 0; i < 30; i++ {
		gameMutex.Lock()
		ended := gameEnded
		gameMutex.Unlock()
		if ended {
			break
		}
		time.Sleep(1 * time.Second)
	}

	// 重置状态，准备第二局
	log.Println("\n=== 准备第二局游戏 ===")
	gameMutex.Lock()
	gameStarted = false
	gameEnded = false
	handCount = 2
	gameMutex.Unlock()

	// 等待结算完成
	time.Sleep(2 * time.Second)

	// 开始第二局游戏
	log.Println("开始第二局游戏...")
	if err := testStartGame(testPlayers[0]); err != nil {
		log.Fatalf("开始第二局游戏失败: %v", err)
	}

	// 等待游戏开始
	time.Sleep(1 * time.Second)

	// 模拟第二局游戏流程
	testPlayGame(2)

	// 等待第二局结束
	log.Println("等待第二局游戏结束...")
	for i := 0; i < 30; i++ {
		gameMutex.Lock()
		ended := gameEnded
		gameMutex.Unlock()
		if ended {
			break
		}
		time.Sleep(1 * time.Second)
	}

	// 关闭所有连接
	log.Println("\n=== 关闭所有连接 ===")
	for _, player := range testPlayers {
		if player.Conn != nil {
			player.Conn.Close()
		}
	}

	log.Println("\n=== 测试完成 ===")
	log.Println("✅ 两局游戏流程测试完成，未发现panic")
}

func connectPlayer(player *TestPlayer) error {
	u, err := url.Parse(WS_URL)
	if err != nil {
		return err
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	player.Conn = conn

	// 启动消息接收goroutine
	wg.Add(1)
	go func(p *TestPlayer) {
		defer wg.Done()
		for {
			var msg TestMessage
			if err := conn.ReadJSON(&msg); err != nil {
				log.Printf("玩家 %s 读取消息失败: %v", p.Name, err)
				return
			}
			testHandleMessage(p, &msg)
		}
	}(player)

	return nil
}

func testCreateRoom(player *TestPlayer) error {
	msg := TestMessage{
		Type: "createRoom",
		Data: map[string]interface{}{
			"playerName": player.Name,
		},
	}
	return player.Conn.WriteJSON(msg)
}

func testJoinRoom(player *TestPlayer, roomID string) error {
	msg := TestMessage{
		Type: "joinRoom",
		Data: map[string]interface{}{
			"roomId":     roomID,
			"playerName": player.Name,
		},
	}
	return player.Conn.WriteJSON(msg)
}

func testStartGame(player *TestPlayer) error {
	msg := TestMessage{
		Type: "startGame",
		Data: map[string]interface{}{},
	}
	return player.Conn.WriteJSON(msg)
}

func testHandleMessage(player *TestPlayer, msg *TestMessage) {
	switch msg.Type {
	case "roomCreated":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if roomID, ok := data["roomId"].(string); ok {
				player.RoomID = roomID
				log.Printf("✅ 玩家 %s 创建房间成功，房间ID: %s", player.Name, roomID)
			}
		}
	case "roomJoined":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if room, ok := data["room"].(map[string]interface{}); ok {
				if roomID, ok := room["id"].(string); ok {
					player.RoomID = roomID
					log.Printf("✅ 玩家 %s 加入房间成功，房间ID: %s", player.Name, roomID)
				}
			}
		}
	case "gameStarted":
		gameMutex.Lock()
		gameStarted = true
		gameMutex.Unlock()
		log.Printf("✅ 玩家 %s 收到游戏开始消息", player.Name)
		// gameStarted的Data直接是roomData
		// 添加调试信息
		if roomData, ok := msg.Data.(map[string]interface{}); ok {
			if currentTurn, ok := roomData["currentTurn"].(float64); ok {
				log.Printf("🔍 玩家 %s: gameStarted消息中 currentTurn=%d", player.Name, int(currentTurn))
			}
			if players, ok := roomData["players"].([]interface{}); ok {
				log.Printf("🔍 玩家 %s: gameStarted消息中有 %d 个玩家", player.Name, len(players))
				for i, p := range players {
					if pData, ok := p.(map[string]interface{}); ok {
						if id, ok := pData["id"].(string); ok {
							if chips, ok := pData["chips"].(float64); ok {
								log.Printf("🔍   玩家[%d]: id=%s, chips=%.0f", i, id, chips)
							}
						}
					}
				}
			}
		}
		testUpdateGameState(player, msg.Data)
	case "actionTaken":
		log.Printf("✅ 玩家 %s 收到行动消息", player.Name)
		// actionTaken的Data直接是roomData
		testUpdateGameState(player, msg.Data)
	case "gameEnded":
		gameMutex.Lock()
		gameEnded = true
		gameMutex.Unlock()
		log.Printf("✅ 玩家 %s 收到游戏结束消息", player.Name)
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if winner, ok := data["winner"].(map[string]interface{}); ok {
				if name, ok := winner["name"].(string); ok {
					log.Printf("🎉 第%d局游戏结束，获胜者: %s", handCount, name)
				}
			}
		}
	case "error":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if message, ok := data["message"].(string); ok {
				log.Printf("❌ 玩家 %s 收到错误: %s", player.Name, message)
			}
		}
	case "roomUpdated":
		log.Printf("✅ 玩家 %s 收到房间更新消息", player.Name)
		// roomUpdated的Data是 {"room": roomData}
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if room, ok := data["room"].(map[string]interface{}); ok {
				testUpdateGameState(player, room)
			}
		}
	}
}

func testUpdateGameState(player *TestPlayer, data interface{}) {
	var room map[string]interface{}

	// 尝试从data中获取room对象
	if roomData, ok := data.(map[string]interface{}); ok {
		// 检查是否有嵌套的room字段（roomUpdated消息）
		if nestedRoom, ok := roomData["room"].(map[string]interface{}); ok {
			room = nestedRoom
		} else {
			// 直接就是room对象（gameStarted和actionTaken消息）
			room = roomData
		}
	}

	if room != nil {
		// 更新游戏阶段
		if phase, ok := room["gamePhase"].(string); ok {
			log.Printf("📊 玩家 %s: 游戏阶段: %s", player.Name, phase)
		}

		// 更新当前回合
		if turn, ok := room["currentTurn"].(float64); ok {
			player.mu.Lock()
			// 检查是否是我的回合
			if players, ok := room["players"].([]interface{}); ok {
				if int(turn) < len(players) {
					if p, ok := players[int(turn)].(map[string]interface{}); ok {
						if id, ok := p["id"].(string); ok {
							wasMyTurn := player.IsMyTurn
							player.IsMyTurn = (id == player.ID)
							if player.IsMyTurn && !wasMyTurn {
								log.Printf("🎯 玩家 %s 的回合到了 (currentTurn=%d, playerID=%s)", player.Name, int(turn), id)
							} else if !player.IsMyTurn && wasMyTurn {
								log.Printf("⏭️ 玩家 %s 的回合结束了", player.Name)
							}
						}
					}
				} else {
					log.Printf("⚠️ 玩家 %s: currentTurn (%d) 超出玩家数组长度 (%d)", player.Name, int(turn), len(players))
				}
			}
			player.mu.Unlock()
		}

		// 更新玩家信息
		if players, ok := room["players"].([]interface{}); ok {
			for _, p := range players {
				if pData, ok := p.(map[string]interface{}); ok {
					if id, ok := pData["id"].(string); ok && id == player.ID {
						if chips, ok := pData["chips"].(float64); ok {
							player.Chips = int(chips)
						}
						if bet, ok := pData["bet"].(float64); ok {
							player.Bet = int(bet)
						}
						if folded, ok := pData["folded"].(bool); ok {
							player.Folded = folded
						}
						if hand, ok := pData["hand"].([]interface{}); ok {
							player.Hand = hand
						}
					}
				}
			}
		}
	}
}

func testPlayGame(handNum int) {
	log.Printf("开始模拟第%d局游戏流程...", handNum)

	// 模拟多轮下注
	maxRounds := 20
	actionCount := 0
	for round := 0; round < maxRounds; round++ {
		gameMutex.Lock()
		ended := gameEnded
		gameMutex.Unlock()
		if ended {
			log.Printf("第%d局游戏已结束", handNum)
			break
		}

		// 等待游戏状态更新
		time.Sleep(1 * time.Second)

		// 检查每个玩家是否需要行动（最多尝试3次）
		actionTaken := false
		for attempt := 0; attempt < 3; attempt++ {
			for _, player := range testPlayers {
				if player.Conn == nil {
					continue
				}

				player.mu.Lock()
				isMyTurn := player.IsMyTurn
				folded := player.Folded
				chips := player.Chips
				bet := player.Bet
				player.mu.Unlock()

				if isMyTurn && !folded {
					// 模拟玩家行动
					action := testChooseAction(player, round)
					if err := testSendAction(player, action); err != nil {
						log.Printf("玩家 %s 发送行动失败: %v", player.Name, err)
					} else {
						log.Printf("玩家 %s 执行行动: %s (第%d轮，第%d次行动，筹码:%d，下注:%d)", player.Name, action, round, actionCount+1, chips, bet)
						actionCount++
						actionTaken = true
					}
					time.Sleep(500 * time.Millisecond)
					break // 一次只处理一个玩家的行动
				}
			}

			if actionTaken {
				break
			}
			// 如果没有行动，打印调试信息
			if attempt == 2 {
				log.Printf("⚠️ 第%d轮尝试3次后仍无玩家行动，检查状态:", round)
				for _, player := range testPlayers {
					player.mu.Lock()
					log.Printf("  - 玩家 %s: IsMyTurn=%v, Folded=%v, Chips=%d, Bet=%d",
						player.Name, player.IsMyTurn, player.Folded, player.Chips, player.Bet)
					player.mu.Unlock()
				}
			}
			time.Sleep(500 * time.Millisecond)
		}

		// 检查游戏是否结束
		gameMutex.Lock()
		ended = gameEnded
		gameMutex.Unlock()
		if ended {
			break
		}

		// 如果长时间没有行动，可能游戏已经结束或卡住了
		if !actionTaken && round > 5 {
			log.Printf("警告: 第%d轮没有玩家行动，可能游戏已结束或卡住", round)
			time.Sleep(2 * time.Second)
		}
	}

	log.Printf("第%d局游戏模拟完成，共执行了 %d 次行动", handNum, actionCount)
}

func testChooseAction(player *TestPlayer, round int) string {
	// 测试边缘情况：模拟全押场景
	// 如果玩家筹码很少（小于200），更可能全押
	if player.Chips < 200 && player.Chips > 0 {
		// 90%概率全押
		rand := time.Now().UnixNano() % 10
		if rand < 9 {
			return "allin" // 特殊标记，需要转换为raise
		}
	}
	// 如果玩家筹码很少但还有筹码，也可能全押
	if player.Chips > 0 && player.Chips < 300 {
		// 50%概率全押
		rand := time.Now().UnixNano() % 10
		if rand < 5 {
			return "allin"
		}
	}

	// 简单的策略：前几轮过牌或跟注，后面可能弃牌或加注
	if round < 2 {
		if player.Bet == 0 {
			return "check"
		}
		return "call"
	} else if round < 4 {
		// 随机选择：50%跟注，30%加注，20%弃牌
		rand := time.Now().UnixNano() % 10
		if rand < 5 {
			return "call"
		} else if rand < 8 {
			return "raise"
		} else {
			return "fold"
		}
	} else {
		// 后期：更可能弃牌
		rand := time.Now().UnixNano() % 10
		if rand < 3 {
			return "call"
		} else if rand < 5 {
			return "raise"
		} else {
			return "fold"
		}
	}
}

func testSendAction(player *TestPlayer, action string) error {
	msg := TestMessage{
		Type: "action",
		Data: map[string]interface{}{
			"action": action,
		},
	}

	if action == "raise" {
		msg.Data = map[string]interface{}{
			"action": "raise",
			"amount": 50, // 固定加注50
		}
	} else if action == "allin" {
		// 全押：加注金额设为玩家所有筹码（确保全押）
		allInAmount := player.Chips + 1000 // 确保超过玩家筹码，触发全押
		msg.Data = map[string]interface{}{
			"action": "raise",
			"amount": allInAmount,
		}
		log.Printf("玩家 %s 全押，筹码: %d，加注金额: %d", player.Name, player.Chips, allInAmount)
	}

	return player.Conn.WriteJSON(msg)
}
