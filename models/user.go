package models

import "time"

type User struct {
	ID           uint          `json:"id" gorm:"primaryKey"`
	Email        string        `json:"email" gorm:"uniqueIndex"`
	Username     string        `json:"username" gorm:"uniqueIndex"`
	Password     string        `json:"-"`        // For local auth, hashed
	Provider     string        `json:"provider"` // google, github, facebook, local, custom
	ProviderID   string        `json:"provider_id"`
	Type         string        `json:"type"` // admin，merchant(商家),client(客户)
	Avatar       string        `json:"avatar"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	MerchantInfo *MerchantInfo `gorm:"foreignKey:UserID" json:"merchant_info,omitempty"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
}
