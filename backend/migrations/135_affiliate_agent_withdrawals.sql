-- Agent affiliate rebates and withdrawal workflow.

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS rebate_mode VARCHAR(20) NOT NULL DEFAULT 'user';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS agent_aff_quota DECIMAL(20,8) NOT NULL DEFAULT 0;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS agent_aff_frozen_quota DECIMAL(20,8) NOT NULL DEFAULT 0;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS agent_aff_history_quota DECIMAL(20,8) NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_rebate_mode
    ON user_affiliates(rebate_mode);

CREATE INDEX IF NOT EXISTS idx_user_affiliates_agent_aff_quota
    ON user_affiliates(agent_aff_quota);

COMMENT ON COLUMN user_affiliates.rebate_mode IS 'Invite rebate mode captured when this user was bound: user|agent';
COMMENT ON COLUMN user_affiliates.agent_aff_quota IS 'Agent rebate amount available to transfer or withdraw';
COMMENT ON COLUMN user_affiliates.agent_aff_frozen_quota IS 'Agent rebate amount frozen until thaw';
COMMENT ON COLUMN user_affiliates.agent_aff_history_quota IS 'Total historical agent rebate amount';

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS rebate_mode VARCHAR(20) NOT NULL DEFAULT 'user';

COMMENT ON COLUMN user_affiliate_ledger.rebate_mode IS 'Rebate bucket for this ledger entry: user|agent';

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_rebate_mode
    ON user_affiliate_ledger(rebate_mode);

CREATE INDEX IF NOT EXISTS idx_ual_agent_frozen_thaw
    ON user_affiliate_ledger (user_id, frozen_until)
    WHERE frozen_until IS NOT NULL AND rebate_mode = 'agent';

CREATE TABLE IF NOT EXISTS user_affiliate_withdrawals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(20,8) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    collection_qr_data TEXT NOT NULL,
    collection_qr_mime VARCHAR(64) NULL,
    collection_qr_size INTEGER NULL,
    payment_proof_data TEXT NULL,
    payment_proof_mime VARCHAR(64) NULL,
    payment_proof_size INTEGER NULL,
    reject_reason TEXT NULL,
    admin_note TEXT NULL,
    processed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    processed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawals_user_id
    ON user_affiliate_withdrawals(user_id);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawals_status
    ON user_affiliate_withdrawals(status);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawals_created_at
    ON user_affiliate_withdrawals(created_at);

COMMENT ON TABLE user_affiliate_withdrawals IS 'Agent affiliate withdrawal requests and processed records';
COMMENT ON COLUMN user_affiliate_withdrawals.status IS 'pending|paid|rejected';
COMMENT ON COLUMN user_affiliate_withdrawals.collection_qr_data IS 'Agent collection QR image data URL';
COMMENT ON COLUMN user_affiliate_withdrawals.payment_proof_data IS 'Admin payment proof image data URL';
