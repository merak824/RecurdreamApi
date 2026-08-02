ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS upstream_first_token_ms INTEGER,
    ADD COLUMN IF NOT EXISTS client_disconnected BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS openai_ttft_context JSONB;
