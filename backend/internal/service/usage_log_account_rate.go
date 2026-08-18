package service

import "time"

const (
	ProfitCostSourceUpstreamProbe  = "upstream_probe"
	ProfitCostSourceGroupBreakEven = "group_break_even_estimate"
	ProfitCostSourceOfficial       = "official_upstream"
	ProfitCostSourceUnknown        = "unknown"
)

// isOfficialUpstreamAccount distinguishes provider-owned credentials from a
// user-configured relay. OAuth/setup-token accounts are provider credentials;
// API-key accounts are official only when their endpoint host is an official
// provider domain. Explicit upstream accounts always remain relays.
func isOfficialUpstreamAccount(account *Account) bool {
	if account == nil {
		return false
	}
	switch account.Type {
	case AccountTypeOAuth, AccountTypeSetupToken:
		return true
	case AccountTypeUpstream:
		return false
	case AccountTypeAPIKey:
		return upstreamBillingProbeTargetIsOfficialAPI(account.GetCredential("base_url"))
	default:
		return false
	}
}

// usageLogProfitCostSnapshot resolves the immutable source used by profit
// monitoring for a request created at at. It intentionally never falls back to
// the account's manually configured multiplier: that value is for quota and
// scheduling, not evidence of what the upstream charged. When the request has
// an account context but no valid probe evidence, its already-recorded actual
// cost (calculated with the effective group billing multiplier) is treated as a
// break-even estimate rather than silently excluded from profit.
func usageLogProfitCostSnapshot(account *Account, at time.Time) (string, *float64) {
	if account == nil {
		return ProfitCostSourceUnknown, nil
	}
	if isOfficialUpstreamAccount(account) {
		return ProfitCostSourceOfficial, nil
	}
	groupEstimate := func() (string, *float64) {
		return ProfitCostSourceGroupBreakEven, nil
	}
	if !upstreamBillingProbeEnabled(account) {
		return groupEstimate()
	}
	snapshot := decodeUpstreamBillingProbeSnapshot(account.Extra)
	if snapshot == nil {
		return groupEstimate()
	}
	if snapshot.Status == UpstreamBillingProbeStatusUnsupported {
		return groupEstimate()
	}
	if snapshot.Status != UpstreamBillingProbeStatusOK ||
		(snapshot.ReceivedAt != nil && at.Before(snapshot.ReceivedAt.UTC())) ||
		snapshot.FreshUntil == nil || !at.Before(snapshot.FreshUntil.UTC()) {
		return groupEstimate()
	}
	rate, ok := upstreamBillingRateAt(snapshot.Data, at)
	if !ok {
		return groupEstimate()
	}
	return ProfitCostSourceUpstreamProbe, &rate
}
