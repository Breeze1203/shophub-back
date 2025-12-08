package server

import (
	"LiteAdmin/config"
	"LiteAdmin/handlers"
	custommiddleware "LiteAdmin/middleware"
	"LiteAdmin/models"
	"LiteAdmin/redis"
	"LiteAdmin/services"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Server struct {
	Echo                   *echo.Echo
	DB                     *gorm.DB
	Config                 *config.Config
	AuthHandler            *handlers.AuthHandler
	RoomHandler            *handlers.RoomHandler
	ChatWebSocketHandler   *handlers.ChatWebSocketHandler
	CustomerServiceHandler *handlers.CustomerServiceHandler
}

func NewServer() *Server {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	if err := models.AutoMigrateAll(db); err != nil {
		log.Fatal("Failed to auto-migrate database:", err)
	}
	// 初始化 Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.PATCH},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
		ExposeHeaders:    []string{echo.HeaderContentLength},
		MaxAge:           86400,
	}))
	authService := services.NewAuthService(db, &cfg.Auth)
	oauthService := services.NewOAuthService(&cfg.Auth)
	roomService := services.NewRoomService(db)
	customerHandler := handlers.NewCustomerServiceHandler(db)
	authHandler := handlers.NewAuthHandler(authService, oauthService)
	roomHandler := handlers.NewRoomHandler(roomService)
	chatWebSocketHandler := handlers.NewChatWebSocketHandler(db, redis.GetRedis().Client)
	s := &Server{
		Echo:                   e,
		DB:                     db,
		Config:                 &cfg,
		AuthHandler:            authHandler,
		RoomHandler:            roomHandler,
		ChatWebSocketHandler:   chatWebSocketHandler,
		CustomerServiceHandler: customerHandler,
	}
	// --- 设置路由 ---
	authMiddleware := custommiddleware.AuthMiddleware(authService)
	s.SetupRoutes(authMiddleware)
	return s
}

func (s *Server) Start(addr string) {
	log.Fatal(s.Echo.Start(addr))
}
