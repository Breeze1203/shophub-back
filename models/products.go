package models

import "time"

type Sort struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SortName  string    `json:"sort_name"`
	ParentID  uint      `json:"parent_id"`
	Icon      string    `json:"icon"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Product struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Name string `json:"name"`
}
