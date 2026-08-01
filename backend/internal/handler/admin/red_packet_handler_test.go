package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminRedPacketApplicationStub struct {
	adminID int64
	draft   service.RedPacketDraft
	err     error
}

func (s *adminRedPacketApplicationStub) ListAdmin(context.Context, int, int) (*service.RedPacketAdminPage, error) {
	return &service.RedPacketAdminPage{}, nil
}

func (s *adminRedPacketApplicationStub) CreateDraft(_ context.Context, adminID int64, draft service.RedPacketDraft) (*service.RedPacketActivity, error) {
	s.adminID = adminID
	s.draft = draft
	if s.err != nil {
		return nil, s.err
	}
	return &service.RedPacketActivity{ID: 11, Name: draft.Name}, nil
}

func (s *adminRedPacketApplicationStub) UpdateDraft(context.Context, int64, service.RedPacketDraft) (*service.RedPacketActivity, error) {
	return nil, nil
}

func (s *adminRedPacketApplicationStub) Publish(context.Context, int64, int64) (*service.RedPacketActivity, error) {
	return nil, nil
}

func (s *adminRedPacketApplicationStub) Cancel(context.Context, int64, int64) error { return nil }

func (s *adminRedPacketApplicationStub) GetActivity(context.Context, int64, int64) (*service.RedPacketActivityDetail, error) {
	return nil, nil
}

func TestAdminRedPacketHandlerCreateUsesAuthenticatedAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &adminRedPacketApplicationStub{}
	h := &RedPacketHandler{service: app}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 9})
		c.Next()
	})
	router.POST("/admin/red-packets", h.Create)

	body := `{"name":"第12期红包","message":"好运","packet_type":"lucky","total_amount_cents":500,"target_participants":10,"winner_count":3}`
	req := httptest.NewRequest(http.MethodPost, "/admin/red-packets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, int64(9), app.adminID)
	require.Equal(t, "第12期红包", app.draft.Name)
	require.Equal(t, int64(500), app.draft.TotalAmountCents)
}

func TestAdminRedPacketHandlerCreateReturnsDraftValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	draft := service.RedPacketDraft{
		Name:               "Invalid fixed packet",
		PacketType:         service.RedPacketTypeFixed,
		TotalAmountCents:   500,
		TargetParticipants: 10,
		WinnerCount:        3,
	}
	app := &adminRedPacketApplicationStub{err: service.ValidateRedPacketDraft(draft)}
	h := &RedPacketHandler{service: app}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 9})
		c.Next()
	})
	router.POST("/admin/red-packets", h.Create)

	body := `{"name":"Invalid fixed packet","packet_type":"fixed","total_amount_cents":500,"target_participants":10,"winner_count":3}`
	req := httptest.NewRequest(http.MethodPost, "/admin/red-packets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
