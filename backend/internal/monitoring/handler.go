package monitoring

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dipesh/bifrost/backend/internal/auth"
	sharedhttp "github.com/dipesh/bifrost/backend/internal/shared/http"
	"github.com/dipesh/bifrost/backend/internal/store"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListServers(c *gin.Context) {
	user := auth.UserFromContext(c)
	servers, err := h.service.ListServers(c.Request.Context(), user.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load servers", "SERVER_LIST_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	response := make([]gin.H, 0, len(servers))
	for _, server := range servers {
		services, servicesErr := h.service.ServicesByServer(c.Request.Context(), user.TenantID, server.ID)
		if servicesErr != nil {
			c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load services", "SERVER_LIST_FAILED", gin.H{"reason": servicesErr.Error()}))
			return
		}
		response = append(response, gin.H{
			"server":   newServerView(server),
			"services": newServicesView(services),
		})
	}

	c.JSON(http.StatusOK, sharedhttp.Success(response))
}

func (h *Handler) ServerDetail(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	bundle, err := h.service.ServerBundle(c.Request.Context(), user.TenantID, server.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"server":           newServerView(bundle.Server),
		"services":         newServicesView(bundle.Services),
		"metrics":          bundle.Metrics,
		"containerMetrics": remapContainerMetricBundle(bundle.ContainerMetrics),
	}))
}

func (h *Handler) ServerMetrics(c *gin.Context) {
	user := auth.UserFromContext(c)
	serverID := c.Param("serverID")
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	metrics, err := h.service.MetricsByServer(c.Request.Context(), server.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load server metrics", "SERVER_METRICS_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(metrics))
}

func (h *Handler) ListProjects(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	projects, err := h.service.ProjectsByServer(c.Request.Context(), user.TenantID, server.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load projects", "PROJECT_LIST_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"server":   newServerView(server),
		"projects": newServicesView(projects),
	}))
}

func (h *Handler) ProjectDetail(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	project, err := h.service.ProjectByID(c.Request.Context(), user.TenantID, server.ID, c.Param("projectID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"server":  newServerView(server),
		"project": newServiceView(project),
	}))
}

func (h *Handler) ProjectMetrics(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	projectID := c.Param("projectID")

	project, err := h.service.ProjectByID(c.Request.Context(), user.TenantID, server.ID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	metrics, err := h.service.ProjectMetrics(c.Request.Context(), user.TenantID, server.ID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"project": newServiceView(project),
		"metrics": remapContainerMetricBundle(metrics),
	}))
}

func (h *Handler) ProjectLogs(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	projectID := c.Param("projectID")

	project, err := h.service.ProjectByID(c.Request.Context(), user.TenantID, server.ID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	search, ok := monitoringSearch(c)
	if !ok {
		return
	}
	logLines, err := h.service.LogsByService(c.Request.Context(), project.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load project logs", "PROJECT_LOGS_FAILED", gin.H{"reason": err.Error()}))
		return
	}
	logs := store.FilterLogs(logLines, search, parseLimit(c, 200))
	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"project": newServiceView(project),
		"logs":    newLogLinesView(logs),
	}))
}

func (h *Handler) ProjectEvents(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	projectID := c.Param("projectID")

	events, project, err := h.service.ProjectEvents(c.Request.Context(), user.TenantID, server.ID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"project": newServiceView(project),
		"events":  limitEvents(events, parseLimit(c, 100)),
	}))
}

func (h *Handler) ListContainers(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	standaloneOnly := strings.EqualFold(c.DefaultQuery("standalone", "false"), "true")
	var containers any
	if standaloneOnly {
		standalone, standaloneErr := h.service.StandaloneContainersByServer(c.Request.Context(), user.TenantID, server.ID)
		if standaloneErr != nil {
			c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load containers", "CONTAINER_LIST_FAILED", gin.H{"reason": standaloneErr.Error()}))
			return
		}
		containerViews := make([]ContainerView, 0, len(standalone))
		for _, container := range standalone {
			containerViews = append(containerViews, newContainerView(container))
		}
		containers = containerViews
	} else {
		bundle, bundleErr := h.service.ServerBundle(c.Request.Context(), user.TenantID, server.ID)
		if bundleErr != nil {
			c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
			return
		}
		containers = newStandaloneContainersView(bundle.Services)
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"server":     newServerView(server),
		"containers": containers,
	}))
}

func (h *Handler) ContainerDetail(c *gin.Context) {
	user := auth.UserFromContext(c)
	serverRouteID := c.Param("serverID")
	containerID := c.Param("containerID")

	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, serverRouteID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	container, project, err := h.service.ContainerByID(c.Request.Context(), user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"server":    newServerView(server),
		"project":   newServiceView(project),
		"container": newContainerView(container),
	}))
}

func (h *Handler) ContainerMetrics(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	containerID := c.Param("containerID")

	metrics, container, project, err := h.service.ContainerMetrics(c.Request.Context(), user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"project":   newServiceView(project),
		"container": newContainerView(container),
		"metrics":   metrics,
	}))
}

func (h *Handler) ContainerLogs(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	containerID := c.Param("containerID")

	container, project, err := h.service.ContainerByID(c.Request.Context(), user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	search, ok := monitoringSearch(c)
	if !ok {
		return
	}
	logLines, err := h.service.LogsByContainer(c.Request.Context(), project.ID, container.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load container logs", "CONTAINER_LOGS_FAILED", gin.H{"reason": err.Error()}))
		return
	}
	logs := store.FilterLogs(logLines, search, parseLimit(c, 200))
	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"project":   newServiceView(project),
		"container": newContainerView(container),
		"logs":      newLogLinesView(logs),
	}))
}

func (h *Handler) ContainerEvents(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	containerID := c.Param("containerID")

	events, container, project, err := h.service.ContainerEvents(c.Request.Context(), user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"project":   newServiceView(project),
		"container": newContainerView(container),
		"events":    limitEvents(events, parseLimit(c, 100)),
	}))
}

func (h *Handler) ContainerEnv(c *gin.Context) {
	user := auth.UserFromContext(c)
	server, err := h.service.ServerByID(c.Request.Context(), user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	containerID := c.Param("containerID")

	env, container, project, err := h.service.ContainerEnv(c.Request.Context(), user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"project":   newServiceView(project),
		"container": newContainerView(container),
		"env":       env,
	}))
}

func (h *Handler) ServiceDetail(c *gin.Context) {
	user := auth.UserFromContext(c)
	service, err := h.service.ServiceByID(c.Request.Context(), user.TenantID, c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("service not found", "SERVICE_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(newServiceView(service)))
}

func (h *Handler) ServiceLogs(c *gin.Context) {
	user := auth.UserFromContext(c)
	service, err := h.service.ServiceByID(c.Request.Context(), user.TenantID, c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("service not found", "SERVICE_NOT_FOUND", nil))
		return
	}

	logLines, err := h.service.LogsByService(c.Request.Context(), service.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load service logs", "SERVICE_LOGS_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"service": newServiceView(service),
		"logs":    newLogLinesView(logLines),
	}))
}

func parseLimit(c *gin.Context, fallback int) int {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}

	if fallback > 0 && value > fallback {
		return fallback
	}

	return value
}

func monitoringSearch(c *gin.Context) (string, bool) {
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 256 {
		c.JSON(http.StatusBadRequest, sharedhttp.Error("search query must be 256 characters or fewer", "INVALID_REQUEST", nil))
		return "", false
	}
	return search, true
}

func limitEvents(events []EventLog, limit int) []EventLog {
	if limit > 0 && len(events) > limit {
		return events[:limit]
	}
	return events
}
