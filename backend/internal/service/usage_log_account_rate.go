package service

import "time"

// usageLogAccountRateMultiplier returns the upstream-declared rate that was
// valid when a usage record was created. The account's rate-sync switch only
// controls whether that declaration is written back to the account for quota
// and scheduling; it must not decide what the usage record records.
func usageLogAccountRateMultiplier(account *Account, at time.Time) float64 {
	if account == nil {
		return 1
	}
	if snapshot := decodeUpstreamBillingProbeSnapshot(account.Extra); snapshot != nil &&
		snapshot.Status == UpstreamBillingProbeStatusOK &&
		(snapshot.ReceivedAt == nil || !at.Before(snapshot.ReceivedAt.UTC())) &&
		snapshot.FreshUntil != nil &&
		at.Before(snapshot.FreshUntil.UTC()) {
		if rate, ok := upstreamBillingRateAt(snapshot.Data, at); ok {
			return rate
		}
	}
	return account.BillingRateMultiplier()
}
