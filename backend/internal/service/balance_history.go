package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	defaultBalanceHistoryPageSize = 20
	maxBalanceHistoryPageSize     = 100
)

// BalanceHistoryItem is a read-only projection of an existing balance source.
// Its ID is namespaced because source tables use independent numeric sequences.
type BalanceHistoryItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	OccurredAt  time.Time `json:"occurred_at"`
	Reference   string    `json:"reference"`
	Description string    `json:"description"`
}

// NormalizeBalanceHistoryPagination returns the effective pagination used by
// balance-history queries so response metadata matches the database query.
func NormalizeBalanceHistoryPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultBalanceHistoryPageSize
	}
	if pageSize > maxBalanceHistoryPageSize {
		pageSize = maxBalanceHistoryPageSize
	}
	return page, pageSize
}

func isBalanceHistoryTypeAllowed(entryType string) bool {
	switch entryType {
	case "", RedeemTypeBalance, "admin_balance", RedeemTypeAffiliateBalance, RedeemTypeRedPacketReward:
		return true
	default:
		return false
	}
}

// GetUserBalanceHistory aggregates existing source ledgers without creating a
// second accounting record. Usage charges remain available through usage logs.
func (s *RedeemService) GetUserBalanceHistory(
	ctx context.Context,
	userID int64,
	page int,
	pageSize int,
	entryType string,
) ([]BalanceHistoryItem, int64, error) {
	entryType = strings.TrimSpace(entryType)
	if userID <= 0 || !isBalanceHistoryTypeAllowed(entryType) {
		return []BalanceHistoryItem{}, 0, nil
	}
	if s == nil || s.entClient == nil {
		return nil, 0, fmt.Errorf("balance history database is unavailable")
	}

	page, pageSize = NormalizeBalanceHistoryPagination(page, pageSize)
	offset := (page - 1) * pageSize
	rows, err := s.entClient.QueryContext(ctx, `
WITH balance_history AS (
    SELECT 'redeem:' || id::text AS id,
           type,
           value::double precision AS amount,
           COALESCE(used_at, created_at) AS occurred_at,
           code AS reference,
           CASE WHEN type = 'admin_balance' THEN COALESCE(notes, '') ELSE '' END AS description
    FROM redeem_codes
    WHERE used_by = $1
      AND type IN ('balance', 'admin_balance')

    UNION ALL

    SELECT 'affiliate:' || id::text AS id,
           'affiliate_balance' AS type,
           amount::double precision AS amount,
           created_at AS occurred_at,
           '' AS reference,
           '' AS description
    FROM user_affiliate_ledger
    WHERE user_id = $1
      AND action = 'transfer'

    UNION ALL

    SELECT 'red-packet:' || ledger.id::text AS id,
           'red_packet_reward' AS type,
           (ledger.amount_cents::double precision / 100) AS amount,
           ledger.created_at AS occurred_at,
           activity.period_no::text AS reference,
           activity.name AS description
    FROM red_packet_reward_ledger ledger
    JOIN red_packet_activities activity ON activity.id = ledger.activity_id
    WHERE ledger.user_id = $1
)
SELECT id,
       type,
       amount,
       occurred_at,
       reference,
       description,
       COUNT(*) OVER() AS total_count
FROM balance_history
WHERE ($2 = '' OR type = $2)
ORDER BY occurred_at DESC, id DESC
LIMIT $3 OFFSET $4`, userID, entryType, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query user balance history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]BalanceHistoryItem, 0, pageSize)
	var total int64
	for rows.Next() {
		var item BalanceHistoryItem
		if err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.Amount,
			&item.OccurredAt,
			&item.Reference,
			&item.Description,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user balance history: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate user balance history: %w", err)
	}
	return items, total, nil
}
