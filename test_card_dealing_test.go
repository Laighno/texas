package main

import (
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"
)

// 测试drawCard的错误处理
func TestDrawCardError(t *testing.T) {
	// 测试空牌组
	emptyDeck := []Card{}
	_, err := drawCard(&emptyDeck)
	if err == nil {
		t.Error("Expected error when drawing from empty deck, got nil")
	}
}

// 测试发牌顺序
func TestDealingOrder(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	// 创建测试房间
	room := &GameRoom{
		ID:          "test_room",
		Players:     make([]*Player, 0),
		DealerIndex: 0,
	}

	// 添加4个测试玩家
	for i := 0; i < 4; i++ {
		player := &Player{
			ID:    fmt.Sprintf("player_%d", i),
			Name:  fmt.Sprintf("Player %d", i),
			Chips: INITIAL_CHIPS,
			Hand:  []Card{},
		}
		room.Players = append(room.Players, player)
	}

	// 创建并洗牌
	room.Deck = createDeck()
	shuffleDeck(room.Deck)

	// 记录初始牌组前8张牌
	expectedCards := make([]Card, 8)
	copy(expectedCards, room.Deck[:8])

	// 按座位顺序发牌（模拟startNewHand的发牌逻辑）
	deckCopy := make([]Card, len(room.Deck))
	copy(deckCopy, room.Deck)

	for round := 0; round < 2; round++ {
		for i := 0; i < len(room.Players); i++ {
			playerIndex := (room.DealerIndex + 1 + i) % len(room.Players)
			card, err := drawCard(&deckCopy)
			if err != nil {
				t.Fatalf("Failed to draw card: %v", err)
			}
			room.Players[playerIndex].Hand = append(room.Players[playerIndex].Hand, card)
		}
	}

	// 验证发牌顺序
	// 第一轮：庄家下一位(索引1)先拿，然后是2,3,0
	// 第二轮：同样顺序
	expectedOrder := []int{1, 2, 3, 0, 1, 2, 3, 0}
	cardIndex := 0

	// 验证每个玩家最终有2张牌
	for _, player := range room.Players {
		if len(player.Hand) != 2 {
			t.Errorf("Player %s should have 2 cards, got %d", player.Name, len(player.Hand))
		}
	}

	// 验证牌按照正确顺序发出
	for round := 0; round < 2; round++ {
		for i := 0; i < 4; i++ {
			playerIdx := expectedOrder[round*4+i]
			player := room.Players[playerIdx]
			// 验证牌是否按照正确顺序发出
			if player.Hand[round].Suit != expectedCards[cardIndex].Suit ||
				player.Hand[round].Rank != expectedCards[cardIndex].Rank {
				t.Errorf("Card mismatch for player %d, round %d: expected %s-%s, got %s-%s",
					playerIdx, round,
					expectedCards[cardIndex].Suit, expectedCards[cardIndex].Rank,
					player.Hand[round].Suit, player.Hand[round].Rank)
			}
			cardIndex++
		}
	}

	log.Printf("✅ 发牌顺序测试通过")
}

// 测试牌组随机性
func TestDeckRandomness(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	// 生成10副牌，检查第一张牌的分布
	firstCards := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		deck := createDeck()
		shuffleDeck(deck)
		firstCard := fmt.Sprintf("%s-%s", deck[0].Suit, deck[0].Rank)
		firstCards[firstCard]++
	}

	// 统计学检验：每张牌出现的概率应该约为 iterations/52
	expectedFreq := float64(iterations) / 52.0
	tolerance := expectedFreq * 0.5 // 允许50%的偏差

	for card, freq := range firstCards {
		if float64(freq) > expectedFreq+tolerance {
			t.Logf("Warning: Card %s appears %d times (expected ~%.1f)", card, freq, expectedFreq)
		}
	}

	// 至少应该有40张不同的牌出现过（52张牌中）
	if len(firstCards) < 40 {
		t.Errorf("Insufficient randomness: only %d different cards appeared in %d shuffles", len(firstCards), iterations)
	}

	log.Printf("✅ 随机性测试通过：%d次洗牌中出现了%d张不同的首牌", iterations, len(firstCards))
}

// 测试牌组完整性
func TestDeckIntegrity(t *testing.T) {
	deck := createDeck()

	if len(deck) != CARDS_IN_DECK {
		t.Errorf("Expected %d cards, got %d", CARDS_IN_DECK, len(deck))
	}

	// 检查每种花色和点数的牌是否都存在
	suits := map[string]int{}
	ranks := map[string]int{}

	for _, card := range deck {
		suits[card.Suit]++
		ranks[card.Rank]++
	}

	if len(suits) != 4 {
		t.Errorf("Expected 4 suits, got %d", len(suits))
	}

	if len(ranks) != 13 {
		t.Errorf("Expected 13 ranks, got %d", len(ranks))
	}

	for suit, count := range suits {
		if count != 13 {
			t.Errorf("Suit %s has %d cards, expected 13", suit, count)
		}
	}

	for rank, count := range ranks {
		if count != 4 {
			t.Errorf("Rank %s has %d cards, expected 4", rank, count)
		}
	}

	log.Printf("✅ 牌组完整性测试通过")
}

// 测试常量值
func TestConstants(t *testing.T) {
	if SMALL_BLIND*2 != BIG_BLIND {
		t.Errorf("BIG_BLIND should be 2x SMALL_BLIND, got %d and %d", SMALL_BLIND, BIG_BLIND)
	}

	if INITIAL_CHIPS < BIG_BLIND*10 {
		t.Errorf("INITIAL_CHIPS (%d) should be at least 10x BIG_BLIND (%d)", INITIAL_CHIPS, BIG_BLIND)
	}

	if MIN_PLAYERS < 2 || MIN_PLAYERS > MAX_PLAYERS {
		t.Errorf("MIN_PLAYERS (%d) should be between 2 and MAX_PLAYERS (%d)", MIN_PLAYERS, MAX_PLAYERS)
	}

	if MAX_PLAYERS > 23 {
		t.Errorf("MAX_PLAYERS (%d) is too large, max possible is 23 (52 cards / 2 per player - 5 community cards)", MAX_PLAYERS)
	}

	log.Printf("✅ 常量值测试通过")
}
