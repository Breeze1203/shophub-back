package services

import (
	"LiteAdmin/config"
	"LiteAdmin/models"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	Db            *gorm.DB
	jwtSecret     []byte
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
}

func NewAuthService(db *gorm.DB, config *config.AuthConfig) *AuthService {
	return &AuthService{
		Db:            db,
		jwtSecret:     []byte(config.JWTSecret),
		tokenExpiry:   time.Duration(config.TokenExpiry) * time.Hour,
		refreshExpiry: time.Duration(config.RefreshExpiry) * time.Hour,
	}
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (s *AuthService) GenerateTokens(user *models.User) (*models.AuthResponse, error) {
	// Access Token
	accessClaims := &Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	// Refresh Token
	refreshClaims := &Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.tokenExpiry.Seconds()),
		User:         *user,
	}, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *AuthService) RegisterLocal(email, username, password string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:    email,
		Username: username,
		Password: string(hashedPassword),
		Provider: "local",
	}

	if err := s.Db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) LoginLocal(email, password string) (*models.User, error) {
	var user models.User
	if err := s.Db.Where("email = ? AND provider = ?", email, "local").First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &user, nil
}

func (s *AuthService) FindOrCreateOAuthUser(userInfo *OAuthUserInfo) (*models.User, error) {
	var user models.User

	// Try to find existing user
	err := s.Db.Where("provider = ? AND provider_id = ?", userInfo.Provider, userInfo.ID).First(&user).Error

	if err == nil {
		// User exists, update info
		user.Email = userInfo.Email
		user.Avatar = userInfo.Avatar
		s.Db.Save(&user)
		return &user, nil
	}

	// Create new user
	user = models.User{
		Email:      userInfo.Email,
		Username:   userInfo.Name,
		Provider:   userInfo.Provider,
		ProviderID: userInfo.ID,
		Avatar:     userInfo.Avatar,
	}

	if err := s.Db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
