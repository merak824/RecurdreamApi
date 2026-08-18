package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestProfitMonitorAggregatesConfirmedRowsAndCountsUnknownSeparately(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FILTER .*profit_cost_source.*FROM usage_logs`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"requests", "tokens", "sales", "cost", "unknown", "upstream_count", "estimate_count", "official_count",
		}).AddRow(int64(2), int64(100), 10.0, 8.0, int64(3), int64(1), int64(1), int64(0)))
	mock.ExpectQuery(`(?s)SELECT TO_CHAR.*profit_cost_source.*GROUP BY 1 ORDER BY 1`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"date", "sales", "cost"}).AddRow("2026-08-10", 10.0, 8.0))

	dimensionRows := func(id int64, name string) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "name", "requests", "tokens", "sales", "cost", "cost_source", "verification_status"}).
			AddRow(id, name, int64(2), int64(100), 10.0, 8.0, "mixed", "unverified")
	}
	for _, row := range []*sqlmock.Rows{
		dimensionRows(1, "stable"),
		dimensionRows(0, "gpt-test"),
		dimensionRows(9, "upstream-9"),
	} {
		mock.ExpectQuery(`(?s)SELECT .*profit_cost_source.*FROM usage_logs.*ORDER BY 5 DESC`).
			WithArgs(start, end).
			WillReturnRows(row)
	}

	got, err := repo.GetProfitMonitor(context.Background(), start, end, "day", 0, 0, 0, 0, "", nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), got.Summary.Requests)
	require.Equal(t, int64(3), got.Summary.UnknownCostCount)
	require.Equal(t, int64(3), got.Summary.UnverifiedCostCount)
	require.Equal(t, "mixed", got.Summary.CostSource)
	require.Equal(t, 10.0, got.Summary.Sales)
	require.Equal(t, 8.0, got.Summary.Cost)
	require.Equal(t, 2.0, got.Summary.Profit)
	require.InDelta(t, 20.0, *got.Summary.MarginPercent, 0.0001)
	require.Len(t, got.Trend, 1)
	require.Len(t, got.Groups, 1)
	require.Len(t, got.Models, 1)
	require.Len(t, got.Accounts, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfitMonitorWhereKeepsHalfOpenWindowAndSourceAwareCostInputs(t *testing.T) {
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	where, args := profitMonitorWhere(start, end, 0, 0, 42, 7, "gpt-test", nil, nil, nil)

	require.Len(t, args, 5)
	require.Equal(t, start, args[0])
	require.Equal(t, end, args[1])
	require.Contains(t, where, "ul.created_at >= GREATEST($1")
	require.Contains(t, where, "profit_monitor_cost_valid_after")
	require.Contains(t, where, "ul.created_at < $2")
	require.Contains(t, where, "ul.account_id = $3")
	require.Contains(t, where, "ul.group_id = $4")
	require.Contains(t, where, "requested_model")
	require.Contains(t, profitMonitorConfirmedWhere(where), "profit_cost_source")
	require.Equal(
		t,
		"CASE WHEN ul.profit_cost_source = 'official_upstream' THEN 0 WHEN ul.profit_cost_source = 'upstream_probe' AND ul.account_rate_multiplier IS NOT NULL THEN COALESCE(ul.account_stats_cost, ul.total_cost) * ul.account_rate_multiplier WHEN ul.profit_cost_source = 'group_break_even_estimate' THEN ul.actual_cost END",
		profitMonitorCostExpr,
	)
	require.Contains(t, profitMonitorConfirmedExpr, "upstream_probe")
	require.Contains(t, profitMonitorConfirmedExpr, "group_break_even_estimate")
	require.Contains(t, profitMonitorConfirmedExpr, "official_upstream")
	require.Contains(t, profitMonitorCostExpr, "official_upstream")
}

func TestProfitMonitorReconciliationScopeRejectsPartialAccountViews(t *testing.T) {
	require.True(t, profitMonitorReconciliationScopeEligible(0, 0, 0, "", nil, nil, nil))
	require.False(t, profitMonitorReconciliationScopeEligible(1, 0, 0, "", nil, nil, nil))
	require.False(t, profitMonitorReconciliationScopeEligible(0, 1, 0, "", nil, nil, nil))
	require.False(t, profitMonitorReconciliationScopeEligible(0, 0, 1, "", nil, nil, nil))
	require.False(t, profitMonitorReconciliationScopeEligible(0, 0, 0, "gpt-5", nil, nil, nil))
}
