-- Simplify affiliate rebates to one wallet and explicit per-user withdrawal permission.
-- Legacy agent/frozen columns remain for migration compatibility but are no longer active.

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS withdrawal_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE user_affiliate_withdrawals
    ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20);

-- Historical agent withdrawals did not capture a channel. Use a valid legacy
-- placeholder so the new invariant can be enforced for all future rows.
UPDATE user_affiliate_withdrawals
SET payment_method = 'alipay'
WHERE payment_method IS NULL;

ALTER TABLE user_affiliate_withdrawals
    ALTER COLUMN payment_method SET NOT NULL;

DO $$
BEGIN
    ALTER TABLE user_affiliate_withdrawals
        ADD CONSTRAINT user_affiliate_withdrawals_payment_method_check
        CHECK (payment_method IN ('wechat', 'alipay'));
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Snapshot old agents before their role is normalized. Existing exclusive
-- rates win; otherwise use the valid global rate at migration time, or 20%.
WITH old_agents AS (
    SELECT
        ua.user_id,
        COALESCE(
            ua.aff_rebate_rate_percent,
            (
                SELECT LEAST(100, GREATEST(0, value::numeric))
                FROM settings
                WHERE key = 'affiliate_rebate_rate'
                  AND value ~ '^[0-9]+([.][0-9]+)?$'
                LIMIT 1
            ),
            20
        ) AS effective_rate
    FROM user_affiliates ua
    JOIN users u ON u.id = ua.user_id
    WHERE u.role = 'agent'
)
UPDATE user_affiliates ua
SET aff_rebate_rate_percent = old_agents.effective_rate,
    withdrawal_enabled = TRUE,
    updated_at = NOW()
FROM old_agents
WHERE ua.user_id = old_agents.user_id;

-- Zeroing every source compatibility column makes this conversion idempotent.
UPDATE user_affiliates
SET aff_quota = aff_quota + aff_frozen_quota + agent_aff_quota + agent_aff_frozen_quota,
    aff_history_quota = aff_history_quota + agent_aff_history_quota,
    aff_frozen_quota = 0,
    agent_aff_quota = 0,
    agent_aff_frozen_quota = 0,
    agent_aff_history_quota = 0,
    rebate_mode = 'user',
    updated_at = NOW()
WHERE aff_frozen_quota <> 0
   OR agent_aff_quota <> 0
   OR agent_aff_frozen_quota <> 0
   OR agent_aff_history_quota <> 0
   OR rebate_mode <> 'user';

UPDATE user_affiliate_ledger
SET rebate_mode = 'user',
    frozen_until = NULL
WHERE rebate_mode <> 'user'
   OR frozen_until IS NOT NULL;

UPDATE users
SET role = 'user',
    updated_at = NOW()
WHERE role = 'agent';

COMMENT ON COLUMN user_affiliates.withdrawal_enabled IS
    'Whether an exclusively configured affiliate may request WeChat or Alipay withdrawal';
COMMENT ON COLUMN user_affiliate_withdrawals.payment_method IS
    'Collection channel selected for this request: wechat|alipay';
