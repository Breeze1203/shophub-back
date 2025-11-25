package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

// RedisConfig 用于配置 Redis 连接
type RedisConfig struct {
	Addr     string // addr
	Password string // 密码
	DB       int    // 数据库编号
	PoolSize int    // 连接池大小
}

// NewRedisClient 初始化并返回一个新的 RedisClient 实例
func NewRedisClient(cfg *RedisConfig) (*RedisClient, error) {
	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password, // 密码，没有则留空
		DB:       cfg.DB,       // 数据库
		PoolSize: cfg.PoolSize, // 连接池大小
		// 可选：添加超时配置
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// PING 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis client connection test failed: %w", err)
	}

	return &RedisClient{
		Client: client,
	}, nil
}

// Redis客户端
func GetRedis() *RedisClient {
	cfg := RedisConfig{
		Addr:     "localhost:6379",
		Password: "123",
		DB:       0,
		PoolSize: 10,
	}
	client, err := NewRedisClient(&cfg)
	if err != nil {
		return nil
	}
	return client
}

// Close 关闭 Redis 连接
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

type HGetAllResult struct {
	Data   map[string]string `json:"data"`
	Error  string            `json:"error,omitempty"`
	Status int               `json:"status"`
}

// 用户信息结构（用于在线列表）
type UserInfo struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Color    string `json:"color"`
}

// GetOnlineUsers 获取指定房间的在线用户
func (r *RedisClient) GetOnlineUsers(ctx context.Context, roomType string, roomID uint) (map[string]string, error) {
	key := fmt.Sprintf("%s:room:%d:online_users", roomType, roomID)
	result, err := r.Client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch online users for key %s: %w", key, err)
	}
	return result, nil
}
