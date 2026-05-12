package agent

import "github.com/gin-gonic/gin"

func RegisterRoutes(api *gin.RouterGroup, handler *Handler) {
	api.POST("/agent/heartbeat", handler.Heartbeat)
	api.POST("/agent/enroll", handler.Enroll)
	api.POST("/agent/snapshot", handler.Snapshot)
}
