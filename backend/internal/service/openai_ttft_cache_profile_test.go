package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type blockingOpenAITTFTCacheProfileStore struct {
	started   chan struct{}
	release   chan struct{}
	startOnce sync.Once
	mu        sync.Mutex
	calls     int
}

type recordingOpenAITTFTCacheProfileStore struct {
	written chan OpenAITTFTCacheProfileObservation
}

func (s *recordingOpenAITTFTCacheProfileStore) GetOpenAITTFTCacheState(context.Context, OpenAITTFTCacheProfileKey) (OpenAITTFTCacheState, error) {
	return OpenAITTFTCacheState{}, nil
}

func (s *recordingOpenAITTFTCacheProfileStore) PutOpenAITTFTCacheProfile(_ context.Context, key OpenAITTFTCacheProfileKey, profile OpenAITTFTCacheProfile, _ time.Duration) error {
	s.written <- OpenAITTFTCacheProfileObservation{Key: key, Profile: profile}
	return nil
}

func (s *recordingOpenAITTFTCacheProfileStore) PutOpenAITTFTSwitchDebounce(context.Context, OpenAITTFTSwitchDebounceKey, OpenAITTFTSwitchDebounce, time.Duration) error {
	return nil
}

type recordingOpenAITTFTCacheGateway struct {
	GatewayCache
	*recordingOpenAITTFTCacheProfileStore
}

func (s *blockingOpenAITTFTCacheProfileStore) GetOpenAITTFTCacheState(context.Context, OpenAITTFTCacheProfileKey) (OpenAITTFTCacheState, error) {
	return OpenAITTFTCacheState{}, nil
}

func (s *blockingOpenAITTFTCacheProfileStore) PutOpenAITTFTCacheProfile(ctx context.Context, _ OpenAITTFTCacheProfileKey, _ OpenAITTFTCacheProfile, _ time.Duration) error {
	s.startOnce.Do(func() { close(s.started) })
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *blockingOpenAITTFTCacheProfileStore) PutOpenAITTFTSwitchDebounce(context.Context, OpenAITTFTSwitchDebounceKey, OpenAITTFTSwitchDebounce, time.Duration) error {
	return nil
}

func (s *blockingOpenAITTFTCacheProfileStore) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestOpenAITTFTCacheProfileWriterIsBoundedWhenRedisStalls(t *testing.T) {
	store := &blockingOpenAITTFTCacheProfileStore{started: make(chan struct{}), release: make(chan struct{})}
	writer := newOpenAITTFTCacheProfileWriter(store, 1)
	observation := OpenAITTFTCacheProfileObservation{
		Key:     OpenAITTFTCacheProfileKey{GroupID: 7, SessionHash: "session", AccountID: 9},
		Profile: OpenAITTFTCacheProfile{ObservedAt: time.Now().UTC(), TotalContextTokens: 100_000, CacheReadTokens: 80_000},
	}
	require.True(t, writer.Enqueue(observation))
	select {
	case <-store.started:
	case <-time.After(time.Second):
		t.Fatal("cache profile writer did not start")
	}
	require.True(t, writer.Enqueue(observation))
	require.False(t, writer.Enqueue(observation))
	require.Equal(t, uint64(1), writer.Dropped())

	close(store.release)
	require.Eventually(t, func() bool { return store.callCount() >= 2 }, time.Second, 10*time.Millisecond)
}

func TestBuildOpenAITTFTCacheProfileObservationUsesNormalizedUsageBuckets(t *testing.T) {
	groupID := int64(7)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	input := &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			Stream: true,
			Usage:  OpenAIUsage{InputTokens: 100_000, CacheReadInputTokens: 80_000, CacheCreationInputTokens: 10_000},
		},
		APIKey:      &APIKey{GroupID: &groupID},
		Account:     &Account{ID: 9, Platform: PlatformOpenAI},
		SessionHash: "session-hash",
	}

	got, ok := buildOpenAITTFTCacheProfileObservation(input, now)
	require.True(t, ok)
	require.Equal(t, OpenAITTFTCacheProfileKey{GroupID: groupID, SessionHash: "session-hash", AccountID: 9}, got.Key)
	require.Equal(t, OpenAITTFTCacheProfile{ObservedAt: now, TotalContextTokens: 100_000, CacheReadTokens: 80_000, CacheWriteTokens: 10_000}, got.Profile)
}

func TestBuildOpenAITTFTCacheProfileObservationRejectsUnsafeRequests(t *testing.T) {
	groupID := int64(7)
	valid := func() *OpenAIRecordUsageInput {
		return &OpenAIRecordUsageInput{
			Result:      &OpenAIForwardResult{Stream: true, Usage: OpenAIUsage{InputTokens: 100_000, CacheReadInputTokens: 80_000}},
			APIKey:      &APIKey{GroupID: &groupID},
			Account:     &Account{ID: 9, Platform: PlatformOpenAI},
			SessionHash: "session-hash",
		}
	}
	for _, tc := range []struct {
		name   string
		mutate func(*OpenAIRecordUsageInput)
	}{
		{name: "non streaming", mutate: func(in *OpenAIRecordUsageInput) { in.Result.Stream = false }},
		{name: "client disconnected", mutate: func(in *OpenAIRecordUsageInput) { in.Result.ClientDisconnect = true }},
		{name: "cyber request", mutate: func(in *OpenAIRecordUsageInput) { in.CyberBlocked = true }},
		{name: "other platform", mutate: func(in *OpenAIRecordUsageInput) { in.Account.Platform = PlatformAnthropic }},
		{name: "missing group", mutate: func(in *OpenAIRecordUsageInput) { in.APIKey.GroupID = nil }},
		{name: "missing session", mutate: func(in *OpenAIRecordUsageInput) { in.SessionHash = "" }},
		{name: "inconsistent usage", mutate: func(in *OpenAIRecordUsageInput) {
			in.Result.Usage.CacheReadInputTokens = 90_000
			in.Result.Usage.CacheCreationInputTokens = 20_000
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := valid()
			tc.mutate(input)
			_, ok := buildOpenAITTFTCacheProfileObservation(input, time.Now().UTC())
			require.False(t, ok)
		})
	}
}

func TestRecordOpenAITTFTCacheProfileEnqueuesSuccessfulStream(t *testing.T) {
	resetOpenAITTFTRuntimeSettingsCacheForTest()
	t.Cleanup(resetOpenAITTFTRuntimeSettingsCacheForTest)
	groupID := int64(7)
	store := &recordingOpenAITTFTCacheProfileStore{written: make(chan OpenAITTFTCacheProfileObservation, 1)}
	svc := &OpenAIGatewayService{cache: &recordingOpenAITTFTCacheGateway{recordingOpenAITTFTCacheProfileStore: store}}
	input := &OpenAIRecordUsageInput{
		Result:      &OpenAIForwardResult{Stream: true, Usage: OpenAIUsage{InputTokens: 100_000, CacheReadInputTokens: 80_000}},
		APIKey:      &APIKey{GroupID: &groupID},
		Account:     &Account{ID: 9, Platform: PlatformOpenAI},
		SessionHash: "session-hash",
	}

	svc.recordOpenAITTFTCacheProfile(input, time.Now().UTC())
	select {
	case got := <-store.written:
		require.Equal(t, int64(9), got.Key.AccountID)
		require.Equal(t, 80_000, got.Profile.CacheReadTokens)
	case <-time.After(time.Second):
		t.Fatal("cache profile was not enqueued")
	}
}

func TestRecordOpenAITTFTCacheProfileSkipsWhenOptimizerOrCacheProtectionIsDisabled(t *testing.T) {
	resetOpenAITTFTRuntimeSettingsCacheForTest()
	defer resetOpenAITTFTRuntimeSettingsCacheForTest()
	groupID := int64(7)
	input := &OpenAIRecordUsageInput{
		Account:     &Account{ID: 9, Platform: PlatformOpenAI},
		APIKey:      &APIKey{GroupID: &groupID},
		SessionHash: "session",
		Result: &OpenAIForwardResult{
			Stream: true,
			Usage:  OpenAIUsage{InputTokens: 100_000, CacheReadInputTokens: 80_000},
		},
	}
	store := &recordingOpenAITTFTCacheProfileStore{written: make(chan OpenAITTFTCacheProfileObservation, 1)}
	svc := &OpenAIGatewayService{cache: &recordingOpenAITTFTCacheGateway{recordingOpenAITTFTCacheProfileStore: store}}
	svc.openAITTFTCacheWriter = newOpenAITTFTCacheProfileWriter(store, 1)
	svc.openAITTFTCacheWriterOnce.Do(func() {})

	storeOpenAITTFTRuntimeSettings(openAITTFTRuntimeSettings{Enabled: false, BaseP90Ms: 10_000, CacheProtectionEnabled: true, MinContextTokens: 100_000, MinHitRatePercent: 80, ElasticP90CapMs: 30_000}, time.Minute)
	svc.recordOpenAITTFTCacheProfile(input, time.Now().UTC())

	select {
	case <-store.written:
		t.Fatal("disabled optimizer must not enqueue cache profile")
	case <-time.After(20 * time.Millisecond):
	}

	storeOpenAITTFTRuntimeSettings(openAITTFTRuntimeSettings{Enabled: true, BaseP90Ms: 10_000, CacheProtectionEnabled: false, MinContextTokens: 100_000, MinHitRatePercent: 80, ElasticP90CapMs: 30_000}, time.Minute)
	svc.recordOpenAITTFTCacheProfile(input, time.Now().UTC())
	select {
	case <-store.written:
		t.Fatal("disabled cache protection must not enqueue cache profile")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestEvaluateOpenAITTFTCacheProfileUsesElasticP90Boundaries(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	policy := OpenAITTFTCachePolicy{
		Enabled:           true,
		BaseP90Ms:         10_000,
		MinContextTokens:  100_000,
		MinHitRatePercent: 80,
		ElasticP90CapMs:   30_000,
	}
	for _, tc := range []struct {
		name  string
		total int
		read  int
		want  int
	}{
		{name: "below threshold", total: 99_999, read: 90_000, want: 10_000},
		{name: "first step", total: 100_000, read: 80_000, want: 15_000},
		{name: "before second step", total: 149_999, read: 120_000, want: 15_000},
		{name: "second step", total: 150_000, read: 120_000, want: 20_000},
		{name: "third step", total: 200_000, read: 160_000, want: 25_000},
		{name: "capped", total: 250_000, read: 200_000, want: 30_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateOpenAITTFTCacheProfile(policy, OpenAITTFTCacheProfile{
				ObservedAt:         now,
				TotalContextTokens: tc.total,
				CacheReadTokens:    tc.read,
			}, now)
			require.Equal(t, tc.want, got.EffectiveP90Ms)
			require.Equal(t, tc.total, got.TotalContextTokens)
		})
	}
}

func TestEvaluateOpenAITTFTCacheProfileRequiresFreshHighHitImage(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	policy := OpenAITTFTCachePolicy{
		Enabled:           true,
		BaseP90Ms:         10_000,
		MinContextTokens:  100_000,
		MinHitRatePercent: 80,
		ElasticP90CapMs:   30_000,
	}
	for _, tc := range []struct {
		name    string
		profile OpenAITTFTCacheProfile
		want    string
	}{
		{name: "below hit rate", profile: OpenAITTFTCacheProfile{ObservedAt: now, TotalContextTokens: 100_000, CacheReadTokens: 79_999}, want: "low_hit_rate"},
		{name: "expired", profile: OpenAITTFTCacheProfile{ObservedAt: now.Add(-11 * time.Minute), TotalContextTokens: 100_000, CacheReadTokens: 90_000}, want: "expired"},
		{name: "inconsistent buckets", profile: OpenAITTFTCacheProfile{ObservedAt: now, TotalContextTokens: 100_000, CacheReadTokens: 80_000, CacheWriteTokens: 30_000}, want: "invalid"},
		{name: "missing", profile: OpenAITTFTCacheProfile{}, want: "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateOpenAITTFTCacheProfile(policy, tc.profile, now)
			require.False(t, got.Eligible)
			require.Equal(t, tc.want, got.Status)
			require.Equal(t, policy.BaseP90Ms, got.EffectiveP90Ms)
		})
	}
}

func TestEvaluateOpenAITTFTCacheProfileHonorsDisabledProtectionAndHitBoundary(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	base := OpenAITTFTCachePolicy{Enabled: true, BaseP90Ms: 10_000, MinContextTokens: 100_000, MinHitRatePercent: 80, ElasticP90CapMs: 30_000}
	eligible := evaluateOpenAITTFTCacheProfile(base, OpenAITTFTCacheProfile{ObservedAt: now, TotalContextTokens: 100_000, CacheReadTokens: 80_000}, now)
	require.True(t, eligible.Eligible)
	require.Equal(t, "eligible", eligible.Status)

	disabled := base
	disabled.Enabled = false
	got := evaluateOpenAITTFTCacheProfile(disabled, OpenAITTFTCacheProfile{ObservedAt: now, TotalContextTokens: 100_000, CacheReadTokens: 100_000}, now)
	require.False(t, got.Eligible)
	require.Equal(t, "disabled", got.Status)
}
