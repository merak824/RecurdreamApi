package service

import (
	"context"
	"encoding/base64"
	"errors"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	ErrAffiliateProfileNotFound       = infraerrors.NotFound("AFFILIATE_PROFILE_NOT_FOUND", "affiliate profile not found")
	ErrAffiliateCodeInvalid           = infraerrors.BadRequest("AFFILIATE_CODE_INVALID", "invalid affiliate code")
	ErrAffiliateCodeTaken             = infraerrors.Conflict("AFFILIATE_CODE_TAKEN", "affiliate code already in use")
	ErrAffiliateAlreadyBound          = infraerrors.Conflict("AFFILIATE_ALREADY_BOUND", "affiliate inviter already bound")
	ErrAffiliateQuotaEmpty            = infraerrors.BadRequest("AFFILIATE_QUOTA_EMPTY", "no affiliate quota available to transfer")
	ErrAffiliateTransferAmount        = infraerrors.BadRequest("AFFILIATE_TRANSFER_AMOUNT_INVALID", "invalid transfer amount")
	ErrAffiliateQuotaInsufficient     = infraerrors.BadRequest("AFFILIATE_QUOTA_INSUFFICIENT", "transfer amount exceeds available affiliate quota")
	ErrAffiliateWithdrawalForbidden   = infraerrors.Forbidden("AFFILIATE_WITHDRAWAL_FORBIDDEN", "affiliate withdrawal is not enabled for this user")
	ErrAffiliateWithdrawAmount        = infraerrors.BadRequest("AFFILIATE_WITHDRAW_AMOUNT_INVALID", "invalid withdrawal amount")
	ErrAffiliateWithdrawQuota         = infraerrors.BadRequest("AFFILIATE_WITHDRAW_QUOTA_INSUFFICIENT", "withdrawal amount exceeds available affiliate rebate")
	ErrAffiliatePaymentMethod         = infraerrors.BadRequest("AFFILIATE_PAYMENT_METHOD_INVALID", "payment method must be wechat or alipay")
	ErrAffiliateExclusiveRateRequired = infraerrors.BadRequest("AFFILIATE_EXCLUSIVE_RATE_REQUIRED", "exclusive rebate rate is required")
	ErrAffiliateWithdrawMissing       = infraerrors.NotFound("AFFILIATE_WITHDRAWAL_NOT_FOUND", "affiliate withdrawal not found")
	ErrAffiliateWithdrawStatus        = infraerrors.Conflict("AFFILIATE_WITHDRAWAL_STATUS_INVALID", "affiliate withdrawal status invalid")
	ErrAffiliateImageInvalid          = infraerrors.BadRequest("AFFILIATE_IMAGE_INVALID", "image must be a valid png, jpg or webp data URL")
	ErrAffiliateImageTooLarge         = infraerrors.BadRequest("AFFILIATE_IMAGE_TOO_LARGE", "image must be 2MB or smaller")
)

const (
	affiliateInviteesLimit     = 100
	affiliateImageMaxBytes     = 2 * 1024 * 1024
	affiliateMinimumWithdrawal = 1.0
	// AffiliateCodeMinLength / AffiliateCodeMaxLength bound both system-generated
	// 12-char codes and admin-customized codes (e.g. "VIP2026").
	AffiliateCodeMinLength = 4
	AffiliateCodeMaxLength = 32

	AffiliateWithdrawalStatusPending   = "pending"
	AffiliateWithdrawalStatusPaid      = "paid"
	AffiliateWithdrawalStatusRejected  = "rejected"
	AffiliateWithdrawalStatusCompleted = "completed"

	AffiliateWithdrawalRecordTypeTransfer   = "transfer"
	AffiliateWithdrawalRecordTypeWithdrawal = "withdrawal"

	AffiliateWithdrawalDestinationBalance      = "balance"
	AffiliateWithdrawalDestinationAlipayWechat = "alipay_wechat"
)

// affiliateCodeValidChar accepts uppercase letters, digits, underscore and dash.
// All input passes through strings.ToUpper before validation, so lowercase from
// users is normalized — admins may supply mixed case in their UI.
var affiliateCodeValidChar = func() [256]bool {
	var tbl [256]bool
	for c := byte('A'); c <= 'Z'; c++ {
		tbl[c] = true
	}
	for c := byte('0'); c <= '9'; c++ {
		tbl[c] = true
	}
	tbl['_'] = true
	tbl['-'] = true
	return tbl
}()

// isValidAffiliateCodeFormat validates code format for both binding (user input)
// and admin updates. Caller is expected to upper-case the input first.
func isValidAffiliateCodeFormat(code string) bool {
	if len(code) < AffiliateCodeMinLength || len(code) > AffiliateCodeMaxLength {
		return false
	}
	for i := 0; i < len(code); i++ {
		if !affiliateCodeValidChar[code[i]] {
			return false
		}
	}
	return true
}

type AffiliateSummary struct {
	UserID               int64     `json:"user_id"`
	AffCode              string    `json:"aff_code"`
	AffCodeCustom        bool      `json:"aff_code_custom"`
	AffRebateRatePercent *float64  `json:"aff_rebate_rate_percent,omitempty"`
	WithdrawalEnabled    bool      `json:"withdrawal_enabled"`
	InviterID            *int64    `json:"inviter_id,omitempty"`
	AffCount             int       `json:"aff_count"`
	AffQuota             float64   `json:"aff_quota"`
	AffHistoryQuota      float64   `json:"aff_history_quota"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type AffiliateInvitee struct {
	UserID      int64      `json:"user_id"`
	Email       string     `json:"email"`
	Username    string     `json:"username"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	TotalRebate float64    `json:"total_rebate"`
}

type AffiliateDetail struct {
	UserID            int64   `json:"user_id"`
	AffCode           string  `json:"aff_code"`
	InviterID         *int64  `json:"inviter_id,omitempty"`
	AffCount          int     `json:"aff_count"`
	AffQuota          float64 `json:"aff_quota"`
	AffHistoryQuota   float64 `json:"aff_history_quota"`
	WithdrawalEnabled bool    `json:"withdrawal_enabled"`
	// EffectiveRebateRatePercent 是当前用户作为邀请人时实际生效的返利比例：
	// 优先用户自己的专属比例（aff_rebate_rate_percent），否则回退到全局比例。
	// 用于在用户的 /affiliate 页面直观展示「分享后能拿到多少」。
	EffectiveRebateRatePercent float64               `json:"effective_rebate_rate_percent"`
	Invitees                   []AffiliateInvitee    `json:"invitees"`
	Withdrawals                []AffiliateWithdrawal `json:"withdrawals"`
}

type AffiliateWithdrawal struct {
	ID               int64      `json:"id"`
	UserID           int64      `json:"user_id"`
	UserEmail        string     `json:"user_email,omitempty"`
	Username         string     `json:"username,omitempty"`
	Amount           float64    `json:"amount"`
	Status           string     `json:"status"`
	PaymentMethod    string     `json:"payment_method"`
	CollectionQRData string     `json:"collection_qr_data,omitempty"`
	CollectionQRMIME string     `json:"collection_qr_mime,omitempty"`
	CollectionQRSize int        `json:"collection_qr_size,omitempty"`
	PaymentProofData string     `json:"payment_proof_data,omitempty"`
	PaymentProofMIME string     `json:"payment_proof_mime,omitempty"`
	PaymentProofSize int        `json:"payment_proof_size,omitempty"`
	RejectReason     string     `json:"reject_reason,omitempty"`
	AdminNote        string     `json:"admin_note,omitempty"`
	ProcessedBy      *int64     `json:"processed_by,omitempty"`
	ProcessedAt      *time.Time `json:"processed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type AffiliateWithdrawalRecord struct {
	ID                  int64      `json:"id"`
	RecordType          string     `json:"record_type"`
	Destination         string     `json:"destination"`
	UserID              int64      `json:"user_id"`
	UserEmail           string     `json:"user_email,omitempty"`
	Username            string     `json:"username,omitempty"`
	Amount              float64    `json:"amount"`
	Status              string     `json:"status"`
	PaymentMethod       string     `json:"payment_method"`
	CollectionQRData    string     `json:"collection_qr_data,omitempty"`
	CollectionQRMIME    string     `json:"collection_qr_mime,omitempty"`
	CollectionQRSize    int        `json:"collection_qr_size,omitempty"`
	PaymentProofData    string     `json:"payment_proof_data,omitempty"`
	PaymentProofMIME    string     `json:"payment_proof_mime,omitempty"`
	PaymentProofSize    int        `json:"payment_proof_size,omitempty"`
	RejectReason        string     `json:"reject_reason,omitempty"`
	AdminNote           string     `json:"admin_note,omitempty"`
	ProcessedBy         *int64     `json:"processed_by,omitempty"`
	ProcessedAt         *time.Time `json:"processed_at,omitempty"`
	BalanceAfter        *float64   `json:"balance_after,omitempty"`
	AvailableQuotaAfter *float64   `json:"available_quota_after,omitempty"`
	FrozenQuotaAfter    *float64   `json:"frozen_quota_after,omitempty"`
	HistoryQuotaAfter   *float64   `json:"history_quota_after,omitempty"`
	SnapshotAvailable   bool       `json:"snapshot_available"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type AffiliateWithdrawalInput struct {
	Amount           float64
	PaymentMethod    string
	CollectionQRData string
}

type AffiliateTransferInput struct {
	Amount float64
}

type AffiliateExclusiveSettingsInput struct {
	AffCode           *string
	RebateRatePercent *float64
	WithdrawalEnabled bool
}

type AffiliateWithdrawalAdminActionInput struct {
	AdminID          int64
	PaymentProofData string
	RejectReason     string
	AdminNote        string
}

type AffiliateImageData struct {
	DataURL string
	MIME    string
	Size    int
}

type AffiliateRepository interface {
	EnsureUserAffiliate(ctx context.Context, userID int64) (*AffiliateSummary, error)
	GetAffiliateByCode(ctx context.Context, code string) (*AffiliateSummary, error)
	BindInviter(ctx context.Context, userID, inviterID int64) (bool, error)
	AccrueQuota(ctx context.Context, inviterID, inviteeUserID int64, amount float64, sourceOrderID *int64) (bool, error)
	TransferQuotaToBalance(ctx context.Context, userID int64, amount float64) (float64, float64, error)
	ListInvitees(ctx context.Context, inviterID int64, limit int) ([]AffiliateInvitee, error)
	CreateWithdrawal(ctx context.Context, userID int64, amount float64, paymentMethod string, collectionQR AffiliateImageData) (*AffiliateWithdrawal, error)
	ListWithdrawalsByUser(ctx context.Context, userID int64, limit int) ([]AffiliateWithdrawal, error)
	GetWithdrawalStats(ctx context.Context, userID int64) (pending float64, paid float64, err error)

	UpdateExclusiveSettings(ctx context.Context, userID int64, input AffiliateExclusiveSettingsInput) error
	ClearExclusiveSettings(ctx context.Context, userID int64) (string, error)
	ListUsersWithCustomSettings(ctx context.Context, filter AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error)
	ListAffiliateInviteRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateInviteRecord, int64, error)
	ListAffiliateRebateRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateRebateRecord, int64, error)
	ListAffiliateTransferRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateTransferRecord, int64, error)
	ListAffiliateWithdrawalRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateWithdrawalRecord, int64, error)
	MarkWithdrawalPaid(ctx context.Context, withdrawalID int64, input AffiliateWithdrawalAdminActionInput, proof AffiliateImageData) (*AffiliateWithdrawal, error)
	RejectWithdrawal(ctx context.Context, withdrawalID int64, input AffiliateWithdrawalAdminActionInput) (*AffiliateWithdrawal, error)
	GetAffiliateUserOverview(ctx context.Context, userID int64) (*AffiliateUserOverview, error)
}

// AffiliateAdminFilter 列表筛选条件
type AffiliateAdminFilter struct {
	Search   string
	Page     int
	PageSize int
}

// AffiliateAdminEntry 专属用户列表条目
type AffiliateAdminEntry struct {
	UserID               int64    `json:"user_id"`
	Email                string   `json:"email"`
	Username             string   `json:"username"`
	Role                 string   `json:"role"`
	AffCode              string   `json:"aff_code"`
	AffCodeCustom        bool     `json:"aff_code_custom"`
	AffRebateRatePercent *float64 `json:"aff_rebate_rate_percent,omitempty"`
	WithdrawalEnabled    bool     `json:"withdrawal_enabled"`
	AffCount             int      `json:"aff_count"`
}

type AffiliateRecordFilter struct {
	Search   string
	Page     int
	PageSize int
	StartAt  *time.Time
	EndAt    *time.Time
	SortBy   string
	SortDesc bool
}

type AffiliateInviteRecord struct {
	InviterID       int64     `json:"inviter_id"`
	InviterEmail    string    `json:"inviter_email"`
	InviterUsername string    `json:"inviter_username"`
	InviteeID       int64     `json:"invitee_id"`
	InviteeEmail    string    `json:"invitee_email"`
	InviteeUsername string    `json:"invitee_username"`
	AffCode         string    `json:"aff_code"`
	TotalRebate     float64   `json:"total_rebate"`
	CreatedAt       time.Time `json:"created_at"`
}

type AffiliateRebateRecord struct {
	OrderID         int64     `json:"order_id"`
	OutTradeNo      string    `json:"out_trade_no"`
	InviterID       int64     `json:"inviter_id"`
	InviterEmail    string    `json:"inviter_email"`
	InviterUsername string    `json:"inviter_username"`
	InviteeID       int64     `json:"invitee_id"`
	InviteeEmail    string    `json:"invitee_email"`
	InviteeUsername string    `json:"invitee_username"`
	OrderAmount     float64   `json:"order_amount"`
	PayAmount       float64   `json:"pay_amount"`
	RebateAmount    float64   `json:"rebate_amount"`
	PaymentType     string    `json:"payment_type"`
	OrderStatus     string    `json:"order_status"`
	CreatedAt       time.Time `json:"created_at"`
}

type AffiliateTransferRecord struct {
	LedgerID            int64     `json:"ledger_id"`
	UserID              int64     `json:"user_id"`
	UserEmail           string    `json:"user_email"`
	Username            string    `json:"username"`
	Amount              float64   `json:"amount"`
	BalanceAfter        *float64  `json:"balance_after,omitempty"`
	AvailableQuotaAfter *float64  `json:"available_quota_after,omitempty"`
	FrozenQuotaAfter    *float64  `json:"frozen_quota_after,omitempty"`
	HistoryQuotaAfter   *float64  `json:"history_quota_after,omitempty"`
	SnapshotAvailable   bool      `json:"snapshot_available"`
	CurrentBalance      float64   `json:"-"`
	RemainingQuota      float64   `json:"-"`
	FrozenQuota         float64   `json:"-"`
	HistoryQuota        float64   `json:"-"`
	CreatedAt           time.Time `json:"created_at"`
}

type AffiliateUserOverview struct {
	UserID              int64   `json:"user_id"`
	Email               string  `json:"email"`
	Username            string  `json:"username"`
	AffCode             string  `json:"aff_code"`
	RebateRatePercent   float64 `json:"rebate_rate_percent"`
	RebateRateCustom    bool    `json:"-"`
	InvitedCount        int     `json:"invited_count"`
	RebatedInviteeCount int     `json:"rebated_invitee_count"`
	AvailableQuota      float64 `json:"available_quota"`
	HistoryQuota        float64 `json:"history_quota"`
}

type AffiliateService struct {
	repo                 AffiliateRepository
	settingService       *SettingService
	authCacheInvalidator APIKeyAuthCacheInvalidator
	billingCacheService  *BillingCacheService
}

func NewAffiliateService(repo AffiliateRepository, settingService *SettingService, authCacheInvalidator APIKeyAuthCacheInvalidator, billingCacheService *BillingCacheService) *AffiliateService {
	return &AffiliateService{
		repo:                 repo,
		settingService:       settingService,
		authCacheInvalidator: authCacheInvalidator,
		billingCacheService:  billingCacheService,
	}
}

// IsEnabled reports whether the affiliate (邀请返利) feature is turned on.
func (s *AffiliateService) IsEnabled(ctx context.Context) bool {
	if s == nil || s.settingService == nil {
		return AffiliateEnabledDefault
	}
	return s.settingService.IsAffiliateEnabled(ctx)
}

func (s *AffiliateService) EnsureUserAffiliate(ctx context.Context, userID int64) (*AffiliateSummary, error) {
	if userID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	return s.repo.EnsureUserAffiliate(ctx, userID)
}

func (s *AffiliateService) GetAffiliateDetail(ctx context.Context, userID int64) (*AffiliateDetail, error) {
	summary, err := s.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return nil, err
	}
	invitees, err := s.listInvitees(ctx, userID)
	if err != nil {
		return nil, err
	}
	withdrawals := []AffiliateWithdrawal{}
	if summary.WithdrawalEnabled {
		withdrawals, err = s.repo.ListWithdrawalsByUser(ctx, userID, 50)
		if err != nil {
			return nil, err
		}
	}
	return &AffiliateDetail{
		UserID:                     summary.UserID,
		AffCode:                    summary.AffCode,
		InviterID:                  summary.InviterID,
		AffCount:                   summary.AffCount,
		AffQuota:                   summary.AffQuota,
		AffHistoryQuota:            summary.AffHistoryQuota,
		WithdrawalEnabled:          summary.WithdrawalEnabled,
		EffectiveRebateRatePercent: s.resolveRebateRatePercent(ctx, summary),
		Invitees:                   invitees,
		Withdrawals:                withdrawals,
	}, nil
}

func (s *AffiliateService) BindInviterByCode(ctx context.Context, userID int64, rawCode string) error {
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		return nil
	}
	if s == nil || s.repo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	// 总开关关闭时，注册阶段静默忽略 aff 参数（不报错，避免阻断注册流程）
	if !s.IsEnabled(ctx) {
		return nil
	}
	if !isValidAffiliateCodeFormat(code) {
		return ErrAffiliateCodeInvalid
	}

	selfSummary, err := s.repo.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return err
	}
	if selfSummary.InviterID != nil {
		return nil
	}

	inviterSummary, err := s.repo.GetAffiliateByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrAffiliateProfileNotFound) {
			return ErrAffiliateCodeInvalid
		}
		return err
	}
	if inviterSummary == nil || inviterSummary.UserID <= 0 || inviterSummary.UserID == userID {
		return ErrAffiliateCodeInvalid
	}

	bound, err := s.repo.BindInviter(ctx, userID, inviterSummary.UserID)
	if err != nil {
		return err
	}
	if !bound {
		return ErrAffiliateAlreadyBound
	}
	return nil
}

func (s *AffiliateService) AccrueInviteRebate(ctx context.Context, inviteeUserID int64, baseRechargeAmount float64) (float64, error) {
	return s.AccrueInviteRebateForOrder(ctx, inviteeUserID, baseRechargeAmount, nil)
}

func (s *AffiliateService) AccrueInviteRebateForOrder(ctx context.Context, inviteeUserID int64, baseRechargeAmount float64, sourceOrderID *int64) (float64, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	if inviteeUserID <= 0 || baseRechargeAmount <= 0 || math.IsNaN(baseRechargeAmount) || math.IsInf(baseRechargeAmount, 0) {
		return 0, nil
	}
	// 总开关关闭时，新充值不再产生返利
	if !s.IsEnabled(ctx) {
		return 0, nil
	}

	inviteeSummary, err := s.repo.EnsureUserAffiliate(ctx, inviteeUserID)
	if err != nil {
		return 0, err
	}
	if inviteeSummary.InviterID == nil || *inviteeSummary.InviterID <= 0 {
		return 0, nil
	}

	// 加载邀请人 profile，优先使用专属比例（覆盖全局）
	inviterSummary, err := s.repo.EnsureUserAffiliate(ctx, *inviteeSummary.InviterID)
	if err != nil {
		return 0, err
	}
	// 有效期检查：超过返利有效期后不再产生返利

	rebateRatePercent := s.resolveRebateRatePercent(ctx, inviterSummary)
	rebate := roundTo(baseRechargeAmount*(rebateRatePercent/100), 8)
	if rebate <= 0 {
		return 0, nil
	}

	// 单人上限检查：精确截断到剩余额度
	applied, err := s.repo.AccrueQuota(ctx, *inviteeSummary.InviterID, inviteeUserID, rebate, sourceOrderID)
	if err != nil {
		return 0, err
	}
	if !applied {
		return 0, nil
	}
	return rebate, nil
}

// resolveRebateRatePercent returns the inviter's exclusive rate when set,
// otherwise the global setting value (clamped to [Min, Max]).
func (s *AffiliateService) resolveRebateRatePercent(ctx context.Context, inviter *AffiliateSummary) float64 {
	if inviter != nil && inviter.AffRebateRatePercent != nil {
		v := *inviter.AffRebateRatePercent
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return s.globalRebateRatePercent(ctx)
		}
		return clampAffiliateRebateRate(v)
	}
	return s.globalRebateRatePercent(ctx)
}

// globalRebateRatePercent reads the system-wide rebate rate via SettingService,
// returning the documented default when SettingService is unavailable.
func (s *AffiliateService) globalRebateRatePercent(ctx context.Context) float64 {
	if s == nil || s.settingService == nil {
		return AffiliateRebateRateDefault
	}
	return s.settingService.GetAffiliateRebateRatePercent(ctx)
}

func (s *AffiliateService) TransferAffiliateQuota(ctx context.Context, userID int64, input AffiliateTransferInput) (float64, float64, error) {
	if s == nil || s.repo == nil {
		return 0, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	amount := roundTo(input.Amount, 8)
	if !isAllowedAffiliateTransferAmount(amount) {
		return 0, 0, ErrAffiliateTransferAmount
	}

	transferred, balance, err := s.repo.TransferQuotaToBalance(ctx, userID, amount)
	if err != nil {
		return 0, 0, err
	}
	if transferred > 0 {
		s.invalidateAffiliateCaches(ctx, userID)
	}
	return transferred, balance, nil
}

func (s *AffiliateService) CreateWithdrawal(ctx context.Context, userID int64, input AffiliateWithdrawalInput) (*AffiliateWithdrawal, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	summary, err := s.repo.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return nil, err
	}
	if summary.AffRebateRatePercent == nil || !summary.WithdrawalEnabled {
		return nil, ErrAffiliateWithdrawalForbidden
	}
	amount := roundTo(input.Amount, 8)
	if !isAllowedAffiliateWithdrawalAmount(amount) {
		return nil, ErrAffiliateWithdrawAmount
	}
	paymentMethod := strings.ToLower(strings.TrimSpace(input.PaymentMethod))
	if !isAllowedAffiliatePaymentMethod(paymentMethod) {
		return nil, ErrAffiliatePaymentMethod
	}
	qr, err := parseAffiliateImageDataURL(input.CollectionQRData, true)
	if err != nil {
		return nil, err
	}
	withdrawal, err := s.repo.CreateWithdrawal(ctx, userID, amount, paymentMethod, qr)
	if err != nil {
		return nil, err
	}
	s.invalidateAffiliateCaches(ctx, userID)
	return withdrawal, nil
}

func (s *AffiliateService) AdminMarkWithdrawalPaid(ctx context.Context, withdrawalID int64, input AffiliateWithdrawalAdminActionInput) (*AffiliateWithdrawal, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	proof, err := parseAffiliateImageDataURL(input.PaymentProofData, false)
	if err != nil {
		return nil, err
	}
	withdrawal, err := s.repo.MarkWithdrawalPaid(ctx, withdrawalID, input, proof)
	if err != nil {
		return nil, err
	}
	if withdrawal != nil {
		s.invalidateAffiliateCaches(ctx, withdrawal.UserID)
	}
	return withdrawal, nil
}

func (s *AffiliateService) AdminRejectWithdrawal(ctx context.Context, withdrawalID int64, input AffiliateWithdrawalAdminActionInput) (*AffiliateWithdrawal, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	withdrawal, err := s.repo.RejectWithdrawal(ctx, withdrawalID, input)
	if err != nil {
		return nil, err
	}
	if withdrawal != nil {
		s.invalidateAffiliateCaches(ctx, withdrawal.UserID)
	}
	return withdrawal, nil
}

func (s *AffiliateService) listInvitees(ctx context.Context, inviterID int64) ([]AffiliateInvitee, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	invitees, err := s.repo.ListInvitees(ctx, inviterID, affiliateInviteesLimit)
	if err != nil {
		return nil, err
	}
	for i := range invitees {
		invitees[i].Email = maskEmail(invitees[i].Email)
	}
	return invitees, nil
}

func roundTo(v float64, scale int) float64 {
	factor := math.Pow10(scale)
	return math.Round(v*factor) / factor
}

func isAllowedAffiliateTransferAmount(amount float64) bool {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return false
	}
	return amount > 0
}

func isAllowedAffiliateWithdrawalAmount(amount float64) bool {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return false
	}
	return amount >= affiliateMinimumWithdrawal
}

func isAllowedAffiliatePaymentMethod(paymentMethod string) bool {
	switch strings.ToLower(strings.TrimSpace(paymentMethod)) {
	case "wechat", "alipay":
		return true
	default:
		return false
	}
}

func parseAffiliateImageDataURL(raw string, required bool) (AffiliateImageData, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if required {
			return AffiliateImageData{}, ErrAffiliateImageInvalid
		}
		return AffiliateImageData{}, nil
	}
	const prefix = "data:"
	if !strings.HasPrefix(raw, prefix) {
		return AffiliateImageData{}, ErrAffiliateImageInvalid
	}
	comma := strings.Index(raw, ",")
	if comma <= len(prefix) {
		return AffiliateImageData{}, ErrAffiliateImageInvalid
	}
	meta := raw[len(prefix):comma]
	payload := raw[comma+1:]
	parts := strings.Split(meta, ";")
	if len(parts) < 2 || !strings.EqualFold(parts[len(parts)-1], "base64") {
		return AffiliateImageData{}, ErrAffiliateImageInvalid
	}
	mime := strings.ToLower(strings.TrimSpace(parts[0]))
	switch mime {
	case "image/png", "image/jpeg", "image/jpg", "image/webp":
	default:
		return AffiliateImageData{}, ErrAffiliateImageInvalid
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return AffiliateImageData{}, ErrAffiliateImageInvalid
	}
	if len(decoded) == 0 {
		return AffiliateImageData{}, ErrAffiliateImageInvalid
	}
	if len(decoded) > affiliateImageMaxBytes {
		return AffiliateImageData{}, ErrAffiliateImageTooLarge
	}
	return AffiliateImageData{
		DataURL: raw,
		MIME:    mime,
		Size:    len(decoded),
	}, nil
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	at := strings.Index(email, "@")
	if at <= 0 || at >= len(email)-1 {
		return "***"
	}

	local := email[:at]
	domain := email[at+1:]
	dot := strings.LastIndex(domain, ".")

	maskedLocal := maskSegment(local)
	if dot <= 0 || dot >= len(domain)-1 {
		return maskedLocal + "@" + maskSegment(domain)
	}

	domainName := domain[:dot]
	tld := domain[dot:]
	return maskedLocal + "@" + maskSegment(domainName) + tld
}

func maskSegment(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return "***"
	}
	if len(r) == 1 {
		return string(r[0]) + "***"
	}
	return string(r[0]) + "***"
}

func (s *AffiliateService) invalidateAffiliateCaches(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService != nil {
		if err := s.billingCacheService.InvalidateUserBalance(ctx, userID); err != nil {
			logger.LegacyPrintf("service.affiliate", "[Affiliate] Failed to invalidate billing cache for user %d: %v", userID, err)
		}
	}
}

// =========================
// Admin: 专属配置管理
// =========================

// validateExclusiveRate ensures a per-user override is finite and within
// [Min, Max]. nil is always valid (means "clear / fall back to global").
func validateExclusiveRate(ratePercent *float64) error {
	if ratePercent == nil {
		return nil
	}
	v := *ratePercent
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return infraerrors.BadRequest("INVALID_RATE", "invalid rebate rate")
	}
	if v < AffiliateRebateRateMin || v > AffiliateRebateRateMax {
		return infraerrors.BadRequest("INVALID_RATE", "rebate rate out of range")
	}
	return nil
}

func (s *AffiliateService) AdminUpdateExclusiveSettings(ctx context.Context, userID int64, input AffiliateExclusiveSettingsInput) error {
	if s == nil || s.repo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	if userID <= 0 {
		return infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	if input.RebateRatePercent == nil {
		return ErrAffiliateExclusiveRateRequired
	}
	if err := validateExclusiveRate(input.RebateRatePercent); err != nil {
		return err
	}
	if input.AffCode != nil {
		code := strings.ToUpper(strings.TrimSpace(*input.AffCode))
		if code != "" && !isValidAffiliateCodeFormat(code) {
			return ErrAffiliateCodeInvalid
		}
		input.AffCode = &code
	}
	return s.repo.UpdateExclusiveSettings(ctx, userID, input)
}

func (s *AffiliateService) AdminClearExclusiveSettings(ctx context.Context, userID int64) (string, error) {
	if s == nil || s.repo == nil {
		return "", infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	if userID <= 0 {
		return "", infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	return s.repo.ClearExclusiveSettings(ctx, userID)
}

// AdminListCustomUsers 列出有专属配置的用户。
func (s *AffiliateService) AdminListCustomUsers(ctx context.Context, filter AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	return s.repo.ListUsersWithCustomSettings(ctx, filter)
}

func (s *AffiliateService) AdminListInviteRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateInviteRecord, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	return s.repo.ListAffiliateInviteRecords(ctx, normalizeAffiliateRecordFilter(filter))
}

func (s *AffiliateService) AdminListRebateRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateRebateRecord, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	return s.repo.ListAffiliateRebateRecords(ctx, normalizeAffiliateRecordFilter(filter))
}

func (s *AffiliateService) AdminListTransferRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateTransferRecord, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	return s.repo.ListAffiliateTransferRecords(ctx, normalizeAffiliateRecordFilter(filter))
}

func (s *AffiliateService) AdminListWithdrawalRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateWithdrawalRecord, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	return s.repo.ListAffiliateWithdrawalRecords(ctx, normalizeAffiliateRecordFilter(filter))
}

func (s *AffiliateService) AdminGetUserOverview(ctx context.Context, userID int64) (*AffiliateUserOverview, error) {
	if userID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	overview, err := s.repo.GetAffiliateUserOverview(ctx, userID)
	if err != nil {
		return nil, err
	}
	if overview != nil {
		if !overview.RebateRateCustom {
			overview.RebateRatePercent = s.globalRebateRatePercent(ctx)
		}
		overview.RebateRatePercent = clampAffiliateRebateRate(overview.RebateRatePercent)
	}
	return overview, nil
}

func normalizeAffiliateRecordFilter(filter AffiliateRecordFilter) AffiliateRecordFilter {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.SortBy = strings.TrimSpace(filter.SortBy)
	return filter
}
