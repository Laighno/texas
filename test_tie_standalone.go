//go:build tie_test
// +build tie_test

package main

import (
	"fmt"
	"sync"
	"time"
)

// 测试打平逻辑的独立程序
func main() {
	fmt.Println("=== 测试打平逻辑 ===")

	// 创建测试房间
	room := &GameRoom{
		ID:        "test-tie-room",
		Players:   []*Player{},
		Pot:       100,
		GamePhase: "showdown",
		Mutex:     sync.RWMutex{},
	}

	// 创建测试玩家
	player1 := &Player{
		ID:     "player1",
		Name:   "玩家1",
		Chips:  500,
		Hand:   []Card{{Rank: "A", Suit: "spades"}, {Rank: "K", Suit: "spades"}},
		Bet:    25,
		Folded: false,
	}

	player2 := &Player{
		ID:     "player2",
		Name:   "玩家2",
		Chips:  500,
		Hand:   []Card{{Rank: "A", Suit: "hearts"}, {Rank: "K", Suit: "hearts"}},
		Bet:    25,
		Folded: false,
	}

	player3 := &Player{
		ID:     "player3",
		Name:   "玩家3",
		Chips:  500,
		Hand:   []Card{{Rank: "Q", Suit: "spades"}, {Rank: "J", Suit: "spades"}},
		Bet:    25,
		Folded: false,
	}

	// 设置公共牌（让玩家1和玩家2打平）
	room.CommunityCards = []Card{
		{Rank: "10", Suit: "spades"},
		{Rank: "9", Suit: "spades"},
		{Rank: "8", Suit: "spades"},
		{Rank: "7", Suit: "spades"},
		{Rank: "6", Suit: "spades"},
	}

	room.Players = []*Player{player1, player2, player3}

	// 测试场景1：两个玩家打平
	fmt.Println("\n--- 测试场景1：两个玩家打平（同花顺）---")
	testTieScenario1(room)

	// 重置
	player1.Chips = 500
	player2.Chips = 500
	player3.Chips = 500
	room.Pot = 100

	// 测试场景2：三个玩家打平
	fmt.Println("\n--- 测试场景2：三个玩家打平 ---")
	testTieScenario2(room)

	// 重置
	player1.Chips = 500
	player2.Chips = 500
	player3.Chips = 500
	room.Pot = 100

	// 测试场景3：单个获胜者
	fmt.Println("\n--- 测试场景3：单个获胜者 ---")
	testSingleWinner(room)

	time.Sleep(1 * time.Second)
}

// 测试两个玩家打平
func testTieScenario1(room *GameRoom) {
	// 设置公共牌为同花顺，让玩家1和玩家2都使用公共牌
	room.CommunityCards = []Card{
		{Rank: "10", Suit: "spades"},
		{Rank: "9", Suit: "spades"},
		{Rank: "8", Suit: "spades"},
		{Rank: "7", Suit: "spades"},
		{Rank: "6", Suit: "spades"},
	}

	player1 := room.Players[0]
	player2 := room.Players[1]
	player3 := room.Players[2]

	// 玩家3弃牌
	player3.Folded = true

	initialChips1 := player1.Chips
	initialChips2 := player2.Chips
	pot := room.Pot

	// 调用determineWinner
	room.Mutex.Lock()
	determineWinner(room)
	// determineWinner已经释放了锁

	// 验证结果
	fmt.Printf("底池: %d\n", pot)
	fmt.Printf("玩家1初始筹码: %d, 现在: %d, 增加: %d\n", initialChips1, player1.Chips, player1.Chips-initialChips1)
	fmt.Printf("玩家2初始筹码: %d, 现在: %d, 增加: %d\n", initialChips2, player2.Chips, player2.Chips-initialChips2)

	expectedShare := pot / 2
	expectedRemainder := pot % 2

	if player1.Chips-initialChips1 == expectedShare+expectedRemainder && 
	   player2.Chips-initialChips2 == expectedShare {
		fmt.Println("✅ 测试通过：两个玩家正确平分底池")
		fmt.Printf("   玩家1获得: %d (包含余数 %d)\n", expectedShare+expectedRemainder, expectedRemainder)
		fmt.Printf("   玩家2获得: %d\n", expectedShare)
	} else {
		fmt.Println("❌ 测试失败：底池分配不正确")
		fmt.Printf("   期望：玩家1获得 %d，玩家2获得 %d\n", expectedShare+expectedRemainder, expectedShare)
		fmt.Printf("   实际：玩家1获得 %d，玩家2获得 %d\n", player1.Chips-initialChips1, player2.Chips-initialChips2)
	}

	// 验证总分配
	totalDistributed := (player1.Chips - initialChips1) + (player2.Chips - initialChips2)
	if totalDistributed == pot {
		fmt.Println("✅ 测试通过：底池完全分配")
	} else {
		fmt.Printf("❌ 测试失败：底池分配不完整，期望: %d, 实际: %d\n", pot, totalDistributed)
	}
}

// 测试三个玩家打平
func testTieScenario2(room *GameRoom) {
	// 设置公共牌，让三个玩家都使用公共牌（打平）
	room.CommunityCards = []Card{
		{Rank: "A", Suit: "spades"},
		{Rank: "K", Suit: "spades"},
		{Rank: "Q", Suit: "spades"},
		{Rank: "J", Suit: "spades"},
		{Rank: "10", Suit: "spades"},
	}

	player1 := room.Players[0]
	player2 := room.Players[1]
	player3 := room.Players[2]

	// 所有玩家都不弃牌
	player1.Folded = false
	player2.Folded = false
	player3.Folded = false

	initialChips1 := player1.Chips
	initialChips2 := player2.Chips
	initialChips3 := player3.Chips
	pot := room.Pot

	// 调用determineWinner
	room.Mutex.Lock()
	determineWinner(room)

	// 验证结果
	fmt.Printf("底池: %d\n", pot)
	fmt.Printf("玩家1初始筹码: %d, 现在: %d, 增加: %d\n", initialChips1, player1.Chips, player1.Chips-initialChips1)
	fmt.Printf("玩家2初始筹码: %d, 现在: %d, 增加: %d\n", initialChips2, player2.Chips, player2.Chips-initialChips2)
	fmt.Printf("玩家3初始筹码: %d, 现在: %d, 增加: %d\n", initialChips3, player3.Chips, player3.Chips-initialChips3)

	expectedShare := pot / 3
	expectedRemainder := pot % 3

	totalDistributed := (player1.Chips - initialChips1) + (player2.Chips - initialChips2) + (player3.Chips - initialChips3)
	if totalDistributed == pot {
		fmt.Println("✅ 测试通过：三个玩家正确分配底池")
		fmt.Printf("   每人应得: %d，余数: %d\n", expectedShare, expectedRemainder)
	} else {
		fmt.Printf("❌ 测试失败：底池分配不完整，期望: %d, 实际: %d\n", pot, totalDistributed)
	}
}

// 测试单个获胜者
func testSingleWinner(room *GameRoom) {
	// 设置公共牌，让玩家1获胜
	room.CommunityCards = []Card{
		{Rank: "10", Suit: "spades"},
		{Rank: "9", Suit: "spades"},
		{Rank: "8", Suit: "spades"},
		{Rank: "7", Suit: "hearts"},
		{Rank: "6", Suit: "spades"},
	}

	player1 := room.Players[0]
	player2 := room.Players[1]
	player3 := room.Players[2]

	// 玩家1有更好的牌（同花），玩家2和3弃牌
	player1.Folded = false
	player2.Folded = true
	player3.Folded = true

	initialChips1 := player1.Chips
	pot := room.Pot

	// 调用determineWinner
	room.Mutex.Lock()
	determineWinner(room)

	// 验证结果
	if player1.Chips-initialChips1 == pot {
		fmt.Println("✅ 测试通过：单个获胜者获得全部底池")
		fmt.Printf("   玩家1获得: %d\n", pot)
	} else {
		fmt.Printf("❌ 测试失败：期望玩家1获得 %d，实际获得 %d\n", pot, player1.Chips-initialChips1)
	}
}
