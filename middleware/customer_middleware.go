package middleware

import (
	"LiteAdmin/limiter"
	"LiteAdmin/models"
	"LiteAdmin/services"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func AuthMiddleware(authService *services.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			var tokenString string
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || parts[0] != "Bearer" {
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "invalid authorization header",
					})
				}
				tokenString = parts[1]
			} else {
				tokenString = c.QueryParam("token")
				if tokenString == "" {
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "missing authorization token",
					})
				}
				tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
			}

			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid token",
				})
			}
			var user models.User
			if err := authService.Db.First(&user, claims.UserID).Error; err != nil {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": "user not found",
				})
			}

			c.Set("user", &user)
			return next(c)
		}
	}
}

func AdminAuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get("user").(*models.User)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"code":    401,
					"message": "未授权访问",
				})
			}
			if user.Type != "admin" {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"code":    403,
					"message": "需要管理员权限",
				})
			}
			return next(c)
		}
	}
}

type RateLimitConfig struct {
	Limit   int                         // 限制次数
	Window  time.Duration               // 时间窗口
	KeyFunc func(c echo.Context) string // 自定义 Key 生成器
}

func NewRateLimitMiddleware(manager *limiter.Manager, config RateLimitConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 生成限流 Key
			key := config.KeyFunc(c)
			if key == "" {
				// 默认使用 IP
				key = c.RealIP()
			}
			// 加上前缀防止 Key 冲突
			redisKey := fmt.Sprintf("limiter:%s", key)
			// 调用工具类检查
			allowed, err := manager.Allow(c.Request().Context(), redisKey, config.Limit, config.Window)

			if err != nil {
				// Redis 报错，建议 Fail Open (放行)，避免 Redis 故障导致业务不可用
				c.Logger().Errorf("Rate limit redis error: %v", err)
				return next(c)
			}

			// 拒绝处理
			if !allowed {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"code": "429",
					"msg":  "Too Many Requests",
				})
			}
			return next(c)
		}
	}
}
