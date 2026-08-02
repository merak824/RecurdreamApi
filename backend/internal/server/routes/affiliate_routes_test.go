package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminAffiliateRoutesExposeSimplifiedWithdrawalWorkflow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{
		Affiliate: adminhandler.NewAffiliateHandler(nil, nil),
	}}
	adminAuth := servermiddleware.AdminAuthMiddleware(func(c *gin.Context) { c.Next() })
	auditLog := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })
	stepUp := servermiddleware.StepUpAuthMiddleware(func(c *gin.Context) { c.Next() })
	RegisterAdminRoutes(router.Group("/api/v1"), handlers, adminAuth, auditLog, stepUp, nil, nil)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	for _, route := range []string{
		"GET /api/v1/admin/affiliates/withdrawals",
		"POST /api/v1/admin/affiliates/withdrawals/:id/paid",
		"POST /api/v1/admin/affiliates/withdrawals/:id/reject",
		"PUT /api/v1/admin/affiliates/users/:user_id",
		"DELETE /api/v1/admin/affiliates/users/:user_id",
	} {
		require.Contains(t, routes, route)
	}
	require.NotContains(t, routes, "POST /api/v1/admin/affiliates/users/batch-rate")
	require.NotContains(t, routes, "GET /api/v1/admin/affiliates/users/:user_id/usage")
}
