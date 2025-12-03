package server

import (
	"github.com/labstack/echo/v4"
)

func (s *Server) SetupRoutes(authMiddleware echo.MiddlewareFunc) {
	e := s.Echo
	api := e.Group("/api/v1")
	// Auth routes (unprotected)
	auth := api.Group("/auth")
	{
		// Get available OAuth providers
		auth.GET("/providers", s.AuthHandler.GetProviders)
		// Local authentication
		auth.POST("/register", s.AuthHandler.Register)
		auth.POST("/login", s.AuthHandler.Login)
		auth.POST("/refresh", s.AuthHandler.RefreshToken)
		// OAuth routes
		auth.GET("/oauth/:provider", s.AuthHandler.OAuthLogin)
		auth.GET("/oauth/:provider/callback", s.AuthHandler.OAuthCallback)
	}

	// Protected routes (apply authMiddleware to all below)
	protected := api.Group("")
	protected.Use(authMiddleware)
	{
		// User routes
		protected.GET("/user", s.AuthHandler.GetCurrentUser)

		// Rooms routes
		rooms := protected.Group("/rooms")
		{
			rooms.POST("", s.RoomHandler.CreateRoom)        // 创建房间
			rooms.GET("", s.RoomHandler.ListRooms)          // 获取房间列表
			rooms.GET("/:id", s.RoomHandler.GetRoom)        // 获取单个房间
			rooms.POST("/:id/join", s.RoomHandler.JoinRoom) // 加入房间（验证密码）
			rooms.DELETE("/:id", s.RoomHandler.DeleteRoom)  // 删除房间
		}

		// Chat routes
		chat := protected.Group("/chat")
		{
			chat.GET("/:roomId/messages", s.ChatWebSocketHandler.GetMessages)        // 获取历史消息
			chat.GET("/:roomId/online-users", s.ChatWebSocketHandler.GetOnlineUsers) // 获取在线用户列表
		}
		protected.GET("/chat/:roomId/ws", s.ChatWebSocketHandler.HandleWebSocket)
	}
}
