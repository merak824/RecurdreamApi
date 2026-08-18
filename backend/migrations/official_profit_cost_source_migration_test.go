package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration226AllowsOfficialProfitCostSource(t *testing.T) {
	content, err := FS.ReadFile("226_allow_official_profit_cost_source.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "official_upstream")
	require.Contains(t, sql, "usage_logs_profit_cost_source_check")
}
