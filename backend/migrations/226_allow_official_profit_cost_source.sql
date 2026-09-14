-- Official provider credentials have no relay-side upstream charge in the
-- profit monitor. Keep this source immutable on each usage record.
ALTER TABLE usage_logs
    DROP CONSTRAINT IF EXISTS usage_logs_profit_cost_source_check;

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_profit_cost_source_check
    CHECK (profit_cost_source IN ('upstream_probe', 'group_break_even_estimate', 'official_upstream', 'unknown'));
