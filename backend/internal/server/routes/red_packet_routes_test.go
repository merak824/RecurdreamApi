package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRedPacketRoutesAreRegisteredForUsersAndAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{
		RedPacket: handler.NewRedPacketHandler(nil),
		Redeem:    handler.NewRedeemHandler(nil),
		Admin: &handler.AdminHandlers{
			RedPacket: adminhandler.NewRedPacketHandler(nil),
		},
	}
	jwtAuth := servermiddleware.JWTAuthMiddleware(func(c *gin.Context) { c.Next() })
	adminAuth := servermiddleware.AdminAuthMiddleware(func(c *gin.Context) { c.Next() })
	auditLog := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })
	stepUp := servermiddleware.StepUpAuthMiddleware(func(c *gin.Context) { c.Next() })

	RegisterUserRoutes(router.Group("/api/v1"), handlers, jwtAuth, auditLog, nil, nil)
	RegisterAdminRoutes(router.Group("/api/v1"), handlers, adminAuth, auditLog, stepUp, nil, nil)

	want := map[string]struct{}{
		"GET /api/v1/balance-history":                {},
		"GET /api/v1/red-packets/current":            {},
		"GET /api/v1/red-packets/eligibility":        {},
		"GET /api/v1/red-packets/recent":             {},
		"GET /api/v1/red-packets/rewards":            {},
		"GET /api/v1/red-packets/:id":                {},
		"POST /api/v1/red-packets/:id/participate":   {},
		"GET /api/v1/admin/red-packets":              {},
		"POST /api/v1/admin/red-packets":             {},
		"PUT /api/v1/admin/red-packets/:id":          {},
		"POST /api/v1/admin/red-packets/:id/publish": {},
		"POST /api/v1/admin/red-packets/:id/cancel":  {},
		"GET /api/v1/admin/red-packets/:id/export":   {},
	}
	for _, route := range router.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	require.Empty(t, want)
}

func TestAdminRedPacketRoutesRejectNonAdminRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{
		RedPacket: adminhandler.NewRedPacketHandler(nil),
	}}
	adminAuth := servermiddleware.AdminAuthMiddleware(func(c *gin.Context) {
		servermiddleware.AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
	})
	auditLog := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })
	stepUp := servermiddleware.StepUpAuthMiddleware(func(c *gin.Context) { c.Next() })
	RegisterAdminRoutes(router.Group("/api/v1"), handlers, adminAuth, auditLog, stepUp, nil, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/red-packets", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}
