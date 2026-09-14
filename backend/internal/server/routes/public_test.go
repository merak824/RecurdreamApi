package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterPublicRoutesExposesChannelMonitorsWithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	called := false

	RegisterPublicRoutes(router.Group("/api/v1"), func(c *gin.Context) {
		called = true
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/channel-monitors", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.True(t, called)
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"items":[]}`, response.Body.String())
}
