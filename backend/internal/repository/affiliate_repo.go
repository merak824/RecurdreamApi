package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const (
	affiliateCodeLength      = 12
	affiliateCodeMaxAttempts = 12
)

var affiliateCodeCharset = []byte("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")

const affiliateUserOverviewSQL = `
SELECT u.id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       ua.aff_code,
       COALESCE(ua.aff_rebate_rate_percent, 0)::double precision,
       (ua.aff_rebate_rate_percent IS NOT NULL) AS has_custom_rate,
       ua.aff_count,
       COALESCE(rebated.rebated_invitee_count, 0),
       ua.aff_quota::double precision,
       ua.aff_history_quota::double precision
FROM user_affiliates ua
JOIN users u ON u.id = ua.user_id
LEFT JOIN (
    SELECT user_id, COUNT(DISTINCT source_user_id)::integer AS rebated_invitee_count
    FROM user_affiliate_ledger
    WHERE action = 'accrue' AND source_user_id IS NOT NULL
    GROUP BY user_id
) rebated ON rebated.user_id = ua.user_id
WHERE ua.user_id = $1
LIMIT 1`

type affiliateQueryExecer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type affiliateRepository struct {
	client *dbent.Client
}

func NewAffiliateRepository(client *dbent.Client, _ *sql.DB) service.AffiliateRepository {
	return &affiliateRepository{client: client}
}

func (r *affiliateRepository) EnsureUserAffiliate(ctx context.Context, userID int64) (*service.AffiliateSummary, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	client := clientFromContext(ctx, r.client)
	return ensureUserAffiliateWithClient(ctx, client, userID)
}

func (r *affiliateRepository) GetAffiliateByCode(ctx context.Context, code string) (*service.AffiliateSummary, error) {
	client := clientFromContext(ctx, r.client)
	return queryAffiliateByCode(ctx, client, code)
}

func (r *affiliateRepository) BindInviter(ctx context.Context, userID, inviterID int64) (bool, error) {
	var bound bool
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, inviterID); err != nil {
			return err
		}

		res, err := txClient.ExecContext(txCtx,
			"UPDATE user_affiliates SET inviter_id = $1, rebate_mode = 'user', updated_at = NOW() WHERE user_id = $2 AND inviter_id IS NULL",
			inviterID, userID,
		)
		if err != nil {
			return fmt.Errorf("bind inviter: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			bound = false
			return nil
		}

		if _, err = txClient.ExecContext(txCtx, `
WITH granted AS (
    INSERT INTO invite_lottery_point_ledger
        (user_id, action, points, invitee_user_id, business_key, created_at)
    VALUES ($1, 'invite_grant', 1, $2::bigint, 'invite:' || ($2::bigint)::text, NOW())
    ON CONFLICT (business_key) DO NOTHING
    RETURNING points
)
UPDATE user_affiliates
SET aff_count = aff_count + 1,
    lottery_points = lottery_points + COALESCE((SELECT SUM(points) FROM granted), 0),
    updated_at = NOW()
WHERE user_id = $1`, inviterID, userID); err != nil {
			return fmt.Errorf("increment inviter counters: %w", err)
		}
		bound = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return bound, nil
}

func (r *affiliateRepository) AccrueQuota(ctx context.Context, inviterID, inviteeUserID int64, amount float64, sourceOrderID *int64) (bool, error) {
	if amount <= 0 {
		return false, nil
	}
	var applied bool
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		res, err := txClient.ExecContext(txCtx,
			"UPDATE user_affiliates SET aff_quota = aff_quota + $1, aff_history_quota = aff_history_quota + $1, updated_at = NOW() WHERE user_id = $2",
			amount, inviterID)
		if err != nil {
			return err
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			applied = false
			return nil
		}

		if _, err = txClient.ExecContext(txCtx, `
INSERT INTO user_affiliate_ledger (user_id, action, amount, source_user_id, source_order_id, rebate_mode, created_at, updated_at)
VALUES ($1, 'accrue', $2, $3, $4, 'user', NOW(), NOW())`, inviterID, amount, inviteeUserID, nullableInt64Arg(sourceOrderID)); err != nil {
			return fmt.Errorf("insert affiliate accrue ledger: %w", err)
		}

		applied = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return applied, nil
}

func (r *affiliateRepository) TransferQuotaToBalance(ctx context.Context, userID int64, amount float64) (float64, float64, error) {
	if userID <= 0 {
		return 0, 0, service.ErrUserNotFound
	}
	if amount <= 0 {
		return 0, 0, service.ErrAffiliateTransferAmount
	}

	var transferred float64
	var newBalance float64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}

		rows, err := txClient.QueryContext(txCtx, `
SELECT aff_quota::double precision
FROM user_affiliates
WHERE user_id = $1
FOR UPDATE`, userID)
		if err != nil {
			return fmt.Errorf("lock affiliate quota: %w", err)
		}

		if !rows.Next() {
			_ = rows.Close()
			if err := rows.Err(); err != nil {
				return err
			}
			return service.ErrAffiliateProfileNotFound
		}
		var available float64
		if err := rows.Scan(&available); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if available <= 0 {
			return service.ErrAffiliateQuotaEmpty
		}
		if available+1e-9 < amount {
			return service.ErrAffiliateQuotaInsufficient
		}
		transferred = amount

		if _, err = txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_quota = aff_quota - $1,
    updated_at = NOW()
WHERE user_id = $2`, transferred, userID); err != nil {
			return fmt.Errorf("deduct affiliate quota: %w", err)
		}

		affected, err := txClient.User.Update().
			Where(user.IDEQ(userID)).
			AddBalance(transferred).
			AddTotalRecharged(transferred).
			Save(txCtx)
		if err != nil {
			return fmt.Errorf("credit user balance by affiliate quota: %w", err)
		}
		if affected == 0 {
			return service.ErrUserNotFound
		}

		newBalance, err = queryUserBalance(txCtx, txClient, userID)
		if err != nil {
			return err
		}

		snapshot, err := queryAffiliateTransferSnapshot(txCtx, txClient, userID)
		if err != nil {
			return err
		}

		if _, err = txClient.ExecContext(txCtx, `
INSERT INTO user_affiliate_ledger (
    user_id,
    action,
    amount,
    source_user_id,
    balance_after,
    aff_quota_after,
    aff_frozen_quota_after,
    aff_history_quota_after,
    rebate_mode,
    created_at,
    updated_at
)
VALUES ($1, 'transfer', $2, NULL, $3, $4, $5, $6, 'user', NOW(), NOW())`,
			userID,
			transferred,
			snapshot.BalanceAfter,
			snapshot.AvailableQuotaAfter,
			snapshot.FrozenQuotaAfter,
			snapshot.HistoryQuotaAfter,
		); err != nil {
			return fmt.Errorf("insert affiliate transfer ledger: %w", err)
		}

		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	return transferred, newBalance, nil
}

func (r *affiliateRepository) CreateWithdrawal(ctx context.Context, userID int64, amount float64, paymentMethod string, collectionQR service.AffiliateImageData) (*service.AffiliateWithdrawal, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if amount <= 0 {
		return nil, service.ErrAffiliateWithdrawAmount
	}

	var withdrawal *service.AffiliateWithdrawal
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		rows, err := txClient.QueryContext(txCtx, `
SELECT aff_quota::double precision,
       aff_rebate_rate_percent,
       withdrawal_enabled
FROM user_affiliates
WHERE user_id = $1
FOR UPDATE`, userID)
		if err != nil {
			return fmt.Errorf("lock affiliate quota: %w", err)
		}
		if !rows.Next() {
			_ = rows.Close()
			if err := rows.Err(); err != nil {
				return err
			}
			return service.ErrAffiliateProfileNotFound
		}
		var available float64
		var rebateRate sql.NullFloat64
		var withdrawalEnabled bool
		if err := rows.Scan(&available, &rebateRate, &withdrawalEnabled); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if !rebateRate.Valid || !withdrawalEnabled {
			return service.ErrAffiliateWithdrawalForbidden
		}
		if available+1e-9 < amount {
			return service.ErrAffiliateWithdrawQuota
		}

		if _, err = txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_quota = aff_quota - $1,
    updated_at = NOW()
WHERE user_id = $2`, amount, userID); err != nil {
			return fmt.Errorf("deduct affiliate quota: %w", err)
		}

		withdrawal, err = insertAffiliateWithdrawal(txCtx, txClient, userID, amount, paymentMethod, collectionQR)
		if err != nil {
			return err
		}

		if err := insertAffiliateWithdrawalLedger(txCtx, txClient, userID, "withdraw_request", amount); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return withdrawal, nil
}

func (r *affiliateRepository) ListWithdrawalsByUser(ctx context.Context, userID int64, limit int) ([]service.AffiliateWithdrawal, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, affiliateWithdrawalSelectSQL()+`
WHERE w.user_id = $1
ORDER BY w.created_at DESC
LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AffiliateWithdrawal, 0)
	for rows.Next() {
		item, err := scanAffiliateWithdrawal(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *affiliateRepository) GetWithdrawalStats(ctx context.Context, userID int64) (float64, float64, error) {
	if userID <= 0 {
		return 0, 0, service.ErrUserNotFound
	}
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
SELECT COALESCE(SUM(amount) FILTER (WHERE status = 'pending'), 0)::double precision,
       COALESCE(SUM(amount) FILTER (WHERE status = 'paid'), 0)::double precision
FROM user_affiliate_withdrawals
WHERE user_id = $1`, userID)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return 0, 0, rows.Err()
	}
	var pending, paid float64
	if err := rows.Scan(&pending, &paid); err != nil {
		return 0, 0, err
	}
	return pending, paid, rows.Err()
}

func (r *affiliateRepository) ListAffiliateWithdrawalRecords(ctx context.Context, filter service.AffiliateRecordFilter) ([]service.AffiliateWithdrawalRecord, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildAffiliateRecordWhere(filter, "r.created_at", []string{
		"r.user_email", "r.username", "r.user_id::text", "r.status", "r.id::text", "r.destination",
	})
	baseFrom := `
FROM (` + affiliateWithdrawalRecordsUnionSQL() + `) r`

	total, err := queryAffiliateRecordCount(ctx, client, "SELECT COUNT(*) "+baseFrom+" "+where, args...)
	if err != nil {
		return nil, 0, err
	}

	orderBy := buildAffiliateRecordOrderBy(filter, map[string]string{
		"user":         "r.user_email",
		"amount":       "r.amount",
		"destination":  "r.destination",
		"status":       "r.status",
		"created_at":   "r.created_at",
		"processed_at": "r.processed_at",
	}, "r.created_at")
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, `
SELECT r.id,
       r.record_type,
       r.destination,
       r.user_id,
       r.user_email,
       r.username,
       r.amount,
       r.status,
       r.payment_method,
       r.collection_qr_data,
       r.collection_qr_mime,
       r.collection_qr_size,
       r.payment_proof_data,
       r.payment_proof_mime,
       r.payment_proof_size,
       r.reject_reason,
       r.admin_note,
       r.processed_by,
       r.processed_at,
       r.balance_after,
       r.available_quota_after,
       r.frozen_quota_after,
       r.history_quota_after,
       r.snapshot_available,
       r.created_at,
       r.updated_at
`+baseFrom+` `+where+`
`+orderBy+`
LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AffiliateWithdrawalRecord, 0)
	for rows.Next() {
		item, err := scanAffiliateWithdrawalRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *affiliateRepository) MarkWithdrawalPaid(ctx context.Context, withdrawalID int64, input service.AffiliateWithdrawalAdminActionInput, proof service.AffiliateImageData) (*service.AffiliateWithdrawal, error) {
	if withdrawalID <= 0 {
		return nil, service.ErrAffiliateWithdrawMissing
	}
	var withdrawal *service.AffiliateWithdrawal
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		current, err := lockAffiliateWithdrawal(txCtx, txClient, withdrawalID)
		if err != nil {
			return err
		}
		if current.Status != service.AffiliateWithdrawalStatusPending {
			return service.ErrAffiliateWithdrawStatus
		}

		rows, err := txClient.QueryContext(txCtx, `
WITH updated AS (
    UPDATE user_affiliate_withdrawals
    SET status = 'paid',
        payment_proof_data = NULLIF($2, ''),
        payment_proof_mime = NULLIF($3, ''),
        payment_proof_size = NULLIF($4, 0),
        admin_note = NULLIF($5, ''),
        processed_by = $6,
        processed_at = NOW(),
        updated_at = NOW()
    WHERE id = $1
    RETURNING *
)
`+affiliateWithdrawalSelectColumnsSQL()+`
FROM updated w
JOIN users u ON u.id = w.user_id`, withdrawalID, proof.DataURL, proof.MIME, proof.Size, strings.TrimSpace(input.AdminNote), nullableAdminIDArg(input.AdminID))
		if err != nil {
			return fmt.Errorf("mark affiliate withdrawal paid: %w", err)
		}
		if !rows.Next() {
			_ = rows.Close()
			return service.ErrAffiliateWithdrawMissing
		}
		item, err := scanAffiliateWithdrawal(rows)
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		withdrawal = &item
		if err := insertAffiliateWithdrawalLedger(txCtx, txClient, current.UserID, "withdraw_paid", current.Amount); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return withdrawal, nil
}

func (r *affiliateRepository) RejectWithdrawal(ctx context.Context, withdrawalID int64, input service.AffiliateWithdrawalAdminActionInput) (*service.AffiliateWithdrawal, error) {
	if withdrawalID <= 0 {
		return nil, service.ErrAffiliateWithdrawMissing
	}
	var withdrawal *service.AffiliateWithdrawal
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		current, err := lockAffiliateWithdrawal(txCtx, txClient, withdrawalID)
		if err != nil {
			return err
		}
		if current.Status != service.AffiliateWithdrawalStatusPending {
			return service.ErrAffiliateWithdrawStatus
		}

		if _, err = txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_quota = aff_quota + $1,
    updated_at = NOW()
WHERE user_id = $2`, current.Amount, current.UserID); err != nil {
			return fmt.Errorf("return rejected withdrawal quota: %w", err)
		}

		rows, err := txClient.QueryContext(txCtx, `
WITH updated AS (
    UPDATE user_affiliate_withdrawals
    SET status = 'rejected',
        reject_reason = NULLIF($2, ''),
        admin_note = NULLIF($3, ''),
        processed_by = $4,
        processed_at = NOW(),
        updated_at = NOW()
    WHERE id = $1
    RETURNING *
)
`+affiliateWithdrawalSelectColumnsSQL()+`
FROM updated w
JOIN users u ON u.id = w.user_id`, withdrawalID, strings.TrimSpace(input.RejectReason), strings.TrimSpace(input.AdminNote), nullableAdminIDArg(input.AdminID))
		if err != nil {
			return fmt.Errorf("reject affiliate withdrawal: %w", err)
		}
		if !rows.Next() {
			_ = rows.Close()
			return service.ErrAffiliateWithdrawMissing
		}
		item, err := scanAffiliateWithdrawal(rows)
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		withdrawal = &item
		if err := insertAffiliateWithdrawalLedger(txCtx, txClient, current.UserID, "withdraw_reject", current.Amount); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return withdrawal, nil
}

func (r *affiliateRepository) ListInvitees(ctx context.Context, inviterID int64, limit int) ([]service.AffiliateInvitee, error) {
	if limit <= 0 {
		limit = 100
	}
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
SELECT ua.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       ua.created_at,
       COALESCE(SUM(ual.amount), 0)::double precision AS total_rebate
FROM user_affiliates ua
LEFT JOIN users u ON u.id = ua.user_id
LEFT JOIN user_affiliate_ledger ual
       ON ual.user_id = $1
      AND ual.source_user_id = ua.user_id
      AND ual.action = 'accrue'
      AND ual.rebate_mode = ua.rebate_mode
WHERE ua.inviter_id = $1
GROUP BY ua.user_id, u.email, u.username, ua.created_at
ORDER BY ua.created_at DESC
LIMIT $2`, inviterID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	invitees := make([]service.AffiliateInvitee, 0)
	for rows.Next() {
		var item service.AffiliateInvitee
		var createdAt time.Time
		if err := rows.Scan(&item.UserID, &item.Email, &item.Username, &createdAt, &item.TotalRebate); err != nil {
			return nil, err
		}
		item.CreatedAt = &createdAt
		invitees = append(invitees, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invitees, nil
}

func (r *affiliateRepository) ListAffiliateInviteRecords(ctx context.Context, filter service.AffiliateRecordFilter) ([]service.AffiliateInviteRecord, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildAffiliateRecordWhere(filter, "ua.created_at", []string{
		"inviter.email", "inviter.username", "invitee.email", "invitee.username",
		"ua.inviter_id::text", "ua.user_id::text", "inviter_aff.aff_code",
	})

	total, err := queryAffiliateRecordCount(ctx, client, `
SELECT COUNT(*)
FROM user_affiliates ua
JOIN users invitee ON invitee.id = ua.user_id
JOIN users inviter ON inviter.id = ua.inviter_id
JOIN user_affiliates inviter_aff ON inviter_aff.user_id = ua.inviter_id
`+where, args...)
	if err != nil {
		return nil, 0, err
	}

	orderBy := buildAffiliateRecordOrderBy(filter, map[string]string{
		"inviter":      "inviter.email",
		"invitee":      "invitee.email",
		"aff_code":     "inviter_aff.aff_code",
		"total_rebate": "total_rebate",
		"created_at":   "ua.created_at",
	}, "ua.created_at")
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, `
SELECT ua.inviter_id,
       COALESCE(inviter.email, ''),
       COALESCE(inviter.username, ''),
       ua.user_id,
       COALESCE(invitee.email, ''),
       COALESCE(invitee.username, ''),
       COALESCE(inviter_aff.aff_code, ''),
       COALESCE(SUM(ual.amount), 0)::double precision AS total_rebate,
       ua.created_at
FROM user_affiliates ua
JOIN users invitee ON invitee.id = ua.user_id
JOIN users inviter ON inviter.id = ua.inviter_id
JOIN user_affiliates inviter_aff ON inviter_aff.user_id = ua.inviter_id
LEFT JOIN user_affiliate_ledger ual
       ON ual.user_id = ua.inviter_id
      AND ual.source_user_id = ua.user_id
      AND ual.action = 'accrue'
      AND ual.rebate_mode = ua.rebate_mode
`+where+`
GROUP BY ua.inviter_id, inviter.email, inviter.username, ua.user_id, invitee.email, invitee.username, inviter_aff.aff_code, ua.created_at
`+orderBy+`
LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AffiliateInviteRecord, 0)
	for rows.Next() {
		var item service.AffiliateInviteRecord
		if err := rows.Scan(
			&item.InviterID,
			&item.InviterEmail,
			&item.InviterUsername,
			&item.InviteeID,
			&item.InviteeEmail,
			&item.InviteeUsername,
			&item.AffCode,
			&item.TotalRebate,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *affiliateRepository) ListAffiliateRebateRecords(ctx context.Context, filter service.AffiliateRecordFilter) ([]service.AffiliateRebateRecord, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildAffiliateRecordWhere(filter, "ual.created_at", []string{
		"inviter.email", "inviter.username", "invitee.email", "invitee.username",
		"po.id::text", "po.out_trade_no", "po.payment_type", "po.status",
	})
	baseJoin := `
FROM user_affiliate_ledger ual
JOIN payment_orders po ON po.id = ual.source_order_id
JOIN users invitee ON invitee.id = ual.source_user_id
JOIN users inviter ON inviter.id = ual.user_id
WHERE ual.action = 'accrue'
  AND ual.source_order_id IS NOT NULL`
	if where != "" {
		where = strings.Replace(where, "WHERE ", " AND ", 1)
	}

	total, err := queryAffiliateRecordCount(ctx, client, "SELECT COUNT(*) "+baseJoin+where, args...)
	if err != nil {
		return nil, 0, err
	}

	orderBy := buildAffiliateRecordOrderBy(filter, map[string]string{
		"order":         "po.id",
		"inviter":       "inviter.email",
		"invitee":       "invitee.email",
		"order_amount":  "po.amount",
		"pay_amount":    "po.pay_amount",
		"rebate_amount": "ual.amount",
		"payment_type":  "po.payment_type",
		"order_status":  "po.status",
		"created_at":    "ual.created_at",
	}, "ual.created_at")
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, `
SELECT po.id,
       po.out_trade_no,
       ual.user_id,
       COALESCE(inviter.email, ''),
       COALESCE(inviter.username, ''),
       ual.source_user_id,
       COALESCE(invitee.email, ''),
       COALESCE(invitee.username, ''),
       po.amount::double precision,
       po.pay_amount::double precision,
       ual.amount::double precision,
       po.payment_type,
       po.status,
       ual.created_at
`+baseJoin+where+`
`+orderBy+`
LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AffiliateRebateRecord, 0)
	for rows.Next() {
		var item service.AffiliateRebateRecord
		if err := rows.Scan(
			&item.OrderID,
			&item.OutTradeNo,
			&item.InviterID,
			&item.InviterEmail,
			&item.InviterUsername,
			&item.InviteeID,
			&item.InviteeEmail,
			&item.InviteeUsername,
			&item.OrderAmount,
			&item.PayAmount,
			&item.RebateAmount,
			&item.PaymentType,
			&item.OrderStatus,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *affiliateRepository) ListAffiliateTransferRecords(ctx context.Context, filter service.AffiliateRecordFilter) ([]service.AffiliateTransferRecord, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildAffiliateRecordWhere(filter, "ual.created_at", []string{
		"u.email", "u.username", "u.id::text",
	})
	baseJoin := `
FROM user_affiliate_ledger ual
JOIN users u ON u.id = ual.user_id
WHERE ual.action = 'transfer'`
	if where != "" {
		where = strings.Replace(where, "WHERE ", " AND ", 1)
	}

	total, err := queryAffiliateRecordCount(ctx, client, "SELECT COUNT(*) "+baseJoin+where, args...)
	if err != nil {
		return nil, 0, err
	}

	orderBy := buildAffiliateRecordOrderBy(filter, map[string]string{
		"user":                  "u.email",
		"amount":                "ual.amount",
		"balance_after":         "ual.balance_after",
		"available_quota_after": "ual.aff_quota_after",
		"frozen_quota_after":    "ual.aff_frozen_quota_after",
		"history_quota_after":   "ual.aff_history_quota_after",
		"created_at":            "ual.created_at",
	}, "ual.created_at")
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, `
SELECT ual.id,
       ual.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       ual.amount::double precision,
       ual.balance_after::double precision,
       ual.aff_quota_after::double precision,
       ual.aff_frozen_quota_after::double precision,
       ual.aff_history_quota_after::double precision,
       ual.created_at
`+baseJoin+where+`
`+orderBy+`
LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AffiliateTransferRecord, 0)
	for rows.Next() {
		var item service.AffiliateTransferRecord
		var balanceAfter sql.NullFloat64
		var availableQuotaAfter sql.NullFloat64
		var frozenQuotaAfter sql.NullFloat64
		var historyQuotaAfter sql.NullFloat64
		if err := rows.Scan(
			&item.LedgerID,
			&item.UserID,
			&item.UserEmail,
			&item.Username,
			&item.Amount,
			&balanceAfter,
			&availableQuotaAfter,
			&frozenQuotaAfter,
			&historyQuotaAfter,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		item.BalanceAfter = nullableFloat64Ptr(balanceAfter)
		item.AvailableQuotaAfter = nullableFloat64Ptr(availableQuotaAfter)
		item.FrozenQuotaAfter = nullableFloat64Ptr(frozenQuotaAfter)
		item.HistoryQuotaAfter = nullableFloat64Ptr(historyQuotaAfter)
		item.SnapshotAvailable = balanceAfter.Valid &&
			availableQuotaAfter.Valid &&
			frozenQuotaAfter.Valid &&
			historyQuotaAfter.Valid
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *affiliateRepository) GetAffiliateUserOverview(ctx context.Context, userID int64) (*service.AffiliateUserOverview, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, affiliateUserOverviewSQL, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrUserNotFound
	}

	var overview service.AffiliateUserOverview
	var customRate float64
	var hasCustomRate bool
	if err := rows.Scan(
		&overview.UserID,
		&overview.Email,
		&overview.Username,
		&overview.AffCode,
		&customRate,
		&hasCustomRate,
		&overview.InvitedCount,
		&overview.RebatedInviteeCount,
		&overview.AvailableQuota,
		&overview.HistoryQuota,
	); err != nil {
		return nil, err
	}
	if hasCustomRate {
		overview.RebateRatePercent = customRate
		overview.RebateRateCustom = true
	}
	return &overview, rows.Err()
}

func buildAffiliateRecordWhere(filter service.AffiliateRecordFilter, timeColumn string, searchColumns []string) (string, []any) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if filter.StartAt != nil {
		args = append(args, *filter.StartAt)
		clauses = append(clauses, fmt.Sprintf("%s >= $%d", timeColumn, len(args)))
	}
	if filter.EndAt != nil {
		args = append(args, *filter.EndAt)
		clauses = append(clauses, fmt.Sprintf("%s <= $%d", timeColumn, len(args)))
	}
	search := strings.TrimSpace(filter.Search)
	if search != "" && len(searchColumns) > 0 {
		args = append(args, "%"+strings.ToLower(search)+"%")
		parts := make([]string, 0, len(searchColumns))
		for _, col := range searchColumns {
			parts = append(parts, fmt.Sprintf("LOWER(%s) LIKE $%d", col, len(args)))
		}
		clauses = append(clauses, "("+strings.Join(parts, " OR ")+")")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func buildAffiliateRecordOrderBy(filter service.AffiliateRecordFilter, sortColumns map[string]string, fallbackColumn string) string {
	column := sortColumns[filter.SortBy]
	if column == "" {
		column = fallbackColumn
	}
	direction := "DESC"
	if !filter.SortDesc {
		direction = "ASC"
	}
	return "ORDER BY " + column + " " + direction + " NULLS LAST"
}

func queryAffiliateRecordCount(ctx context.Context, client affiliateQueryExecer, query string, args ...any) (int64, error) {
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return 0, rows.Err()
	}
	var total int64
	if err := rows.Scan(&total); err != nil {
		return 0, err
	}
	return total, rows.Err()
}

func (r *affiliateRepository) withTx(ctx context.Context, fn func(txCtx context.Context, txClient *dbent.Client) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx, tx.Client())
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin affiliate transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit affiliate transaction: %w", err)
	}
	return nil
}

func ensureUserAffiliateWithClient(ctx context.Context, client affiliateQueryExecer, userID int64) (*service.AffiliateSummary, error) {
	summary, err := queryAffiliateByUserID(ctx, client, userID)
	if err == nil {
		return summary, nil
	}
	if !errors.Is(err, service.ErrAffiliateProfileNotFound) {
		return nil, err
	}

	for i := 0; i < affiliateCodeMaxAttempts; i++ {
		code, codeErr := generateAffiliateCode()
		if codeErr != nil {
			return nil, codeErr
		}
		_, insertErr := client.ExecContext(ctx, `
INSERT INTO user_affiliates (user_id, aff_code, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING`, userID, code)
		if insertErr == nil {
			break
		}
		if isAffiliateUniqueViolation(insertErr) {
			continue
		}
		return nil, insertErr
	}

	return queryAffiliateByUserID(ctx, client, userID)
}

func queryAffiliateByUserID(ctx context.Context, client affiliateQueryExecer, userID int64) (*service.AffiliateSummary, error) {
	rows, err := client.QueryContext(ctx, `
SELECT user_id,
       aff_code,
       aff_code_custom,
       aff_rebate_rate_percent,
       withdrawal_enabled,
       inviter_id,
       aff_count,
       aff_quota::double precision,
       aff_history_quota::double precision,
       created_at,
       updated_at
FROM user_affiliates
WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAffiliateProfileNotFound
	}

	var out service.AffiliateSummary
	var inviterID sql.NullInt64
	var rebateRate sql.NullFloat64
	if err := rows.Scan(
		&out.UserID,
		&out.AffCode,
		&out.AffCodeCustom,
		&rebateRate,
		&out.WithdrawalEnabled,
		&inviterID,
		&out.AffCount,
		&out.AffQuota,
		&out.AffHistoryQuota,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if inviterID.Valid {
		out.InviterID = &inviterID.Int64
	}
	if rebateRate.Valid {
		v := rebateRate.Float64
		out.AffRebateRatePercent = &v
	}
	return &out, nil
}

func queryAffiliateByCode(ctx context.Context, client affiliateQueryExecer, code string) (*service.AffiliateSummary, error) {
	rows, err := client.QueryContext(ctx, `
SELECT user_id,
       aff_code,
       aff_code_custom,
       aff_rebate_rate_percent,
       withdrawal_enabled,
       inviter_id,
       aff_count,
       aff_quota::double precision,
       aff_history_quota::double precision,
       created_at,
       updated_at
FROM user_affiliates
WHERE aff_code = $1
LIMIT 1`, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAffiliateProfileNotFound
	}

	var out service.AffiliateSummary
	var inviterID sql.NullInt64
	var rebateRate sql.NullFloat64
	if err := rows.Scan(
		&out.UserID,
		&out.AffCode,
		&out.AffCodeCustom,
		&rebateRate,
		&out.WithdrawalEnabled,
		&inviterID,
		&out.AffCount,
		&out.AffQuota,
		&out.AffHistoryQuota,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if inviterID.Valid {
		out.InviterID = &inviterID.Int64
	}
	if rebateRate.Valid {
		v := rebateRate.Float64
		out.AffRebateRatePercent = &v
	}
	return &out, nil
}

func affiliateWithdrawalSelectSQL() string {
	return `
SELECT w.id,
       w.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       w.amount::double precision,
       w.status,
       w.payment_method,
       COALESCE(w.collection_qr_data, ''),
       COALESCE(w.collection_qr_mime, ''),
       COALESCE(w.collection_qr_size, 0),
       COALESCE(w.payment_proof_data, ''),
       COALESCE(w.payment_proof_mime, ''),
       COALESCE(w.payment_proof_size, 0),
       COALESCE(w.reject_reason, ''),
       COALESCE(w.admin_note, ''),
       w.processed_by,
       w.processed_at,
       w.created_at,
       w.updated_at
FROM user_affiliate_withdrawals w
JOIN users u ON u.id = w.user_id
`
}

func affiliateWithdrawalSelectColumnsSQL() string {
	return `
SELECT w.id,
       w.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       w.amount::double precision,
       w.status,
       w.payment_method,
       COALESCE(w.collection_qr_data, ''),
       COALESCE(w.collection_qr_mime, ''),
       COALESCE(w.collection_qr_size, 0),
       COALESCE(w.payment_proof_data, ''),
       COALESCE(w.payment_proof_mime, ''),
       COALESCE(w.payment_proof_size, 0),
       COALESCE(w.reject_reason, ''),
       COALESCE(w.admin_note, ''),
       w.processed_by,
       w.processed_at,
       w.created_at,
       w.updated_at
`
}

func affiliateWithdrawalRecordsUnionSQL() string {
	return `
SELECT ual.id,
       'transfer'::text AS record_type,
       'balance'::text AS destination,
       ual.user_id,
       COALESCE(u.email, '') AS user_email,
       COALESCE(u.username, '') AS username,
       ual.amount::double precision AS amount,
       'completed'::text AS status,
       ''::text AS payment_method,
       ''::text AS collection_qr_data,
       ''::text AS collection_qr_mime,
       0::integer AS collection_qr_size,
       ''::text AS payment_proof_data,
       ''::text AS payment_proof_mime,
       0::integer AS payment_proof_size,
       ''::text AS reject_reason,
       ''::text AS admin_note,
       NULL::bigint AS processed_by,
       ual.created_at AS processed_at,
       ual.balance_after::double precision AS balance_after,
       ual.aff_quota_after::double precision AS available_quota_after,
       ual.aff_frozen_quota_after::double precision AS frozen_quota_after,
       ual.aff_history_quota_after::double precision AS history_quota_after,
       (ual.balance_after IS NOT NULL
            AND ual.aff_quota_after IS NOT NULL
            AND ual.aff_frozen_quota_after IS NOT NULL
            AND ual.aff_history_quota_after IS NOT NULL) AS snapshot_available,
       ual.created_at,
       ual.updated_at
FROM user_affiliate_ledger ual
JOIN users u ON u.id = ual.user_id
WHERE ual.action = 'transfer'
UNION ALL
SELECT w.id,
       'withdrawal'::text AS record_type,
       'alipay_wechat'::text AS destination,
       w.user_id,
       COALESCE(u.email, '') AS user_email,
       COALESCE(u.username, '') AS username,
       w.amount::double precision AS amount,
       w.status,
       w.payment_method,
       COALESCE(w.collection_qr_data, '') AS collection_qr_data,
       COALESCE(w.collection_qr_mime, '') AS collection_qr_mime,
       COALESCE(w.collection_qr_size, 0) AS collection_qr_size,
       COALESCE(w.payment_proof_data, '') AS payment_proof_data,
       COALESCE(w.payment_proof_mime, '') AS payment_proof_mime,
       COALESCE(w.payment_proof_size, 0) AS payment_proof_size,
       COALESCE(w.reject_reason, '') AS reject_reason,
       COALESCE(w.admin_note, '') AS admin_note,
       w.processed_by,
       w.processed_at,
       NULL::double precision AS balance_after,
       NULL::double precision AS available_quota_after,
       NULL::double precision AS frozen_quota_after,
       NULL::double precision AS history_quota_after,
       false AS snapshot_available,
       w.created_at,
       w.updated_at
FROM user_affiliate_withdrawals w
JOIN users u ON u.id = w.user_id`
}

type affiliateWithdrawalScanner interface {
	Scan(dest ...any) error
}

func scanAffiliateWithdrawal(scanner affiliateWithdrawalScanner) (service.AffiliateWithdrawal, error) {
	var item service.AffiliateWithdrawal
	var processedBy sql.NullInt64
	var processedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.UserEmail,
		&item.Username,
		&item.Amount,
		&item.Status,
		&item.PaymentMethod,
		&item.CollectionQRData,
		&item.CollectionQRMIME,
		&item.CollectionQRSize,
		&item.PaymentProofData,
		&item.PaymentProofMIME,
		&item.PaymentProofSize,
		&item.RejectReason,
		&item.AdminNote,
		&processedBy,
		&processedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return item, err
	}
	if processedBy.Valid {
		item.ProcessedBy = &processedBy.Int64
	}
	if processedAt.Valid {
		item.ProcessedAt = &processedAt.Time
	}
	return item, nil
}

func scanAffiliateWithdrawalRecord(scanner affiliateWithdrawalScanner) (service.AffiliateWithdrawalRecord, error) {
	var item service.AffiliateWithdrawalRecord
	var processedBy sql.NullInt64
	var processedAt sql.NullTime
	var balanceAfter sql.NullFloat64
	var availableQuotaAfter sql.NullFloat64
	var frozenQuotaAfter sql.NullFloat64
	var historyQuotaAfter sql.NullFloat64
	if err := scanner.Scan(
		&item.ID,
		&item.RecordType,
		&item.Destination,
		&item.UserID,
		&item.UserEmail,
		&item.Username,
		&item.Amount,
		&item.Status,
		&item.PaymentMethod,
		&item.CollectionQRData,
		&item.CollectionQRMIME,
		&item.CollectionQRSize,
		&item.PaymentProofData,
		&item.PaymentProofMIME,
		&item.PaymentProofSize,
		&item.RejectReason,
		&item.AdminNote,
		&processedBy,
		&processedAt,
		&balanceAfter,
		&availableQuotaAfter,
		&frozenQuotaAfter,
		&historyQuotaAfter,
		&item.SnapshotAvailable,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return item, err
	}
	if processedBy.Valid {
		item.ProcessedBy = &processedBy.Int64
	}
	if processedAt.Valid {
		item.ProcessedAt = &processedAt.Time
	}
	item.BalanceAfter = nullableFloat64Ptr(balanceAfter)
	item.AvailableQuotaAfter = nullableFloat64Ptr(availableQuotaAfter)
	item.FrozenQuotaAfter = nullableFloat64Ptr(frozenQuotaAfter)
	item.HistoryQuotaAfter = nullableFloat64Ptr(historyQuotaAfter)
	return item, nil
}

func insertAffiliateWithdrawal(ctx context.Context, client affiliateQueryExecer, userID int64, amount float64, paymentMethod string, collectionQR service.AffiliateImageData) (*service.AffiliateWithdrawal, error) {
	rows, err := client.QueryContext(ctx, `
WITH inserted AS (
    INSERT INTO user_affiliate_withdrawals (
        user_id,
        amount,
        status,
        payment_method,
        collection_qr_data,
        collection_qr_mime,
        collection_qr_size,
        created_at,
        updated_at
    )
    VALUES ($1, $2, 'pending', $3, $4, $5, $6, NOW(), NOW())
    RETURNING *
)
`+affiliateWithdrawalSelectColumnsSQL()+`
FROM inserted w
JOIN users u ON u.id = w.user_id`, userID, amount, paymentMethod, collectionQR.DataURL, collectionQR.MIME, collectionQR.Size)
	if err != nil {
		return nil, fmt.Errorf("insert affiliate withdrawal: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAffiliateWithdrawMissing
	}
	item, err := scanAffiliateWithdrawal(rows)
	if err != nil {
		return nil, err
	}
	return &item, rows.Err()
}

func lockAffiliateWithdrawal(ctx context.Context, client affiliateQueryExecer, withdrawalID int64) (*service.AffiliateWithdrawal, error) {
	rows, err := client.QueryContext(ctx, affiliateWithdrawalSelectSQL()+`
WHERE w.id = $1
FOR UPDATE OF w`, withdrawalID)
	if err != nil {
		return nil, fmt.Errorf("lock affiliate withdrawal: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAffiliateWithdrawMissing
	}
	item, err := scanAffiliateWithdrawal(rows)
	if err != nil {
		return nil, err
	}
	return &item, rows.Err()
}

func insertAffiliateWithdrawalLedger(ctx context.Context, client affiliateQueryExecer, userID int64, action string, amount float64) error {
	snapshot, err := queryAffiliateTransferSnapshot(ctx, client, userID)
	if err != nil {
		return err
	}
	if _, err = client.ExecContext(ctx, `
INSERT INTO user_affiliate_ledger (
    user_id,
    action,
    amount,
    balance_after,
    aff_quota_after,
    aff_frozen_quota_after,
    aff_history_quota_after,
    rebate_mode,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'user', NOW(), NOW())`,
		userID,
		action,
		amount,
		snapshot.BalanceAfter,
		snapshot.AvailableQuotaAfter,
		snapshot.FrozenQuotaAfter,
		snapshot.HistoryQuotaAfter,
	); err != nil {
		return fmt.Errorf("insert affiliate withdrawal ledger: %w", err)
	}
	return nil
}

func nullableAdminIDArg(adminID int64) any {
	if adminID <= 0 {
		return nil
	}
	return adminID
}

func queryUserBalance(ctx context.Context, client affiliateQueryExecer, userID int64) (float64, error) {
	rows, err := client.QueryContext(ctx,
		"SELECT balance::double precision FROM users WHERE id = $1 LIMIT 1",
		userID,
	)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, service.ErrUserNotFound
	}
	var balance float64
	if err := rows.Scan(&balance); err != nil {
		return 0, err
	}
	return balance, nil
}

type affiliateTransferSnapshot struct {
	BalanceAfter        float64
	AvailableQuotaAfter float64
	FrozenQuotaAfter    float64
	HistoryQuotaAfter   float64
}

func queryAffiliateTransferSnapshot(ctx context.Context, client affiliateQueryExecer, userID int64) (*affiliateTransferSnapshot, error) {
	rows, err := client.QueryContext(ctx, `
SELECT u.balance::double precision,
	   ua.aff_quota::double precision,
	   0::double precision,
	   ua.aff_history_quota::double precision
FROM users u
JOIN user_affiliates ua ON ua.user_id = u.id
WHERE u.id = $1
LIMIT 1`, userID)
	if err != nil {
		return nil, fmt.Errorf("query affiliate transfer snapshot: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrUserNotFound
	}

	var snapshot affiliateTransferSnapshot
	if err := rows.Scan(
		&snapshot.BalanceAfter,
		&snapshot.AvailableQuotaAfter,
		&snapshot.FrozenQuotaAfter,
		&snapshot.HistoryQuotaAfter,
	); err != nil {
		return nil, err
	}
	return &snapshot, rows.Err()
}

func nullableFloat64Ptr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

func generateAffiliateCode() (string, error) {
	buf := make([]byte, affiliateCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate affiliate code: %w", err)
	}
	for i := range buf {
		buf[i] = affiliateCodeCharset[int(buf[i])%len(affiliateCodeCharset)]
	}
	return string(buf), nil
}

func isAffiliateUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505"
	}
	return false
}

func (r *affiliateRepository) UpdateExclusiveSettings(ctx context.Context, userID int64, input service.AffiliateExclusiveSettingsInput) error {
	if userID <= 0 {
		return service.ErrUserNotFound
	}
	if input.RebateRatePercent == nil {
		return service.ErrAffiliateExclusiveRateRequired
	}
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		if input.AffCode == nil {
			res, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_rebate_rate_percent = $1,
    withdrawal_enabled = $2,
    updated_at = NOW()
WHERE user_id = $3`, *input.RebateRatePercent, input.WithdrawalEnabled, userID)
			if err != nil {
				return fmt.Errorf("update exclusive affiliate settings: %w", err)
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return service.ErrUserNotFound
			}
			return nil
		}

		code := strings.ToUpper(strings.TrimSpace(*input.AffCode))
		if code != "" {
			res, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_code = $1,
    aff_code_custom = true,
    aff_rebate_rate_percent = $2,
    withdrawal_enabled = $3,
    updated_at = NOW()
WHERE user_id = $4`, code, *input.RebateRatePercent, input.WithdrawalEnabled, userID)
			if err != nil {
				if isAffiliateUniqueViolation(err) {
					return service.ErrAffiliateCodeTaken
				}
				return fmt.Errorf("update exclusive affiliate settings: %w", err)
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return service.ErrUserNotFound
			}
			return nil
		}

		_, err := resetExclusiveAffiliateCode(txCtx, txClient, userID, input.RebateRatePercent, input.WithdrawalEnabled)
		return err
	})
}

func (r *affiliateRepository) ClearExclusiveSettings(ctx context.Context, userID int64) (string, error) {
	if userID <= 0 {
		return "", service.ErrUserNotFound
	}
	var newCode string
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		var err error
		newCode, err = resetExclusiveAffiliateCode(txCtx, txClient, userID, nil, false)
		return err
	})
	return newCode, err
}

func resetExclusiveAffiliateCode(ctx context.Context, client affiliateQueryExecer, userID int64, rate *float64, withdrawalEnabled bool) (string, error) {
	for i := 0; i < affiliateCodeMaxAttempts; i++ {
		candidate, err := generateAffiliateCode()
		if err != nil {
			return "", err
		}
		res, err := client.ExecContext(ctx, `
UPDATE user_affiliates
SET aff_code = $1,
    aff_code_custom = false,
    aff_rebate_rate_percent = $2,
    withdrawal_enabled = $3,
    updated_at = NOW()
WHERE user_id = $4`, candidate, nullableArg(rate), withdrawalEnabled, userID)
		if err != nil {
			if isAffiliateUniqueViolation(err) {
				continue
			}
			return "", fmt.Errorf("reset exclusive affiliate settings: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return "", service.ErrUserNotFound
		}
		return candidate, nil
	}
	return "", fmt.Errorf("reset exclusive affiliate settings: exhausted attempts")
}

// nullableArg unwraps a *float64 into an interface{} suitable for SQL parameter
// binding: nil pointer → SQL NULL, non-nil → the float value.
func nullableArg(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt64Arg(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

// ListUsersWithCustomSettings 列出有专属配置（自定义码或专属比例）的用户。
//
// 单一查询同时处理"无搜索"与"按邮箱/用户名模糊搜索"：
// 空 search 时拼接出的 LIKE 模式为 "%%"，匹配所有行；非空时按 ILIKE 子串匹配。
// 这避免了为两种情况维护两份 SQL 模板。
func (r *affiliateRepository) ListUsersWithCustomSettings(ctx context.Context, filter service.AffiliateAdminFilter) ([]service.AffiliateAdminEntry, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	likePattern := "%" + strings.TrimSpace(filter.Search) + "%"

	const baseFrom = `
FROM users u
LEFT JOIN user_affiliates ua ON ua.user_id = u.id
WHERE u.deleted_at IS NULL
  AND (
      ua.aff_rebate_rate_percent IS NOT NULL
      OR COALESCE(ua.aff_code_custom, false)
  )
  AND (u.email ILIKE $1 OR u.username ILIKE $1 OR u.id::text ILIKE $1)`

	client := clientFromContext(ctx, r.client)

	total, err := scanInt64(ctx, client, "SELECT COUNT(*)"+baseFrom, likePattern)
	if err != nil {
		return nil, 0, fmt.Errorf("count affiliate admin entries: %w", err)
	}

	listQuery := `
SELECT u.id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       COALESCE(u.role, ''),
       COALESCE(ua.aff_code, ''),
       COALESCE(ua.aff_code_custom, false),
       ua.aff_rebate_rate_percent,
       COALESCE(ua.withdrawal_enabled, false),
       COALESCE(ua.aff_count, 0)` + baseFrom + `
ORDER BY ua.updated_at DESC NULLS LAST,
         u.created_at DESC
LIMIT $2 OFFSET $3`

	rows, err := client.QueryContext(ctx, listQuery, likePattern, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list affiliate admin entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	entries := make([]service.AffiliateAdminEntry, 0)
	for rows.Next() {
		var e service.AffiliateAdminEntry
		var rebate sql.NullFloat64
		if err := rows.Scan(&e.UserID, &e.Email, &e.Username, &e.Role, &e.AffCode,
			&e.AffCodeCustom, &rebate, &e.WithdrawalEnabled, &e.AffCount); err != nil {
			return nil, 0, err
		}
		if rebate.Valid {
			v := rebate.Float64
			e.AffRebateRatePercent = &v
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

// scanInt64 runs a query expected to return a single int64 column (e.g. COUNT).
func scanInt64(ctx context.Context, client affiliateQueryExecer, query string, args ...any) (int64, error) {
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, nil
	}
	var v int64
	if err := rows.Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}
