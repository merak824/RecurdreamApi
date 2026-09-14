package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AffiliateHandler handles admin affiliate (邀请返利) management:
// listing users with custom settings, updating per-user invite codes
// and exclusive rebate rates, and batch operations.
type AffiliateHandler struct {
	affiliateService *service.AffiliateService
	adminService     service.AdminService
}

// NewAffiliateHandler creates a new admin affiliate handler.
func NewAffiliateHandler(affiliateService *service.AffiliateService, adminService service.AdminService) *AffiliateHandler {
	return &AffiliateHandler{
		affiliateService: affiliateService,
		adminService:     adminService,
	}
}

// ListUsers returns paginated users with custom affiliate settings.
// GET /api/v1/admin/affiliates/users
func (h *AffiliateHandler) ListUsers(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	search := c.Query("search")

	entries, total, err := h.affiliateService.AdminListCustomUsers(c.Request.Context(), service.AffiliateAdminFilter{
		Search:   search,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, entries, total, page, pageSize)
}

// UpdateUserSettings updates a user's affiliate settings.
// PUT /api/v1/admin/affiliates/users/:user_id
type UpdateAffiliateUserRequest struct {
	AffCode              *string  `json:"aff_code"`
	AffRebateRatePercent *float64 `json:"aff_rebate_rate_percent" binding:"required"`
	WithdrawalEnabled    bool     `json:"withdrawal_enabled"`
}

func (h *AffiliateHandler) UpdateUserSettings(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}

	var req UpdateAffiliateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.affiliateService.AdminUpdateExclusiveSettings(c.Request.Context(), userID, service.AffiliateExclusiveSettingsInput{
		AffCode:           req.AffCode,
		RebateRatePercent: req.AffRebateRatePercent,
		WithdrawalEnabled: req.WithdrawalEnabled,
	}); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"user_id": userID})
}

// ClearUserSettings removes ALL of a user's custom affiliate settings — clears
// the exclusive rebate rate AND regenerates the invite code as a new system
// random one. Conceptually this "removes the user from the custom list".
//
// DELETE /api/v1/admin/affiliates/users/:user_id
func (h *AffiliateHandler) ClearUserSettings(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}
	if _, err := h.affiliateService.AdminClearExclusiveSettings(c.Request.Context(), userID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID})
}

// AffiliateUserSummary is the minimal user shape returned by LookupUsers,
// shared with the frontend's add-custom-user picker.
type AffiliateUserSummary struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// LookupUsers searches users by email/username for the "add custom user" modal.
// GET /api/v1/admin/affiliates/users/lookup?q=
func (h *AffiliateHandler) LookupUsers(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		response.Success(c, []AffiliateUserSummary{})
		return
	}
	users, _, err := h.adminService.ListUsers(c.Request.Context(), 1, 20, service.UserListFilters{Search: keyword}, "email", "asc")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result := make([]AffiliateUserSummary, len(users))
	for i, u := range users {
		result[i] = AffiliateUserSummary{ID: u.ID, Email: u.Email, Username: u.Username}
	}
	response.Success(c, result)
}

// GetUserOverview returns one user's affiliate overview.
// GET /api/v1/admin/affiliates/users/:user_id/overview
func (h *AffiliateHandler) GetUserOverview(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}
	overview, err := h.affiliateService.AdminGetUserOverview(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, overview)
}

// ListInviteRecords returns all inviter-invitee relationships.
// GET /api/v1/admin/affiliates/invites
func (h *AffiliateHandler) ListInviteRecords(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := parseAffiliateRecordFilter(c, page, pageSize)
	items, total, err := h.affiliateService.AdminListInviteRecords(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, filter.Page, filter.PageSize)
}

// ListRebateRecords returns all order-level affiliate rebate records.
// GET /api/v1/admin/affiliates/rebates
func (h *AffiliateHandler) ListRebateRecords(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := parseAffiliateRecordFilter(c, page, pageSize)
	items, total, err := h.affiliateService.AdminListRebateRecords(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, filter.Page, filter.PageSize)
}

// ListTransferRecords returns all affiliate quota-to-balance transfer records.
// GET /api/v1/admin/affiliates/transfers
func (h *AffiliateHandler) ListTransferRecords(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := parseAffiliateRecordFilter(c, page, pageSize)
	items, total, err := h.affiliateService.AdminListTransferRecords(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, filter.Page, filter.PageSize)
}

// ListWithdrawalRecords returns affiliate withdrawal requests and records.
// GET /api/v1/admin/affiliates/withdrawals
func (h *AffiliateHandler) ListWithdrawalRecords(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := parseAffiliateRecordFilter(c, page, pageSize)
	items, total, err := h.affiliateService.AdminListWithdrawalRecords(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, filter.Page, filter.PageSize)
}

type AffiliateWithdrawalPaidRequest struct {
	PaymentProofData string `json:"payment_proof_data"`
	AdminNote        string `json:"admin_note"`
}

// MarkWithdrawalPaid marks a pending affiliate withdrawal as paid.
// POST /api/v1/admin/affiliates/withdrawals/:id/paid
func (h *AffiliateHandler) MarkWithdrawalPaid(c *gin.Context) {
	withdrawalID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || withdrawalID <= 0 {
		response.BadRequest(c, "Invalid withdrawal id")
		return
	}
	var req AffiliateWithdrawalPaidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	item, err := h.affiliateService.AdminMarkWithdrawalPaid(c.Request.Context(), withdrawalID, service.AffiliateWithdrawalAdminActionInput{
		AdminID:          subject.UserID,
		PaymentProofData: req.PaymentProofData,
		AdminNote:        req.AdminNote,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

type AffiliateWithdrawalRejectRequest struct {
	RejectReason string `json:"reject_reason"`
	AdminNote    string `json:"admin_note"`
}

// RejectWithdrawal rejects a pending affiliate withdrawal and returns its quota.
// POST /api/v1/admin/affiliates/withdrawals/:id/reject
func (h *AffiliateHandler) RejectWithdrawal(c *gin.Context) {
	withdrawalID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || withdrawalID <= 0 {
		response.BadRequest(c, "Invalid withdrawal id")
		return
	}
	var req AffiliateWithdrawalRejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	item, err := h.affiliateService.AdminRejectWithdrawal(c.Request.Context(), withdrawalID, service.AffiliateWithdrawalAdminActionInput{
		AdminID:      subject.UserID,
		RejectReason: req.RejectReason,
		AdminNote:    req.AdminNote,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func parseAffiliateRecordFilter(c *gin.Context, page, pageSize int) service.AffiliateRecordFilter {
	filter := service.AffiliateRecordFilter{
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
		SortBy:   c.Query("sort_by"),
		SortDesc: c.Query("sort_order") != "asc",
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	userTZ := c.Query("timezone")
	if t := parseAffiliateRecordStartTime(c.Query("start_at"), userTZ); t != nil {
		filter.StartAt = t
	}
	if t := parseAffiliateRecordEndTime(c.Query("end_at"), userTZ); t != nil {
		filter.EndAt = t
	}
	return filter
}

func parseAffiliateRecordStartTime(raw string, userTZ string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed
	}
	if parsed, err := timezone.ParseInUserLocation("2006-01-02", raw, userTZ); err == nil {
		return &parsed
	}
	return nil
}

func parseAffiliateRecordEndTime(raw string, userTZ string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed
	}
	if parsed, err := timezone.ParseInUserLocation("2006-01-02", raw, userTZ); err == nil {
		end := parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
		return &end
	}
	return nil
}
