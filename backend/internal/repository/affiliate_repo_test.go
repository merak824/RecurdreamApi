package repository

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAffiliateUserOverviewSQLUsesOnlyUnifiedAvailableQuota(t *testing.T) {
	query := strings.Join(strings.Fields(affiliateUserOverviewSQL), " ")

	require.Contains(t, query, "ua.aff_quota::double precision")
	require.NotContains(t, query, "matured_frozen_quota")
	require.NotContains(t, query, "frozen_until")
}

func TestAffiliateRecordQueriesUseLedgerAuditFields(t *testing.T) {
	source, err := os.ReadFile("affiliate_repo.go")
	require.NoError(t, err)
	content := string(source)

	require.Contains(t, content, "JOIN payment_orders po ON po.id = ual.source_order_id")
	require.Contains(t, content, "ual.amount::double precision")
	require.Contains(t, content, "ual.balance_after::double precision")
	require.NotContains(t, content, "parseAffiliateRebateAmount")
	require.NotContains(t, content, `"current_balance": "u.balance"`)
}

func TestRejectWithdrawalRefundsUnifiedAffiliateQuota(t *testing.T) {
	source, err := os.ReadFile("affiliate_repo.go")
	require.NoError(t, err)
	content := string(source)

	start := strings.Index(content, "func (r *affiliateRepository) RejectWithdrawal")
	require.NotEqual(t, -1, start)
	endOffset := strings.Index(content[start:], "\nfunc (r *affiliateRepository) ListInvitees")
	require.NotEqual(t, -1, endOffset)
	rejectWithdrawal := content[start : start+endOffset]

	require.Contains(t, rejectWithdrawal, "SET aff_quota = aff_quota + $1")
	require.NotContains(t, rejectWithdrawal, "agent_aff_quota")
}
