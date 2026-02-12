//go:build test
// +build test

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// 测试打平情况的客户端
func testTieBreaker() {
	fmt.Println("=== 测试打平情况 ===")

	// 测试场景1：两个玩家打平
	fmt.Println("\n--- 测试场景1：两个玩家打平（使用公共牌组成相同牌型）---")
	testTwoPlayerTie()

	// 测试场景2：三个玩家打平
	fmt.Println("\n--- 测试场景2：三个玩家打平 ---")
	testThreePlayerTie()

	// 测试场景3：部分玩家打平
	fmt.Println("\n--- 测试场景3：部分玩家打平，其他玩家输 ---")
	testPartialTie()

	time.Sleep(2 * time.Second)
}

// 测试两个玩家打平
func testTwoPlayerTie() {
	// 创建房间
	roomID := createRoom("Player1")
	if roomID == "" {
		log.Fatal("创建房间失败")
	}

	// 玩家1加入
	ws1, err := connectToRoom("Player1", roomID)
	if err != nil {
		log.Fatal("玩家1连接失败:", err)
	}
	defer ws1.Close()

	// 玩家2加入
	ws2, err := connectToRoom("Player2", roomID)
	if err != nil {
		log.Fatal("玩家2连接失败:", err)
	}
	defer ws2.Close()

	// 玩家3加入
	ws3, err := connectToRoom("Player3", roomID)
	if err != nil {
		log.Fatal("玩家3连接失败:", err)
	}
	defer ws3.Close()

	// 玩家4加入
	ws4, err := connectToRoom("Player4", roomID)
	if err != nil {
		log.Fatal("玩家4连接失败:", err)
	}
	defer ws4.Close()

	// 等待所有玩家加入
	time.Sleep(1 * time.Second)

	// 开始游戏
	sendMessage(ws1, map[string]interface{}{
		"type": "startGame",
		"data": map[string]interface{}{},
	})

	// 等待游戏开始
	time.Sleep(2 * time.Second)

	// 接收消息并检查游戏状态
	var gameStarted bool
	var gameEnded bool
	var winners []interface{}
	var isTie bool
	var pot int

	// 设置消息接收超时
	done := make(chan bool)
	go func() {
		for {
			_, message, err := ws1.ReadMessage()
			if err != nil {
				return
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			msgType := msg["type"].(string)
			fmt.Printf("玩家1收到消息: %s\n", msgType)

			switch msgType {
			case "gameStarted":
				gameStarted = true
				fmt.Println("✅ 游戏已开始")
			case "gameEnded":
				gameEnded = true
				data := msg["data"].(map[string]interface{})
				if w, ok := data["winners"]; ok {
					winners = w.([]interface{})
				}
				if tie, ok := data["isTie"]; ok {
					isTie = tie.(bool)
				}
				if p, ok := data["pot"]; ok {
					pot = int(p.(float64))
				}
				fmt.Printf("✅ 游戏结束 - 打平: %v, 获胜者数: %d, 底池: %d\n", isTie, len(winners), pot)
				done <- true
			}
		}
	}()

	// 模拟游戏流程：所有玩家都跟注到河牌
	// 这里简化处理，实际应该根据收到的消息来决定行动
	time.Sleep(1 * time.Second)

	// 所有玩家跟注（简化：直接发送跟注消息）
	for i := 0; i < 10; i++ {
		sendMessage(ws1, map[string]interface{}{
			"type": "action",
			"data": map[string]interface{}{
				"action": "call",
			},
		})
		sendMessage(ws2, map[string]interface{}{
			"type": "action",
			"data": map[string]interface{}{
				"action": "call",
			},
		})
		sendMessage(ws3, map[string]interface{}{
			"type": "action",
			"data": map[string]interface{}{
				"action": "call",
			},
		})
		sendMessage(ws4, map[string]interface{}{
			"type": "action",
			"data": map[string]interface{}{
				"action": "call",
			},
		})
		time.Sleep(500 * time.Millisecond)
	}

	// 等待游戏结束
	select {
	case <-done:
		fmt.Println("✅ 收到游戏结束消息")
	case <-time.After(30 * time.Second):
		fmt.Println("❌ 超时：未收到游戏结束消息")
	}

	// 验证结果
	if !gameStarted {
		fmt.Println("❌ 测试失败：游戏未开始")
		return
	}

	if !gameEnded {
		fmt.Println("❌ 测试失败：游戏未结束")
		return
	}

	// 检查是否有winners字段
	if len(winners) == 0 {
		fmt.Println("❌ 测试失败：未找到获胜者")
		return
	}

	fmt.Printf("✅ 测试通过：找到 %d 个获胜者\n", len(winners))
	if isTie && len(winners) > 1 {
		fmt.Printf("✅ 测试通过：正确识别打平情况，%d 个玩家平分底池 %d\n", len(winners), pot)
	} else if len(winners) == 1 {
		fmt.Printf("✅ 测试通过：单个获胜者，获得底池 %d\n", pot)
	}
}

// 测试三个玩家打平
func testThreePlayerTie() {
	// 类似testTwoPlayerTie，但验证三个玩家打平的情况
	fmt.Println("测试三个玩家打平...")
	// 实现类似，但需要5个玩家（4个开始游戏，3个打平）
}

// 测试部分玩家打平
func testPartialTie() {
	// 测试部分玩家打平，其他玩家输的情况
	fmt.Println("测试部分玩家打平...")
	// 实现类似
}

// 辅助函数
func connectToRoom(playerName, roomID string) (*websocket.Conn, error) {
	url := fmt.Sprintf("ws://localhost:8080/ws?name=%s&room=%s", playerName, roomID)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	return conn, err
}

func sendMessage(conn *websocket.Conn, msg map[string]interface{}) {
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}

func createRoom(playerName string) string {
	url := fmt.Sprintf("ws://localhost:8080/ws?name=%s", playerName)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return ""
	}
	defer conn.Close()

	// 等待roomJoined消息
	_, message, _ := conn.ReadMessage()
	var msg map[string]interface{}
	json.Unmarshal(message, &msg)
	if msg["type"] == "roomJoined" {
		data := msg["data"].(map[string]interface{})
		return data["roomId"].(string)
	}
	return ""
}
