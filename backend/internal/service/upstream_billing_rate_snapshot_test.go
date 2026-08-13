package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUsageLogAccountRateMultiplierUsesProbeSnapshotWithoutRateSync(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	manualRate := 1.1
	account := &Account{
		RateMultiplier: &manualRate,
		Extra: map[string]any{
			UpstreamBillingProbeExtraKey: map[string]any{
				"status":      UpstreamBillingProbeStatusOK,
				"data":        map[string]any{"billing_scope": "token", "resolved_rate_multiplier": 2.5, "peak_rate_enabled": false},
				"fresh_until": now.Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	require.Equal(t, 2.5, usageLogAccountRateMultiplier(account, now))
}

func TestUsageLogAccountRateMultiplierFallsBackWhenProbeSnapshotIsStale(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	manualRate := 1.1
	account := &Account{
		RateMultiplier: &manualRate,
		Extra: map[string]any{
			UpstreamBillingProbeExtraKey: map[string]any{
				"status":      UpstreamBillingProbeStatusOK,
				"data":        map[string]any{"billing_scope": "token", "resolved_rate_multiplier": 2.5, "peak_rate_enabled": false},
				"fresh_until": now.Add(-time.Minute).Format(time.RFC3339),
			},
		},
	}

	require.Equal(t, manualRate, usageLogAccountRateMultiplier(account, now))
}

func TestUsageLogAccountRateMultiplierAppliesPeakAtRequestTime(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC) // 20:00 in Shanghai
	account := &Account{
		Extra: map[string]any{
			UpstreamBillingProbeExtraKey: map[string]any{
				"status": UpstreamBillingProbeStatusOK,
				"data": map[string]any{
					"billing_scope":            "token",
					"resolved_rate_multiplier": 2.5,
					"peak_rate_enabled":        true,
					"peak_start":               "20:00",
					"peak_end":                 "22:00",
					"peak_rate_multiplier":     3.0,
					"timezone":                 "Asia/Shanghai",
				},
				"fresh_until": now.Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	require.Equal(t, 7.5, usageLogAccountRateMultiplier(account, now))
}

func TestUsageLogAccountRateMultiplierIgnoresSnapshotCreatedAfterRequest(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	manualRate := 1.1
	account := &Account{
		RateMultiplier: &manualRate,
		Extra: map[string]any{
			UpstreamBillingProbeExtraKey: map[string]any{
				"status":      UpstreamBillingProbeStatusOK,
				"data":        map[string]any{"billing_scope": "token", "resolved_rate_multiplier": 2.5, "peak_rate_enabled": false},
				"received_at": now.Add(time.Minute).Format(time.RFC3339),
				"fresh_until": now.Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	require.Equal(t, manualRate, usageLogAccountRateMultiplier(account, now))
}
