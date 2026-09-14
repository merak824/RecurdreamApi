package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const redPacketActivityColumns = `
id, period_no, name, message, packet_type, total_amount_cents,
target_participants, winner_count, participant_count, status,
recharge_threshold_cents, invitation_points_threshold, invitation_points_cost,
recharge_priority, published_at, drawing_at, completed_at, canceled_at,
created_at, updated_at`

type redPacketRepository struct {
	db *sql.DB
}

func NewRedPacketRepository(db *sql.DB) service.RedPacketRepository {
	return &redPacketRepository{db: db}
}

type redPacketScanner interface {
	Scan(dest ...any) error
}

func scanRedPacketActivity(scanner redPacketScanner, activity *service.RedPacketActivity, extra ...any) error {
	var publishedAt, drawingAt, completedAt, canceledAt sql.NullTime
	targets := []any{
		&activity.ID,
		&activity.PeriodNo,
		&activity.Name,
		&activity.Message,
		&activity.PacketType,
		&activity.TotalAmountCents,
		&activity.TargetParticipants,
		&activity.WinnerCount,
		&activity.ParticipantCount,
		&activity.Status,
		&activity.RechargeThresholdCents,
		&activity.InvitationPointsThreshold,
		&activity.InvitationPointsCost,
		&activity.RechargePriority,
		&publishedAt,
		&drawingAt,
		&completedAt,
		&canceledAt,
		&activity.CreatedAt,
		&activity.UpdatedAt,
	}
	targets = append(targets, extra...)
	if err := scanner.Scan(targets...); err != nil {
		return err
	}
	activity.PublishedAt = nullTimePointer(publishedAt)
	activity.DrawingAt = nullTimePointer(drawingAt)
	activity.CompletedAt = nullTimePointer(completedAt)
	activity.CanceledAt = nullTimePointer(canceledAt)
	return nil
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func (r *redPacketRepository) GetCurrent(ctx context.Context, userID int64) (*service.RedPacketActivity, error) {
	activity := new(service.RedPacketActivity)
	err := scanRedPacketActivity(r.db.QueryRowContext(ctx, `SELECT `+redPacketActivityColumns+`
FROM red_packet_activities
WHERE status IN ('active', 'drawing')
ORDER BY period_no DESC
LIMIT 1`), activity)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get current red packet activity: %w", err)
	}
	if err := r.decorateUserActivity(ctx, activity, userID); err != nil {
		return nil, err
	}
	return activity, nil
}

func (r *redPacketRepository) GetActivity(ctx context.Context, activityID, userID int64) (*service.RedPacketActivityDetail, error) {
	activity := new(service.RedPacketActivity)
	err := scanRedPacketActivity(r.db.QueryRowContext(ctx, `SELECT `+redPacketActivityColumns+`
FROM red_packet_activities
WHERE id = $1 AND status <> 'draft'`, activityID), activity)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, redPacketNotFound()
	}
	if err != nil {
		return nil, fmt.Errorf("get red packet activity: %w", err)
	}
	if err := r.decorateUserActivity(ctx, activity, userID); err != nil {
		return nil, err
	}

	detail := &service.RedPacketActivityDetail{Activity: *activity, Winners: []service.RedPacketWinner{}}
	if activity.Status != service.RedPacketStatusCompleted {
		return detail, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT w.user_id, COALESCE(NULLIF(u.email, ''), u.username), w.amount_cents,
       w.is_luckiest, w.credited_at
FROM red_packet_winners w
JOIN users u ON u.id = w.user_id
WHERE w.activity_id = $1
ORDER BY w.id`, activityID)
	if err != nil {
		return nil, fmt.Errorf("list red packet winners: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var winner service.RedPacketWinner
		if err := rows.Scan(&winner.UserID, &winner.Username, &winner.AmountCents, &winner.IsLuckiest, &winner.CreditedAt); err != nil {
			return nil, err
		}
		detail.Winners = append(detail.Winners, winner)
	}
	return detail, rows.Err()
}

func (r *redPacketRepository) decorateUserActivity(ctx context.Context, activity *service.RedPacketActivity, userID int64) error {
	if userID <= 0 {
		return nil
	}
	var qualification string
	var reward sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
SELECT p.qualification_type,
       (SELECT w.amount_cents FROM red_packet_winners w
        WHERE w.activity_id = p.activity_id AND w.user_id = p.user_id)
FROM red_packet_participants p
WHERE p.activity_id = $1 AND p.user_id = $2`, activity.ID, userID).Scan(&qualification, &reward)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get red packet participation: %w", err)
	}
	activity.HasParticipated = true
	activity.MyQualificationType = qualification
	if reward.Valid {
		amount := reward.Int64
		activity.MyRewardCents = &amount
	}
	return nil
}

func (r *redPacketRepository) GetEligibility(ctx context.Context, userID int64) (*service.RedPacketEligibility, error) {
	netRecharge, err := queryNetRechargeCents(ctx, r.db, userID)
	if err != nil {
		return nil, err
	}
	var points int
	err = r.db.QueryRowContext(ctx, `SELECT COALESCE((
    SELECT lottery_points FROM user_affiliates WHERE user_id = $1
), 0)`, userID).Scan(&points)
	if err != nil {
		return nil, fmt.Errorf("get invitation lottery points: %w", err)
	}

	threshold := service.DefaultRedPacketRechargeThresholdCents
	pointsRequired := service.DefaultRedPacketInvitationThreshold
	pointsCost := service.DefaultRedPacketInvitationCost
	err = r.db.QueryRowContext(ctx, `
SELECT recharge_threshold_cents, invitation_points_threshold, invitation_points_cost
FROM red_packet_activities
WHERE status IN ('active', 'drawing')
ORDER BY period_no DESC
LIMIT 1`).Scan(&threshold, &pointsRequired, &pointsCost)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get red packet eligibility snapshot: %w", err)
	}

	result := &service.RedPacketEligibility{
		NetRechargeCents:         netRecharge,
		RechargeThresholdCents:   threshold,
		LotteryPoints:            points,
		InvitationPointsRequired: pointsRequired,
		InvitationPointsCost:     pointsCost,
		RechargeQualified:        netRecharge >= threshold,
		PointsQualified:          points >= pointsRequired && points >= pointsCost,
	}
	if result.RechargeQualified {
		result.PreferredQualification = service.RedPacketQualificationRecharge
	} else if result.PointsQualified {
		result.PreferredQualification = service.RedPacketQualificationPoints
	}
	if netRecharge < threshold {
		result.RechargeShortfallCents = threshold - netRecharge
	}
	minimumPoints := pointsRequired
	if pointsCost > minimumPoints {
		minimumPoints = pointsCost
	}
	if points < minimumPoints {
		result.PointsShortfall = minimumPoints - points
	}
	return result, nil
}

func (r *redPacketRepository) ListRecent(ctx context.Context, userID int64, limit int) ([]service.RedPacketActivity, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+redPacketActivityColumns+`
FROM red_packet_activities
WHERE status <> 'draft'
ORDER BY period_no DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent red packet activities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	activities := make([]service.RedPacketActivity, 0, limit)
	for rows.Next() {
		var activity service.RedPacketActivity
		if err := scanRedPacketActivity(rows, &activity); err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range activities {
		if err := r.decorateUserActivity(ctx, &activities[i], userID); err != nil {
			return nil, err
		}
	}
	return activities, nil
}

func (r *redPacketRepository) ListRewards(ctx context.Context, userID int64, limit int) ([]service.RedPacketReward, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT a.id, a.period_no, a.name, l.amount_cents, l.created_at
FROM red_packet_reward_ledger l
JOIN red_packet_activities a ON a.id = l.activity_id
WHERE l.user_id = $1
ORDER BY l.created_at DESC
LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list red packet rewards: %w", err)
	}
	defer func() { _ = rows.Close() }()
	rewards := make([]service.RedPacketReward, 0)
	for rows.Next() {
		var reward service.RedPacketReward
		if err := rows.Scan(&reward.ActivityID, &reward.PeriodNo, &reward.ActivityName, &reward.AmountCents, &reward.CreditedAt); err != nil {
			return nil, err
		}
		rewards = append(rewards, reward)
	}
	return rewards, rows.Err()
}

func (r *redPacketRepository) Participate(ctx context.Context, activityID, userID int64, seed []byte) (*service.RedPacketParticipationResult, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin red packet participation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	activity := new(service.RedPacketActivity)
	err = scanRedPacketActivity(tx.QueryRowContext(ctx, `SELECT `+redPacketActivityColumns+`
FROM red_packet_activities
WHERE id = $1
FOR UPDATE`, activityID), activity)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, redPacketNotFound()
	}
	if err != nil {
		return nil, fmt.Errorf("lock red packet activity: %w", err)
	}
	if activity.Status != service.RedPacketStatusActive {
		return nil, infraerrors.Conflict("RED_PACKET_NOT_ACTIVE", "the activity is no longer accepting participants")
	}
	if activity.ParticipantCount >= activity.TargetParticipants {
		return nil, infraerrors.Conflict("RED_PACKET_FULL", "the activity has reached its participant target")
	}

	var userStatus string
	var deletedAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT status, deleted_at
FROM users
WHERE id = $1
FOR UPDATE`, userID).Scan(&userStatus, &deletedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("RED_PACKET_USER_NOT_FOUND", "user not found")
		}
		return nil, fmt.Errorf("lock red packet participant user: %w", err)
	}
	if userStatus != service.StatusActive || deletedAt.Valid {
		return nil, infraerrors.Forbidden("RED_PACKET_USER_INACTIVE", "the account cannot participate in this activity")
	}

	var alreadyParticipated bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
    SELECT 1 FROM red_packet_participants WHERE activity_id = $1 AND user_id = $2
)`, activityID, userID).Scan(&alreadyParticipated); err != nil {
		return nil, fmt.Errorf("check red packet participation: %w", err)
	}
	if alreadyParticipated {
		return nil, infraerrors.Conflict("RED_PACKET_ALREADY_PARTICIPATED", "you already participated in this activity")
	}

	netRecharge, err := queryNetRechargeCents(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	qualification := service.RedPacketQualificationRecharge
	pointsSpent := 0
	lotteryPoints := 0
	if netRecharge < activity.RechargeThresholdCents {
		qualification = service.RedPacketQualificationPoints
		var availablePoints int
		err := tx.QueryRowContext(ctx, `SELECT lottery_points
FROM user_affiliates
WHERE user_id = $1
FOR UPDATE`, userID).Scan(&availablePoints)
		if errors.Is(err, sql.ErrNoRows) {
			availablePoints = 0
		} else if err != nil {
			return nil, fmt.Errorf("lock invitation lottery points: %w", err)
		}
		minimumPoints := activity.InvitationPointsThreshold
		if activity.InvitationPointsCost > minimumPoints {
			minimumPoints = activity.InvitationPointsCost
		}
		if availablePoints < minimumPoints {
			return nil, redPacketIneligible(activity, netRecharge, availablePoints, minimumPoints)
		}
		pointsSpent = activity.InvitationPointsCost
	}

	var participantID int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO red_packet_participants
    (activity_id, user_id, qualification_type, net_recharge_cents, points_spent, created_at)
VALUES ($1, $2, $3, $4, $5, NOW())
RETURNING id`, activityID, userID, qualification, netRecharge, pointsSpent).Scan(&participantID); err != nil {
		if isRedPacketUniqueViolation(err) {
			return nil, infraerrors.Conflict("RED_PACKET_ALREADY_PARTICIPATED", "you already participated in this activity")
		}
		return nil, fmt.Errorf("insert red packet participant: %w", err)
	}

	if qualification == service.RedPacketQualificationPoints {
		err = tx.QueryRowContext(ctx, `
UPDATE user_affiliates
SET lottery_points = lottery_points - $1, updated_at = NOW()
WHERE user_id = $2 AND lottery_points >= $1
RETURNING lottery_points`, pointsSpent, userID).Scan(&lotteryPoints)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.Conflict("RED_PACKET_POINTS_CHANGED", "invitation points changed; please retry")
		}
		if err != nil {
			return nil, fmt.Errorf("consume invitation lottery points: %w", err)
		}
		businessKey := fmt.Sprintf("red-packet:%d:user:%d", activityID, userID)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO invite_lottery_point_ledger
    (user_id, action, points, activity_id, participant_id, business_key, created_at)
VALUES ($1, 'participation_consume', $2, $3, $4, $5, NOW())`, userID, pointsSpent, activityID, participantID, businessKey); err != nil {
			return nil, fmt.Errorf("insert invitation point consumption ledger: %w", err)
		}
	} else {
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE((
    SELECT lottery_points FROM user_affiliates WHERE user_id = $1
), 0)`, userID).Scan(&lotteryPoints); err != nil {
			return nil, fmt.Errorf("get remaining invitation points: %w", err)
		}
	}

	newCount := activity.ParticipantCount + 1
	triggeredDrawing := newCount >= activity.TargetParticipants
	if triggeredDrawing {
		if len(seed) != 32 {
			return nil, infraerrors.BadRequest("RED_PACKET_SEED_INVALID", "draw seed must contain 32 bytes")
		}
		result, err := tx.ExecContext(ctx, `
UPDATE red_packet_activities
SET participant_count = $1, status = 'drawing', draw_seed = $2,
    drawing_at = NOW(), updated_at = NOW()
WHERE id = $3 AND status = 'active'`, newCount, seed, activityID)
		if err != nil {
			return nil, fmt.Errorf("transition red packet activity to drawing: %w", err)
		}
		if err := requireOneAffected(result, "transition red packet activity to drawing"); err != nil {
			return nil, err
		}
		activity.Status = service.RedPacketStatusDrawing
		now := time.Now().UTC()
		activity.DrawingAt = &now
	} else {
		result, err := tx.ExecContext(ctx, `
UPDATE red_packet_activities
SET participant_count = $1, updated_at = NOW()
WHERE id = $2`, newCount, activityID)
		if err != nil {
			return nil, fmt.Errorf("increment red packet participant count: %w", err)
		}
		if err := requireOneAffected(result, "increment red packet participant count"); err != nil {
			return nil, err
		}
	}
	activity.ParticipantCount = newCount
	activity.HasParticipated = true
	activity.MyQualificationType = qualification
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit red packet participation: %w", err)
	}
	return &service.RedPacketParticipationResult{
		Activity:          *activity,
		QualificationType: qualification,
		PointsSpent:       pointsSpent,
		LotteryPoints:     lotteryPoints,
		TriggeredDrawing:  triggeredDrawing,
	}, nil
}

type redPacketQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func queryNetRechargeCents(ctx context.Context, queryer redPacketQueryer, userID int64) (int64, error) {
	var cents int64
	err := queryer.QueryRowContext(ctx, `
SELECT ROUND(COALESCE(SUM(GREATEST(amount - refund_amount, 0)), 0) * 100)::BIGINT
FROM payment_orders
WHERE user_id = $1
  AND order_type = 'balance'
  AND completed_at IS NOT NULL`, userID).Scan(&cents)
	if err != nil {
		return 0, fmt.Errorf("calculate net successful recharge: %w", err)
	}
	return cents, nil
}

func redPacketIneligible(activity *service.RedPacketActivity, netRecharge int64, points, minimumPoints int) error {
	rechargeShortfall := activity.RechargeThresholdCents - netRecharge
	if rechargeShortfall < 0 {
		rechargeShortfall = 0
	}
	pointsShortfall := minimumPoints - points
	if pointsShortfall < 0 {
		pointsShortfall = 0
	}
	return infraerrors.BadRequest("RED_PACKET_INELIGIBLE", "recharge or invitation-point eligibility is required").WithMetadata(map[string]string{
		"recharge_shortfall_cents": strconv.FormatInt(rechargeShortfall, 10),
		"points_shortfall":         strconv.Itoa(pointsShortfall),
	})
}

func (r *redPacketRepository) CreateDraft(ctx context.Context, adminID int64, draft service.RedPacketDraft) (*service.RedPacketActivity, error) {
	activity := new(service.RedPacketActivity)
	err := scanRedPacketActivity(r.db.QueryRowContext(ctx, `
INSERT INTO red_packet_activities
    (name, message, packet_type, total_amount_cents, target_participants, winner_count,
     recharge_threshold_cents, invitation_points_threshold, invitation_points_cost,
     recharge_priority, status, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, 'draft', $10, NOW(), NOW())
RETURNING `+redPacketActivityColumns,
		draft.Name, draft.Message, draft.PacketType, draft.TotalAmountCents,
		draft.TargetParticipants, draft.WinnerCount,
		service.DefaultRedPacketRechargeThresholdCents,
		service.DefaultRedPacketInvitationThreshold,
		service.DefaultRedPacketInvitationCost,
		adminID), activity)
	if err != nil {
		return nil, fmt.Errorf("create red packet draft: %w", err)
	}
	return activity, nil
}

func (r *redPacketRepository) UpdateDraft(ctx context.Context, activityID int64, draft service.RedPacketDraft) (*service.RedPacketActivity, error) {
	activity := new(service.RedPacketActivity)
	err := scanRedPacketActivity(r.db.QueryRowContext(ctx, `
UPDATE red_packet_activities
SET name = $1, message = $2, packet_type = $3, total_amount_cents = $4,
    target_participants = $5, winner_count = $6, updated_at = NOW()
WHERE id = $7 AND status = 'draft'
RETURNING `+redPacketActivityColumns,
		draft.Name, draft.Message, draft.PacketType, draft.TotalAmountCents,
		draft.TargetParticipants, draft.WinnerCount, activityID), activity)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, infraerrors.Conflict("RED_PACKET_DRAFT_LOCKED", "only draft activities can be edited")
	}
	if err != nil {
		return nil, fmt.Errorf("update red packet draft: %w", err)
	}
	return activity, nil
}

func (r *redPacketRepository) Publish(ctx context.Context, activityID, adminID int64) (*service.RedPacketActivity, error) {
	_ = adminID
	activity := new(service.RedPacketActivity)
	err := scanRedPacketActivity(r.db.QueryRowContext(ctx, `
UPDATE red_packet_activities
SET status = 'active', published_at = NOW(), updated_at = NOW()
WHERE id = $1 AND status = 'draft'
RETURNING `+redPacketActivityColumns, activityID), activity)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, infraerrors.Conflict("RED_PACKET_DRAFT_LOCKED", "only draft activities can be published")
	}
	if isRedPacketUniqueViolation(err) {
		return nil, infraerrors.Conflict("RED_PACKET_ACTIVITY_ALREADY_RUNNING", "another red packet activity is active or drawing")
	}
	if err != nil {
		return nil, fmt.Errorf("publish red packet activity: %w", err)
	}
	return activity, nil
}

func (r *redPacketRepository) Cancel(ctx context.Context, activityID, adminID int64) error {
	_ = adminID
	result, err := r.db.ExecContext(ctx, `
UPDATE red_packet_activities
SET status = 'canceled', canceled_at = NOW(), updated_at = NOW()
WHERE id = $1 AND status = 'active'`, activityID)
	if err != nil {
		return fmt.Errorf("cancel red packet activity: %w", err)
	}
	if err := requireOneAffected(result, "cancel red packet activity"); err != nil {
		return infraerrors.Conflict("RED_PACKET_CANNOT_CANCEL", "only an active activity can be canceled")
	}
	return nil
}

func (r *redPacketRepository) ListAdmin(ctx context.Context, page, pageSize int) (*service.RedPacketAdminPage, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM red_packet_activities`).Scan(&total); err != nil {
		return nil, fmt.Errorf("count red packet activities: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+redPacketActivityColumns+`
FROM red_packet_activities
ORDER BY period_no DESC
LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list admin red packet activities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.RedPacketActivity, 0, pageSize)
	for rows.Next() {
		var activity service.RedPacketActivity
		if err := scanRedPacketActivity(rows, &activity); err != nil {
			return nil, err
		}
		items = append(items, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	return &service.RedPacketAdminPage{Items: items, Total: total, Page: page, PageSize: pageSize, Pages: pages}, nil
}

func (r *redPacketRepository) ListDrawing(ctx context.Context, limit int) ([]service.RedPacketDrawJob, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+redPacketActivityColumns+`, draw_seed
FROM red_packet_activities
WHERE status = 'drawing'
ORDER BY drawing_at, id
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list drawing red packet activities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	jobs := make([]service.RedPacketDrawJob, 0)
	for rows.Next() {
		var job service.RedPacketDrawJob
		if err := scanRedPacketActivity(rows, &job.Activity, &job.Seed); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range jobs {
		participantRows, err := r.db.QueryContext(ctx, `
SELECT id, user_id
FROM red_packet_participants
WHERE activity_id = $1
ORDER BY id`, jobs[i].Activity.ID)
		if err != nil {
			return nil, fmt.Errorf("list participants for drawing activity %d: %w", jobs[i].Activity.ID, err)
		}
		for participantRows.Next() {
			var participant service.RedPacketDrawParticipant
			if err := participantRows.Scan(&participant.ParticipantID, &participant.UserID); err != nil {
				_ = participantRows.Close()
				return nil, err
			}
			jobs[i].Participants = append(jobs[i].Participants, participant)
		}
		if err := participantRows.Err(); err != nil {
			_ = participantRows.Close()
			return nil, err
		}
		_ = participantRows.Close()
	}
	return jobs, nil
}

func (r *redPacketRepository) Settle(ctx context.Context, job service.RedPacketDrawJob, winners []service.RedPacketWinnerAllocation) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin red packet settlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var status, packetType string
	var totalAmount int64
	var winnerCount int
	err = tx.QueryRowContext(ctx, `SELECT status, packet_type, total_amount_cents, winner_count FROM red_packet_activities WHERE id = $1 FOR UPDATE`, job.Activity.ID).
		Scan(&status, &packetType, &totalAmount, &winnerCount)
	if errors.Is(err, sql.ErrNoRows) {
		return redPacketNotFound()
	}
	if err != nil {
		return fmt.Errorf("lock drawing red packet activity: %w", err)
	}
	if status == service.RedPacketStatusCompleted {
		return tx.Commit()
	}
	if status != service.RedPacketStatusDrawing {
		return infraerrors.Conflict("RED_PACKET_NOT_DRAWING", "activity is not waiting for settlement")
	}
	if len(winners) != winnerCount {
		return infraerrors.BadRequest("RED_PACKET_SETTLEMENT_INVALID", "winner allocation count does not match activity")
	}
	var allocatedTotal int64
	for _, winner := range winners {
		if winner.AmountCents <= 0 {
			return infraerrors.BadRequest("RED_PACKET_SETTLEMENT_INVALID", "winner allocation must be positive")
		}
		allocatedTotal += winner.AmountCents
	}
	if allocatedTotal != totalAmount {
		return infraerrors.BadRequest("RED_PACKET_SETTLEMENT_INVALID", "winner allocations do not equal the activity total")
	}

	for _, winner := range winners {
		var validParticipant bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
    SELECT 1 FROM red_packet_participants
    WHERE activity_id = $1 AND id = $2 AND user_id = $3
)`, job.Activity.ID, winner.ParticipantID, winner.UserID).Scan(&validParticipant); err != nil {
			return fmt.Errorf("validate red packet winner participant: %w", err)
		}
		if !validParticipant {
			return infraerrors.BadRequest("RED_PACKET_SETTLEMENT_INVALID", "winner does not belong to this activity")
		}

		var winnerID int64
		if err := tx.QueryRowContext(ctx, `
INSERT INTO red_packet_winners
    (activity_id, participant_id, user_id, amount_cents, is_luckiest, credited_at, created_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
RETURNING id`, job.Activity.ID, winner.ParticipantID, winner.UserID, winner.AmountCents, winner.IsLuckiest).Scan(&winnerID); err != nil {
			return fmt.Errorf("insert red packet winner: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
UPDATE users
SET balance = balance + ($1::NUMERIC / 100), updated_at = NOW()
WHERE id = $2 AND deleted_at IS NULL`, winner.AmountCents, winner.UserID)
		if err != nil {
			return fmt.Errorf("credit red packet winner balance: %w", err)
		}
		if err := requireOneAffected(result, "credit red packet winner balance"); err != nil {
			return err
		}
		businessKey := fmt.Sprintf("red-packet:%d:winner:%d", job.Activity.ID, winnerID)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO red_packet_reward_ledger
    (activity_id, winner_id, user_id, amount_cents, business_key, created_at)
VALUES ($1, $2, $3, $4, $5, NOW())`, job.Activity.ID, winnerID, winner.UserID, winner.AmountCents, businessKey); err != nil {
			return fmt.Errorf("insert red packet reward ledger: %w", err)
		}
	}
	result, err := tx.ExecContext(ctx, `
UPDATE red_packet_activities
SET status = 'completed', completed_at = NOW(), updated_at = NOW()
WHERE id = $1 AND status = 'drawing'`, job.Activity.ID)
	if err != nil {
		return fmt.Errorf("complete red packet activity: %w", err)
	}
	if err := requireOneAffected(result, "complete red packet activity"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit red packet settlement: %w", err)
	}
	return nil
}

func requireOneAffected(result sql.Result, operation string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows affected: %w", operation, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s affected %d rows", operation, affected)
	}
	return nil
}

func redPacketNotFound() error {
	return infraerrors.NotFound("RED_PACKET_NOT_FOUND", "red packet activity not found")
}

func isRedPacketUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
