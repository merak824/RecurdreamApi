//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func querySingleFloat(t *testing.T, ctx context.Context, client *dbent.Client, query string, args ...any) float64 {
	t.Helper()
	rows, err := client.QueryContext(ctx, query, args...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	require.True(t, rows.Next(), "expected one row")
	var value float64
	require.NoError(t, rows.Scan(&value))
	require.NoError(t, rows.Err())
	return value
}

func querySingleInt(t *testing.T, ctx context.Context, client *dbent.Client, query string, args ...any) int {
	t.Helper()
	rows, err := client.QueryContext(ctx, query, args...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	require.True(t, rows.Next(), "expected one row")
	var value int
	require.NoError(t, rows.Scan(&value))
	require.NoError(t, rows.Err())
	return value
}

func TestAffiliateRepository_TransferQuotaToBalance_TransfersSelectedQuota(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateRepository(client, integrationDB)

	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-transfer-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      5.5,
		Concurrency:  5,
	})

	affCode := fmt.Sprintf("AFF%09d", time.Now().UnixNano()%1_000_000_000)
	_, err := client.ExecContext(txCtx, `
INSERT INTO user_affiliates (user_id, aff_code, aff_quota, aff_history_quota, created_at, updated_at)
VALUES ($1, $2, $3, $3, NOW(), NOW())`, u.ID, affCode, 12.34)
	require.NoError(t, err)

	transferred, balance, err := repo.TransferQuotaToBalance(txCtx, u.ID, 10)
	require.NoError(t, err)
	require.InDelta(t, 10.0, transferred, 1e-9)
	require.InDelta(t, 15.5, balance, 1e-9)

	affQuota := querySingleFloat(t, txCtx, client,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", u.ID)
	require.InDelta(t, 2.34, affQuota, 1e-9)

	persistedBalance := querySingleFloat(t, txCtx, client,
		"SELECT balance::double precision FROM users WHERE id = $1", u.ID)
	require.InDelta(t, 15.5, persistedBalance, 1e-9)

	ledgerCount := querySingleInt(t, txCtx, client,
		"SELECT COUNT(*) FROM user_affiliate_ledger WHERE user_id = $1 AND action = 'transfer'", u.ID)
	require.Equal(t, 1, ledgerCount)

	rows, err := client.QueryContext(txCtx, `
SELECT amount::double precision,
       balance_after::double precision,
       aff_quota_after::double precision,
       aff_frozen_quota_after::double precision,
       aff_history_quota_after::double precision
FROM user_affiliate_ledger
WHERE user_id = $1 AND action = 'transfer'
LIMIT 1`, u.ID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	require.True(t, rows.Next(), "expected transfer ledger")
	var amount, balanceAfter, quotaAfter, frozenAfter, historyAfter float64
	require.NoError(t, rows.Scan(&amount, &balanceAfter, &quotaAfter, &frozenAfter, &historyAfter))
	require.InDelta(t, 10.0, amount, 1e-9)
	require.InDelta(t, 15.5, balanceAfter, 1e-9)
	require.InDelta(t, 2.34, quotaAfter, 1e-9)
	require.InDelta(t, 0.0, frozenAfter, 1e-9)
	require.InDelta(t, 12.34, historyAfter, 1e-9)
}

// TestAffiliateRepository_AccrueQuota_ReusesOuterTransaction guards the
// cross-layer tx propagation invariant: when AccrueQuota is called with a ctx
// that already carries a transaction (via dbent.NewTxContext), repo.withTx
// must reuse that tx rather than opening a nested one. If this invariant
// breaks, AccrueQuota would commit independently and survive a rollback of
// the outer tx, which would violate payment_fulfillment's all-or-nothing
// semantics.
func TestAffiliateRepository_AccrueQuota_ReusesOuterTransaction(t *testing.T) {
	ctx := context.Background()

	outerTx, err := integrationEntClient.Tx(ctx)
	require.NoError(t, err, "begin outer tx")
	// Defensive cleanup: if any require.* below fires before the explicit
	// Rollback, this prevents the tx from leaking until container teardown.
	// Rollback is idempotent at the driver level (extra rollback returns an
	// error we ignore).
	t.Cleanup(func() { _ = outerTx.Rollback() })
	client := outerTx.Client()
	txCtx := dbent.NewTxContext(ctx, outerTx)

	inviter := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})
	invitee := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-invitee-%d@example.com", time.Now().UnixNano()+1),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})

	repo := NewAffiliateRepository(client, integrationDB)
	_, err = repo.EnsureUserAffiliate(txCtx, inviter.ID)
	require.NoError(t, err)
	_, err = repo.EnsureUserAffiliate(txCtx, invitee.ID)
	require.NoError(t, err)

	bound, err := repo.BindInviter(txCtx, invitee.ID, inviter.ID)
	require.NoError(t, err)
	require.True(t, bound, "invitee must bind to inviter")

	lotteryPoints := querySingleInt(t, txCtx, client,
		"SELECT lottery_points FROM user_affiliates WHERE user_id = $1", inviter.ID)
	require.Equal(t, 1, lotteryPoints)
	grantCount := querySingleInt(t, txCtx, client,
		"SELECT COUNT(*) FROM invite_lottery_point_ledger WHERE invitee_user_id = $1 AND action = 'invite_grant'", invitee.ID)
	require.Equal(t, 1, grantCount)

	applied, err := repo.AccrueQuota(txCtx, inviter.ID, invitee.ID, 3.5, nil)
	require.NoError(t, err)
	require.True(t, applied, "AccrueQuota must report applied=true")

	// Visible inside the outer tx.
	innerQuota := querySingleFloat(t, txCtx, client,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", inviter.ID)
	require.InDelta(t, 3.5, innerQuota, 1e-9)

	// Roll back the outer tx; if AccrueQuota had opened its own inner tx and
	// committed it, the rows would still be visible to the global client.
	require.NoError(t, outerTx.Rollback())

	rows, err := integrationEntClient.QueryContext(ctx,
		"SELECT COUNT(*) FROM user_affiliates WHERE user_id IN ($1, $2)",
		inviter.ID, invitee.ID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	require.True(t, rows.Next())
	var postRollbackCount int
	require.NoError(t, rows.Scan(&postRollbackCount))
	require.Equal(t, 0, postRollbackCount,
		"AccrueQuota must propagate the outer tx — found persisted rows after rollback")
}

func TestAffiliateRepository_TransferQuotaToBalance_EmptyQuota(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateRepository(client, integrationDB)

	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-empty-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      3.21,
		Concurrency:  5,
	})

	affCode := fmt.Sprintf("AFF%09d", time.Now().UnixNano()%1_000_000_000)
	_, err := client.ExecContext(txCtx, `
INSERT INTO user_affiliates (user_id, aff_code, aff_quota, aff_history_quota, created_at, updated_at)
VALUES ($1, $2, 0, 0, NOW(), NOW())`, u.ID, affCode)
	require.NoError(t, err)

	transferred, balance, err := repo.TransferQuotaToBalance(txCtx, u.ID, 5)
	require.ErrorIs(t, err, service.ErrAffiliateQuotaEmpty)
	require.InDelta(t, 0.0, transferred, 1e-9)
	require.InDelta(t, 0.0, balance, 1e-9)

	persistedBalance := querySingleFloat(t, txCtx, client,
		"SELECT balance::double precision FROM users WHERE id = $1", u.ID)
	require.InDelta(t, 3.21, persistedBalance, 1e-9)
}

// TestAffiliateRepository_ListUsersWithCustomSettings verifies the admin list
// only includes users with at least one override applied.
func TestAffiliateRepository_ListUsersWithCustomSettings(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	repo := NewAffiliateRepository(client, integrationDB)

	// User without any custom config — should NOT appear in the list.
	plainEmail := fmt.Sprintf("affiliate-plain-%d@example.com", time.Now().UnixNano())
	uPlain := mustCreateUser(t, client, &service.User{
		Email: plainEmail, PasswordHash: "hash",
		Role: service.RoleUser, Status: service.StatusActive,
	})
	_, err := repo.EnsureUserAffiliate(txCtx, uPlain.ID)
	require.NoError(t, err)

	// Legacy user with a custom code but no exclusive rate must remain visible
	// so an administrator can explicitly configure a rate or clear the code.
	uCodeOnly := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-codeonly-legacy-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	codeOnly := fmt.Sprintf("LEGACY%06d", time.Now().UnixNano()%1_000_000)
	_, err = client.ExecContext(txCtx, `
INSERT INTO user_affiliates (
    user_id, aff_code, aff_code_custom, aff_rebate_rate_percent,
    withdrawal_enabled, created_at, updated_at
)
VALUES ($1, $2, TRUE, NULL, FALSE, NOW(), NOW())`, uCodeOnly.ID, codeOnly)
	require.NoError(t, err)

	// User with a custom code and required exclusive rate - should appear.
	uCode := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-codeonly-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser, Status: service.StatusActive,
	})
	customCode := fmt.Sprintf("VIP%09d", time.Now().UnixNano()%1_000_000_000)
	customRate := 15.0
	require.NoError(t, repo.UpdateExclusiveSettings(txCtx, uCode.ID, service.AffiliateExclusiveSettingsInput{
		AffCode: &customCode, RebateRatePercent: &customRate,
	}))

	// User with only an exclusive rate — should appear.
	uRate := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-rateonly-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser, Status: service.StatusActive,
	})
	r := 33.3
	require.NoError(t, repo.UpdateExclusiveSettings(txCtx, uRate.ID, service.AffiliateExclusiveSettingsInput{
		RebateRatePercent: &r,
	}))

	entries, total, err := repo.ListUsersWithCustomSettings(txCtx, service.AffiliateAdminFilter{
		Page: 1, PageSize: 100,
	})
	require.NoError(t, err)

	// Build a quick lookup to assert per-user attributes (other tests may have
	// inserted custom rows in the same DB; we only care about our 3).
	byUserID := make(map[int64]service.AffiliateAdminEntry, len(entries))
	for _, e := range entries {
		byUserID[e.UserID] = e
	}

	require.NotContains(t, byUserID, uPlain.ID, "users without overrides must not appear")

	codeOnlyEntry, ok := byUserID[uCodeOnly.ID]
	require.True(t, ok, "legacy custom-code user missing from list")
	require.True(t, codeOnlyEntry.AffCodeCustom)
	require.Nil(t, codeOnlyEntry.AffRebateRatePercent)

	codeEntry, ok := byUserID[uCode.ID]
	require.True(t, ok, "custom-code user missing from list")
	require.True(t, codeEntry.AffCodeCustom)
	require.NotNil(t, codeEntry.AffRebateRatePercent)
	require.InDelta(t, 15.0, *codeEntry.AffRebateRatePercent, 1e-9)

	rateEntry, ok := byUserID[uRate.ID]
	require.True(t, ok, "custom-rate user missing from list")
	require.False(t, rateEntry.AffCodeCustom)
	require.NotNil(t, rateEntry.AffRebateRatePercent)
	require.InDelta(t, 33.3, *rateEntry.AffRebateRatePercent, 1e-9)

	require.GreaterOrEqual(t, total, int64(3), "total must include at least our 3 custom rows")
}

func TestAffiliateRepository_UpdateExclusiveSettings_RollsBackAllFieldsOnCodeConflict(t *testing.T) {
	ctx := context.Background()
	repo := NewAffiliateRepository(integrationEntClient, integrationDB)

	taker := mustCreateUser(t, integrationEntClient, &service.User{
		Email:        fmt.Sprintf("affiliate-exclusive-taker-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	target := mustCreateUser(t, integrationEntClient, &service.User{
		Email:        fmt.Sprintf("affiliate-exclusive-target-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	t.Cleanup(func() {
		_, _ = integrationEntClient.ExecContext(context.Background(),
			"DELETE FROM users WHERE id IN ($1, $2)", taker.ID, target.ID)
	})

	takenCode := fmt.Sprintf("TAKEN%07d", time.Now().UnixNano()%10_000_000)
	takerRate := 10.0
	require.NoError(t, repo.UpdateExclusiveSettings(ctx, taker.ID, service.AffiliateExclusiveSettingsInput{
		AffCode:           &takenCode,
		RebateRatePercent: &takerRate,
	}))

	originalCode := fmt.Sprintf("ORIG%08d", time.Now().UnixNano()%100_000_000)
	originalRate := 25.0
	require.NoError(t, repo.UpdateExclusiveSettings(ctx, target.ID, service.AffiliateExclusiveSettingsInput{
		AffCode:           &originalCode,
		RebateRatePercent: &originalRate,
		WithdrawalEnabled: false,
	}))

	replacementRate := 60.0
	err := repo.UpdateExclusiveSettings(ctx, target.ID, service.AffiliateExclusiveSettingsInput{
		AffCode:           &takenCode,
		RebateRatePercent: &replacementRate,
		WithdrawalEnabled: true,
	})
	require.ErrorIs(t, err, service.ErrAffiliateCodeTaken)

	actual, err := repo.EnsureUserAffiliate(ctx, target.ID)
	require.NoError(t, err)
	require.Equal(t, originalCode, actual.AffCode)
	require.True(t, actual.AffCodeCustom)
	require.NotNil(t, actual.AffRebateRatePercent)
	require.InDelta(t, originalRate, *actual.AffRebateRatePercent, 1e-9)
	require.False(t, actual.WithdrawalEnabled)
}

func TestAffiliateRepository_ClearExclusiveSettings_ResetsCodeRateAndWithdrawalPermission(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()
	repo := NewAffiliateRepository(client, integrationDB)

	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-exclusive-clear-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	customCode := fmt.Sprintf("CLEAR%07d", time.Now().UnixNano()%10_000_000)
	rate := 32.5
	require.NoError(t, repo.UpdateExclusiveSettings(txCtx, u.ID, service.AffiliateExclusiveSettingsInput{
		AffCode:           &customCode,
		RebateRatePercent: &rate,
		WithdrawalEnabled: true,
	}))

	newCode, err := repo.ClearExclusiveSettings(txCtx, u.ID)
	require.NoError(t, err)
	require.NotEmpty(t, newCode)
	require.NotEqual(t, customCode, newCode)

	actual, err := repo.EnsureUserAffiliate(txCtx, u.ID)
	require.NoError(t, err)
	require.Equal(t, newCode, actual.AffCode)
	require.False(t, actual.AffCodeCustom)
	require.Nil(t, actual.AffRebateRatePercent)
	require.False(t, actual.WithdrawalEnabled)
}

func TestAffiliateRepository_CreateWithdrawalEnforcesPermissionInsideTransaction(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()
	repo := NewAffiliateRepository(client, integrationDB)

	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-withdraw-disabled-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	code := fmt.Sprintf("NOWD%08d", time.Now().UnixNano()%100_000_000)
	_, err := client.ExecContext(txCtx, `
INSERT INTO user_affiliates (
    user_id, aff_code, aff_rebate_rate_percent, withdrawal_enabled,
    aff_quota, aff_history_quota, created_at, updated_at
)
VALUES ($1, $2, 30, FALSE, 10, 10, NOW(), NOW())`, u.ID, code)
	require.NoError(t, err)

	_, err = repo.CreateWithdrawal(txCtx, u.ID, 4, "wechat", service.AffiliateImageData{
		DataURL: "data:image/png;base64,iVBORw0KGgo=",
		MIME:    "image/png",
		Size:    8,
	})
	require.ErrorIs(t, err, service.ErrAffiliateWithdrawalForbidden)
	require.InDelta(t, 10.0, querySingleFloat(t, txCtx, client,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", u.ID), 1e-9)
	require.Equal(t, 0, querySingleInt(t, txCtx, client,
		"SELECT COUNT(*) FROM user_affiliate_withdrawals WHERE user_id = $1", u.ID))
}

func TestAffiliateRepository_RejectWithdrawal_ReturnsUnifiedQuotaOnce(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()
	repo := NewAffiliateRepository(client, integrationDB)

	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-withdraw-reject-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	affCode := fmt.Sprintf("WD%010d", time.Now().UnixNano()%10_000_000_000)
	_, err := client.ExecContext(txCtx, `
INSERT INTO user_affiliates (
    user_id, aff_code, aff_rebate_rate_percent, withdrawal_enabled,
    aff_quota, aff_history_quota, created_at, updated_at
)
VALUES ($1, $2, 30, TRUE, 10, 10, NOW(), NOW())`, u.ID, affCode)
	require.NoError(t, err)

	withdrawal, err := repo.CreateWithdrawal(txCtx, u.ID, 4, "wechat", service.AffiliateImageData{
		DataURL: "data:image/png;base64,iVBORw0KGgo=",
		MIME:    "image/png",
		Size:    8,
	})
	require.NoError(t, err)
	require.InDelta(t, 6.0, querySingleFloat(t, txCtx, client,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", u.ID), 1e-9)

	rejected, err := repo.RejectWithdrawal(txCtx, withdrawal.ID, service.AffiliateWithdrawalAdminActionInput{
		RejectReason: "invalid collection code",
	})
	require.NoError(t, err)
	require.Equal(t, service.AffiliateWithdrawalStatusRejected, rejected.Status)
	require.InDelta(t, 10.0, querySingleFloat(t, txCtx, client,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", u.ID), 1e-9)
	require.InDelta(t, 0.0, querySingleFloat(t, txCtx, client,
		"SELECT agent_aff_quota::double precision FROM user_affiliates WHERE user_id = $1", u.ID), 1e-9)

	_, err = repo.RejectWithdrawal(txCtx, withdrawal.ID, service.AffiliateWithdrawalAdminActionInput{})
	require.ErrorIs(t, err, service.ErrAffiliateWithdrawStatus)
	require.InDelta(t, 10.0, querySingleFloat(t, txCtx, client,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", u.ID), 1e-9)
	require.Equal(t, 1, querySingleInt(t, txCtx, client,
		"SELECT COUNT(*) FROM user_affiliate_ledger WHERE user_id = $1 AND action = 'withdraw_reject'", u.ID))
}
