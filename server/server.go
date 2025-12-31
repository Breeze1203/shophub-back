package server

import (
	"LiteAdmin/config"
	"LiteAdmin/handlers"
	"LiteAdmin/limiter"
	custommiddleware "LiteAdmin/middleware"
	"LiteAdmin/models"
	"LiteAdmin/redis"
	"LiteAdmin/services"
	"time"

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
	CategoryHandler        *handlers.CategoryServiceHandler
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
	roomService := services.NewRoomService(db, &cfg.RedisConfig)
	customerHandler := handlers.NewCustomerServiceHandler(db)
	authHandler := handlers.NewAuthHandler(authService, oauthService)
	roomHandler := handlers.NewRoomHandler(roomService)
	categoryHandler := handlers.NewCategoryHandler(db)
	chatWebSocketHandler := handlers.NewChatWebSocketHandler(db, redis.GetRedis(&cfg.RedisConfig).Client)
	s := &Server{
		Echo:                   e,
		DB:                     db,
		Config:                 &cfg,
		AuthHandler:            authHandler,
		RoomHandler:            roomHandler,
		ChatWebSocketHandler:   chatWebSocketHandler,
		CustomerServiceHandler: customerHandler,
		CategoryHandler:        categoryHandler,
	}
	// --- 设置路由中间件 ---
	strategy := &limiter.TokenBucketStrategy{}
	limitManager := limiter.NewManager(redis.GetRedis(&cfg.RedisConfig).Client, strategy)
	limiterConfig := custommiddleware.RateLimitConfig{
		Limit:  10,              // 桶容量 / 限制次数
		Window: 1 * time.Second, // 时间单位
		KeyFunc: func(c echo.Context) string {
			return c.RealIP() + ":" + c.Path()
		},
	}
	authMiddleware := custommiddleware.AuthMiddleware(authService)
	adminMiddleware := custommiddleware.AdminAuthMiddleware()
	limitMiddleware := custommiddleware.NewRateLimitMiddleware(limitManager, limiterConfig)
	s.SetupRoutes(authMiddleware, adminMiddleware, limitMiddleware)
	return s
}

func (s *Server) Start(addr string) {
	log.Fatal(s.Echo.Start(addr))
}
