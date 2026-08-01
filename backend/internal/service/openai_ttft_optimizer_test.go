package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type openAITTFTSampleReaderStub struct {
	samples []OpenAITTFTSample
	since   time.Time
	limit   int
	err     error
}

func (s *openAITTFTSampleReaderStub) ListRecentOpenAITTFTSamples(_ context.Context, since time.Time, limit int) ([]OpenAITTFTSample, error) {
	s.since = since
	s.limit = limit
	return s.samples, s.err
}

type openAITTFTSnapshotStoreStub struct {
	windows map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot
	err     error
}

func (s *openAITTFTSnapshotStoreStub) AddSample(context.Context, int64, OpenAITTFTTransport, time.Time, int) error {
	return nil
}

func (s *openAITTFTSnapshotStoreStub) GetWindows(context.Context, []OpenAITTFTWindowKey, time.Time) (map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.windows, nil
}

func (s *openAITTFTSnapshotStoreStub) TryBeginExploration(context.Context, OpenAITTFTWindowKey, string, time.Duration) (bool, error) {
	return false, nil
}

func (s *openAITTFTSnapshotStoreStub) FinishExploration(context.Context, OpenAITTFTWindowKey, string, time.Duration) error {
	return nil
}

func (s *openAITTFTSnapshotStoreStub) ExplorationCoolingDown(context.Context, OpenAITTFTWindowKey) (bool, error) {
	return false, nil
}

func (s *openAITTFTSnapshotStoreStub) TryAcquireExplorationQuota(context.Context, string, int, time.Duration) (bool, error) {
	return true, nil
}

type openAITTFTSnapshotGatewayCacheStub struct {
	GatewayCache
	*openAITTFTSnapshotStoreStub
}

type openAITTFTBlockingStore struct {
	openAITTFTSnapshotStoreStub
	started   chan struct{}
	release   chan struct{}
	startOnce sync.Once
	mu        sync.Mutex
	callCount int
}

func (s *openAITTFTBlockingStore) AddSample(ctx context.Context, accountID int64, transport OpenAITTFTTransport, observedAt time.Time, ttftMs int) error {
	s.startOnce.Do(func() { close(s.started) })
	s.mu.Lock()
	s.callCount++
	s.mu.Unlock()
	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *openAITTFTBlockingStore) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callCount
}

func TestOpenAITTFTSampleWriterIsBoundedWhenRedisStalls(t *testing.T) {
	store := &openAITTFTBlockingStore{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	writer := newOpenAITTFTSampleWriter(store, 1)
	sample := OpenAITTFTSample{AccountID: 7, Transport: OpenAITTFTTransportHTTPSSE, ObservedAt: time.Now().UTC(), TTFTMs: 240}

	require.True(t, writer.Enqueue(sample))
	select {
	case <-store.started:
	case <-time.After(time.Second):
		t.Fatal("sample writer did not start")
	}
	require.True(t, writer.Enqueue(sample))
	require.False(t, writer.Enqueue(sample))
	require.Equal(t, uint64(1), writer.Dropped())

	close(store.release)
	require.Eventually(t, func() bool { return store.calls() >= 2 }, time.Second, time.Millisecond*10)
}

func TestRecordOpenAITTFTSampleUsesDisplayedFirstToken(t *testing.T) {
	firstTokenMs := 321
	upstreamFirstTokenMs := 80
	now := time.Now().UTC()
	svc := &OpenAIGatewayService{openAITTFTWindow: newOpenAITTFTLocalWindow()}

	svc.recordOpenAITTFTSample(&UsageLog{
		AccountID:            7,
		RequestType:          RequestTypeStream,
		CreatedAt:            now,
		FirstTokenMs:         &firstTokenMs,
		UpstreamFirstTokenMs: &upstreamFirstTokenMs,
		OpenAITTFTContext:    &OpenAITTFTContext{Transport: string(OpenAITTFTTransportHTTPSSE)},
	})

	snapshot := svc.openAITTFTWindow.SnapshotFor(OpenAITTFTWindowKey{
		AccountID: 7,
		Transport: OpenAITTFTTransportHTTPSSE,
	}, now)
	require.Equal(t, OpenAITTFTWindowSnapshot{Count: 1, P50Ms: firstTokenMs, P90Ms: firstTokenMs}, snapshot)
}

func TestRecordOpenAITTFTSampleSkipsNonTokenStreams(t *testing.T) {
	firstTokenMs := 321
	now := time.Now().UTC()
	svc := &OpenAIGatewayService{openAITTFTWindow: newOpenAITTFTLocalWindow()}
	transportContext := &OpenAITTFTContext{Transport: string(OpenAITTFTTransportHTTPSSE)}

	imageMode := string(BillingModeImage)
	svc.recordOpenAITTFTSample(&UsageLog{
		AccountID:         7,
		RequestType:       RequestTypeStream,
		CreatedAt:         now,
		FirstTokenMs:      &firstTokenMs,
		ImageCount:        1,
		BillingMode:       &imageMode,
		OpenAITTFTContext: transportContext,
	})

	snapshot := svc.openAITTFTWindow.SnapshotFor(OpenAITTFTWindowKey{
		AccountID: 7,
		Transport: OpenAITTFTTransportHTTPSSE,
	}, now)
	require.Equal(t, OpenAITTFTWindowSnapshot{}, snapshot)
}

func TestOpenAITTFTSnapshotsUsesRedisWindowForSharedScheduling(t *testing.T) {
	now := time.Now().UTC()
	key := OpenAITTFTWindowKey{AccountID: 7, Transport: OpenAITTFTTransportHTTPSSE}
	store := &openAITTFTSnapshotStoreStub{windows: map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		key: {Count: 10, P50Ms: 180, P90Ms: 420},
	}}
	svc := &OpenAIGatewayService{
		cache:            &openAITTFTSnapshotGatewayCacheStub{openAITTFTSnapshotStoreStub: store},
		openAITTFTWindow: newOpenAITTFTLocalWindow(),
	}
	svc.openAITTFTWindow.AddSample(OpenAITTFTSample{AccountID: key.AccountID, Transport: key.Transport, ObservedAt: now, TTFTMs: 900})

	snapshots := svc.openAITTFTSnapshots(context.Background(), []OpenAITTFTWindowKey{key}, now)
	require.Equal(t, store.windows[key], snapshots[key])
	require.True(t, svc.openAITTFTRedisHealthy.Load())
}

func TestOpenAITTFTSnapshotsFallsBackToLocalWhenRedisFails(t *testing.T) {
	now := time.Now().UTC()
	key := OpenAITTFTWindowKey{AccountID: 8, Transport: OpenAITTFTTransportHTTPSSE}
	store := &openAITTFTSnapshotStoreStub{err: errors.New("redis unavailable")}
	svc := &OpenAIGatewayService{
		cache:            &openAITTFTSnapshotGatewayCacheStub{openAITTFTSnapshotStoreStub: store},
		openAITTFTWindow: newOpenAITTFTLocalWindow(),
	}
	svc.openAITTFTWindow.AddSample(OpenAITTFTSample{AccountID: key.AccountID, Transport: key.Transport, ObservedAt: now, TTFTMs: 700})

	snapshots := svc.openAITTFTSnapshots(context.Background(), []OpenAITTFTWindowKey{key}, now)
	require.Equal(t, OpenAITTFTWindowSnapshot{Count: 1, P50Ms: 700, P90Ms: 700}, snapshots[key])
	require.False(t, svc.openAITTFTRedisHealthy.Load())
}

func TestOpenAITTFTWindowUsesLatestTenValidSamples(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	window := newOpenAITTFTLocalWindow()
	window.Add(now.Add(-25*time.Hour), 1)
	for i := 1; i <= 12; i++ {
		window.Add(now.Add(time.Duration(i)*time.Minute), i*100)
	}

	snapshot := window.Snapshot(now)
	require.Equal(t, 10, snapshot.Count)
	require.Equal(t, 800, snapshot.P50Ms)
	require.Equal(t, 1200, snapshot.P90Ms)
}

func TestOpenAITTFTWindowSeparatesTransportAndIgnoresInvalidSamples(t *testing.T) {
	now := time.Now().UTC()
	window := newOpenAITTFTLocalWindow()
	window.AddSample(OpenAITTFTSample{AccountID: 7, Transport: OpenAITTFTTransportHTTPSSE, ObservedAt: now, TTFTMs: 100})
	window.AddSample(OpenAITTFTSample{AccountID: 7, Transport: OpenAITTFTTransportResponsesWS, ObservedAt: now, TTFTMs: 200})
	window.AddSample(OpenAITTFTSample{AccountID: 7, Transport: OpenAITTFTTransportHTTPSSE, ObservedAt: now, TTFTMs: 0})

	sse := window.SnapshotFor(OpenAITTFTWindowKey{AccountID: 7, Transport: OpenAITTFTTransportHTTPSSE}, now)
	ws := window.SnapshotFor(OpenAITTFTWindowKey{AccountID: 7, Transport: OpenAITTFTTransportResponsesWS}, now)
	require.Equal(t, 1, sse.Count)
	require.Equal(t, 100, sse.P50Ms)
	require.Equal(t, 1, ws.Count)
	require.Equal(t, 200, ws.P90Ms)
}

func TestHydrateOpenAITTFTWindowLoadsRecentSamplesWithoutBlockingRequests(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	reader := &openAITTFTSampleReaderStub{samples: []OpenAITTFTSample{{
		AccountID: 7, Transport: OpenAITTFTTransportHTTPSSE, ObservedAt: now.Add(-time.Minute), TTFTMs: 240,
	}}}
	window := newOpenAITTFTLocalWindow()

	require.NoError(t, hydrateOpenAITTFTWindow(context.Background(), reader, window, now))
	require.Equal(t, 10, reader.limit)
	require.Equal(t, now.Add(-24*time.Hour), reader.since)
	snapshot := window.SnapshotFor(OpenAITTFTWindowKey{AccountID: 7, Transport: OpenAITTFTTransportHTTPSSE}, now)
	require.Equal(t, 1, snapshot.Count)
	require.Equal(t, 240, snapshot.P90Ms)
}
