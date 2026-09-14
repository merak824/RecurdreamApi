package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type redPacketRepositoryStub struct {
	createCalls       int
	createdDraft      RedPacketDraft
	activityDetail    *RedPacketActivityDetail
	participateCalls  int
	participateSeed   []byte
	participateResult *RedPacketParticipationResult
	drawJobs          []RedPacketDrawJob
	settled           []RedPacketWinnerAllocation
}

func (s *redPacketRepositoryStub) GetCurrent(context.Context, int64) (*RedPacketActivity, error) {
	return nil, nil
}

func (s *redPacketRepositoryStub) GetActivity(context.Context, int64, int64) (*RedPacketActivityDetail, error) {
	return s.activityDetail, nil
}

func (s *redPacketRepositoryStub) GetEligibility(context.Context, int64) (*RedPacketEligibility, error) {
	return nil, nil
}

func (s *redPacketRepositoryStub) ListRecent(context.Context, int64, int) ([]RedPacketActivity, error) {
	return nil, nil
}

func (s *redPacketRepositoryStub) ListRewards(context.Context, int64, int) ([]RedPacketReward, error) {
	return nil, nil
}

func (s *redPacketRepositoryStub) Participate(_ context.Context, _, _ int64, seed []byte) (*RedPacketParticipationResult, error) {
	s.participateCalls++
	s.participateSeed = append([]byte(nil), seed...)
	return s.participateResult, nil
}

func (s *redPacketRepositoryStub) CreateDraft(_ context.Context, _ int64, draft RedPacketDraft) (*RedPacketActivity, error) {
	s.createCalls++
	s.createdDraft = draft
	return &RedPacketActivity{ID: 1, Name: draft.Name}, nil
}

func (s *redPacketRepositoryStub) UpdateDraft(context.Context, int64, RedPacketDraft) (*RedPacketActivity, error) {
	return nil, nil
}

func (s *redPacketRepositoryStub) Publish(context.Context, int64, int64) (*RedPacketActivity, error) {
	return nil, nil
}

func (s *redPacketRepositoryStub) Cancel(context.Context, int64, int64) error { return nil }

func (s *redPacketRepositoryStub) ListAdmin(context.Context, int, int) (*RedPacketAdminPage, error) {
	return nil, nil
}

func (s *redPacketRepositoryStub) ListDrawing(context.Context, int) ([]RedPacketDrawJob, error) {
	return s.drawJobs, nil
}

func (s *redPacketRepositoryStub) Settle(_ context.Context, _ RedPacketDrawJob, winners []RedPacketWinnerAllocation) error {
	s.settled = append([]RedPacketWinnerAllocation(nil), winners...)
	return nil
}

func TestValidateRedPacketDraftFixedRequiresDivisibleCents(t *testing.T) {
	err := ValidateRedPacketDraft(RedPacketDraft{
		Name:               "fixed",
		PacketType:         RedPacketTypeFixed,
		TotalAmountCents:   100,
		TargetParticipants: 10,
		WinnerCount:        3,
	})
	require.Error(t, err)

	err = ValidateRedPacketDraft(RedPacketDraft{
		Name:               "fixed",
		PacketType:         RedPacketTypeFixed,
		TotalAmountCents:   99,
		TargetParticipants: 10,
		WinnerCount:        3,
	})
	require.NoError(t, err)
}

func TestValidateRedPacketDraftLuckyRequiresAtLeastOneCentPerWinner(t *testing.T) {
	err := ValidateRedPacketDraft(RedPacketDraft{
		Name:               "lucky",
		PacketType:         RedPacketTypeLucky,
		TotalAmountCents:   2,
		TargetParticipants: 5,
		WinnerCount:        3,
	})
	require.Error(t, err)
}

func TestAllocateLuckyAmountsDeterministicPositiveAndExact(t *testing.T) {
	seed := []byte("01234567890123456789012345678901")
	first, err := AllocateRedPacketAmounts(19, seed, RedPacketTypeLucky, 997, 7)
	require.NoError(t, err)
	second, err := AllocateRedPacketAmounts(19, seed, RedPacketTypeLucky, 997, 7)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, first, 7)

	var total int64
	for _, amount := range first {
		require.GreaterOrEqual(t, amount, int64(1))
		total += amount
	}
	require.Equal(t, int64(997), total)
}

func TestAllocateFixedAmountsAreEqual(t *testing.T) {
	amounts, err := AllocateRedPacketAmounts(20, []byte("01234567890123456789012345678901"), RedPacketTypeFixed, 1200, 6)
	require.NoError(t, err)
	require.Equal(t, []int64{200, 200, 200, 200, 200, 200}, amounts)
}

func TestSelectWinnerIDsDeterministicAndUnique(t *testing.T) {
	seed := []byte("abcdefghijklmnopqrstuvwxyz123456")
	participants := []int64{101, 102, 103, 104, 105, 106, 107, 108}
	first, err := SelectRedPacketWinnerIDs(33, seed, participants, 4)
	require.NoError(t, err)
	second, err := SelectRedPacketWinnerIDs(33, seed, participants, 4)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, first, 4)

	seen := make(map[int64]struct{}, len(first))
	for _, id := range first {
		_, duplicate := seen[id]
		require.False(t, duplicate)
		seen[id] = struct{}{}
	}
}

func TestMaskRedPacketUsername(t *testing.T) {
	require.Equal(t, "", MaskRedPacketUsername(""))
	require.Equal(t, "张", MaskRedPacketUsername("张"))
	require.Equal(t, "A", MaskRedPacketUsername("AB"))
	require.Equal(t, "A***e", MaskRedPacketUsername("Alice"))
	require.Equal(t, "王***五", MaskRedPacketUsername("王小五"))
	require.Equal(t, "r***1@example.com", MaskRedPacketUsername("redpacket-test-20260731-01@example.com"))
}

func TestRedPacketServiceCreateDraftValidatesBeforeRepository(t *testing.T) {
	repo := &redPacketRepositoryStub{}
	svc := NewRedPacketService(repo)

	_, err := svc.CreateDraft(context.Background(), 9, RedPacketDraft{
		Name:               "bad fixed",
		PacketType:         RedPacketTypeFixed,
		TotalAmountCents:   100,
		TargetParticipants: 10,
		WinnerCount:        3,
	})
	require.Error(t, err)
	require.Zero(t, repo.createCalls)

	created, err := svc.CreateDraft(context.Background(), 9, RedPacketDraft{
		Name:               " valid draft ",
		Message:            " good luck ",
		PacketType:         RedPacketTypeLucky,
		TotalAmountCents:   100,
		TargetParticipants: 10,
		WinnerCount:        3,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), created.ID)
	require.Equal(t, "valid draft", repo.createdDraft.Name)
	require.Equal(t, "good luck", repo.createdDraft.Message)
}

func TestRedPacketServiceGetActivityMarksOnlyAuthenticatedWinner(t *testing.T) {
	repo := &redPacketRepositoryStub{activityDetail: &RedPacketActivityDetail{
		Winners: []RedPacketWinner{
			{UserID: 42, Username: "redpacket-test-20260731-01@example.com"},
			{UserID: 99, Username: "another-user@example.com"},
		},
	}}
	svc := NewRedPacketService(repo)

	detail, err := svc.GetActivity(context.Background(), 7, 42)
	require.NoError(t, err)
	require.Len(t, detail.Winners, 2)
	require.True(t, detail.Winners[0].IsCurrentUser)
	require.False(t, detail.Winners[1].IsCurrentUser)
	require.Equal(t, "r***1@example.com", detail.Winners[0].MaskedUsername)
}

func TestRedPacketServiceParticipateGeneratesCryptographicSeed(t *testing.T) {
	repo := &redPacketRepositoryStub{participateResult: &RedPacketParticipationResult{
		QualificationType: RedPacketQualificationRecharge,
	}}
	svc := NewRedPacketService(repo)

	result, err := svc.Participate(context.Background(), 17, 42)
	require.NoError(t, err)
	require.Equal(t, RedPacketQualificationRecharge, result.QualificationType)
	require.Equal(t, 1, repo.participateCalls)
	require.Len(t, repo.participateSeed, 32)
	require.NotEqual(t, make([]byte, 32), repo.participateSeed)
}

func TestRedPacketWorkerProcessOnceSettlesDeterministicDrawing(t *testing.T) {
	seed := []byte("01234567890123456789012345678901")
	repo := &redPacketRepositoryStub{drawJobs: []RedPacketDrawJob{{
		Activity: RedPacketActivity{
			ID:               31,
			PacketType:       RedPacketTypeLucky,
			TotalAmountCents: 501,
			WinnerCount:      3,
			Status:           RedPacketStatusDrawing,
		},
		Seed: seed,
		Participants: []RedPacketDrawParticipant{
			{ParticipantID: 1, UserID: 101},
			{ParticipantID: 2, UserID: 102},
			{ParticipantID: 3, UserID: 103},
			{ParticipantID: 4, UserID: 104},
			{ParticipantID: 5, UserID: 105},
		},
	}}}
	worker := NewRedPacketWorker(repo, nil, nil, time.Second)

	require.NoError(t, worker.ProcessOnce(context.Background()))
	require.Len(t, repo.settled, 3)
	seenUsers := map[int64]struct{}{}
	var total int64
	var luckiest int
	for _, allocation := range repo.settled {
		_, duplicate := seenUsers[allocation.UserID]
		require.False(t, duplicate)
		seenUsers[allocation.UserID] = struct{}{}
		require.GreaterOrEqual(t, allocation.AmountCents, int64(1))
		total += allocation.AmountCents
		if allocation.IsLuckiest {
			luckiest++
		}
	}
	require.Equal(t, int64(501), total)
	require.Equal(t, 1, luckiest)
}
