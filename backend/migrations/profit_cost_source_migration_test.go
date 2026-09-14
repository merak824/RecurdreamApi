package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration224AddsConstrainedProfitCostSource(t *testing.T) {
	content, err := FS.ReadFile("224_usage_log_profit_cost_source.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS profit_cost_source TEXT NOT NULL DEFAULT 'unknown'")
	require.Contains(t, sql, "'upstream_probe'")
	require.Contains(t, sql, "'group_break_even_estimate'")
	require.Contains(t, sql, "'unknown'")
}
