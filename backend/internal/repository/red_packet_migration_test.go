package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestRedPacketMigrationDefinesAtomicAndIdempotentContracts(t *testing.T) {
	content, err := migrations.FS.ReadFile("191_red_packet_activity.sql")
	require.NoError(t, err)
	sqlText := strings.ToLower(string(content))

	for _, required := range []string{
		"add column if not exists lottery_points",
		"create table if not exists invite_lottery_point_ledger",
		"create table if not exists red_packet_activities",
		"create table if not exists red_packet_participants",
		"create table if not exists red_packet_winners",
		"create table if not exists red_packet_reward_ledger",
		"where status in ('active', 'drawing')",
		"unique (activity_id, user_id)",
		"unique (business_key)",
		"total_amount_cents >= winner_count",
		"packet_type <> 'fixed' or total_amount_cents % winner_count = 0",
	} {
		require.Contains(t, sqlText, required)
	}
}
