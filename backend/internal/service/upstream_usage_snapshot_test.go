package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type upstreamUsageByAccountStub struct {
	mu       sync.Mutex
	requests map[int64][]*http.Request
}

func (u *upstreamUsageByAccountStub) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.mu.Lock()
	if u.requests == nil {
		u.requests = make(map[int64][]*http.Request)
	}
	u.requests[accountID] = append(u.requests[accountID], req)
	u.mu.Unlock()
	if accountID == 2 {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"error":"invalid key"}`)),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"usage":{"total":{"actual_cost":12.5}},"balance":87.5}`)),
	}, nil
}

func (u *upstreamUsageByAccountStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func (u *upstreamUsageByAccountStub) callCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	total := 0
	for _, requests := range u.requests {
		total += len(requests)
	}
	return total
}

func (r *upstreamBillingProbeAccountRepo) ListActive(_ context.Context) ([]Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	accounts := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.IsActive() {
			clone := *account
			clone.Credentials = mergeMap(nil, account.Credentials)
			clone.Extra = mergeMap(nil, account.Extra)
			accounts = append(accounts, clone)
		}
	}
	return accounts, nil
}

func TestUpstreamUsageSnapshotParsesCumulativeActualCost(t *testing.T) {
	observedAt := time.Date(2026, time.August, 18, 8, 0, 0, 0, time.UTC)
	snapshot := parseUpstreamUsageSnapshot(http.StatusOK, []byte(`{
		"usage": {
			"total": {"actual_cost": 12.345678}
		},
		"daily_usage": [{"date": "2026-08-18", "actual_cost": 1.25}],
		"model_stats": [{"model": "gpt-5", "actual_cost": 1.25}],
		"balance": 87.654322
	}`), observedAt)

	require.Equal(t, UpstreamUsageSnapshotStatusOK, snapshot.Status)
	require.Equal(t, observedAt, snapshot.ObservedAt)
	require.Equal(t, http.StatusOK, snapshot.HTTPStatus)
	require.NotNil(t, snapshot.CumulativeActualCost)
	require.Equal(t, "12.345678", snapshot.CumulativeActualCost.String())
	require.NotNil(t, snapshot.Balance)
	require.Equal(t, "87.654322", snapshot.Balance.String())
	require.JSONEq(t, `[{"date":"2026-08-18","actual_cost":1.25}]`, string(snapshot.DailyUsage))
	require.JSONEq(t, `[{"model":"gpt-5","actual_cost":1.25}]`, string(snapshot.ModelUsage))
	require.Empty(t, snapshot.ErrorCode)
}

func TestUpstreamUsageSnapshotMarksUnauthorizedWithoutRetainingBody(t *testing.T) {
	snapshot := parseUpstreamUsageSnapshot(http.StatusUnauthorized, []byte(`{
		"error": "Bearer sk-sensitive is invalid"
	}`), time.Now())

	require.Equal(t, UpstreamUsageSnapshotStatusUnauthorized, snapshot.Status)
	require.Equal(t, "unauthorized", snapshot.ErrorCode)
	require.Nil(t, snapshot.CumulativeActualCost)
	require.Empty(t, snapshot.DailyUsage)
	require.Empty(t, snapshot.ModelUsage)
}

func TestUpstreamUsageSnapshotRejectsMissingCumulativeActualCost(t *testing.T) {
	snapshot := parseUpstreamUsageSnapshot(http.StatusOK, []byte(`{
		"usage": {"total": {"requests": 12}},
		"balance": 50
	}`), time.Now())

	require.Equal(t, UpstreamUsageSnapshotStatusInvalid, snapshot.Status)
	require.Equal(t, "missing_actual_cost", snapshot.ErrorCode)
	require.Nil(t, snapshot.CumulativeActualCost)
}

func TestUpstreamUsageReconciliationTreatsCounterDecreaseAsReset(t *testing.T) {
	previous := decimal.RequireFromString("15")
	current := decimal.RequireFromString("2")

	delta, status := reconcileUpstreamUsageCumulative(&previous, &current)

	require.Equal(t, UpstreamUsageSnapshotStatusReset, status)
	require.Nil(t, delta, "a reset must never become a negative upstream cost")
}

func TestUpstreamUsageSnapshotPersistenceMarksCounterReset(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	observedAt := time.Date(2026, time.August, 18, 8, 10, 0, 0, time.UTC)
	snapshot := parseUpstreamUsageSnapshot(http.StatusOK, []byte(`{
		"usage": {"total": {"actual_cost": 2}},
		"balance": 98
	}`), observedAt)
	mock.ExpectQuery(`(?s)SELECT cumulative_actual_cost.*FROM upstream_usage_snapshots.*account_id = \$1.*ORDER BY observed_at DESC, id DESC.*LIMIT 1`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"cumulative_actual_cost"}).AddRow("15"))
	mock.ExpectExec(`(?s)INSERT INTO upstream_usage_snapshots`).
		WithArgs(
			int64(7), observedAt, UpstreamUsageSnapshotStatusReset, http.StatusOK,
			sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, "counter_reset",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc := NewUpstreamBillingProbeService(nil, nil, nil)
	svc.SetLeaderLock(nil, db)
	require.NoError(t, svc.persistUpstreamUsageSnapshot(context.Background(), 7, snapshot))
	require.Equal(t, UpstreamUsageSnapshotStatusReset, snapshot.Status)
	require.Equal(t, "counter_reset", snapshot.ErrorCode)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpstreamUsageSnapshotBuildsUsageEndpointFromRootOrV1Base(t *testing.T) {
	for _, baseURL := range []string{"https://upstream.example", "https://upstream.example/v1"} {
		t.Run(baseURL, func(t *testing.T) {
			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"usage":{"total":{"actual_cost":1}}}`)),
			}}
			svc := newUpstreamBillingProbeTestService(&upstreamBillingProbeAccountRepo{}, upstream, &upstreamBillingProbeSettingRepo{})
			account := &Account{
				ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test", "base_url": baseURL},
			}

			snapshot := svc.fetchUpstreamUsageSnapshot(context.Background(), account)

			require.Equal(t, UpstreamUsageSnapshotStatusOK, snapshot.Status)
			require.NotNil(t, upstream.lastReq)
			require.Equal(t, http.MethodGet, upstream.lastReq.Method)
			require.Equal(t, "https://upstream.example/v1/usage", upstream.lastReq.URL.String())
			require.Equal(t, "application/json", upstream.lastReq.Header.Get("Accept"))
			require.Equal(t, "Bearer sk-test", upstream.lastReq.Header.Get("Authorization"))
			require.Empty(t, upstream.lastBody)
		})
	}
}

func TestUpstreamUsageSnapshotMapsTimeoutToIsolatedFailure(t *testing.T) {
	upstream := &httpUpstreamRecorder{err: context.DeadlineExceeded}
	svc := newUpstreamBillingProbeTestService(&upstreamBillingProbeAccountRepo{}, upstream, &upstreamBillingProbeSettingRepo{})
	account := &Account{
		ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://upstream.example"},
	}

	snapshot := svc.fetchUpstreamUsageSnapshot(context.Background(), account)

	require.Equal(t, UpstreamUsageSnapshotStatusFailed, snapshot.Status)
	require.Equal(t, "request_failed", snapshot.ErrorCode)
}

func TestUpstreamUsageSnapshotRunnerFiltersAccountsIsolatesFailuresAndRunsOncePerWindow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.MatchExpectationsInOrder(false)

	accounts := map[int64]*Account{
		1: {ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-one", "base_url": "https://one.example"}},
		2: {ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-two", "base_url": "https://two.example/v1"}},
		3: {ID: 3, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-three", "base_url": "https://three.example"}},
		4: {ID: 4, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Concurrency: 1, Credentials: map[string]any{"base_url": "https://four.example"}},
		5: {ID: 5, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-five"}},
		6: {ID: 6, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusDisabled, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-six", "base_url": "https://six.example"}},
	}
	repo := &upstreamBillingProbeAccountRepo{accounts: accounts}
	upstream := &upstreamUsageByAccountStub{}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})
	svc.SetLeaderLock(&fakeLeaderLockCache{}, db)
	svc.now = func() time.Time { return time.Date(2026, time.August, 18, 8, 12, 0, 0, time.UTC) }

	mock.ExpectQuery(`(?s)SELECT cumulative_actual_cost.*FROM upstream_usage_snapshots`).
		WithArgs(int64(1)).
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectExec(`(?s)INSERT INTO upstream_usage_snapshots`).
		WithArgs(int64(2), sqlmock.AnyArg(), UpstreamUsageSnapshotStatusUnauthorized, http.StatusUnauthorized, nil, nil, nil, nil, "unauthorized").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, svc.RunUpstreamUsageSnapshotsDue(context.Background()))
	require.NoError(t, svc.RunUpstreamUsageSnapshotsDue(context.Background()))
	require.Equal(t, 2, upstream.callCount(), "only two eligible accounts should be sampled once in the ten-minute window")
	require.NoError(t, mock.ExpectationsWereMet())
}
