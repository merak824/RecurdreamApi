package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAITTFTOptimizerMigrationsPreserveAppliedMigration(t *testing.T) {
	columnContent, err := FS.ReadFile("192_openai_ttft_optimizer.sql")
	require.NoError(t, err)

	columnSQL := strings.Join(strings.Fields(string(columnContent)), " ")
	require.Contains(t, columnSQL, "ADD COLUMN IF NOT EXISTS upstream_first_token_ms INTEGER")
	require.Contains(t, columnSQL, "ADD COLUMN IF NOT EXISTS client_disconnected BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, columnSQL, "ADD COLUMN IF NOT EXISTS openai_ttft_context JSONB")
	require.Contains(t, columnSQL, "CREATE INDEX IF NOT EXISTS idx_usage_logs_openai_ttft_hydration")
	require.Contains(t, columnSQL, "CREATE INDEX IF NOT EXISTS idx_usage_logs_openai_first_token_hydration")
	require.NotContains(t, columnSQL, "CREATE INDEX CONCURRENTLY")

	indexContent, err := FS.ReadFile("192a_openai_ttft_optimizer_indexes_notx.sql")
	require.NoError(t, err)

	indexSQL := strings.Join(strings.Fields(string(indexContent)), " ")
	require.Contains(t, indexSQL, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_openai_ttft_hydration")
	require.Contains(t, indexSQL, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_openai_first_token_hydration")
	require.Contains(t, indexSQL, "ON usage_logs (account_id, created_at DESC)")
	require.Contains(t, indexSQL, "WHERE upstream_first_token_ms IS NOT NULL AND client_disconnected = FALSE")
	require.Contains(t, indexSQL, "WHERE first_token_ms IS NOT NULL AND client_disconnected = FALSE AND openai_ttft_context IS NOT NULL")
}
