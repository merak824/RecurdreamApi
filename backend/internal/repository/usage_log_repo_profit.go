package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// profitMonitorCostExpr uses the upstream rate snapshot recorded on each usage
// row. A missing snapshot falls back to a neutral multiplier of 1.0; recorded
// discounts and zero-cost rates remain valid values.
const profitMonitorCostExpr = "COALESCE(ul.account_stats_cost, ul.total_cost) * COALESCE(ul.account_rate_multiplier, 1)"

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
	var requests, tokens, unknown int64
	summaryQuery := fmt.Sprintf(`
		SELECT COUNT(*),
		       COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0),
		       COALESCE(SUM(ul.actual_cost), 0),
		       COALESCE(SUM(%s), 0),
		       COALESCE(SUM(CASE WHEN ul.account_rate_multiplier IS NULL THEN 1 ELSE 0 END), 0)
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		LEFT JOIN accounts a ON a.id = ul.account_id
		%s`, profitMonitorCostExpr, where)
	if err := scanSingleRow(ctx, r.sql, summaryQuery, args, &requests, &tokens, &sales, &cost, &unknown); err != nil {
		return nil, err
	}
	profit := sales - cost
	result.Summary = usagestats.ProfitMonitorSummary{
		Sales:               sales,
		Cost:                cost,
		Profit:              profit,
		MarginPercent:       profitMonitorMargin(sales, profit),
		Requests:            requests,
		Tokens:              tokens,
		UnknownCostCount:    unknown,
		UnverifiedCostCount: requests,
		CostSource:          "usage_record_upstream_rate",
		VerificationStatus:  "unverified",
	}

	dateFormat := safeDateFormat(granularity)
	trendQuery := fmt.Sprintf(`
		SELECT TO_CHAR(ul.created_at, '%s'),
		       COALESCE(SUM(ul.actual_cost), 0),
		       COALESCE(SUM(%s), 0)
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		LEFT JOIN accounts a ON a.id = ul.account_id
		%s
		GROUP BY 1 ORDER BY 1`, dateFormat, profitMonitorCostExpr, where)
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
	result.Groups, err = r.getProfitDimension(ctx, where, args, groupExpr, groupNameExpr, "group")
	if err != nil {
		return nil, err
	}
	modelExpr := resolveModelDimensionExpressionWithAlias(usagestats.ModelSourceRequested, "ul")
	result.Models, err = r.getProfitDimension(ctx, where, args, "0", modelExpr, "model")
	if err != nil {
		return nil, err
	}
	accountExpr := "COALESCE(ul.account_id, 0)"
	accountNameExpr := "COALESCE(NULLIF(a.name, ''), CONCAT('账号 #', ul.account_id))"
	result.Accounts, err = r.getProfitDimension(ctx, where, args, accountExpr, accountNameExpr, "account")
	if err != nil {
		return nil, err
	}
	return result, nil
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
		       CASE WHEN SUM(CASE WHEN ul.account_stats_cost IS NOT NULL THEN 1 ELSE 0 END) = COUNT(*) THEN 'channel_pricing'
		            WHEN SUM(CASE WHEN ul.account_stats_cost IS NOT NULL THEN 1 ELSE 0 END) > 0 THEN 'mixed'
		            ELSE 'legacy_formula' END,
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
	defer rows.Close()
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
