package models

import "time"

type CustomerSession struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"index"`
	RoomID      uint      `json:"room_id" gorm:"uniqueIndex"`
	Status      string    `json:"status" gorm:"default:'pending'"` // pending, active, closed
	LastMessage string    `json:"last_message"`
	UnreadCount int       `json:"unread_count" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// 关联
	User User `json:"user" gorm:"foreignKey:UserID"`
}
