package service

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	openAITTFTCacheProfileTTL           = 10 * time.Minute
	openAITTFTCacheSwitchDebounceTTL    = 5 * time.Minute
	openAITTFTCacheEvaluationStatusOK   = "eligible"
	openAITTFTCacheEvaluationStatusOff  = "disabled"
	openAITTFTCacheProfileQueueSize     = 256
	openAITTFTCacheWriteTimeout         = 500 * time.Millisecond
	openAITTFTCacheReadTimeout          = 3 * time.Millisecond
	openAITTFTCacheDebounceWriteTimeout = 5 * time.Millisecond
)

type OpenAITTFTCacheProfileKey struct {
	GroupID     int64
	SessionHash string
	AccountID   int64
}

type OpenAITTFTCacheProfile struct {
	ObservedAt         time.Time
	TotalContextTokens int
	CacheReadTokens    int
	CacheWriteTokens   int
}

type OpenAITTFTSwitchDebounceKey struct {
	GroupID     int64
	SessionHash string
}

type OpenAITTFTSwitchDebounce struct {
	FromAccountID int64
	ToAccountID   int64
	SwitchedAt    time.Time
}

type OpenAITTFTCacheState struct {
	Profile     OpenAITTFTCacheProfile
	HasImage    bool
	Debounce    OpenAITTFTSwitchDebounce
	HasDebounce bool
}

type OpenAITTFTCacheProfileStore interface {
	GetOpenAITTFTCacheState(context.Context, OpenAITTFTCacheProfileKey) (OpenAITTFTCacheState, error)
	PutOpenAITTFTCacheProfile(context.Context, OpenAITTFTCacheProfileKey, OpenAITTFTCacheProfile, time.Duration) error
	PutOpenAITTFTSwitchDebounce(context.Context, OpenAITTFTSwitchDebounceKey, OpenAITTFTSwitchDebounce, time.Duration) error
}

type OpenAITTFTCacheProfileObservation struct {
	Key     OpenAITTFTCacheProfileKey
	Profile OpenAITTFTCacheProfile
}

type openAITTFTCacheProfileWriter struct {
	store   OpenAITTFTCacheProfileStore
	queue   chan OpenAITTFTCacheProfileObservation
	dropped atomic.Uint64
	once    sync.Once
}

func newOpenAITTFTCacheProfileWriter(store OpenAITTFTCacheProfileStore, capacity int) *openAITTFTCacheProfileWriter {
	if store == nil {
		return nil
	}
	if capacity <= 0 {
		capacity = openAITTFTCacheProfileQueueSize
	}
	return &openAITTFTCacheProfileWriter{store: store, queue: make(chan OpenAITTFTCacheProfileObservation, capacity)}
}

func (w *openAITTFTCacheProfileWriter) Enqueue(observation OpenAITTFTCacheProfileObservation) bool {
	if w == nil || w.store == nil || observation.Key.AccountID <= 0 || observation.Key.GroupID < 0 || observation.Key.SessionHash == "" || observation.Profile.ObservedAt.IsZero() || observation.Profile.TotalContextTokens <= 0 || observation.Profile.CacheReadTokens < 0 || observation.Profile.CacheWriteTokens < 0 || observation.Profile.CacheReadTokens+observation.Profile.CacheWriteTokens > observation.Profile.TotalContextTokens {
		return false
	}
	w.once.Do(func() { go w.run() })
	select {
	case w.queue <- observation:
		return true
	default:
		w.dropped.Add(1)
		return false
	}
}

func (w *openAITTFTCacheProfileWriter) Dropped() uint64 {
	if w == nil {
		return 0
	}
	return w.dropped.Load()
}

func (w *openAITTFTCacheProfileWriter) run() {
	for observation := range w.queue {
		ctx, cancel := context.WithTimeout(context.Background(), openAITTFTCacheWriteTimeout)
		_ = w.store.PutOpenAITTFTCacheProfile(ctx, observation.Key, observation.Profile, openAITTFTCacheProfileTTL)
		cancel()
	}
}

func buildOpenAITTFTCacheProfileObservation(input *OpenAIRecordUsageInput, observedAt time.Time) (OpenAITTFTCacheProfileObservation, bool) {
	if input == nil || input.Result == nil || input.APIKey == nil || input.APIKey.GroupID == nil || input.Account == nil {
		return OpenAITTFTCacheProfileObservation{}, false
	}
	result := input.Result
	sessionHash := strings.TrimSpace(input.SessionHash)
	if !result.Stream || result.ClientDisconnect || input.CyberBlocked || input.Account.Platform != PlatformOpenAI || input.Account.ID <= 0 || sessionHash == "" {
		return OpenAITTFTCacheProfileObservation{}, false
	}
	total := result.Usage.InputTokens
	read := result.Usage.CacheReadInputTokens
	written := result.Usage.CacheCreationInputTokens
	if total <= 0 || read < 0 || written < 0 || read+written > total {
		return OpenAITTFTCacheProfileObservation{}, false
	}
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	return OpenAITTFTCacheProfileObservation{
		Key: OpenAITTFTCacheProfileKey{
			GroupID:     *input.APIKey.GroupID,
			SessionHash: sessionHash,
			AccountID:   input.Account.ID,
		},
		Profile: OpenAITTFTCacheProfile{
			ObservedAt:         observedAt,
			TotalContextTokens: total,
			CacheReadTokens:    read,
			CacheWriteTokens:   written,
		},
	}, true
}

func (s *OpenAIGatewayService) recordOpenAITTFTCacheProfile(input *OpenAIRecordUsageInput, observedAt time.Time) {
	if s == nil {
		return
	}
	runtimeSettings := s.openAITTFTRuntimeSettings(context.Background())
	if !runtimeSettings.Enabled || !runtimeSettings.CacheProtectionEnabled {
		return
	}
	observation, ok := buildOpenAITTFTCacheProfileObservation(input, observedAt)
	if !ok {
		return
	}
	if writer := s.openAITTFTCacheProfileWriterForRedis(); writer != nil {
		_ = writer.Enqueue(observation)
	}
}

func (s *OpenAIGatewayService) openAITTFTCacheProfileWriterForRedis() *openAITTFTCacheProfileWriter {
	if s == nil {
		return nil
	}
	s.openAITTFTCacheWriterOnce.Do(func() {
		if store, ok := s.cache.(OpenAITTFTCacheProfileStore); ok {
			s.openAITTFTCacheWriter = newOpenAITTFTCacheProfileWriter(store, openAITTFTCacheProfileQueueSize)
		}
	})
	return s.openAITTFTCacheWriter
}

type OpenAITTFTCachePolicy struct {
	Enabled           bool
	BaseP90Ms         int
	MinContextTokens  int
	MinHitRatePercent int
	ElasticP90CapMs   int
}

type OpenAITTFTCacheEvaluation struct {
	Eligible           bool
	Status             string
	TotalContextTokens int
	HitRatePercent     float64
	BaseP90Ms          int
	EffectiveP90Ms     int
}

func evaluateOpenAITTFTCacheProfile(policy OpenAITTFTCachePolicy, profile OpenAITTFTCacheProfile, now time.Time) OpenAITTFTCacheEvaluation {
	policy = normalizeOpenAITTFTCachePolicy(policy)
	evaluation := OpenAITTFTCacheEvaluation{
		Status:         openAITTFTCacheEvaluationStatusOK,
		BaseP90Ms:      policy.BaseP90Ms,
		EffectiveP90Ms: policy.BaseP90Ms,
	}
	if !policy.Enabled {
		evaluation.Status = openAITTFTCacheEvaluationStatusOff
		return evaluation
	}
	if profile.ObservedAt.IsZero() {
		evaluation.Status = "missing"
		return evaluation
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if profile.ObservedAt.Before(now.Add(-openAITTFTCacheProfileTTL)) {
		evaluation.Status = "expired"
		return evaluation
	}
	if profile.TotalContextTokens <= 0 || profile.CacheReadTokens < 0 || profile.CacheWriteTokens < 0 || profile.CacheReadTokens+profile.CacheWriteTokens > profile.TotalContextTokens {
		evaluation.Status = "invalid"
		return evaluation
	}
	evaluation.TotalContextTokens = profile.TotalContextTokens
	evaluation.HitRatePercent = float64(profile.CacheReadTokens) * 100 / float64(profile.TotalContextTokens)
	if profile.TotalContextTokens < policy.MinContextTokens {
		evaluation.Status = "short_context"
		return evaluation
	}
	if evaluation.HitRatePercent+1e-9 < float64(policy.MinHitRatePercent) {
		evaluation.Status = "low_hit_rate"
		return evaluation
	}
	evaluation.Eligible = true
	steps := (profile.TotalContextTokens-policy.MinContextTokens)/50_000 + 1
	evaluation.EffectiveP90Ms = policy.BaseP90Ms + steps*5_000
	if evaluation.EffectiveP90Ms > policy.ElasticP90CapMs {
		evaluation.EffectiveP90Ms = policy.ElasticP90CapMs
	}
	return evaluation
}

func normalizeOpenAITTFTCachePolicy(policy OpenAITTFTCachePolicy) OpenAITTFTCachePolicy {
	if policy.BaseP90Ms <= 0 {
		policy.BaseP90Ms = 10_000
	}
	if policy.MinContextTokens <= 0 {
		policy.MinContextTokens = 100_000
	}
	if policy.MinHitRatePercent <= 0 {
		policy.MinHitRatePercent = 80
	}
	if policy.ElasticP90CapMs < policy.BaseP90Ms {
		policy.ElasticP90CapMs = policy.BaseP90Ms
	}
	return policy
}
