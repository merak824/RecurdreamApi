# Affiliate Referral Simplification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the role-based agent rebate branch with one invitation rebate wallet plus administrator-managed exclusive rates and per-user cash-withdrawal permission.

**Architecture:** Keep the existing invitation profile, ledger, withdrawal table, services, handlers, and Vue views. Collapse active reads and writes onto `aff_quota`/`aff_history_quota`, make exclusive configuration an atomic repository operation, authorize cash withdrawal from `withdrawal_enabled`, and preserve legacy database columns only for upgrade compatibility.

**Tech Stack:** Go 1.26, Gin, PostgreSQL SQL repositories and migrations, Testify/sqlmock, Vue 3, TypeScript, Pinia, Tailwind CSS, Vitest, vue-test-utils, pnpm.

---

## File Map

- `backend/migrations/193_simplify_affiliate_referrals.sql`: add withdrawal permission/channel fields and migrate old agent/frozen balances without dropping compatibility columns.
- `backend/migrations/affiliate_referral_simplification_migration_test.go`: guard the data-conversion and constraints in migration 193.
- `backend/internal/service/affiliate_service.go`: expose the simplified detail/configuration/transfer/withdrawal rules and remove role, mode, thaw, cap, and agent-usage behavior.
- `backend/internal/service/affiliate_service_test.go`: specify unified accrual, arbitrary transfers, withdrawal authorization, channel validation, and atomic exclusive settings.
- `backend/internal/repository/affiliate_repo.go`: use one wallet, persist payment method, transact exclusive settings, and remove active agent/frozen SQL.
- `backend/internal/repository/affiliate_repo_test.go`: assert simplified SQL no longer selects legacy wallet columns.
- `backend/internal/repository/affiliate_repo_integration_test.go`: verify transactional accrual, transfers, withdrawals, and rejected-request refunds against PostgreSQL.
- `backend/internal/service/payment_fulfillment.go`: calculate paid recharge and subscription rebates from `PayAmount`.
- `backend/internal/service/payment_fulfillment_test.go`: lock the paid-amount calculation and order idempotency behavior.
- `backend/internal/service/redeem_service.go`, `backend/internal/service/redeem_service_test.go`: remove redeem-code rebate issuance and its obsolete dependency path.
- `backend/internal/service/admin_user.go`, `backend/internal/service/admin_user_test.go`: remove manual-admin-recharge rebate issuance.
- `backend/internal/handler/user_handler.go`, `backend/internal/handler/user_handler_test.go`: accept arbitrary amounts and a `wechat|alipay` payment method.
- `backend/internal/handler/admin/affiliate_handler.go`, `backend/internal/handler/admin/affiliate_handler_test.go`: accept one atomic exclusive-user payload and remove agent-usage/batch-rate APIs.
- `backend/internal/server/routes/admin.go`, `backend/internal/server/routes/user.go`: connect the existing withdrawal review routes and retain unrelated red-packet routes already present in the worktree.
- `backend/internal/server/routes/affiliate_routes_test.go`: verify user and admin affiliate route registration.
- `frontend/src/api/user.ts`, `frontend/src/api/admin/affiliates.ts`, `frontend/src/types/index.ts`: align request and response types with the simplified API.
- `frontend/src/views/user/AffiliateView.vue`, `frontend/src/views/user/__tests__/AffiliateView.spec.ts`: show one wallet, arbitrary transfer input, and permission-gated WeChat/Alipay withdrawal.
- `frontend/src/views/admin/SettingsView.vue`, `frontend/src/views/admin/__tests__/SettingsView.spec.ts`: retain only the global switch/rate and exclusive-user configuration table.
- `frontend/src/views/admin/affiliates/AdminAffiliateWithdrawalsView.vue`, `frontend/src/views/admin/affiliates/__tests__/AdminAffiliateWithdrawalsView.spec.ts`: display channel and drive paid/rejected review actions.
- `frontend/src/router/index.ts`, `frontend/src/components/layout/AppSidebar.vue`, `frontend/src/components/layout/__tests__/AppSidebar.spec.ts`: register and link withdrawal review while preserving current red-packet navigation.
- `frontend/src/i18n/locales/{zh,en}/affiliate.ts`, `frontend/src/i18n/locales/{zh,en}/admin/affiliate.ts`, `frontend/src/i18n/locales/{zh,en}/admin/settings.ts`, `frontend/src/i18n/locales/{zh,en}/common.ts`: replace agent/frozen terminology with exclusive-user and withdrawal-channel copy.

### Task 1: Migrate Legacy Affiliate Data

**Files:**
- Create: `backend/migrations/193_simplify_affiliate_referrals.sql`
- Create: `backend/migrations/affiliate_referral_simplification_migration_test.go`

- [ ] **Step 1: Write the failing migration regression test**

Create a test that reads migration 193 from the embedded migration FS and requires these exact safety operations:

```go
func TestMigration193SimplifiesAffiliateReferrals(t *testing.T) {
	content, err := FS.ReadFile("193_simplify_affiliate_referrals.sql")
	require.NoError(t, err)
	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS withdrawal_enabled BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20)")
	require.Contains(t, sql, "aff_quota + aff_frozen_quota + agent_aff_quota + agent_aff_frozen_quota")
	require.Contains(t, sql, "aff_history_quota + agent_aff_history_quota")
	require.Contains(t, sql, "WHERE u.role = 'agent'")
	require.Contains(t, sql, "SET role = 'user'")
	require.Contains(t, sql, "withdrawal_enabled = TRUE")
	require.Contains(t, sql, "CHECK (payment_method IN ('wechat', 'alipay'))")
}
```

- [ ] **Step 2: Run the migration test and confirm it fails**

Run: `cd backend && go test ./migrations -run TestMigration193SimplifiesAffiliateReferrals -count=1`

Expected: FAIL because `193_simplify_affiliate_referrals.sql` does not exist.

- [ ] **Step 3: Add the idempotent migration**

Implement SQL with this order so old agents are identified before role conversion and repeated execution cannot double-credit balances:

```sql
ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS withdrawal_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE user_affiliate_withdrawals
    ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20);

DO $$ BEGIN
    ALTER TABLE user_affiliate_withdrawals
        ADD CONSTRAINT user_affiliate_withdrawals_payment_method_check
        CHECK (payment_method IN ('wechat', 'alipay'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

WITH old_agents AS (
    SELECT ua.user_id,
           COALESCE(ua.aff_rebate_rate_percent,
                    (SELECT NULLIF(value, '')::numeric FROM settings WHERE key = 'affiliate_rebate_rate'),
                    20) AS effective_rate
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

UPDATE user_affiliates
SET aff_quota = aff_quota + aff_frozen_quota + agent_aff_quota + agent_aff_frozen_quota,
    aff_history_quota = aff_history_quota + agent_aff_history_quota,
    aff_frozen_quota = 0,
    agent_aff_quota = 0,
    agent_aff_frozen_quota = 0,
    agent_aff_history_quota = 0,
    rebate_mode = 'user',
    updated_at = NOW()
WHERE aff_frozen_quota <> 0 OR agent_aff_quota <> 0
   OR agent_aff_frozen_quota <> 0 OR agent_aff_history_quota <> 0
   OR rebate_mode <> 'user';

UPDATE user_affiliate_ledger
SET rebate_mode = 'user', frozen_until = NULL
WHERE rebate_mode <> 'user' OR frozen_until IS NOT NULL;

UPDATE users SET role = 'user', updated_at = NOW() WHERE role = 'agent';
```

Also backfill a valid payment method for legacy withdrawal rows before making the column `NOT NULL`.

- [ ] **Step 4: Run migration tests**

Run: `cd backend && go test ./migrations -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the migration**

```bash
git add -f backend/migrations/193_simplify_affiliate_referrals.sql backend/migrations/affiliate_referral_simplification_migration_test.go
git commit -m "feat: migrate affiliate rebates to unified wallet"
```

### Task 2: Simplify the Service Contract and Detail Model

**Files:**
- Modify: `backend/internal/service/affiliate_service.go`
- Modify: `backend/internal/service/affiliate_service_test.go`

- [ ] **Step 1: Write failing service tests for the public model**

Add tests that build an `AffiliateSummary` with a custom rate and withdrawal permission and expect the returned detail to contain only the unified wallet values:

```go
func TestGetAffiliateDetailUsesUnifiedWalletAndWithdrawalPermission(t *testing.T) {
	rate := 35.0
	repo := &affiliateRepositoryStub{summary: &AffiliateSummary{
		UserID: 7, AffCode: "DREAM7", AffQuota: 12.5,
		AffHistoryQuota: 40, AffRebateRatePercent: &rate,
		WithdrawalEnabled: true,
	}}
	svc := NewAffiliateService(repo, nil, nil, nil)
	detail, err := svc.GetAffiliateDetail(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 12.5, detail.AffQuota)
	require.Equal(t, 40.0, detail.AffHistoryQuota)
	require.True(t, detail.WithdrawalEnabled)
	require.Equal(t, 35.0, detail.EffectiveRebateRatePercent)
}
```

Add compile-time expectations for the repository contract:

```go
type AffiliateRepository interface {
	EnsureUserAffiliate(context.Context, int64) (*AffiliateSummary, error)
	GetAffiliateByCode(context.Context, string) (*AffiliateSummary, error)
	BindInviter(context.Context, int64, int64) (bool, error)
	AccrueQuota(context.Context, int64, int64, float64, *int64) (bool, error)
	TransferQuotaToBalance(context.Context, int64, float64) (float64, float64, error)
	CreateWithdrawal(context.Context, int64, float64, string, AffiliateImageData) (*AffiliateWithdrawal, error)
}
```

- [ ] **Step 2: Run focused service tests and confirm failure**

Run: `cd backend && go test ./internal/service -run 'Test(GetAffiliateDetailUsesUnifiedWallet|AccrueInviteRebate|TransferAffiliateQuota|CreateWithdrawal)' -count=1`

Expected: FAIL because the current service still branches on role/mode/frozen/agent fields.

- [ ] **Step 3: Replace role-based models and methods**

Make `AffiliateSummary`/`AffiliateDetail` expose the unified fields and permission:

```go
type AffiliateSummary struct {
	UserID int64 `json:"user_id"`
	AffCode string `json:"aff_code"`
	AffCodeCustom bool `json:"aff_code_custom"`
	AffRebateRatePercent *float64 `json:"aff_rebate_rate_percent,omitempty"`
	WithdrawalEnabled bool `json:"withdrawal_enabled"`
	InviterID *int64 `json:"inviter_id,omitempty"`
	AffCount int `json:"aff_count"`
	AffQuota float64 `json:"aff_quota"`
	AffHistoryQuota float64 `json:"aff_history_quota"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AffiliateWithdrawalInput struct {
	Amount float64
	PaymentMethod string
	CollectionQRData string
}
```

Remove `GetUserRole`, mode parameters, thaw methods, accrued-cap lookup, agent usage methods/types, fixed amount maps, and agent tier constants from active service contracts. `GetAffiliateDetail` always reads `AffQuota`/`AffHistoryQuota`, always lists invitees, and lists withdrawals only when `WithdrawalEnabled` is true.

- [ ] **Step 4: Run focused service tests**

Run: `cd backend && go test ./internal/service -run 'Test(GetAffiliateDetailUsesUnifiedWallet|AccrueInviteRebate|TransferAffiliateQuota|CreateWithdrawal)' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the service contract**

```bash
git add backend/internal/service/affiliate_service.go backend/internal/service/affiliate_service_test.go
git commit -m "refactor: unify affiliate rebate service"
```

### Task 3: Implement Unified Wallet Transactions

**Files:**
- Modify: `backend/internal/repository/affiliate_repo.go`
- Modify: `backend/internal/repository/affiliate_repo_test.go`
- Modify: `backend/internal/repository/affiliate_repo_integration_test.go`

- [ ] **Step 1: Write failing repository tests**

Cover these database invariants:

```go
func TestAffiliateRepositoryTransferUsesOnlyUnifiedQuota(t *testing.T) {
	require.NotContains(t, affiliateTransferQuotaSQL, "agent_aff_quota")
	require.NotContains(t, affiliateTransferQuotaSQL, "aff_frozen_quota")
	require.Contains(t, affiliateTransferQuotaSQL, "aff_quota = aff_quota - $2")
}

func TestAffiliateRepositoryWithdrawalCarriesPaymentMethod(t *testing.T) {
	require.Contains(t, affiliateCreateWithdrawalSQL, "payment_method")
	require.Contains(t, affiliateCreateWithdrawalSQL, "aff_quota = aff_quota - $2")
}
```

Extend the PostgreSQL integration test to accrue `3.5`, transfer `1.25`, create a `wechat` withdrawal for `1`, reject it, and assert the final available balance is `2.25`.

- [ ] **Step 2: Run repository tests and confirm failure**

Run: `cd backend && go test ./internal/repository -run Affiliate -count=1`

Expected: FAIL because repository SQL still selects buckets by rebate mode.

- [ ] **Step 3: Rewrite repository SQL around one wallet**

Use a single atomic balance update for transfers:

```sql
UPDATE user_affiliates
SET aff_quota = aff_quota - $2, updated_at = NOW()
WHERE user_id = $1 AND aff_quota >= $2
RETURNING aff_quota
```

Create withdrawals inside the existing transaction helper by decrementing `aff_quota` and inserting:

```sql
INSERT INTO user_affiliate_withdrawals
    (user_id, amount, status, payment_method, collection_qr_data,
     collection_qr_mime, collection_qr_size, created_at, updated_at)
VALUES ($1, $2, 'pending', $3, $4, $5, $6, NOW(), NOW())
RETURNING id, user_id, amount, status, payment_method,
          collection_qr_data, collection_qr_mime, collection_qr_size,
          created_at, updated_at
```

On rejection, lock the pending row, set it to `rejected`, and execute `aff_quota = aff_quota + withdrawal.amount` in the same transaction. Accrual always increments `aff_quota` and `aff_history_quota` and writes ledger mode `user` with `frozen_until = NULL` for compatibility.

- [ ] **Step 4: Run repository tests**

Run: `cd backend && go test ./internal/repository -run Affiliate -count=1`

Expected: PASS, including the integration test when `TEST_DATABASE_URL` is configured; otherwise the repository's established integration skip is acceptable.

- [ ] **Step 5: Commit repository changes**

```bash
git add backend/internal/repository/affiliate_repo.go backend/internal/repository/affiliate_repo_test.go backend/internal/repository/affiliate_repo_integration_test.go
git commit -m "refactor: transact affiliate funds in one wallet"
```

### Task 4: Enforce Exclusive Configuration and Withdrawal Rules

**Files:**
- Modify: `backend/internal/service/affiliate_service.go`
- Modify: `backend/internal/service/affiliate_service_test.go`
- Modify: `backend/internal/repository/affiliate_repo.go`
- Modify: `backend/internal/repository/affiliate_repo_test.go`

- [ ] **Step 1: Write failing validation and atomicity tests**

Specify all boundary cases with a recording repository stub. The stub returns
`summary` from `EnsureUserAffiliate`, records `withdrawalInput` in
`CreateWithdrawal`, records `transferAmount` in `TransferQuotaToBalance`, and
sets `exclusiveUpdateCalled` in `UpdateExclusiveSettings`:

```go
func TestCreateWithdrawalRequiresExclusivePermission(t *testing.T) {
	repo := &affiliateRepositoryStub{summary: &AffiliateSummary{UserID: 1, AffQuota: 10}}
	_, err := NewAffiliateService(repo, nil, nil, nil).CreateWithdrawal(
		context.Background(), 1,
		AffiliateWithdrawalInput{Amount: 1, PaymentMethod: "wechat", CollectionQRData: onePixelPNG},
	)
	require.ErrorIs(t, err, ErrAffiliateWithdrawalForbidden)
}

func TestCreateWithdrawalAcceptsMinimumOneYuan(t *testing.T) {
	rate := 20.0
	repo := &affiliateRepositoryStub{summary: &AffiliateSummary{
		UserID: 1, AffQuota: 10, AffRebateRatePercent: &rate, WithdrawalEnabled: true,
	}}
	_, err := NewAffiliateService(repo, nil, nil, nil).CreateWithdrawal(
		context.Background(), 1,
		AffiliateWithdrawalInput{Amount: 1, PaymentMethod: "alipay", CollectionQRData: onePixelPNG},
	)
	require.NoError(t, err)
	require.Equal(t, "alipay", repo.withdrawalInput.PaymentMethod)
}

func TestCreateWithdrawalRejectsUnsupportedPaymentMethod(t *testing.T) {
	rate := 20.0
	repo := &affiliateRepositoryStub{summary: &AffiliateSummary{
		UserID: 1, AffQuota: 10, AffRebateRatePercent: &rate, WithdrawalEnabled: true,
	}}
	_, err := NewAffiliateService(repo, nil, nil, nil).CreateWithdrawal(
		context.Background(), 1,
		AffiliateWithdrawalInput{Amount: 1, PaymentMethod: "bank", CollectionQRData: onePixelPNG},
	)
	require.ErrorIs(t, err, ErrAffiliatePaymentMethod)
}

func TestTransferAffiliateQuotaAcceptsArbitraryPositiveAmount(t *testing.T) {
	repo := &affiliateRepositoryStub{}
	_, _, err := NewAffiliateService(repo, nil, nil, nil).TransferAffiliateQuota(
		context.Background(), 1, AffiliateTransferInput{Amount: 1.23},
	)
	require.NoError(t, err)
	require.Equal(t, 1.23, repo.transferAmount)
}

func TestAdminUpdateExclusiveSettingsRequiresFiniteRate(t *testing.T) {
	repo := &affiliateRepositoryStub{}
	svc := NewAffiliateService(repo, nil, nil, nil)
	for _, rate := range []float64{math.NaN(), -0.01, 100.01} {
		err := svc.AdminUpdateExclusiveSettings(context.Background(), 1,
			AffiliateExclusiveSettingsInput{RebateRatePercent: &rate})
		require.Error(t, err)
	}
	require.False(t, repo.exclusiveUpdateCalled)
}

func TestAdminCannotEnableWithdrawalWithoutExclusiveRate(t *testing.T) {
	repo := &affiliateRepositoryStub{}
	err := NewAffiliateService(repo, nil, nil, nil).AdminUpdateExclusiveSettings(
		context.Background(), 1, AffiliateExclusiveSettingsInput{WithdrawalEnabled: true},
	)
	require.ErrorIs(t, err, ErrAffiliateExclusiveRateRequired)
	require.False(t, repo.exclusiveUpdateCalled)
}
```

- [ ] **Step 2: Run validation tests and confirm failure**

Run: `cd backend && go test ./internal/service -run 'Test(CreateWithdrawal|TransferAffiliateQuota|AdminUpdateExclusive)' -count=1`

Expected: FAIL on fixed amount validation, role checks, and separate setting updates.

- [ ] **Step 3: Add the simplified inputs and validation**

Use these service rules:

```go
const affiliateMinimumWithdrawal = 1.0

func validAffiliatePaymentMethod(value string) bool {
	return value == "wechat" || value == "alipay"
}

type AffiliateExclusiveSettingsInput struct {
	AffCode *string
	RebateRatePercent *float64
	WithdrawalEnabled bool
}
```

Transfers accept any finite amount `> 0`. Withdrawals require finite amount `>= 1`, enough `AffQuota`, a non-nil exclusive rate, `WithdrawalEnabled`, a valid channel, and a valid PNG/JPEG/WebP data URL up to 2MB.

Replace independent code/rate updates with:

```go
UpdateExclusiveSettings(ctx context.Context, userID int64, input AffiliateExclusiveSettingsInput) error
ClearExclusiveSettings(ctx context.Context, userID int64) (string, error)
```

`ClearExclusiveSettings` regenerates the system code, clears the custom rate, and disables withdrawal in one transaction.

- [ ] **Step 4: Run service and repository tests**

Run: `cd backend && go test ./internal/service ./internal/repository -run 'Affiliate|Withdrawal' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit rule enforcement**

```bash
git add backend/internal/service/affiliate_service.go backend/internal/service/affiliate_service_test.go backend/internal/repository/affiliate_repo.go backend/internal/repository/affiliate_repo_test.go
git commit -m "feat: configure exclusive affiliate withdrawals"
```

### Task 5: Restrict Rebate Sources to Paid Orders

**Files:**
- Modify: `backend/internal/service/payment_fulfillment.go`
- Modify: `backend/internal/service/payment_fulfillment_test.go`
- Modify: `backend/internal/service/redeem_service.go`
- Modify: `backend/internal/service/redeem_service_test.go`
- Modify: `backend/internal/service/admin_user.go`
- Modify: `backend/internal/service/admin_user_test.go`
- Modify: generated dependency wiring only if constructor removal requires it: `backend/cmd/server/wire_gen.go`

- [ ] **Step 1: Change payment tests to actual paid amount**

For the existing subscription case with `Amount = 9.99`, `PayAmount = 71.36`, and `15%`, change the expected rebate to:

```go
require.InDelta(t, 10.704, accruedAmount, 0.00000001)
```

Add tests proving a positive-balance redeem code and an admin manual balance adjustment do not call `AccrueInviteRebate`.

- [ ] **Step 2: Run the three service test groups and confirm failure**

Run: `cd backend && go test ./internal/service -run 'TestExecute.*Affiliate|Test.*Redeem.*Affiliate|Test.*Admin.*Recharge.*Affiliate' -count=1`

Expected: FAIL because fulfillment uses order face value and non-payment paths can issue rebates.

- [ ] **Step 3: Narrow accrual call sites**

Pass actual paid amount at both paid-order fulfillment sites:

```go
_, err := s.affiliateService.AccrueInviteRebateForOrder(ctx, order.UserID, order.PayAmount, &order.ID)
```

Delete the redeem-level affiliate context marker/call and delete the admin manual recharge call. Remove constructor dependencies only when no longer used elsewhere; regenerate Wire using the repository's established `go generate`/Wire command if signatures change.

- [ ] **Step 4: Run focused payment/redeem/admin tests**

Run: `cd backend && go test ./internal/service -run 'TestExecute.*Affiliate|Test.*Redeem.*Affiliate|Test.*Admin.*Recharge.*Affiliate' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit accrual source changes**

```bash
git add backend/internal/service/payment_fulfillment.go backend/internal/service/payment_fulfillment_test.go backend/internal/service/redeem_service.go backend/internal/service/redeem_service_test.go backend/internal/service/admin_user.go backend/internal/service/admin_user_test.go backend/cmd/server/wire_gen.go
git commit -m "fix: rebate only actual paid order amounts"
```

### Task 6: Align HTTP APIs and Register Review Routes

**Files:**
- Modify: `backend/internal/handler/user_handler.go`
- Modify: `backend/internal/handler/user_handler_test.go`
- Modify: `backend/internal/handler/admin/affiliate_handler.go`
- Modify: `backend/internal/handler/admin/affiliate_handler_test.go`
- Modify: `backend/internal/server/routes/admin.go`
- Modify: `backend/internal/server/routes/user.go`
- Create: `backend/internal/server/routes/affiliate_routes_test.go`

- [ ] **Step 1: Write failing handler and route tests**

Decode and assert the new user request:

```go
type CreateAffiliateWithdrawalRequest struct {
	Amount float64 `json:"amount" binding:"required"`
	PaymentMethod string `json:"payment_method" binding:"required"`
	CollectionQRData string `json:"collection_qr_data" binding:"required"`
}
```

Assert admin configuration accepts all fields in one request:

```go
type UpdateAffiliateUserRequest struct {
	AffCode *string `json:"aff_code"`
	AffRebateRatePercent *float64 `json:"aff_rebate_rate_percent" binding:"required"`
	WithdrawalEnabled bool `json:"withdrawal_enabled"`
}
```

Assert registered routes include:

```text
GET  /api/v1/admin/affiliates/withdrawals
POST /api/v1/admin/affiliates/withdrawals/:id/paid
POST /api/v1/admin/affiliates/withdrawals/:id/reject
```

and exclude `/api/v1/admin/affiliates/users/:user_id/usage` and `/batch-rate`.

- [ ] **Step 2: Run handler/route tests and confirm failure**

Run: `cd backend && go test ./internal/handler ./internal/handler/admin ./internal/server/routes -run 'Affiliate|Withdrawal' -count=1`

Expected: FAIL because payment method is absent and review routes are not registered.

- [ ] **Step 3: Update handlers and routes**

Map the request directly to service input:

```go
AffiliateWithdrawalInput{
	Amount: req.Amount,
	PaymentMethod: strings.ToLower(strings.TrimSpace(req.PaymentMethod)),
	CollectionQRData: req.CollectionQRData,
}
```

Have the admin update handler call one `AdminUpdateExclusiveSettings` method. Delete the batch-rate and agent-usage handlers/parsers. Add withdrawal list/paid/reject routes under the existing affiliate group, retaining the current red-packet additions in shared route files.

- [ ] **Step 4: Run handler/route tests**

Run: `cd backend && go test ./internal/handler ./internal/handler/admin ./internal/server/routes -run 'Affiliate|Withdrawal' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit HTTP changes**

```bash
git add backend/internal/handler/user_handler.go backend/internal/handler/user_handler_test.go backend/internal/handler/admin/affiliate_handler.go backend/internal/handler/admin/affiliate_handler_test.go backend/internal/server/routes/admin.go backend/internal/server/routes/user.go backend/internal/server/routes/affiliate_routes_test.go
git commit -m "feat: expose simplified affiliate withdrawal api"
```

### Task 7: Simplify the User Affiliate Page

**Files:**
- Modify: `frontend/src/api/user.ts`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/views/user/AffiliateView.vue`
- Modify: `frontend/src/views/user/__tests__/AffiliateView.spec.ts`
- Modify: `frontend/src/i18n/locales/zh/affiliate.ts`
- Modify: `frontend/src/i18n/locales/en/affiliate.ts`

- [ ] **Step 1: Use ui-ux-pro-max guidance before editing the page**

Read `ui-ux-pro-max/SKILL.md`, query recommendations for a work-focused SaaS financial/referral page, and retain the project's existing homepage/login/admin gradient language without adding decorative cards or heavy animation.

- [ ] **Step 2: Write failing component tests**

Mock a unified detail response and assert:

```ts
expect(wrapper.text()).not.toContain('affiliate.agent.identity')
expect(wrapper.text()).not.toContain('affiliate.stats.frozenQuota')
expect(wrapper.find('input[name="transfer-amount"]').exists()).toBe(true)
expect(wrapper.find('[data-testid="withdraw-button"]').exists()).toBe(false)
```

Then remount with `withdrawal_enabled: true`, open withdrawal, select Alipay, submit amount `1.23`, and assert:

```ts
expect(createAffiliateWithdrawal).toHaveBeenCalledWith({
	amount: 1.23,
	payment_method: 'alipay',
	collection_qr_data: expect.stringMatching(/^data:image\/png;base64,/),
})
```

- [ ] **Step 3: Run the component test and confirm failure**

Run: `cd frontend && pnpm test:run src/views/user/__tests__/AffiliateView.spec.ts`

Expected: FAIL because agent labels and fixed amount buttons remain.

- [ ] **Step 4: Implement the unified page**

Define the response surface as:

```ts
export interface UserAffiliateDetail {
	user_id: number
	aff_code: string
	inviter_id?: number
	aff_count: number
	aff_quota: number
	aff_history_quota: number
	effective_rebate_rate_percent: number
	withdrawal_enabled: boolean
	invitees: AffiliateInvitee[]
	withdrawals: AffiliateWithdrawal[]
}
```

Replace amount chips with numeric inputs using `min="0.01" step="0.01"` for transfers and `min="1" step="0.01"` for withdrawals. Add a compact segmented `wechat|alipay` control, keep the existing image validation/preview, show withdrawal history only with permission, and remove role/mode/frozen-agent UI.

- [ ] **Step 5: Run user UI tests**

Run: `cd frontend && pnpm test:run src/views/user/__tests__/AffiliateView.spec.ts`

Expected: PASS.

- [ ] **Step 6: Commit the user page**

```bash
git add frontend/src/api/user.ts frontend/src/types/index.ts frontend/src/views/user/AffiliateView.vue frontend/src/views/user/__tests__/AffiliateView.spec.ts frontend/src/i18n/locales/zh/affiliate.ts frontend/src/i18n/locales/en/affiliate.ts
git commit -m "feat: simplify affiliate rebate experience"
```

### Task 8: Simplify Exclusive User Administration

**Files:**
- Modify: `frontend/src/api/admin/affiliates.ts`
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Modify: `frontend/src/views/admin/__tests__/SettingsView.spec.ts`
- Modify: `frontend/src/i18n/locales/zh/admin/settings.ts`
- Modify: `frontend/src/i18n/locales/en/admin/settings.ts`

- [ ] **Step 1: Write failing settings tests**

Assert obsolete settings are absent and the atomic payload is submitted:

```ts
expect(wrapper.find('[name="affiliate_rebate_freeze_hours"]').exists()).toBe(false)
expect(wrapper.find('[name="affiliate_rebate_duration_days"]').exists()).toBe(false)
expect(wrapper.find('[name="affiliate_rebate_per_invitee_cap"]').exists()).toBe(false)
expect(wrapper.find('[name="affiliate_admin_recharge_enabled"]').exists()).toBe(false)
expect(wrapper.find('[data-testid="affiliate-batch-rate"]').exists()).toBe(false)

expect(updateUserSettings).toHaveBeenCalledWith(userId, {
	aff_code: 'DREAMVIP',
	aff_rebate_rate_percent: 32,
	withdrawal_enabled: true,
})
```

- [ ] **Step 2: Run the settings test and confirm failure**

Run: `cd frontend && pnpm test:run src/views/admin/__tests__/SettingsView.spec.ts`

Expected: FAIL because obsolete controls and batch state remain and withdrawal permission is absent.

- [ ] **Step 3: Implement the exclusive configuration UI**

Update the API type:

```ts
export interface AffiliateAdminEntry {
	user_id: number
	email: string
	username: string
	aff_code: string
	aff_code_custom: boolean
	aff_rebate_rate_percent: number
	withdrawal_enabled: boolean
	aff_count: number
}
```

Keep the affiliate enabled switch, global percentage, and exclusive-user table. Add a labeled toggle for withdrawal permission in the add/edit modal, default it to false, require a finite `0..100` custom rate, remove selection/batch-rate state and obsolete setting controls, and keep reset as the delete-exclusive-configuration action.

- [ ] **Step 4: Run settings tests**

Run: `cd frontend && pnpm test:run src/views/admin/__tests__/SettingsView.spec.ts`

Expected: PASS.

- [ ] **Step 5: Commit admin settings UI**

```bash
git add frontend/src/api/admin/affiliates.ts frontend/src/views/admin/SettingsView.vue frontend/src/views/admin/__tests__/SettingsView.spec.ts frontend/src/i18n/locales/zh/admin/settings.ts frontend/src/i18n/locales/en/admin/settings.ts
git commit -m "feat: manage exclusive affiliate users"
```

### Task 9: Connect Withdrawal Review Navigation

**Files:**
- Modify: `frontend/src/views/admin/affiliates/AdminAffiliateWithdrawalsView.vue`
- Create: `frontend/src/views/admin/affiliates/__tests__/AdminAffiliateWithdrawalsView.spec.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/components/layout/__tests__/AppSidebar.spec.ts`
- Modify: `frontend/src/i18n/locales/zh/admin/affiliate.ts`
- Modify: `frontend/src/i18n/locales/en/admin/affiliate.ts`
- Modify: `frontend/src/i18n/locales/zh/common.ts`
- Modify: `frontend/src/i18n/locales/en/common.ts`

- [ ] **Step 1: Write failing review and navigation tests**

Assert the admin view renders `wechat`/`alipay`, QR preview, amount, status, and paid/reject actions. Assert router and sidebar expose `/admin/affiliates/withdrawals` while the existing red-packet routes/items remain present.

```ts
expect(router.resolve('/admin/affiliates/withdrawals').name).toBe('AdminAffiliateWithdrawals')
expect(wrapper.find('a[href="/admin/affiliates/withdrawals"]').exists()).toBe(true)
expect(wrapper.find('a[href="/admin/red-packets"]').exists()).toBe(true)
```

- [ ] **Step 2: Run review/navigation tests and confirm failure**

Run: `cd frontend && pnpm test:run src/views/admin/affiliates/__tests__/AdminAffiliateWithdrawalsView.spec.ts src/components/layout/__tests__/AppSidebar.spec.ts`

Expected: FAIL because the review page is not routed or linked and has no payment method column.

- [ ] **Step 3: Register and polish the existing review page**

Add this route after the transfer-record route:

```ts
{
	path: '/admin/affiliates/withdrawals',
	name: 'AdminAffiliateWithdrawals',
	component: () => import('@/views/admin/affiliates/AdminAffiliateWithdrawalsView.vue'),
	meta: { requiresAuth: true, requiresAdmin: true,
		titleKey: 'nav.affiliateWithdrawalRecords',
		descriptionKey: 'admin.affiliates.withdrawalsDescription' }
}
```

Add the sidebar child with the existing credit-card icon. Add the payment-method column and channel labels to the existing table/dialog without nesting cards or introducing animation. Preserve all unrelated red-packet changes in both shared files.

- [ ] **Step 4: Run review/navigation tests**

Run: `cd frontend && pnpm test:run src/views/admin/affiliates/__tests__/AdminAffiliateWithdrawalsView.spec.ts src/components/layout/__tests__/AppSidebar.spec.ts`

Expected: PASS.

- [ ] **Step 5: Commit navigation and review UI**

```bash
git add frontend/src/views/admin/affiliates/AdminAffiliateWithdrawalsView.vue frontend/src/views/admin/affiliates/__tests__/AdminAffiliateWithdrawalsView.spec.ts frontend/src/router/index.ts frontend/src/components/layout/AppSidebar.vue frontend/src/components/layout/__tests__/AppSidebar.spec.ts frontend/src/i18n/locales/zh/admin/affiliate.ts frontend/src/i18n/locales/en/admin/affiliate.ts frontend/src/i18n/locales/zh/common.ts frontend/src/i18n/locales/en/common.ts
git commit -m "feat: connect affiliate withdrawal review"
```

### Task 10: Remove Obsolete Settings and Verify End to End

**Files:**
- Modify: `backend/internal/service/setting_getters.go`
- Modify: `backend/internal/service/setting_service.go`
- Modify: `backend/internal/service/setting_update.go`
- Modify: `backend/internal/handler/dto/types.go`
- Modify: `backend/internal/handler/dto/mappers.go`
- Modify: `frontend/src/i18n/locales/zh/index.ts`
- Modify: `frontend/src/i18n/locales/en/index.ts`

- [ ] **Step 1: Remove active legacy-setting reads and writes**

Delete active DTO/getter/update mappings for:

```text
affiliate_rebate_freeze_hours
affiliate_rebate_duration_days
affiliate_rebate_per_invitee_cap
affiliate_admin_recharge_enabled
```

Keep setting key constants and historical migrations only where upstream/database compatibility requires them. Run formatting after deletion:

Run: `cd backend && gofmt -w internal/service/setting_*.go internal/handler/dto/*.go`

Expected: files are formatted with no output.

- [ ] **Step 2: Run all backend verification**

Run: `cd backend && go test -p 1 ./...`

Expected: PASS with no compile failures or test failures.

- [ ] **Step 3: Run all frontend verification**

Run:

```bash
cd frontend
pnpm test:run
pnpm typecheck
pnpm lint:check
pnpm build
```

Expected: every command exits 0.

- [ ] **Step 4: Verify prohibited active concepts are gone**

Run:

```bash
rg -n "AgentAff|AffiliateRebateModeAgent|GetAffiliateAgentUsage|affiliateActionAllowedAmounts|affiliate_rebate_freeze_hours|affiliate_rebate_duration_days|affiliate_rebate_per_invitee_cap|affiliate_admin_recharge_enabled" backend/internal frontend/src
```

Expected: no active business/UI hits; compatibility-only migration/constants hits outside these paths are acceptable.

- [ ] **Step 5: Verify the local Docker application**

Run: `docker compose ps`

Expected: backend, PostgreSQL, and Redis containers are healthy. Open `http://127.0.0.1:3000/affiliate` and `http://127.0.0.1:3000/admin/affiliates/withdrawals`; verify the user page shows one balance and the admin page loads withdrawal records without console errors.

- [ ] **Step 6: Commit final cleanup**

```bash
git add backend/internal/service/setting_getters.go backend/internal/service/setting_service.go backend/internal/service/setting_update.go backend/internal/handler/dto/types.go backend/internal/handler/dto/mappers.go frontend/src/i18n/locales/zh/index.ts frontend/src/i18n/locales/en/index.ts
git commit -m "chore: remove obsolete affiliate settings"
```

- [ ] **Step 7: Review only this feature's diff**

Run: `git log --oneline --decorate origin/main..HEAD` and `git diff origin/main...HEAD -- backend/migrations/193_simplify_affiliate_referrals.sql backend/internal/service/affiliate_service.go backend/internal/repository/affiliate_repo.go backend/internal/handler/admin/affiliate_handler.go frontend/src/views/user/AffiliateView.vue frontend/src/views/admin/SettingsView.vue`

Expected: the affiliate simplification is complete and unrelated worktree edits are neither reverted nor accidentally included in an affiliate-only commit.
