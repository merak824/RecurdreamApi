package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func float64Pointer(value float64) *float64 { return &value }

func timePointer(value time.Time) *time.Time { return &value }

func TestUpstreamUsageSnapshotReconciliationMatchesWithinTolerance(t *testing.T) {
	start := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	row := &usagestats.ProfitMonitorDimensionStat{ID: 7, Cost: 8, CostSource: "upstream_probe"}
	boundary := &upstreamUsageBoundary{
		StartCost:       float64Pointer(10),
		StartObservedAt: timePointer(start.Add(-5 * time.Minute)),
		EndCost:         float64Pointer(18.005),
		EndObservedAt:   timePointer(end.Add(-5 * time.Minute)),
	}

	applyProfitAccountReconciliation(row, boundary, start, end)

	require.Equal(t, "matched", row.ReconciliationStatus)
	require.NotNil(t, row.UpstreamActualCost)
	require.InDelta(t, 8.005, *row.UpstreamActualCost, 1e-9)
	require.NotNil(t, row.ReconciliationDifference)
	require.InDelta(t, 0.005, *row.ReconciliationDifference, 1e-9)
	require.Equal(t, end.Add(-5*time.Minute).Format(time.RFC3339), row.ReconciliationObservedAt)
}

func TestUpstreamUsageSnapshotReconciliationFlagsDifferenceBeyondTolerance(t *testing.T) {
	start := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	row := &usagestats.ProfitMonitorDimensionStat{ID: 7, Cost: 8, CostSource: "upstream_probe"}
	boundary := &upstreamUsageBoundary{
		StartCost:       float64Pointer(10),
		StartObservedAt: timePointer(start.Add(-5 * time.Minute)),
		EndCost:         float64Pointer(20),
		EndObservedAt:   timePointer(end.Add(-5 * time.Minute)),
	}

	applyProfitAccountReconciliation(row, boundary, start, end)

	require.Equal(t, "difference", row.ReconciliationStatus)
	require.InDelta(t, 10, *row.UpstreamActualCost, 1e-9)
	require.InDelta(t, 2, *row.ReconciliationDifference, 1e-9)
	require.InDelta(t, 20, *row.ReconciliationDifferencePercent, 1e-9)
}

func TestUpstreamUsageSnapshotReconciliationWaitsForFreshBoundarySamples(t *testing.T) {
	start := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	row := &usagestats.ProfitMonitorDimensionStat{ID: 7, Cost: 8, CostSource: "upstream_probe"}
	boundary := &upstreamUsageBoundary{
		StartCost:       float64Pointer(10),
		StartObservedAt: timePointer(start.Add(-30 * time.Minute)),
		EndCost:         float64Pointer(18),
		EndObservedAt:   timePointer(end.Add(-5 * time.Minute)),
	}

	applyProfitAccountReconciliation(row, boundary, start, end)

	require.Equal(t, "pending", row.ReconciliationStatus)
	require.Nil(t, row.UpstreamActualCost)
}

func TestUpstreamUsageSnapshotReconciliationMarksMissingStartSample(t *testing.T) {
	start := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	end := start.Add(8 * time.Hour)
	row := &usagestats.ProfitMonitorDimensionStat{ID: 7, Cost: 8, CostSource: "upstream_probe"}
	boundary := &upstreamUsageBoundary{
		EndCost:           float64Pointer(18),
		EndObservedAt:     timePointer(end.Add(-5 * time.Minute)),
		FirstOKObservedAt: timePointer(start.Add(9 * time.Minute)),
	}

	applyProfitAccountReconciliation(row, boundary, start, end)

	require.Equal(t, "missing_start", row.ReconciliationStatus)
}

func TestNextUpstreamUsageSampleAtRoundsToNextTenMinuteBoundary(t *testing.T) {
	now := time.Date(2026, 8, 19, 7, 26, 42, 0, time.UTC)
	require.Equal(t, time.Date(2026, 8, 19, 7, 30, 0, 0, time.UTC), nextUpstreamUsageSampleAt(now))
}

func TestProfitMonitorSamplingTimesKeepsObservedCadenceAndNeverReturnsPastTime(t *testing.T) {
	now := time.Date(2026, 8, 19, 7, 30, 25, 0, time.UTC)
	lastSample := time.Date(2026, 8, 19, 7, 20, 13, 0, time.UTC)
	boundaries := map[int64]*upstreamUsageBoundary{
		7: {LastAttemptAt: &lastSample},
	}

	lastSampleAt, nextSampleAt := profitMonitorSamplingTimes(now, boundaries)

	require.Equal(t, lastSample.Format(time.RFC3339), lastSampleAt)
	require.Equal(t, time.Date(2026, 8, 19, 7, 40, 13, 0, time.UTC).Format(time.RFC3339), nextSampleAt)
}

func TestProfitReconciliationSummaryPrioritizesMissingStartOverPending(t *testing.T) {
	summary := &usagestats.ProfitMonitorSummary{}
	accounts := []usagestats.ProfitMonitorDimensionStat{
		{ReconciliationStatus: "pending"},
		{ReconciliationStatus: "missing_start"},
	}

	summarizeProfitReconciliation(summary, accounts)

	require.Equal(t, "missing_start", summary.ReconciliationStatus)
}

func TestUpstreamUsageSnapshotReconciliationRejectsResetAndMarksEstimates(t *testing.T) {
	start := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	resetRow := &usagestats.ProfitMonitorDimensionStat{ID: 7, Cost: 8, CostSource: "upstream_probe"}
	boundary := &upstreamUsageBoundary{
		StartCost:       float64Pointer(10),
		StartObservedAt: timePointer(start.Add(-5 * time.Minute)),
		EndCost:         float64Pointer(2),
		EndObservedAt:   timePointer(end.Add(-5 * time.Minute)),
		HasReset:        true,
	}
	applyProfitAccountReconciliation(resetRow, boundary, start, end)
	require.Equal(t, "unavailable", resetRow.ReconciliationStatus)
	require.Nil(t, resetRow.UpstreamActualCost)

	estimatedRow := &usagestats.ProfitMonitorDimensionStat{ID: 8, Cost: 8, CostSource: "group_break_even_estimate"}
	applyProfitAccountReconciliation(estimatedRow, nil, start, end)
	require.Equal(t, "estimated", estimatedRow.ReconciliationStatus)
}

func TestUpstreamUsageSnapshotBoundaryQueryIsBatchedAndDetectsReset(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	start := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	mock.ExpectQuery(`(?s)WITH requested_accounts AS.*unnest\(\$1::bigint\[\]\).*status = 'reset'.*reset_snapshot\.observed_at <= \$3.*LEFT JOIN LATERAL.*status = 'ok'.*observed_at <= \$2.*LEFT JOIN LATERAL.*status = 'ok'.*observed_at <= \$3`).
		WithArgs(sqlmock.AnyArg(), start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"account_id", "start_cost", "start_observed_at", "end_cost", "end_observed_at",
			"first_ok_observed_at", "last_attempt_at", "last_attempt_status", "last_attempt_error", "has_reset",
		}).AddRow(
			int64(7), 10.0, start.Add(-5*time.Minute), 18.0, end.Add(-5*time.Minute),
			start.Add(-5*time.Minute), end.Add(-5*time.Minute), "ok", nil, true,
		))

	boundaries, err := repo.getUpstreamUsageBoundaries(context.Background(), []int64{7}, start, end)
	require.NoError(t, err)
	require.Len(t, boundaries, 1)
	require.True(t, boundaries[7].HasReset)
	require.NoError(t, mock.ExpectationsWereMet())
}
