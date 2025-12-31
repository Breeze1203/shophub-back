package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)


// Strategy 定义限流算法策略接口
type Strategy interface {
	// Allow 检查是否允许通过
	// key: 限流标识 (如 IP)
	// limit: 限制次数 (或令牌桶容量)
	// window: 时间窗口 (或令牌生成速率单位)
	Allow(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) (bool, error)
}

// Manager 限流管理器
type Manager struct {
	rdb      *redis.Client
	strategy Strategy
}

func NewManager(rdb *redis.Client, strategy Strategy) *Manager {
	return &Manager{
		rdb:      rdb,
		strategy: strategy,
	}
}

// Allow 代理执行具体的策略
func (m *Manager) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	return m.strategy.Allow(ctx, m.rdb, key, limit, window)
}

// 固定窗口 (Fixed Window / Counter)
type FixedWindowStrategy struct{}

func (s *FixedWindowStrategy) Allow(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) (bool, error) {
	// Lua 脚本：原子性执行 INCR 和 EXPIRE
	const script = `
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		
		-- 自增
		local current = redis.call("INCR", key)
		
		-- 如果是第一次访问 (值为1)，设置过期时间
		if current == 1 then
			redis.call("EXPIRE", key, window)
		end
		
		-- 判断是否超限
		if current > limit then
			return 0 -- 拒绝
		end
		return 1 -- 允许
	`

	result, err := rdb.Eval(ctx, script, []string{key}, limit, int(window.Seconds())).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}


// 策略 2: 令牌桶 (Token Bucket)
type TokenBucketStrategy struct{}

func (s *TokenBucketStrategy) Allow(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) (bool, error) {
	// 简单版令牌桶 Lua 脚本
	// 逻辑：记录上次剩余令牌数和更新时间，请求来时根据时间差计算新生成的令牌
	// KEYS[1]: 存储令牌信息的 hash key
	// ARGV[1]: 桶容量 (burst/limit)
	// ARGV[2]: 令牌生成速率 (rate: token/second)
	// ARGV[3]: 当前时间戳 (秒)
	const script = `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		-- 获取当前桶内令牌数和上次刷新时间
		local info = redis.call("HMGET", key, "tokens", "last_time")
		local tokens = tonumber(info[1])
		local last_time = tonumber(info[2])
		
		-- 初始化
		if tokens == nil then
			tokens = capacity
			last_time = now
		end
		
		-- 计算时间差内生成的令牌
		local delta = math.max(0, now - last_time)
		local generated = delta * rate
		
		-- 更新令牌数 (不能超过容量)
		tokens = math.min(capacity, tokens + generated)
		
		-- 判断是否足够消费 1 个令牌
		if tokens >= 1 then
			tokens = tokens - 1
			-- 更新 Redis，设置较长的过期时间防止死数据
			redis.call("HMSET", key, "tokens", tokens, "last_time", now)
			redis.call("EXPIRE", key, 60) 
			return 1 -- 允许
		else
			-- 为了保证时间更新，即使拒绝也可以更新一下时间(可选)，这里简单处理不更新
			return 0 -- 拒绝
		end
	`

	// 计算速率：limit / window秒数。例如 limit=10, window=1s -> rate=10
	rate := float64(limit) / window.Seconds()
	// 必须保证 rate > 0
	if rate <= 0 {
		rate = 1
	}

	now := time.Now().Unix()
	result, err := rdb.Eval(ctx, script, []string{key}, limit, rate, now).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}