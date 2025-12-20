package services

import (
	"LiteAdmin/config"
	"LiteAdmin/models"
	"LiteAdmin/redis"
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrRoomNotFound      = errors.New("room not found")
	ErrAccessDenied      = errors.New("access denied")
	ErrPasswordRequired  = errors.New("password required")
	ErrIncorrectPassword = errors.New("incorrect password")
)

type CreateRoomDTO struct {
	Name        string `json:"name"        validate:"required,min=3,max=50"`
	Description string `json:"description" validate:"max=255"`                                // 假设描述是可选的，但有最大长度
	Type        string `json:"type"        validate:"required,oneof=chat game collaboration"` // 示例：类型必须是这三者之一
	Privacy     string `json:"privacy"     validate:"required,oneof=public password"`         // 隐私设置必须是 'public' 或 'password'
	Language    string `json:"language"    validate:"required"`
	Password    string `json:"password,omitempty" validate:"omitempty,required_if=Privacy password,min=6"`
}

type RoomService struct {
	db  *gorm.DB
	cfg *config.RedisConfig
}

func NewRoomService(db *gorm.DB, cfg *config.RedisConfig) *RoomService {
	return &RoomService{db: db}
}

func (s *RoomService) CreateRoom(inputRoom models.Room, user *models.User) (*models.Room, error) {
	var hashedPassword string
	if inputRoom.Privacy == "password" {
		if inputRoom.Password == "" {
			return nil, errors.New("password is required for private rooms")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(inputRoom.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashedPassword = string(hash)
	}
	room := models.Room{
		Name:        inputRoom.Name,
		Description: inputRoom.Description,
		Type:        inputRoom.Type,
		Privacy:     inputRoom.Privacy,
		Language:    inputRoom.Language,
		Password:    hashedPassword,
		OwnerID:     user.ID,
		IsActive:    true,
	}
	if err := s.db.Create(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *RoomService) ListRooms() ([]models.RoomWithUser, error) {
	var results []models.RoomWithUser
	err := s.db.Table("rooms").
		Select("rooms.*, users.username").
		Joins("LEFT JOIN users ON users.id = rooms.owner_id").
		Order("rooms.created_at DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(results); i++ {
		ctx := context.Background()
		users, redisErr := redis.GetRedis(s.cfg).GetOnlineUsers(ctx, results[i].Type, results[i].ID)
		if redisErr != nil {
			continue
		}
		results[i].OnlineUsers = uint(len(users))
		results[i].Password = ""
	}
	return results, nil
}

// AuthorizeRoomEntry 验证用户是否有权进入房间（例如通过密码）
// 业务逻辑：检查房间是否存在，并验证密码（如果需要）
func (s *RoomService) AuthorizeRoomEntry(roomID uint, password string) (*models.Room, error) {
	var room models.Room
	if err := s.db.First(&room, roomID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	if room.Privacy == "password" {
		if password == "" {
			return nil, ErrPasswordRequired
		}

		if err := bcrypt.CompareHashAndPassword([]byte(room.Password), []byte(password)); err != nil {
			return nil, ErrIncorrectPassword
		}
	}
	return &room, nil
}

// DeleteRoom 删除房间
// 业务逻辑：检查房间是否存在，并验证是否为房主
func (s *RoomService) DeleteRoom(roomID uint, user *models.User) error {
	var room models.Room
	// 必须用事务，确保“先查后删”的原子性
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&room, roomID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRoomNotFound
			}
			return err
		}

		if room.OwnerID != user.ID {
			return ErrAccessDenied
		}

		if err := tx.Delete(&room).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *RoomService) GetRoomByID(id uint) (models.RoomWithUser, error) {
	var results models.RoomWithUser
	err := s.db.Table("rooms").
		Select("rooms.*, users.username").
		Joins("LEFT JOIN users ON users.id = rooms.owner_id").
		Where("rooms.id = ?", id).
		Order("rooms.created_at DESC").
		Scan(&results).Error
	return results, err

}
