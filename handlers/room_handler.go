package handlers

import (
	"LiteAdmin/models"
	"LiteAdmin/services"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

type RoomHandler struct {
	roomService *services.RoomService
}

func NewRoomHandler(roomService *services.RoomService) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

func (h *RoomHandler) CreateRoom(c echo.Context) error {
	user := c.Get("user").(*models.User)
	var inputRoom models.Room
	if err := c.Bind(&inputRoom); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}
	room, err := h.roomService.CreateRoom(inputRoom, user)
	if err != nil {
		if err.Error() == "password is required for private rooms" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to create room",
		})
	}
	return c.JSON(http.StatusCreated, room)
}

// ListRooms 获取所有房间
func (h *RoomHandler) ListRooms(c echo.Context) error {
	rooms, err := h.roomService.ListRooms()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch rooms",
		})
	}

	return c.JSON(http.StatusOK, rooms)
}

// GetRoom 获取单个房间
func (h *RoomHandler) GetRoom(c echo.Context) error {
	roomIDStr := c.Param("id")
	// HTTP 职责：转换 param string 为 uint
	roomID64, err := strconv.ParseUint(roomIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid room ID"})
	}
	roomID := uint(roomID64)

	// 调用 Service
	room, err := h.roomService.GetRoomByID(roomID)

	// HTTP 职责：将 Service error 映射为 HTTP 状态码
	if err != nil {
		switch err {
		case services.ErrRoomNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		case services.ErrAccessDenied:
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch room"})
		}
	}

	return c.JSON(http.StatusOK, room)
}

// JoinRoom 加入房间（验证密码）
func (h *RoomHandler) JoinRoom(c echo.Context) error {
	// HTTP 职责：解析 Param
	roomIDStr := c.Param("id")
	roomID64, err := strconv.ParseUint(roomIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid room ID"})
	}
	roomID := uint(roomID64)

	// HTTP 职责：绑定 body
	var req struct {
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// 调用 Service
	room, err := h.roomService.AuthorizeRoomEntry(roomID, req.Password)

	// HTTP 职责：映射 error
	if err != nil {
		switch err {
		case services.ErrRoomNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		case services.ErrPasswordRequired:
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		case services.ErrIncorrectPassword:
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify room access"})
		}
	}

	// 成功（与你原先的返回一致）
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "access granted",
		"room_id": room.ID, // room.ID 现在是 uint
	})
}

// DeleteRoom 删除房间
func (h *RoomHandler) DeleteRoom(c echo.Context) error {
	// HTTP 职责：获取 user 和 param
	user := c.Get("user").(*models.User)
	roomIDStr := c.Param("id")

	// 转换 param
	roomID64, err := strconv.ParseUint(roomIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid room ID"})
	}
	roomID := uint(roomID64)

	// 调用 Service
	if err := h.roomService.DeleteRoom(roomID, user); err != nil {
		switch err {
		case services.ErrRoomNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		case services.ErrAccessDenied:
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete room"})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "room deleted",
	})
}
