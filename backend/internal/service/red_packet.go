package service

import (
	"context"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	RedPacketTypeLucky = "lucky"
	RedPacketTypeFixed = "fixed"

	RedPacketStatusDraft     = "draft"
	RedPacketStatusActive    = "active"
	RedPacketStatusDrawing   = "drawing"
	RedPacketStatusCompleted = "completed"
	RedPacketStatusCanceled  = "canceled"

	RedPacketQualificationRecharge = "recharge"
	RedPacketQualificationPoints   = "invitation_points"

	DefaultRedPacketRechargeThresholdCents = int64(100)
	DefaultRedPacketInvitationThreshold    = 2
	DefaultRedPacketInvitationCost         = 2
)

type RedPacketDraft struct {
	Name               string `json:"name"`
	Message            string `json:"message"`
	PacketType         string `json:"packet_type"`
	TotalAmountCents   int64  `json:"total_amount_cents"`
	TargetParticipants int    `json:"target_participants"`
	WinnerCount        int    `json:"winner_count"`
}

type RedPacketActivity struct {
	ID                        int64      `json:"id"`
	PeriodNo                  int64      `json:"period_no"`
	Name                      string     `json:"name"`
	Message                   string     `json:"message"`
	PacketType                string     `json:"packet_type"`
	TotalAmountCents          int64      `json:"total_amount_cents"`
	TargetParticipants        int        `json:"target_participants"`
	WinnerCount               int        `json:"winner_count"`
	ParticipantCount          int        `json:"participant_count"`
	Status                    string     `json:"status"`
	RechargeThresholdCents    int64      `json:"recharge_threshold_cents"`
	InvitationPointsThreshold int        `json:"invitation_points_threshold"`
	InvitationPointsCost      int        `json:"invitation_points_cost"`
	RechargePriority          bool       `json:"recharge_priority"`
	PublishedAt               *time.Time `json:"published_at,omitempty"`
	DrawingAt                 *time.Time `json:"drawing_at,omitempty"`
	CompletedAt               *time.Time `json:"completed_at,omitempty"`
	CanceledAt                *time.Time `json:"canceled_at,omitempty"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
	HasParticipated           bool       `json:"has_participated"`
	MyQualificationType       string     `json:"my_qualification_type,omitempty"`
	MyRewardCents             *int64     `json:"my_reward_cents,omitempty"`
}

type RedPacketWinner struct {
	UserID         int64      `json:"-"`
	Username       string     `json:"-"`
	MaskedUsername string     `json:"masked_username"`
	AmountCents    int64      `json:"amount_cents"`
	IsLuckiest     bool       `json:"is_luckiest"`
	IsCurrentUser  bool       `json:"is_current_user"`
	CreditedAt     *time.Time `json:"credited_at,omitempty"`
}

type RedPacketActivityDetail struct {
	Activity RedPacketActivity `json:"activity"`
	Winners  []RedPacketWinner `json:"winners"`
}

type RedPacketEligibility struct {
	NetRechargeCents         int64  `json:"net_recharge_cents"`
	RechargeThresholdCents   int64  `json:"recharge_threshold_cents"`
	LotteryPoints            int    `json:"lottery_points"`
	InvitationPointsRequired int    `json:"invitation_points_required"`
	InvitationPointsCost     int    `json:"invitation_points_cost"`
	RechargeQualified        bool   `json:"recharge_qualified"`
	PointsQualified          bool   `json:"points_qualified"`
	PreferredQualification   string `json:"preferred_qualification,omitempty"`
	RechargeShortfallCents   int64  `json:"recharge_shortfall_cents"`
	PointsShortfall          int    `json:"points_shortfall"`
}

type RedPacketReward struct {
	ActivityID   int64     `json:"activity_id"`
	PeriodNo     int64     `json:"period_no"`
	ActivityName string    `json:"activity_name"`
	AmountCents  int64     `json:"amount_cents"`
	CreditedAt   time.Time `json:"credited_at"`
}

type RedPacketParticipationResult struct {
	Activity          RedPacketActivity `json:"activity"`
	QualificationType string            `json:"qualification_type"`
	PointsSpent       int               `json:"points_spent"`
	LotteryPoints     int               `json:"lottery_points"`
	TriggeredDrawing  bool              `json:"triggered_drawing"`
}

type RedPacketAdminPage struct {
	Items    []RedPacketActivity `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Pages    int                 `json:"pages"`
}

type RedPacketDrawParticipant struct {
	ParticipantID int64
	UserID        int64
}

type RedPacketDrawJob struct {
	Activity     RedPacketActivity
	Seed         []byte
	Participants []RedPacketDrawParticipant
}

type RedPacketWinnerAllocation struct {
	ParticipantID int64
	UserID        int64
	AmountCents   int64
	IsLuckiest    bool
}

type RedPacketRepository interface {
	GetCurrent(ctx context.Context, userID int64) (*RedPacketActivity, error)
	GetActivity(ctx context.Context, activityID, userID int64) (*RedPacketActivityDetail, error)
	GetEligibility(ctx context.Context, userID int64) (*RedPacketEligibility, error)
	ListRecent(ctx context.Context, userID int64, limit int) ([]RedPacketActivity, error)
	ListRewards(ctx context.Context, userID int64, limit int) ([]RedPacketReward, error)
	Participate(ctx context.Context, activityID, userID int64, seed []byte) (*RedPacketParticipationResult, error)
	CreateDraft(ctx context.Context, adminID int64, draft RedPacketDraft) (*RedPacketActivity, error)
	UpdateDraft(ctx context.Context, activityID int64, draft RedPacketDraft) (*RedPacketActivity, error)
	Publish(ctx context.Context, activityID, adminID int64) (*RedPacketActivity, error)
	Cancel(ctx context.Context, activityID, adminID int64) error
	ListAdmin(ctx context.Context, page, pageSize int) (*RedPacketAdminPage, error)
	ListDrawing(ctx context.Context, limit int) ([]RedPacketDrawJob, error)
	Settle(ctx context.Context, job RedPacketDrawJob, winners []RedPacketWinnerAllocation) error
}

type RedPacketService struct {
	repo RedPacketRepository
}

func NewRedPacketService(repo RedPacketRepository) *RedPacketService {
	return &RedPacketService{repo: repo}
}

func (s *RedPacketService) GetCurrent(ctx context.Context, userID int64) (*RedPacketActivity, error) {
	return s.repo.GetCurrent(ctx, userID)
}

func (s *RedPacketService) GetActivity(ctx context.Context, activityID, userID int64) (*RedPacketActivityDetail, error) {
	detail, err := s.repo.GetActivity(ctx, activityID, userID)
	if err != nil || detail == nil {
		return detail, err
	}
	for i := range detail.Winners {
		winner := &detail.Winners[i]
		winner.IsCurrentUser = winner.UserID == userID
		winner.MaskedUsername = MaskRedPacketUsername(winner.Username)
		winner.UserID = 0
		winner.Username = ""
	}
	return detail, nil
}

func (s *RedPacketService) GetEligibility(ctx context.Context, userID int64) (*RedPacketEligibility, error) {
	return s.repo.GetEligibility(ctx, userID)
}

func (s *RedPacketService) ListRecent(ctx context.Context, userID int64, limit int) ([]RedPacketActivity, error) {
	if limit <= 0 || limit > 5 {
		limit = 5
	}
	return s.repo.ListRecent(ctx, userID, limit)
}

func (s *RedPacketService) ListRewards(ctx context.Context, userID int64, limit int) ([]RedPacketReward, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListRewards(ctx, userID, limit)
}

func (s *RedPacketService) Participate(ctx context.Context, activityID, userID int64) (*RedPacketParticipationResult, error) {
	seed := make([]byte, sha256.Size)
	if _, err := crand.Read(seed); err != nil {
		return nil, fmt.Errorf("generate red packet draw seed: %w", err)
	}
	return s.repo.Participate(ctx, activityID, userID, seed)
}

func (s *RedPacketService) CreateDraft(ctx context.Context, adminID int64, draft RedPacketDraft) (*RedPacketActivity, error) {
	draft.Name = strings.TrimSpace(draft.Name)
	draft.Message = strings.TrimSpace(draft.Message)
	if err := ValidateRedPacketDraft(draft); err != nil {
		return nil, err
	}
	return s.repo.CreateDraft(ctx, adminID, draft)
}

func (s *RedPacketService) UpdateDraft(ctx context.Context, activityID int64, draft RedPacketDraft) (*RedPacketActivity, error) {
	draft.Name = strings.TrimSpace(draft.Name)
	draft.Message = strings.TrimSpace(draft.Message)
	if err := ValidateRedPacketDraft(draft); err != nil {
		return nil, err
	}
	return s.repo.UpdateDraft(ctx, activityID, draft)
}

func (s *RedPacketService) Publish(ctx context.Context, activityID, adminID int64) (*RedPacketActivity, error) {
	return s.repo.Publish(ctx, activityID, adminID)
}

func (s *RedPacketService) Cancel(ctx context.Context, activityID, adminID int64) error {
	return s.repo.Cancel(ctx, activityID, adminID)
}

func (s *RedPacketService) ListAdmin(ctx context.Context, page, pageSize int) (*RedPacketAdminPage, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListAdmin(ctx, page, pageSize)
}

func ValidateRedPacketDraft(draft RedPacketDraft) error {
	draft.Name = strings.TrimSpace(draft.Name)
	if draft.Name == "" || utf8.RuneCountInString(draft.Name) > 100 {
		return infraerrors.BadRequest("RED_PACKET_NAME_INVALID", "activity name is required and must not exceed 100 characters")
	}
	if utf8.RuneCountInString(strings.TrimSpace(draft.Message)) > 255 {
		return infraerrors.BadRequest("RED_PACKET_MESSAGE_INVALID", "activity message must not exceed 255 characters")
	}
	if draft.PacketType != RedPacketTypeLucky && draft.PacketType != RedPacketTypeFixed {
		return infraerrors.BadRequest("RED_PACKET_TYPE_INVALID", "packet_type must be lucky or fixed")
	}
	if draft.TargetParticipants <= 0 {
		return infraerrors.BadRequest("RED_PACKET_TARGET_INVALID", "target_participants must be greater than zero")
	}
	if draft.WinnerCount <= 0 || draft.WinnerCount > draft.TargetParticipants {
		return infraerrors.BadRequest("RED_PACKET_WINNER_COUNT_INVALID", "winner_count must be between 1 and target_participants")
	}
	if draft.TotalAmountCents < int64(draft.WinnerCount) {
		return infraerrors.BadRequest("RED_PACKET_AMOUNT_TOO_SMALL", "total amount must provide at least one cent per winner")
	}
	if draft.PacketType == RedPacketTypeFixed && draft.TotalAmountCents%int64(draft.WinnerCount) != 0 {
		return infraerrors.BadRequest("RED_PACKET_FIXED_AMOUNT_NOT_DIVISIBLE", "fixed packet amount must divide evenly by winner_count")
	}
	return nil
}

func AllocateRedPacketAmounts(activityID int64, seed []byte, packetType string, totalAmountCents int64, winnerCount int) ([]int64, error) {
	if winnerCount <= 0 || totalAmountCents < int64(winnerCount) {
		return nil, infraerrors.BadRequest("RED_PACKET_ALLOCATION_INVALID", "amount must provide at least one cent per winner")
	}
	if len(seed) != sha256.Size {
		return nil, infraerrors.BadRequest("RED_PACKET_SEED_INVALID", "draw seed must contain 32 bytes")
	}
	if packetType == RedPacketTypeFixed {
		if totalAmountCents%int64(winnerCount) != 0 {
			return nil, infraerrors.BadRequest("RED_PACKET_FIXED_AMOUNT_NOT_DIVISIBLE", "fixed packet amount must divide evenly by winner_count")
		}
		amount := totalAmountCents / int64(winnerCount)
		result := make([]int64, winnerCount)
		for i := range result {
			result[i] = amount
		}
		return result, nil
	}
	if packetType != RedPacketTypeLucky {
		return nil, infraerrors.BadRequest("RED_PACKET_TYPE_INVALID", "packet_type must be lucky or fixed")
	}
	if winnerCount == 1 {
		return []int64{totalAmountCents}, nil
	}

	rng := newRedPacketDeterministicRandom(activityID, seed, "lucky-amount-cuts")
	cuts := make(map[int64]struct{}, winnerCount-1)
	for len(cuts) < winnerCount-1 {
		value, err := rng.int64n(totalAmountCents - 1)
		if err != nil {
			return nil, err
		}
		cuts[value+1] = struct{}{}
	}
	orderedCuts := make([]int64, 0, len(cuts))
	for cut := range cuts {
		orderedCuts = append(orderedCuts, cut)
	}
	sort.Slice(orderedCuts, func(i, j int) bool { return orderedCuts[i] < orderedCuts[j] })

	amounts := make([]int64, 0, winnerCount)
	previous := int64(0)
	for _, cut := range orderedCuts {
		amounts = append(amounts, cut-previous)
		previous = cut
	}
	amounts = append(amounts, totalAmountCents-previous)

	shuffle := newRedPacketDeterministicRandom(activityID, seed, "lucky-amount-order")
	for i := len(amounts) - 1; i > 0; i-- {
		j, err := shuffle.int64n(int64(i + 1))
		if err != nil {
			return nil, err
		}
		amounts[i], amounts[j] = amounts[j], amounts[i]
	}
	return amounts, nil
}

func SelectRedPacketWinnerIDs(activityID int64, seed []byte, participantIDs []int64, winnerCount int) ([]int64, error) {
	if len(seed) != sha256.Size {
		return nil, infraerrors.BadRequest("RED_PACKET_SEED_INVALID", "draw seed must contain 32 bytes")
	}
	if winnerCount <= 0 || winnerCount > len(participantIDs) {
		return nil, infraerrors.BadRequest("RED_PACKET_WINNER_COUNT_INVALID", "winner_count exceeds available participants")
	}
	shuffled := append([]int64(nil), participantIDs...)
	rng := newRedPacketDeterministicRandom(activityID, seed, "winner-selection")
	for i := len(shuffled) - 1; i > 0; i-- {
		j, err := rng.int64n(int64(i + 1))
		if err != nil {
			return nil, err
		}
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return append([]int64(nil), shuffled[:winnerCount]...), nil
}

func MaskRedPacketUsername(username string) string {
	username = strings.TrimSpace(username)
	if strings.Contains(username, "@") {
		return MaskEmail(username)
	}
	runes := []rune(username)
	if len(runes) == 0 {
		return ""
	}
	if len(runes) < 3 {
		return string(runes[0])
	}
	return string(runes[0]) + "***" + string(runes[len(runes)-1])
}

type redPacketDeterministicRandom struct {
	activityID int64
	seed       []byte
	purpose    string
	counter    uint64
}

func newRedPacketDeterministicRandom(activityID int64, seed []byte, purpose string) *redPacketDeterministicRandom {
	return &redPacketDeterministicRandom{
		activityID: activityID,
		seed:       append([]byte(nil), seed...),
		purpose:    purpose,
	}
}

func (r *redPacketDeterministicRandom) uint64() uint64 {
	mac := hmac.New(sha256.New, r.seed)
	var encoded [16]byte
	binary.BigEndian.PutUint64(encoded[:8], uint64(r.activityID))
	binary.BigEndian.PutUint64(encoded[8:], r.counter)
	_, _ = mac.Write([]byte(r.purpose))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write(encoded[:])
	r.counter++
	return binary.BigEndian.Uint64(mac.Sum(nil)[:8])
}

func (r *redPacketDeterministicRandom) int64n(bound int64) (int64, error) {
	if bound <= 0 {
		return 0, fmt.Errorf("random bound must be positive: %d", bound)
	}
	unsignedBound := uint64(bound)
	limit := ^uint64(0) - (^uint64(0) % unsignedBound)
	for {
		value := r.uint64()
		if value < limit {
			return int64(value % unsignedBound), nil
		}
	}
}

type RedPacketWorker struct {
	repo         RedPacketRepository
	billingCache *BillingCacheService
	authCache    APIKeyAuthCacheInvalidator
	interval     time.Duration
	mu           sync.Mutex
	cancel       context.CancelFunc
	done         chan struct{}
}

func NewRedPacketWorker(repo RedPacketRepository, billingCache *BillingCacheService, authCache APIKeyAuthCacheInvalidator, interval time.Duration) *RedPacketWorker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &RedPacketWorker{
		repo:         repo,
		billingCache: billingCache,
		authCache:    authCache,
		interval:     interval,
	}
}

func (w *RedPacketWorker) ProcessOnce(ctx context.Context) error {
	jobs, err := w.repo.ListDrawing(ctx, 20)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		participantIDs := make([]int64, len(job.Participants))
		participantsByID := make(map[int64]RedPacketDrawParticipant, len(job.Participants))
		for i, participant := range job.Participants {
			participantIDs[i] = participant.ParticipantID
			participantsByID[participant.ParticipantID] = participant
		}
		winnerIDs, err := SelectRedPacketWinnerIDs(job.Activity.ID, job.Seed, participantIDs, job.Activity.WinnerCount)
		if err != nil {
			return fmt.Errorf("select red packet winners for activity %d: %w", job.Activity.ID, err)
		}
		amounts, err := AllocateRedPacketAmounts(job.Activity.ID, job.Seed, job.Activity.PacketType, job.Activity.TotalAmountCents, job.Activity.WinnerCount)
		if err != nil {
			return fmt.Errorf("allocate red packet amounts for activity %d: %w", job.Activity.ID, err)
		}

		luckiestIndex := -1
		if job.Activity.PacketType == RedPacketTypeLucky {
			luckiestIndex = 0
			for i := 1; i < len(amounts); i++ {
				if amounts[i] > amounts[luckiestIndex] {
					luckiestIndex = i
				}
			}
		}
		allocations := make([]RedPacketWinnerAllocation, len(winnerIDs))
		for i, participantID := range winnerIDs {
			participant, ok := participantsByID[participantID]
			if !ok {
				return fmt.Errorf("participant %d missing from red packet activity %d", participantID, job.Activity.ID)
			}
			allocations[i] = RedPacketWinnerAllocation{
				ParticipantID: participantID,
				UserID:        participant.UserID,
				AmountCents:   amounts[i],
				IsLuckiest:    i == luckiestIndex,
			}
		}
		if err := w.repo.Settle(ctx, job, allocations); err != nil {
			return fmt.Errorf("settle red packet activity %d: %w", job.Activity.ID, err)
		}
		w.invalidateWinnerBalances(ctx, allocations)
	}
	return nil
}

func (w *RedPacketWorker) invalidateWinnerBalances(ctx context.Context, allocations []RedPacketWinnerAllocation) {
	seen := make(map[int64]struct{}, len(allocations))
	for _, allocation := range allocations {
		if _, ok := seen[allocation.UserID]; ok {
			continue
		}
		seen[allocation.UserID] = struct{}{}
		if w.authCache != nil {
			w.authCache.InvalidateAuthCacheByUserID(ctx, allocation.UserID)
		}
		if w.billingCache != nil {
			_ = w.billingCache.InvalidateUserBalance(ctx, allocation.UserID)
		}
	}
}

func (w *RedPacketWorker) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.done = make(chan struct{})
	go func(done chan struct{}) {
		defer close(done)
		_ = w.ProcessOnce(ctx)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = w.ProcessOnce(ctx)
			}
		}
	}(w.done)
}

func (w *RedPacketWorker) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	done := w.done
	w.cancel = nil
	w.done = nil
	w.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}
