package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration225CreatesAppendOnlyUpstreamUsageSnapshots(t *testing.T) {
	content, err := FS.ReadFile("225_upstream_usage_snapshots.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS upstream_usage_snapshots")
	require.Contains(t, sql, "account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE")
	require.Contains(t, sql, "cumulative_actual_cost DECIMAL(30, 10)")
	require.Contains(t, sql, "daily_usage JSONB")
	require.Contains(t, sql, "model_usage JSONB")
	require.Contains(t, sql, "upstream_usage_snapshots_account_observed_idx")
	require.NotContains(t, sql, "UPDATE upstream_usage_snapshots")
	require.NotContains(t, sql, "DELETE FROM upstream_usage_snapshots")
}
