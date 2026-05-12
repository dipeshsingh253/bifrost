package monitoring

import "github.com/gin-gonic/gin"

func RegisterRoutes(protected *gin.RouterGroup, handler *Handler) {
	protected.GET("/servers", handler.ListServers)
	protected.GET("/servers/:serverID", handler.ServerDetail)
	protected.GET("/servers/:serverID/metrics", handler.ServerMetrics)
	protected.GET("/servers/:serverID/projects", handler.ListProjects)
	protected.GET("/servers/:serverID/projects/:projectID", handler.ProjectDetail)
	protected.GET("/servers/:serverID/projects/:projectID/metrics", handler.ProjectMetrics)
	protected.GET("/servers/:serverID/projects/:projectID/logs", handler.ProjectLogs)
	protected.GET("/servers/:serverID/projects/:projectID/events", handler.ProjectEvents)
	protected.GET("/servers/:serverID/containers", handler.ListContainers)
	protected.GET("/servers/:serverID/containers/:containerID", handler.ContainerDetail)
	protected.GET("/servers/:serverID/containers/:containerID/metrics", handler.ContainerMetrics)
	protected.GET("/servers/:serverID/containers/:containerID/logs", handler.ContainerLogs)
	protected.GET("/servers/:serverID/containers/:containerID/events", handler.ContainerEvents)
	protected.GET("/servers/:serverID/containers/:containerID/env", handler.ContainerEnv)
	protected.GET("/services/:serviceID", handler.ServiceDetail)
	protected.GET("/services/:serviceID/logs", handler.ServiceLogs)
}
