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

// WhiteboardClient 画板客户端
type WhiteboardClient struct {
	ID       string
	UserID   uint
	Username string
	Color    string
	Conn     *websocket.Conn
	Room     *WhiteboardRoom
	Send     chan map[string]interface{}
	ctx      context.Context
	cancel   context.CancelFunc
}

// WhiteboardRoom 画板房间
type WhiteboardRoom struct {
	ID         string
	Clients    map[string]*WhiteboardClient
	mu         sync.RWMutex
	Broadcast  chan *BroadcastMessage
	Register   chan *WhiteboardClient
	Unregister chan *WhiteboardClient
	CanvasData string
	ctx        context.Context
	cancel     context.CancelFunc
	redis      *redis.Client
}

// WhiteboardRoomManager 画板房间管理器
type WhiteboardRoomManager struct {
	rooms map[string]*WhiteboardRoom
	mu    sync.RWMutex
	redis *redis.Client
}

func NewWhiteboardRoomManager(redisClient *redis.Client) *WhiteboardRoomManager {
	return &WhiteboardRoomManager{
		rooms: make(map[string]*WhiteboardRoom),
		redis: redisClient,
	}
}

func (m *WhiteboardRoomManager) GetOrCreateRoom(roomID string) *WhiteboardRoom {
	m.mu.Lock()
	defer m.mu.Unlock()

	if room, exists := m.rooms[roomID]; exists {
		return room
	}

	ctx, cancel := context.WithCancel(context.Background())
	room := &WhiteboardRoom{
		ID:         roomID,
		Clients:    make(map[string]*WhiteboardClient),
		Broadcast:  make(chan *BroadcastMessage, 256),
		Register:   make(chan *WhiteboardClient, 16),
		Unregister: make(chan *WhiteboardClient, 16),
		ctx:        ctx,
		cancel:     cancel,
		redis:      m.redis, // 🔑 初始化redis字段
	}
	m.rooms[roomID] = room

	go room.run()

	return room
}

func (room *WhiteboardRoom) run() {
	for {
		select {
		case <-room.ctx.Done():
			return

		case client := <-room.Register:
			room.mu.Lock()
			room.Clients[client.ID] = client
			room.mu.Unlock()

			// 🔑 添加用户到Redis
			room.addUserToRedis(client)

		case client := <-room.Unregister:
			room.mu.Lock()
			if _, ok := room.Clients[client.ID]; ok {
				delete(room.Clients, client.ID)
				close(client.Send)
			}
			room.mu.Unlock()

			// 🔑 从Redis移除用户
			room.removeUserFromRedis(client)

		case message := <-room.Broadcast:
			room.mu.RLock()
			clients := make([]*WhiteboardClient, 0, len(room.Clients))
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

// 🔑 添加用户到Redis
func (room *WhiteboardRoom) addUserToRedis(client *WhiteboardClient) {
	if room.redis == nil {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("whiteboard:room:%s:online_users", room.ID)
	field := fmt.Sprintf("%d", client.UserID)

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

	if err := room.redis.HSet(ctx, key, field, data).Err(); err != nil {
		log.Printf("Failed to add user to Redis: %v", err)
		return
	}

	// 设置过期时间（24小时）
	room.redis.Expire(ctx, key, 24*time.Hour)
}

// 🔑 从Redis移除用户
func (room *WhiteboardRoom) removeUserFromRedis(client *WhiteboardClient) {
	if room.redis == nil {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("whiteboard:room:%s:online_users", room.ID)

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

// 🔑 从Redis获取在线用户列表（统一方法名）
func (room *WhiteboardRoom) GetOnlineUsers() ([]UserInfo, error) {
	if room.redis == nil {
		return []UserInfo{}, nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("whiteboard:room:%s:online_users", room.ID)

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

// WhiteboardWebSocketHandler 画板WebSocket处理器
type WhiteboardWebSocketHandler struct {
	db          *gorm.DB
	roomManager *WhiteboardRoomManager
}

func NewWhiteboardWebSocketHandler(db *gorm.DB, redisClient *redis.Client) *WhiteboardWebSocketHandler {
	return &WhiteboardWebSocketHandler{
		db:          db,
		roomManager: NewWhiteboardRoomManager(redisClient),
	}
}

func (h *WhiteboardWebSocketHandler) HandleWebSocket(c echo.Context) error {
	roomID := c.Param("roomId")
	user := c.Get("user").(*models.User)

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &WhiteboardClient{
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

	// 广播用户加入
	h.broadcastUserJoined(room, client)

	// 启动写入goroutine
	go h.writePump(client)

	// 当前goroutine处理读取
	h.readPump(client)

	return nil
}

func (h *WhiteboardWebSocketHandler) readPump(client *WhiteboardClient) {
	defer func() {
		client.cancel()
		client.Room.Unregister <- client
		client.Conn.Close()
		h.broadcastUserLeft(client.Room, client)
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

func (h *WhiteboardWebSocketHandler) writePump(client *WhiteboardClient) {
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

func (h *WhiteboardWebSocketHandler) sendInitData(client *WhiteboardClient, room *WhiteboardRoom) {
	// 🔑 使用统一的方法名
	users, err := room.GetOnlineUsers()
	if err != nil {
		log.Printf("Failed to get online users from Redis: %v", err)
		users = []UserInfo{}
	}

	initMsg := map[string]interface{}{
		"type": "init",
		"payload": map[string]interface{}{
			"users":       users,
			"canvas_data": room.CanvasData,
		},
	}

	client.Send <- initMsg
}

func (h *WhiteboardWebSocketHandler) handleMessage(client *WhiteboardClient, msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	payload, _ := msg["payload"].(map[string]interface{})

	switch msgType {
	case "draw":
		h.handleDraw(client, payload)
	case "cursor":
		h.handleCursor(client, payload)
	case "clear":
		h.handleClear(client)
	case "save":
		h.handleSave(client, payload)
	}
}

func (h *WhiteboardWebSocketHandler) handleDraw(client *WhiteboardClient, payload map[string]interface{}) {
	drawMsg := map[string]interface{}{
		"type":    "draw",
		"payload": payload,
	}

	client.Room.Broadcast <- &BroadcastMessage{
		Data:      drawMsg,
		ExceptIDs: map[string]bool{client.ID: true},
	}
}

func (h *WhiteboardWebSocketHandler) handleCursor(client *WhiteboardClient, payload map[string]interface{}) {
	cursorMsg := map[string]interface{}{
		"type": "cursor",
		"payload": map[string]interface{}{
			"user_id":  client.UserID,
			"username": client.Username,
			"color":    client.Color,
			"x":        payload["x"],
			"y":        payload["y"],
		},
	}

	client.Room.Broadcast <- &BroadcastMessage{
		Data:      cursorMsg,
		ExceptIDs: map[string]bool{client.ID: true},
	}
}

func (h *WhiteboardWebSocketHandler) handleClear(client *WhiteboardClient) {
	client.Room.mu.Lock()
	client.Room.CanvasData = ""
	client.Room.mu.Unlock()

	clearMsg := map[string]interface{}{
		"type":    "clear",
		"payload": map[string]interface{}{},
	}

	client.Room.Broadcast <- &BroadcastMessage{
		Data: clearMsg,
	}
}

func (h *WhiteboardWebSocketHandler) handleSave(client *WhiteboardClient, payload map[string]interface{}) {
	canvasData, ok := payload["canvas_data"].(string)
	if !ok {
		return
	}

	client.Room.mu.Lock()
	client.Room.CanvasData = canvasData
	client.Room.mu.Unlock()
}

func (h *WhiteboardWebSocketHandler) broadcastUserJoined(room *WhiteboardRoom, client *WhiteboardClient) {
	msg := map[string]interface{}{
		"type": "user_joined",
		"payload": map[string]interface{}{
			"user_id":  client.UserID,
			"username": client.Username,
			"color":    client.Color,
		},
	}

	room.Broadcast <- &BroadcastMessage{
		Data:      msg,
		ExceptIDs: map[string]bool{client.ID: true},
	}
}

func (h *WhiteboardWebSocketHandler) broadcastUserLeft(room *WhiteboardRoom, client *WhiteboardClient) {
	msg := map[string]interface{}{
		"type": "user_left",
		"payload": map[string]interface{}{
			"user_id":  client.UserID,
			"username": client.Username,
		},
	}

	room.Broadcast <- &BroadcastMessage{
		Data: msg,
	}
}

// 🔑 获取画板房间在线用户API
func (h *WhiteboardWebSocketHandler) GetWhiteboardRoomOnlineUsers(c echo.Context) error {
	roomID := c.Param("roomId")

	room := h.roomManager.GetOrCreateRoom(roomID)
	if room == nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "房间不存在",
		})
	}

	users, err := room.GetOnlineUsers()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "获取在线用户失败",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
	})
}
