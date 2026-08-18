package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/lib/pq"
)

const profitMonitorReconciliationFreshness = 20 * time.Minute

type upstreamUsageBoundary struct {
	StartCost       *float64
	StartObservedAt *time.Time
	EndCost         *float64
	EndObservedAt   *time.Time
	HasReset        bool
}

// profitMonitorCostExpr only returns a cost for a source whose evidence is
// explicitly known. Official provider accounts have no relay-side upstream
// charge, so their cost is zero. Unknown rows remain NULL.
const profitMonitorCostExpr = "CASE WHEN ul.profit_cost_source = 'official_upstream' THEN 0 WHEN ul.profit_cost_source = 'upstream_probe' AND ul.account_rate_multiplier IS NOT NULL THEN COALESCE(ul.account_stats_cost, ul.total_cost) * ul.account_rate_multiplier WHEN ul.profit_cost_source = 'group_break_even_estimate' THEN ul.actual_cost END"

const profitMonitorConfirmedExpr = "(ul.profit_cost_source = 'official_upstream' OR ul.profit_cost_source = 'group_break_even_estimate' OR (ul.profit_cost_source = 'upstream_probe' AND ul.account_rate_multiplier IS NOT NULL))"

const profitMonitorEligibleExpr = "(COALESCE(ul.actual_cost, 0) > 0 OR COALESCE(ul.total_cost, 0) > 0 OR COALESCE(ul.input_tokens, 0) + COALESCE(ul.output_tokens, 0) + COALESCE(ul.cache_creation_tokens, 0) + COALESCE(ul.cache_read_tokens, 0) > 0)"

// profitMonitorValidAfterExpr is an optional deployment-specific floor for
// profit reporting. It excludes rows whose stored upstream-rate snapshot is
// known to predate reliable capture without rewriting the usage ledger.
const profitMonitorValidAfterExpr = "COALESCE((SELECT NULLIF(s.value, '')::timestamptz FROM settings s WHERE s.key = 'profit_monitor_cost_valid_after'), '-infinity'::timestamptz)"

func profitMonitorWhere(startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) (string, []any) {
	conditions := []string{
		fmt.Sprintf("ul.created_at >= GREATEST($1, %s)", profitMonitorValidAfterExpr),
		"ul.created_at < $2",
		profitMonitorEligibleExpr,
	}
	args := []any{startTime, endTime}
	if userID > 0 {
		conditions = append(conditions, fmt.Sprintf("ul.user_id = $%d", len(args)+1))
		args = append(args, userID)
	}
	if apiKeyID > 0 {
		conditions = append(conditions, fmt.Sprintf("ul.api_key_id = $%d", len(args)+1))
		args = append(args, apiKeyID)
	}
	if accountID > 0 {
		conditions = append(conditions, fmt.Sprintf("ul.account_id = $%d", len(args)+1))
		args = append(args, accountID)
	}
	if groupID > 0 {
		conditions = append(conditions, fmt.Sprintf("ul.group_id = $%d", len(args)+1))
		args = append(args, groupID)
	}
	if model != "" {
		modelExpr := resolveModelDimensionExpressionWithAlias(usagestats.ModelSourceRequested, "ul")
		conditions = append(conditions, fmt.Sprintf("%s = $%d", modelExpr, len(args)+1))
		args = append(args, model)
	}
	if requestType != nil {
		condition, conditionArgs := buildRequestTypeFilterConditionWithAlias(len(args)+1, *requestType, "ul")
		conditions = append(conditions, condition)
		args = append(args, conditionArgs...)
	} else if stream != nil {
		conditions = append(conditions, fmt.Sprintf("ul.stream = $%d", len(args)+1))
		args = append(args, *stream)
	}
	if billingType != nil {
		conditions = append(conditions, fmt.Sprintf("ul.billing_type = $%d", len(args)+1))
		args = append(args, int16(*billingType))
	}
	return "WHERE " + joinConditions(conditions), args
}

func profitMonitorConfirmedWhere(where string) string {
	return where + " AND " + profitMonitorConfirmedExpr
}

func joinConditions(conditions []string) string {
	result := ""
	for i, condition := range conditions {
		if i > 0 {
			result += " AND "
		}
		result += condition
	}
	return result
}

func profitMonitorMargin(sales, profit float64) *float64 {
	if sales == 0 {
		return nil
	}
	margin := profit / sales * 100
	return &margin
}

func profitMonitorReconciliationScopeEligible(userID, apiKeyID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) bool {
	return userID == 0 && apiKeyID == 0 && groupID == 0 && model == "" && requestType == nil && stream == nil && billingType == nil
}

func (r *usageLogRepository) GetProfitMonitor(ctx context.Context, startTime, endTime time.Time, granularity string, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) (*usagestats.ProfitMonitorResponse, error) {
	where, args := profitMonitorWhere(startTime, endTime, userID, apiKeyID, accountID, groupID, model, requestType, stream, billingType)
	result := &usagestats.ProfitMonitorResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Trend:       []usagestats.ProfitMonitorTrendPoint{},
		Groups:      []usagestats.ProfitMonitorDimensionStat{},
		Models:      []usagestats.ProfitMonitorDimensionStat{},
		Accounts:    []usagestats.ProfitMonitorDimensionStat{},
	}

	var sales, cost float64
	var requests, tokens, unknown, upstreamCount, estimateCount, officialCount int64
	summaryQuery := fmt.Sprintf(`
		SELECT COUNT(*) FILTER (WHERE %s),
		       COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens) FILTER (WHERE %s), 0),
		       COALESCE(SUM(ul.actual_cost) FILTER (WHERE %s), 0),
		       COALESCE(SUM(%s) FILTER (WHERE %s), 0),
		       COUNT(*) FILTER (WHERE NOT %s),
		       COUNT(*) FILTER (WHERE ul.profit_cost_source = 'upstream_probe' AND ul.account_rate_multiplier IS NOT NULL),
		       COUNT(*) FILTER (WHERE ul.profit_cost_source = 'group_break_even_estimate'),
		       COUNT(*) FILTER (WHERE ul.profit_cost_source = 'official_upstream')
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		LEFT JOIN accounts a ON a.id = ul.account_id
		%s`, profitMonitorConfirmedExpr, profitMonitorConfirmedExpr, profitMonitorConfirmedExpr, profitMonitorCostExpr, profitMonitorConfirmedExpr, profitMonitorConfirmedExpr, where)
	if err := scanSingleRow(ctx, r.sql, summaryQuery, args, &requests, &tokens, &sales, &cost, &unknown, &upstreamCount, &estimateCount, &officialCount); err != nil {
		return nil, err
	}
	profit := sales - cost
	costSource := "unknown"
	switch {
	case officialCount > 0 && (upstreamCount > 0 || estimateCount > 0):
		costSource = "mixed"
	case officialCount > 0:
		costSource = "official_upstream"
	case upstreamCount > 0 && estimateCount > 0:
		costSource = "mixed"
	case upstreamCount > 0:
		costSource = "upstream_probe"
	case estimateCount > 0:
		costSource = "group_break_even_estimate"
	}
	result.Summary = usagestats.ProfitMonitorSummary{
		Sales:               sales,
		Cost:                cost,
		Profit:              profit,
		MarginPercent:       profitMonitorMargin(sales, profit),
		Requests:            requests,
		Tokens:              tokens,
		UnknownCostCount:    unknown,
		UnverifiedCostCount: unknown,
		CostSource:          costSource,
		VerificationStatus:  "unverified",
	}

	confirmedWhere := profitMonitorConfirmedWhere(where)
	dateFormat := safeDateFormat(granularity)
	trendQuery := fmt.Sprintf(`
		SELECT TO_CHAR(ul.created_at, '%s'),
		       COALESCE(SUM(ul.actual_cost), 0),
		       COALESCE(SUM(%s), 0)
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		LEFT JOIN accounts a ON a.id = ul.account_id
		%s
		GROUP BY 1 ORDER BY 1`, dateFormat, profitMonitorCostExpr, confirmedWhere)
	trendRows, err := r.sql.QueryContext(ctx, trendQuery, args...)
	if err != nil {
		return nil, err
	}
	for trendRows.Next() {
		var row usagestats.ProfitMonitorTrendPoint
		if err := trendRows.Scan(&row.Date, &row.Sales, &row.Cost); err != nil {
			_ = trendRows.Close()
			return nil, err
		}
		row.Profit = row.Sales - row.Cost
		row.MarginPercent = profitMonitorMargin(row.Sales, row.Profit)
		result.Trend = append(result.Trend, row)
	}
	if err := trendRows.Err(); err != nil {
		_ = trendRows.Close()
		return nil, err
	}
	if err := trendRows.Close(); err != nil {
		return nil, err
	}

	groupExpr := "COALESCE(ul.group_id, 0)"
	groupNameExpr := "COALESCE(NULLIF(g.name, ''), '未分组')"
	result.Groups, err = r.getProfitDimension(ctx, confirmedWhere, args, groupExpr, groupNameExpr, "group")
	if err != nil {
		return nil, err
	}
	modelExpr := resolveModelDimensionExpressionWithAlias(usagestats.ModelSourceRequested, "ul")
	result.Models, err = r.getProfitDimension(ctx, confirmedWhere, args, "0", modelExpr, "model")
	if err != nil {
		return nil, err
	}
	accountExpr := "COALESCE(ul.account_id, 0)"
	accountNameExpr := "COALESCE(NULLIF(a.name, ''), CONCAT('账号 #', ul.account_id))"
	result.Accounts, err = r.getProfitDimension(ctx, confirmedWhere, args, accountExpr, accountNameExpr, "account")
	if err != nil {
		return nil, err
	}
	effectiveEnd := endTime
	if now := time.Now(); effectiveEnd.After(now) {
		effectiveEnd = now
	}
	if err := r.attachProfitReconciliation(
		ctx,
		result,
		startTime,
		effectiveEnd,
		profitMonitorReconciliationScopeEligible(userID, apiKeyID, groupID, model, requestType, stream, billingType),
	); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *usageLogRepository) attachProfitReconciliation(
	ctx context.Context,
	result *usagestats.ProfitMonitorResponse,
	startTime, endTime time.Time,
	scopeEligible bool,
) error {
	accountIDs := make([]int64, 0, len(result.Accounts))
	for i := range result.Accounts {
		row := &result.Accounts[i]
		if row.CostSource == "official_upstream" {
			row.ReconciliationStatus = "official_zero"
			continue
		}
		if row.CostSource == "group_break_even_estimate" || row.CostSource == "mixed" {
			row.ReconciliationStatus = "estimated"
			continue
		}
		if row.CostSource != "upstream_probe" || row.ID <= 0 || !scopeEligible {
			row.ReconciliationStatus = "unavailable"
			continue
		}
		accountIDs = append(accountIDs, row.ID)
	}

	boundaries := map[int64]*upstreamUsageBoundary{}
	if len(accountIDs) > 0 {
		var err error
		boundaries, err = r.getUpstreamUsageBoundaries(ctx, accountIDs, startTime, endTime)
		if err != nil {
			return err
		}
	}
	for i := range result.Accounts {
		row := &result.Accounts[i]
		if row.ReconciliationStatus != "" {
			continue
		}
		applyProfitAccountReconciliation(row, boundaries[row.ID], startTime, endTime)
	}
	summarizeProfitReconciliation(&result.Summary, result.Accounts)
	return nil
}

func (r *usageLogRepository) getUpstreamUsageBoundaries(
	ctx context.Context,
	accountIDs []int64,
	startTime, endTime time.Time,
) (map[int64]*upstreamUsageBoundary, error) {
	if len(accountIDs) == 0 {
		return map[int64]*upstreamUsageBoundary{}, nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		WITH requested_accounts AS (
			SELECT unnest($1::bigint[]) AS account_id
		)
		SELECT requested_accounts.account_id,
		       start_snapshot.cumulative_actual_cost,
		       start_snapshot.observed_at,
		       end_snapshot.cumulative_actual_cost,
		       end_snapshot.observed_at,
		       EXISTS (
			       SELECT 1
			       FROM upstream_usage_snapshots reset_snapshot
			       WHERE reset_snapshot.account_id = requested_accounts.account_id
			         AND reset_snapshot.status = 'reset'
			         AND reset_snapshot.observed_at > COALESCE(start_snapshot.observed_at, $2)
			         AND reset_snapshot.observed_at <= $3
		       ) AS has_reset
		FROM requested_accounts
		LEFT JOIN LATERAL (
			SELECT cumulative_actual_cost, observed_at
			FROM upstream_usage_snapshots
			WHERE account_id = requested_accounts.account_id
			  AND status = 'ok'
			  AND observed_at <= $2
			ORDER BY observed_at DESC, id DESC
			LIMIT 1
		) start_snapshot ON TRUE
		LEFT JOIN LATERAL (
			SELECT cumulative_actual_cost, observed_at
			FROM upstream_usage_snapshots
			WHERE account_id = requested_accounts.account_id
			  AND status = 'ok'
			  AND observed_at <= $3
			ORDER BY observed_at DESC, id DESC
			LIMIT 1
		) end_snapshot ON TRUE
	`, pq.Array(accountIDs), startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query upstream usage reconciliation boundaries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64]*upstreamUsageBoundary, len(accountIDs))
	for rows.Next() {
		var accountID int64
		var startCost, endCost sql.NullFloat64
		var startObservedAt, endObservedAt sql.NullTime
		var hasReset bool
		if err := rows.Scan(&accountID, &startCost, &startObservedAt, &endCost, &endObservedAt, &hasReset); err != nil {
			return nil, err
		}
		boundary := &upstreamUsageBoundary{HasReset: hasReset}
		if startCost.Valid {
			value := startCost.Float64
			boundary.StartCost = &value
		}
		if endCost.Valid {
			value := endCost.Float64
			boundary.EndCost = &value
		}
		if startObservedAt.Valid {
			value := startObservedAt.Time
			boundary.StartObservedAt = &value
		}
		if endObservedAt.Valid {
			value := endObservedAt.Time
			boundary.EndObservedAt = &value
		}
		result[accountID] = boundary
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func applyProfitAccountReconciliation(
	row *usagestats.ProfitMonitorDimensionStat,
	boundary *upstreamUsageBoundary,
	startTime, endTime time.Time,
) {
	if row == nil {
		return
	}
	if row.CostSource == "official_upstream" {
		row.ReconciliationStatus = "official_zero"
		return
	}
	if row.CostSource == "group_break_even_estimate" || row.CostSource == "mixed" {
		row.ReconciliationStatus = "estimated"
		return
	}
	if boundary == nil || boundary.StartCost == nil || boundary.EndCost == nil ||
		boundary.StartObservedAt == nil || boundary.EndObservedAt == nil || !endTime.After(startTime) {
		row.ReconciliationStatus = "pending"
		return
	}
	if boundary.HasReset || *boundary.EndCost < *boundary.StartCost {
		row.ReconciliationStatus = "unavailable"
		return
	}
	if boundary.StartObservedAt.After(startTime) || startTime.Sub(*boundary.StartObservedAt) > profitMonitorReconciliationFreshness ||
		boundary.EndObservedAt.After(endTime) || endTime.Sub(*boundary.EndObservedAt) > profitMonitorReconciliationFreshness ||
		!boundary.EndObservedAt.After(*boundary.StartObservedAt) {
		row.ReconciliationStatus = "pending"
		return
	}

	upstreamCost := *boundary.EndCost - *boundary.StartCost
	difference := upstreamCost - row.Cost
	tolerance := math.Max(0.01, math.Abs(upstreamCost)*0.01)
	row.UpstreamActualCost = float64Ptr(upstreamCost)
	row.ReconciliationDifference = float64Ptr(difference)
	if upstreamCost != 0 {
		row.ReconciliationDifferencePercent = float64Ptr(difference / math.Abs(upstreamCost) * 100)
	}
	row.ReconciliationObservedAt = boundary.EndObservedAt.UTC().Format(time.RFC3339)
	if math.Abs(difference) <= tolerance {
		row.ReconciliationStatus = "matched"
	} else {
		row.ReconciliationStatus = "difference"
	}
}

func summarizeProfitReconciliation(summary *usagestats.ProfitMonitorSummary, accounts []usagestats.ProfitMonitorDimensionStat) {
	if summary == nil {
		return
	}
	statusCounts := make(map[string]int)
	var upstreamCost, difference float64
	var reconciled int
	var oldestObserved time.Time
	for i := range accounts {
		row := &accounts[i]
		statusCounts[row.ReconciliationStatus]++
		if row.UpstreamActualCost == nil || row.ReconciliationDifference == nil {
			continue
		}
		upstreamCost += *row.UpstreamActualCost
		difference += *row.ReconciliationDifference
		reconciled++
		if observed, err := time.Parse(time.RFC3339, row.ReconciliationObservedAt); err == nil &&
			(oldestObserved.IsZero() || observed.Before(oldestObserved)) {
			oldestObserved = observed
		}
	}
	if reconciled > 0 {
		summary.UpstreamActualCost = float64Ptr(upstreamCost)
		summary.ReconciliationDifference = float64Ptr(difference)
		if upstreamCost != 0 {
			summary.ReconciliationDifferencePercent = float64Ptr(difference / math.Abs(upstreamCost) * 100)
		}
		if !oldestObserved.IsZero() {
			summary.ReconciliationObservedAt = oldestObserved.UTC().Format(time.RFC3339)
		}
	}
	switch {
	case statusCounts["difference"] > 0:
		summary.ReconciliationStatus = "difference"
	case statusCounts["pending"] > 0:
		summary.ReconciliationStatus = "pending"
	case statusCounts["unavailable"] > 0:
		summary.ReconciliationStatus = "unavailable"
	case statusCounts["estimated"] > 0:
		summary.ReconciliationStatus = "estimated"
	case statusCounts["matched"] > 0:
		summary.ReconciliationStatus = "matched"
	case statusCounts["official_zero"] > 0:
		summary.ReconciliationStatus = "official_zero"
	default:
		summary.ReconciliationStatus = "pending"
	}
}

func float64Ptr(value float64) *float64 {
	return &value
}

func (r *usageLogRepository) getProfitDimension(ctx context.Context, where string, args []any, idExpr, nameExpr, _ string) ([]usagestats.ProfitMonitorDimensionStat, error) {
	groupBy := fmt.Sprintf("%s, %s", idExpr, nameExpr)
	// PostgreSQL interprets GROUP BY 0 as a positional reference. Model rows
	// use a constant ID, so group only by the model expression in that case.
	if idExpr == "0" {
		groupBy = nameExpr
	}
	query := fmt.Sprintf(`
		SELECT %s, %s, COUNT(*),
		       COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0),
		       COALESCE(SUM(ul.actual_cost), 0),
		       COALESCE(SUM(%s), 0),
			CASE WHEN COUNT(*) FILTER (WHERE ul.profit_cost_source = 'official_upstream') > 0
			              AND (COUNT(*) FILTER (WHERE ul.profit_cost_source = 'upstream_probe' AND ul.account_rate_multiplier IS NOT NULL) > 0
			                   OR COUNT(*) FILTER (WHERE ul.profit_cost_source = 'group_break_even_estimate') > 0) THEN 'mixed'
			     WHEN COUNT(*) FILTER (WHERE ul.profit_cost_source = 'official_upstream') > 0 THEN 'official_upstream'
			     WHEN COUNT(*) FILTER (WHERE ul.profit_cost_source = 'upstream_probe' AND ul.account_rate_multiplier IS NOT NULL) > 0
		                  AND COUNT(*) FILTER (WHERE ul.profit_cost_source = 'group_break_even_estimate') > 0 THEN 'mixed'
		            WHEN COUNT(*) FILTER (WHERE ul.profit_cost_source = 'upstream_probe' AND ul.account_rate_multiplier IS NOT NULL) > 0 THEN 'upstream_probe'
		            ELSE 'group_break_even_estimate' END,
		       'unverified'
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		LEFT JOIN accounts a ON a.id = ul.account_id
		%s
		GROUP BY %s ORDER BY 5 DESC`, idExpr, nameExpr, profitMonitorCostExpr, where, groupBy)
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	results := make([]usagestats.ProfitMonitorDimensionStat, 0)
	for rows.Next() {
		var row usagestats.ProfitMonitorDimensionStat
		if err := rows.Scan(&row.ID, &row.Name, &row.Requests, &row.Tokens, &row.Sales, &row.Cost, &row.CostSource, &row.VerificationStatus); err != nil {
			return nil, err
		}
		row.Profit = row.Sales - row.Cost
		row.MarginPercent = profitMonitorMargin(row.Sales, row.Profit)
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
