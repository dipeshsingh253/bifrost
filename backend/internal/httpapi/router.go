package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dipesh/bifrost/backend/internal/config"
	"github.com/dipesh/bifrost/backend/internal/domain"
	"github.com/dipesh/bifrost/backend/internal/store"
)

type Router struct {
	config config.Config
	store  *store.MemoryStore
}

func NewRouter(cfg config.Config, dataStore *store.MemoryStore) *gin.Engine {
	router := &Router{
		config: cfg,
		store:  dataStore,
	}

	engine := gin.Default()
	engine.GET("/health", router.health)

	api := engine.Group("/api/v1")
	{
		api.POST("/auth/login", router.login)
		api.POST("/agents/enroll", router.enrollAgent)
		api.POST("/agents/heartbeat", router.agentHeartbeat)
		api.POST("/ingest/snapshot", router.ingestSnapshot)
	}

	protected := api.Group("/")
	protected.Use(router.authRequired())
	{
		protected.GET("/me", router.me)
		protected.GET("/servers", router.listServers)
		protected.GET("/servers/:serverID", router.serverDetail)
		protected.GET("/servers/:serverID/metrics", router.serverMetrics)
		protected.GET("/servers/:serverID/projects", router.listProjects)
		protected.GET("/servers/:serverID/projects/:projectID", router.projectDetail)
		protected.GET("/servers/:serverID/projects/:projectID/metrics", router.projectMetrics)
		protected.GET("/servers/:serverID/projects/:projectID/logs", router.projectLogs)
		protected.GET("/servers/:serverID/projects/:projectID/events", router.projectEvents)
		protected.GET("/servers/:serverID/containers", router.listContainers)
		protected.GET("/servers/:serverID/containers/:containerID", router.containerDetail)
		protected.GET("/servers/:serverID/containers/:containerID/metrics", router.containerMetrics)
		protected.GET("/servers/:serverID/containers/:containerID/logs", router.containerLogs)
		protected.GET("/servers/:serverID/containers/:containerID/events", router.containerEvents)
		protected.GET("/servers/:serverID/containers/:containerID/env", router.containerEnv)
		protected.GET("/services/:serviceID", router.serviceDetail)
		protected.GET("/services/:serviceID/logs", router.serviceLogs)
	}

	return engine
}

func (r *Router) health(c *gin.Context) {
	c.JSON(http.StatusOK, successResponse(gin.H{
		"service": "bifrost-backend",
		"status":  "ok",
	}))
}

func (r *Router) login(c *gin.Context) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		return
	}

	user, err := r.store.Authenticate(request.Email, request.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("invalid email or password", "INVALID_CREDENTIALS", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"token": user.AuthToken,
		"user":  user,
	}))
}

func (r *Router) me(c *gin.Context) {
	user := userFromContext(c)
	c.JSON(http.StatusOK, successResponse(user))
}

func (r *Router) listServers(c *gin.Context) {
	user := userFromContext(c)
	servers := r.store.ListServers(user.TenantID)

	response := make([]gin.H, 0, len(servers))
	for _, server := range servers {
		response = append(response, gin.H{
			"server":   server,
			"services": r.store.ServicesByServer(user.TenantID, server.ID),
		})
	}

	c.JSON(http.StatusOK, successResponse(response))
}

func (r *Router) serverDetail(c *gin.Context) {
	user := userFromContext(c)
	bundle, err := r.store.ServerBundle(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(bundle))
}

func (r *Router) serverMetrics(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	if _, err := r.store.ServerByID(user.TenantID, serverID); err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(r.store.MetricsByServer(serverID)))
}

func (r *Router) listProjects(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"server":   server,
		"projects": r.store.ProjectsByServer(user.TenantID, server.ID),
	}))
}

func (r *Router) projectDetail(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	project, err := r.store.ProjectByID(user.TenantID, server.ID, c.Param("projectID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"server":  server,
		"project": project,
	}))
}

func (r *Router) projectMetrics(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	projectID := c.Param("projectID")

	project, err := r.store.ProjectByID(user.TenantID, serverID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	metrics, err := r.store.ProjectMetrics(user.TenantID, serverID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project": project,
		"metrics": metrics,
	}))
}

func (r *Router) projectLogs(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	projectID := c.Param("projectID")

	project, err := r.store.ProjectByID(user.TenantID, serverID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	logs := store.FilterLogs(r.store.LogsByService(project.ID), c.Query("search"), parseLimit(c, 200))
	c.JSON(http.StatusOK, successResponse(gin.H{
		"project": project,
		"logs":    logs,
	}))
}

func (r *Router) projectEvents(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	projectID := c.Param("projectID")

	events, project, err := r.store.ProjectEvents(user.TenantID, serverID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project": project,
		"events":  limitEvents(events, parseLimit(c, 20)),
	}))
}

func (r *Router) listContainers(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	standaloneOnly := strings.EqualFold(c.DefaultQuery("standalone", "false"), "true")
	var containers any
	if standaloneOnly {
		containers = r.store.StandaloneContainersByServer(user.TenantID, server.ID)
	} else {
		bundle, bundleErr := r.store.ServerBundle(user.TenantID, server.ID)
		if bundleErr != nil {
			c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
			return
		}
		containers = flattenContainers(bundle.Services)
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"server":     server,
		"containers": containers,
	}))
}

func (r *Router) containerDetail(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	containerID := c.Param("containerID")

	server, err := r.store.ServerByID(user.TenantID, serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	container, project, err := r.store.ContainerByID(user.TenantID, serverID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"server":    server,
		"project":   project,
		"container": container,
	}))
}

func (r *Router) containerMetrics(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	containerID := c.Param("containerID")

	metrics, container, project, err := r.store.ContainerMetrics(user.TenantID, serverID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project":   project,
		"container": container,
		"metrics":   metrics,
	}))
}

func (r *Router) containerLogs(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	containerID := c.Param("containerID")

	container, project, err := r.store.ContainerByID(user.TenantID, serverID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	logs := store.FilterLogs(r.store.LogsByContainer(project.ID, container.ID), c.Query("search"), parseLimit(c, 200))
	c.JSON(http.StatusOK, successResponse(gin.H{
		"project":   project,
		"container": container,
		"logs":      logs,
	}))
}

func (r *Router) containerEvents(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	containerID := c.Param("containerID")

	events, container, project, err := r.store.ContainerEvents(user.TenantID, serverID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project":   project,
		"container": container,
		"events":    limitEvents(events, parseLimit(c, 20)),
	}))
}

func (r *Router) containerEnv(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	containerID := c.Param("containerID")

	env, container, project, err := r.store.ContainerEnv(user.TenantID, serverID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project":   project,
		"container": container,
		"env":       env,
	}))
}

func (r *Router) serviceDetail(c *gin.Context) {
	user := userFromContext(c)
	service, err := r.store.ServiceByID(user.TenantID, c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("service not found", "SERVICE_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(service))
}

func (r *Router) serviceLogs(c *gin.Context) {
	user := userFromContext(c)
	service, err := r.store.ServiceByID(user.TenantID, c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("service not found", "SERVICE_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"service": service,
		"logs":    r.store.LogsByService(service.ID),
	}))
}

func (r *Router) enrollAgent(c *gin.Context) {
	var request struct {
		AgentID     string `json:"agent_id"`
		TenantID    string `json:"tenant_id"`
		ServerID    string `json:"server_id"`
		Name        string `json:"name"`
		Version     string `json:"version"`
		ServerName  string `json:"server_name"`
		Hostname    string `json:"hostname"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		return
	}

	agent := r.store.EnrollAgent(domain.Agent{
		ID:          request.AgentID,
		TenantID:    request.TenantID,
		ServerID:    request.ServerID,
		Name:        request.Name,
		APIKey:      request.AgentID + "-key",
		Version:     request.Version,
		ServerName:  request.ServerName,
		Hostname:    request.Hostname,
		Description: request.Description,
	})

	c.JSON(http.StatusCreated, successResponse(agent))
}

func (r *Router) agentHeartbeat(c *gin.Context) {
	var request struct {
		AgentID string `json:"agent_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		return
	}

	if err := r.store.UpdateAgentLastSeen(request.AgentID); err != nil {
		c.JSON(http.StatusNotFound, errorResponse("agent not found", "AGENT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{"status": "ok"}))
}

func (r *Router) ingestSnapshot(c *gin.Context) {
	apiKey := c.GetHeader("X-Agent-Key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("missing agent api key", "UNAUTHORIZED", nil))
		return
	}

	if _, err := r.store.AgentByAPIKey(apiKey); err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("invalid agent api key", "UNAUTHORIZED", nil))
		return
	}

	var payload domain.IngestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		return
	}

	if payload.Server.CollectedAt.IsZero() {
		payload.Server.CollectedAt = time.Now().UTC()
	}

	if err := r.store.Ingest(payload); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("failed to persist snapshot", "INGEST_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusAccepted, successResponse(gin.H{"status": "accepted"}))
}

func (r *Router) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" || token == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("missing bearer token", "UNAUTHORIZED", nil))
			return
		}

		user, err := r.store.UserByToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("invalid token", "UNAUTHORIZED", nil))
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func userFromContext(c *gin.Context) domain.User {
	value, _ := c.Get("user")
	user, _ := value.(domain.User)
	return user
}

func successResponse(data any) gin.H {
	return gin.H{
		"success": true,
		"data":    data,
	}
}

func errorResponse(message string, code string, details any) gin.H {
	return gin.H{
		"success": false,
		"error": gin.H{
			"message": message,
			"code":    code,
			"details": details,
		},
	}
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

	return value
}

func limitEvents(events []domain.EventLog, limit int) []domain.EventLog {
	if limit > 0 && len(events) > limit {
		return events[:limit]
	}
	return events
}

func flattenContainers(services []domain.Service) []domain.Container {
	containers := make([]domain.Container, 0)
	for _, service := range services {
		for _, container := range service.Containers {
			containerCopy := container
			containerCopy.Ports = append([]string(nil), container.Ports...)
			containers = append(containers, containerCopy)
		}
	}
	return containers
}
