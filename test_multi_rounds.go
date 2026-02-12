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

type MultiRoundTestPlayer struct {
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

type MultiRoundTestMessage struct {
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	PlayerID string      `json:"playerId,omitempty"`
}

var (
	multiRoundPlayers []*MultiRoundTestPlayer
	multiRoundWg     sync.WaitGroup
	multiRoundMutex  sync.Mutex
	gameStarted      bool
	gameEnded        bool
	roundCount       int
	maxRounds         = 4
)

func runMultiRoundTest() {
	log.Println("=== 开始多轮游戏测试（包含新玩家加入和退出） ===")

	// 创建初始4个玩家
	multiRoundPlayers = make([]*MultiRoundTestPlayer, 4)
	for i := 0; i < 4; i++ {
		multiRoundPlayers[i] = &MultiRoundTestPlayer{
			ID:   fmt.Sprintf("player_%d", i+1),
			Name: fmt.Sprintf("玩家%d", i+1),
		}
	}

	// 连接初始玩家
	log.Println("正在连接初始4个玩家...")
	for _, player := range multiRoundPlayers {
		if err := connectMultiRoundPlayer(player); err != nil {
			log.Fatalf("玩家 %s 连接失败: %v", player.Name, err)
		}
		log.Printf("✅ 玩家 %s 连接成功", player.Name)
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	// 第一轮：创建房间并开始游戏
	log.Println("\n=== 第一轮游戏 ===")
	roundCount = 1
	if err := testMultiRoundCreateRoom(multiRoundPlayers[0]); err != nil {
		log.Fatalf("创建房间失败: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// 其他玩家加入
	for i := 1; i < 4; i++ {
		if err := testMultiRoundJoinRoom(multiRoundPlayers[i], multiRoundPlayers[0].RoomID); err != nil {
			log.Fatalf("玩家%d加入房间失败: %v", i+1, err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	// 开始第一轮游戏
	if err := testMultiRoundStartGame(multiRoundPlayers[0]); err != nil {
		log.Fatalf("开始第一轮游戏失败: %v", err)
	}
	time.Sleep(1 * time.Second)

	// 模拟第一轮游戏
	testMultiRoundPlayGame(1)

	// 等待第一轮结束
	waitForGameEnd(1)

	// 第二轮：添加新玩家
	log.Println("\n=== 第二轮游戏（添加新玩家） ===")
	roundCount = 2
	time.Sleep(2 * time.Second)

	// 添加新玩家
	newPlayer1 := &MultiRoundTestPlayer{
		ID:   "player_5",
		Name: "新玩家1",
	}
	if err := connectMultiRoundPlayer(newPlayer1); err != nil {
		log.Fatalf("新玩家1连接失败: %v", err)
	}
	multiRoundPlayers = append(multiRoundPlayers, newPlayer1)
	time.Sleep(200 * time.Millisecond)

	// 新玩家加入房间（应该在等待列表）
	if err := testMultiRoundJoinRoom(newPlayer1, multiRoundPlayers[0].RoomID); err != nil {
		log.Fatalf("新玩家1加入房间失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 开始第二轮游戏（新玩家应该自动加入）
	if err := testMultiRoundStartGame(multiRoundPlayers[0]); err != nil {
		log.Fatalf("开始第二轮游戏失败: %v", err)
	}
	time.Sleep(1 * time.Second)

	testMultiRoundPlayGame(2)
	waitForGameEnd(2)

	// 第三轮：玩家退出并添加新玩家
	log.Println("\n=== 第三轮游戏（玩家退出并添加新玩家） ===")
	roundCount = 3
	time.Sleep(2 * time.Second)

	// 模拟玩家退出（关闭连接）
	log.Println("玩家3退出游戏")
	if multiRoundPlayers[2].Conn != nil {
		multiRoundPlayers[2].Conn.Close()
		multiRoundPlayers[2].Conn = nil
	}
	time.Sleep(500 * time.Millisecond)

	// 添加新玩家
	newPlayer2 := &MultiRoundTestPlayer{
		ID:   "player_6",
		Name: "新玩家2",
	}
	if err := connectMultiRoundPlayer(newPlayer2); err != nil {
		log.Fatalf("新玩家2连接失败: %v", err)
	}
	multiRoundPlayers = append(multiRoundPlayers, newPlayer2)
	time.Sleep(200 * time.Millisecond)

	if err := testMultiRoundJoinRoom(newPlayer2, multiRoundPlayers[0].RoomID); err != nil {
		log.Fatalf("新玩家2加入房间失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 开始第三轮游戏
	if err := testMultiRoundStartGame(multiRoundPlayers[0]); err != nil {
		log.Fatalf("开始第三轮游戏失败: %v", err)
	}
	time.Sleep(1 * time.Second)

	testMultiRoundPlayGame(3)
	waitForGameEnd(3)

	// 第四轮：再次添加新玩家
	log.Println("\n=== 第四轮游戏（再次添加新玩家） ===")
	roundCount = 4
	time.Sleep(2 * time.Second)

	newPlayer3 := &MultiRoundTestPlayer{
		ID:   "player_7",
		Name: "新玩家3",
	}
	if err := connectMultiRoundPlayer(newPlayer3); err != nil {
		log.Fatalf("新玩家3连接失败: %v", err)
	}
	multiRoundPlayers = append(multiRoundPlayers, newPlayer3)
	time.Sleep(200 * time.Millisecond)

	if err := testMultiRoundJoinRoom(newPlayer3, multiRoundPlayers[0].RoomID); err != nil {
		log.Fatalf("新玩家3加入房间失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 开始第四轮游戏
	if err := testMultiRoundStartGame(multiRoundPlayers[0]); err != nil {
		log.Fatalf("开始第四轮游戏失败: %v", err)
	}
	time.Sleep(1 * time.Second)

	testMultiRoundPlayGame(4)
	waitForGameEnd(4)

	// 关闭所有连接
	log.Println("\n=== 关闭所有连接 ===")
	for _, player := range multiRoundPlayers {
		if player.Conn != nil {
			player.Conn.Close()
		}
	}

	log.Println("\n=== 多轮游戏测试完成 ===")
	log.Println("✅ 4轮游戏测试完成，包含新玩家加入和退出场景")
}

func connectMultiRoundPlayer(player *MultiRoundTestPlayer) error {
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

	multiRoundWg.Add(1)
	go func(p *MultiRoundTestPlayer) {
		defer multiRoundWg.Done()
		for {
			var msg MultiRoundTestMessage
			if err := conn.ReadJSON(&msg); err != nil {
				log.Printf("玩家 %s 读取消息失败: %v", p.Name, err)
				return
			}
			handleMultiRoundMessage(p, &msg)
		}
	}(player)

	return nil
}

func handleMultiRoundMessage(player *MultiRoundTestPlayer, msg *MultiRoundTestMessage) {
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
					isWaiting, _ := data["isWaiting"].(bool)
					if isWaiting {
						log.Printf("✅ 玩家 %s 加入等待列表，房间ID: %s", player.Name, roomID)
					} else {
						log.Printf("✅ 玩家 %s 加入房间成功，房间ID: %s", player.Name, roomID)
					}
				}
			}
		}
	case "gameStarted":
		multiRoundMutex.Lock()
		gameStarted = true
		gameEnded = false
		multiRoundMutex.Unlock()
		log.Printf("✅ 玩家 %s 收到游戏开始消息（第%d轮）", player.Name, roundCount)
		updateMultiRoundGameState(player, msg.Data)
	case "actionTaken":
		updateMultiRoundGameState(player, msg.Data)
	case "gameEnded":
		multiRoundMutex.Lock()
		gameEnded = true
		multiRoundMutex.Unlock()
		log.Printf("✅ 玩家 %s 收到游戏结束消息（第%d轮）", player.Name, roundCount)
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if winner, ok := data["winner"].(map[string]interface{}); ok {
				if name, ok := winner["name"].(string); ok {
					log.Printf("🎉 第%d轮游戏结束，获胜者: %s", roundCount, name)
				}
			}
		}
	case "roomUpdated":
		updateMultiRoundGameState(player, msg.Data)
	}
}

func updateMultiRoundGameState(player *MultiRoundTestPlayer, data interface{}) {
	var room map[string]interface{}

	if roomData, ok := data.(map[string]interface{}); ok {
		if nestedRoom, ok := roomData["room"].(map[string]interface{}); ok {
			room = nestedRoom
		} else {
			room = roomData
		}
	}

	if room != nil {
		if players, ok := room["players"].([]interface{}); ok {
			for _, p := range players {
				if pData, ok := p.(map[string]interface{}); ok {
					if id, ok := pData["id"].(string); ok && id == player.ID {
						if chips, ok := pData["chips"].(float64); ok {
							player.mu.Lock()
							player.Chips = int(chips)
							player.mu.Unlock()
						}
						if bet, ok := pData["bet"].(float64); ok {
							player.mu.Lock()
							player.Bet = int(bet)
							player.mu.Unlock()
						}
						if folded, ok := pData["folded"].(bool); ok {
							player.mu.Lock()
							player.Folded = folded
							player.mu.Unlock()
						}
					}
				}
			}
		}

		if turn, ok := room["currentTurn"].(float64); ok {
			if players, ok := room["players"].([]interface{}); ok {
				if int(turn) < len(players) {
					if p, ok := players[int(turn)].(map[string]interface{}); ok {
						if id, ok := p["id"].(string); ok {
							player.mu.Lock()
							player.IsMyTurn = (id == player.ID)
							player.mu.Unlock()
						}
					}
				}
			}
		}
	}
}

func testMultiRoundCreateRoom(player *MultiRoundTestPlayer) error {
	msg := MultiRoundTestMessage{
		Type: "createRoom",
		Data: map[string]interface{}{
			"playerName": player.Name,
		},
	}
	return player.Conn.WriteJSON(msg)
}

func testMultiRoundJoinRoom(player *MultiRoundTestPlayer, roomID string) error {
	msg := MultiRoundTestMessage{
		Type: "joinRoom",
		Data: map[string]interface{}{
			"roomId":     roomID,
			"playerName": player.Name,
		},
	}
	return player.Conn.WriteJSON(msg)
}

func testMultiRoundStartGame(player *MultiRoundTestPlayer) error {
	msg := MultiRoundTestMessage{
		Type: "startGame",
		Data: map[string]interface{}{},
	}
	return player.Conn.WriteJSON(msg)
}

func testMultiRoundPlayGame(roundNum int) {
	log.Printf("开始模拟第%d轮游戏流程...", roundNum)

	maxActions := 30
	actionCount := 0
	for i := 0; i < maxActions; i++ {
		multiRoundMutex.Lock()
		ended := gameEnded
		multiRoundMutex.Unlock()
		if ended {
			break
		}

		time.Sleep(1 * time.Second)

		actionTaken := false
		for _, player := range multiRoundPlayers {
			if player.Conn == nil {
				continue
			}

			player.mu.Lock()
			isMyTurn := player.IsMyTurn
			folded := player.Folded
			chips := player.Chips
			player.mu.Unlock()

			if isMyTurn && !folded && chips > 0 {
				action := "call"
				if chips < 50 {
					action = "fold"
				}

				msg := MultiRoundTestMessage{
					Type: "action",
					Data: map[string]interface{}{
						"action": action,
					},
				}

				if err := player.Conn.WriteJSON(msg); err == nil {
					log.Printf("玩家 %s 执行行动: %s (第%d轮)", player.Name, action, roundNum)
					actionCount++
					actionTaken = true
					time.Sleep(500 * time.Millisecond)
					break
				}
			}
		}

		if !actionTaken && i > 5 {
			time.Sleep(2 * time.Second)
		}
	}

	log.Printf("第%d轮游戏模拟完成，共执行了 %d 次行动", roundNum, actionCount)
}

func waitForGameEnd(roundNum int) {
	log.Printf("等待第%d轮游戏结束...", roundNum)
	for i := 0; i < 30; i++ {
		multiRoundMutex.Lock()
		ended := gameEnded
		multiRoundMutex.Unlock()
		if ended {
			log.Printf("✅ 第%d轮游戏已结束", roundNum)
			return
		}
		time.Sleep(1 * time.Second)
	}
	log.Printf("⚠️ 第%d轮游戏超时", roundNum)
}
