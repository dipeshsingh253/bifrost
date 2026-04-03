package onboarding

import "github.com/gin-gonic/gin"

func RegisterRoutes(admin *gin.RouterGroup, handler *Handler) {
	admin.GET("/systems", handler.List)
	admin.GET("/systems/:systemID", handler.Detail)
	admin.POST("/systems", handler.Create)
	admin.POST("/systems/:systemID/cancel", handler.Cancel)
	admin.POST("/systems/:systemID/reissue", handler.Reissue)
}
