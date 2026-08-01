package service

import (
	"context"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// OpenAITTFTTransport identifies the actual upstream transport. HTTP SSE and
// Responses WebSocket have materially different startup behavior and must not
// share a performance window.
type OpenAITTFTTransport string

const (
	OpenAITTFTTransportHTTPSSE     OpenAITTFTTransport = "http_sse"
	OpenAITTFTTransportResponsesWS OpenAITTFTTransport = "responses_ws"
)

func (t OpenAITTFTTransport) Valid() bool {
	return t == OpenAITTFTTransportHTTPSSE || t == OpenAITTFTTransportResponsesWS
}

type OpenAITTFTWindowKey struct {
	AccountID int64
	Transport OpenAITTFTTransport
}

type OpenAITTFTSample struct {
	AccountID  int64
	Transport  OpenAITTFTTransport
	ObservedAt time.Time
	TTFTMs     int
}

type OpenAITTFTWindowSnapshot struct {
	Count int
	P50Ms int
	P90Ms int
}

// OpenAITTFTStore is deliberately narrower than GatewayCache. Existing cache
// mocks keep compiling, while a concrete Redis gateway cache can opt into this
// capability through a type assertion.
type OpenAITTFTStore interface {
	AddSample(ctx context.Context, accountID int64, transport OpenAITTFTTransport, observedAt time.Time, ttftMs int) error
	GetWindows(ctx context.Context, keys []OpenAITTFTWindowKey, now time.Time) (map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot, error)
	TryBeginExploration(ctx context.Context, key OpenAITTFTWindowKey, token string, ttl time.Duration) (bool, error)
	FinishExploration(ctx context.Context, key OpenAITTFTWindowKey, token string, cooldown time.Duration) error
	ExplorationCoolingDown(ctx context.Context, key OpenAITTFTWindowKey) (bool, error)
	TryAcquireExplorationQuota(ctx context.Context, scope string, percent int, ttl time.Duration) (bool, error)
}

type openAITTFTSampleReader interface {
	ListRecentOpenAITTFTSamples(ctx context.Context, since time.Time, perWindowLimit int) ([]OpenAITTFTSample, error)
}

const (
	openAITTFTWindowTTL        = 25 * time.Hour
	openAITTFTWindowAge        = 24 * time.Hour
	openAITTFTWindowSampleCap  = 10
	openAITTFTDefaultTransport = OpenAITTFTTransportHTTPSSE
	// Redis is consulted on the scheduling path only with a bounded budget.
	// A slow or unavailable shared cache must never add an unbounded delay to a
	// user request; the local window remains a valid best-effort fallback.
	openAITTFTRedisReadTimeout = 3 * time.Millisecond
	// Redis sample persistence is deliberately bounded and detached from the
	// request. A stalled Redis connection must not create one goroutine per
	// completed stream.
	openAITTFTSampleQueueCapacity = 256
	openAITTFTSampleWriteTimeout  = 500 * time.Millisecond
)

type openAITTFTSampleWriter struct {
	store   OpenAITTFTStore
	queue   chan OpenAITTFTSample
	dropped atomic.Uint64
	once    sync.Once
}

func newOpenAITTFTSampleWriter(store OpenAITTFTStore, capacity int) *openAITTFTSampleWriter {
	if store == nil {
		return nil
	}
	if capacity <= 0 {
		capacity = openAITTFTSampleQueueCapacity
	}
	return &openAITTFTSampleWriter{store: store, queue: make(chan OpenAITTFTSample, capacity)}
}

func (w *openAITTFTSampleWriter) Enqueue(sample OpenAITTFTSample) bool {
	if w == nil || w.store == nil || sample.AccountID <= 0 || !sample.Transport.Valid() || sample.ObservedAt.IsZero() || sample.TTFTMs <= 0 {
		return false
	}
	w.once.Do(func() { go w.run() })
	select {
	case w.queue <- sample:
		return true
	default:
		w.dropped.Add(1)
		return false
	}
}

func (w *openAITTFTSampleWriter) Dropped() uint64 {
	if w == nil {
		return 0
	}
	return w.dropped.Load()
}

func (w *openAITTFTSampleWriter) run() {
	for sample := range w.queue {
		ctx, cancel := context.WithTimeout(context.Background(), openAITTFTSampleWriteTimeout)
		_ = w.store.AddSample(ctx, sample.AccountID, sample.Transport, sample.ObservedAt, sample.TTFTMs)
		cancel()
	}
}

type openAITTFTLocalWindow struct {
	mu      sync.RWMutex
	samples map[OpenAITTFTWindowKey][]OpenAITTFTSample
}

func newOpenAITTFTLocalWindow() *openAITTFTLocalWindow {
	return &openAITTFTLocalWindow{samples: make(map[OpenAITTFTWindowKey][]OpenAITTFTSample)}
}

// Add is a small default-key helper used by unit tests and by callers that are
// still collecting a single HTTP SSE window during migration.
func (w *openAITTFTLocalWindow) Add(observedAt time.Time, ttftMs int) {
	w.AddSample(OpenAITTFTSample{Transport: openAITTFTDefaultTransport, ObservedAt: observedAt, TTFTMs: ttftMs})
}

func (w *openAITTFTLocalWindow) AddSample(sample OpenAITTFTSample) {
	if w == nil || sample.TTFTMs <= 0 || sample.ObservedAt.IsZero() || !sample.Transport.Valid() {
		return
	}
	key := OpenAITTFTWindowKey{AccountID: sample.AccountID, Transport: sample.Transport}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.samples[key] = append(w.samples[key], sample)
	if len(w.samples[key]) > openAITTFTWindowSampleCap*4 {
		w.samples[key] = latestValidSamples(w.samples[key], time.Now().UTC())
	}
}

func (w *openAITTFTLocalWindow) Snapshot(now time.Time) OpenAITTFTWindowSnapshot {
	return w.SnapshotFor(OpenAITTFTWindowKey{Transport: openAITTFTDefaultTransport}, now)
}

func (w *openAITTFTLocalWindow) SnapshotFor(key OpenAITTFTWindowKey, now time.Time) OpenAITTFTWindowSnapshot {
	if w == nil {
		return OpenAITTFTWindowSnapshot{}
	}
	w.mu.RLock()
	samples := append([]OpenAITTFTSample(nil), w.samples[key]...)
	w.mu.RUnlock()
	return snapshotForSamples(latestValidSamples(samples, now), openAITTFTWindowSampleCap)
}

func latestValidSamples(samples []OpenAITTFTSample, now time.Time) []OpenAITTFTSample {
	cutoff := now.Add(-openAITTFTWindowAge)
	valid := make([]OpenAITTFTSample, 0, len(samples))
	for _, sample := range samples {
		if sample.TTFTMs <= 0 || sample.ObservedAt.Before(cutoff) || !sample.Transport.Valid() {
			continue
		}
		valid = append(valid, sample)
	}
	sort.SliceStable(valid, func(i, j int) bool {
		return valid[i].ObservedAt.After(valid[j].ObservedAt)
	})
	if len(valid) > openAITTFTWindowSampleCap {
		valid = valid[:openAITTFTWindowSampleCap]
	}
	return valid
}

func snapshotForSamples(samples []OpenAITTFTSample, cap int) OpenAITTFTWindowSnapshot {
	if cap > 0 && len(samples) > cap {
		samples = samples[:cap]
	}
	if len(samples) == 0 {
		return OpenAITTFTWindowSnapshot{}
	}
	values := make([]int, 0, len(samples))
	for _, sample := range samples {
		values = append(values, sample.TTFTMs)
	}
	sort.Ints(values)
	nearestRank := func(percentile float64) int {
		rank := int(math.Ceil(percentile * float64(len(values)+1)))
		if rank < 1 {
			rank = 1
		}
		if rank > len(values) {
			rank = len(values)
		}
		return values[rank-1]
	}
	return OpenAITTFTWindowSnapshot{Count: len(values), P50Ms: nearestRank(0.50), P90Ms: nearestRank(0.90)}
}

// hydrateOpenAITTFTWindow is intentionally a one-shot, caller-controlled
// operation. It performs the bounded history query off the request path; a
// failure leaves the local window empty and callers can continue in EWMA/load
// fallback mode without delaying user traffic.
func hydrateOpenAITTFTWindow(ctx context.Context, reader openAITTFTSampleReader, window *openAITTFTLocalWindow, now time.Time) error {
	if reader == nil || window == nil {
		return nil
	}
	samples, err := reader.ListRecentOpenAITTFTSamples(ctx, now.Add(-openAITTFTWindowAge), openAITTFTWindowSampleCap)
	if err != nil {
		return err
	}
	for _, sample := range samples {
		window.AddSample(sample)
	}
	return nil
}

func (s *OpenAIGatewayService) ensureOpenAITTFTHydrated() {
	if s == nil || s.openAITTFTWindow == nil {
		return
	}
	s.openAITTFTHydrateOnce.Do(func() {
		reader, ok := s.usageLogRepo.(openAITTFTSampleReader)
		if !ok {
			return
		}
		go func() {
			_ = hydrateOpenAITTFTWindow(context.Background(), reader, s.openAITTFTWindow, time.Now().UTC())
		}()
	})
}

func (s *OpenAIGatewayService) openAITTFTSnapshots(ctx context.Context, keys []OpenAITTFTWindowKey, now time.Time) map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot {
	result := make(map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot, len(keys))
	if s == nil {
		return result
	}
	s.ensureOpenAITTFTHydrated()
	if s.openAITTFTWindow != nil {
		for _, key := range keys {
			result[key] = s.openAITTFTWindow.SnapshotFor(key, now)
		}
	}

	store, ok := s.cache.(OpenAITTFTStore)
	if !ok || len(keys) == 0 {
		s.openAITTFTRedisHealthy.Store(false)
		return result
	}
	readCtx := ctx
	if readCtx == nil {
		readCtx = context.Background()
	}
	readCtx, cancel := context.WithTimeout(readCtx, openAITTFTRedisReadTimeout)
	defer cancel()
	shared, err := store.GetWindows(readCtx, keys, now)
	if err != nil {
		s.openAITTFTRedisHealthy.Store(false)
		return result
	}
	s.openAITTFTRedisHealthy.Store(true)
	for _, key := range keys {
		// Keep a local sample when the asynchronous Redis write has not landed
		// yet. Once Redis has data, it is the cross-instance source of truth.
		if snapshot, exists := shared[key]; exists && snapshot.Count > 0 {
			result[key] = snapshot
		}
	}
	return result
}

func (s *OpenAIGatewayService) recordOpenAITTFTSample(log *UsageLog) {
	if s == nil || log == nil || log.AccountID <= 0 || log.FirstTokenMs == nil || *log.FirstTokenMs <= 0 || log.ClientDisconnected {
		return
	}
	if log.RequestType != RequestTypeStream && log.RequestType != RequestTypeWSV2 {
		return
	}
	if log.ImageCount > 0 || log.VideoCount > 0 {
		return
	}
	if log.BillingMode != nil {
		switch BillingMode(*log.BillingMode) {
		case BillingModeImage, BillingModeVideo:
			return
		}
	}
	transport := OpenAITTFTTransportHTTPSSE
	if log.OpenAITTFTContext != nil && log.OpenAITTFTContext.Transport != "" {
		transport = OpenAITTFTTransport(log.OpenAITTFTContext.Transport)
	}
	if !transport.Valid() {
		return
	}
	s.ensureOpenAITTFTHydrated()
	if s.openAITTFTWindow == nil {
		s.openAITTFTWindow = newOpenAITTFTLocalWindow()
	}
	now := log.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	// The optimizer targets the same user-visible "first token" value shown in
	// usage records. UpstreamFirstTokenMs remains an admin-only diagnostic and
	// deliberately does not affect account ranking.
	s.openAITTFTWindow.AddSample(OpenAITTFTSample{AccountID: log.AccountID, Transport: transport, ObservedAt: now, TTFTMs: *log.FirstTokenMs})
	if writer := s.openAITTFTSampleWriterForRedis(); writer != nil {
		_ = writer.Enqueue(OpenAITTFTSample{
			AccountID:  log.AccountID,
			Transport:  transport,
			ObservedAt: now,
			TTFTMs:     *log.FirstTokenMs,
		})
	}
}

func (s *OpenAIGatewayService) openAITTFTSampleWriterForRedis() *openAITTFTSampleWriter {
	if s == nil {
		return nil
	}
	s.openAITTFTSampleWriterOnce.Do(func() {
		store, ok := s.cache.(OpenAITTFTStore)
		if ok {
			s.openAITTFTSampleWriter = newOpenAITTFTSampleWriter(store, openAITTFTSampleQueueCapacity)
		}
	})
	return s.openAITTFTSampleWriter
}
