package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis 版本的账号风控计数器存储。
//
// 该文件是 service 包内唯一允许 import redis 的账号风控文件
// （见 backend/.golangci.yml 的 depguard 例外，与 ops_* 系列同性质：
// 需要跨实例共享的窗口计数，无法放在进程内存里）。
// 其余风控逻辑只依赖 AccountRiskCounterStore 接口，便于替换与单测。

var accountRiskIncrScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
elseif redis.call('PTTL', KEYS[1]) < 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return current
`)

// redisAccountRiskStore 基于 Redis 的窗口计数实现（多实例共享）。
type redisAccountRiskStore struct {
	client *redis.Client
}

// NewRedisAccountRiskStore 创建 Redis 计数器存储。
// client 为 nil 时降级为进程内存实现（仅适用于单实例/测试）。
func NewRedisAccountRiskStore(client *redis.Client) AccountRiskCounterStore {
	if client == nil {
		return newMemoryAccountRiskStore()
	}
	return &redisAccountRiskStore{client: client}
}

func (s *redisAccountRiskStore) Incr(ctx context.Context, key string, window time.Duration) (int64, error) {
	if window <= 0 {
		window = time.Hour
	}
	value, err := accountRiskIncrScript.Run(ctx, s.client, []string{key}, window.Milliseconds()).Int64()
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (s *redisAccountRiskStore) Get(ctx context.Context, key string) (int64, error) {
	value, err := s.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (s *redisAccountRiskStore) Reset(ctx context.Context, key string) {
	if err := s.client.Del(ctx, key).Err(); err != nil {
		slog.Warn("account_risk.reset_counter_failed", "key", key, "error", err)
	}
}

func (s *redisAccountRiskStore) SetFlag(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return s.client.SetNX(ctx, key, "1", ttl).Result()
}

func (s *redisAccountRiskStore) HasFlag(ctx context.Context, key string) bool {
	value, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		slog.Warn("account_risk.read_flag_failed", "key", key, "error", err)
		return false
	}
	return value > 0
}
