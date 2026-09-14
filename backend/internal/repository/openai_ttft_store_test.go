package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestOpenAITTFTStoreKeepsLatestTenSamplesAndSeparatesTransport(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewOpenAITTFTStore(rdb)
	now := time.Now().UTC().Truncate(time.Minute)
	ctx := context.Background()

	for i := 1; i <= 12; i++ {
		require.NoError(t, store.AddSample(ctx, 9, service.OpenAITTFTTransportHTTPSSE, now.Add(time.Duration(i)*time.Minute), i*100))
	}
	require.NoError(t, store.AddSample(ctx, 9, service.OpenAITTFTTransportResponsesWS, now.Add(time.Minute), 50))

	keySSE := service.OpenAITTFTWindowKey{AccountID: 9, Transport: service.OpenAITTFTTransportHTTPSSE}
	keyWS := service.OpenAITTFTWindowKey{AccountID: 9, Transport: service.OpenAITTFTTransportResponsesWS}
	windows, err := store.GetWindows(ctx, []service.OpenAITTFTWindowKey{keySSE, keyWS}, now.Add(13*time.Minute))
	require.NoError(t, err)
	require.Equal(t, 10, windows[keySSE].Count)
	require.Equal(t, 700, windows[keySSE].P50Ms)
	require.Equal(t, 50, windows[keyWS].P90Ms)
}

func TestOpenAITTFTStoreExplorationLeaseAndCooldownAreTokenSafe(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewOpenAITTFTStore(rdb)
	ctx := context.Background()
	key := service.OpenAITTFTWindowKey{AccountID: 11, Transport: service.OpenAITTFTTransportHTTPSSE}

	require.True(t, mustBeginExploration(t, store, ctx, key, "token-a"))
	require.False(t, mustBeginExploration(t, store, ctx, key, "token-b"))
	require.NoError(t, store.FinishExploration(ctx, key, "token-b", time.Minute))
	require.False(t, mustCoolingDown(t, store, ctx, key))
	require.NoError(t, store.FinishExploration(ctx, key, "token-a", time.Minute))
	require.True(t, mustCoolingDown(t, store, ctx, key))
	require.False(t, mustBeginExploration(t, store, ctx, key, "token-c"), "cooldown must block a new exploration lease")
}

func TestOpenAITTFTStoreExplorationQuotaNeverExceedsConfiguredPercent(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewOpenAITTFTStore(rdb)
	ctx := context.Background()
	allowed := 0
	for i := 0; i < 40; i++ {
		ok, err := store.TryAcquireExplorationQuota(ctx, "group:7:priority:1:http_sse", 5, 24*time.Hour)
		require.NoError(t, err)
		if ok {
			allowed++
		}
	}
	require.Equal(t, 2, allowed)
}

func TestOpenAITTFTCacheProfileStoreIsolatesKeysAndKeepsNewestObservation(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewOpenAITTFTCacheProfileStore(rdb)
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	key := service.OpenAITTFTCacheProfileKey{GroupID: 7, SessionHash: "session-hash", AccountID: 9}
	newer := service.OpenAITTFTCacheProfile{ObservedAt: now, TotalContextTokens: 200_000, CacheReadTokens: 180_000}
	older := service.OpenAITTFTCacheProfile{ObservedAt: now.Add(-time.Minute), TotalContextTokens: 100_000, CacheReadTokens: 80_000}

	require.NoError(t, store.PutOpenAITTFTCacheProfile(ctx, key, newer, 10*time.Minute))
	require.NoError(t, store.PutOpenAITTFTCacheProfile(ctx, key, older, 10*time.Minute))
	state, err := store.GetOpenAITTFTCacheState(ctx, key)
	require.NoError(t, err)
	require.True(t, state.HasImage)
	require.Equal(t, newer.TotalContextTokens, state.Profile.TotalContextTokens)

	other, err := store.GetOpenAITTFTCacheState(ctx, service.OpenAITTFTCacheProfileKey{GroupID: 7, SessionHash: "other-session", AccountID: 9})
	require.NoError(t, err)
	require.False(t, other.HasImage)

	mr.FastForward(11 * time.Minute)
	expired, err := store.GetOpenAITTFTCacheState(ctx, key)
	require.NoError(t, err)
	require.False(t, expired.HasImage)
}

func TestOpenAITTFTCacheProfileStoreKeepsSwitchDebounceIsolated(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewOpenAITTFTCacheProfileStore(rdb)
	ctx := context.Background()
	key := service.OpenAITTFTSwitchDebounceKey{GroupID: 7, SessionHash: "session-hash"}

	require.NoError(t, store.PutOpenAITTFTSwitchDebounce(ctx, key, service.OpenAITTFTSwitchDebounce{FromAccountID: 9, ToAccountID: 10, SwitchedAt: time.Now().UTC()}, 5*time.Minute))
	state, err := store.GetOpenAITTFTCacheState(ctx, service.OpenAITTFTCacheProfileKey{GroupID: 7, SessionHash: key.SessionHash, AccountID: 10})
	require.NoError(t, err)
	require.True(t, state.HasDebounce)
	require.Equal(t, int64(10), state.Debounce.ToAccountID)

	other, err := store.GetOpenAITTFTCacheState(ctx, service.OpenAITTFTCacheProfileKey{GroupID: 8, SessionHash: key.SessionHash, AccountID: 10})
	require.NoError(t, err)
	require.False(t, other.HasDebounce)

	mr.FastForward(6 * time.Minute)
	expired, err := store.GetOpenAITTFTCacheState(ctx, service.OpenAITTFTCacheProfileKey{GroupID: 7, SessionHash: key.SessionHash, AccountID: 10})
	require.NoError(t, err)
	require.False(t, expired.HasDebounce)
}

func TestListRecentOpenAITTFTSamplesUsesPerAccountTransportHydration(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	since := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	createdAt := since.Add(time.Hour)
	rows := sqlmock.NewRows([]string{"account_id", "transport", "first_token_ms", "created_at"}).
		AddRow(int64(7), "http_sse", 123, createdAt).
		AddRow(int64(7), "responses_ws", 456, createdAt)
	mock.ExpectQuery("AND first_token_ms > 0").
		WithArgs(since, 10).
		WillReturnRows(rows)

	samples, err := repo.ListRecentOpenAITTFTSamples(context.Background(), since, 10)
	require.NoError(t, err)
	require.Len(t, samples, 2)
	require.Equal(t, service.OpenAITTFTTransportHTTPSSE, samples[0].Transport)
	require.Equal(t, 456, samples[1].TTFTMs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func mustBeginExploration(t *testing.T, store service.OpenAITTFTStore, ctx context.Context, key service.OpenAITTFTWindowKey, token string) bool {
	ok, err := store.TryBeginExploration(ctx, key, token, time.Minute)
	require.NoError(t, err)
	return ok
}

func mustCoolingDown(t *testing.T, store service.OpenAITTFTStore, ctx context.Context, key service.OpenAITTFTWindowKey) bool {
	cooling, err := store.ExplorationCoolingDown(ctx, key)
	require.NoError(t, err)
	return cooling
}
