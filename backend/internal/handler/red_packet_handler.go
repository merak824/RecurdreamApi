package handler

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type redPacketApplication interface {
	GetCurrent(ctx context.Context, userID int64) (*service.RedPacketActivity, error)
	GetActivity(ctx context.Context, activityID, userID int64) (*service.RedPacketActivityDetail, error)
	GetEligibility(ctx context.Context, userID int64) (*service.RedPacketEligibility, error)
	ListRecent(ctx context.Context, userID int64, limit int) ([]service.RedPacketActivity, error)
	ListRewards(ctx context.Context, userID int64, limit int) ([]service.RedPacketReward, error)
	Participate(ctx context.Context, activityID, userID int64) (*service.RedPacketParticipationResult, error)
}

// RedPacketHandler handles authenticated user red-packet operations.
type RedPacketHandler struct {
	service redPacketApplication
}

func NewRedPacketHandler(redPacketService *service.RedPacketService) *RedPacketHandler {
	return &RedPacketHandler{service: redPacketService}
}

func (h *RedPacketHandler) Current(c *gin.Context) {
	subject, ok := redPacketSubject(c)
	if !ok {
		return
	}
	activity, err := h.service.GetCurrent(c.Request.Context(), subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, activity)
}

func (h *RedPacketHandler) Eligibility(c *gin.Context) {
	subject, ok := redPacketSubject(c)
	if !ok {
		return
	}
	eligibility, err := h.service.GetEligibility(c.Request.Context(), subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, eligibility)
}

func (h *RedPacketHandler) Recent(c *gin.Context) {
	subject, ok := redPacketSubject(c)
	if !ok {
		return
	}
	limit, ok := parsePositiveQuery(c, "limit", 5)
	if !ok {
		return
	}
	activities, err := h.service.ListRecent(c.Request.Context(), subject.UserID, limit)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, activities)
}

func (h *RedPacketHandler) Rewards(c *gin.Context) {
	subject, ok := redPacketSubject(c)
	if !ok {
		return
	}
	limit, ok := parsePositiveQuery(c, "limit", 50)
	if !ok {
		return
	}
	rewards, err := h.service.ListRewards(c.Request.Context(), subject.UserID, limit)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, rewards)
}

func (h *RedPacketHandler) Detail(c *gin.Context) {
	subject, ok := redPacketSubject(c)
	if !ok {
		return
	}
	activityID, ok := parseRedPacketID(c)
	if !ok {
		return
	}
	detail, err := h.service.GetActivity(c.Request.Context(), activityID, subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, detail)
}

func (h *RedPacketHandler) Participate(c *gin.Context) {
	subject, ok := redPacketSubject(c)
	if !ok {
		return
	}
	activityID, ok := parseRedPacketID(c)
	if !ok {
		return
	}
	result, err := h.service.Participate(c.Request.Context(), activityID, subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func redPacketSubject(c *gin.Context) (middleware2.AuthSubject, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not found in context")
		return middleware2.AuthSubject{}, false
	}
	return subject, true
}

func parseRedPacketID(c *gin.Context) (int64, bool) {
	activityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityID <= 0 {
		response.BadRequest(c, "Invalid red packet activity ID")
		return 0, false
	}
	return activityID, true
}

func parsePositiveQuery(c *gin.Context, name string, defaultValue int) (int, bool) {
	raw := c.Query(name)
	if raw == "" {
		return defaultValue, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		response.BadRequest(c, "Invalid "+name)
		return 0, false
	}
	return value, true
}
