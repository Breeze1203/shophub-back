package middleware

import (
	"LiteAdmin/models"
	"LiteAdmin/services"
	"net/http"
	"strings"

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
				if strings.HasPrefix(tokenString, "Bearer ") {
					tokenString = strings.TrimPrefix(tokenString, "Bearer ")
				}
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
