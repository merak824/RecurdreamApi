package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type redPacketApplicationStub struct {
	activityID int64
	userID     int64
	called     bool
}

func (s *redPacketApplicationStub) GetCurrent(context.Context, int64) (*service.RedPacketActivity, error) {
	return nil, nil
}

func (s *redPacketApplicationStub) GetActivity(context.Context, int64, int64) (*service.RedPacketActivityDetail, error) {
	return nil, nil
}

func (s *redPacketApplicationStub) GetEligibility(context.Context, int64) (*service.RedPacketEligibility, error) {
	return nil, nil
}

func (s *redPacketApplicationStub) ListRecent(context.Context, int64, int) ([]service.RedPacketActivity, error) {
	return nil, nil
}

func (s *redPacketApplicationStub) ListRewards(context.Context, int64, int) ([]service.RedPacketReward, error) {
	return nil, nil
}

func (s *redPacketApplicationStub) Participate(_ context.Context, activityID, userID int64) (*service.RedPacketParticipationResult, error) {
	s.called = true
	s.activityID = activityID
	s.userID = userID
	return &service.RedPacketParticipationResult{
		QualificationType: service.RedPacketQualificationRecharge,
	}, nil
}

func TestRedPacketHandlerParticipateRejectsMissingAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &redPacketApplicationStub{}
	h := &RedPacketHandler{service: app}
	router := gin.New()
	router.POST("/red-packets/:id/participate", h.Participate)

	req := httptest.NewRequest(http.MethodPost, "/red-packets/7/participate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.False(t, app.called)
}

func TestRedPacketHandlerParticipateRejectsInvalidActivityID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &redPacketApplicationStub{}
	h := &RedPacketHandler{service: app}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.POST("/red-packets/:id/participate", h.Participate)

	req := httptest.NewRequest(http.MethodPost, "/red-packets/not-a-number/participate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.False(t, app.called)
}

func TestRedPacketHandlerParticipateUsesAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &redPacketApplicationStub{}
	h := &RedPacketHandler{service: app}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.POST("/red-packets/:id/participate", h.Participate)

	req := httptest.NewRequest(http.MethodPost, "/red-packets/7/participate", strings.NewReader(`{"user_id":999}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(7), app.activityID)
	require.Equal(t, int64(42), app.userID)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.Equal(t, float64(0), payload["code"])
}
