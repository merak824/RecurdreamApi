-- Freeze the evidence used by the admin profit monitor at request time.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS profit_cost_source TEXT NOT NULL DEFAULT 'unknown';

ALTER TABLE usage_logs
    DROP CONSTRAINT IF EXISTS usage_logs_profit_cost_source_check;

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_profit_cost_source_check
    CHECK (profit_cost_source IN ('upstream_probe', 'group_break_even_estimate', 'unknown'));

COMMENT ON COLUMN usage_logs.profit_cost_source IS
    'Immutable profit-monitor cost source captured when the usage record was created';
