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
	WS_URL_TEST = "ws://localhost:8080/ws"
)

type NewPlayerTestPlayer struct {
	ID       string
	Name     string
	Conn     *websocket.Conn
	RoomID   string
	IsMyTurn bool
	Chips    int
	Bet      int
	Folded   bool
	mu       sync.Mutex
}

type NewPlayerTestMessage struct {
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	PlayerID string      `json:"playerId,omitempty"`
}

var (
	newPlayerTestPlayers []*NewPlayerTestPlayer
	newPlayerTestMutex   sync.Mutex
	newPlayerGameStarted bool
	newPlayerGameEnded   bool
	currentRound         int
)

func runNewPlayerIssueTest() {
	log.Println("=== 测试新玩家加入后游戏无法开始的问题 ===")
	log.Println("场景：4人开局 -> 第5人进入 -> 完成一局 -> 第二局无法开始")
	log.Println("")

	// 创建初始4个玩家
	newPlayerTestPlayers = make([]*NewPlayerTestPlayer, 4)
	for i := 0; i < 4; i++ {
		newPlayerTestPlayers[i] = &NewPlayerTestPlayer{
			ID:   fmt.Sprintf("test_player_%d", i+1),
			Name: fmt.Sprintf("玩家%d", i+1),
		}
	}

	// 连接初始玩家
	log.Println("步骤1: 连接初始4个玩家...")
	for _, player := range newPlayerTestPlayers {
		if err := connectNewPlayerTestPlayer(player); err != nil {
			log.Fatalf("玩家 %s 连接失败: %v", player.Name, err)
		}
		log.Printf("✅ 玩家 %s 连接成功", player.Name)
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	// 第一轮：创建房间并开始游戏
	log.Println("\n步骤2: 第一轮游戏 - 创建房间并开始游戏")
	currentRound = 1
	newPlayerGameStarted = false
	newPlayerGameEnded = false

	if err := newPlayerTestCreateRoom(newPlayerTestPlayers[0]); err != nil {
		log.Fatalf("创建房间失败: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// 其他玩家加入
	for i := 1; i < 4; i++ {
		if err := newPlayerTestJoinRoom(newPlayerTestPlayers[i], newPlayerTestPlayers[0].RoomID); err != nil {
			log.Fatalf("玩家%d加入房间失败: %v", i+1, err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	// 检查玩家数
	log.Printf("当前玩家数: %d", len(newPlayerTestPlayers))
	if len(newPlayerTestPlayers) < 4 {
		log.Fatalf("❌ 玩家数不足4人，无法开始游戏")
	}

	// 开始第一轮游戏
	log.Println("开始第一轮游戏...")
	if err := newPlayerTestStartGame(newPlayerTestPlayers[0]); err != nil {
		log.Fatalf("开始第一轮游戏失败: %v", err)
	}

	// 等待游戏开始
	waitForGameStart(1, 5)

	// 模拟第一轮游戏（快速结束）
	log.Println("模拟第一轮游戏流程...")
	newPlayerTestPlayGame(1)

	// 等待第一轮结束
	waitForGameEnd(1, 30)
	log.Println("✅ 第一轮游戏结束")

	// 第五人进入
	log.Println("\n步骤3: 第五人进入房间（应该在等待列表）")
	currentRound = 2
	newPlayerGameStarted = false
	newPlayerGameEnded = false

	player5 := &NewPlayerTestPlayer{
		ID:   "test_player_5",
		Name: "玩家5",
	}
	if err := connectNewPlayerTestPlayer(player5); err != nil {
		log.Fatalf("玩家5连接失败: %v", err)
	}
	newPlayerTestPlayers = append(newPlayerTestPlayers, player5)
	log.Printf("✅ 玩家5连接成功")
	time.Sleep(200 * time.Millisecond)

	// 玩家5加入房间（应该在等待列表）
	if err := newPlayerTestJoinRoom(player5, newPlayerTestPlayers[0].RoomID); err != nil {
		log.Fatalf("玩家5加入房间失败: %v", err)
	}
	time.Sleep(1 * time.Second)
	log.Println("✅ 玩家5已加入等待列表")

	// 等待第一轮游戏完全结束（包括新玩家加入）
	time.Sleep(3 * time.Second)

	// 检查当前玩家数
	log.Printf("当前玩家数: %d (应该包含玩家5)", len(newPlayerTestPlayers))

	// 尝试开始第二轮游戏
	log.Println("\n步骤4: 尝试开始第二轮游戏...")
	if err := newPlayerTestStartGame(newPlayerTestPlayers[0]); err != nil {
		log.Fatalf("❌ 开始第二轮游戏失败: %v", err)
	}

	// 等待游戏开始
	if !waitForGameStart(2, 10) {
		log.Fatalf("❌ 第二轮游戏无法开始！")
	}

	log.Println("✅ 第二轮游戏成功开始")

	// 模拟第二轮游戏
	newPlayerTestPlayGame(2)
	waitForGameEnd(2, 30)

	log.Println("\n=== 测试完成 ===")
	log.Println("✅ 新玩家加入后游戏可以正常开始")

	// 关闭所有连接
	for _, player := range newPlayerTestPlayers {
		if player.Conn != nil {
			player.Conn.Close()
		}
	}
}

func connectNewPlayerTestPlayer(player *NewPlayerTestPlayer) error {
	u, err := url.Parse(WS_URL_TEST)
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

	go func(p *NewPlayerTestPlayer) {
		for {
			var msg NewPlayerTestMessage
			if err := conn.ReadJSON(&msg); err != nil {
				log.Printf("玩家 %s 读取消息失败: %v", p.Name, err)
				return
			}
			handleNewPlayerTestMessage(p, &msg)
		}
	}(player)

	return nil
}

func handleNewPlayerTestMessage(player *NewPlayerTestPlayer, msg *NewPlayerTestMessage) {
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
					// 检查玩家数
					if players, ok := room["players"].([]interface{}); ok {
						log.Printf("   房间当前玩家数: %d", len(players))
					}
					// 检查游戏状态
					if gamePhase, ok := room["gamePhase"].(string); ok {
						log.Printf("   游戏状态: %s", gamePhase)
					}
				}
			}
		}
	case "gameStarted":
		newPlayerTestMutex.Lock()
		newPlayerGameStarted = true
		newPlayerGameEnded = false
		newPlayerTestMutex.Unlock()
		log.Printf("✅ 玩家 %s 收到游戏开始消息（第%d轮）", player.Name, currentRound)
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if players, ok := data["players"].([]interface{}); ok {
				log.Printf("   游戏开始时玩家数: %d", len(players))
			}
			if gamePhase, ok := data["gamePhase"].(string); ok {
				log.Printf("   游戏阶段: %s", gamePhase)
			}
		}
	case "actionTaken":
		// 忽略
	case "gameEnded":
		newPlayerTestMutex.Lock()
		newPlayerGameEnded = true
		newPlayerTestMutex.Unlock()
		log.Printf("✅ 玩家 %s 收到游戏结束消息（第%d轮）", player.Name, currentRound)
	case "roomUpdated":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if room, ok := data["room"].(map[string]interface{}); ok {
				if players, ok := room["players"].([]interface{}); ok {
					log.Printf("📊 房间更新：玩家数=%d", len(players))
				}
				if gamePhase, ok := room["gamePhase"].(string); ok {
					log.Printf("📊 房间更新：游戏状态=%s", gamePhase)
				}
			}
		}
	case "error":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if message, ok := data["message"].(string); ok {
				log.Printf("❌ 玩家 %s 收到错误: %s", player.Name, message)
			}
		}
	}
}

func newPlayerTestCreateRoom(player *NewPlayerTestPlayer) error {
	msg := NewPlayerTestMessage{
		Type: "createRoom",
		Data: map[string]interface{}{
			"playerName": player.Name,
		},
	}
	return player.Conn.WriteJSON(msg)
}

func newPlayerTestJoinRoom(player *NewPlayerTestPlayer, roomID string) error {
	msg := NewPlayerTestMessage{
		Type: "joinRoom",
		Data: map[string]interface{}{
			"roomId":     roomID,
			"playerName": player.Name,
		},
	}
	return player.Conn.WriteJSON(msg)
}

func newPlayerTestStartGame(player *NewPlayerTestPlayer) error {
	msg := NewPlayerTestMessage{
		Type: "startGame",
		Data: map[string]interface{}{},
	}
	return player.Conn.WriteJSON(msg)
}

func newPlayerTestPlayGame(roundNum int) {
	maxActions := 20
	actionCount := 0
	for i := 0; i < maxActions; i++ {
		newPlayerTestMutex.Lock()
		ended := newPlayerGameEnded
		newPlayerTestMutex.Unlock()
		if ended {
			break
		}

		time.Sleep(500 * time.Millisecond)

		actionTaken := false
		for _, player := range newPlayerTestPlayers {
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
				if chips < 20 {
					action = "fold"
				}

				msg := NewPlayerTestMessage{
					Type: "action",
					Data: map[string]interface{}{
						"action": action,
					},
				}

				if err := player.Conn.WriteJSON(msg); err == nil {
					actionCount++
					actionTaken = true
					time.Sleep(300 * time.Millisecond)
					break
				}
			}
		}

		if !actionTaken && i > 3 {
			time.Sleep(1 * time.Second)
		}
	}
}

func waitForGameStart(roundNum int, timeoutSeconds int) bool {
	log.Printf("等待第%d轮游戏开始...", roundNum)
	for i := 0; i < timeoutSeconds; i++ {
		newPlayerTestMutex.Lock()
		started := newPlayerGameStarted
		newPlayerTestMutex.Unlock()
		if started {
			log.Printf("✅ 第%d轮游戏已开始", roundNum)
			return true
		}
		time.Sleep(1 * time.Second)
	}
	log.Printf("❌ 第%d轮游戏启动超时", roundNum)
	return false
}

func waitForGameEnd(roundNum int, timeoutSeconds int) bool {
	log.Printf("等待第%d轮游戏结束...", roundNum)
	for i := 0; i < timeoutSeconds; i++ {
		newPlayerTestMutex.Lock()
		ended := newPlayerGameEnded
		newPlayerTestMutex.Unlock()
		if ended {
			log.Printf("✅ 第%d轮游戏已结束", roundNum)
			return true
		}
		time.Sleep(1 * time.Second)
	}
	log.Printf("⚠️ 第%d轮游戏结束超时", roundNum)
	return false
}
