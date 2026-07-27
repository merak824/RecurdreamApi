package routes

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes registers read-only endpoints used by unauthenticated pages.
func RegisterPublicRoutes(v1 *gin.RouterGroup, channelMonitorList gin.HandlerFunc) {
	public := v1.Group("/public")
	{
		public.GET("/channel-monitors", channelMonitorList)
	}
}
