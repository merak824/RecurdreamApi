package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUsageLogProfitCostSnapshotUsesProbeSnapshotWithoutRateSync(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	manualRate := 1.1
	account := &Account{
		RateMultiplier: &manualRate,
		Extra: map[string]any{
			UpstreamBillingProbeEnabledExtraKey: true,
			UpstreamBillingProbeExtraKey: map[string]any{
				"status":      UpstreamBillingProbeStatusOK,
				"data":        map[string]any{"billing_scope": "token", "resolved_rate_multiplier": 2.5, "peak_rate_enabled": false},
				"fresh_until": now.Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	source, rate := usageLogProfitCostSnapshot(account, now)
	require.Equal(t, "upstream_probe", source)
	require.NotNil(t, rate)
	require.Equal(t, 2.5, *rate)
}

func TestUsageLogProfitCostSnapshotTreatsOfficialOAuthAsZeroCost(t *testing.T) {
	account := &Account{Type: AccountTypeOAuth}

	source, rate := usageLogProfitCostSnapshot(account, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))

	require.Equal(t, "official_upstream", source)
	require.Nil(t, rate)
}

func TestUsageLogProfitCostSnapshotTreatsOfficialAPIKeyAsZeroCost(t *testing.T) {
	account := &Account{
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.openai.com/v1"},
	}

	source, rate := usageLogProfitCostSnapshot(account, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))

	require.Equal(t, "official_upstream", source)
	require.Nil(t, rate)
}

func TestUsageLogProfitCostSnapshotKeepsCustomUpstreamAsEstimated(t *testing.T) {
	account := &Account{
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://relay.example/v1"},
	}

	source, rate := usageLogProfitCostSnapshot(account, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))

	require.Equal(t, "group_break_even_estimate", source)
	require.Nil(t, rate)
}

func TestUsageLogProfitCostSnapshotDoesNotTreatExplicitUpstreamTypeAsOfficial(t *testing.T) {
	account := &Account{
		Type:        AccountTypeUpstream,
		Credentials: map[string]any{"base_url": "https://api.openai.com/v1"},
	}

	source, rate := usageLogProfitCostSnapshot(account, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))

	require.Equal(t, "group_break_even_estimate", source)
	require.Nil(t, rate)
}

func TestUsageLogProfitCostSnapshotUsesGroupEstimateWhenProbeSnapshotIsStale(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	manualRate := 1.1
	account := &Account{
		RateMultiplier: &manualRate,
		Extra: map[string]any{
			UpstreamBillingProbeEnabledExtraKey: true,
			UpstreamBillingProbeExtraKey: map[string]any{
				"status":      UpstreamBillingProbeStatusOK,
				"data":        map[string]any{"billing_scope": "token", "resolved_rate_multiplier": 2.5, "peak_rate_enabled": false},
				"fresh_until": now.Add(-time.Minute).Format(time.RFC3339),
			},
		},
	}

	source, rate := usageLogProfitCostSnapshot(account, now)
	require.Equal(t, "group_break_even_estimate", source)
	require.Nil(t, rate)
}

func TestUsageLogProfitCostSnapshotAppliesPeakAtRequestTime(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC) // 20:00 in Shanghai
	account := &Account{
		Extra: map[string]any{
			UpstreamBillingProbeEnabledExtraKey: true,
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

	source, rate := usageLogProfitCostSnapshot(account, now)
	require.Equal(t, "upstream_probe", source)
	require.NotNil(t, rate)
	require.Equal(t, 7.5, *rate)
}

func TestUsageLogProfitCostSnapshotUsesGroupEstimateWhenSnapshotCreatedAfterRequest(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	manualRate := 1.1
	account := &Account{
		RateMultiplier: &manualRate,
		Extra: map[string]any{
			UpstreamBillingProbeEnabledExtraKey: true,
			UpstreamBillingProbeExtraKey: map[string]any{
				"status":      UpstreamBillingProbeStatusOK,
				"data":        map[string]any{"billing_scope": "token", "resolved_rate_multiplier": 2.5, "peak_rate_enabled": false},
				"received_at": now.Add(time.Minute).Format(time.RFC3339),
				"fresh_until": now.Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	source, rate := usageLogProfitCostSnapshot(account, now)
	require.Equal(t, "group_break_even_estimate", source)
	require.Nil(t, rate)
}

func TestUsageLogProfitCostSnapshotUsesBreakEvenForUnsupportedUpstream(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	manualRate := 1.1
	account := &Account{
		RateMultiplier: &manualRate,
		Extra: map[string]any{
			UpstreamBillingProbeEnabledExtraKey: true,
			UpstreamBillingProbeExtraKey: map[string]any{
				"status": UpstreamBillingProbeStatusUnsupported,
			},
		},
	}

	source, rate := usageLogProfitCostSnapshot(account, now)
	require.Equal(t, "group_break_even_estimate", source)
	require.Nil(t, rate)
}

func TestUsageLogProfitCostSnapshotUsesGroupEstimateWhenProbeEvidenceIsUnavailable(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	for name, account := range map[string]*Account{
		"missing snapshot": {Extra: map[string]any{UpstreamBillingProbeEnabledExtraKey: true}},
		"disabled probe": {
			Extra: map[string]any{
				UpstreamBillingProbeEnabledExtraKey: false,
				UpstreamBillingProbeExtraKey:        map[string]any{"status": UpstreamBillingProbeStatusUnsupported},
			},
		},
		"failed probe": {
			Extra: map[string]any{
				UpstreamBillingProbeEnabledExtraKey: true,
				UpstreamBillingProbeExtraKey:        map[string]any{"status": UpstreamBillingProbeStatusFailed},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, rate := usageLogProfitCostSnapshot(account, now)
			require.Equal(t, "group_break_even_estimate", source)
			require.Nil(t, rate)
		})
	}
}

func TestUsageLogProfitCostSnapshotKeepsUnknownWithoutAccountContext(t *testing.T) {
	source, rate := usageLogProfitCostSnapshot(nil, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))
	require.Equal(t, "unknown", source)
	require.Nil(t, rate)
}
