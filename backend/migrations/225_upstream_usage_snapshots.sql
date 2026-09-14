-- Append-only observations used to reconcile local cost estimates with the
-- cumulative actual charge reported by each upstream API key.
CREATE TABLE IF NOT EXISTS upstream_usage_snapshots (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    observed_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    http_status INTEGER NOT NULL DEFAULT 0,
    cumulative_actual_cost DECIMAL(30, 10),
    balance DECIMAL(30, 10),
    daily_usage JSONB,
    model_usage JSONB,
    error_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT upstream_usage_snapshots_status_check CHECK (
        status IN ('ok', 'reset', 'unauthorized', 'unsupported', 'failed', 'invalid_response')
    )
);

CREATE INDEX IF NOT EXISTS upstream_usage_snapshots_account_observed_idx
    ON upstream_usage_snapshots (account_id, observed_at DESC, id DESC);

COMMENT ON TABLE upstream_usage_snapshots IS
    'Append-only sanitized cumulative upstream billing observations; raw responses and credentials are never stored';
