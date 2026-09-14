package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserMonitorViewToPublicItemRedactsInternalFields(t *testing.T) {
	latency := 1200
	ping := 90
	view := &service.UserMonitorView{
		ID:                   42,
		Name:                 "GPT PLUS",
		Provider:             "openai",
		GroupName:            "internal-vip-group",
		PrimaryModel:         "gpt-5.6-sol",
		PrimaryStatus:        "operational",
		PrimaryLatencyMs:     &latency,
		PrimaryPingLatencyMs: &ping,
		Availability7d:       99.5,
		ExtraModels: []service.ExtraModelStatus{
			{Model: "internal-fallback-model", Status: "operational"},
		},
		Timeline: []service.UserMonitorTimelinePoint{
			{
				Status:        "operational",
				LatencyMs:     &latency,
				PingLatencyMs: &ping,
				CheckedAt:     time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	payload, err := json.Marshal(userMonitorViewToPublicItem(view))
	require.NoError(t, err)
	require.JSONEq(t, `{
		"name":"GPT PLUS",
		"provider":"openai",
		"primary_model":"gpt-5.6-sol",
		"primary_status":"operational",
		"primary_latency_ms":1200,
		"primary_ping_latency_ms":90,
		"availability_7d":99.5,
		"timeline":[{
			"status":"operational",
			"latency_ms":1200,
			"ping_latency_ms":90,
			"checked_at":"2026-07-27T00:00:00Z"
		}]
	}`, string(payload))
	require.NotContains(t, string(payload), "42")
	require.NotContains(t, string(payload), "internal-vip-group")
	require.NotContains(t, string(payload), "internal-fallback-model")
}
