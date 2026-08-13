package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderOpenAIAccountCandidatesByTTFTKeepsPriorityAsOuterLayer(t *testing.T) {
	accounts := []*Account{
		{ID: 1, Priority: 1},
		{ID: 2, Priority: 1},
		{ID: 3, Priority: 2},
	}
	candidates := make([]openAIAccountCandidateScore, 0, len(accounts))
	for _, account := range accounts {
		candidates = append(candidates, openAIAccountCandidateScore{account: account, loadInfo: &AccountLoadInfo{AccountID: account.ID, LoadRate: 80, WaitingCount: 2}})
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 1, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 500, P90Ms: 7000},
		{AccountID: 2, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 300, P90Ms: 1000},
		{AccountID: 3, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 100, P90Ms: 100},
	}

	ordered := orderOpenAIAccountCandidatesByTTFT(candidates, snapshots, OpenAITTFTTransportHTTPSSE, 5000)
	require.Equal(t, int64(2), ordered[0].account.ID)
	// A lower P90 cannot cross the manual priority boundary.
	require.Equal(t, int64(1), ordered[1].account.ID)
	require.Equal(t, int64(3), ordered[2].account.ID)
}

func TestOrderOpenAIAccountCandidatesByTTFTUsesP90ThenP50AndFallsBackToLoad(t *testing.T) {
	accounts := []*Account{{ID: 1, Priority: 1}, {ID: 2, Priority: 1}, {ID: 3, Priority: 1}}
	candidates := []openAIAccountCandidateScore{
		{account: accounts[0], loadInfo: &AccountLoadInfo{AccountID: 1, LoadRate: 90, WaitingCount: 9}},
		{account: accounts[1], loadInfo: &AccountLoadInfo{AccountID: 2, LoadRate: 10, WaitingCount: 1}},
		{account: accounts[2], loadInfo: &AccountLoadInfo{AccountID: 3, LoadRate: 20, WaitingCount: 2}},
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 1, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 200, P90Ms: 7000},
		{AccountID: 2, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 800, P90Ms: 7000},
		{AccountID: 3, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 500, P90Ms: 7000},
	}

	ordered := orderOpenAIAccountCandidatesByTTFT(candidates, snapshots, OpenAITTFTTransportHTTPSSE, 5000)
	// No account is stable, so the best P90 ties and P50/load decide order.
	require.Equal(t, int64(1), ordered[0].account.ID)
	require.Equal(t, int64(3), ordered[1].account.ID)
	require.Equal(t, int64(2), ordered[2].account.ID)
}

func TestOrderOpenAIAccountCandidatesByTTFTPrefersMatureAccountsWhenNoStableAccountExists(t *testing.T) {
	accounts := []*Account{{ID: 11, Priority: 1}, {ID: 12, Priority: 1}}
	candidates := []openAIAccountCandidateScore{
		{account: accounts[0], loadInfo: &AccountLoadInfo{AccountID: 11, LoadRate: 5, WaitingCount: 0}},
		{account: accounts[1], loadInfo: &AccountLoadInfo{AccountID: 12, LoadRate: 90, WaitingCount: 4}},
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 11, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 1, P50Ms: 100, P90Ms: 100},
		{AccountID: 12, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 9000, P90Ms: 12000},
	}

	ordered := orderOpenAIAccountCandidatesByTTFT(candidates, snapshots, OpenAITTFTTransportHTTPSSE, 10000)
	require.Equal(t, int64(12), ordered[0].account.ID)
	require.Equal(t, int64(11), ordered[1].account.ID)
}

func TestOrderOpenAIAccountCandidatesByTTFTUsesLoadWhenAllAccountsAreImmature(t *testing.T) {
	accounts := []*Account{{ID: 21, Priority: 1}, {ID: 22, Priority: 1}}
	candidates := []openAIAccountCandidateScore{
		{account: accounts[0], loadInfo: &AccountLoadInfo{AccountID: 21, LoadRate: 90, WaitingCount: 4}},
		{account: accounts[1], loadInfo: &AccountLoadInfo{AccountID: 22, LoadRate: 10, WaitingCount: 1}},
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 21, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 1, P50Ms: 100, P90Ms: 100},
		{AccountID: 22, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 2, P50Ms: 900, P90Ms: 900},
	}

	ordered := orderOpenAIAccountCandidatesByTTFT(candidates, snapshots, OpenAITTFTTransportHTTPSSE, 10000)
	require.Equal(t, int64(22), ordered[0].account.ID)
	require.Equal(t, int64(21), ordered[1].account.ID)
}

func TestPromoteOpenAITTFTExplorationCandidateStaysWithinManualPriority(t *testing.T) {
	accounts := []*Account{{ID: 1, Priority: 1}, {ID: 2, Priority: 1}, {ID: 3, Priority: 2}}
	candidates := []openAIAccountCandidateScore{
		{account: accounts[0], loadInfo: &AccountLoadInfo{AccountID: 1, LoadRate: 10}},
		{account: accounts[1], loadInfo: &AccountLoadInfo{AccountID: 2, LoadRate: 20}},
		{account: accounts[2], loadInfo: &AccountLoadInfo{AccountID: 3, LoadRate: 0}},
	}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 1, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 2, P50Ms: 100, P90Ms: 200},
		{AccountID: 2, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 100, P90Ms: 200},
		{AccountID: 3, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 1, P50Ms: 50, P90Ms: 50},
	}

	ordered, explorationID, ok := promoteOpenAITTFTExplorationCandidate(
		candidates,
		snapshots,
		OpenAITTFTTransportHTTPSSE,
		OpenAIAccountScheduleRequest{Streaming: true},
		5,
		1,
	)
	require.True(t, ok)
	require.Equal(t, int64(1), explorationID)
	require.True(t, ordered[0].exploration)
	require.Equal(t, int64(1), ordered[0].account.ID)
	// The lower-priority account cannot be promoted over priority 1.
	for _, candidate := range ordered {
		require.False(t, candidate.account.ID == 3 && candidate.exploration)
	}
}

func TestPromoteOpenAITTFTExplorationCandidateSkipsStickyAndMatureRequests(t *testing.T) {
	candidates := []openAIAccountCandidateScore{{account: &Account{ID: 1, Priority: 1}}}
	snapshots := map[OpenAITTFTWindowKey]OpenAITTFTWindowSnapshot{
		{AccountID: 1, Transport: OpenAITTFTTransportHTTPSSE}: {Count: 10, P50Ms: 100, P90Ms: 100},
	}
	for _, req := range []OpenAIAccountScheduleRequest{
		{Streaming: false},
		{Streaming: true, SessionHash: "sticky"},
		{Streaming: true, PreviousResponseID: "resp_1"},
	} {
		_, _, ok := promoteOpenAITTFTExplorationCandidate(candidates, snapshots, OpenAITTFTTransportHTTPSSE, req, 5, 1)
		require.False(t, ok)
	}
}
