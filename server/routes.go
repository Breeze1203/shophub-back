package server

import (
	"github.com/labstack/echo/v4"
)

func (s *Server) SetupRoutes(authMiddleware echo.MiddlewareFunc, adminMiddleware echo.MiddlewareFunc) {
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
	// 公开路由
	public := api.Group("/public")
	{
		public.GET("/categories", s.CategoryHandler.GetCategories)        // 获取分类树
		public.GET("/categories/all", s.CategoryHandler.GetAllCategories) // 获取所有分类
		public.GET("/categories/:id", s.CategoryHandler.GetCategoryByID)  // 获取分类详情
	}
	// 需要认证
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
		customer := protected.Group("/customer")
		{
			customer.POST("/session", s.CustomerServiceHandler.CreateOrGetSession)             // 用户创建会话
			customer.GET("/sessions", s.CustomerServiceHandler.GetAllSessions)                 // 管理员获取会话列表
			customer.PUT("/sessions/:sessionId", s.CustomerServiceHandler.UpdateSessionStatus) // 更新状态
		}
		admin := e.Group("/admin")
		admin.Use(adminMiddleware)
		admin.POST("/categories", s.CategoryHandler.CreateCategory)       // 创建分类
		admin.PUT("/categories/:id", s.CategoryHandler.UpdateCategory)    // 更新分类
		admin.DELETE("/categories/:id", s.CategoryHandler.DeleteCategory) // 删除分类
	}
}
