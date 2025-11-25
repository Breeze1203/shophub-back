package handlers

import (
	"LiteAdmin/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 消息结构
type BroadcastMessage struct {
	Data      map[string]interface{} // 要广播的消息数据
	ExceptIDs map[string]bool        // 排除的客户端ID（不发送给这些客户端）
}

// 用户信息结构（用于在线列表）
type UserInfo struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Color    string `json:"color"`
}

// 聊天客户端 代表一个 WebSocket 连接的客户端，包含连接、用户信息和消息通道
type ChatClient struct {
	ID       string                      // 客户端唯一标识（UUID）
	UserID   uint                        // 用户数据库ID
	Username string                      // 用户名
	Color    string                      // 用户颜色标识
	Conn     *websocket.Conn             // WebSocket连接
	Room     *ChatRoom                   // 所属聊天室
	Send     chan map[string]interface{} // 发送消息队列（缓冲256条）
	ctx      context.Context             // 上下文管理
	cancel   context.CancelFunc          // 取消函数
}

// 管理一个聊天室内的所有连接和消息分发
type ChatRoom struct {
	ID         string                 // 房间ID
	Clients    map[string]*ChatClient // 房间内所有客户端
	mu         sync.RWMutex           // 读写锁（保护Clients）
	Broadcast  chan *BroadcastMessage // 广播消息通道（缓冲256条）
	Register   chan *ChatClient       // 客户端注册通道（缓冲16个）
	Unregister chan *ChatClient       // 客户端注销通道（缓冲16个）
	ctx        context.Context        // 房间上下文
	cancel     context.CancelFunc     // 房间关闭函数
	redis      *redis.Client          // Redis客户端
}

// 房间管理器
type ChatRoomManager struct {
	rooms map[string]*ChatRoom // 所有聊天室
	mu    sync.RWMutex         // 读写锁
	redis *redis.Client        // Redis客户端
}

func NewChatRoomManager(redisClient *redis.Client) *ChatRoomManager {
	return &ChatRoomManager{
		rooms: make(map[string]*ChatRoom),
		redis: redisClient,
	}
}

func (m *ChatRoomManager) GetOrCreateRoom(roomID string) *ChatRoom {
	m.mu.Lock()
	defer m.mu.Unlock()

	if room, exists := m.rooms[roomID]; exists {
		return room
	}

	ctx, cancel := context.WithCancel(context.Background())
	room := &ChatRoom{
		ID:         roomID,
		Clients:    make(map[string]*ChatClient),
		Broadcast:  make(chan *BroadcastMessage, 256),
		Register:   make(chan *ChatClient, 16),
		Unregister: make(chan *ChatClient, 16),
		ctx:        ctx,
		cancel:     cancel,
		redis:      m.redis,
	}
	m.rooms[roomID] = room

	go room.run()

	return room
}

// 房间的核心消息分发循环
func (room *ChatRoom) run() {
	for {
		select {
		case <-room.ctx.Done():
			return

		case client := <-room.Register:
			room.mu.Lock()
			room.Clients[client.ID] = client
			room.mu.Unlock()

			// 添加用户到Redis在线列表
			room.addUserToRedis(client)

		case client := <-room.Unregister:
			room.mu.Lock()
			if _, ok := room.Clients[client.ID]; ok {
				delete(room.Clients, client.ID)
				close(client.Send)
			}
			room.mu.Unlock()

			// 从Redis在线列表移除用户
			room.removeUserFromRedis(client)

		case message := <-room.Broadcast:
			room.mu.RLock()
			clients := make([]*ChatClient, 0, len(room.Clients))
			for _, client := range room.Clients {
				clients = append(clients, client)
			}
			room.mu.RUnlock()

			for _, client := range clients {
				if message.ExceptIDs != nil && message.ExceptIDs[client.ID] {
					continue
				}

				select {
				case client.Send <- message.Data:
				default:
					log.Printf("Client %s send buffer full, disconnecting", client.ID)
					room.Unregister <- client
				}
			}
		}
	}
}

// 添加用户到Redis在线列表
func (room *ChatRoom) addUserToRedis(client *ChatClient) {
	ctx := context.Background()
	key := fmt.Sprintf("chat:room:%s:online_users", room.ID)

	userInfo := UserInfo{
		UserID:   client.UserID,
		Username: client.Username,
		Color:    client.Color,
	}

	data, err := json.Marshal(userInfo)
	if err != nil {
		log.Printf("Failed to marshal user info: %v", err)
		return
	}

	// 使用Hash存储，field为user_id，value为用户信息JSON
	field := fmt.Sprintf("%d", client.UserID)
	if err := room.redis.HSet(ctx, key, field, data).Err(); err != nil {
		log.Printf("Failed to add user to Redis: %v", err)
		return
	}

	// 设置过期时间（24小时）
	room.redis.Expire(ctx, key, 24*time.Hour)
}

// 从Redis在线列表移除用户
func (room *ChatRoom) removeUserFromRedis(client *ChatClient) {
	ctx := context.Background()
	key := fmt.Sprintf("chat:room:%s:online_users", room.ID)

	// 检查是否还有其他连接使用同一个user_id
	room.mu.RLock()
	hasOtherConnection := false
	for _, c := range room.Clients {
		if c.UserID == client.UserID && c.ID != client.ID {
			hasOtherConnection = true
			break
		}
	}
	room.mu.RUnlock()

	// 只有在没有其他连接时才从Redis删除
	if !hasOtherConnection {
		field := fmt.Sprintf("%d", client.UserID)
		if err := room.redis.HDel(ctx, key, field).Err(); err != nil {
			log.Printf("Failed to remove user from Redis: %v", err)
		}
	}
}

// 从Redis获取房间当前在线用户列表
func (room *ChatRoom) GetOnlineUsersFromRedis() ([]UserInfo, error) {
	ctx := context.Background()
	key := fmt.Sprintf("chat:room:%s:online_users", room.ID)

	// 获取所有在线用户
	result, err := room.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	users := make([]UserInfo, 0, len(result))
	for _, data := range result {
		var userInfo UserInfo
		if err := json.Unmarshal([]byte(data), &userInfo); err != nil {
			log.Printf("Failed to unmarshal user info: %v", err)
			continue
		}
		users = append(users, userInfo)
	}

	return users, nil
}

type ChatWebSocketHandler struct {
	db          *gorm.DB             // 数据库连接
	redis       *redis.Client        // Redis客户端
	roomManager *ChatRoomManager     // 房间管理器
	dbQueue     chan *models.Message // 数据库写入队列（缓冲1000条）
	dbWorkers   int                  // 数据库工作协程数（4个）
}

func NewChatWebSocketHandler(db *gorm.DB, redisClient *redis.Client) *ChatWebSocketHandler {
	h := &ChatWebSocketHandler{
		db:          db,
		redis:       redisClient,
		roomManager: NewChatRoomManager(redisClient),
		dbQueue:     make(chan *models.Message, 1000),
		dbWorkers:   4,
	}

	for i := 0; i < h.dbWorkers; i++ {
		go h.dbWorker()
	}

	return h
}

func (h *ChatWebSocketHandler) dbWorker() {
	for message := range h.dbQueue {
		if err := h.db.Create(message).Error; err != nil {
			log.Printf("Failed to save message: %v", err)
		}
	}
}

func (h *ChatWebSocketHandler) HandleWebSocket(c echo.Context) error {
	roomID := c.Param("roomId")
	user := c.Get("user").(*models.User)

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &ChatClient{
		ID:       uuid.New().String(),
		UserID:   user.ID,
		Username: user.Username,
		Color:    getUserColor(user.ID),
		Conn:     ws,
		Send:     make(chan map[string]interface{}, 256),
		ctx:      ctx,
		cancel:   cancel,
	}

	room := h.roomManager.GetOrCreateRoom(roomID)
	client.Room = room

	// 注册到房间
	room.Register <- client

	// 发送初始化数据
	h.sendInitData(client, room)

	// 广播用户加入（通知其他用户）
	h.broadcastUserJoined(room, client)

	// 发送系统消息：用户加入
	h.sendSystemMessage(room, client, "joined")

	// 启动写入goroutine
	go h.writePump(client)

	// 当前goroutine处理读取
	h.readPump(client)

	return nil
}

// 读取客户端消息
func (h *ChatWebSocketHandler) readPump(client *ChatClient) {
	defer func() {
		client.cancel()
		client.Room.Unregister <- client
		client.Conn.Close()

		// 广播用户离开
		h.broadcastUserLeft(client.Room, client)

		// 发送系统消息：用户离开
		h.sendSystemMessage(client.Room, client, "left")
	}()

	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg map[string]interface{}
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		h.handleMessage(client, msg)
	}
}

// 向客户端写入消息
func (h *ChatWebSocketHandler) writePump(client *ChatClient) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case <-client.ctx.Done():
			return

		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteJSON(message); err != nil {
				log.Printf("WriteJSON error: %v", err)
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// 发送初始化数据（从Redis获取在线用户列表）
func (h *ChatWebSocketHandler) sendInitData(client *ChatClient, room *ChatRoom) {
	users, err := room.GetOnlineUsersFromRedis()
	if err != nil {
		log.Printf("Failed to get online users from Redis: %v", err)
		users = []UserInfo{}
	}

	initMsg := map[string]interface{}{
		"type": "init",
		"payload": map[string]interface{}{
			"users": users,
		},
	}

	client.Send <- initMsg
}

// 发送系统消息（用户加入/离开）
func (h *ChatWebSocketHandler) sendSystemMessage(room *ChatRoom, client *ChatClient, action string) {
	var content string
	if action == "joined" {
		content = client.Username + " 加入了聊天室"
	} else if action == "left" {
		content = client.Username + " 离开了聊天室"
	}
	systemMsg := map[string]interface{}{
		"type": "message",
		"payload": map[string]interface{}{
			"id":         uuid.New().String(),
			"room_id":    room.ID,
			"type":       "system",
			"content":    content,
			"created_at": time.Now(),
		},
	}

	room.Broadcast <- &BroadcastMessage{
		Data: systemMsg,
	}
}

// 消息类型分发
func (h *ChatWebSocketHandler) handleMessage(client *ChatClient, msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	payload, _ := msg["payload"].(map[string]interface{})

	switch msgType {
	case "message":
		h.handleChatMessage(client, payload)
	case "typing":
		h.handleTyping(client, payload)
	}
}

// 异步数据库写入
func (h *ChatWebSocketHandler) handleChatMessage(client *ChatClient, payload map[string]interface{}) {
	content, ok := payload["content"].(string)
	if !ok || content == "" {
		return
	}

	now := time.Now()
	message := models.Message{
		RoomID:    client.Room.ID,
		UserID:    client.UserID,
		Content:   content,
		Type:      "text",
		CreatedAt: now,
	}

	// 异步保存到数据库
	select {
	case h.dbQueue <- &message:
	default:
		log.Println("Database queue full, dropping message")
	}

	// 立即广播消息
	broadcastMsg := map[string]interface{}{
		"type": "message",
		"payload": map[string]interface{}{
			"id":         message.ID,
			"room_id":    message.RoomID,
			"user_id":    client.UserID,
			"username":   client.Username,
			"user_color": client.Color,
			"content":    content,
			"type":       "text",
			"created_at": now,
		},
	}

	client.Room.Broadcast <- &BroadcastMessage{
		Data: broadcastMsg,
	}
}

func (h *ChatWebSocketHandler) handleTyping(client *ChatClient, payload map[string]interface{}) {
	isTyping, ok := payload["is_typing"].(bool)
	if !ok {
		return
	}

	typingMsg := map[string]interface{}{
		"type": "typing",
		"payload": map[string]interface{}{
			"user_id":   client.UserID,
			"username":  client.Username,
			"is_typing": isTyping,
		},
	}

	client.Room.Broadcast <- &BroadcastMessage{
		Data:      typingMsg,
		ExceptIDs: map[string]bool{client.ID: true},
	}
}

// 广播用户加入
func (h *ChatWebSocketHandler) broadcastUserJoined(room *ChatRoom, client *ChatClient) {
	msg := map[string]interface{}{
		"type": "user_joined",
		"payload": map[string]interface{}{
			"user_id":             client.UserID,
			"username":            client.Username,
			"color":               client.Color,
			"system_message_sent": true,
		},
	}

	room.Broadcast <- &BroadcastMessage{
		Data:      msg,
		ExceptIDs: map[string]bool{client.ID: true},
	}
}

// 广播用户离开
func (h *ChatWebSocketHandler) broadcastUserLeft(room *ChatRoom, client *ChatClient) {
	msg := map[string]interface{}{
		"type": "user_left",
		"payload": map[string]interface{}{
			"user_id":             client.UserID,
			"username":            client.Username,
			"system_message_sent": true,
		},
	}

	room.Broadcast <- &BroadcastMessage{
		Data: msg,
	}
}

// HTTP接口：获取房间在线用户列表
func (h *ChatWebSocketHandler) GetOnlineUsers(c echo.Context) error {
	roomID := c.Param("roomId")

	ctx := context.Background()
	key := fmt.Sprintf("chat:room:%s:online_users", roomID)

	// 从Redis获取所有在线用户
	result, err := h.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch online users",
		})
	}
	users := make([]UserInfo, 0, len(result))
	for _, data := range result {
		var userInfo UserInfo
		if err := json.Unmarshal([]byte(data), &userInfo); err != nil {
			log.Printf("Failed to unmarshal user info: %v", err)
			continue
		}
		users = append(users, userInfo)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"room_id": roomID,
		"count":   len(users),
		"users":   users,
	})
}

// 获取聊天历史消息
func (h *ChatWebSocketHandler) GetMessages(c echo.Context) error {
	roomID := c.Param("roomId")

	// 分页参数
	limit := 50
	offset := 0
	if c.QueryParam("offset") != "" {
		fmt.Sscanf(c.QueryParam("offset"), "%d", &offset)
	}

	var messages []struct {
		models.Message
		Username  string `json:"username"`
		UserColor string `json:"user_color"`
	}

	err := h.db.Raw(`
		SELECT messages.*, users.username, users.avatar as user_color
		FROM messages
		LEFT JOIN users ON messages.user_id = users.id
		WHERE messages.room_id = ?
		ORDER BY messages.created_at ASC
		LIMIT ? OFFSET ?
	`, roomID, limit, offset).Scan(&messages).Error

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch messages",
		})
	}

	return c.JSON(http.StatusOK, messages)
}

// 获取用户颜色（根据ID生成）
func getUserColor(userID uint) string {
	colors := []string{"#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A", "#98D8C8", "#F7DC6F", "#BB8FCE"}
	return colors[userID%uint(len(colors))]
}
