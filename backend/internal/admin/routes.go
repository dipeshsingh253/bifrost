package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(admin *gin.RouterGroup, handler *Handler) {
	admin.GET("/access", handler.ViewerAccess)
	admin.GET("/summary", handler.Summary)
	admin.POST("/invites", handler.CreateViewerInvite)
	admin.POST("/invites/:inviteID/revoke", handler.RevokeViewerInvite)
	admin.POST("/viewers/:userID/disable", handler.DisableViewer)
	admin.DELETE("/viewers/:userID", handler.DeleteViewer)
}
