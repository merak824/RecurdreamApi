package admin

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type redPacketApplication interface {
	ListAdmin(ctx context.Context, page, pageSize int) (*service.RedPacketAdminPage, error)
	CreateDraft(ctx context.Context, adminID int64, draft service.RedPacketDraft) (*service.RedPacketActivity, error)
	UpdateDraft(ctx context.Context, activityID int64, draft service.RedPacketDraft) (*service.RedPacketActivity, error)
	Publish(ctx context.Context, activityID, adminID int64) (*service.RedPacketActivity, error)
	Cancel(ctx context.Context, activityID, adminID int64) error
	GetActivity(ctx context.Context, activityID, userID int64) (*service.RedPacketActivityDetail, error)
}

// RedPacketHandler handles administrator red-packet activity operations.
type RedPacketHandler struct {
	service redPacketApplication
}

func NewRedPacketHandler(redPacketService *service.RedPacketService) *RedPacketHandler {
	return &RedPacketHandler{service: redPacketService}
}

func (h *RedPacketHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.ListAdmin(c.Request.Context(), page, pageSize)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *RedPacketHandler) Create(c *gin.Context) {
	subject, ok := adminRedPacketSubject(c)
	if !ok {
		return
	}
	var draft service.RedPacketDraft
	if err := c.ShouldBindJSON(&draft); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	activity, err := h.service.CreateDraft(c.Request.Context(), subject.UserID, draft)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, activity)
}

func (h *RedPacketHandler) Update(c *gin.Context) {
	activityID, ok := adminRedPacketID(c)
	if !ok {
		return
	}
	var draft service.RedPacketDraft
	if err := c.ShouldBindJSON(&draft); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	activity, err := h.service.UpdateDraft(c.Request.Context(), activityID, draft)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, activity)
}

func (h *RedPacketHandler) Publish(c *gin.Context) {
	subject, ok := adminRedPacketSubject(c)
	if !ok {
		return
	}
	activityID, ok := adminRedPacketID(c)
	if !ok {
		return
	}
	activity, err := h.service.Publish(c.Request.Context(), activityID, subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, activity)
}

func (h *RedPacketHandler) Cancel(c *gin.Context) {
	subject, ok := adminRedPacketSubject(c)
	if !ok {
		return
	}
	activityID, ok := adminRedPacketID(c)
	if !ok {
		return
	}
	if err := h.service.Cancel(c.Request.Context(), activityID, subject.UserID); response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func (h *RedPacketHandler) Export(c *gin.Context) {
	subject, ok := adminRedPacketSubject(c)
	if !ok {
		return
	}
	activityID, ok := adminRedPacketID(c)
	if !ok {
		return
	}
	detail, err := h.service.GetActivity(c.Request.Context(), activityID, subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	if detail == nil {
		response.NotFound(c, "Red packet activity not found")
		return
	}

	filename := fmt.Sprintf("red-packet-period-%d.csv", detail.Activity.PeriodNo)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	writer := csv.NewWriter(c.Writer)
	if err := writer.Write([]string{"period_no", "masked_username", "amount_cents", "is_luckiest", "credited_at"}); err != nil {
		return
	}
	for _, winner := range detail.Winners {
		creditedAt := ""
		if winner.CreditedAt != nil {
			creditedAt = winner.CreditedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		if err := writer.Write([]string{
			strconv.FormatInt(detail.Activity.PeriodNo, 10),
			winner.MaskedUsername,
			strconv.FormatInt(winner.AmountCents, 10),
			strconv.FormatBool(winner.IsLuckiest),
			creditedAt,
		}); err != nil {
			return
		}
	}
	writer.Flush()
}

func adminRedPacketSubject(c *gin.Context) (middleware2.AuthSubject, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Admin not found in context")
		return middleware2.AuthSubject{}, false
	}
	return subject, true
}

func adminRedPacketID(c *gin.Context) (int64, bool) {
	activityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityID <= 0 {
		response.BadRequest(c, "Invalid red packet activity ID")
		return 0, false
	}
	return activityID, true
}
