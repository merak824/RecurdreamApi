package repository

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPrepareUsageLogInsertPersistsOpenAITTFTFieldsAtTail(t *testing.T) {
	upstream := 240
	contextValue := &service.OpenAITTFTContext{
		Version:              1,
		Transport:            "http_sse",
		UpstreamFirstTokenMs: &upstream,
		SampleCount:          10,
		SampleP90Ms:          intPtr(400),
	}
	prepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:               1,
		APIKeyID:             2,
		AccountID:            3,
		RequestID:            "req-ttft-fields",
		Model:                "gpt-5",
		CreatedAt:            time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
		UpstreamFirstTokenMs: &upstream,
		ClientDisconnected:   true,
		OpenAITTFTContext:    contextValue,
	})

	require.Len(t, prepared.args, len(usageLogInsertArgTypes))
	require.Equal(t, sql.NullInt64{Int64: int64(upstream), Valid: true}, prepared.args[len(prepared.args)-3])
	require.Equal(t, true, prepared.args[len(prepared.args)-2])
	payloadValue, ok := prepared.args[len(prepared.args)-1].(sql.NullString)
	require.True(t, ok)
	require.True(t, payloadValue.Valid)
	var decoded service.OpenAITTFTContext
	require.NoError(t, json.Unmarshal([]byte(payloadValue.String), &decoded))
	require.Equal(t, contextValue.Version, decoded.Version)
	require.Equal(t, contextValue.Transport, decoded.Transport)
}

func TestScanUsageLogHydratesOpenAITTFTFields(t *testing.T) {
	now := time.Now().UTC()
	upstream := 321
	payload := `{"version":1,"transport":"responses_ws","sample_count":10,"sample_p50_ms":120,"sample_p90_ms":300}`
	values := usageLogScanTestValues(now)
	values = append(values, sql.NullInt64{Int64: int64(upstream), Valid: true}, true, sql.NullString{String: payload, Valid: true})

	log, err := scanUsageLog(usageLogScannerStub{values: values})
	require.NoError(t, err)
	require.NotNil(t, log.UpstreamFirstTokenMs)
	require.Equal(t, upstream, *log.UpstreamFirstTokenMs)
	require.True(t, log.ClientDisconnected)
	require.NotNil(t, log.OpenAITTFTContext)
	require.Equal(t, "responses_ws", log.OpenAITTFTContext.Transport)
	require.Equal(t, 300, *log.OpenAITTFTContext.SampleP90Ms)
}

func TestUsageLogDTOExposesTTFTFieldsOnlyForAdmin(t *testing.T) {
	upstream := 123
	log := &service.UsageLog{
		ID:                   1,
		Model:                "gpt-5",
		UpstreamFirstTokenMs: &upstream,
		ClientDisconnected:   true,
		OpenAITTFTContext:    &service.OpenAITTFTContext{Version: 1, Transport: "http_sse"},
	}

	userPayload, err := json.Marshal(dto.UsageLogFromService(log))
	require.NoError(t, err)
	require.NotContains(t, string(userPayload), "upstream_first_token_ms")
	require.NotContains(t, string(userPayload), "openai_ttft_context")

	adminPayload, err := json.Marshal(dto.UsageLogFromServiceAdmin(log))
	require.NoError(t, err)
	require.Contains(t, string(adminPayload), "upstream_first_token_ms")
	require.Contains(t, string(adminPayload), "openai_ttft_context")
}

func usageLogScanTestValues(now time.Time) []any {
	return []any{
		int64(1), int64(10), int64(20), int64(30),
		sql.NullString{Valid: true, String: "req-ttft-scan"}, "gpt-5",
		sql.NullString{Valid: true, String: "gpt-5"}, sql.NullString{}, sql.NullString{}, sql.NullBool{},
		sql.NullInt64{}, sql.NullInt64{},
		1, 2, 3, 4, 5, 6,
		0, 0.0, 0, 0.0, 0.1, 0.2, 0.3, 0.4, 1.0, 0.9, 1.0,
		sql.NullFloat64{}, int16(service.BillingTypeBalance), int16(service.RequestTypeStream), true, false,
		sql.NullInt64{}, sql.NullInt64{}, sql.NullString{}, sql.NullString{}, 0,
		sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{},
		0, sql.NullString{}, sql.NullInt64{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{},
		false, false, sql.NullInt64{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullFloat64{}, sql.NullString{}, now,
	}
}

func intPtr(v int) *int { return &v }
