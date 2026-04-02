package auth

import "github.com/gin-gonic/gin"

func RegisterPublicRoutes(api *gin.RouterGroup, handler *Handler) {
	api.GET("/auth/bootstrap/status", handler.BootstrapStatus)
	api.GET("/auth/invites/:token", handler.InviteDetail)
	api.POST("/auth/invites/accept", handler.AcceptInvite)
	api.POST("/auth/bootstrap", handler.BootstrapAdmin)
	api.POST("/auth/login", handler.Login)
}

func RegisterProtectedRoutes(protected *gin.RouterGroup, handler *Handler) {
	protected.GET("/auth/session", handler.Me)
	protected.POST("/auth/logout", handler.Logout)
	protected.GET("/auth/me", handler.Me)
	protected.PATCH("/auth/me", handler.UpdateProfile)
	protected.POST("/auth/me/password", handler.ChangePassword)
}
