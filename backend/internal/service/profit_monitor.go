package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// ProfitMonitorRepository is optional so repositories and test doubles that do
// not yet implement profit monitoring remain compatible with the dashboard.
type ProfitMonitorRepository interface {
	GetProfitMonitor(
		ctx context.Context,
		startTime, endTime time.Time,
		granularity string,
		userID, apiKeyID, accountID, groupID int64,
		model string,
		requestType *int16,
		stream *bool,
		billingType *int8,
	) (*usagestats.ProfitMonitorResponse, error)
}

// ProfitMetrics is the small, deterministic calculation shared by SQL rows and
// tests. Costs come from the recorded upstream-rate estimate; a negative result
// is a loss.
func calculateProfitMetrics(sales, cost float64, requests, tokens int64, unknownCostCount int64) usagestats.ProfitMonitorSummary {
	profit := sales - cost
	var margin *float64
	if sales != 0 {
		value := profit / sales * 100
		margin = &value
	}
	return usagestats.ProfitMonitorSummary{
		Sales:               sales,
		Cost:                cost,
		Profit:              profit,
		MarginPercent:       margin,
		Requests:            requests,
		Tokens:              tokens,
		UnknownCostCount:    unknownCostCount,
		UnverifiedCostCount: requests,
		CostSource:          "usage_record_upstream_rate",
		VerificationStatus:  "unverified",
	}
}

// GetProfitMonitor returns the data rendered inside the existing admin dashboard.
func (s *DashboardService) GetProfitMonitor(
	ctx context.Context,
	startTime, endTime time.Time,
	granularity string,
	userID, apiKeyID, accountID, groupID int64,
	model string,
	requestType *int16,
	stream *bool,
	billingType *int8,
) (*usagestats.ProfitMonitorResponse, error) {
	repo, ok := s.usageRepo.(ProfitMonitorRepository)
	if !ok {
		return &usagestats.ProfitMonitorResponse{
			Summary: usagestats.ProfitMonitorSummary{
				CostSource:         "unavailable",
				VerificationStatus: "unavailable",
			},
			Trend:    []usagestats.ProfitMonitorTrendPoint{},
			Groups:   []usagestats.ProfitMonitorDimensionStat{},
			Models:   []usagestats.ProfitMonitorDimensionStat{},
			Accounts: []usagestats.ProfitMonitorDimensionStat{},
		}, nil
	}
	return repo.GetProfitMonitor(ctx, startTime, endTime, granularity, userID, apiKeyID, accountID, groupID, model, requestType, stream, billingType)
}
