package service

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestRedeemServiceGetUserBalanceHistoryAggregatesAllCreditSources(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	createdAt := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)WITH balance_history AS .*FROM redeem_codes.*FROM user_affiliate_ledger.*FROM red_packet_reward_ledger.*ORDER BY occurred_at DESC`).
		WithArgs(int64(42), "", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "type", "amount", "occurred_at", "reference", "description", "total_count",
		}).
			AddRow("red-packet:9", RedeemTypeRedPacketReward, 1.25, createdAt, "12", "周末红包", int64(3)).
			AddRow("affiliate:8", RedeemTypeAffiliateBalance, 2.50, createdAt.Add(-time.Minute), "", "", int64(3)).
			AddRow("redeem:7", RedeemTypeBalance, 10.00, createdAt.Add(-2*time.Minute), "ABCD-EFGH", "", int64(3)))

	service := &RedeemService{entClient: client}
	items, total, err := service.GetUserBalanceHistory(context.Background(), 42, 1, 20, "")

	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, items, 3)
	require.Equal(t, RedeemTypeRedPacketReward, items[0].Type)
	require.Equal(t, 1.25, items[0].Amount)
	require.Equal(t, "12", items[0].Reference)
	require.Equal(t, "周末红包", items[0].Description)
	require.Equal(t, RedeemTypeAffiliateBalance, items[1].Type)
	require.Equal(t, RedeemTypeBalance, items[2].Type)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRedeemServiceGetUserBalanceHistoryRejectsUnknownTypeWithoutQuery(t *testing.T) {
	service := &RedeemService{}
	items, total, err := service.GetUserBalanceHistory(context.Background(), 42, 1, 20, "unknown")

	require.NoError(t, err)
	require.Empty(t, items)
	require.Zero(t, total)
}
