package models

import "time"

type Room struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`               // chat
	Privacy     string    `json:"privacy"`            // public, private, password,customer
	Password    string    `json:"password"`           // 密码不返回给前端
	Language    string    `json:"language,omitempty"` // 仅 code 类型有
	OwnerID     uint      `json:"owner_id"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RoomWithUser struct {
	Room
	OwnerName   string `json:"owner_name" gorm:"column:username"` // 明确告诉 GORM 映射 username 列
	OnlineUsers uint   `json:"online_users"`
}
