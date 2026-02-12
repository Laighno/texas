//go:build test
// +build test

package main

import (
	"fmt"
	"log"
	"testing"
)

// 测试牌型评估功能
func TestHandEvaluation(t *testing.T) {
	log.Println("=== 测试牌型评估功能 ===")

	// 测试1: 高牌
	testHighCard(t)
	
	// 测试2: 一对
	testOnePair(t)
	
	// 测试3: 两对
	testTwoPair(t)
	
	// 测试4: 三条
	testThreeOfAKind(t)
	
	// 测试5: 顺子
	testStraight(t)
	
	// 测试6: 同花
	testFlush(t)
	
	// 测试7: 葫芦
	testFullHouse(t)
	
	// 测试8: 四条
	testFourOfAKind(t)
	
	// 测试9: 同花顺
	testStraightFlush(t)
	
	// 测试10: 皇家同花顺
	testRoyalFlush(t)
	
	// 测试11: A-2-3-4-5顺子（最小顺子）
	testWheelStraight(t)
	
	// 测试12: 边界情况 - 空数组
	testEmptyHand(t)
	
	// 测试13: 边界情况 - 少于5张牌
	testLessThanFiveCards(t)
}

func testHighCard(t *testing.T) {
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
		t.Errorf("期望高牌，得到: %s", rank.Description)
	}
	log.Printf("✅ 高牌测试通过: %s", rank.Description)
}

func testOnePair(t *testing.T) {
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
		t.Errorf("期望一对，得到: %s", rank.Description)
	}
	log.Printf("✅ 一对测试通过: %s", rank.Description)
}

func testTwoPair(t *testing.T) {
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
		t.Errorf("期望两对，得到: %s", rank.Description)
	}
	log.Printf("✅ 两对测试通过: %s", rank.Description)
}

func testThreeOfAKind(t *testing.T) {
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
		t.Errorf("期望三条，得到: %s", rank.Description)
	}
	log.Printf("✅ 三条测试通过: %s", rank.Description)
}

func testStraight(t *testing.T) {
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
		t.Errorf("期望顺子，得到: %s", rank.Description)
	}
	log.Printf("✅ 顺子测试通过: %s", rank.Description)
}

func testFlush(t *testing.T) {
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
		t.Errorf("期望同花，得到: %s", rank.Description)
	}
	log.Printf("✅ 同花测试通过: %s", rank.Description)
}

func testFullHouse(t *testing.T) {
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
		t.Errorf("期望葫芦，得到: %s", rank.Description)
	}
	log.Printf("✅ 葫芦测试通过: %s", rank.Description)
}

func testFourOfAKind(t *testing.T) {
	playerHand := []Card{
		{Suit: "spades", Rank: "A"},
		{Suit: "hearts", Rank: "A"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "A"},
		{Suit: "clubs", Rank: "A"},
		{Suit: "spades", Rank: "9"},
	}
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != FOUR_OF_A_KIND {
		t.Errorf("期望四条，得到: %s", rank.Description)
	}
	log.Printf("✅ 四条测试通过: %s", rank.Description)
}

func testStraightFlush(t *testing.T) {
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
		t.Errorf("期望同花顺，得到: %s", rank.Description)
	}
	log.Printf("✅ 同花顺测试通过: %s", rank.Description)
}

func testRoyalFlush(t *testing.T) {
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
		t.Errorf("期望皇家同花顺，得到: %s", rank.Description)
	}
	log.Printf("✅ 皇家同花顺测试通过: %s", rank.Description)
}

func testWheelStraight(t *testing.T) {
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
		t.Errorf("期望A-2-3-4-5顺子，得到: %s", rank.Description)
	}
	if rank.Kickers[0] != 5 {
		t.Errorf("期望顺子高点数为5，得到: %d", rank.Kickers[0])
	}
	log.Printf("✅ A-2-3-4-5顺子测试通过: %s", rank.Description)
}

func testEmptyHand(t *testing.T) {
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

func testLessThanFiveCards(t *testing.T) {
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

// 测试边界情况：四条时kickers可能为空
func TestFourOfAKindEdgeCase(t *testing.T) {
	log.Println("=== 测试四条边界情况 ===")
	
	// 这种情况在德州扑克中理论上不可能（需要5张A），但测试代码健壮性
	// 实际测试：正常的四条情况
	playerHand := []Card{
		{Suit: "spades", Rank: "A"},
		{Suit: "hearts", Rank: "A"},
	}
	communityCards := []Card{
		{Suit: "diamonds", Rank: "A"},
		{Suit: "clubs", Rank: "A"},
		{Suit: "spades", Rank: "K"}, // 确保有kicker
	}
	
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("四条评估触发panic: %v", r)
		}
	}()
	
	rank := evaluateHand(playerHand, communityCards)
	if rank.Rank != FOUR_OF_A_KIND {
		t.Errorf("期望四条，得到: %s", rank.Description)
	}
	if len(rank.Kickers) == 0 {
		t.Errorf("四条应该有kicker，但kickers为空")
	}
	log.Printf("✅ 四条边界测试通过: %s, Kickers: %v", rank.Description, rank.Kickers)
}

// 测试比较手牌功能
func TestCompareHandRanks(t *testing.T) {
	log.Println("=== 测试手牌比较功能 ===")
	
	// 测试1: 不同牌型
	rank1 := HandRank{Rank: ONE_PAIR, Kickers: []int{14, 13, 12}}
	rank2 := HandRank{Rank: TWO_PAIR, Kickers: []int{13, 12, 11}}
	result := compareHandRanks(rank1, rank2)
	if result >= 0 {
		t.Errorf("两对应该大于一对，但比较结果: %d", result)
	}
	log.Printf("✅ 不同牌型比较测试通过")
	
	// 测试2: 相同牌型，不同kicker
	rank3 := HandRank{Rank: ONE_PAIR, Kickers: []int{14, 13, 12}}
	rank4 := HandRank{Rank: ONE_PAIR, Kickers: []int{13, 12, 11}}
	result = compareHandRanks(rank3, rank4)
	if result <= 0 {
		t.Errorf("A对应该大于K对，但比较结果: %d", result)
	}
	log.Printf("✅ 相同牌型比较测试通过")
	
	// 测试3: 完全相同
	rank5 := HandRank{Rank: ONE_PAIR, Kickers: []int{14, 13, 12}}
	rank6 := HandRank{Rank: ONE_PAIR, Kickers: []int{14, 13, 12}}
	result = compareHandRanks(rank5, rank6)
	if result != 0 {
		t.Errorf("相同手牌应该返回0，但得到: %d", result)
	}
	log.Printf("✅ 相同手牌比较测试通过")
}

func runComprehensiveTests() {
	fmt.Println("\n=== 开始综合测试 ===")
	
	// 运行牌型评估测试
	t := &testing.T{}
	TestHandEvaluation(t)
	TestFourOfAKindEdgeCase(t)
	TestCompareHandRanks(t)
	
	fmt.Println("\n=== 综合测试完成 ===")
}
