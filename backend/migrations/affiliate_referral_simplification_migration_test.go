package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration193SimplifiesAffiliateReferrals(t *testing.T) {
	content, err := FS.ReadFile("193_simplify_affiliate_referrals.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS withdrawal_enabled BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20)")
	require.Contains(t, sql, "aff_quota + aff_frozen_quota + agent_aff_quota + agent_aff_frozen_quota")
	require.Contains(t, sql, "aff_history_quota + agent_aff_history_quota")
	require.Contains(t, sql, "WHERE u.role = 'agent'")
	require.Contains(t, sql, "SET role = 'user'")
	require.Contains(t, sql, "withdrawal_enabled = TRUE")
	require.Contains(t, sql, "CHECK (payment_method IN ('wechat', 'alipay'))")
}
