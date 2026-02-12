package main

import (
	"sort"
	"strconv"
)

// 牌型等级
const (
	HIGH_CARD = iota
	ONE_PAIR
	TWO_PAIR
	THREE_OF_A_KIND
	STRAIGHT
	FLUSH
	FULL_HOUSE
	FOUR_OF_A_KIND
	STRAIGHT_FLUSH
	ROYAL_FLUSH
)

// 手牌评估结果
type HandRank struct {
	Rank        int
	Kickers     []int
	Description string
}

// 评估玩家手牌
func evaluateHand(playerHand []Card, communityCards []Card) HandRank {
	allCards := append(playerHand, communityCards...)
	
	// 尝试所有5张牌的组合
	bestRank := HandRank{Rank: HIGH_CARD, Kickers: []int{}}
	
	// 从7张牌中选择5张的最佳组合
	combinations := getCombinations(allCards, 5)
	
	for _, combo := range combinations {
		rank := evaluateFiveCards(combo)
		if compareHandRanks(rank, bestRank) > 0 {
			bestRank = rank
		}
	}
	
	return bestRank
}

// 从n张牌中选择k张的所有组合
func getCombinations(cards []Card, k int) [][]Card {
	if k == 0 {
		return [][]Card{{}}
	}
	if len(cards) < k {
		return [][]Card{}
	}
	
	var result [][]Card
	for i := 0; i <= len(cards)-k; i++ {
		subCombos := getCombinations(cards[i+1:], k-1)
		for _, sub := range subCombos {
			combo := append([]Card{cards[i]}, sub...)
			result = append(result, combo)
		}
	}
	return result
}

// 评估5张牌
func evaluateFiveCards(cards []Card) HandRank {
	if len(cards) != 5 {
		return HandRank{Rank: HIGH_CARD}
	}
	
	// 按点数排序
	sort.Slice(cards, func(i, j int) bool {
		return cardValue(cards[i].Rank) < cardValue(cards[j].Rank)
	})
	
	// 检查同花
	isFlush := isFlush(cards)
	
	// 检查顺子
	isStraight, highCard := isStraight(cards)
	
	// 检查对子、三条、四条
	rankCounts := make(map[int]int)
	for _, card := range cards {
		val := cardValue(card.Rank)
		rankCounts[val]++
	}
	
	var pairs []int
	var threeKind int
	var fourKind int
	var kickers []int
	
	for rank, count := range rankCounts {
		switch count {
		case 4:
			fourKind = rank
		case 3:
			threeKind = rank
		case 2:
			pairs = append(pairs, rank)
		case 1:
			kickers = append(kickers, rank)
		}
	}
	
	sort.Sort(sort.Reverse(sort.IntSlice(pairs)))
	sort.Sort(sort.Reverse(sort.IntSlice(kickers)))
	
	// 判断牌型
	if isFlush && isStraight && highCard == 14 {
		return HandRank{
			Rank:        ROYAL_FLUSH,
			Kickers:     []int{14},
			Description: "皇家同花顺",
		}
	}
	
	if isFlush && isStraight {
		return HandRank{
			Rank:        STRAIGHT_FLUSH,
			Kickers:     []int{highCard},
			Description: "同花顺",
		}
	}
	
	if fourKind > 0 {
		return HandRank{
			Rank:        FOUR_OF_A_KIND,
			Kickers:     []int{fourKind, kickers[0]},
			Description: "四条",
		}
	}
	
	if threeKind > 0 && len(pairs) > 0 {
		return HandRank{
			Rank:        FULL_HOUSE,
			Kickers:     []int{threeKind, pairs[0]},
			Description: "葫芦",
		}
	}
	
	if isFlush {
		return HandRank{
			Rank:        FLUSH,
			Kickers:     kickers,
			Description: "同花",
		}
	}
	
	if isStraight {
		return HandRank{
			Rank:        STRAIGHT,
			Kickers:     []int{highCard},
			Description: "顺子",
		}
	}
	
	if threeKind > 0 {
		return HandRank{
			Rank:        THREE_OF_A_KIND,
			Kickers:     append([]int{threeKind}, kickers...),
			Description: "三条",
		}
	}
	
	if len(pairs) >= 2 {
		return HandRank{
			Rank:        TWO_PAIR,
			Kickers:     append(pairs[:2], kickers...),
			Description: "两对",
		}
	}
	
	if len(pairs) == 1 {
		return HandRank{
			Rank:        ONE_PAIR,
			Kickers:     append([]int{pairs[0]}, kickers...),
			Description: "一对",
		}
	}
	
	return HandRank{
		Rank:        HIGH_CARD,
		Kickers:     kickers,
		Description: "高牌",
	}
}

// 检查是否同花
func isFlush(cards []Card) bool {
	if len(cards) == 0 {
		return false
	}
	suit := cards[0].Suit
	for _, card := range cards[1:] {
		if card.Suit != suit {
			return false
		}
	}
	return true
}

// 检查是否顺子
func isStraight(cards []Card) (bool, int) {
	if len(cards) != 5 {
		return false, 0
	}
	
	values := make([]int, len(cards))
	for i, card := range cards {
		values[i] = cardValue(card.Rank)
	}
	sort.Ints(values)
	
	// 检查普通顺子
	isStraight := true
	for i := 1; i < len(values); i++ {
		if values[i] != values[i-1]+1 {
			isStraight = false
			break
		}
	}
	
	if isStraight {
		return true, values[4]
	}
	
	// 检查A-2-3-4-5顺子（A作为1）
	if values[0] == 2 && values[1] == 3 && values[2] == 4 && values[3] == 5 && values[4] == 14 {
		return true, 5
	}
	
	return false, 0
}

// 获取牌的点数值
func cardValue(rank string) int {
	switch rank {
	case "A":
		return 14
	case "K":
		return 13
	case "Q":
		return 12
	case "J":
		return 11
	default:
		val, _ := strconv.Atoi(rank)
		return val
	}
}

// 比较两个手牌等级
func compareHandRanks(rank1, rank2 HandRank) int {
	if rank1.Rank != rank2.Rank {
		return rank1.Rank - rank2.Rank
	}
	
	// 相同牌型，比较踢脚牌
	for i := 0; i < len(rank1.Kickers) && i < len(rank2.Kickers); i++ {
		if rank1.Kickers[i] != rank2.Kickers[i] {
			return rank1.Kickers[i] - rank2.Kickers[i]
		}
	}
	
	return 0
}

// 比较两个手牌字符串（用于旧代码兼容）
func compareHands(hand1, hand2 string) int {
	// 这个函数保留用于向后兼容，实际应该使用compareHandRanks
	return 0
}
