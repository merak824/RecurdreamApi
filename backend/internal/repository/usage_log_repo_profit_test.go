package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProfitMonitorWhereKeepsHalfOpenWindowAndRecordedCostInputs(t *testing.T) {
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	where, args := profitMonitorWhere(start, end, 0, 0, 42, 7, "gpt-test", nil, nil, nil)

	require.Len(t, args, 5)
	require.Equal(t, start, args[0])
	require.Equal(t, end, args[1])
	require.Contains(t, where, "ul.created_at >= $1")
	require.Contains(t, where, "ul.created_at < $2")
	require.Contains(t, where, "ul.account_id = $3")
	require.Contains(t, where, "ul.group_id = $4")
	require.Contains(t, where, "requested_model")
	require.Equal(
		t,
		"COALESCE(ul.account_stats_cost, ul.total_cost) * COALESCE(ul.account_rate_multiplier, 1)",
		profitMonitorCostExpr,
	)
}
