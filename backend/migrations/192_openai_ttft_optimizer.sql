ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS upstream_first_token_ms INTEGER,
    ADD COLUMN IF NOT EXISTS client_disconnected BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS openai_ttft_context JSONB;

CREATE INDEX IF NOT EXISTS idx_usage_logs_openai_ttft_hydration
    ON usage_logs (account_id, created_at DESC)
    WHERE upstream_first_token_ms IS NOT NULL
      AND client_disconnected = FALSE;

CREATE INDEX IF NOT EXISTS idx_usage_logs_openai_first_token_hydration
    ON usage_logs (account_id, created_at DESC)
    WHERE first_token_ms IS NOT NULL
      AND client_disconnected = FALSE
      AND openai_ttft_context IS NOT NULL;
