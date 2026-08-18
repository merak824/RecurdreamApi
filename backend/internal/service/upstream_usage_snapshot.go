package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

const (
	UpstreamUsageSnapshotStatusOK           = "ok"
	UpstreamUsageSnapshotStatusReset        = "reset"
	UpstreamUsageSnapshotStatusUnauthorized = "unauthorized"
	UpstreamUsageSnapshotStatusUnsupported  = "unsupported"
	UpstreamUsageSnapshotStatusFailed       = "failed"
	UpstreamUsageSnapshotStatusInvalid      = "invalid_response"

	upstreamUsageSnapshotMaxDetailBytes = 16 * 1024
)

var upstreamUsageSnapshotMaxAmount = decimal.New(1, 20)

// UpstreamUsageSnapshot contains only the upstream fields needed for cost
// reconciliation. Raw response bodies and authorization data are never kept.
type UpstreamUsageSnapshot struct {
	ObservedAt           time.Time
	Status               string
	HTTPStatus           int
	CumulativeActualCost *decimal.Decimal
	Balance              *decimal.Decimal
	DailyUsage           json.RawMessage
	ModelUsage           json.RawMessage
	ErrorCode            string
}

func upstreamUsageSnapshotLeaderLockKeyAt(now time.Time) string {
	return fmt.Sprintf("%s:%d", upstreamUsageSnapshotLeaderLockKey, now.Unix()/int64(upstreamUsageSnapshotInterval/time.Second))
}

// RunUpstreamUsageSnapshotsDue samples every active, statically configured API
// key at most once per ten-minute window. Account failures are persisted and do
// not fail the rest of the batch.
func (s *UpstreamBillingProbeService) RunUpstreamUsageSnapshotsDue(ctx context.Context) error {
	if s == nil || s.accountRepo == nil || s.db == nil {
		return nil
	}
	s.usageCycleMu.Lock()
	defer s.usageCycleMu.Unlock()

	now := s.currentTime().UTC()
	bucket := now.Unix() / int64(upstreamUsageSnapshotInterval/time.Second)
	if s.lastUsageSnapshotBucket == bucket {
		return nil
	}
	lockKey := upstreamUsageSnapshotLeaderLockKeyAt(now)
	release, acquired, err := s.tryAcquireLeaderLockWithTTL(ctx, lockKey, upstreamUsageSnapshotLeaderLockTTL)
	if err != nil {
		return fmt.Errorf("acquire upstream usage snapshot leader lock: %w", err)
	}
	if !acquired {
		return nil
	}
	s.lastUsageSnapshotBucket = bucket
	releaseAt := now.Truncate(upstreamUsageSnapshotInterval).Add(upstreamUsageSnapshotInterval)
	defer releaseUpstreamBillingProbeLeaderLock(release, releaseAt)

	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active upstream usage accounts: %w", err)
	}
	eligible := make([]Account, 0, len(accounts))
	for i := range accounts {
		account := accounts[i]
		if !eligibleUpstreamUsageSnapshotAccount(&account) {
			continue
		}
		eligible = append(eligible, account)
	}
	sort.SliceStable(eligible, func(i, j int) bool { return eligible[i].ID < eligible[j].ID })

	var group errgroup.Group
	for i := range eligible {
		account := eligible[i]
		group.Go(func() error {
			if err := s.collectUpstreamUsageSnapshot(ctx, &account); err != nil {
				logger.LegacyPrintf(
					"service.upstream_billing_probe",
					"usage_snapshot_failed: account_id=%d error_code=persist_failed",
					account.ID,
				)
			}
			return nil
		})
	}
	return group.Wait()
}

func eligibleUpstreamUsageSnapshotAccount(account *Account) bool {
	return account != nil &&
		account.IsActive() &&
		isUpstreamBillingProbeAccount(account) &&
		strings.TrimSpace(account.GetCredential("api_key")) != "" &&
		strings.TrimSpace(account.GetCredential("base_url")) != ""
}

func (s *UpstreamBillingProbeService) collectUpstreamUsageSnapshot(ctx context.Context, account *Account) error {
	select {
	case s.probeSlots <- struct{}{}:
		defer func() { <-s.probeSlots }()
	case <-ctx.Done():
		return ctx.Err()
	}

	key := "usage:" + strconv.FormatInt(account.ID, 10)
	_, err, _ := s.probeGroup.Do(key, func() (any, error) {
		snapshot := s.fetchUpstreamUsageSnapshot(ctx, account)
		return nil, s.persistUpstreamUsageSnapshot(ctx, account.ID, snapshot)
	})
	return err
}

func (s *UpstreamBillingProbeService) fetchUpstreamUsageSnapshot(ctx context.Context, account *Account) *UpstreamUsageSnapshot {
	now := s.currentTime().UTC()
	failure := func(status, errorCode string, httpStatus int) *UpstreamUsageSnapshot {
		return &UpstreamUsageSnapshot{
			ObservedAt: now,
			Status:     status,
			HTTPStatus: httpStatus,
			ErrorCode:  errorCode,
		}
	}
	if s.accountTestService == nil || s.accountTestService.httpUpstream == nil {
		return failure(UpstreamUsageSnapshotStatusFailed, "transport_unavailable", 0)
	}
	if !eligibleUpstreamUsageSnapshotAccount(account) {
		return failure(UpstreamUsageSnapshotStatusFailed, "invalid_account", 0)
	}

	baseURL, err := s.accountTestService.validateUpstreamBaseURL(account.GetCredential("base_url"))
	if err != nil {
		return failure(UpstreamUsageSnapshotStatusFailed, "invalid_base_url", 0)
	}
	proxyURL := ""
	if account.ProxyID != nil {
		if account.Proxy == nil || account.Proxy.ID != *account.ProxyID {
			return failure(UpstreamUsageSnapshotStatusFailed, "proxy_unavailable", 0)
		}
		proxyURL = account.Proxy.URL()
	}

	requestCtx, cancel := context.WithTimeout(ctx, upstreamBillingProbeRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodGet,
		buildOpenAIEndpointURL(baseURL, "/v1/usage"),
		nil,
	)
	if err != nil {
		return failure(UpstreamUsageSnapshotStatusFailed, "request_build_failed", 0)
	}
	reqCtx := WithHTTPUpstreamRedirectsDisabled(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileDefault))
	if account.Platform == PlatformOpenAI {
		reqCtx = WithHTTPUpstreamRedirectsDisabled(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	}
	req = req.WithContext(reqCtx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+account.GetCredential("api_key"))

	var tlsProfile *tlsfingerprint.Profile
	if s.accountTestService.tlsFPProfileService != nil {
		tlsProfile = s.accountTestService.tlsFPProfileService.ResolveTLSProfile(account)
	}
	resp, err := s.accountTestService.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		return failure(UpstreamUsageSnapshotStatusFailed, "request_failed", 0)
	}
	if resp == nil || resp.Body == nil {
		return failure(UpstreamUsageSnapshotStatusFailed, "empty_response", 0)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, upstreamBillingProbeMaxBodyBytes+1))
	if err != nil {
		return failure(UpstreamUsageSnapshotStatusFailed, "response_read_failed", resp.StatusCode)
	}
	if len(body) > upstreamBillingProbeMaxBodyBytes {
		return failure(UpstreamUsageSnapshotStatusFailed, "response_too_large", resp.StatusCode)
	}
	return parseUpstreamUsageSnapshot(resp.StatusCode, body, now)
}

func parseUpstreamUsageSnapshot(statusCode int, body []byte, observedAt time.Time) *UpstreamUsageSnapshot {
	snapshot := &UpstreamUsageSnapshot{
		ObservedAt: observedAt.UTC(),
		HTTPStatus: statusCode,
	}
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		snapshot.Status = UpstreamUsageSnapshotStatusUnauthorized
		snapshot.ErrorCode = "unauthorized"
		return snapshot
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		snapshot.Status = UpstreamUsageSnapshotStatusUnsupported
		snapshot.ErrorCode = "unsupported"
		return snapshot
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		snapshot.Status = UpstreamUsageSnapshotStatusFailed
		if statusCode == http.StatusTooManyRequests {
			snapshot.ErrorCode = "rate_limited"
		} else {
			snapshot.ErrorCode = "http_error"
		}
		return snapshot
	}

	data, err := decodeUpstreamUsageObject(body)
	if err != nil {
		snapshot.Status = UpstreamUsageSnapshotStatusInvalid
		snapshot.ErrorCode = "invalid_response"
		return snapshot
	}
	actualCost, found, valid := upstreamUsageDecimalAt(data, "usage", "total", "actual_cost")
	if !found {
		snapshot.Status = UpstreamUsageSnapshotStatusInvalid
		snapshot.ErrorCode = "missing_actual_cost"
		return snapshot
	}
	if !valid || actualCost.IsNegative() || actualCost.GreaterThanOrEqual(upstreamUsageSnapshotMaxAmount) {
		snapshot.Status = UpstreamUsageSnapshotStatusInvalid
		snapshot.ErrorCode = "invalid_actual_cost"
		return snapshot
	}

	snapshot.Status = UpstreamUsageSnapshotStatusOK
	snapshot.CumulativeActualCost = &actualCost
	if balance, ok := firstUpstreamUsageDecimal(data, []string{"balance"}, []string{"remaining"}); ok &&
		balance.Abs().LessThan(upstreamUsageSnapshotMaxAmount) {
		snapshot.Balance = &balance
	}
	snapshot.DailyUsage = boundedUpstreamUsageJSON(data["daily_usage"])
	snapshot.ModelUsage = boundedUpstreamUsageJSON(data["model_stats"])
	return snapshot
}

func decodeUpstreamUsageObject(body []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var data map[string]any
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("usage response is not an object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("usage response contains trailing data")
	}
	return data, nil
}

func upstreamUsageDecimalAt(data map[string]any, path ...string) (decimal.Decimal, bool, bool) {
	var current any = data
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return decimal.Zero, false, false
		}
		current, ok = object[key]
		if !ok {
			return decimal.Zero, false, false
		}
	}
	number, ok := current.(json.Number)
	if !ok {
		return decimal.Zero, true, false
	}
	value, err := decimal.NewFromString(number.String())
	if err != nil {
		return decimal.Zero, true, false
	}
	floatValue, _ := value.Float64()
	if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
		return decimal.Zero, true, false
	}
	return value, true, true
}

func firstUpstreamUsageDecimal(data map[string]any, paths ...[]string) (decimal.Decimal, bool) {
	for _, path := range paths {
		if value, found, valid := upstreamUsageDecimalAt(data, path...); found && valid {
			return value, true
		}
	}
	return decimal.Zero, false
}

func boundedUpstreamUsageJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) > upstreamUsageSnapshotMaxDetailBytes {
		return nil
	}
	return encoded
}

func reconcileUpstreamUsageCumulative(previous, current *decimal.Decimal) (*decimal.Decimal, string) {
	if current == nil {
		return nil, UpstreamUsageSnapshotStatusInvalid
	}
	if previous == nil {
		return nil, UpstreamUsageSnapshotStatusOK
	}
	if current.LessThan(*previous) {
		return nil, UpstreamUsageSnapshotStatusReset
	}
	delta := current.Sub(*previous)
	return &delta, UpstreamUsageSnapshotStatusOK
}

func (s *UpstreamBillingProbeService) persistUpstreamUsageSnapshot(ctx context.Context, accountID int64, snapshot *UpstreamUsageSnapshot) error {
	if s == nil || s.db == nil || snapshot == nil {
		return ErrUpstreamBillingProbeUnavailable
	}
	if snapshot.Status == UpstreamUsageSnapshotStatusOK && snapshot.CumulativeActualCost != nil {
		var previousText string
		err := s.db.QueryRowContext(ctx, `
			SELECT cumulative_actual_cost::TEXT
			FROM upstream_usage_snapshots
			WHERE account_id = $1
			  AND status IN ('ok', 'reset')
			  AND cumulative_actual_cost IS NOT NULL
			ORDER BY observed_at DESC, id DESC
			LIMIT 1
		`, accountID).Scan(&previousText)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read previous upstream usage snapshot: %w", err)
		}
		if err == nil {
			previous, parseErr := decimal.NewFromString(previousText)
			if parseErr != nil {
				return fmt.Errorf("parse previous upstream usage total: %w", parseErr)
			}
			if _, status := reconcileUpstreamUsageCumulative(&previous, snapshot.CumulativeActualCost); status == UpstreamUsageSnapshotStatusReset {
				snapshot.Status = status
				snapshot.ErrorCode = "counter_reset"
			}
		}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO upstream_usage_snapshots (
			account_id,
			observed_at,
			status,
			http_status,
			cumulative_actual_cost,
			balance,
			daily_usage,
			model_usage,
			error_code
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9)
	`,
		accountID,
		snapshot.ObservedAt,
		snapshot.Status,
		snapshot.HTTPStatus,
		nullableUpstreamUsageDecimal(snapshot.CumulativeActualCost),
		nullableUpstreamUsageDecimal(snapshot.Balance),
		nullableUpstreamUsageJSON(snapshot.DailyUsage),
		nullableUpstreamUsageJSON(snapshot.ModelUsage),
		nullableUpstreamUsageString(snapshot.ErrorCode),
	)
	if err != nil {
		return fmt.Errorf("insert upstream usage snapshot: %w", err)
	}
	return nil
}

func nullableUpstreamUsageDecimal(value *decimal.Decimal) any {
	if value == nil {
		return nil
	}
	return value.String()
}

func nullableUpstreamUsageJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}

func nullableUpstreamUsageString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
