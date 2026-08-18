package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	openAITTFTCacheProfilePrefix  = "openai:ttft:cache-profile:"
	openAITTFTCacheDebouncePrefix = "openai:ttft:cache-debounce:"
)

var openAITTFTCacheProfileWriteScript = redis.NewScript(`
local key = KEYS[1]
local observed = tonumber(ARGV[1])
local current = redis.call('HGET', key, 'observed_at_ms')
if current and tonumber(current) > observed then
  return 0
end
redis.call('HSET', key,
  'observed_at_ms', ARGV[1],
  'total_context_tokens', ARGV[2],
  'cache_read_tokens', ARGV[3],
  'cache_write_tokens', ARGV[4])
redis.call('EXPIRE', key, ARGV[5])
return 1
`)

func NewOpenAITTFTCacheProfileStore(rdb *redis.Client) service.OpenAITTFTCacheProfileStore {
	return &gatewayCache{rdb: rdb}
}

var _ service.OpenAITTFTCacheProfileStore = (*gatewayCache)(nil)

func openAITTFTCacheSessionPart(sessionHash string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(sessionHash)))
	return fmt.Sprintf("%x", digest[:])
}

func openAITTFTCacheProfileKey(key service.OpenAITTFTCacheProfileKey) string {
	return fmt.Sprintf("%s%d:%s:%d", openAITTFTCacheProfilePrefix, key.GroupID, openAITTFTCacheSessionPart(key.SessionHash), key.AccountID)
}

func openAITTFTCacheDebounceKey(key service.OpenAITTFTSwitchDebounceKey) string {
	return fmt.Sprintf("%s%d:%s", openAITTFTCacheDebouncePrefix, key.GroupID, openAITTFTCacheSessionPart(key.SessionHash))
}

func (c *gatewayCache) GetOpenAITTFTCacheState(ctx context.Context, key service.OpenAITTFTCacheProfileKey) (service.OpenAITTFTCacheState, error) {
	state := service.OpenAITTFTCacheState{}
	if c == nil || c.rdb == nil {
		return state, fmt.Errorf("openai ttft cache profile redis is unavailable")
	}
	if key.AccountID <= 0 || strings.TrimSpace(key.SessionHash) == "" {
		return state, fmt.Errorf("invalid openai ttft cache profile key")
	}
	pipe := c.rdb.Pipeline()
	profileCmd := pipe.HGetAll(ctx, openAITTFTCacheProfileKey(key))
	debounceCmd := pipe.Get(ctx, openAITTFTCacheDebounceKey(service.OpenAITTFTSwitchDebounceKey{GroupID: key.GroupID, SessionHash: key.SessionHash}))
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return state, err
	}
	profileValues, err := profileCmd.Result()
	if err != nil && err != redis.Nil {
		return state, err
	}
	if observed, parseErr := strconv.ParseInt(profileValues["observed_at_ms"], 10, 64); parseErr == nil && observed > 0 {
		total, totalErr := strconv.Atoi(profileValues["total_context_tokens"])
		read, readErr := strconv.Atoi(profileValues["cache_read_tokens"])
		written, writeErr := strconv.Atoi(profileValues["cache_write_tokens"])
		if totalErr == nil && readErr == nil && writeErr == nil {
			state.Profile = service.OpenAITTFTCacheProfile{ObservedAt: time.UnixMilli(observed), TotalContextTokens: total, CacheReadTokens: read, CacheWriteTokens: written}
			state.HasImage = true
		}
	}
	debounceValue, debounceErr := debounceCmd.Result()
	if debounceErr == nil {
		parts := strings.Split(debounceValue, ":")
		if len(parts) == 3 {
			fromID, fromErr := strconv.ParseInt(parts[0], 10, 64)
			toID, toErr := strconv.ParseInt(parts[1], 10, 64)
			switchedAt, timeErr := strconv.ParseInt(parts[2], 10, 64)
			if fromErr == nil && toErr == nil && timeErr == nil && toID > 0 && switchedAt > 0 {
				state.Debounce = service.OpenAITTFTSwitchDebounce{FromAccountID: fromID, ToAccountID: toID, SwitchedAt: time.UnixMilli(switchedAt)}
				state.HasDebounce = true
			}
		}
	} else if debounceErr != redis.Nil {
		return state, debounceErr
	}
	return state, nil
}

func (c *gatewayCache) PutOpenAITTFTCacheProfile(ctx context.Context, key service.OpenAITTFTCacheProfileKey, profile service.OpenAITTFTCacheProfile, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("openai ttft cache profile redis is unavailable")
	}
	if key.AccountID <= 0 || strings.TrimSpace(key.SessionHash) == "" || profile.ObservedAt.IsZero() || profile.TotalContextTokens <= 0 || profile.CacheReadTokens < 0 || profile.CacheWriteTokens < 0 || profile.CacheReadTokens+profile.CacheWriteTokens > profile.TotalContextTokens || ttl <= 0 {
		return fmt.Errorf("invalid openai ttft cache profile")
	}
	_, err := openAITTFTCacheProfileWriteScript.Run(ctx, c.rdb, []string{openAITTFTCacheProfileKey(key)}, profile.ObservedAt.UnixMilli(), profile.TotalContextTokens, profile.CacheReadTokens, profile.CacheWriteTokens, int(ttl.Seconds())).Result()
	return err
}

func (c *gatewayCache) PutOpenAITTFTSwitchDebounce(ctx context.Context, key service.OpenAITTFTSwitchDebounceKey, debounce service.OpenAITTFTSwitchDebounce, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("openai ttft cache debounce redis is unavailable")
	}
	if strings.TrimSpace(key.SessionHash) == "" || debounce.ToAccountID <= 0 || debounce.SwitchedAt.IsZero() || ttl <= 0 {
		return fmt.Errorf("invalid openai ttft cache debounce")
	}
	value := fmt.Sprintf("%d:%d:%d", debounce.FromAccountID, debounce.ToAccountID, debounce.SwitchedAt.UnixMilli())
	return c.rdb.Set(ctx, openAITTFTCacheDebounceKey(key), value, ttl).Err()
}
