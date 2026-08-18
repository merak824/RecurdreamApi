package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type openAITTFTCacheStateStoreStub struct {
	state         OpenAITTFTCacheState
	err           error
	debounceKeys  []OpenAITTFTSwitchDebounceKey
	debounces     []OpenAITTFTSwitchDebounce
	debounceTTLs  []time.Duration
	debounceError error
}

func (s *openAITTFTCacheStateStoreStub) GetOpenAITTFTCacheState(context.Context, OpenAITTFTCacheProfileKey) (OpenAITTFTCacheState, error) {
	return s.state, s.err
}

func (s *openAITTFTCacheStateStoreStub) PutOpenAITTFTCacheProfile(context.Context, OpenAITTFTCacheProfileKey, OpenAITTFTCacheProfile, time.Duration) error {
	return nil
}

func (s *openAITTFTCacheStateStoreStub) PutOpenAITTFTSwitchDebounce(_ context.Context, key OpenAITTFTSwitchDebounceKey, debounce OpenAITTFTSwitchDebounce, ttl time.Duration) error {
	s.debounceKeys = append(s.debounceKeys, key)
	s.debounces = append(s.debounces, debounce)
	s.debounceTTLs = append(s.debounceTTLs, ttl)
	return s.debounceError
}

type openAITTFTCacheStateGatewayStub struct {
	GatewayCache
	*openAITTFTCacheStateStoreStub
}

func newOpenAITTFTStickyEvaluationService(store *openAITTFTCacheStateStoreStub, accountID int64, ttftMs int) *OpenAIGatewayService {
	cfg := &config.Config{}
	cfg.Gateway.OpenAITTFTOptimizerEnabled = true
	cfg.Gateway.OpenAITTFTOptimizerRolloutPercent = 100
	cfg.Gateway.OpenAITTFTStableThresholdMs = 10_000
	svc := &OpenAIGatewayService{cfg: cfg, openAITTFTWindow: newOpenAITTFTLocalWindow()}
	if store != nil {
		svc.cache = &openAITTFTCacheStateGatewayStub{openAITTFTCacheStateStoreStub: store}
	}
	now := time.Now().UTC()
	for i := 0; i < openAITTFTWindowSampleCap; i++ {
		svc.openAITTFTWindow.AddSample(OpenAITTFTSample{AccountID: accountID, Transport: OpenAITTFTTransportHTTPSSE, ObservedAt: now.Add(-time.Duration(i) * time.Second), TTFTMs: ttftMs})
	}
	return svc
}

func requireDefaultOpenAIAccountScheduler(t *testing.T, svc *OpenAIGatewayService) *defaultOpenAIAccountScheduler {
	t.Helper()
	scheduler, ok := newDefaultOpenAIAccountScheduler(svc, nil).(*defaultOpenAIAccountScheduler)
	require.True(t, ok)
	return scheduler
}

func TestEvaluateOpenAITTFTStickyUsesElasticCacheThreshold(t *testing.T) {
	const accountID int64 = 9
	now := time.Now().UTC()
	store := &openAITTFTCacheStateStoreStub{state: OpenAITTFTCacheState{
		HasImage: true,
		Profile:  OpenAITTFTCacheProfile{ObservedAt: now, TotalContextTokens: 100_000, CacheReadTokens: 80_000},
	}}
	svc := newOpenAITTFTStickyEvaluationService(store, accountID, 12_000)
	scheduler := requireDefaultOpenAIAccountScheduler(t, svc)
	groupID := int64(7)

	got := scheduler.evaluateOpenAITTFTSticky(context.Background(), OpenAIAccountScheduleRequest{GroupID: &groupID, Platform: PlatformOpenAI, SessionHash: "session", Streaming: true}, accountID, now)
	require.False(t, got.ShouldEscape)
	require.True(t, got.CacheEligible)
	require.Equal(t, 15_000, got.EffectiveP90Ms)
	require.Equal(t, 12_000, got.SampleP90Ms)
}

func TestEvaluateOpenAITTFTStickyEscapesAboveElasticThresholdAndFallsBackOnRedisError(t *testing.T) {
	const accountID int64 = 9
	now := time.Now().UTC()
	groupID := int64(7)
	req := OpenAIAccountScheduleRequest{GroupID: &groupID, Platform: PlatformOpenAI, SessionHash: "session", Streaming: true}

	eligibleStore := &openAITTFTCacheStateStoreStub{state: OpenAITTFTCacheState{HasImage: true, Profile: OpenAITTFTCacheProfile{ObservedAt: now, TotalContextTokens: 100_000, CacheReadTokens: 80_000}}}
	eligibleScheduler := requireDefaultOpenAIAccountScheduler(t, newOpenAITTFTStickyEvaluationService(eligibleStore, accountID, 16_000))
	eligible := eligibleScheduler.evaluateOpenAITTFTSticky(context.Background(), req, accountID, now)
	require.True(t, eligible.ShouldEscape)
	require.Equal(t, 15_000, eligible.EffectiveP90Ms)

	errorStore := &openAITTFTCacheStateStoreStub{err: errors.New("redis unavailable")}
	errorScheduler := requireDefaultOpenAIAccountScheduler(t, newOpenAITTFTStickyEvaluationService(errorStore, accountID, 12_000))
	fallback := errorScheduler.evaluateOpenAITTFTSticky(context.Background(), req, accountID, now)
	require.True(t, fallback.ShouldEscape)
	require.Equal(t, "read_error", fallback.CacheProfileStatus)
	require.Equal(t, 10_000, fallback.EffectiveP90Ms)
}

func TestEvaluateOpenAITTFTStickyDebounceKeepsCurrentAccount(t *testing.T) {
	const accountID int64 = 9
	now := time.Now().UTC()
	store := &openAITTFTCacheStateStoreStub{state: OpenAITTFTCacheState{
		HasDebounce: true,
		Debounce:    OpenAITTFTSwitchDebounce{FromAccountID: 8, ToAccountID: accountID, SwitchedAt: now.Add(-time.Minute)},
	}}
	svc := newOpenAITTFTStickyEvaluationService(store, accountID, 20_000)
	scheduler := requireDefaultOpenAIAccountScheduler(t, svc)
	groupID := int64(7)

	got := scheduler.evaluateOpenAITTFTSticky(context.Background(), OpenAIAccountScheduleRequest{GroupID: &groupID, Platform: PlatformOpenAI, SessionHash: "session", Streaming: true}, accountID, now)
	require.False(t, got.ShouldEscape)
	require.True(t, got.DebounceKept)
	require.Equal(t, 10_000, got.EffectiveP90Ms)
}

func TestRecordOpenAITTFTSwitchDebounceWritesOnlyForSuccessfulTTFTSwitch(t *testing.T) {
	store := &openAITTFTCacheStateStoreStub{}
	svc := newOpenAITTFTStickyEvaluationService(store, 9, 20_000)
	scheduler := requireDefaultOpenAIAccountScheduler(t, svc)
	groupID := int64(7)
	req := OpenAIAccountScheduleRequest{GroupID: &groupID, SessionHash: "session"}
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

	scheduler.recordOpenAITTFTSwitchDebounce(context.Background(), req, openAIStickyEscapeOutcome{
		Escaped:       true,
		Reason:        openAIStickyEscapeReasonTTFTP90,
		FromAccountID: 9,
	}, 10, now)

	require.Equal(t, []OpenAITTFTSwitchDebounceKey{{GroupID: groupID, SessionHash: "session"}}, store.debounceKeys)
	require.Equal(t, []OpenAITTFTSwitchDebounce{{FromAccountID: 9, ToAccountID: 10, SwitchedAt: now}}, store.debounces)
	require.Equal(t, []time.Duration{openAITTFTCacheSwitchDebounceTTL}, store.debounceTTLs)

	scheduler.recordOpenAITTFTSwitchDebounce(context.Background(), req, openAIStickyEscapeOutcome{
		Escaped:       true,
		Reason:        openAIStickyEscapeReasonTTFTP90,
		FromAccountID: 9,
	}, 9, now)
	scheduler.recordOpenAITTFTSwitchDebounce(context.Background(), req, openAIStickyEscapeOutcome{
		Escaped:       true,
		Reason:        "error_rate",
		FromAccountID: 9,
	}, 10, now)
	require.Len(t, store.debounces, 1)
}

func TestApplyOpenAITTFTScheduleDecisionCopiesCacheDiagnostics(t *testing.T) {
	fromAccountID := int64(9)
	result := &OpenAIForwardResult{Stream: true}
	service := &OpenAIGatewayService{}

	service.ApplyOpenAITTFTScheduleDecision(result, OpenAIAccountScheduleDecision{
		TTFTOptimized:             true,
		TTFTCacheProfileStatus:    "eligible",
		TTFTCacheEligible:         true,
		TTFTCacheContextTokens:    150_000,
		TTFTCacheHitRatePercent:   85,
		TTFTBaseP90Ms:             10_000,
		TTFTEffectiveP90Ms:        20_000,
		TTFTStickySwitched:        true,
		TTFTDebounceKept:          false,
		TTFTStickyEscapeReason:    openAIStickyEscapeReasonTTFTP90,
		TTFTSwitchedFromAccountID: fromAccountID,
	})

	require.True(t, result.TTFTOptimized)
	require.Equal(t, "eligible", result.TTFTCacheProfileStatus)
	require.True(t, result.TTFTCacheEligible)
	require.Equal(t, 150_000, result.TTFTCacheContextTokens)
	require.Equal(t, 85.0, result.TTFTCacheHitRatePercent)
	require.Equal(t, 10_000, result.TTFTBaseP90Ms)
	require.Equal(t, 20_000, result.TTFTEffectiveP90Ms)
	require.True(t, result.TTFTStickySwitched)
	require.Equal(t, openAIStickyEscapeReasonTTFTP90, result.TTFTStickyEscapeReason)
	require.Equal(t, fromAccountID, result.TTFTSwitchedFromAccountID)
}

func TestOrderOpenAIAccountCandidatesByTTFTKeepsPriorityAsOuterLayer(t *testing.T) {
	accounts := []*Account{
		{ID: 1, Priority: 1},
		{ID: 2, Priority: 1},
		{ID: 3, Priority: 2},
	}
	candidates := make([]openAIAccountCandidateScore, 0, len(accounts))
	for _, account := range accounts {
		candidates = append(candidates, openAIAccountCandidateScore{account: account, loadInfo: &AccountLoadInfo{AccountID: account.ID, LoadRate: 80, WaitingCount: 2}})
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 1, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 500, P90Ms: 7000},
		{AccountID: 2, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 300, P90Ms: 1000},
		{AccountID: 3, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 100, P90Ms: 100},
	}

	ordered := orderOpenAIAccountCandidatesByTTFT(candidates, snapshots, OpenAITTFTTransportHTTPSSE, 5000)
	require.Equal(t, int64(2), ordered[0].account.ID)
	// A lower P90 cannot cross the manual priority boundary.
	require.Equal(t, int64(1), ordered[1].account.ID)
	require.Equal(t, int64(3), ordered[2].account.ID)
}

func TestOrderOpenAIAccountCandidatesByTTFTUsesP90ThenP50AndFallsBackToLoad(t *testing.T) {
	accounts := []*Account{{ID: 1, Priority: 1}, {ID: 2, Priority: 1}, {ID: 3, Priority: 1}}
	candidates := []openAIAccountCandidateScore{
		{account: accounts[0], loadInfo: &AccountLoadInfo{AccountID: 1, LoadRate: 90, WaitingCount: 9}},
		{account: accounts[1], loadInfo: &AccountLoadInfo{AccountID: 2, LoadRate: 10, WaitingCount: 1}},
		{account: accounts[2], loadInfo: &AccountLoadInfo{AccountID: 3, LoadRate: 20, WaitingCount: 2}},
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 1, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 200, P90Ms: 7000},
		{AccountID: 2, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 800, P90Ms: 7000},
		{AccountID: 3, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 500, P90Ms: 7000},
	}

	ordered := orderOpenAIAccountCandidatesByTTFT(candidates, snapshots, OpenAITTFTTransportHTTPSSE, 5000)
	// No account is stable, so the best P90 ties and P50/load decide order.
	require.Equal(t, int64(1), ordered[0].account.ID)
	require.Equal(t, int64(3), ordered[1].account.ID)
	require.Equal(t, int64(2), ordered[2].account.ID)
}

func TestOrderOpenAIAccountCandidatesByTTFTPrefersMatureAccountsWhenNoStableAccountExists(t *testing.T) {
	accounts := []*Account{{ID: 11, Priority: 1}, {ID: 12, Priority: 1}}
	candidates := []openAIAccountCandidateScore{
		{account: accounts[0], loadInfo: &AccountLoadInfo{AccountID: 11, LoadRate: 5, WaitingCount: 0}},
		{account: accounts[1], loadInfo: &AccountLoadInfo{AccountID: 12, LoadRate: 90, WaitingCount: 4}},
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 11, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 1, P50Ms: 100, P90Ms: 100},
		{AccountID: 12, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 9000, P90Ms: 12000},
	}

	ordered := orderOpenAIAccountCandidatesByTTFT(candidates, snapshots, OpenAITTFTTransportHTTPSSE, 10000)
	require.Equal(t, int64(12), ordered[0].account.ID)
	require.Equal(t, int64(11), ordered[1].account.ID)
}

func TestOrderOpenAIAccountCandidatesByTTFTUsesLoadWhenAllAccountsAreImmature(t *testing.T) {
	accounts := []*Account{{ID: 21, Priority: 1}, {ID: 22, Priority: 1}}
	candidates := []openAIAccountCandidateScore{
		{account: accounts[0], loadInfo: &AccountLoadInfo{AccountID: 21, LoadRate: 90, WaitingCount: 4}},
		{account: accounts[1], loadInfo: &AccountLoadInfo{AccountID: 22, LoadRate: 10, WaitingCount: 1}},
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 21, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 1, P50Ms: 100, P90Ms: 100},
		{AccountID: 22, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 2, P50Ms: 900, P90Ms: 900},
	}

	ordered := orderOpenAIAccountCandidatesByTTFT(candidates, snapshots, OpenAITTFTTransportHTTPSSE, 10000)
	require.Equal(t, int64(22), ordered[0].account.ID)
	require.Equal(t, int64(21), ordered[1].account.ID)
}

func TestPromoteOpenAITTFTExplorationCandidateStaysWithinManualPriority(t *testing.T) {
	accounts := []*Account{{ID: 1, Priority: 1}, {ID: 2, Priority: 1}, {ID: 3, Priority: 2}}
	candidates := []openAIAccountCandidateScore{
		{account: accounts[0], loadInfo: &AccountLoadInfo{AccountID: 1, LoadRate: 10}},
		{account: accounts[1], loadInfo: &AccountLoadInfo{AccountID: 2, LoadRate: 20}},
		{account: accounts[2], loadInfo: &AccountLoadInfo{AccountID: 3, LoadRate: 0}},
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 1, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 2, P50Ms: 100, P90Ms: 200},
		{AccountID: 2, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 100, P90Ms: 200},
		{AccountID: 3, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 1, P50Ms: 50, P90Ms: 50},
	}

	ordered, explorationID, ok := promoteOpenAITTFTExplorationCandidate(
		candidates,
		snapshots,
		OpenAITTFTTransportHTTPSSE,
		OpenAIAccountScheduleRequest{Streaming: true},
		5,
		1,
	)
	require.True(t, ok)
	require.Equal(t, int64(1), explorationID)
	require.True(t, ordered[0].exploration)
	require.Equal(t, int64(1), ordered[0].account.ID)
	// The lower-priority account cannot be promoted over priority 1.
	for _, candidate := range ordered {
		require.False(t, candidate.account.ID == 3 && candidate.exploration)
	}
}

func TestPromoteOpenAITTFTExplorationCandidateSkipsStickyAndMatureRequests(t *testing.T) {
	candidates := []openAIAccountCandidateScore{{account: &Account{ID: 1, Priority: 1}}}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 1, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 100, P90Ms: 100},
	}
	for _, req := range []OpenAIAccountScheduleRequest{
		{Streaming: false},
		{Streaming: true, SessionHash: "sticky"},
		{Streaming: true, PreviousResponseID: "resp_1"},
	} {
		_, _, ok := promoteOpenAITTFTExplorationCandidate(candidates, snapshots, OpenAITTFTTransportHTTPSSE, req, 5, 1)
		require.False(t, ok)
	}
}
