package repository

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	openAITTFTRedisPrefix       = "openai:ttft:window:"
	openAITTFTExplorationPrefix = "openai:ttft:exploration:"
	openAITTFTQuotaPrefix       = "openai:ttft:quota:"
	openAITTFTLeaseSuffix       = ":lease"
	openAITTFTCooldownSuffix    = ":cooldown"
)

var openAITTFTAddSampleScript = redis.NewScript(`
local key = KEYS[1]
local cutoff = tonumber(ARGV[1])
local score = tonumber(ARGV[2])
local member = ARGV[3]
local cap = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])
redis.call('ZREMRANGEBYSCORE', key, '-inf', cutoff)
redis.call('ZADD', key, score, member)
local count = redis.call('ZCARD', key)
if count > cap then
  redis.call('ZREMRANGEBYRANK', key, 0, count - cap - 1)
end
redis.call('EXPIRE', key, ttl)
return 1
`)

var openAITTFTFinishExplorationScript = redis.NewScript(`
local leaseKey = KEYS[1]
local cooldownKey = KEYS[2]
if redis.call('GET', leaseKey) ~= ARGV[1] then
  return 0
end
redis.call('DEL', leaseKey)
redis.call('SET', cooldownKey, '1', 'EX', ARGV[2])
return 1
`)

var openAITTFTBeginExplorationScript = redis.NewScript(`
local leaseKey = KEYS[1]
local cooldownKey = KEYS[2]
if redis.call('EXISTS', cooldownKey) == 1 then
  return 0
end
return redis.call('SET', leaseKey, ARGV[1], 'NX', 'EX', ARGV[2]) and 1 or 0
`)

var openAITTFTQuotaScript = redis.NewScript(`
local key = KEYS[1]
local percent = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])
local total = redis.call('HINCRBY', key, 'total', 1)
local explored = tonumber(redis.call('HGET', key, 'explored') or '0')
if (explored + 1) * 100 <= percent * total then
  redis.call('HINCRBY', key, 'explored', 1)
  redis.call('EXPIRE', key, ttl)
  return 1
end
redis.call('EXPIRE', key, ttl)
return 0
`)

func NewOpenAITTFTStore(rdb *redis.Client) service.OpenAITTFTStore {
	return &gatewayCache{rdb: rdb}
}

var _ service.OpenAITTFTStore = (*gatewayCache)(nil)

func openAITTFTWindowKey(key service.OpenAITTFTWindowKey) string {
	return fmt.Sprintf("%s%d:%s", openAITTFTRedisPrefix, key.AccountID, key.Transport)
}

func openAITTFTExplorationKey(key service.OpenAITTFTWindowKey) string {
	return fmt.Sprintf("%s%d:%s", openAITTFTExplorationPrefix, key.AccountID, key.Transport)
}

func openAITTFTQuotaKey(scope string) string {
	return openAITTFTQuotaPrefix + scope
}

func (c *gatewayCache) AddSample(ctx context.Context, accountID int64, transport service.OpenAITTFTTransport, observedAt time.Time, ttftMs int) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("openai ttft redis cache is unavailable")
	}
	if accountID <= 0 || !transport.Valid() || ttftMs <= 0 || observedAt.IsZero() {
		return fmt.Errorf("invalid openai ttft sample")
	}
	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour).UnixMilli()
	score := observedAt.UnixMilli()
	member := fmt.Sprintf("%d:%d:%d", score, ttftMs, time.Now().UnixNano())
	_, err := openAITTFTAddSampleScript.Run(ctx, c.rdb, []string{openAITTFTWindowKey(service.OpenAITTFTWindowKey{AccountID: accountID, Transport: transport})}, cutoff, score, member, 10, int((25 * time.Hour).Seconds())).Result()
	return err
}

func (c *gatewayCache) GetWindows(ctx context.Context, keys []service.OpenAITTFTWindowKey, now time.Time) (map[service.OpenAITTFTWindowKey]service.OpenAITTFTWindowSnapshot, error) {
	result := make(map[service.OpenAITTFTWindowKey]service.OpenAITTFTWindowSnapshot, len(keys))
	if c == nil || c.rdb == nil {
		return result, fmt.Errorf("openai ttft redis cache is unavailable")
	}
	if len(keys) == 0 {
		return result, nil
	}
	type pending struct {
		key service.OpenAITTFTWindowKey
		cmd *redis.StringSliceCmd
	}
	pipe := c.rdb.Pipeline()
	pendingCommands := make([]pending, 0, len(keys))
	seen := make(map[service.OpenAITTFTWindowKey]struct{}, len(keys))
	for _, key := range keys {
		if key.AccountID <= 0 || !key.Transport.Valid() {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		pendingCommands = append(pendingCommands, pending{key: key, cmd: pipe.ZRevRange(ctx, openAITTFTWindowKey(key), 0, 9)})
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	cutoff := now.Add(-24 * time.Hour)
	for _, item := range pendingCommands {
		members, err := item.cmd.Result()
		if err != nil {
			return nil, err
		}
		values := make([]int, 0, len(members))
		for _, member := range members {
			observedMs, ttftMs, ok := parseOpenAITTFTMember(member)
			if !ok || ttftMs <= 0 || time.UnixMilli(observedMs).Before(cutoff) {
				continue
			}
			values = append(values, ttftMs)
		}
		result[item.key] = snapshotOpenAITTFTValues(values)
	}
	return result, nil
}

func parseOpenAITTFTMember(member string) (int64, int, bool) {
	parts := strings.Split(member, ":")
	if len(parts) < 2 {
		return 0, 0, false
	}
	observedMs, err1 := strconv.ParseInt(parts[0], 10, 64)
	ttftMs, err2 := strconv.Atoi(parts[1])
	return observedMs, ttftMs, err1 == nil && err2 == nil
}

func snapshotOpenAITTFTValues(values []int) service.OpenAITTFTWindowSnapshot {
	if len(values) == 0 {
		return service.OpenAITTFTWindowSnapshot{}
	}
	// Values arrive newest-first from Redis; percentile ranking is independent
	// of arrival order, so sort a small bounded copy in place.
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
	nearestRank := func(percentile float64) int {
		rank := int(math.Ceil(percentile * float64(len(values))))
		if rank < 1 {
			rank = 1
		}
		if rank > len(values) {
			rank = len(values)
		}
		return values[rank-1]
	}
	return service.OpenAITTFTWindowSnapshot{Count: len(values), P50Ms: nearestRank(0.50), P90Ms: nearestRank(0.90)}
}

func (c *gatewayCache) TryBeginExploration(ctx context.Context, key service.OpenAITTFTWindowKey, token string, ttl time.Duration) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, fmt.Errorf("openai ttft redis cache is unavailable")
	}
	if key.AccountID <= 0 || !key.Transport.Valid() || strings.TrimSpace(token) == "" || ttl <= 0 {
		return false, fmt.Errorf("invalid openai ttft exploration lease")
	}
	result, err := openAITTFTBeginExplorationScript.Run(ctx, c.rdb, []string{
		openAITTFTExplorationKey(key) + openAITTFTLeaseSuffix,
		openAITTFTExplorationKey(key) + openAITTFTCooldownSuffix,
	}, token, int(ttl.Seconds())).Int()
	return result == 1, err
}

func (c *gatewayCache) FinishExploration(ctx context.Context, key service.OpenAITTFTWindowKey, token string, cooldown time.Duration) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("openai ttft redis cache is unavailable")
	}
	if key.AccountID <= 0 || !key.Transport.Valid() || strings.TrimSpace(token) == "" || cooldown <= 0 {
		return fmt.Errorf("invalid openai ttft exploration cooldown")
	}
	_, err := openAITTFTFinishExplorationScript.Run(ctx, c.rdb, []string{
		openAITTFTExplorationKey(key) + openAITTFTLeaseSuffix,
		openAITTFTExplorationKey(key) + openAITTFTCooldownSuffix,
	}, token, int(cooldown.Seconds())).Result()
	return err
}

func (c *gatewayCache) ExplorationCoolingDown(ctx context.Context, key service.OpenAITTFTWindowKey) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, fmt.Errorf("openai ttft redis cache is unavailable")
	}
	if key.AccountID <= 0 || !key.Transport.Valid() {
		return false, fmt.Errorf("invalid openai ttft exploration key")
	}
	n, err := c.rdb.Exists(ctx, openAITTFTExplorationKey(key)+openAITTFTCooldownSuffix).Result()
	return n > 0, err
}

func (c *gatewayCache) TryAcquireExplorationQuota(ctx context.Context, scope string, percent int, ttl time.Duration) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, fmt.Errorf("openai ttft redis cache is unavailable")
	}
	if strings.TrimSpace(scope) == "" || percent <= 0 || percent > 5 || ttl <= 0 {
		return false, fmt.Errorf("invalid openai ttft exploration quota")
	}
	result, err := openAITTFTQuotaScript.Run(ctx, c.rdb, []string{openAITTFTQuotaKey(scope)}, percent, int(ttl.Seconds())).Int()
	return result == 1, err
}
