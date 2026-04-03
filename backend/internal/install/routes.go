package install

import "github.com/gin-gonic/gin"

func RegisterRoutes(api *gin.RouterGroup, handler *Handler) {
	api.GET("/agent/install", handler.InstallBinary)
	api.GET("/agent/install.sh", handler.InstallScript)
}
