// WebSocket连接
let ws = null;
let currentRoom = null;
let currentPlayer = null;
let gameState = null;
let turnTimer = null; // 回合倒计时定时器
let isSettlement = false; // 是否在结算状态
let settlementData = null; // 结算数据
let heartbeatInterval = null; // 心跳定时器
let isSpectating = false; // 是否在观战状态

// DOM元素
const loginScreen = document.getElementById('loginScreen');
const lobbyScreen = document.getElementById('lobbyScreen');
const gameScreen = document.getElementById('gameScreen');
const gameEndScreen = document.getElementById('gameEndScreen');

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    setupEventListeners();
    // 检查URL参数（分享链接）
    checkUrlParams();
    // 连接WebSocket
    connectWebSocket();
    
    // 下一局按钮事件
    const nextHandBtn = document.getElementById('nextHandBtn');
    if (nextHandBtn) {
        nextHandBtn.addEventListener('click', () => {
            console.log('点击下一局按钮');
            // 重置结算状态
            isSettlement = false;
            settlementData = null;
            
            // 隐藏结算信息面板
            const settlementPanel = document.getElementById('settlementInfo');
            if (settlementPanel) {
                settlementPanel.classList.add('hidden');
            }
            
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'startGame',
                    data: {}
                }));
            }
        });
    }
});

function setupEventListeners() {
    // 登录界面
    document.getElementById('joinBtn').addEventListener('click', joinGame);
    document.getElementById('playerName').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') joinGame();
    });
    document.getElementById('roomId').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') joinGame();
    });

    // 大厅界面
    document.getElementById('startGameBtn').addEventListener('click', startGame);
    document.getElementById('leaveRoomBtn').addEventListener('click', leaveRoom);
    
    // 游戏界面中的开始按钮
    document.getElementById('startGameBtnInGame').addEventListener('click', startGame);
    
    // 上桌按钮
    const joinTableBtn = document.getElementById('joinTableBtn');
    if (joinTableBtn) {
        joinTableBtn.addEventListener('click', joinTable);
    }
    
    // 观战面板的买一手按钮
    const buyHandBtnSpectating = document.getElementById('buyHandBtnSpectating');
    if (buyHandBtnSpectating) {
        buyHandBtnSpectating.addEventListener('click', () => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'buyHand',
                    data: {}
                }));
            }
        });
    }
    
    // 分享房间按钮
    const shareRoomBtn = document.getElementById('shareRoomBtn');
    if (shareRoomBtn) {
        shareRoomBtn.addEventListener('click', shareRoom);
    }

    // 游戏界面
    document.getElementById('foldBtn').addEventListener('click', () => sendAction('fold'));
    document.getElementById('checkBtn').addEventListener('click', () => sendAction('check'));
    document.getElementById('callBtn').addEventListener('click', () => sendAction('call'));
    
    // 买一手按钮（操作面板中的）
    const buyHandBtn = document.getElementById('buyHandBtn');
    if (buyHandBtn) {
        buyHandBtn.addEventListener('click', () => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'buyHand',
                    data: {}
                }));
            }
        });
    }
    
    // 买一手按钮（等待面板中的）
    const buyHandBtnWaiting = document.getElementById('buyHandBtnWaiting');
    if (buyHandBtnWaiting) {
        buyHandBtnWaiting.addEventListener('click', () => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'buyHand',
                    data: {}
                }));
            }
        });
    }
    
    // 买一手统计按钮
    const buyHandStatsBtn = document.getElementById('buyHandStatsBtn');
    if (buyHandStatsBtn) {
        buyHandStatsBtn.addEventListener('click', () => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'getBuyHandStats',
                    data: {}
                }));
            }
        });
    }
    
    // 关闭买一手统计模态框
    const closeBuyHandStatsBtn = document.getElementById('closeBuyHandStatsBtn');
    if (closeBuyHandStatsBtn) {
        closeBuyHandStatsBtn.addEventListener('click', () => {
            const modal = document.getElementById('buyHandStatsModal');
            if (modal) {
                modal.classList.add('hidden');
            }
        });
    }
    
    // 点击模态框外部关闭
    const buyHandStatsModal = document.getElementById('buyHandStatsModal');
    if (buyHandStatsModal) {
        buyHandStatsModal.addEventListener('click', (e) => {
            if (e.target === buyHandStatsModal) {
                buyHandStatsModal.classList.add('hidden');
            }
        });
    }
    document.getElementById('raiseBtn').addEventListener('click', () => {
        const amount = parseInt(document.getElementById('raiseAmount').value);
        if (amount > 0) {
            sendAction('raise', amount);
        }
    });
    
    // 加注快捷按钮（固定金额）
    const raise20Btn = document.getElementById('raise20Btn');
    if (raise20Btn) {
        raise20Btn.addEventListener('click', () => {
            sendAction('raise', 20);
        });
    }
    
    const raise50Btn = document.getElementById('raise50Btn');
    if (raise50Btn) {
        raise50Btn.addEventListener('click', () => {
            sendAction('raise', 50);
        });
    }
    
    const raise100Btn = document.getElementById('raise100Btn');
    if (raise100Btn) {
        raise100Btn.addEventListener('click', () => {
            sendAction('raise', 100);
        });
    }
    
    // 加注快捷按钮（直接加注）
    const halfPotBtn = document.getElementById('halfPotBtn');
    if (halfPotBtn) {
        halfPotBtn.addEventListener('click', () => {
            const potEl = document.getElementById('potAmount');
            const currentBetEl = document.getElementById('currentBet');
            const playerBetEl = document.getElementById('playerBet');
            
            if (!potEl || !currentBetEl || !playerBetEl) return;
            
            const pot = parseInt(potEl.textContent) || 0;
            const currentBet = parseInt(currentBetEl.textContent) || 0;
            const playerBet = parseInt(playerBetEl.textContent) || 0;
            // 半池 = 底池的一半，向上取整到5的倍数
            // 例如：底池15，半池=7.5，向上取整到5的倍数=10
            const halfPotRaw = pot / 2;
            const halfPot = Math.ceil(halfPotRaw / 5) * 5;
            // raiseAmount 就是半池的金额（服务端会在当前下注基础上加这个金额）
            const raiseAmount = halfPot;
            
            // 验证最小加注金额（至少5）
            if (raiseAmount >= 5) {
                sendAction('raise', raiseAmount);
            } else if (currentBet > playerBet) {
                // 如果半池不足，至少跟注
                sendAction('call');
            }
        });
    }
    
    const fullPotBtn = document.getElementById('fullPotBtn');
    if (fullPotBtn) {
        fullPotBtn.addEventListener('click', () => {
            const potEl = document.getElementById('potAmount');
            const currentBetEl = document.getElementById('currentBet');
            const playerBetEl = document.getElementById('playerBet');
            
            if (!potEl || !currentBetEl || !playerBetEl) return;
            
            const pot = parseInt(potEl.textContent) || 0;
            const currentBet = parseInt(currentBetEl.textContent) || 0;
            const playerBet = parseInt(playerBetEl.textContent) || 0;
            // 满池 = 底池（就是底池本身）
            // 服务端计算：如果raiseAmount == pot，则 newTotalBet = currentPlayerBet + raiseAmount
            // 否则：newTotalBet = CurrentBet + raiseAmount
            const fullPot = pot;
            // raiseAmount 就是满池的金额（服务端会在当前下注基础上加这个金额）
            const raiseAmount = fullPot;
            
            // 验证最小加注金额（至少5）
            if (raiseAmount >= 5) {
                sendAction('raise', raiseAmount);
            } else if (currentBet > playerBet) {
                // 如果满池不足，至少跟注
                sendAction('call');
            }
        });
    }
    
    const allInBtn = document.getElementById('allInBtn');
    if (allInBtn) {
        allInBtn.addEventListener('click', () => {
            const playerChipsEl = document.getElementById('playerChips');
            const currentBetEl = document.getElementById('currentBet');
            const playerBetEl = document.getElementById('playerBet');
            
            if (!playerChipsEl || !currentBetEl || !playerBetEl) return;
            
            const playerChips = parseInt(playerChipsEl.textContent) || 0;
            const currentBet = parseInt(currentBetEl.textContent) || 0;
            const playerBet = parseInt(playerBetEl.textContent) || 0;
            const callAmount = Math.max(0, currentBet - playerBet);
            const raiseAmount = callAmount + playerChips;
            
            if (raiseAmount > 0) {
                sendAction('raise', raiseAmount);
            }
        });
    }

    // 游戏结束界面
    document.getElementById('newHandBtn').addEventListener('click', startGame);
}

function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    console.log('正在连接WebSocket:', wsUrl);
    
    try {
        ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('✅ WebSocket连接已建立');
            // 清除之前的错误提示
            const errorDiv = document.getElementById('loginError');
            if (errorDiv) {
                errorDiv.textContent = '';
                errorDiv.style.display = 'none';
            }
            // 启动心跳
            startHeartbeat();
        };

        ws.onmessage = (event) => {
            console.log('📨 收到WebSocket消息:', event.data);
            try {
                const message = JSON.parse(event.data);
                console.log('✅ 解析后的消息:', message);
                if (message.type === 'gameStarted') {
                    console.log('🎮🎮🎮 收到游戏开始消息！', message.data);
                }
                if (message.type === 'error') {
                    console.error('❌ 收到错误消息:', message.data);
                }
                handleMessage(message);
            } catch (error) {
                console.error('❌ 解析消息失败:', error, '原始数据:', event.data);
                showError('收到无效消息，请刷新页面');
            }
        };

        ws.onerror = (error) => {
            console.error('❌ WebSocket错误:', error);
            console.error('WebSocket状态:', ws ? ws.readyState : 'null');
            showError('WebSocket连接错误，请检查服务器是否运行');
        };

        ws.onclose = (event) => {
            console.log('WebSocket连接已关闭:', event.code, event.reason);
            stopHeartbeat();
            if (event.code !== 1000) {
                showError('连接已断开，请刷新页面重试');
            }
        };
    } catch (error) {
        console.error('创建WebSocket失败:', error);
        showError('无法创建WebSocket连接，请检查浏览器支持');
    }
}

function joinGame() {
    console.log('=== 点击加入游戏按钮 ===');
    const playerName = document.getElementById('playerName').value.trim();
    const roomId = document.getElementById('roomId').value.trim();

    console.log('输入信息:', { playerName, roomId });

    if (!playerName) {
        showError('请输入你的名字');
        return;
    }

    // 检查WebSocket状态
    const wsState = ws ? ws.readyState : null;
    console.log('WebSocket状态:', wsState);
    console.log('WebSocket状态说明:', 
        wsState === WebSocket.CONNECTING ? '连接中' :
        wsState === WebSocket.OPEN ? '已连接' :
        wsState === WebSocket.CLOSING ? '关闭中' :
        wsState === WebSocket.CLOSED ? '已关闭' : '未初始化');

    // 确保WebSocket已连接
    if (!ws || wsState === WebSocket.CLOSED || wsState === WebSocket.CLOSING) {
        console.log('WebSocket未连接或已关闭，正在重新连接...');
        connectWebSocket();
        // 等待连接建立
        let attempts = 0;
        const maxAttempts = 50; // 5秒
        const checkConnection = setInterval(() => {
            attempts++;
            const currentState = ws ? ws.readyState : null;
            console.log(`检查连接状态 (${attempts}/${maxAttempts}):`, currentState);
            
            if (ws && ws.readyState === WebSocket.OPEN) {
                clearInterval(checkConnection);
                console.log('✅ WebSocket连接已建立，发送加入游戏消息');
                doJoinGame(playerName, roomId);
            } else if (ws && (ws.readyState === WebSocket.CLOSED || ws.readyState === WebSocket.CLOSING)) {
                clearInterval(checkConnection);
                console.error('❌ 连接失败');
                showError('连接失败，请刷新页面重试');
            } else if (attempts >= maxAttempts) {
                clearInterval(checkConnection);
                console.error('❌ 连接超时');
                showError('连接超时，请检查服务器是否运行');
            }
        }, 100);
    } else if (wsState === WebSocket.CONNECTING) {
        console.log('WebSocket连接中，等待连接建立...');
        let attempts = 0;
        const maxAttempts = 50;
        const waitForConnection = setInterval(() => {
            attempts++;
            if (ws.readyState === WebSocket.OPEN) {
                clearInterval(waitForConnection);
                console.log('✅ 连接已建立');
                doJoinGame(playerName, roomId);
            } else if (ws.readyState === WebSocket.CLOSED) {
                clearInterval(waitForConnection);
                console.error('❌ 连接失败');
                showError('连接失败，请刷新页面重试');
            } else if (attempts >= maxAttempts) {
                clearInterval(waitForConnection);
                console.error('❌ 连接超时');
                showError('连接超时，请刷新页面重试');
            }
        }, 100);
    } else if (wsState === WebSocket.OPEN) {
        console.log('✅ WebSocket已连接，直接发送加入游戏消息');
        doJoinGame(playerName, roomId);
    } else {
        console.error('❌ WebSocket状态异常:', wsState);
        showError('连接状态异常，请刷新页面');
    }
}

function doJoinGame(playerName, roomId) {
    console.log('执行加入游戏:', { playerName, roomId });
    if (roomId) {
        // 加入现有房间
        console.log('加入现有房间:', roomId);
        sendMessage({
            type: 'joinRoom',
            data: {
                roomId: roomId,
                playerName: playerName
            }
        });
    } else {
        // 创建新房间
        console.log('创建新房间');
        sendMessage({
            type: 'createRoom',
            data: {
                playerName: playerName
            }
        });
    }
}

function startGame() {
    console.log('点击开始游戏按钮');
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        showError('WebSocket未连接，请刷新页面');
        return;
    }
    sendMessage({
        type: 'startGame',
        data: {}
    });
    console.log('已发送开始游戏消息');
}

function leaveRoom() {
    if (ws) {
        ws.close();
    }
    showScreen('loginScreen');
    currentRoom = null;
    currentPlayer = null;
}

function sendAction(action, amount = 0) {
    // 检查WebSocket连接
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        console.error('WebSocket未连接，无法发送行动');
        showError('连接已断开，请刷新页面');
        return;
    }
    
    // 验证行动类型
    if (!['fold', 'check', 'call', 'raise'].includes(action)) {
        console.error('无效的行动类型:', action);
        return;
    }
    
    // 验证加注金额
    if (action === 'raise') {
        amount = parseInt(amount) || 0;
        if (amount < 5) {
            showError('最小加注金额为5');
            return;
        }
    }
    
    // 停止倒计时
    stopTurnTimer();
    
    sendMessage({
        type: 'action',
        data: {
            action: action,
            amount: amount
        }
    });
}

function sendMessage(message) {
    console.log('发送消息:', message);
    if (ws && ws.readyState === WebSocket.OPEN) {
        try {
            ws.send(JSON.stringify(message));
            console.log('消息已发送');
        } catch (error) {
            console.error('发送消息失败:', error);
            showError('发送消息失败，请刷新页面');
        }
    } else {
        console.error('WebSocket未连接，状态:', ws ? ws.readyState : 'null');
        showError('连接已断开，请刷新页面');
    }
}

function handleMessage(message) {
    console.log('收到消息:', message);

    console.log('处理消息，类型:', message.type);
    switch (message.type) {
        case 'roomCreated':
            console.log('✅ 收到房间创建消息:', message.data);
            currentRoom = message.data.roomId;
            console.log('设置房间ID:', currentRoom);
            // 更新房间ID显示
            updateRoomIdDisplay(currentRoom);
            // 检查是否在观战状态
            if (message.data.isSpectating) {
                isSpectating = true;
                if (message.data.room) {
                    updateGameState(message.data.room);
                    showSpectatingPanel(message.data.room);
                }
                showScreen('gameScreen');
            } else {
                // 直接进入游戏界面，不显示大厅
                if (message.data.room) {
                    updateGameState(message.data.room);
                    // 找到当前玩家
                    if (message.data.room.players && message.data.room.players.length > 0) {
                        const playerName = document.getElementById('playerName').value.trim();
                        currentPlayer = message.data.room.players.find(p => p.name === playerName) || 
                                       message.data.room.players[message.data.room.players.length - 1];
                    }
                    showScreen('gameScreen');
                } else {
                    // 如果没有房间数据，等待roomJoined消息
                    showScreen('gameScreen');
                }
            }
            break;

        case 'roomJoined':
            currentRoom = message.data.room.id;
            // 更新房间ID显示
            updateRoomIdDisplay(currentRoom);
            
            // 检查是否在观战状态
            if (message.data.isSpectating) {
                console.log('进入观战状态');
                isSpectating = true;
                if (message.data.room) {
                    updateGameState(message.data.room);
                    showScreen('gameScreen');
                    showSpectatingPanel(message.data.room);
                }
            }
            // 检查是否在等待状态
            else if (message.data.isWaiting) {
                console.log('游戏正在进行中，需要等待下一局');
                isSpectating = false;
                // 显示游戏界面，但玩家处于等待状态
                if (message.data.room) {
                    updateGameState(message.data.room);
                    showScreen('gameScreen');
                    // 显示等待提示
                    const waitingPanel = document.getElementById('waitingPanel');
                    const actionPanel = document.getElementById('actionPanel');
                    const spectatingPanel = document.getElementById('spectatingPanel');
                    if (waitingPanel) {
                        waitingPanel.innerHTML = '<p style="font-size: 1.2em; color: #ffd700;">⏳ 游戏正在进行中，请等待下一局开始</p>';
                        waitingPanel.classList.remove('hidden');
                    }
                    if (actionPanel) {
                        actionPanel.classList.add('hidden');
                    }
                    if (spectatingPanel) {
                        spectatingPanel.classList.add('hidden');
                    }
                    // 清空手牌显示
                    const handCard0 = document.getElementById('handCard0');
                    const handCard1 = document.getElementById('handCard1');
                    if (handCard0) handCard0.innerHTML = '';
                    if (handCard1) handCard1.innerHTML = '';
                }
            } else {
                // 找到当前玩家
                isSpectating = false;
                if (message.data.room.players && message.data.room.players.length > 0) {
                    const playerName = document.getElementById('playerName').value.trim();
                    currentPlayer = message.data.room.players.find(p => p.name === playerName) || 
                                   message.data.room.players[message.data.room.players.length - 1];
                }
                // 直接进入游戏界面
                updateGameState(message.data.room);
                showScreen('gameScreen');
                hideSpectatingPanel();
            }
            break;
            
        case 'playerJoinedTable':
            console.log('玩家上桌成功', message.data);
            // 检查是否是自己上桌
            const playerName = document.getElementById('playerName')?.value.trim();
            const joinedPlayer = message.data.player;
            
            if (joinedPlayer && joinedPlayer.name === playerName) {
                // 是自己上桌，隐藏观战面板
                console.log('自己上桌成功');
                isSpectating = false;
                if (message.data.room) {
                    updateGameState(message.data.room);
                    hideSpectatingPanel();
                }
            } else {
                // 是其他玩家上桌，只更新游戏状态，保持自己的观战状态
                console.log('其他玩家上桌，保持观战状态');
                if (message.data.room) {
                    updateGameState(message.data.room);
                    // 如果自己在观战，保持观战面板显示
                    if (isSpectating) {
                        showSpectatingPanel(message.data.room);
                    }
                }
            }
            break;
            
        case 'playerMovedToSpectating':
            console.log('玩家被移入观战状态');
            isSpectating = true;
            if (message.data.room) {
                updateGameState(message.data.room);
                showSpectatingPanel(message.data.room);
            }
            break;
            
        case 'roomUpdated':
            console.log('收到房间更新消息:', message.data);
            if (message.data.room) {
                // 检查玩家是否在等待列表中
                const room = message.data.room;
                const playerName = document.getElementById('playerName')?.value.trim();
                
                // 检查自己的状态
                let isInWaitingList = false;
                let isInSpectators = false;
                let isInPlayers = false;
                
                if (room.waitingPlayers && Array.isArray(room.waitingPlayers)) {
                    isInWaitingList = room.waitingPlayers.some(p => p && p.name === playerName);
                }
                
                if (room.spectators && Array.isArray(room.spectators)) {
                    isInSpectators = room.spectators.some(p => p && p.name === playerName);
                }
                
                if (room.players && Array.isArray(room.players)) {
                    isInPlayers = room.players.some(p => p && p.name === playerName);
                }
                
                // 更新自己的观战状态
                if (isInSpectators) {
                    isSpectating = true;
                } else if (isInPlayers) {
                    isSpectating = false;
                }
                
                // 如果玩家在等待列表中，显示等待提示
                if (isInWaitingList) {
                    const waitingPanel = document.getElementById('waitingPanel');
                    const actionPanel = document.getElementById('actionPanel');
                    const spectatingPanel = document.getElementById('spectatingPanel');
                    if (waitingPanel) {
                        waitingPanel.innerHTML = '<p style="font-size: 1.2em; color: #ffd700;">⏳ 游戏正在进行中，请等待下一局开始</p>';
                        waitingPanel.classList.remove('hidden');
                    }
                    if (actionPanel) {
                        actionPanel.classList.add('hidden');
                    }
                    if (spectatingPanel) {
                        spectatingPanel.classList.add('hidden');
                    }
                    // 清空手牌显示
                    const handCard0 = document.getElementById('handCard0');
                    const handCard1 = document.getElementById('handCard1');
                    if (handCard0) handCard0.innerHTML = '';
                    if (handCard1) handCard1.innerHTML = '';
                } else if (isInSpectators) {
                    // 在观战列表中，显示观战面板
                    showSpectatingPanel(room);
                } else if (isInPlayers) {
                    // 在游戏列表中，隐藏观战面板
                    hideSpectatingPanel();
                }
                
                // 先更新当前玩家信息，确保updatePlayerInfo使用正确的玩家信息
                if (message.data.room.players && message.data.room.players.length > 0) {
                    const player = message.data.room.players.find(p => p.name === playerName);
                    if (player) {
                        currentPlayer = player;
                        console.log('更新当前玩家:', currentPlayer);
                    }
                }
                
                // 然后更新游戏状态（会调用updatePlayerInfo，使用正确的player参数）
                updateGameState(message.data.room);
            }
            break;

        case 'playerJoined':
            // 如果游戏正在进行中，不更新游戏状态，避免影响当前游戏
            // 新玩家应该已经在等待列表中，不会影响当前游戏
            if (message.data.room) {
                // 只更新房间信息，不更新游戏状态（避免影响当前游戏）
                const room = message.data.room;
                // 如果游戏正在进行中，不更新游戏状态
                if (room.gamePhase && room.gamePhase !== 'waiting') {
                    console.log('游戏正在进行中，新玩家加入但不影响当前游戏');
                    // 不更新游戏状态，保持当前游戏状态
                } else {
                    // 游戏在等待状态，可以更新
                    updateGameState(message.data.room);
                    // 设置当前玩家
                    if (message.data.room.players) {
                        const playerName = document.getElementById('playerName').value.trim();
                        const player = message.data.room.players.find(p => p.name === playerName);
                        if (player) {
                            currentPlayer = player;
                        }
                    }
                }
            }
            break;

        case 'gameStarted':
            console.log('🎮 收到游戏开始消息:', message.data);
            gameState = message.data;
            
            // 重置结算状态
            isSettlement = false;
            settlementData = null;
            
            // 隐藏结算信息面板
            const settlementPanel = document.getElementById('settlementInfo');
            if (settlementPanel) {
                settlementPanel.classList.add('hidden');
            }
            
            // 新游戏开始时，清空公共牌和玩家手牌显示
            console.log('新游戏开始，清空公共牌和玩家手牌显示');
            updateCommunityCards([]);
            
            // 确保当前玩家信息已设置
            if (gameState.players) {
                if (!currentPlayer) {
                    // 通过名字找到当前玩家
                    const playerName = document.getElementById('playerName')?.value.trim();
                    if (playerName) {
                        currentPlayer = gameState.players.find(p => p.name === playerName);
                        console.log('通过名字找到当前玩家:', currentPlayer);
                    }
                }
                
                if (currentPlayer) {
                    // 更新当前玩家信息（从服务器获取最新数据）
                    const player = gameState.players.find(p => p.id === currentPlayer.id);
                    if (player) {
                        currentPlayer = player;
                        console.log('当前玩家手牌:', player.hand);
                    }
                }
            }
            
            updateGameState(gameState);
            showScreen('gameScreen');
            break;

        case 'gameWaiting':
            console.log('⏳ 收到等待消息:', message.data);
            // 玩家在等待列表中，不参与当前游戏
            if (message.data.room) {
                updateGameState(message.data.room);
                showScreen('gameScreen');
                // 显示等待提示
                const waitingPanel = document.getElementById('waitingPanel');
                const actionPanel = document.getElementById('actionPanel');
                if (waitingPanel) {
                    waitingPanel.innerHTML = '<p style="font-size: 1.2em; color: #ffd700;">⏳ 游戏正在进行中，请等待下一局开始</p>';
                    waitingPanel.classList.remove('hidden');
                }
                if (actionPanel) {
                    actionPanel.classList.add('hidden');
                }
                // 清空手牌显示
                const handCard0 = document.getElementById('handCard0');
                const handCard1 = document.getElementById('handCard1');
                if (handCard0) handCard0.innerHTML = '';
                if (handCard1) handCard1.innerHTML = '';
            }
            break;

        case 'actionTaken':
            gameState = message.data;
            // 先更新当前玩家信息，确保updatePlayerInfo使用正确的玩家信息
            if (gameState && gameState.players && gameState.players.length > 0) {
                const playerName = document.getElementById('playerName')?.value.trim();
                if (playerName) {
                    const player = gameState.players.find(p => p.name === playerName);
                    if (player) {
                        currentPlayer = player;
                        console.log('actionTaken: 更新当前玩家:', currentPlayer);
                    }
                }
            }
            updateGameState(gameState);
            break;

        case 'gameEnded':
            showSettlement(message.data);
            break;

        case 'buyHandSuccess':
            console.log('✅ 买一手成功，新筹码:', message.data.chips);
            if (message.data && message.data.chips !== undefined) {
                const playerChipsEl = document.getElementById('playerChips');
                const playerChipsWaitingEl = document.getElementById('playerChipsWaiting');
                if (playerChipsEl) {
                    playerChipsEl.textContent = message.data.chips;
                }
                if (playerChipsWaitingEl) {
                    playerChipsWaitingEl.textContent = message.data.chips;
                }
            }
            break;
            
        case 'buyHandStats':
            console.log('收到买一手统计:', message.data);
            showBuyHandStats(message.data.stats);
            break;
            
        case 'error':
            const errorMsg = message.data.message || message.data || '发生错误';
            console.error('收到错误消息:', errorMsg);
            showError(errorMsg);
            break;

        case 'playerLeft':
            if (message.data.room) {
                updateLobby({ room: message.data.room });
            }
            break;
    }
}

function updateLobby(data) {
    const room = data.room;
    if (!room) return;

    currentRoom = room.id;
    document.getElementById('displayRoomId').textContent = room.id;
    document.getElementById('playerCount').textContent = room.players.length;

    const playersList = document.getElementById('playersList');
    playersList.innerHTML = '';

    room.players.forEach(player => {
        const playerItem = document.createElement('div');
        playerItem.className = 'player-item';
        playerItem.innerHTML = `
            <span>${player.name || '玩家' + player.id.substring(0, 4)}</span>
            <span>筹码: ${player.chips}</span>
        `;
        playersList.appendChild(playerItem);
    });
}

function updateRoomIdDisplay(roomId) {
    const roomIdElement = document.getElementById('gameRoomId');
    if (roomIdElement && roomId) {
        roomIdElement.textContent = roomId;
    }
}

function updateGameState(room) {
    if (!room) {
        console.error('updateGameState: room为空');
        return;
    }

    console.log('更新游戏状态:', room);
    
    // 更新房间ID显示
    if (room.id) {
        updateRoomIdDisplay(room.id);
    }

    // 更新底池和当前下注
    document.getElementById('potAmount').textContent = room.pot || 0;
    document.getElementById('currentBet').textContent = room.currentBet || 0;

    // 更新游戏阶段
    const phaseNames = {
        'preflop': '翻牌前',
        'flop': '翻牌',
        'turn': '转牌',
        'river': '河牌',
        'showdown': '比牌',
        'waiting': '等待开始'
    };
    document.getElementById('gamePhase').textContent = phaseNames[room.gamePhase] || room.gamePhase;
    
    // 显示/隐藏开始游戏按钮（游戏界面中的）
    const startGamePanel = document.getElementById('startGamePanel');
    const startGameBtnInGame = document.getElementById('startGameBtnInGame');
    if (startGamePanel && startGameBtnInGame) {
        if (room.gamePhase === 'waiting' && room.players && room.players.length >= 4) {
            startGamePanel.classList.remove('hidden');
        } else {
            startGamePanel.classList.add('hidden');
        }
    }

    // 更新公共牌
    // 如果是新游戏开始（没有公共牌），清空显示
    // 如果是结算状态，保持显示上一局的牌
    if (!isSettlement || (room.communityCards && room.communityCards.length > 0)) {
        updateCommunityCards(room.communityCards || []);
    }

    // 更新玩家区域
    // 如果是新游戏开始且不是结算状态，会清空玩家手牌显示
    updatePlayersArea(room.players || [], room.currentTurn, room.dealerIndex);

    // 如果玩家在观战状态，更新观战面板的筹码显示
    if (isSpectating && room.spectators) {
        const playerName = document.getElementById('playerName')?.value.trim();
        const spectator = room.spectators.find(p => p && p.name === playerName);
        if (spectator) {
            const chipsEl = document.getElementById('playerChipsSpectating');
            if (chipsEl) {
                chipsEl.textContent = spectator.chips || 500;
            }
        }
    }

    // 更新当前玩家信息
    if (room.players) {
        // 如果没有currentPlayer，尝试通过名字找到
        if (!currentPlayer) {
            const playerName = document.getElementById('playerName')?.value.trim();
            if (playerName) {
                currentPlayer = room.players.find(p => p.name === playerName);
                console.log('通过名字找到当前玩家:', currentPlayer);
            }
        }
        
        // 如果还是找不到，使用第一个玩家（临时方案）
        if (!currentPlayer && room.players.length > 0) {
            currentPlayer = room.players[0];
            console.log('使用第一个玩家作为当前玩家:', currentPlayer);
        }
        
        if (currentPlayer) {
            const player = room.players.find(p => p.id === currentPlayer.id);
            if (player) {
                console.log('更新玩家信息，手牌:', player.hand);
                updatePlayerInfo(player, room);
            } else {
                console.warn('未找到当前玩家信息，ID:', currentPlayer.id, '所有玩家:', room.players.map(p => ({id: p.id, name: p.name})));
            }
        } else {
            console.warn('无法确定当前玩家，所有玩家:', room.players.map(p => ({id: p.id, name: p.name})));
        }
    }
}

function updateCommunityCards(cards) {
    console.log('updateCommunityCards: 更新公共牌，数量:', cards ? cards.length : 0, cards);
    if (!cards) {
        cards = [];
    }
    
    let newCardIndex = 0; // 新牌计数器，用于错开动画时间
    
    for (let i = 0; i < 5; i++) {
        const cardSlot = document.getElementById(`card${i}`);
        if (!cardSlot) {
            console.warn('找不到card slot:', `card${i}`);
            continue;
        }
        
        const hadCard = cardSlot.innerHTML !== '';
        
        if (i < cards.length && cards[i]) {
            const cardHTML = createCardHTML(cards[i]);
            if (!hadCard) {
                // 新牌发牌动画
                cardSlot.classList.remove('slot-waiting');
                cardSlot.innerHTML = cardHTML;
                const cardEl = cardSlot.querySelector('.card');
                if (cardEl) {
                    cardEl.style.animationDelay = (newCardIndex * 0.18) + 's';
                    cardEl.classList.add('deal-community');
                    // 发牌落地后添加金光
                    const glowDelay = newCardIndex * 180 + 550;
                    setTimeout(() => {
                        cardEl.classList.add('card-land-glow');
                    }, glowDelay);
                    // 动画结束后清理
                    cardEl.addEventListener('animationend', function handler(e) {
                        if (e.animationName === 'landGlow') {
                            cardEl.classList.remove('deal-community', 'card-land-glow');
                            cardEl.style.animationDelay = '';
                            cardEl.removeEventListener('animationend', handler);
                        }
                    });
                }
                newCardIndex++;
            } else {
                // 直接更新（结算时）
                cardSlot.innerHTML = cardHTML;
            }
        } else {
            // 没有牌，清空显示
            cardSlot.innerHTML = '';
            // 下一张将要发的牌槽 - 添加等待呼吸灯
            if (i === cards.length && cards.length > 0 && cards.length < 5) {
                cardSlot.classList.add('slot-waiting');
            } else {
                cardSlot.classList.remove('slot-waiting');
            }
        }
    }
}

function updatePlayersArea(players, currentTurn, dealerIndex) {
    const playersArea = document.getElementById('playersArea');
    if (!playersArea) return;
    
    // 检查玩家数组是否有效
    if (!players || players.length === 0) {
        // 如果玩家列表为空且不是结算状态，清空显示
        if (!isSettlement) {
            playersArea.innerHTML = '';
        }
        return;
    }

    // 如果不是结算状态且是新游戏开始（所有玩家都没有手牌），清空之前的显示
    // 结算状态时保持显示上一局的牌
    if (!isSettlement) {
        // 检查是否所有玩家都没有手牌（新游戏开始）
        const allPlayersHaveNoHands = players.every(p => !p.hand || p.hand.length === 0);
        if (allPlayersHaveNoHands) {
            // 新游戏开始，清空玩家区域
            playersArea.innerHTML = '';
        } else {
            // 游戏进行中，清空后重新渲染
            playersArea.innerHTML = '';
        }
    } else {
        // 结算状态，清空后重新渲染（保持显示上一局的牌）
        playersArea.innerHTML = '';
    }

    // 计算圆角矩形牌桌位置（12个位置，玩家均匀分布）
    const positions = calculateRectangularTablePositions(players.length);

    players.forEach((player, index) => {
        if (!player) return;
        const seat = document.createElement('div');
        seat.className = 'player-seat';
        
        // 设置位置
        if (positions[index]) {
            seat.style.top = positions[index].top + '%';
            seat.style.left = positions[index].left + '%';
            seat.style.transform = positions[index].transform || 'translate(-50%, -50%)';
        }
        
        if (index === currentTurn && !isSettlement) {
            seat.classList.add('active');
        }
        if (index === dealerIndex) {
            seat.classList.add('dealer');
        }
        if (player.folded) {
            seat.classList.add('folded');
        }
        
        // 结算时标记获胜者
        if (isSettlement && settlementData && settlementData.winner && 
            player.id && settlementData.winner.id && 
            player.id === settlementData.winner.id) {
            seat.classList.add('winner');
        }

        let status = '';
        if (player.isDealer) status = '庄家';
        if (player.isSmall) status = '小盲';
        if (player.isBig) status = '大盲';
        if (player.allIn) status = '全押';

        // 显示底牌
        // 结算时：只显示未弃牌玩家的真实底牌（从settlementData.allHands获取）
        // 游戏进行中：只显示自己的牌，其他玩家显示背面或空
        let cardsHTML = '';
        const showCards = isSettlement || (currentPlayer && player.id === currentPlayer.id);
        
        // 确保player.hand是数组
        // 结算时优先使用settlementData中的手牌数据
        let playerHand = Array.isArray(player.hand) ? player.hand : [];
        let isFoldedInSettlement = player.folded;
        if (isSettlement && settlementData && settlementData.allHands) {
            const handData = settlementData.allHands.find(h => h && h.id === player.id);
            if (handData) {
                if (Array.isArray(handData.hand)) {
                    playerHand = handData.hand;
                }
                // 结算时使用settlementData中的folded状态
                isFoldedInSettlement = handData.folded || false;
            }
        }
        
        if (isSettlement) {
            // 结算时：只显示未弃牌玩家的手牌
            if (!isFoldedInSettlement && playerHand.length === 2) {
                playerHand.forEach(card => {
                    if (card && card.suit && card.rank) {
                        cardsHTML += createCardHTML(card);
                    }
                });
            }
            // 已弃牌的玩家不显示手牌
        } else if (showCards && playerHand.length === 2) {
            // 游戏进行中且是自己的牌：显示真实牌面
            playerHand.forEach(card => {
                if (card && card.suit && card.rank) {
                    cardsHTML += createCardHTML(card);
                }
            });
        } else if (playerHand.length === 2 && !player.folded) {
            // 游戏进行中且不是自己的牌：显示背面（带滑入动画）
            cardsHTML = `
                <div class="card card-back deal-back"></div>
                <div class="card card-back deal-back" style="animation-delay:0.1s"></div>
            `;
        }

        seat.innerHTML = `
            <div class="player-seat-name">${player.name || '玩家' + player.id.substring(0, 4)}</div>
            <div class="player-seat-chips">筹码: ${player.chips}</div>
            <div class="player-seat-bet">下注: ${player.bet}</div>
            <div class="player-seat-status">${status}</div>
            <div class="player-seat-cards">${cardsHTML}</div>
        `;

        playersArea.appendChild(seat);
        
        // 结算时：给翻开的牌添加动画
        if (isSettlement && !isFoldedInSettlement && playerHand.length === 2) {
            const cardEls = seat.querySelectorAll('.player-seat-cards .card');
            cardEls.forEach((cardEl, ci) => {
                // 翻牌动画 + 错开延迟
                cardEl.classList.add('reveal-flip');
                cardEl.style.animationDelay = (ci * 0.15) + 's';
                // 赢家的牌额外加金色闪烁
                if (settlementData && settlementData.winner && 
                    player.id === settlementData.winner.id) {
                    setTimeout(() => {
                        cardEl.classList.add('winner-highlight');
                    }, 700 + ci * 150);
                }
            });
        }
    });
}

// 计算圆角矩形牌桌位置（12个位置，玩家均匀分布在牌桌四周）
function calculateRectangularTablePositions(playerCount) {
    const positions = [];
    const MAX_SEATS = 12;
    const totalSeats = Math.min(playerCount, MAX_SEATS);
    
    if (totalSeats === 0) return positions;
    
    // 圆角矩形的四个边：上、右、下、左
    // 每个边分配3个位置，总共12个位置
    // 均匀分布：每个边的位置数尽量相等
    const seatsPerSide = Math.ceil(totalSeats / 4);
    let remainingSeats = totalSeats;
    
    // 上边（从左上到右上）
    const topSeats = Math.min(seatsPerSide, remainingSeats);
    for (let i = 0; i < topSeats; i++) {
        const x = 15 + (i + 1) * (70 / (topSeats + 1));
        positions.push({
            top: 8,
            left: x,
            transform: 'translate(-50%, 0)'
        });
    }
    remainingSeats -= topSeats;
    
    // 右边（从右上到右下）
    const rightSeats = Math.min(seatsPerSide, remainingSeats);
    for (let i = 0; i < rightSeats; i++) {
        const y = 15 + (i + 1) * (70 / (rightSeats + 1));
        positions.push({
            top: y,
            left: 92,
            transform: 'translate(-50%, -50%)'
        });
    }
    remainingSeats -= rightSeats;
    
    // 下边（从右下到左下）
    const bottomSeats = Math.min(seatsPerSide, remainingSeats);
    for (let i = 0; i < bottomSeats; i++) {
        const x = 92 - (i + 1) * (70 / (bottomSeats + 1));
        positions.push({
            top: 92,
            left: x,
            transform: 'translate(-50%, -100%)'
        });
    }
    remainingSeats -= bottomSeats;
    
    // 左边（从左下到左上）
    const leftSeats = remainingSeats;
    for (let i = 0; i < leftSeats; i++) {
        const y = 92 - (i + 1) * (70 / (leftSeats + 1));
        positions.push({
            top: y,
            left: 8,
            transform: 'translate(0, -50%)'
        });
    }
    
    return positions;
}

function updatePlayerInfo(player, room) {
    if (!player || !room) {
        console.warn('updatePlayerInfo: player或room为空', { player, room });
        return;
    }
    
    console.log('🃏 更新玩家信息:', { 
        playerId: player.id, 
        playerName: player.name, 
        hand: player.hand,
        handLength: player.hand ? player.hand.length : 0
    });
    
    // 更新玩家信息（在操作面板和等待面板中都要显示）
    const playerChipsEl = document.getElementById('playerChips');
    const playerBetEl = document.getElementById('playerBet');
    const playerChipsWaitingEl = document.getElementById('playerChipsWaiting');
    const playerBetWaitingEl = document.getElementById('playerBetWaiting');
    
    if (playerChipsEl) playerChipsEl.textContent = player.chips;
    if (playerBetEl) playerBetEl.textContent = player.bet;
    if (playerChipsWaitingEl) playerChipsWaitingEl.textContent = player.chips;
    if (playerBetWaitingEl) playerBetWaitingEl.textContent = player.bet;

    // 更新手牌 - 添加发牌动画
    const handCard0 = document.getElementById('handCard0');
    const handCard1 = document.getElementById('handCard1');
    
    if (!handCard0 || !handCard1) {
        console.error('找不到手牌元素');
        return;
    }
    
    // 确保player.hand是数组
    const playerHand = Array.isArray(player.hand) ? player.hand : [];
    
    if (playerHand.length === 2 && playerHand[0] && playerHand[1] && 
        playerHand[0].suit && playerHand[0].rank && 
        playerHand[1].suit && playerHand[1].rank) {
        console.log('✅ 显示手牌:', playerHand);
        const hadCards = handCard0.innerHTML !== '' && handCard1.innerHTML !== '';
        
        if (!hadCards) {
            // 发牌动画 - 两张牌先后飞入
            handCard0.innerHTML = createCardHTML(playerHand[0]);
            handCard1.innerHTML = createCardHTML(playerHand[1]);
            
            const card0El = handCard0.querySelector('.card');
            const card1El = handCard1.querySelector('.card');
            
            if (card0El) {
                card0El.classList.add('deal-hand');
                // 落地金光
                setTimeout(() => card0El.classList.add('card-land-glow'), 700);
                card0El.addEventListener('animationend', function handler(e) {
                    if (e.animationName === 'landGlow') {
                        card0El.classList.remove('deal-hand', 'card-land-glow');
                        card0El.removeEventListener('animationend', handler);
                    }
                });
            }
            if (card1El) {
                card1El.style.animationDelay = '0.2s';
                card1El.classList.add('deal-hand');
                setTimeout(() => card1El.classList.add('card-land-glow'), 900);
                card1El.addEventListener('animationend', function handler(e) {
                    if (e.animationName === 'landGlow') {
                        card1El.classList.remove('deal-hand', 'card-land-glow');
                        card1El.style.animationDelay = '';
                        card1El.removeEventListener('animationend', handler);
                    }
                });
            }
        } else {
            // 直接更新
            handCard0.innerHTML = createCardHTML(playerHand[0]);
            handCard1.innerHTML = createCardHTML(playerHand[1]);
        }
    } else {
        console.log('⚠️ 没有手牌或手牌数量不对:', playerHand);
        handCard0.innerHTML = '';
        handCard1.innerHTML = '';
    }

    // 显示/隐藏操作面板
    const actionPanel = document.getElementById('actionPanel');
    const waitingPanel = document.getElementById('waitingPanel');
    const foldBtn = document.getElementById('foldBtn');
    const checkBtn = document.getElementById('checkBtn');
    const callBtn = document.getElementById('callBtn');
    const raiseBtn = document.getElementById('raiseBtn');
    const raiseAmount = document.getElementById('raiseAmount');
    const raiseGroup = raiseBtn ? raiseBtn.parentElement : null;
    
    // 判断是否是当前回合：使用传入的player参数，而不是currentPlayer
    // 因为currentPlayer可能没有及时更新
    const isMyTurn = room.currentTurn !== undefined && 
                     room.players && 
                     room.players[room.currentTurn] && 
                     room.players[room.currentTurn].id === player.id;
    
    if (isMyTurn && !player.folded && !player.allIn && room.gamePhase !== 'waiting') {
        actionPanel.classList.remove('hidden');
        waitingPanel.classList.add('hidden');
        
        // 启动倒计时
        startTurnTimer();
        
        // 计算需要跟注的金额
        const callAmount = room.currentBet - player.bet;
        
        // 重置所有按钮的显示状态
        if (foldBtn) foldBtn.style.display = 'inline-block';
        if (checkBtn) checkBtn.style.display = 'none';
        if (callBtn) callBtn.style.display = 'none';
        if (raiseGroup) raiseGroup.style.display = 'none';
        
        // 根据游戏状态显示可用的按钮
        if (callAmount > 0) {
            // 需要跟注，显示跟注和加注按钮
            if (callBtn) {
                callBtn.style.display = 'inline-block';
                callBtn.textContent = `跟注 (${callAmount})`;
                callBtn.disabled = player.chips < callAmount;
            }
            
            // 如果筹码足够，显示加注按钮
            if (raiseGroup && player.chips >= callAmount) {
                raiseGroup.style.display = 'flex';
                
                // 更新半池和满池按钮文本，显示真实的加注金额
                // 服务端计算逻辑：
                // 如果 raiseAmount == pot（满池），则 newTotalBet = currentPlayerBet + raiseAmount
                // 否则：newTotalBet = CurrentBet + raiseAmount
                // 所以显示的加注金额就是半池/满池本身，不是总下注
                const pot = room.pot || 0;
                // 半池 = 底池的一半，向上取整到5的倍数
                const halfPotRaw = pot / 2;
                const halfPot = Math.ceil(halfPotRaw / 5) * 5;
                // 满池 = 底池（就是底池本身）
                const fullPot = pot;
                
                const halfPotBtn = document.getElementById('halfPotBtn');
                const fullPotBtn = document.getElementById('fullPotBtn');
                if (halfPotBtn) {
                    halfPotBtn.textContent = `半池 (${halfPot})`;
                }
                if (fullPotBtn) {
                    fullPotBtn.textContent = `满池 (${fullPot})`;
                }
            }
        } else {
            // 可以过牌，显示过牌和加注按钮
            if (checkBtn) checkBtn.style.display = 'inline-block';
            if (raiseGroup) {
                raiseGroup.style.display = 'flex';
                
                // 更新半池和满池按钮文本，显示真实的加注金额
                // 服务端计算逻辑：
                // 如果 raiseAmount == pot（满池），则 newTotalBet = currentPlayerBet + raiseAmount
                // 否则：newTotalBet = CurrentBet + raiseAmount
                // 所以显示的加注金额就是半池/满池本身，不是总下注
                const pot = room.pot || 0;
                // 半池 = 底池的一半，向上取整到5的倍数
                const halfPotRaw = pot / 2;
                const halfPot = Math.ceil(halfPotRaw / 5) * 5;
                // 满池 = 底池（就是底池本身）
                const fullPot = pot;
                
                const halfPotBtn = document.getElementById('halfPotBtn');
                const fullPotBtn = document.getElementById('fullPotBtn');
                if (halfPotBtn) {
                    halfPotBtn.textContent = `半池 (${halfPot})`;
                }
                if (fullPotBtn) {
                    fullPotBtn.textContent = `满池 (${fullPot})`;
                }
            }
        }
    } else {
        actionPanel.classList.add('hidden');
        waitingPanel.classList.remove('hidden');
        // 停止倒计时
        stopTurnTimer();
    }
}

// 启动回合倒计时
function startTurnTimer() {
    // 清除之前的定时器
    stopTurnTimer();
    
    const timerDisplay = document.getElementById('timerCountdown');
    if (!timerDisplay) return;
    
    let timeLeft = 60; // 60秒
    timerDisplay.textContent = timeLeft;
    timerDisplay.className = 'timer-countdown';
    
    // 更新倒计时显示
    turnTimer = setInterval(() => {
        timeLeft--;
        timerDisplay.textContent = timeLeft;
        
        // 根据剩余时间改变颜色
        if (timeLeft <= 10) {
            timerDisplay.className = 'timer-countdown timer-warning';
        } else if (timeLeft <= 30) {
            timerDisplay.className = 'timer-countdown timer-urgent';
        } else {
            timerDisplay.className = 'timer-countdown';
        }
        
        if (timeLeft <= 0) {
            stopTurnTimer();
            timerDisplay.textContent = '0';
        }
    }, 1000);
}

// 停止回合倒计时
function stopTurnTimer() {
    if (turnTimer) {
        clearInterval(turnTimer);
        turnTimer = null;
    }
    
    const timerDisplay = document.getElementById('timerCountdown');
    if (timerDisplay) {
        timerDisplay.textContent = '60';
        timerDisplay.className = 'timer-countdown';
    }
}

function createCardHTML(card) {
    if (!card) return '';
    
    const suitSymbols = {
        'spades': '♠',
        'hearts': '♥',
        'diamonds': '♦',
        'clubs': '♣'
    };
    
    const isRed = card.suit === 'hearts' || card.suit === 'diamonds';
    const colorClass = isRed ? 'red' : 'black';
    const suit = suitSymbols[card.suit];
    
    return `
        <div class="card ${colorClass}">
            <div class="card-tl">${card.rank}<br>${suit}</div>
            <div class="card-center">${suit}</div>
            <div class="card-br">${card.rank}<br>${suit}</div>
        </div>
    `;
}

function showSettlement(data) {
    // 设置结算状态
    isSettlement = true;
    settlementData = data;
    
    // 隐藏操作面板
    const actionPanel = document.getElementById('actionPanel');
    if (actionPanel) {
        actionPanel.classList.add('hidden');
    }
    const waitingPanel = document.getElementById('waitingPanel');
    if (waitingPanel) {
        waitingPanel.classList.add('hidden');
    }
    
    // 显示结算信息面板
    const settlementPanel = document.getElementById('settlementInfo');
    const winnerNameEl = document.getElementById('settlementWinnerName');
    const potEl = document.getElementById('settlementPot');
    const handEl = document.getElementById('settlementHand');
    
    // 显示获胜者信息
    const winnerName = data.winner.name || '玩家' + data.winner.id.substring(0, 4);
    winnerNameEl.textContent = winnerName;
    potEl.textContent = data.pot || 0;
    
    if (data.winningHand) {
        handEl.textContent = `牌型: ${data.winningHand}`;
        handEl.style.display = 'block';
    } else {
        handEl.style.display = 'none';
    }
    
    // 更新公共牌显示（确保显示所有公共牌）
    // 优先使用gameEnded消息中的公共牌数据
    if (data.communityCards && Array.isArray(data.communityCards)) {
        console.log('结算时更新公共牌（从gameEnded消息）:', data.communityCards);
        updateCommunityCards(data.communityCards);
    } else if (gameState && gameState.communityCards && Array.isArray(gameState.communityCards)) {
        console.log('从gameState更新公共牌:', gameState.communityCards);
        updateCommunityCards(gameState.communityCards);
    } else {
        // 如果没有公共牌数据，清空显示
        console.log('没有公共牌数据，清空显示');
        updateCommunityCards([]);
    }
    
    // 更新玩家区域，显示玩家底牌
    // 结算时只显示未弃牌玩家的手牌，已弃牌的玩家不显示手牌
    if (data.allHands && Array.isArray(data.allHands) && data.allHands.length > 0) {
        // 使用gameState或currentRoom获取玩家列表
        const room = gameState || (typeof currentRoom === 'object' ? currentRoom : null);
        let updatedPlayers = [];
        
        if (room && room.players && Array.isArray(room.players) && room.players.length > 0) {
            // 更新玩家数据，包含手牌信息
            updatedPlayers = room.players
                .filter(p => p && p.id) // 过滤无效玩家
                .map(p => {
                    const handData = data.allHands.find(h => h && h.id === p.id);
                    if (handData) {
                        // 使用gameEnded消息中的手牌数据
                        return { 
                            ...p, 
                            hand: Array.isArray(handData.hand) ? handData.hand : [],
                            folded: handData.folded !== undefined ? handData.folded : p.folded,
                            chips: handData.chips !== undefined ? handData.chips : p.chips
                        };
                    }
                    return { ...p, hand: Array.isArray(p.hand) ? p.hand : [] };
                });
        } else {
            // 如果没有room数据，直接从allHands构建玩家列表
            updatedPlayers = data.allHands
                .filter(handData => handData && handData.id) // 过滤无效数据
                .map(handData => ({
                    id: handData.id,
                    name: handData.name || '玩家',
                    chips: handData.chips || 0,
                    bet: 0,
                    folded: handData.folded || false,
                    hand: Array.isArray(handData.hand) ? handData.hand : [],
                    isDealer: false,
                    isSmall: false,
                    isBig: false,
                    allIn: false
                }));
        }
        
        if (updatedPlayers.length > 0) {
            console.log('结算时更新玩家区域，玩家数量:', updatedPlayers.length, '手牌数据:', updatedPlayers.map(p => ({ id: p.id, handCount: p.hand ? p.hand.length : 0 })));
            updatePlayersArea(updatedPlayers, -1, room ? (room.dealerIndex || 0) : 0);
        }
    }
    
    // 显示结算信息面板，放在赢家旁边
    if (settlementPanel) {
        settlementPanel.classList.remove('hidden');
        
        // 找到赢家的座位元素
        const winnerSeat = document.querySelector('.player-seat.winner');
        if (winnerSeat) {
            // 获取赢家座位的位置
            const rect = winnerSeat.getBoundingClientRect();
            const seatTop = rect.top + window.scrollY;
            const seatLeft = rect.left + window.scrollX;
            const seatWidth = rect.width;
            const seatHeight = rect.height;
            
            // 将结算面板放在赢家座位旁边（右侧）
            settlementPanel.style.position = 'absolute';
            settlementPanel.style.top = (seatTop + seatHeight / 2) + 'px';
            settlementPanel.style.left = (seatLeft + seatWidth + 20) + 'px';
            settlementPanel.style.transform = 'translateY(-50%)';
        } else {
            // 如果找不到赢家座位，使用默认位置（右上角）
            settlementPanel.style.position = 'fixed';
            settlementPanel.style.top = '20px';
            settlementPanel.style.right = '20px';
            settlementPanel.style.left = 'auto';
            settlementPanel.style.transform = 'none';
        }
    }
}

// 分享房间功能
function shareRoom() {
    if (!currentRoom) {
        showError('没有房间信息');
        return;
    }
    
    // 生成分享链接
    const shareUrl = `${window.location.origin}${window.location.pathname}?room=${currentRoom}`;
    
    // 尝试使用Web Share API（移动端）
    if (navigator.share) {
        navigator.share({
            title: '德州扑克房间',
            text: `加入我的德州扑克房间，房间ID: ${currentRoom}`,
            url: shareUrl
        }).catch(err => {
            console.log('分享失败:', err);
            copyToClipboard(shareUrl);
        });
    } else {
        // 桌面端：复制到剪贴板
        copyToClipboard(shareUrl);
    }
}

// 复制到剪贴板
function copyToClipboard(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
            showError('链接已复制到剪贴板！');
        }).catch(err => {
            console.error('复制失败:', err);
            fallbackCopyToClipboard(text);
        });
    } else {
        fallbackCopyToClipboard(text);
    }
}

// 备用复制方法
function fallbackCopyToClipboard(text) {
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.opacity = '0';
    document.body.appendChild(textArea);
    textArea.select();
    try {
        document.execCommand('copy');
        showError('链接已复制到剪贴板！');
    } catch (err) {
        console.error('复制失败:', err);
        showError('复制失败，请手动复制链接: ' + text);
    }
    document.body.removeChild(textArea);
}

// 检查URL参数，如果有房间ID，自动填充
function checkUrlParams() {
    const urlParams = new URLSearchParams(window.location.search);
    const roomId = urlParams.get('room');
    if (roomId) {
        const roomIdInput = document.getElementById('roomId');
        if (roomIdInput) {
            roomIdInput.value = roomId;
        }
        // 如果已经有名字，自动加入
        const playerNameInput = document.getElementById('playerName');
        if (playerNameInput && playerNameInput.value.trim()) {
            // 延迟一下，确保WebSocket已连接
            setTimeout(() => {
                joinGame();
            }, 500);
        }
    }
}

// 下一局按钮事件
document.addEventListener('DOMContentLoaded', () => {
    const nextHandBtn = document.getElementById('nextHandBtn');
    if (nextHandBtn) {
        nextHandBtn.addEventListener('click', () => {
            console.log('点击下一局按钮');
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'startGame',
                    data: {}
                }));
            }
            // 隐藏结算界面
            const overlay = document.getElementById('settlementOverlay');
            if (overlay) {
                overlay.classList.add('hidden');
            }
        });
    }
});

function showScreen(screenName) {
    loginScreen.classList.add('hidden');
    lobbyScreen.classList.add('hidden');
    gameScreen.classList.add('hidden');
    gameEndScreen.classList.add('hidden');
    
    document.getElementById(screenName).classList.remove('hidden');
}

function showError(message) {
    console.error('显示错误:', message);
    // 尝试在登录界面显示
    const loginErrorDiv = document.getElementById('loginError');
    if (loginErrorDiv && !loginScreen.classList.contains('hidden')) {
        loginErrorDiv.textContent = message;
        loginErrorDiv.style.display = 'block';
        setTimeout(() => {
            loginErrorDiv.textContent = '';
            loginErrorDiv.style.display = 'none';
        }, 5000);
        return;
    }
    
    // 尝试在大厅界面显示
    const lobbyErrorDiv = document.getElementById('lobbyError');
    if (lobbyErrorDiv && !lobbyScreen.classList.contains('hidden')) {
        lobbyErrorDiv.textContent = message;
        lobbyErrorDiv.style.display = 'block';
        setTimeout(() => {
            lobbyErrorDiv.textContent = '';
            lobbyErrorDiv.style.display = 'none';
        }, 5000);
        return;
    }
    
    // 如果都不在，使用alert
    alert(message);
}

// 保存当前玩家信息
function setCurrentPlayer(player) {
    currentPlayer = player;
}

// 启动心跳
function startHeartbeat() {
    // 清除旧的定时器
    if (heartbeatInterval) {
        clearInterval(heartbeatInterval);
    }
    
    // 每20秒发送一次心跳
    heartbeatInterval = setInterval(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'heartbeat',
                data: {}
            }));
        }
    }, 20000); // 20秒发送一次，30秒超时
}

// 停止心跳
function stopHeartbeat() {
    if (heartbeatInterval) {
        clearInterval(heartbeatInterval);
        heartbeatInterval = null;
    }
}

// 上桌功能
function joinTable() {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        showError('连接未建立，请刷新页面');
        return;
    }
    
    ws.send(JSON.stringify({
        type: 'joinTable',
        data: {}
    }));
}

// 显示观战面板
function showSpectatingPanel(room) {
    const spectatingPanel = document.getElementById('spectatingPanel');
    const actionPanel = document.getElementById('actionPanel');
    const waitingPanel = document.getElementById('waitingPanel');
    
    if (spectatingPanel) {
        spectatingPanel.classList.remove('hidden');
        
        // 更新筹码显示
        const playerName = document.getElementById('playerName')?.value.trim();
        if (room && room.spectators) {
            const spectator = room.spectators.find(p => p && p.name === playerName);
            if (spectator) {
                const chipsEl = document.getElementById('playerChipsSpectating');
                if (chipsEl) {
                    chipsEl.textContent = spectator.chips || 500;
                }
            }
        }
    }
    
    if (actionPanel) {
        actionPanel.classList.add('hidden');
    }
    
    if (waitingPanel) {
        waitingPanel.classList.add('hidden');
    }
    
    // 清空手牌显示
    const handCard0 = document.getElementById('handCard0');
    const handCard1 = document.getElementById('handCard1');
    if (handCard0) handCard0.innerHTML = '';
    if (handCard1) handCard1.innerHTML = '';
}

// 隐藏观战面板
function hideSpectatingPanel() {
    const spectatingPanel = document.getElementById('spectatingPanel');
    if (spectatingPanel) {
        spectatingPanel.classList.add('hidden');
    }
}

// 显示买一手统计
function showBuyHandStats(stats) {
    const modal = document.getElementById('buyHandStatsModal');
    const statsList = document.getElementById('buyHandStatsList');
    
    if (!modal || !statsList) {
        return;
    }
    
    // 清空列表
    statsList.innerHTML = '';
    
    if (!stats || Object.keys(stats).length === 0) {
        statsList.innerHTML = '<p style="text-align: center; color: #999; padding: 20px;">暂无统计数据</p>';
    } else {
        // 转换为数组并排序（按次数降序）
        const statsArray = Object.entries(stats)
            .map(([name, count]) => ({ name, count }))
            .sort((a, b) => b.count - a.count);
        
        // 创建列表
        const list = document.createElement('ul');
        list.style.listStyle = 'none';
        list.style.padding = '0';
        list.style.margin = '0';
        
        statsArray.forEach(({ name, count }) => {
            const item = document.createElement('li');
            item.style.padding = '12px 15px';
            item.style.borderBottom = '1px solid rgba(255, 255, 255, 0.1)';
            item.style.display = 'flex';
            item.style.justifyContent = 'space-between';
            item.style.alignItems = 'center';
            
            const nameSpan = document.createElement('span');
            nameSpan.textContent = name;
            nameSpan.style.fontWeight = 'bold';
            nameSpan.style.color = '#fff';
            
            const countSpan = document.createElement('span');
            countSpan.textContent = `${count} 次`;
            countSpan.style.color = '#4CAF50';
            countSpan.style.fontWeight = 'bold';
            countSpan.style.fontSize = '1.1em';
            
            item.appendChild(nameSpan);
            item.appendChild(countSpan);
            list.appendChild(item);
        });
        
        statsList.appendChild(list);
    }
    
    // 显示模态框
    modal.classList.remove('hidden');
}
