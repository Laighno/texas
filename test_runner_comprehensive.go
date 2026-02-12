// +build test

package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("\n=== 开始综合测试 ===")
	
	// 运行牌型评估测试
	runHandEvaluationTests()
	
	// 运行边界情况测试
	runEdgeCaseTests()
	
	fmt.Println("\n=== 综合测试完成 ===")
}

func runHandEvaluationTests() {
	fmt.Println("\n--- 牌型评估测试 ---")
	
	// 测试1: 高牌
	testHighCard()
	
	// 测试2: 一对
	testOnePair()
	
	// 测试3: 两对
	testTwoPair()
	
	// 测试4: 三条
	testThreeOfAKind()
	
	// 测试5: 顺子
	testStraight()
	
	// 测试6: 同花
	testFlush()
	
	// 测试7: 葫芦
	testFullHouse()
	
	// 测试8: 四条
	testFourOfAKind()
	
	// 测试9: 同花顺
	testStraightFlush()
	
	// 测试10: 皇家同花顺
	testRoyalFlush()
	
	// 测试11: A-2-3-4-5顺子（最小顺子）
	testWheelStraight()
}

func testHighCard() {
	playerHand := []Card{
		{Suit: "spades", Rank: "2"},
		{Suit: "hearts", Rank: "7"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "9"},
		{Suit: "clubs", Rank: "J"},
		{Suit: "spades", Rank: "K"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != HIGH_CARD {
		log.Printf("❌ 高牌测试失败: 期望高牌，得到: %s", rank.Description)
	} else {
		log.Printf("✅ 高牌测试通过: %s", rank.Description)
	}
}

func testOnePair() {
	playerHand := []Card{
		{Suit: "spades", Rank: "A"},
		{Suit: "hearts", Rank: "A"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "2"},
		{Suit: "clubs", Rank: "5"},
		{Suit: "spades", Rank: "9"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != ONE_PAIR {
		log.Printf("❌ 一对测试失败: 期望一对，得到: %s", rank.Description)
	} else {
		log.Printf("✅ 一对测试通过: %s", rank.Description)
	}
}

func testTwoPair() {
	playerHand := []Card{
		{Suit: "spades", Rank: "A"},
		{Suit: "hearts", Rank: "A"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "K"},
		{Suit: "clubs", Rank: "K"},
		{Suit: "spades", Rank: "9"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != TWO_PAIR {
		log.Printf("❌ 两对测试失败: 期望两对，得到: %s", rank.Description)
	} else {
		log.Printf("✅ 两对测试通过: %s", rank.Description)
	}
}

func testThreeOfAKind() {
	playerHand := []Card{
		{Suit: "spades", Rank: "A"},
		{Suit: "hearts", Rank: "A"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "A"},
		{Suit: "clubs", Rank: "K"},
		{Suit: "spades", Rank: "9"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != THREE_OF_A_KIND {
		log.Printf("❌ 三条测试失败: 期望三条，得到: %s", rank.Description)
	} else {
		log.Printf("✅ 三条测试通过: %s", rank.Description)
	}
}

func testStraight() {
	playerHand := []Card{
		{Suit: "spades", Rank: "5"},
		{Suit: "hearts", Rank: "6"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "7"},
		{Suit: "clubs", Rank: "8"},
		{Suit: "spades", Rank: "9"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != STRAIGHT {
		log.Printf("❌ 顺子测试失败: 期望顺子，得到: %s", rank.Description)
	} else {
		log.Printf("✅ 顺子测试通过: %s", rank.Description)
	}
}

func testFlush() {
	playerHand := []Card{
		{Suit: "spades", Rank: "2"},
		{Suit: "spades", Rank: "5"},
	}
	communityCards := []Card{
		{Suit: "spades", Rank: "7"},
		{Suit: "spades", Rank: "9"},
		{Suit: "spades", Rank: "J"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != FLUSH {
		log.Printf("❌ 同花测试失败: 期望同花，得到: %s", rank.Description)
	} else {
		log.Printf("✅ 同花测试通过: %s", rank.Description)
	}
}

func testFullHouse() {
	playerHand := []Card{
		{Suit: "spades", Rank: "A"},
		{Suit: "hearts", Rank: "A"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "A"},
		{Suit: "clubs", Rank: "K"},
		{Suit: "spades", Rank: "K"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != FULL_HOUSE {
		log.Printf("❌ 葫芦测试失败: 期望葫芦，得到: %s", rank.Description)
	} else {
		log.Printf("✅ 葫芦测试通过: %s", rank.Description)
	}
}

func testFourOfAKind() {
	playerHand := []Card{
		{Suit: "spades", Rank: "A"},
		{Suit: "hearts", Rank: "A"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "A"},
		{Suit: "clubs", Rank: "A"},
		{Suit: "spades", Rank: "K"},
	}
	
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ 四条测试触发panic: %v", r)
		}
	}()
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != FOUR_OF_A_KIND {
		log.Printf("❌ 四条测试失败: 期望四条，得到: %s", rank.Description)
	} else {
		if len(rank.Kickers) == 0 {
			log.Printf("⚠️ 四条测试: kickers为空，可能存在bug")
		} else {
			log.Printf("✅ 四条测试通过: %s, Kickers: %v", rank.Description, rank.Kickers)
		}
	}
}

func testStraightFlush() {
	playerHand := []Card{
		{Suit: "spades", Rank: "5"},
		{Suit: "spades", Rank: "6"},
	}
	communityCards := []Card{
		{Suit: "spades", Rank: "7"},
		{Suit: "spades", Rank: "8"},
		{Suit: "spades", Rank: "9"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != STRAIGHT_FLUSH {
		log.Printf("❌ 同花顺测试失败: 期望同花顺，得到: %s", rank.Description)
	} else {
		log.Printf("✅ 同花顺测试通过: %s", rank.Description)
	}
}

func testRoyalFlush() {
	playerHand := []Card{
		{Suit: "spades", Rank: "10"},
		{Suit: "spades", Rank: "J"},
	}
	communityCards := []Card{
		{Suit: "spades", Rank: "Q"},
		{Suit: "spades", Rank: "K"},
		{Suit: "spades", Rank: "A"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != ROYAL_FLUSH {
		log.Printf("❌ 皇家同花顺测试失败: 期望皇家同花顺，得到: %s", rank.Description)
	} else {
		log.Printf("✅ 皇家同花顺测试通过: %s", rank.Description)
	}
}

func testWheelStraight() {
	playerHand := []Card{
		{Suit: "spades", Rank: "A"},
		{Suit: "hearts", Rank: "2"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "3"},
		{Suit: "clubs", Rank: "4"},
		{Suit: "spades", Rank: "5"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != STRAIGHT {
		log.Printf("❌ A-2-3-4-5顺子测试失败: 期望顺子，得到: %s", rank.Description)
	} else {
		if rank.Kickers[0] != 5 {
			log.Printf("⚠️ A-2-3-4-5顺子测试: 期望高点数为5，得到: %d", rank.Kickers[0])
		} else {
			log.Printf("✅ A-2-3-4-5顺子测试通过: %s", rank.Description)
		}
	}
}

func runEdgeCaseTests() {
	fmt.Println("\n--- 边界情况测试 ---")
	
	// 测试空手牌
	testEmptyHand()
	
	// 测试少于5张牌
	testLessThanFiveCards()
	
	// 测试比较手牌
	testCompareHandRanks()
}

func testEmptyHand() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ 空手牌测试触发panic: %v", r)
		}
	}()
	
	playerHand := []Card{}
	communityCards := []Card{}
	
	rank := evaluateHand(playerHand, communityCards)
	log.Printf("空手牌测试结果: %s (Rank: %d)", rank.Description, rank.Rank)
}

func testLessThanFiveCards() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ 少于5张牌测试触发panic: %v", r)
		}
	}()
	
	playerHand := []Card{
		{Suit: "spades", Rank: "A"},
		{Suit: "hearts", Rank: "K"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "Q"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	log.Printf("少于5张牌测试结果: %s (Rank: %d)", rank.Description, rank.Rank)
}

func testCompareHandRanks() {
	// 测试不同牌型
	rank1 := HandRank{Rank: ONE_PAIR, Kickers: []int{14, 13, 12}}
	rank2 := HandRank{Rank: TWO_PAIR, Kickers: []int{13, 12, 11}}
	result := compareHandRanks(rank1, rank2)
	if result >= 0 {
		log.Printf("❌ 不同牌型比较测试失败: 两对应该大于一对")
	} else {
		log.Printf("✅ 不同牌型比较测试通过")
	}
	
	// 测试相同牌型，不同kicker
	rank3 := HandRank{Rank: ONE_PAIR, Kickers: []int{14, 13, 12}}
	rank4 := HandRank{Rank: ONE_PAIR, Kickers: []int{13, 12, 11}}
	result = compareHandRanks(rank3, rank4)
	if result <= 0 {
		log.Printf("❌ 相同牌型比较测试失败: A对应该大于K对")
	} else {
		log.Printf("✅ 相同牌型比较测试通过")
	}
	
	// 测试完全相同
	rank5 := HandRank{Rank: ONE_PAIR, Kickers: []int{14, 13, 12}}
	rank6 := HandRank{Rank: ONE_PAIR, Kickers: []int{14, 13, 12}}
	result = compareHandRanks(rank5, rank6)
	if result != 0 {
		log.Printf("❌ 相同手牌比较测试失败: 应该返回0")
	} else {
		log.Printf("✅ 相同手牌比较测试通过")
	}
}
