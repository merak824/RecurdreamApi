package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ListRecentOpenAITTFTSamples hydrates the bounded in-memory/Redis windows
// after a restart. The partition is account + actual transport, and the query
// never scans more than the requested per-window limit for each partition.
func (r *usageLogRepository) ListRecentOpenAITTFTSamples(ctx context.Context, since time.Time, perWindowLimit int) ([]service.OpenAITTFTSample, error) {
	if perWindowLimit <= 0 || perWindowLimit > 10 {
		perWindowLimit = 10
	}
	if r == nil || r.sql == nil {
		return nil, fmt.Errorf("usage log sql database is unavailable")
	}
	const transportExpr = "COALESCE(openai_ttft_context->>'transport', CASE WHEN openai_ws_mode THEN 'responses_ws' ELSE 'http_sse' END)"
	query := fmt.Sprintf(`
		WITH ranked AS (
			SELECT account_id,
				%s AS transport,
				first_token_ms,
				created_at,
				ROW_NUMBER() OVER (
					PARTITION BY account_id, %s
					ORDER BY created_at DESC, id DESC
				) AS row_number
			FROM usage_logs
			WHERE created_at >= $1
				AND stream = TRUE
				AND first_token_ms > 0
				AND client_disconnected = FALSE
				AND openai_ttft_context IS NOT NULL
		)
		SELECT account_id, transport, first_token_ms, created_at
		FROM ranked
		WHERE row_number <= $2
			AND transport IN ('http_sse', 'responses_ws')
		ORDER BY account_id, transport, created_at DESC
	`, transportExpr, transportExpr)
	rows, err := r.sql.QueryContext(ctx, query, since, perWindowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]service.OpenAITTFTSample, 0)
	for rows.Next() {
		var (
			accountID  int64
			transport  string
			firstToken sql.NullInt64
			createdAt  time.Time
		)
		if err := rows.Scan(&accountID, &transport, &firstToken, &createdAt); err != nil {
			return nil, err
		}
		if !firstToken.Valid || firstToken.Int64 <= 0 {
			continue
		}
		parsedTransport := service.OpenAITTFTTransport(transport)
		if !parsedTransport.Valid() {
			continue
		}
		result = append(result, service.OpenAITTFTSample{
			AccountID:  accountID,
			Transport:  parsedTransport,
			ObservedAt: createdAt,
			TTFTMs:     int(firstToken.Int64),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
