package handlers

import (
	"LiteAdmin/models"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type CustomerServiceHandler struct {
	db *gorm.DB
}

func NewCustomerServiceHandler(db *gorm.DB) *CustomerServiceHandler {
	return &CustomerServiceHandler{db: db}
}

// 创建或获取客服会话
func (h *CustomerServiceHandler) CreateOrGetSession(c echo.Context) error {
	user := c.Get("user").(*models.User)
	var session models.CustomerSession
	// 查找会话并预加载 Room
	err := h.db.Where("user_id = ?", user.ID).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 创建房间
		room := models.Room{
			Name:        fmt.Sprintf("客户 %s", user.Username),
			Description: "客户服务",
			Privacy:     "customer",
			Type:        "chat",
		}
		if err := h.db.Create(&room).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to create room",
			})
		}

		// 创建新会话
		session = models.CustomerSession{
			UserID:    user.ID,
			RoomID:    room.ID,
			Status:    "pending",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := h.db.Create(&session).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to create session",
			})
		}
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "database error",
		})
	}

	// 如果会话是关闭状态,重新激活
	if session.Status == "closed" {
		session.Status = "pending"
		session.UpdatedAt = time.Now()
		h.db.Save(&session)
	}

	// 统一返回
	return c.JSON(http.StatusOK, map[string]interface{}{
		"session": session,
	})
}

// 获取所有客服会话列表(服务端使用)

func (h *CustomerServiceHandler) GetAllSessions(c echo.Context) error {
	status := c.QueryParam("status") // pending, active, closed
	var sessions []models.CustomerSession
	query := h.db.Preload("User").Order("updated_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&sessions).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch sessions",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// 更新会话状态
func (h *CustomerServiceHandler) UpdateSessionStatus(c echo.Context) error {
	sessionID := c.Param("sessionId")

	var req struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	var session models.CustomerSession
	if err := h.db.First(&session, sessionID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "session not found",
		})
	}

	session.Status = req.Status
	session.UpdatedAt = time.Now()

	if err := h.db.Save(&session).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to update session",
		})
	}

	return c.JSON(http.StatusOK, session)
}

// 更新会话最后一条消息(在 WebSocket 消息处理中调用)
func (h *CustomerServiceHandler) UpdateLastMessage(roomID string, content string) {
	var session models.CustomerSession
	if err := h.db.Where("room_id = ?", roomID).First(&session).Error; err != nil {
		return
	}

	session.LastMessage = content
	session.UpdatedAt = time.Now()

	// 如果状态是 pending,改为 active
	if session.Status == "pending" {
		session.Status = "active"
	}

	h.db.Save(&session)
}
