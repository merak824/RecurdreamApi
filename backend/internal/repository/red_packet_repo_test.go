package repository

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

var redPacketActivityTestColumns = []string{
	"id", "period_no", "name", "message", "packet_type", "total_amount_cents",
	"target_participants", "winner_count", "participant_count", "status",
	"recharge_threshold_cents", "invitation_points_threshold", "invitation_points_cost",
	"recharge_priority", "published_at", "drawing_at", "completed_at", "canceled_at",
	"created_at", "updated_at",
}

func redPacketActivityRow(now time.Time, participantCount, target int, status string) *sqlmock.Rows {
	return sqlmock.NewRows(redPacketActivityTestColumns).AddRow(
		int64(7), int64(12), "Summer draw", "Good luck", service.RedPacketTypeLucky, int64(500),
		target, 2, participantCount, status,
		int64(100), 2, 2, true, now, nil, nil, nil, now, now,
	)
}

func TestRedPacketRepositoryGetActivityUsesLoginAccountForWinnerDisplay(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	now := time.Now().UTC()

	mock.ExpectQuery(`(?s)FROM red_packet_activities.*WHERE id = \$1 AND status <> 'draft'`).
		WithArgs(int64(7)).
		WillReturnRows(redPacketActivityRow(now, 1, 1, service.RedPacketStatusCompleted))
	mock.ExpectQuery(`(?s)SELECT w\.user_id, COALESCE\(NULLIF\(u\.email, ''\), u\.username\).*FROM red_packet_winners`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "amount_cents", "is_luckiest", "credited_at"}).
			AddRow(int64(42), "redpacket-test-20260731-01@example.com", int64(100), true, now))

	repo := NewRedPacketRepository(db)
	detail, err := repo.GetActivity(context.Background(), 7, 0)
	require.NoError(t, err)
	require.Len(t, detail.Winners, 1)
	require.Equal(t, "redpacket-test-20260731-01@example.com", detail.Winners[0].Username)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRedPacketRepositoryParticipateUsesRechargeWithoutSpendingPoints(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM red_packet_activities.*FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(redPacketActivityRow(now, 1, 3, service.RedPacketStatusActive))
	mock.ExpectQuery(`(?s)SELECT status, deleted_at.*FROM users.*FOR UPDATE`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "deleted_at"}).AddRow(service.StatusActive, nil))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(7), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(`(?s)FROM payment_orders.*order_type = 'balance'.*completed_at IS NOT NULL`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"net_recharge_cents"}).AddRow(int64(150)))
	mock.ExpectQuery(`(?s)INSERT INTO red_packet_participants.*RETURNING id`).
		WithArgs(int64(7), int64(42), service.RedPacketQualificationRecharge, int64(150), 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(88)))
	mock.ExpectQuery(`SELECT COALESCE`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"lottery_points"}).AddRow(4))
	mock.ExpectExec(`UPDATE red_packet_activities`).
		WithArgs(2, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewRedPacketRepository(db)
	result, err := repo.Participate(context.Background(), 7, 42, []byte("01234567890123456789012345678901"))
	require.NoError(t, err)
	require.Equal(t, service.RedPacketQualificationRecharge, result.QualificationType)
	require.Zero(t, result.PointsSpent)
	require.Equal(t, 4, result.LotteryPoints)
	require.Equal(t, 2, result.Activity.ParticipantCount)
	require.False(t, result.TriggeredDrawing)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRedPacketRepositoryParticipateConsumesPointsAndTriggersDrawing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	now := time.Now().UTC()
	seed := []byte("abcdefghijklmnopqrstuvwxyz123456")

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM red_packet_activities.*FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(redPacketActivityRow(now, 1, 2, service.RedPacketStatusActive))
	mock.ExpectQuery(`(?s)SELECT status, deleted_at.*FROM users.*FOR UPDATE`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "deleted_at"}).AddRow(service.StatusActive, nil))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(7), int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(`(?s)FROM payment_orders.*order_type = 'balance'.*completed_at IS NOT NULL`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{"net_recharge_cents"}).AddRow(int64(0)))
	mock.ExpectQuery(`(?s)SELECT lottery_points.*FROM user_affiliates.*FOR UPDATE`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{"lottery_points"}).AddRow(3))
	mock.ExpectQuery(`(?s)INSERT INTO red_packet_participants.*RETURNING id`).
		WithArgs(int64(7), int64(51), service.RedPacketQualificationPoints, int64(0), 2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	mock.ExpectQuery(`(?s)UPDATE user_affiliates.*lottery_points = lottery_points -.*RETURNING lottery_points`).
		WithArgs(2, int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{"lottery_points"}).AddRow(1))
	mock.ExpectExec(`(?s)INSERT INTO invite_lottery_point_ledger`).
		WithArgs(int64(51), 2, int64(7), int64(99), "red-packet:7:user:51").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)UPDATE red_packet_activities.*status = 'drawing'.*draw_seed`).
		WithArgs(2, seed, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewRedPacketRepository(db)
	result, err := repo.Participate(context.Background(), 7, 51, seed)
	require.NoError(t, err)
	require.Equal(t, service.RedPacketQualificationPoints, result.QualificationType)
	require.Equal(t, 2, result.PointsSpent)
	require.Equal(t, 1, result.LotteryPoints)
	require.True(t, result.TriggeredDrawing)
	require.Equal(t, service.RedPacketStatusDrawing, result.Activity.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRedPacketRepositoryParticipateDuplicateRollsBackBeforeEligibilityMutation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM red_packet_activities.*FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(redPacketActivityRow(now, 1, 3, service.RedPacketStatusActive))
	mock.ExpectQuery(`(?s)SELECT status, deleted_at.*FROM users.*FOR UPDATE`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "deleted_at"}).AddRow(service.StatusActive, nil))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(7), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectRollback()

	repo := NewRedPacketRepository(db)
	_, err = repo.Participate(context.Background(), 7, 42, make([]byte, 32))
	require.Error(t, err)
	require.Equal(t, "RED_PACKET_ALREADY_PARTICIPATED", infraerrors.Reason(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRedPacketRepositorySettleCreditsBalancesAndCompletesAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	job := service.RedPacketDrawJob{Activity: service.RedPacketActivity{ID: 7}}
	winners := []service.RedPacketWinnerAllocation{
		{ParticipantID: 91, UserID: 41, AmountCents: 120, IsLuckiest: true},
		{ParticipantID: 92, UserID: 42, AmountCents: 180},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, packet_type, total_amount_cents, winner_count FROM red_packet_activities WHERE id = $1 FOR UPDATE")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "packet_type", "total_amount_cents", "winner_count"}).
			AddRow(service.RedPacketStatusDrawing, service.RedPacketTypeLucky, int64(300), 2))
	for i, winner := range winners {
		mock.ExpectQuery(`(?s)SELECT EXISTS.*red_packet_participants`).
			WithArgs(int64(7), winner.ParticipantID, winner.UserID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`(?s)INSERT INTO red_packet_winners.*RETURNING id`).
			WithArgs(int64(7), winner.ParticipantID, winner.UserID, winner.AmountCents, winner.IsLuckiest).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100 + i)))
		mock.ExpectExec(`(?s)UPDATE users.*balance = balance`).
			WithArgs(winner.AmountCents, winner.UserID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`(?s)INSERT INTO red_packet_reward_ledger`).
			WithArgs(int64(7), int64(100+i), winner.UserID, winner.AmountCents, regexpMatcher("^red-packet:7:winner:")).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectExec(`(?s)UPDATE red_packet_activities.*status = 'completed'`).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewRedPacketRepository(db)
	require.NoError(t, repo.Settle(context.Background(), job, winners))
	require.NoError(t, mock.ExpectationsWereMet())
}

type regexpMatcher string

func (m regexpMatcher) Match(value driver.Value) bool {
	s, ok := value.(string)
	return ok && regexp.MustCompile(string(m)).MatchString(s)
}
