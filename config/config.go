package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Database DatabaseConfig `json:"database"`
	Auth     AuthConfig     `json:"auth"`
}

type DatabaseConfig struct {
	DSN string `json:"dsn"`
}

type OAuthProvider struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
	AuthURL      string   `json:"auth_url"`  // For custom OAuth providers
	TokenURL     string   `json:"token_url"` // For custom OAuth providers
}

type AuthConfig struct {
	JWTSecret     string `json:"jwt_secret"`
	TokenExpiry   int    `json:"token_expiry"`   // in hours
	RefreshExpiry int    `json:"refresh_expiry"` // in hours
	OAuth         struct {
		Google   OAuthProvider            `json:"google"`
		GitHub   OAuthProvider            `json:"github"`
		Facebook OAuthProvider            `json:"facebook"`
		Wechat   OAuthProvider            `json:"wechat"`
		Custom   map[string]OAuthProvider `json:"custom"`
	} `json:"oauth"`
}

func LoadConfig() (config Config, err error) {
	file, err := os.Open("config/config.json")
	if err != nil {
		return config, err
	}
	defer func(file *os.File) {
		closeErr := file.Close()
		if closeErr != nil {
			log.Printf("Error closing config file: %v", closeErr)
		}
	}(file)
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}
