//go:build !tie_test
// +build !tie_test

package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	MIN_PLAYERS     = 4
	MAX_PLAYERS     = 12
	PORT            = ":8080"
	SMALL_BLIND     = 5   // 小盲注
	BIG_BLIND       = 10  // 大盲注
	INITIAL_CHIPS   = 500 // 初始筹码
	BUY_IN_AMOUNT   = 500 // 买入金额
	TURN_TIMEOUT    = 60  // 回合超时时间（秒）
	CARDS_IN_DECK   = 52  // 一副牌的牌数
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// 扑克牌
type Card struct {
	Suit string `json:"suit"` // 花色: spades, hearts, diamonds, clubs
	Rank string `json:"rank"` // 点数: 2-10, J, Q, K, A
}

// 玩家状态
const (
	PlayerStatusSpectating = "spectating" // 观战状态
	PlayerStatusPlaying    = "playing"    // 游戏中
)

// 玩家
type Player struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Conn          *websocket.Conn `json:"-"`
	Hand          []Card          `json:"hand"`
	Chips         int             `json:"chips"`
	Bet           int             `json:"bet"`
	Folded        bool            `json:"folded"`
	IsDealer      bool            `json:"isDealer"`
	IsSmall       bool            `json:"isSmall"`
	IsBig         bool            `json:"isBig"`
	AllIn         bool            `json:"allIn"`
	Status        string          `json:"status"`        // spectating 或 playing
	LastHeartbeat time.Time       `json:"-"`             // 最后心跳时间
	HeartbeatTimer *time.Timer    `json:"-"`             // 心跳超时定时器
	HeartbeatTimeout bool         `json:"-"`             // 心跳超时标记（游戏结束后移入观战）
}

// 游戏房间
type GameRoom struct {
	ID                string       `json:"id"`
	Players           []*Player    `json:"players"`
	Spectators        []*Player    `json:"spectators"`        // 观战玩家列表
	WaitingPlayers    []*Player    `json:"waitingPlayers"`    // 等待加入的玩家列表（游戏进行中时）
	CommunityCards    []Card       `json:"communityCards"`
	Pot               int          `json:"pot"`
	CurrentBet        int          `json:"currentBet"`
	DealerIndex       int          `json:"dealerIndex"`
	CurrentTurn       int          `json:"currentTurn"`
	GamePhase         string       `json:"gamePhase"`         // preflop, flop, turn, river, showdown, waiting
	LastRaiseIndex    int          `json:"lastRaiseIndex"`    // 最后加注的玩家索引，用于判断是否所有人都行动过一轮
	BettingStartIndex int          `json:"bettingStartIndex"` // 当前下注轮开始行动的玩家索引
	TurnTimer         *time.Timer  `json:"-"`                 // 当前回合的超时定时器
	Deck              []Card       `json:"-"`
	BuyHandCount      map[string]int `json:"buyHandCount"`    // 玩家买一手次数（按昵称）
	Mutex             sync.RWMutex `json:"-"`
}

// 筹码存储（按房间ID+玩家昵称）
var chipsStorage = make(map[string]map[string]int) // roomID -> playerName -> chips
var chipsMutex sync.RWMutex

// 用于JSON序列化的房间数据
func (room *GameRoom) ToJSON() map[string]interface{} {
	// 注意：调用此函数时不应该持有写锁，只应该持有读锁或没有锁
	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	log.Printf("ToJSON: 序列化房间 %s，玩家数: %d", room.ID, len(room.Players))

	// 创建玩家数据的副本，避免并发问题
	playersData := make([]map[string]interface{}, len(room.Players))
	for i, p := range room.Players {
		playersData[i] = map[string]interface{}{
			"id":       p.ID,
			"name":     p.Name,
			"hand":     p.Hand,
			"chips":    p.Chips,
			"bet":      p.Bet,
			"folded":   p.Folded,
			"isDealer": p.IsDealer,
			"isSmall":  p.IsSmall,
			"isBig":    p.IsBig,
			"allIn":    p.AllIn,
			"status":   p.Status,
		}
	}

	// 创建观战玩家数据的副本
	spectatorsData := make([]map[string]interface{}, len(room.Spectators))
	for i, p := range room.Spectators {
		spectatorsData[i] = map[string]interface{}{
			"id":     p.ID,
			"name":   p.Name,
			"chips":  p.Chips,
			"status": p.Status,
		}
	}

	// 创建等待玩家数据的副本
	waitingPlayersData := make([]map[string]interface{}, len(room.WaitingPlayers))
	for i, p := range room.WaitingPlayers {
		waitingPlayersData[i] = map[string]interface{}{
			"id":     p.ID,
			"name":   p.Name,
			"chips":  p.Chips,
			"status": p.Status,
		}
	}

	result := map[string]interface{}{
		"id":             room.ID,
		"players":        playersData,
		"spectators":     spectatorsData,
		"waitingPlayers": waitingPlayersData,
		"communityCards": room.CommunityCards,
		"pot":            room.Pot,
		"currentBet":     room.CurrentBet,
		"dealerIndex":    room.DealerIndex,
		"currentTurn":    room.CurrentTurn,
		"gamePhase":      room.GamePhase,
	}

	log.Printf("ToJSON: 序列化完成，房间 %s", room.ID)
	return result
}

// 消息类型
type Message struct {
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	PlayerID string      `json:"playerId,omitempty"`
}

// 全局房间管理
var rooms = make(map[string]*GameRoom)
var roomsMutex sync.RWMutex

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", serveStatic)

	log.Printf("德州扑克服务器启动在端口 %s", PORT)
	log.Fatal(http.ListenAndServe(PORT, nil))
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "index.html")
	} else {
		http.ServeFile(w, r, r.URL.Path[1:])
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到WebSocket连接请求: %s", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	playerID := generateID()
	player := &Player{
		ID:            playerID,
		Conn:          conn,
		Chips:         INITIAL_CHIPS, // 初始筹码（一手）
		Status:        PlayerStatusSpectating,
		LastHeartbeat: time.Now(),
	}

	log.Printf("新玩家连接成功: ID=%s, 地址=%s", playerID, r.RemoteAddr)

	// 设置连接读写超时
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		player.LastHeartbeat = time.Now()
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动心跳检测
	go startHeartbeatCheck(player)

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("读取消息失败 (玩家=%s): %v", playerID, err)
			removePlayer(player)
			break
		}

		// 更新心跳时间
		player.LastHeartbeat = time.Now()
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		log.Printf("收到消息 (玩家=%s): 类型=%s", playerID, msg.Type)
		handleMessage(player, &msg)
	}
}

func handleMessage(player *Player, msg *Message) {
	log.Printf("收到消息: 玩家=%s, 类型=%s", player.ID, msg.Type)
	switch msg.Type {
	case "joinRoom":
		joinRoom(player, msg)
	case "createRoom":
		createRoom(player, msg)
	case "joinTable":
		joinTable(player, msg)
	case "action":
		handleAction(player, msg)
	case "startGame":
		startGame(player, msg)
	case "buyHand":
		buyHand(player, msg)
	case "getBuyHandStats":
		getBuyHandStats(player, msg)
	case "heartbeat":
		// 心跳消息，已在连接层处理
		player.LastHeartbeat = time.Now()
	default:
		log.Printf("未知消息类型: %s", msg.Type)
	}
}

func createRoom(player *Player, msg *Message) {
	log.Printf("创建房间请求: 玩家=%s", player.ID)

	data, ok := msg.Data.(map[string]interface{})
	if ok {
		if playerName, exists := data["playerName"].(string); exists && playerName != "" {
			player.Name = playerName
		}
	}

	if player.Name == "" {
		player.Name = "玩家" + player.ID[:4]
	}

	roomID := generateID()
	
	// 加载玩家筹码
	player.Chips = loadPlayerChips(roomID, player.Name)
	player.Status = PlayerStatusSpectating // 新玩家默认观战状态

	room := &GameRoom{
		ID:             roomID,
		Players:        []*Player{},
		Spectators:     []*Player{player}, // 创建者先进入观战状态
		WaitingPlayers: []*Player{},
		GamePhase:      "waiting",
		CommunityCards: []Card{},
		BuyHandCount:   make(map[string]int), // 初始化买一手次数统计
	}

	roomsMutex.Lock()
	rooms[roomID] = room
	roomsMutex.Unlock()

	log.Printf("房间创建成功: 房间ID=%s, 玩家=%s(%s), 筹码=%d", roomID, player.Name, player.ID, player.Chips)

	// 发送房间信息（包含完整房间数据）
	sendMessage(player, Message{
		Type: "roomCreated",
		Data: map[string]interface{}{
			"roomId": roomID,
			"room":   room.ToJSON(),
			"isSpectating": true,
		},
	})
}

func joinRoom(player *Player, msg *Message) {
	log.Printf("加入房间请求: 玩家=%s", player.ID)

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Printf("加入房间失败: 数据格式错误")
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "请求数据格式错误"},
		})
		return
	}

	roomID, ok := data["roomId"].(string)
	if !ok || roomID == "" {
		log.Printf("加入房间失败: 房间ID无效")
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "房间ID无效"},
		})
		return
	}

	playerName, _ := data["playerName"].(string)
	if playerName == "" {
		playerName = "玩家" + player.ID[:4]
	}
	player.Name = playerName

	log.Printf("尝试加入房间: 房间ID=%s, 玩家=%s", roomID, player.Name)

	roomsMutex.RLock()
	room, exists := rooms[roomID]
	roomsMutex.RUnlock()

	if !exists {
		log.Printf("加入房间失败: 房间不存在, 房间ID=%s", roomID)
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "房间不存在"},
		})
		return
	}

	room.Mutex.Lock()

	// 检查是否已有同名玩家
	if hasPlayerWithName(room, player.Name, player.ID) {
		room.Mutex.Unlock()
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "房间中已有同名玩家，请使用不同的昵称"},
		})
		return
	}

	// 检查玩家是否已在观战列表中（重连）
	for i, p := range room.Spectators {
		if p.ID == player.ID {
			// 重连，更新连接
			room.Spectators[i].Conn = player.Conn
			room.Spectators[i].LastHeartbeat = time.Now()
			room.Mutex.Unlock()
			
			// 发送房间信息
			sendMessage(player, Message{
				Type: "roomJoined",
				Data: map[string]interface{}{
					"room":         room.ToJSON(),
					"isSpectating": true,
				},
			})
			return
		}
	}

	// 检查玩家是否已在游戏中（重连）
	for i, p := range room.Players {
		if p.ID == player.ID {
			// 重连，更新连接
			room.Players[i].Conn = player.Conn
			room.Players[i].LastHeartbeat = time.Now()
			room.Mutex.Unlock()
			
			// 发送房间信息
			sendMessage(player, Message{
				Type: "roomJoined",
				Data: map[string]interface{}{
					"room":         room.ToJSON(),
					"isSpectating": false,
				},
			})
			return
		}
	}

	// 新玩家：加载筹码并加入观战状态
	player.Chips = loadPlayerChips(roomID, player.Name)
	player.Status = PlayerStatusSpectating
	
	// 添加到观战列表
	room.Spectators = append(room.Spectators, player)
	
	spectators := make([]*Player, len(room.Spectators))
	copy(spectators, room.Spectators)
	players := make([]*Player, len(room.Players))
	copy(players, room.Players)
	room.Mutex.Unlock()

	log.Printf("玩家 %s 加入房间 %s 观战，筹码=%d", player.Name, roomID, player.Chips)

	// 发送房间信息给新加入的玩家
	sendMessage(player, Message{
		Type: "roomJoined",
		Data: map[string]interface{}{
			"room":         room.ToJSON(),
			"isSpectating": true,
			"message":      "您已进入观战状态，点击'上桌'按钮加入游戏",
		},
	})

	// 广播玩家加入消息
	roomData := room.ToJSON()
	broadcastMsg := Message{
		Type: "playerJoined",
		Data: map[string]interface{}{
			"player": player,
			"room":   roomData,
		},
	}
	
	// 广播给所有玩家和观战者
	for _, p := range players {
		if p.Conn != nil {
			sendMessage(p, broadcastMsg)
		}
	}
	for _, p := range spectators {
		if p.Conn != nil {
			sendMessage(p, broadcastMsg)
		}
	}
}

func startGame(player *Player, msg *Message) {
	log.Printf("处理开始游戏请求: 玩家=%s, 玩家名称=%s", player.ID, player.Name)
	room := findPlayerRoom(player)
	if room == nil {
		log.Printf("❌ 开始游戏失败: 未找到房间，玩家=%s, 玩家名称=%s", player.ID, player.Name)
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "未找到房间，请重新加入"},
		})
		return
	}
	log.Printf("✅ 找到房间: 玩家=%s, 房间ID=%s", player.ID, room.ID)

	room.Mutex.Lock()
	log.Printf("🔍 开始游戏检查: 玩家=%s, 房间=%s, 玩家数=%d, 游戏阶段=%s", player.ID, room.ID, len(room.Players), room.GamePhase)

	if len(room.Players) < MIN_PLAYERS {
		room.Mutex.Unlock()
		log.Printf("开始游戏失败: 玩家数不足，玩家=%s, 当前玩家数=%d, 需要=%d", player.ID, len(room.Players), MIN_PLAYERS)
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "至少需要4个玩家才能开始游戏"},
		})
		return
	}

	if room.GamePhase != "waiting" {
		room.Mutex.Unlock()
		log.Printf("开始游戏失败: 游戏已在进行中，玩家=%s, 阶段=%s (期望: waiting)，静默返回", player.ID, room.GamePhase)
		// 不发送错误消息，静默返回
		return
	}

	log.Printf("✅ 玩家 %s 开始游戏，房间 %s，玩家数: %d, 游戏阶段: %s", player.Name, room.ID, len(room.Players), room.GamePhase)

	// 开始新游戏（startNewHand会自己管理锁）
	room.Mutex.Unlock()
	log.Printf("准备调用startNewHand，房间 %s", room.ID)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ startNewHand发生panic: %v", r)
		}
	}()
	startNewHand(room)
	log.Printf("startNewHand执行完成，房间 %s", room.ID)
}

func startNewHand(room *GameRoom) {
	// 注意：调用此函数时不应该持有room.Mutex锁
	log.Printf("startNewHand开始执行，房间 %s", room.ID)
	room.Mutex.Lock()
	log.Printf("startNewHand已获取锁，房间 %s", room.ID)

	// 停止旧的超时定时器（如果存在）
	if room.TurnTimer != nil {
		room.TurnTimer.Stop()
		room.TurnTimer = nil
		log.Printf("已停止旧的超时定时器，房间 %s", room.ID)
	}

	// 重置游戏状态
	room.Pot = 0
	room.CurrentBet = 0
	room.CommunityCards = []Card{}
	room.GamePhase = "preflop"
	log.Printf("游戏状态已重置，房间 %s", room.ID)

	// 重置玩家状态
	for _, p := range room.Players {
		p.Hand = []Card{}
		p.Bet = 0
		p.Folded = false
		p.AllIn = false
	}

	// 创建并洗牌
	room.Deck = createDeck()
	shuffleDeck(room.Deck)

	// 设置庄家
	room.DealerIndex = (room.DealerIndex + 1) % len(room.Players)

	// 设置大小盲注
	smallBlindIndex := (room.DealerIndex + 1) % len(room.Players)
	bigBlindIndex := (room.DealerIndex + 2) % len(room.Players)

	for i, p := range room.Players {
		p.IsDealer = (i == room.DealerIndex)
		p.IsSmall = (i == smallBlindIndex)
		p.IsBig = (i == bigBlindIndex)
	}

	// 按座位顺序发牌（从庄家下一位开始，发两轮）
	// 第一轮：每人发一张
	for round := 0; round < 2; round++ {
		for i := 0; i < len(room.Players); i++ {
			playerIndex := (room.DealerIndex + 1 + i) % len(room.Players)
			card, err := drawCard(&room.Deck)
			if err != nil {
				log.Printf("发牌失败: %v，房间 %s", err, room.ID)
				// 重新创建并洗牌
				room.Deck = createDeck()
				shuffleDeck(room.Deck)
				card, err = drawCard(&room.Deck)
				if err != nil {
					log.Printf("严重错误：重新洗牌后仍然无法发牌: %v", err)
					return
				}
			}
			room.Players[playerIndex].Hand = append(room.Players[playerIndex].Hand, card)
		}
	}

	// 下大小盲注
	room.Players[smallBlindIndex].Bet = SMALL_BLIND
	room.Players[smallBlindIndex].Chips -= SMALL_BLIND
	room.Players[bigBlindIndex].Bet = BIG_BLIND
	room.Players[bigBlindIndex].Chips -= BIG_BLIND

	room.Pot = SMALL_BLIND + BIG_BLIND
	room.CurrentBet = BIG_BLIND
	room.CurrentTurn = (bigBlindIndex + 1) % len(room.Players)
	// 在翻牌前，初始化为-1，表示还没有人加注（大盲注不算加注，只是初始下注）
	// 在nextTurn中会特殊处理翻牌前的情况，确保大盲注也行动后才能进入下一阶段
	room.LastRaiseIndex = -1
	room.BettingStartIndex = (bigBlindIndex + 1) % len(room.Players) // 翻牌前从大盲注下一位开始

	// 跳过已弃牌和全押的玩家，找到第一个可以行动的玩家并启动超时定时器
	startTurn := room.CurrentTurn
	for i := 0; i < len(room.Players); i++ {
		p := room.Players[room.CurrentTurn]
		if !p.Folded && !p.AllIn {
			// 启动超时定时器（1分钟）
			room.startTurnTimer()
			break
		}
		room.CurrentTurn = (room.CurrentTurn + 1) % len(room.Players)
		// 如果转了一圈还没找到，说明所有玩家都已行动或全押
		if room.CurrentTurn == startTurn {
			break
		}
	}

	// 准备广播消息（需要在锁外发送）
	// 先复制玩家列表、观战列表和等待列表（必须在锁内复制）
	players := make([]*Player, len(room.Players))
	copy(players, room.Players)
	spectators := make([]*Player, len(room.Spectators))
	copy(spectators, room.Spectators)
	waitingPlayers := make([]*Player, len(room.WaitingPlayers))
	copy(waitingPlayers, room.WaitingPlayers)
	log.Printf("玩家列表已复制，房间 %s，玩家数: %d，观战数: %d，等待玩家数: %d", room.ID, len(players), len(spectators), len(waitingPlayers))

	// 释放写锁，然后序列化数据
	room.Mutex.Unlock()
	log.Printf("锁已释放，准备序列化房间数据，房间 %s", room.ID)

	// 现在可以安全地调用ToJSON()了（它会获取读锁）
	roomData := room.ToJSON()
	log.Printf("房间数据序列化完成，房间 %s", room.ID)

	msg := Message{
		Type: "gameStarted",
		Data: roomData,
	}
	log.Printf("准备广播游戏开始消息，房间 %s，玩家数: %d，等待玩家数: %d", room.ID, len(players), len(waitingPlayers))
	for i, p := range players {
		if p.Conn != nil {
			log.Printf("发送游戏开始消息给玩家 %d: %s (ID: %s)", i, p.Name, p.ID)
			sendMessage(p, msg)
		} else {
			log.Printf("警告: 玩家 %s 连接为空，跳过", p.Name)
		}
	}
	// 给观战者发送游戏开始消息（可以看到游戏状态）
	for i, p := range spectators {
		if p.Conn != nil {
			log.Printf("发送游戏开始消息给观战者 %d: %s (ID: %s)", i, p.Name, p.ID)
			sendMessage(p, msg)
		}
	}
	// 给等待列表中的玩家发送等待消息（不参与当前游戏）
	for i, p := range waitingPlayers {
		if p.Conn != nil {
			log.Printf("发送等待消息给等待玩家 %d: %s (ID: %s)", i, p.Name, p.ID)
			waitingMsg := Message{
				Type: "gameWaiting",
				Data: map[string]interface{}{
					"room":      roomData,
					"message":   "游戏正在进行中，请等待下一局开始",
					"isWaiting": true,
				},
			}
			sendMessage(p, waitingMsg)
		}
	}

	log.Printf("✅ 游戏已开始，房间 %s，已广播给 %d 个玩家，%d 个观战者，%d 个等待玩家收到等待消息", room.ID, len(players), len(spectators), len(waitingPlayers))
}

func handleAction(player *Player, msg *Message) {
	room := findPlayerRoom(player)
	if room == nil {
		return
	}

	room.Mutex.Lock()
	// 注意：不在defer中解锁，因为需要在函数中间解锁

	// 取消当前回合的超时定时器
	if room.TurnTimer != nil {
		room.TurnTimer.Stop()
		room.TurnTimer = nil
	}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		room.Mutex.Unlock()
		return
	}

		action, _ := data["action"].(string)
		amount, _ := data["amount"].(float64)

		playerIndex := -1
	for i, p := range room.Players {
		if p.ID == player.ID {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 || room.Players[playerIndex].ID != room.Players[room.CurrentTurn].ID {
		room.Mutex.Unlock()
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "不是你的回合"},
		})
		return
	}

	switch action {
	case "fold":
		room.Players[playerIndex].Folded = true
	case "call":
		callAmount := room.CurrentBet - room.Players[playerIndex].Bet
		if callAmount < 0 {
			callAmount = 0 // 如果已经下注超过当前最高下注，则不需要跟注
		}
		if callAmount > room.Players[playerIndex].Chips {
			callAmount = room.Players[playerIndex].Chips
			room.Players[playerIndex].AllIn = true
		}
		if callAmount > 0 {
			room.Players[playerIndex].Bet += callAmount
			room.Players[playerIndex].Chips -= callAmount
			room.Pot += callAmount
			// 如果跟注后筹码为0，确保AllIn标志已设置
			if room.Players[playerIndex].Chips == 0 {
				room.Players[playerIndex].AllIn = true
				log.Printf("玩家 %s 跟注后筹码为0，设置AllIn标志，房间 %s", room.Players[playerIndex].Name, room.ID)
			}
		}
	case "raise":
		raiseAmount := int(amount)
		// 验证最小加注金额（最小加注 = 大盲注）
		if raiseAmount < BIG_BLIND {
			room.Mutex.Unlock()
			sendMessage(player, Message{
				Type: "error",
				Data: map[string]string{"message": fmt.Sprintf("最小加注金额为 %d", BIG_BLIND)},
			})
			return
		}

		// 计算需要下注的总金额
		currentPlayerBet := room.Players[playerIndex].Bet
		// 满池：如果加注金额等于底池，那么新的总下注 = 当前玩家下注 + 底池金额
		// 否则：新的总下注 = 当前最高下注 + 加注金额
		var newTotalBet int
		if raiseAmount == room.Pot {
			// 满池：下注金额等于底池
			newTotalBet = currentPlayerBet + raiseAmount
		} else {
			// 普通加注：在当前最高下注基础上加注
			newTotalBet = room.CurrentBet + raiseAmount
		}

		// 检查筹码是否足够
		totalNeeded := newTotalBet - currentPlayerBet
		if totalNeeded <= 0 {
			// 如果计算出的需要金额为0或负数，说明加注金额无效
			room.Mutex.Unlock()
			sendMessage(player, Message{
				Type: "error",
				Data: map[string]string{"message": "加注金额无效"},
			})
			return
		}

		if totalNeeded > room.Players[playerIndex].Chips {
			// 全押
			totalNeeded = room.Players[playerIndex].Chips
			if totalNeeded <= 0 {
				// 玩家没有筹码
				room.Mutex.Unlock()
				sendMessage(player, Message{
					Type: "error",
					Data: map[string]string{"message": "筹码不足"},
				})
				return
			}
			room.Players[playerIndex].AllIn = true
			newTotalBet = currentPlayerBet + totalNeeded
		}

		// 更新玩家下注和筹码
		room.Players[playerIndex].Bet = newTotalBet
		room.Players[playerIndex].Chips -= totalNeeded
		room.Pot += totalNeeded

		// 如果玩家全押后筹码为0，确保AllIn标志已设置
		if room.Players[playerIndex].Chips == 0 {
			room.Players[playerIndex].AllIn = true
			log.Printf("玩家 %s 全押后筹码为0，设置AllIn标志，房间 %s", room.Players[playerIndex].Name, room.ID)
		}

		// 更新当前最高下注和最后加注位置
		if newTotalBet > room.CurrentBet {
			room.CurrentBet = newTotalBet
			room.LastRaiseIndex = playerIndex // 记录最后加注的玩家
		}
	case "check":
		// 检查是否可以过牌
		if room.Players[playerIndex].Bet < room.CurrentBet {
			room.Mutex.Unlock()
			sendMessage(player, Message{
				Type: "error",
				Data: map[string]string{"message": "不能过牌，需要跟注或加注"},
			})
			return
		}
	}

	// 移动到下一个玩家
	gameEnded := nextTurn(room)

	// 如果游戏结束，nextTurn已经释放了锁，直接返回
	if gameEnded {
		return
	}

	// 准备广播消息（需要在锁外发送）
	// 包括等待列表和观战列表中的玩家，让他们也能看到游戏状态
	players := make([]*Player, len(room.Players))
	copy(players, room.Players)
	spectators := make([]*Player, len(room.Spectators))
	copy(spectators, room.Spectators)
	waitingPlayers := make([]*Player, len(room.WaitingPlayers))
	copy(waitingPlayers, room.WaitingPlayers)
	room.Mutex.Unlock()

	// 序列化数据并广播（此时锁已释放）
	roomData := room.ToJSON()
	broadcastMsg := Message{
		Type: "actionTaken",
		Data: roomData,
	}
	// 广播给游戏中的玩家
	for _, p := range players {
		if p.Conn != nil {
			sendMessage(p, broadcastMsg)
		}
	}
	// 广播给观战者
	for _, p := range spectators {
		if p.Conn != nil {
			sendMessage(p, broadcastMsg)
		}
	}
	// 也广播给等待列表中的玩家
	for _, p := range waitingPlayers {
		if p.Conn != nil {
			sendMessage(p, broadcastMsg)
		}
	}
	// 函数结束，不需要重新加锁
}

func nextTurn(room *GameRoom) bool {
	// 注意：调用此函数时应该持有写锁
	// 返回值：true表示游戏结束且锁已释放，false表示游戏继续且锁还在
	// 检查是否只剩一个未弃牌玩家，如果是则自动获胜
	activePlayers := []*Player{}
	for _, p := range room.Players {
		if !p.Folded {
			activePlayers = append(activePlayers, p)
		}
	}

	// 如果只剩一个未弃牌玩家，自动获胜
	if len(activePlayers) == 1 {
		log.Printf("只剩一个未弃牌玩家 %s，自动获胜，房间 %s", activePlayers[0].Name, room.ID)
		activePlayers[0].Chips += room.Pot
		// 保存获胜者筹码
		savePlayerChips(room.ID, activePlayers[0].Name, activePlayers[0].Chips)
		room.GamePhase = "showdown"
		
		// 保存所有玩家的筹码
		for _, p := range room.Players {
			savePlayerChips(room.ID, p.Name, p.Chips)
		}
		
		// 检查并处理心跳超时的玩家（游戏结束后移入观战）
		timeoutPlayers := []*Player{}
		for _, p := range room.Players {
			if p.HeartbeatTimeout {
				timeoutPlayers = append(timeoutPlayers, p)
			}
		}
		
		// 将心跳超时的玩家移入观战
		for _, p := range timeoutPlayers {
			log.Printf("游戏结束，将心跳超时的玩家 %s 移入观战", p.Name)
			// 从游戏玩家列表移除
			for i, player := range room.Players {
				if player.ID == p.ID {
					room.Players = append(room.Players[:i], room.Players[i+1:]...)
					break
				}
			}
			// 添加到观战列表（如果不在）
			inSpectators := false
			for _, sp := range room.Spectators {
				if sp.ID == p.ID {
					inSpectators = true
					break
				}
			}
			if !inSpectators {
				p.Status = PlayerStatusSpectating
				p.HeartbeatTimeout = false
				room.Spectators = append(room.Spectators, p)
			}
		}
		
		// 准备广播消息（在释放锁之前复制所有需要的数据）
		players := make([]*Player, len(room.Players))
		copy(players, room.Players)
		spectatorsForGameEnd := make([]*Player, len(room.Spectators))
		copy(spectatorsForGameEnd, room.Spectators)
		// 复制等待列表中的玩家（观战者）- 必须在锁内复制
		waitingPlayersForGameEnd := make([]*Player, len(room.WaitingPlayers))
		copy(waitingPlayersForGameEnd, room.WaitingPlayers)
		potCopy := room.Pot
		communityCardsCopy := make([]Card, len(room.CommunityCards))
		copy(communityCardsCopy, room.CommunityCards)
		room.Mutex.Unlock()

		// 准备所有玩家的手牌信息
		allPlayersHands := make([]map[string]interface{}, len(players))
		for i, p := range players {
			allPlayersHands[i] = map[string]interface{}{
				"id":     p.ID,
				"name":   p.Name,
				"hand":   p.Hand,
				"folded": p.Folded,
				"chips":  p.Chips,
			}
		}

		msg := Message{
			Type: "gameEnded",
			Data: map[string]interface{}{
				"winner":         activePlayers[0],
				"pot":            potCopy,
				"winningHand":    "",
				"allHands":       allPlayersHands,
				"communityCards": communityCardsCopy,
			},
		}
		// 广播给游戏中的玩家
		for _, p := range players {
			if p.Conn != nil {
				sendMessage(p, msg)
			}
		}
		// 也广播给观战者
		for _, p := range spectatorsForGameEnd {
			if p.Conn != nil {
				sendMessage(p, msg)
			}
		}
		// 也广播给等待列表中的玩家
		for _, p := range waitingPlayersForGameEnd {
			if p.Conn != nil {
				sendMessage(p, msg)
			}
		}

		// 游戏结束后，将游戏状态重置为waiting，让等待的玩家可以加入
		roomsMutex.RLock()
		r, exists := rooms[room.ID]
		roomsMutex.RUnlock()

		if exists {
			r.Mutex.Lock()
			// 停止超时定时器
			if r.TurnTimer != nil {
				r.TurnTimer.Stop()
				r.TurnTimer = nil
				log.Printf("游戏结束，已停止超时定时器，房间 %s", r.ID)
			}
			r.GamePhase = "waiting"
			// 重置游戏状态（为新一局游戏做准备）
			r.Pot = 0
			r.CurrentBet = 0
			r.CommunityCards = []Card{}
			r.LastRaiseIndex = -1
			r.BettingStartIndex = -1
			r.CurrentTurn = -1
			// 重置DealerIndex（如果玩家数变化，需要确保索引有效）
			if r.DealerIndex >= len(r.Players) {
				r.DealerIndex = 0
			}
			// 重置所有玩家的游戏状态
			for _, p := range r.Players {
				p.Hand = []Card{}
				p.Bet = 0
				p.Folded = false
				p.AllIn = false
				p.IsDealer = false
				p.IsSmall = false
				p.IsBig = false
			}
			// 将等待列表中的玩家加入到游戏中
			if len(r.WaitingPlayers) > 0 {
				log.Printf("游戏结束，将 %d 个等待玩家加入到游戏中，房间 %s", len(r.WaitingPlayers), r.ID)
				for _, waitingPlayer := range r.WaitingPlayers {
					if len(r.Players) < MAX_PLAYERS {
						r.Players = append(r.Players, waitingPlayer)
						waitingPlayer.Hand = []Card{}
						waitingPlayer.Bet = 0
						waitingPlayer.Folded = false
						waitingPlayer.AllIn = false
						waitingPlayer.IsDealer = false
						waitingPlayer.IsSmall = false
						waitingPlayer.IsBig = false
						if waitingPlayer.Chips == 0 {
							waitingPlayer.Chips = INITIAL_CHIPS
						}
						log.Printf("等待玩家 %s 已加入游戏，房间 %s，当前玩家数: %d", waitingPlayer.Name, r.ID, len(r.Players))
					}
				}
				r.WaitingPlayers = []*Player{}

				allPlayers := make([]*Player, len(r.Players))
				copy(allPlayers, r.Players)
				// 先释放写锁，再调用ToJSON（ToJSON需要读锁）
				r.Mutex.Unlock()
				roomData := r.ToJSON()

				updateMsg := Message{
					Type: "roomUpdated",
					Data: map[string]interface{}{
						"room": roomData,
					},
				}
				for _, p := range allPlayers {
					if p.Conn != nil {
						sendMessage(p, updateMsg)
					}
				}
			} else {
				// 即使没有等待玩家，也要广播房间更新，确保所有玩家知道游戏状态已重置
				allPlayers := make([]*Player, len(r.Players))
				copy(allPlayers, r.Players)
				// 先释放写锁，再调用ToJSON（ToJSON需要读锁）
				r.Mutex.Unlock()
				roomData := r.ToJSON()

				updateMsg := Message{
					Type: "roomUpdated",
					Data: map[string]interface{}{
						"room": roomData,
					},
				}
				for _, p := range allPlayers {
					if p.Conn != nil {
						sendMessage(p, updateMsg)
					}
				}
			}
			log.Printf("✅ 游戏状态已重置为waiting，房间 %s，玩家数: %d，游戏阶段: %s", r.ID, len(r.Players), r.GamePhase)
		}
		return true // 游戏结束，锁已释放
	}

	// 检查是否所有玩家都已行动
	playersActed := 0
	for _, p := range room.Players {
		if !p.Folded {
			// 玩家已行动的条件：下注等于当前最高下注，或者全押
			if p.Bet == room.CurrentBet || p.AllIn {
				playersActed++
			}
		}
	}
	activePlayersCount := len(activePlayers)

	// 判断是否可以进入下一阶段：
	// 1. 所有活跃玩家都已行动（下注相等或全押/弃牌）
	// 2. 如果有人在当前轮加注，需要确保从最后加注的玩家开始，所有人都行动过一轮
	canAdvance := false

	// 特殊情况：如果所有活跃玩家都已全押，应该直接进入下一阶段或开牌
	allAllIn := true
	for _, p := range room.Players {
		if !p.Folded && !p.AllIn {
			allAllIn = false
			break
		}
	}

	if allAllIn && activePlayersCount > 1 {
		// 所有活跃玩家都已全押，直接进入下一阶段或开牌
		log.Printf("所有活跃玩家都已全押，直接进入下一阶段或开牌，房间 %s，当前阶段: %s", room.ID, room.GamePhase)
		canAdvance = true
	} else if playersActed == activePlayersCount && activePlayersCount > 1 {
		if room.LastRaiseIndex == -1 {
			// 没有人加注，所有人都跟注或过牌
			// 需要确保所有人都行动过一轮
			if room.GamePhase == "preflop" {
				// 翻牌前，需要轮到大盲注且大盲注已行动
				bigBlindIndex := (room.DealerIndex + 2) % len(room.Players)
				bigBlindPlayer := room.Players[bigBlindIndex]
				// 如果当前轮到大盲注，且大盲注已行动（下注相等或全押），可以进入下一阶段
				if room.CurrentTurn == bigBlindIndex && (bigBlindPlayer.Bet == room.CurrentBet || bigBlindPlayer.AllIn || bigBlindPlayer.Folded) {
					canAdvance = true
				}
			} else {
				// 翻牌后（flop, turn, river），从小盲注开始行动
				// 需要确保从开始行动的玩家开始，所有人都行动过一轮
				// 如果当前轮到的是开始行动的玩家的前一位，说明所有人都行动过一轮了
				lastPlayerIndex := (room.BettingStartIndex - 1 + len(room.Players)) % len(room.Players)
				lastPlayer := room.Players[lastPlayerIndex]
				// 如果当前轮到的是最后应该行动的玩家，且该玩家已行动（下注相等或全押），可以进入下一阶段
				if room.CurrentTurn == lastPlayerIndex && (lastPlayer.Bet == room.CurrentBet || lastPlayer.AllIn || lastPlayer.Folded) {
					canAdvance = true
				}
			}
		} else {
			// 有人加注，需要检查从最后加注的玩家开始，是否所有人都行动过一轮
			// 当有人加注后，轮到下一个玩家行动，依次行动直到回到最后加注的玩家
			// 当轮到最后加注的玩家时，该玩家应该有机会再次行动（过牌或再次加注）
			// 只有当加注者行动后，且所有人都行动过一轮，才能进入下一阶段

			// 计算从最后加注的玩家开始，下一个应该行动的玩家
			nextPlayerAfterRaise := (room.LastRaiseIndex + 1) % len(room.Players)

			// 如果当前轮到的是最后加注玩家的下一位，说明已经转了一圈
			// 此时需要检查最后加注的玩家是否已经再次行动过
			if room.CurrentTurn == nextPlayerAfterRaise {
				lastRaisePlayer := room.Players[room.LastRaiseIndex]
				// 如果最后加注的玩家已经再次行动过（下注等于当前最高下注或全押），说明所有人都行动过一轮了
				if lastRaisePlayer.Bet == room.CurrentBet || lastRaisePlayer.AllIn || lastRaisePlayer.Folded {
					canAdvance = true
				}
			}
		}
	}

	if canAdvance {
		// 进入下一阶段
		room.LastRaiseIndex = -1 // 重置最后加注位置

		// 保存当前游戏阶段，以便检查是否进入showdown
		oldPhase := room.GamePhase

		// 如果所有玩家都已全押，需要特殊处理：直接发完所有公共牌并开牌
		if allAllIn && activePlayersCount > 1 {
			log.Printf("所有玩家都已全押，直接发完公共牌并开牌，房间 %s，当前阶段: %s", room.ID, oldPhase)
			// 如果还没发完所有公共牌，先发完
			if oldPhase == "preflop" {
				// 发翻牌
				for i := 0; i < 3; i++ {
					card, err := drawCard(&room.Deck)
					if err != nil {
						log.Printf("发翻牌失败（全押）: %v，房间 %s", err, room.ID)
						room.Mutex.Unlock()
						return true
					}
					room.CommunityCards = append(room.CommunityCards, card)
				}
				room.GamePhase = "flop"
				log.Printf("发翻牌，房间 %s", room.ID)
			}
			if room.GamePhase == "flop" {
				// 发转牌
				card, err := drawCard(&room.Deck)
				if err != nil {
					log.Printf("发转牌失败（全押）: %v，房间 %s", err, room.ID)
					room.Mutex.Unlock()
					return true
				}
				room.CommunityCards = append(room.CommunityCards, card)
				room.GamePhase = "turn"
				log.Printf("发转牌，房间 %s", room.ID)
			}
			if room.GamePhase == "turn" {
				// 发河牌
				card, err := drawCard(&room.Deck)
				if err != nil {
					log.Printf("发河牌失败（全押）: %v，房间 %s", err, room.ID)
					room.Mutex.Unlock()
					return true
				}
				room.CommunityCards = append(room.CommunityCards, card)
				room.GamePhase = "river"
				log.Printf("发河牌，房间 %s", room.ID)
			}

			// 先广播公共牌更新，让前端显示所有公共牌
			allPlayersForUpdate := make([]*Player, len(room.Players))
			copy(allPlayersForUpdate, room.Players)
			// 在释放锁之前复制需要的数据
			communityCardsCopy := make([]Card, len(room.CommunityCards))
			copy(communityCardsCopy, room.CommunityCards)
			potCopy := room.Pot
			currentBetCopy := room.CurrentBet
			dealerIndexCopy := room.DealerIndex
			currentTurnCopy := room.CurrentTurn
			gamePhaseCopy := room.GamePhase
			// 复制等待列表中的玩家（观战者）- 必须在锁内复制
			waitingPlayersForUpdate := make([]*Player, len(room.WaitingPlayers))
			copy(waitingPlayersForUpdate, room.WaitingPlayers)
			room.Mutex.Unlock()

			// 在锁外构建房间数据
			playersDataForUpdate := make([]map[string]interface{}, len(allPlayersForUpdate))
			for i, p := range allPlayersForUpdate {
				playersDataForUpdate[i] = map[string]interface{}{
					"id":       p.ID,
					"name":     p.Name,
					"chips":    p.Chips,
					"bet":      p.Bet,
					"folded":   p.Folded,
					"allIn":    p.AllIn,
					"hand":     p.Hand,
					"isDealer": p.IsDealer,
					"isSmall":  p.IsSmall,
					"isBig":    p.IsBig,
				}
			}
			roomDataForUpdate := map[string]interface{}{
				"id":             room.ID,
				"players":        playersDataForUpdate,
				"communityCards": communityCardsCopy,
				"pot":            potCopy,
				"currentBet":     currentBetCopy,
				"dealerIndex":    dealerIndexCopy,
				"currentTurn":    currentTurnCopy,
				"gamePhase":      gamePhaseCopy,
			}

			updateMsg := Message{
				Type: "roomUpdated",
				Data: map[string]interface{}{
					"room": roomDataForUpdate,
				},
			}
			// 广播给游戏中的玩家
			for _, p := range allPlayersForUpdate {
				if p.Conn != nil {
					sendMessage(p, updateMsg)
				}
			}
			// 也广播给等待列表中的玩家（观战者）
			for _, p := range waitingPlayersForUpdate {
				if p.Conn != nil {
					sendMessage(p, updateMsg)
				}
			}

			// 等待一小段时间让前端显示公共牌
			time.Sleep(500 * time.Millisecond)

			// 重新获取锁并进入比牌阶段
			roomsMutex.RLock()
			r, exists := rooms[room.ID]
			roomsMutex.RUnlock()
			if !exists {
				return true
			}
			r.Mutex.Lock()
			room = r
			room.GamePhase = "showdown"
			determineWinner(room)
			return true // 游戏结束，锁已被determineWinner释放
		}

		// 如果当前阶段是river，调用advancePhase会进入showdown并调用determineWinner
		// determineWinner会释放锁，所以需要特殊处理
		if oldPhase == "river" {
			// 直接调用advancePhase，它会调用determineWinner并释放锁
			advancePhase(room)
			// determineWinner已经释放了锁，直接返回
			// 注意：此时不能再访问room，因为锁已经被释放
			return true // 游戏结束，锁已被determineWinner释放
		}

		// 其他阶段，正常调用advancePhase
		advancePhase(room)
		// advancePhase不会释放锁，所以可以继续访问room
	} else {
		// 移动到下一个未弃牌且未全押的玩家
		startTurn := room.CurrentTurn
		foundNextPlayer := false
		for i := 0; i < len(room.Players); i++ {
			room.CurrentTurn = (room.CurrentTurn + 1) % len(room.Players)
			p := room.Players[room.CurrentTurn]
			// 找到下一个可以行动的玩家（未弃牌且未全押）
			if !p.Folded && !p.AllIn {
				// 检查玩家是否心跳超时，如果是，自动执行操作
				if p.HeartbeatTimeout {
					log.Printf("玩家 %s 轮到行动但心跳超时，自动执行操作", p.Name)
					// 取消当前回合的超时定时器（如果存在）
					if room.TurnTimer != nil {
						room.TurnTimer.Stop()
						room.TurnTimer = nil
					}
					// 检查是否可以过牌
					if p.Bet == room.CurrentBet {
						// 可以过牌，自动过牌
						log.Printf("玩家 %s 自动过牌（下注已匹配）", p.Name)
						// 过牌不需要改变状态，继续循环找下一个玩家
						continue
					} else {
						// 无法过牌，自动弃牌
						log.Printf("玩家 %s 无法过牌（需要跟注 %d），自动弃牌", p.Name, room.CurrentBet-p.Bet)
						p.Folded = true
						// 继续循环找下一个玩家
						continue
					}
				}
				// 启动超时定时器（1分钟）
				room.startTurnTimer()
				foundNextPlayer = true
				break
			}
			// 如果转了一圈还没找到，说明所有玩家都已行动或全押
			if room.CurrentTurn == startTurn {
				break
			}
		}

		// 如果找不到下一个可以行动的玩家，说明所有玩家都已全押或弃牌
		// 应该直接开牌（进入比牌阶段）
		if !foundNextPlayer {
			log.Printf("所有玩家都已全押或弃牌，无人可以行动，直接开牌，房间 %s，当前阶段: %s", room.ID, room.GamePhase)

			// 重新计算活跃玩家
			remainingActivePlayers := []*Player{}
			for _, p := range room.Players {
				if !p.Folded {
					remainingActivePlayers = append(remainingActivePlayers, p)
				}
			}

			if len(remainingActivePlayers) > 1 {
				// 如果还没发完所有公共牌，先发完
				if room.GamePhase == "preflop" {
					// 发翻牌
					for i := 0; i < 3; i++ {
						card, err := drawCard(&room.Deck)
						if err != nil {
							log.Printf("发翻牌失败（无人可行动）: %v，房间 %s", err, room.ID)
							room.Mutex.Unlock()
							return true
						}
						room.CommunityCards = append(room.CommunityCards, card)
					}
					room.GamePhase = "flop"
					log.Printf("发翻牌，房间 %s", room.ID)
				}
				if room.GamePhase == "flop" {
					// 发转牌
					card, err := drawCard(&room.Deck)
					if err != nil {
						log.Printf("发转牌失败（无人可行动）: %v，房间 %s", err, room.ID)
						room.Mutex.Unlock()
						return true
					}
					room.CommunityCards = append(room.CommunityCards, card)
					room.GamePhase = "turn"
					log.Printf("发转牌，房间 %s", room.ID)
				}
				if room.GamePhase == "turn" {
					// 发河牌
					card, err := drawCard(&room.Deck)
					if err != nil {
						log.Printf("发河牌失败（无人可行动）: %v，房间 %s", err, room.ID)
						room.Mutex.Unlock()
						return true
					}
					room.CommunityCards = append(room.CommunityCards, card)
					room.GamePhase = "river"
					log.Printf("发河牌，房间 %s", room.ID)
				}

				// 先广播公共牌更新，让前端显示所有公共牌
				allPlayersForUpdate := make([]*Player, len(room.Players))
				copy(allPlayersForUpdate, room.Players)
				// 在释放锁之前复制需要的数据
				communityCardsCopy := make([]Card, len(room.CommunityCards))
				copy(communityCardsCopy, room.CommunityCards)
				potCopy := room.Pot
				currentBetCopy := room.CurrentBet
				dealerIndexCopy := room.DealerIndex
				currentTurnCopy := room.CurrentTurn
				gamePhaseCopy := room.GamePhase
				// 复制等待列表中的玩家（观战者）
				waitingPlayersForUpdate := make([]*Player, len(room.WaitingPlayers))
				copy(waitingPlayersForUpdate, room.WaitingPlayers)
				room.Mutex.Unlock()

				// 在锁外构建房间数据
				playersDataForUpdate := make([]map[string]interface{}, len(allPlayersForUpdate))
				for i, p := range allPlayersForUpdate {
					playersDataForUpdate[i] = map[string]interface{}{
						"id":       p.ID,
						"name":     p.Name,
						"chips":    p.Chips,
						"bet":      p.Bet,
						"folded":   p.Folded,
						"allIn":    p.AllIn,
						"hand":     p.Hand,
						"isDealer": p.IsDealer,
						"isSmall":  p.IsSmall,
						"isBig":    p.IsBig,
					}
				}
				roomDataForUpdate := map[string]interface{}{
					"id":             room.ID,
					"players":        playersDataForUpdate,
					"communityCards": communityCardsCopy,
					"pot":            potCopy,
					"currentBet":     currentBetCopy,
					"dealerIndex":    dealerIndexCopy,
					"currentTurn":    currentTurnCopy,
					"gamePhase":      gamePhaseCopy,
				}

				// 广播公共牌更新（包括等待列表中的玩家）
				updateMsg := Message{
					Type: "roomUpdated",
					Data: map[string]interface{}{
						"room": roomDataForUpdate,
					},
				}
				// 广播给游戏中的玩家
				for _, p := range allPlayersForUpdate {
					if p.Conn != nil {
						sendMessage(p, updateMsg)
					}
				}
				// 也广播给等待列表中的玩家（观战者）
				for _, p := range waitingPlayersForUpdate {
					if p.Conn != nil {
						sendMessage(p, updateMsg)
					}
				}

				// 等待一小段时间让前端显示公共牌
				time.Sleep(500 * time.Millisecond)

				// 重新获取锁并进入比牌阶段
				room.Mutex.Lock()
				room.GamePhase = "showdown"
				room.LastRaiseIndex = -1
				log.Printf("所有玩家都已全押，直接进入比牌阶段，房间 %s", room.ID)

				// 调用determineWinner（会释放锁）
				determineWinner(room)
				return true // 游戏结束，锁已被determineWinner释放
			} else if len(remainingActivePlayers) == 1 {
				// 只剩一个玩家，直接获胜
				log.Printf("只剩一个活跃玩家，直接获胜，房间 %s", room.ID)
				remainingActivePlayers[0].Chips += room.Pot
				room.GamePhase = "showdown"
				// 准备广播消息（在释放锁之前复制所有需要的数据）
				players := make([]*Player, len(room.Players))
				copy(players, room.Players)
				waitingPlayersForGameEnd := make([]*Player, len(room.WaitingPlayers))
				copy(waitingPlayersForGameEnd, room.WaitingPlayers)
				potCopy := room.Pot
				communityCardsCopy := make([]Card, len(room.CommunityCards))
				copy(communityCardsCopy, room.CommunityCards)
				winnerCopy := remainingActivePlayers[0]
				room.Mutex.Unlock()

				// 准备所有玩家的手牌信息
				allPlayersHands := make([]map[string]interface{}, len(players))
				for i, p := range players {
					allPlayersHands[i] = map[string]interface{}{
						"id":     p.ID,
						"name":   p.Name,
						"hand":   p.Hand,
						"folded": p.Folded,
						"chips":  p.Chips,
					}
				}

				msg := Message{
					Type: "gameEnded",
					Data: map[string]interface{}{
						"winner":         winnerCopy,
						"pot":            potCopy,
						"winningHand":    "",
						"allHands":       allPlayersHands,
						"communityCards": communityCardsCopy,
					},
				}
				for _, p := range players {
					if p.Conn != nil {
						sendMessage(p, msg)
					}
				}
				// 也广播给等待列表中的玩家（观战者）
				for _, p := range waitingPlayersForGameEnd {
					if p.Conn != nil {
						sendMessage(p, msg)
					}
				}

				// 游戏结束后，将游戏状态重置为waiting
				roomsMutex.RLock()
				r, exists := rooms[room.ID]
				roomsMutex.RUnlock()

				if exists {
					r.Mutex.Lock()
					// 停止超时定时器
					if r.TurnTimer != nil {
						r.TurnTimer.Stop()
						r.TurnTimer = nil
						log.Printf("游戏结束，已停止超时定时器，房间 %s", r.ID)
					}
					r.GamePhase = "waiting"
					// 重置游戏状态（为新一局游戏做准备）
					r.Pot = 0
					r.CurrentBet = 0
					r.CommunityCards = []Card{}
					r.LastRaiseIndex = -1
					r.BettingStartIndex = -1
					r.CurrentTurn = -1
					// 重置DealerIndex（如果玩家数变化，需要确保索引有效）
					if r.DealerIndex >= len(r.Players) {
						r.DealerIndex = 0
					}
					// 重置所有玩家的游戏状态
					for _, p := range r.Players {
						p.Hand = []Card{}
						p.Bet = 0
						p.Folded = false
						p.AllIn = false
						p.IsDealer = false
						p.IsSmall = false
						p.IsBig = false
					}
					// 将等待列表中的玩家加入到游戏中
					if len(r.WaitingPlayers) > 0 {
						log.Printf("游戏结束，将 %d 个等待玩家加入到游戏中，房间 %s", len(r.WaitingPlayers), r.ID)
						for _, waitingPlayer := range r.WaitingPlayers {
							if len(r.Players) < MAX_PLAYERS {
								r.Players = append(r.Players, waitingPlayer)
								waitingPlayer.Hand = []Card{}
								waitingPlayer.Bet = 0
								waitingPlayer.Folded = false
								waitingPlayer.AllIn = false
								waitingPlayer.IsDealer = false
								waitingPlayer.IsSmall = false
								waitingPlayer.IsBig = false
								if waitingPlayer.Chips == 0 {
									waitingPlayer.Chips = 1000
								}
								log.Printf("等待玩家 %s 已加入游戏，房间 %s，当前玩家数: %d", waitingPlayer.Name, r.ID, len(r.Players))
							}
						}
						r.WaitingPlayers = []*Player{}

						allPlayers := make([]*Player, len(r.Players))
						copy(allPlayers, r.Players)
						// 先释放写锁，再调用ToJSON（ToJSON需要读锁）
						r.Mutex.Unlock()
						roomData := r.ToJSON()

						updateMsg := Message{
							Type: "roomUpdated",
							Data: map[string]interface{}{
								"room": roomData,
							},
						}
						for _, p := range allPlayers {
							if p.Conn != nil {
								sendMessage(p, updateMsg)
							}
						}
					} else {
						// 即使没有等待玩家，也要广播房间更新，确保所有玩家知道游戏状态已重置
						allPlayers := make([]*Player, len(r.Players))
						copy(allPlayers, r.Players)
						// 先释放写锁，再调用ToJSON（ToJSON需要读锁）
						r.Mutex.Unlock()
						roomData := r.ToJSON()

						updateMsg := Message{
							Type: "roomUpdated",
							Data: map[string]interface{}{
								"room": roomData,
							},
						}
						for _, p := range allPlayers {
							if p.Conn != nil {
								sendMessage(p, updateMsg)
							}
						}
					}
					log.Printf("✅ 游戏状态已重置为waiting，房间 %s，玩家数: %d，游戏阶段: %s", r.ID, len(r.Players), r.GamePhase)
				}
				return true
			}
		}
	}
	return false // 游戏继续，锁还在
}

// 启动回合超时定时器
func (room *GameRoom) startTurnTimer() {
	// 取消之前的定时器
	if room.TurnTimer != nil {
		room.TurnTimer.Stop()
		room.TurnTimer = nil
	}

	// 检查当前玩家是否有效
	if room.CurrentTurn < 0 || room.CurrentTurn >= len(room.Players) {
		return
	}

	currentPlayer := room.Players[room.CurrentTurn]
	if currentPlayer.Folded || currentPlayer.AllIn {
		return
	}

	// 保存房间ID和玩家索引，避免在goroutine中访问room
	roomID := room.ID
	playerIndex := room.CurrentTurn

	// 创建新的定时器
	room.TurnTimer = time.AfterFunc(TURN_TIMEOUT*time.Second, func() {
		// 超时处理
		roomsMutex.RLock()
		r, exists := rooms[roomID]
		roomsMutex.RUnlock()

		if !exists {
			return
		}

		r.Mutex.Lock()

		// 检查游戏状态和当前回合
		if r.GamePhase == "showdown" || r.GamePhase == "waiting" {
			r.Mutex.Unlock()
			return
		}

		// 检查玩家列表是否有效
		if len(r.Players) == 0 {
			r.Mutex.Unlock()
			log.Printf("警告：房间 %s 的玩家列表为空，取消超时处理", roomID)
			return
		}

		// 检查玩家索引是否有效
		if playerIndex < 0 || playerIndex >= len(r.Players) {
			r.Mutex.Unlock()
			log.Printf("警告：玩家索引 %d 无效，房间 %s 的玩家数: %d，取消超时处理", playerIndex, roomID, len(r.Players))
			return
		}

		// 检查当前回合是否还是这个玩家
		if r.CurrentTurn != playerIndex {
			r.Mutex.Unlock()
			return
		}

		player := r.Players[playerIndex]
		if player == nil {
			r.Mutex.Unlock()
			log.Printf("警告：玩家索引 %d 处的玩家为nil，房间 %s，取消超时处理", playerIndex, roomID)
			return
		}

		if player.Folded || player.AllIn {
			r.Mutex.Unlock()
			return
		}

		log.Printf("玩家 %s 超时，自动行动，房间 %s，当前下注: %d，玩家下注: %d", player.Name, roomID, r.CurrentBet, player.Bet)

		// 检查是否可以过牌
		if player.Bet == r.CurrentBet {
			// 可以过牌，自动过牌
			log.Printf("玩家 %s 自动过牌（下注已匹配）", player.Name)
			// 过牌不需要改变状态，直接进入下一回合
		} else {
			// 无法过牌，自动弃牌
			log.Printf("玩家 %s 无法过牌（需要跟注 %d），自动弃牌", player.Name, r.CurrentBet-player.Bet)
			player.Folded = true
		}

		// 移动到下一个玩家
		log.Printf("超时处理：调用nextTurn，房间 %s", roomID)
		gameEnded := nextTurn(r)
		log.Printf("超时处理：nextTurn返回，游戏结束: %v，房间 %s", gameEnded, roomID)

		// 如果游戏结束，nextTurn已经释放了锁，直接返回
		if gameEnded {
			return
		}

		// 准备广播消息
		players := make([]*Player, len(r.Players))
		copy(players, r.Players)
		r.Mutex.Unlock()

		roomData := r.ToJSON()
		msg := Message{
			Type: "actionTaken",
			Data: roomData,
		}
		for _, p := range players {
			if p.Conn != nil {
				sendMessage(p, msg)
			}
		}
	})
}

func advancePhase(room *GameRoom) {
	// 注意：调用此函数时应该持有写锁
	switch room.GamePhase {
	case "preflop":
		room.GamePhase = "flop"
		// 发3张公共牌（翻牌）
		for i := 0; i < 3; i++ {
			card, err := drawCard(&room.Deck)
			if err != nil {
				log.Printf("发翻牌失败: %v，房间 %s", err, room.ID)
				return
			}
			room.CommunityCards = append(room.CommunityCards, card)
		}
		// 重置下注（新的一轮）
		for _, p := range room.Players {
			p.Bet = 0
		}
		room.CurrentBet = 0
		room.LastRaiseIndex = -1 // 重置最后加注位置
		// 翻牌后从小盲注（庄家下一位）开始
		smallBlindIndex := (room.DealerIndex + 1) % len(room.Players)
		room.CurrentTurn = smallBlindIndex
		room.BettingStartIndex = smallBlindIndex // 记录开始行动的玩家
	case "flop":
		room.GamePhase = "turn"
		// 发第4张公共牌（转牌）
		card, err := drawCard(&room.Deck)
		if err != nil {
			log.Printf("发转牌失败: %v，房间 %s", err, room.ID)
			return
		}
		room.CommunityCards = append(room.CommunityCards, card)
		// 重置下注
		for _, p := range room.Players {
			p.Bet = 0
		}
		room.CurrentBet = 0
		room.LastRaiseIndex = -1 // 重置最后加注位置
		// 从小盲注开始
		smallBlindIndex := (room.DealerIndex + 1) % len(room.Players)
		room.CurrentTurn = smallBlindIndex
		room.BettingStartIndex = smallBlindIndex // 记录开始行动的玩家
	case "turn":
		room.GamePhase = "river"
		// 发第5张公共牌（河牌）
		card, err := drawCard(&room.Deck)
		if err != nil {
			log.Printf("发河牌失败: %v，房间 %s", err, room.ID)
			return
		}
		room.CommunityCards = append(room.CommunityCards, card)
		// 重置下注
		for _, p := range room.Players {
			p.Bet = 0
		}
		room.CurrentBet = 0
		room.LastRaiseIndex = -1 // 重置最后加注位置
		// 从小盲注开始
		smallBlindIndex := (room.DealerIndex + 1) % len(room.Players)
		room.CurrentTurn = smallBlindIndex
		room.BettingStartIndex = smallBlindIndex // 记录开始行动的玩家
	case "river":
		room.GamePhase = "showdown"
		// 比牌（determineWinner会自己释放锁）
		// 注意：determineWinner会释放锁，所以这里不需要return，让调用者知道锁已释放
		determineWinner(room)
		// determineWinner已经释放了锁，这里不应该再访问room
		return
	}

	// 跳过已弃牌和全押的玩家，找到第一个可以行动的玩家
	startTurn := room.CurrentTurn
	for i := 0; i < len(room.Players); i++ {
		p := room.Players[room.CurrentTurn]
		if !p.Folded && !p.AllIn {
			// 启动超时定时器（1分钟）
			room.startTurnTimer()
			break
		}
		room.CurrentTurn = (room.CurrentTurn + 1) % len(room.Players)
		// 如果转了一圈还没找到，说明所有玩家都已行动或全押，进入下一阶段
		if room.CurrentTurn == startTurn {
			// 所有玩家都已行动，应该不会到这里，但为了安全还是处理一下
			break
		}
	}
}

func determineWinner(room *GameRoom) {
	// 注意：调用此函数时应该持有写锁
	activePlayers := []*Player{}
	for _, p := range room.Players {
		if !p.Folded {
			activePlayers = append(activePlayers, p)
		}
	}

	var winners []*Player
	var winningHand string
	pot := room.Pot

	if len(activePlayers) == 1 {
		// 只有一个玩家，直接获胜
		winners = []*Player{activePlayers[0]}
		winners[0].Chips += pot
		winningHand = ""
	} else {
		// 计算每个玩家的最佳牌型，找出所有获胜者（可能打平）
		var bestRank HandRank
		bestRank.Rank = -1 // 初始化为无效值

		for _, p := range activePlayers {
			handRank := evaluateHand(p.Hand, room.CommunityCards)
			comparison := compareHandRanks(handRank, bestRank)

			if comparison > 0 {
				// 发现更好的牌型，重置获胜者列表
				bestRank = handRank
				winners = []*Player{p}
			} else if comparison == 0 {
				// 牌型相同，加入获胜者列表（打平）
				winners = append(winners, p)
			}
		}

		// 如果有多个获胜者，平分底池
		if len(winners) > 1 {
			share := pot / len(winners)
			remainder := pot % len(winners)
			for i, w := range winners {
				w.Chips += share
				// 余数给第一个玩家（或可以随机分配，这里简单处理）
				if i == 0 {
					w.Chips += remainder
				}
			}
			winningHand = bestRank.Description + " (多人打平)"
			log.Printf("多人打平，房间 %s，获胜者数: %d，底池: %d，每人分得: %d，余数: %d",
				room.ID, len(winners), pot, share, remainder)
		} else if len(winners) == 1 {
			// 只有一个获胜者
			winners[0].Chips += pot
			winningHand = bestRank.Description
		} else {
			// 理论上不应该到这里，但为了安全还是处理
			log.Printf("警告：未找到获胜者，房间 %s", room.ID)
			if len(activePlayers) > 0 {
				winners = []*Player{activePlayers[0]}
				winners[0].Chips += pot
			}
		}
	}

	// 保存所有玩家的筹码
	for _, p := range room.Players {
		savePlayerChips(room.ID, p.Name, p.Chips)
	}

	// 检查并处理心跳超时的玩家（游戏结束后移入观战）
	timeoutPlayers := []*Player{}
	for _, p := range room.Players {
		if p.HeartbeatTimeout {
			timeoutPlayers = append(timeoutPlayers, p)
		}
	}
	
	// 将心跳超时的玩家移入观战
	for _, p := range timeoutPlayers {
		log.Printf("游戏结束，将心跳超时的玩家 %s 移入观战", p.Name)
		// 从游戏玩家列表移除
		for i, player := range room.Players {
			if player.ID == p.ID {
				room.Players = append(room.Players[:i], room.Players[i+1:]...)
				break
			}
		}
		// 添加到观战列表（如果不在）
		inSpectators := false
		for _, sp := range room.Spectators {
			if sp.ID == p.ID {
				inSpectators = true
				break
			}
		}
		if !inSpectators {
			p.Status = PlayerStatusSpectating
			p.HeartbeatTimeout = false
			room.Spectators = append(room.Spectators, p)
		}
	}

	// 准备广播消息（需要在锁外发送）
	players := make([]*Player, len(room.Players))
	copy(players, room.Players)
	spectatorsForGameEnd := make([]*Player, len(room.Spectators))
	copy(spectatorsForGameEnd, room.Spectators)
	waitingPlayersForGameEnd := make([]*Player, len(room.WaitingPlayers))
	copy(waitingPlayersForGameEnd, room.WaitingPlayers)
	// 复制公共牌（必须在锁内复制）
	communityCardsCopy := make([]Card, len(room.CommunityCards))
	copy(communityCardsCopy, room.CommunityCards)
	room.Mutex.Unlock()

	// 准备所有玩家的手牌信息（包括已弃牌的玩家）
	allPlayersHands := make([]map[string]interface{}, len(players))
	for i, p := range players {
		allPlayersHands[i] = map[string]interface{}{
			"id":     p.ID,
			"name":   p.Name,
			"hand":   p.Hand,
			"folded": p.Folded,
			"chips":  p.Chips,
		}
	}

	// 广播消息（此时锁已释放）
	// 为了兼容性，winner字段保留第一个获胜者，但添加winners字段
	msgData := map[string]interface{}{
		"pot":            pot,
		"winningHand":    winningHand,
		"allHands":       allPlayersHands,    // 所有玩家的手牌
		"communityCards": communityCardsCopy, // 公共牌（使用复制的数据）
	}

	// 兼容旧代码：winner字段（第一个获胜者）
	if len(winners) > 0 {
		msgData["winner"] = winners[0]
	} else {
		msgData["winner"] = nil
	}

	// 新字段：winners数组（所有获胜者，支持打平）
	winnersData := make([]map[string]interface{}, len(winners))
	for i, w := range winners {
		winnersData[i] = map[string]interface{}{
			"id":    w.ID,
			"name":  w.Name,
			"chips": w.Chips,
		}
	}
	msgData["winners"] = winnersData
	msgData["isTie"] = len(winners) > 1 // 是否打平

	msg := Message{
		Type: "gameEnded",
		Data: msgData,
	}
	// 广播给游戏中的玩家
	for _, p := range players {
		if p.Conn != nil {
			sendMessage(p, msg)
		}
	}
	// 也广播给观战者
	for _, p := range spectatorsForGameEnd {
		if p.Conn != nil {
			sendMessage(p, msg)
		}
	}
	// 也广播给等待列表中的玩家
	for _, p := range waitingPlayersForGameEnd {
		if p.Conn != nil {
			sendMessage(p, msg)
		}
	}

	// 游戏结束后，将游戏状态重置为waiting，让等待的玩家可以加入
	// 注意：这里需要重新获取房间，因为之前已经释放了锁
	roomsMutex.RLock()
	r, exists := rooms[room.ID]
	roomsMutex.RUnlock()

	if exists {
		r.Mutex.Lock()
		// 停止超时定时器
		if r.TurnTimer != nil {
			r.TurnTimer.Stop()
			r.TurnTimer = nil
			log.Printf("游戏结束，已停止超时定时器，房间 %s", r.ID)
		}
		
		// 再次检查并处理心跳超时的玩家（确保所有超时玩家都被移入观战）
		timeoutPlayers := []*Player{}
		for _, p := range r.Players {
			if p.HeartbeatTimeout {
				timeoutPlayers = append(timeoutPlayers, p)
			}
		}
		
		for _, p := range timeoutPlayers {
			log.Printf("游戏结束，将心跳超时的玩家 %s 移入观战", p.Name)
			// 从游戏玩家列表移除
			for i, player := range r.Players {
				if player.ID == p.ID {
					r.Players = append(r.Players[:i], r.Players[i+1:]...)
					break
				}
			}
			// 添加到观战列表（如果不在）
			inSpectators := false
			for _, sp := range r.Spectators {
				if sp.ID == p.ID {
					inSpectators = true
					break
				}
			}
			if !inSpectators {
				p.Status = PlayerStatusSpectating
				p.HeartbeatTimeout = false
				r.Spectators = append(r.Spectators, p)
			}
		}
		
		r.GamePhase = "waiting"
		// 重置游戏状态（为新一局游戏做准备）
		r.Pot = 0
		r.CurrentBet = 0
		r.CommunityCards = []Card{}
		r.LastRaiseIndex = -1
		r.BettingStartIndex = -1
		r.CurrentTurn = -1
		// 重置DealerIndex（如果玩家数变化，需要确保索引有效）
		if r.DealerIndex >= len(r.Players) {
			r.DealerIndex = 0
		}
		// 重置所有玩家的游戏状态
		for _, p := range r.Players {
			p.Hand = []Card{}
			p.Bet = 0
			p.Folded = false
			p.AllIn = false
			p.IsDealer = false
			p.IsSmall = false
			p.IsBig = false
		}
		// 将等待列表中的玩家加入到游戏中
		if len(r.WaitingPlayers) > 0 {
			log.Printf("游戏结束，将 %d 个等待玩家加入到游戏中，房间 %s", len(r.WaitingPlayers), r.ID)
			for _, waitingPlayer := range r.WaitingPlayers {
				// 检查是否超过最大玩家数
				if len(r.Players) < MAX_PLAYERS {
					r.Players = append(r.Players, waitingPlayer)
					// 初始化等待玩家的状态
					waitingPlayer.Hand = []Card{}
					waitingPlayer.Bet = 0
					waitingPlayer.Folded = false
					waitingPlayer.AllIn = false
					waitingPlayer.IsDealer = false
					waitingPlayer.IsSmall = false
					waitingPlayer.IsBig = false
					if waitingPlayer.Chips == 0 {
						waitingPlayer.Chips = INITIAL_CHIPS // 给新玩家初始筹码
					}
					log.Printf("等待玩家 %s 已加入游戏，房间 %s，当前玩家数: %d", waitingPlayer.Name, r.ID, len(r.Players))
				}
			}
			// 清空等待列表
			r.WaitingPlayers = []*Player{}

			// 通知所有玩家房间状态更新
			allPlayers := make([]*Player, len(r.Players))
			copy(allPlayers, r.Players)
			// 先释放写锁，再调用ToJSON（ToJSON需要读锁）
			r.Mutex.Unlock()
			roomData := r.ToJSON()

			// 广播房间更新消息
			updateMsg := Message{
				Type: "roomUpdated",
				Data: map[string]interface{}{
					"room": roomData,
				},
			}
			for _, p := range allPlayers {
				if p.Conn != nil {
					sendMessage(p, updateMsg)
				}
			}
		} else {
			// 即使没有等待玩家，也要广播房间更新，确保所有玩家知道游戏状态已重置
			allPlayers := make([]*Player, len(r.Players))
			copy(allPlayers, r.Players)
			// 先释放写锁，再调用ToJSON（ToJSON需要读锁）
			r.Mutex.Unlock()
			roomData := r.ToJSON()

			updateMsg := Message{
				Type: "roomUpdated",
				Data: map[string]interface{}{
					"room": roomData,
				},
			}
			for _, p := range allPlayers {
				if p.Conn != nil {
					sendMessage(p, updateMsg)
				}
			}
		}
		log.Printf("✅ 游戏状态已重置为waiting，房间 %s，玩家数: %d，游戏阶段: %s", r.ID, len(r.Players), r.GamePhase)
	}
}

func createDeck() []Card {
	suits := []string{"spades", "hearts", "diamonds", "clubs"}
	ranks := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	deck := []Card{}

	for _, suit := range suits {
		for _, rank := range ranks {
			deck = append(deck, Card{Suit: suit, Rank: rank})
		}
	}

	return deck
}

func shuffleDeck(deck []Card) {
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}

// drawCard 从牌组中抽取一张牌
// 返回抽取的牌和错误信息（如果牌组为空）
func drawCard(deck *[]Card) (Card, error) {
	if len(*deck) == 0 {
		return Card{}, fmt.Errorf("deck is empty, cannot draw card")
	}
	card := (*deck)[0]
	*deck = (*deck)[1:]
	return card, nil
}

func findPlayerRoom(player *Player) *GameRoom {
	roomsMutex.RLock()
	defer roomsMutex.RUnlock()

	playerName := player.Name
	if playerName == "" {
		playerName = "未知玩家"
	}
	log.Printf("🔍 findPlayerRoom: 查找玩家 %s (ID: %s) 的房间，当前房间数: %d", playerName, player.ID, len(rooms))
	for roomID, room := range rooms {
		room.Mutex.RLock()
		// 检查玩家是否在游戏列表中
		for i, p := range room.Players {
			if p.ID == player.ID {
				room.Mutex.RUnlock()
				log.Printf("✅ 找到玩家 %s (ID: %s) 在房间 %s 的游戏列表中 (索引: %d)", playerName, player.ID, roomID, i)
				return room
			}
		}
		// 也检查观战列表
		for i, p := range room.Spectators {
			if p.ID == player.ID {
				room.Mutex.RUnlock()
				log.Printf("✅ 找到玩家 %s (ID: %s) 在房间 %s 的观战列表中 (索引: %d)", playerName, player.ID, roomID, i)
				return room
			}
		}
		// 也检查等待列表
		for i, p := range room.WaitingPlayers {
			if p.ID == player.ID {
				room.Mutex.RUnlock()
				log.Printf("✅ 找到玩家 %s (ID: %s) 在房间 %s 的等待列表中 (索引: %d)", playerName, player.ID, roomID, i)
				return room
			}
		}
		room.Mutex.RUnlock()
	}
	log.Printf("❌ 未找到玩家 %s (ID: %s) 的房间，已检查 %d 个房间", playerName, player.ID, len(rooms))
	return nil
}

func removePlayer(player *Player) {
	room := findPlayerRoom(player)
	if room != nil {
		room.Mutex.Lock()
		
		// 从游戏玩家列表中移除
		removed := false
		for i, p := range room.Players {
			if p.ID == player.ID {
				// 保存筹码
				savePlayerChips(room.ID, player.Name, player.Chips)
				room.Players = append(room.Players[:i], room.Players[i+1:]...)
				removed = true
				break
			}
		}
		
		// 从观战列表中移除
		if !removed {
			for i, p := range room.Spectators {
				if p.ID == player.ID {
					// 保存筹码
					savePlayerChips(room.ID, player.Name, player.Chips)
					room.Spectators = append(room.Spectators[:i], room.Spectators[i+1:]...)
					removed = true
					break
				}
			}
		}
		
		// 从等待列表中移除
		if !removed {
			for i, p := range room.WaitingPlayers {
				if p.ID == player.ID {
					savePlayerChips(room.ID, player.Name, player.Chips)
					room.WaitingPlayers = append(room.WaitingPlayers[:i], room.WaitingPlayers[i+1:]...)
					break
				}
			}
		}

		// 准备广播消息（需要在锁外发送）
		players := make([]*Player, len(room.Players))
		copy(players, room.Players)
		spectators := make([]*Player, len(room.Spectators))
		copy(spectators, room.Spectators)
		room.Mutex.Unlock()

		// 序列化数据并广播（此时锁已释放）
		roomData := room.ToJSON()
		msg := Message{
			Type: "playerLeft",
			Data: map[string]interface{}{
				"playerId": player.ID,
				"room":     roomData,
			},
		}
		for _, p := range players {
			if p.Conn != nil {
				sendMessage(p, msg)
			}
		}
		for _, p := range spectators {
			if p.Conn != nil {
				sendMessage(p, msg)
			}
		}
	}
}

func broadcastToRoom(room *GameRoom, msg Message) {
	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	// 广播给游戏中的玩家
	for _, player := range room.Players {
		if player.Conn != nil {
			sendMessage(player, msg)
		}
	}
	// 广播给观战者
	for _, player := range room.Spectators {
		if player.Conn != nil {
			sendMessage(player, msg)
		}
	}
}

func sendMessage(player *Player, msg Message) {
	if player.Conn != nil {
		err := player.Conn.WriteJSON(msg)
		if err != nil {
			log.Printf("发送消息失败 (玩家=%s, 类型=%s): %v", player.ID, msg.Type, err)
		} else {
			log.Printf("消息已发送 (玩家=%s, 类型=%s)", player.ID, msg.Type)
		}
	} else {
		log.Printf("无法发送消息: 玩家连接为空 (玩家=%s, 类型=%s)", player.ID, msg.Type)
	}
}

// 买一手：增加500筹码
func buyHand(player *Player, msg *Message) {
	room := findPlayerRoom(player)
	if room == nil {
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "房间不存在"},
		})
		return
	}

	room.Mutex.Lock()

	// 找到玩家在房间中的位置
	playerIndex := -1
	for i, p := range room.Players {
		if p.ID == player.ID {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 {
		// 检查是否在观战列表中
		spectatorIndex := -1
		for i, p := range room.Spectators {
			if p.ID == player.ID {
				spectatorIndex = i
				break
			}
		}
		
		if spectatorIndex != -1 {
			// 给观战玩家增加筹码
			room.Spectators[spectatorIndex].Chips += 500
			newChips := room.Spectators[spectatorIndex].Chips
			// 保存筹码
			savePlayerChips(room.ID, player.Name, newChips)
			// 增加买一手次数
			if room.BuyHandCount == nil {
				room.BuyHandCount = make(map[string]int)
			}
			room.BuyHandCount[player.Name]++
			log.Printf("观战玩家 %s 买一手，筹码: %d，累计买一手次数: %d", player.Name, newChips, room.BuyHandCount[player.Name])
			room.Mutex.Unlock()
			// 立即发送成功消息
			sendMessage(player, Message{
				Type: "buyHandSuccess",
				Data: map[string]interface{}{
					"chips": newChips,
				},
			})
			return
		}
		
		// 检查是否在等待列表中
		for i, p := range room.WaitingPlayers {
			if p.ID == player.ID {
				// 给等待玩家增加筹码
				room.WaitingPlayers[i].Chips += BUY_IN_AMOUNT
				newChips := room.WaitingPlayers[i].Chips
				// 保存筹码
				savePlayerChips(room.ID, player.Name, newChips)
				// 增加买一手次数
				if room.BuyHandCount == nil {
					room.BuyHandCount = make(map[string]int)
				}
				room.BuyHandCount[player.Name]++
				log.Printf("等待玩家 %s 买一手，筹码: %d，累计买一手次数: %d", player.Name, newChips, room.BuyHandCount[player.Name])
				room.Mutex.Unlock()
				// 立即发送成功消息
				sendMessage(player, Message{
					Type: "buyHandSuccess",
					Data: map[string]interface{}{
						"chips": newChips,
					},
				})
				return
			}
		}
		room.Mutex.Unlock()
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "玩家不在房间中"},
		})
		return
	}

	// 增加筹码
	room.Players[playerIndex].Chips += BUY_IN_AMOUNT
	newChips := room.Players[playerIndex].Chips
	// 保存筹码
	savePlayerChips(room.ID, player.Name, newChips)
	// 增加买一手次数
	if room.BuyHandCount == nil {
		room.BuyHandCount = make(map[string]int)
	}
	room.BuyHandCount[player.Name]++
	log.Printf("玩家 %s 买一手，筹码: %d，累计买一手次数: %d", player.Name, newChips, room.BuyHandCount[player.Name])

	// 立即发送成功消息给玩家（在广播之前）
	room.Mutex.Unlock()
	sendMessage(player, Message{
		Type: "buyHandSuccess",
		Data: map[string]interface{}{
			"chips": newChips,
		},
	})

	// 广播更新（重新获取锁）
	room.Mutex.RLock()
	allPlayers := make([]*Player, len(room.Players))
	copy(allPlayers, room.Players)
	spectators := make([]*Player, len(room.Spectators))
	copy(spectators, room.Spectators)
	waitingPlayers := make([]*Player, len(room.WaitingPlayers))
	copy(waitingPlayers, room.WaitingPlayers)
	roomData := room.ToJSON()
	room.Mutex.RUnlock()

	updateMsg := Message{
		Type: "roomUpdated",
		Data: map[string]interface{}{
			"room": roomData,
		},
	}
	// 广播给游戏中的玩家
	for _, p := range allPlayers {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}
	// 广播给观战者
	for _, p := range spectators {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}
	// 也广播给等待列表中的玩家
	for _, p := range waitingPlayers {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}
}

// 获取买一手次数统计
func getBuyHandStats(player *Player, msg *Message) {
	room := findPlayerRoom(player)
	if room == nil {
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "房间不存在"},
		})
		return
	}

	room.Mutex.RLock()
	buyHandCount := make(map[string]int)
	if room.BuyHandCount != nil {
		// 复制统计数据
		for name, count := range room.BuyHandCount {
			buyHandCount[name] = count
		}
	}
	room.Mutex.RUnlock()

	sendMessage(player, Message{
		Type: "buyHandStats",
		Data: map[string]interface{}{
			"stats": buyHandCount,
		},
	})
}

// 心跳检测
func startHeartbeatCheck(player *Player) {
	ticker := time.NewTicker(5 * time.Second) // 每5秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Since(player.LastHeartbeat) > 30*time.Second {
				log.Printf("玩家 %s 心跳超时（30秒无响应），标记为超时", player.Name)
				// 只标记，等游戏结束后再移入观战
				markPlayerHeartbeatTimeout(player)
				return
			}
		}
	}
}

// 标记玩家心跳超时（游戏进行中自动执行操作，游戏结束后移入观战）
func markPlayerHeartbeatTimeout(player *Player) {
	room := findPlayerRoom(player)
	if room == nil {
		return
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	// 检查玩家是否在游戏中
	playerIndex := -1
	for i, p := range room.Players {
		if p.ID == player.ID {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 {
		// 玩家不在游戏中，可能是观战者，只标记即可
		for _, p := range room.Spectators {
			if p.ID == player.ID {
				p.HeartbeatTimeout = true
				log.Printf("观战玩家 %s 已标记为心跳超时", player.Name)
				return
			}
		}
		return
	}

	// 玩家在游戏中，标记为超时
	room.Players[playerIndex].HeartbeatTimeout = true
	log.Printf("玩家 %s 已标记为心跳超时", player.Name)

	// 如果游戏正在进行中，且是当前回合，自动执行操作
	if room.GamePhase != "waiting" && room.GamePhase != "showdown" {
		if room.CurrentTurn == playerIndex {
			// 是当前回合，立即自动执行操作
			log.Printf("玩家 %s 在游戏中离线且是当前回合，自动执行操作", player.Name)
			handleTimeoutAction(room, playerIndex)
		} else {
			// 不是当前回合，等轮到他的时候会自动处理
			log.Printf("玩家 %s 在游戏中离线但不是当前回合，等轮到他的时候会自动处理", player.Name)
		}
	} else {
		// 游戏未开始或已结束，等游戏结束后再移入观战
		log.Printf("玩家 %s 心跳超时，游戏状态: %s，等游戏结束后移入观战", player.Name, room.GamePhase)
	}
}

// 处理超时玩家的自动操作（弃牌或过牌）
func handleTimeoutAction(room *GameRoom, playerIndex int) {
	if playerIndex < 0 || playerIndex >= len(room.Players) {
		return
	}

	player := room.Players[playerIndex]
	if player == nil || player.Folded || player.AllIn {
		return
	}

	// 取消当前回合的超时定时器
	if room.TurnTimer != nil {
		room.TurnTimer.Stop()
		room.TurnTimer = nil
	}

	log.Printf("玩家 %s 超时，自动行动，房间 %s，当前下注: %d，玩家下注: %d", player.Name, room.ID, room.CurrentBet, player.Bet)

	// 检查是否可以过牌
	if player.Bet == room.CurrentBet {
		// 可以过牌，自动过牌
		log.Printf("玩家 %s 自动过牌（下注已匹配）", player.Name)
		// 过牌不需要改变状态，直接进入下一回合
	} else {
		// 无法过牌，自动弃牌
		log.Printf("玩家 %s 无法过牌（需要跟注 %d），自动弃牌", player.Name, room.CurrentBet-player.Bet)
		player.Folded = true
	}

	// 移动到下一个玩家
	log.Printf("超时处理：调用nextTurn，房间 %s", room.ID)
	gameEnded := nextTurn(room)

	// 如果游戏结束，nextTurn已经释放了锁，直接返回
	if gameEnded {
		return
	}

	// 准备广播消息
	players := make([]*Player, len(room.Players))
	copy(players, room.Players)
	spectators := make([]*Player, len(room.Spectators))
	copy(spectators, room.Spectators)
	waitingPlayers := make([]*Player, len(room.WaitingPlayers))
	copy(waitingPlayers, room.WaitingPlayers)
	roomData := room.ToJSON()
	room.Mutex.Unlock()

	updateMsg := Message{
		Type: "roomUpdated",
		Data: map[string]interface{}{
			"room": roomData,
		},
	}

	// 广播给所有玩家和观战者
	for _, p := range players {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}
	for _, p := range spectators {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}
	for _, p := range waitingPlayers {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}
}

// 将玩家移入观战状态
func movePlayerToSpectating(player *Player) {
	room := findPlayerRoom(player)
	if room == nil {
		return
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	// 从玩家列表中移除
	for i, p := range room.Players {
		if p.ID == player.ID {
			// 保存筹码
			savePlayerChips(room.ID, player.Name, player.Chips)
			// 移除玩家
			room.Players = append(room.Players[:i], room.Players[i+1:]...)
			break
		}
	}

	// 添加到观战列表（如果不在）
	inSpectators := false
	for _, p := range room.Spectators {
		if p.ID == player.ID {
			inSpectators = true
			break
		}
	}
	if !inSpectators {
		player.Status = PlayerStatusSpectating
		room.Spectators = append(room.Spectators, player)
	}

	// 广播更新
	players := make([]*Player, len(room.Players))
	copy(players, room.Players)
	spectators := make([]*Player, len(room.Spectators))
	copy(spectators, room.Spectators)
	room.Mutex.Unlock()

	roomData := room.ToJSON()
	updateMsg := Message{
		Type: "playerMovedToSpectating",
		Data: map[string]interface{}{
			"playerId": player.ID,
			"room":     roomData,
		},
	}

	for _, p := range players {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}
	for _, p := range spectators {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}
}

// 保存玩家筹码
func savePlayerChips(roomID, playerName string, chips int) {
	chipsMutex.Lock()
	defer chipsMutex.Unlock()

	if chipsStorage[roomID] == nil {
		chipsStorage[roomID] = make(map[string]int)
	}
	chipsStorage[roomID][playerName] = chips
	log.Printf("保存玩家筹码: 房间=%s, 玩家=%s, 筹码=%d", roomID, playerName, chips)
}

// 加载玩家筹码
func loadPlayerChips(roomID, playerName string) int {
	chipsMutex.RLock()
	defer chipsMutex.RUnlock()

	if chipsStorage[roomID] != nil {
		if chips, exists := chipsStorage[roomID][playerName]; exists {
			log.Printf("加载玩家筹码: 房间=%s, 玩家=%s, 筹码=%d", roomID, playerName, chips)
			return chips
		}
	}
	// 默认筹码
	return 500
}

// 检查房间中是否有同名玩家
func hasPlayerWithName(room *GameRoom, name string, excludeID string) bool {
	// 检查游戏中的玩家
	for _, p := range room.Players {
		if p.ID != excludeID && p.Name == name {
			return true
		}
	}
	// 检查观战玩家
	for _, p := range room.Spectators {
		if p.ID != excludeID && p.Name == name {
			return true
		}
	}
	// 检查等待玩家
	for _, p := range room.WaitingPlayers {
		if p.ID != excludeID && p.Name == name {
			return true
		}
	}
	return false
}

// 上桌功能
func joinTable(player *Player, msg *Message) {
	log.Printf("上桌请求: 玩家=%s", player.ID)

	room := findPlayerRoom(player)
	if room == nil {
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "未找到房间"},
		})
		return
	}

	room.Mutex.Lock()

	// 检查是否在观战列表中
	inSpectators := false
	spectatorIndex := -1
	for i, p := range room.Spectators {
		if p.ID == player.ID {
			inSpectators = true
			spectatorIndex = i
			break
		}
	}

	if !inSpectators {
		room.Mutex.Unlock()
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "您不在观战列表中"},
		})
		return
	}

	// 检查游戏是否正在进行
	if room.GamePhase != "waiting" {
		room.Mutex.Unlock()
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "游戏正在进行中，无法加入"},
		})
		return
	}

	// 检查房间是否已满
	if len(room.Players) >= MAX_PLAYERS {
		room.Mutex.Unlock()
		sendMessage(player, Message{
			Type: "error",
			Data: map[string]string{"message": "房间已满"},
		})
		return
	}

	// 从观战列表移除
	room.Spectators = append(room.Spectators[:spectatorIndex], room.Spectators[spectatorIndex+1:]...)

	// 添加到游戏玩家列表
	player.Status = PlayerStatusPlaying
	room.Players = append(room.Players, player)

	players := make([]*Player, len(room.Players))
	copy(players, room.Players)
	spectators := make([]*Player, len(room.Spectators))
	copy(spectators, room.Spectators)
	room.Mutex.Unlock()

	// 广播更新
	roomData := room.ToJSON()
	updateMsg := Message{
		Type: "playerJoinedTable",
		Data: map[string]interface{}{
			"player": player,
			"room":   roomData,
		},
	}

	for _, p := range players {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}
	for _, p := range spectators {
		if p.Conn != nil {
			sendMessage(p, updateMsg)
		}
	}

	log.Printf("玩家 %s 成功上桌，房间 %s", player.Name, room.ID)
}

func generateID() string {
	// 生成6位纯数字房间ID
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
