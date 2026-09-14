package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateProfitMetricsUsesRecordedUpstreamRateCost(t *testing.T) {
	margin := 20.0
	got := calculateProfitMetrics(100, 80, 1000, 3, 1)

	require.Equal(t, 100.0, got.Sales)
	require.Equal(t, 80.0, got.Cost)
	require.Equal(t, 20.0, got.Profit)
	require.Equal(t, &margin, got.MarginPercent)
	require.Equal(t, int64(1000), got.Requests)
	require.Equal(t, int64(3), got.Tokens)
	require.Equal(t, "unverified", got.VerificationStatus)
	require.Equal(t, "usage_record_upstream_rate", got.CostSource)
}

func TestCalculateProfitMetricsOmitsMarginWhenSalesAreZero(t *testing.T) {
	got := calculateProfitMetrics(0, 2, 0, 0, 0)

	require.Equal(t, -2.0, got.Profit)
	require.Nil(t, got.MarginPercent)
}
