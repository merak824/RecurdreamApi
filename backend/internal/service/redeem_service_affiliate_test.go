//go:build unit

package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedeemServiceDoesNotIssueAffiliateRebates(t *testing.T) {
	source, err := os.ReadFile("redeem_service.go")
	require.NoError(t, err)

	content := string(source)
	require.NotContains(t, content, "affiliateService")
	require.NotContains(t, content, "tryAccrueAffiliateRebateForRedeem")
	require.NotContains(t, content, "ContextSkipRedeemAffiliate")
	require.NotContains(t, content, "ctxKeySkipRedeemAffiliate")
}
