package models

import "time"

type Message struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RoomID    string    `json:"room_id"`
	UserID    uint      `json:"user_id"`
	Content   string    `json:"content" gorm:"type:text"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username" gorm:"-"`
	UserColor string    `json:"user_color" gorm:"-"`
}
