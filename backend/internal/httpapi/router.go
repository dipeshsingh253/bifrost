package httpapi

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dipesh/bifrost/backend/internal/config"
	"github.com/dipesh/bifrost/backend/internal/domain"
	"github.com/dipesh/bifrost/backend/internal/service/systemonboarding"
	"github.com/dipesh/bifrost/backend/internal/store"
)

const sessionCookieName = "bifrost_session"
const sessionCookieMaxAgeSeconds = 30 * 24 * 60 * 60

type Router struct {
	store             store.Repository
	systemOnboardings *systemonboarding.Service
	agentDockerImage  string
	agentBinaryPath   string
	agentSourceDir    string
}

func NewRouter(cfg config.Config, dataStore store.Repository) *gin.Engine {
	router := &Router{
		store:             dataStore,
		systemOnboardings: systemonboarding.New(dataStore, cfg.AgentBackendURL, cfg.AgentDockerImage),
		agentDockerImage:  cfg.AgentDockerImage,
		agentBinaryPath:   cfg.AgentBinaryPath,
		agentSourceDir:    cfg.AgentSourceDir,
	}

	engine := gin.Default()
	engine.GET("/health", router.health)

	api := engine.Group("/api/v1")
	{
		api.GET("/install/agent", router.installAgentBinary)
		api.GET("/install/agent.sh", router.installAgentScript)
		api.GET("/auth/bootstrap/status", router.bootstrapStatus)
		api.GET("/auth/invites/:token", router.inviteDetail)
		api.POST("/auth/invites/accept", router.acceptInvite)
		api.POST("/auth/bootstrap", router.bootstrapAdmin)
		api.POST("/agents/enroll", router.agentEnroll)
		api.POST("/auth/login", router.login)
		api.POST("/agents/heartbeat", router.agentHeartbeat)
		api.POST("/ingest/snapshot", router.ingestSnapshot)
	}

	protected := api.Group("/")
	protected.Use(router.authRequired())
	{
		protected.GET("/auth/session", router.me)
		protected.POST("/auth/logout", router.logout)
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

	admin := protected.Group("/admin")
	admin.Use(router.adminRequired())
	{
		admin.GET("/access", router.viewerAccess)
		admin.GET("/summary", router.adminSummary)
		admin.GET("/systems", router.listSystemOnboardings)
		admin.GET("/systems/:systemID", router.systemOnboardingDetail)
		admin.POST("/invites", router.createViewerInvite)
		admin.POST("/systems", router.createSystemOnboarding)
		admin.POST("/systems/:systemID/cancel", router.cancelSystemOnboarding)
		admin.POST("/systems/:systemID/reissue", router.reissueSystemOnboarding)
		admin.POST("/invites/:inviteID/revoke", router.revokeViewerInvite)
		admin.POST("/viewers/:userID/disable", router.disableViewer)
		admin.DELETE("/viewers/:userID", router.deleteViewer)
	}

	return engine
}

func (r *Router) health(c *gin.Context) {
	c.JSON(http.StatusOK, successResponse(gin.H{
		"service": "bifrost-backend",
		"status":  "ok",
	}))
}

func (r *Router) installAgentScript(c *gin.Context) {
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"",
		`require_var() {`,
		`  eval "value=\${$1:-}"`,
		`  if [ -z "$value" ]; then`,
		`    echo "$1 is required" >&2`,
		`    exit 1`,
		`  fi`,
		`}`,
		"",
		`require_var BIFROST_AGENT_ID`,
		`require_var BIFROST_SERVER_ID`,
		`require_var BIFROST_SERVER_NAME`,
		`require_var BIFROST_TENANT_ID`,
		`require_var BIFROST_BACKEND_URL`,
		`require_var BIFROST_ENROLLMENT_TOKEN`,
		"",
		`if ! command -v systemctl >/dev/null 2>&1; then`,
		`  echo "systemctl is required for the host install flow" >&2`,
		`  exit 1`,
		`fi`,
		"",
		`AGENT_IMAGE="${BIFROST_AGENT_IMAGE:-` + r.agentDockerImage + `}"`,
		`AGENT_BINARY_URL="${BIFROST_AGENT_BINARY_URL:-` + installAgentBinaryURL(defaultBackendURL(c.Request)) + `}"`,
		`AGENT_DIR="${BIFROST_AGENT_DIR:-/etc/bifrost-agent}"`,
		`STATE_DIR="${BIFROST_AGENT_STATE_DIR:-/var/lib/bifrost-agent}"`,
		`ENV_FILE="${BIFROST_AGENT_ENV_FILE:-/etc/default/bifrost-agent}"`,
		`SERVICE_FILE="${BIFROST_AGENT_SERVICE_FILE:-/etc/systemd/system/bifrost-agent.service}"`,
		`BINARY_PATH="${BIFROST_AGENT_BINARY_PATH:-/usr/local/bin/bifrost-agent}"`,
		`CONFIG_PATH="${BIFROST_CONFIG_PATH:-$AGENT_DIR/config.yaml}"`,
		"",
		`mkdir -p "$AGENT_DIR" "$STATE_DIR"`,
		"",
		`cat > "$ENV_FILE" <<EOF`,
		`BIFROST_CONFIG_PATH=$CONFIG_PATH`,
		`BIFROST_AGENT_ID=$BIFROST_AGENT_ID`,
		`BIFROST_SERVER_ID=$BIFROST_SERVER_ID`,
		`BIFROST_SERVER_NAME=$BIFROST_SERVER_NAME`,
		`BIFROST_TENANT_ID=$BIFROST_TENANT_ID`,
		`BIFROST_BACKEND_URL=$BIFROST_BACKEND_URL`,
		`BIFROST_ENROLLMENT_TOKEN=$BIFROST_ENROLLMENT_TOKEN`,
		`EOF`,
		"",
		`install_binary_from_image() {`,
		`  if ! command -v docker >/dev/null 2>&1; then`,
		`    return 1`,
		`  fi`,
		`  tmp_container="bifrost-agent-install-$$"`,
		`  if ! docker image inspect "$AGENT_IMAGE" >/dev/null 2>&1; then`,
		`    docker pull "$AGENT_IMAGE" >/dev/null`,
		`  fi`,
		`  docker create --name "$tmp_container" "$AGENT_IMAGE" >/dev/null`,
		`  docker cp "$tmp_container:/usr/local/bin/bifrost-agent" "$BINARY_PATH"`,
		`  docker rm -f "$tmp_container" >/dev/null`,
		`  chmod 0755 "$BINARY_PATH"`,
		`}`,
		"",
		`download_binary() {`,
		`  curl -fsSL "$AGENT_BINARY_URL" -o "$BINARY_PATH"`,
		`  chmod 0755 "$BINARY_PATH"`,
		`}`,
		"",
		`if [ -n "$AGENT_BINARY_URL" ]; then`,
		`  if ! command -v curl >/dev/null 2>&1; then`,
		`    echo "curl is required when BIFROST_AGENT_BINARY_URL is set" >&2`,
		`    exit 1`,
		`  fi`,
		`  if ! download_binary; then`,
		`    if ! install_binary_from_image; then`,
		`      echo "failed to download agent binary and no usable docker image fallback was available" >&2`,
		`      exit 1`,
		`    fi`,
		`  fi`,
		`elif ! install_binary_from_image; then`,
		`  echo "docker is required to extract the agent binary, or set BIFROST_AGENT_BINARY_URL to a downloadable binary" >&2`,
		`  exit 1`,
		`fi`,
		"",
		`cat > "$SERVICE_FILE" <<EOF`,
		`[Unit]`,
		`Description=Bifrost Agent`,
		`After=network-online.target`,
		`Wants=network-online.target`,
		``,
		`[Service]`,
		`WorkingDirectory=$AGENT_DIR`,
		`EnvironmentFile=$ENV_FILE`,
		`ExecStart=$BINARY_PATH`,
		`Restart=always`,
		`RestartSec=5`,
		``,
		`[Install]`,
		`WantedBy=multi-user.target`,
		`EOF`,
		"",
		`systemctl daemon-reload`,
		`systemctl enable --now bifrost-agent.service`,
		`echo "Bifrost agent installed and started."`,
	}, "\n") + "\n"

	c.Header("Content-Type", "text/x-shellscript; charset=utf-8")
	c.String(http.StatusOK, script)
}

func (r *Router) installAgentBinary(c *gin.Context) {
	binaryPath, cleanup, err := r.resolveAgentBinaryPath()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("failed to prepare agent binary", "AGENT_BINARY_UNAVAILABLE", gin.H{"reason": err.Error()}))
		return
	}
	defer cleanup()

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="bifrost-agent"`)
	c.File(binaryPath)
}

func (r *Router) resolveAgentBinaryPath() (string, func(), error) {
	if binaryPath := strings.TrimSpace(r.agentBinaryPath); binaryPath != "" {
		if _, err := os.Stat(binaryPath); err != nil {
			return "", nil, fmt.Errorf("read configured agent binary: %w", err)
		}
		return binaryPath, func() {}, nil
	}

	sourceDir := strings.TrimSpace(r.agentSourceDir)
	if sourceDir == "" {
		return "", nil, fmt.Errorf("agent source dir is not configured")
	}

	tempDir, err := os.MkdirTemp("", "bifrost-agent-build-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp build dir: %w", err)
	}

	binaryPath := filepath.Join(tempDir, "bifrost-agent")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = sourceDir
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("build agent binary: %v: %s", err, strings.TrimSpace(string(output)))
	}

	return binaryPath, func() {
		_ = os.RemoveAll(tempDir)
	}, nil
}

func defaultBackendURL(request *http.Request) string {
	if request == nil {
		return ""
	}
	if request.Host == "" {
		return ""
	}

	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(request.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}

	return scheme + "://" + request.Host
}

func installAgentBinaryURL(backendURL string) string {
	return strings.TrimRight(strings.TrimSpace(backendURL), "/") + "/api/v1/install/agent"
}

func (r *Router) bootstrapStatus(c *gin.Context) {
	needsBootstrap, err := r.store.BootstrapStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("failed to read bootstrap status", "BOOTSTRAP_STATUS_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"needs_bootstrap": needsBootstrap,
	}))
}

func (r *Router) inviteDetail(c *gin.Context) {
	invite, err := r.store.InviteByToken(strings.TrimSpace(c.Param("token")))
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, errorResponse("invite is no longer available", "INVITE_UNAVAILABLE", nil))
			return
		}
		c.JSON(http.StatusNotFound, errorResponse("invite not found", "INVITE_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(invite))
}

func (r *Router) acceptInvite(c *gin.Context) {
	var request struct {
		Token    string `json:"token"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		return
	}

	request.Token = strings.TrimSpace(request.Token)
	request.Name = strings.TrimSpace(request.Name)
	request.Password = strings.TrimSpace(request.Password)
	if request.Token == "" || request.Name == "" || request.Password == "" {
		c.JSON(http.StatusBadRequest, errorResponse("token, name, and password are required", "INVALID_REQUEST", nil))
		return
	}

	user, err := r.store.AcceptViewerInvite(request.Token, request.Name, request.Password)
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, errorResponse("invite is no longer available", "INVITE_UNAVAILABLE", nil))
			return
		}
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, errorResponse("invite not found", "INVITE_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to accept invite", "INVITE_ACCEPT_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	r.setSessionCookie(c, user.AuthToken)
	c.JSON(http.StatusCreated, successResponse(gin.H{
		"user": user,
	}))
}

func (r *Router) bootstrapAdmin(c *gin.Context) {
	var request struct {
		TenantName string `json:"tenant_name"`
		Name       string `json:"name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		return
	}

	request.TenantName = strings.TrimSpace(request.TenantName)
	request.Name = strings.TrimSpace(request.Name)
	request.Email = strings.TrimSpace(strings.ToLower(request.Email))
	request.Password = strings.TrimSpace(request.Password)

	if request.Name == "" || request.Email == "" || request.Password == "" {
		c.JSON(http.StatusBadRequest, errorResponse("name, email, and password are required", "INVALID_REQUEST", nil))
		return
	}

	user, err := r.store.BootstrapAdmin(request.TenantName, request.Name, request.Email, request.Password)
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, errorResponse("bootstrap has already been completed", "BOOTSTRAP_ALREADY_COMPLETED", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to bootstrap admin", "BOOTSTRAP_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	r.setSessionCookie(c, user.AuthToken)
	c.JSON(http.StatusCreated, successResponse(gin.H{
		"user": user,
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

	needsBootstrap, err := r.store.BootstrapStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("failed to read bootstrap status", "BOOTSTRAP_STATUS_FAILED", gin.H{"reason": err.Error()}))
		return
	}
	if needsBootstrap {
		c.JSON(http.StatusConflict, errorResponse("bootstrap is required before login", "BOOTSTRAP_REQUIRED", nil))
		return
	}

	user, err := r.store.Authenticate(request.Email, request.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("invalid email or password", "INVALID_CREDENTIALS", nil))
		return
	}

	r.setSessionCookie(c, user.AuthToken)
	c.JSON(http.StatusOK, successResponse(gin.H{
		"user": user,
	}))
}

func (r *Router) me(c *gin.Context) {
	user := userFromContext(c)
	c.JSON(http.StatusOK, successResponse(user))
}

func (r *Router) logout(c *gin.Context) {
	token := authTokenFromContext(c)
	if token != "" {
		if err := r.store.RevokeSession(token); err != nil && err != store.ErrNotFound {
			c.JSON(http.StatusInternalServerError, errorResponse("failed to revoke session", "LOGOUT_FAILED", gin.H{"reason": err.Error()}))
			return
		}
	}

	r.clearSessionCookie(c)
	c.JSON(http.StatusOK, successResponse(gin.H{
		"status": "logged_out",
	}))
}

func (r *Router) adminSummary(c *gin.Context) {
	user := userFromContext(c)

	summary, err := r.store.TenantSummary(user.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("failed to load tenant summary", "TENANT_SUMMARY_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"tenant": summary,
		"user":   user,
	}))
}

func (r *Router) viewerAccess(c *gin.Context) {
	user := userFromContext(c)

	access, err := r.store.ViewerAccess(user.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("failed to load viewer access", "VIEWER_ACCESS_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(access))
}

func (r *Router) createViewerInvite(c *gin.Context) {
	user := userFromContext(c)

	var request struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		return
	}

	request.Email = strings.TrimSpace(strings.ToLower(request.Email))
	if request.Email == "" {
		c.JSON(http.StatusBadRequest, errorResponse("email is required", "INVALID_REQUEST", nil))
		return
	}

	invite, err := r.store.CreateViewerInvite(user.TenantID, user.ID, request.Email)
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, errorResponse("viewer or invite already exists", "INVITE_CONFLICT", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to create invite", "INVITE_CREATE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusCreated, successResponse(invite))
}

func (r *Router) createSystemOnboarding(c *gin.Context) {
	user := userFromContext(c)

	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		return
	}

	onboarding, err := r.systemOnboardings.Create(systemonboarding.CreateInput{
		TenantID:        user.TenantID,
		CreatedByUserID: user.ID,
		Name:            request.Name,
		Description:     request.Description,
	})
	if err == systemonboarding.ErrInvalidName {
		c.JSON(http.StatusBadRequest, errorResponse("name is required", "INVALID_REQUEST", nil))
		return
	}
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, errorResponse("system could not be created", "SYSTEM_ONBOARDING_CONFLICT", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to create system onboarding", "SYSTEM_ONBOARDING_CREATE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusCreated, successResponse(onboarding))
}

func (r *Router) systemOnboardingDetail(c *gin.Context) {
	user := userFromContext(c)

	onboarding, err := r.systemOnboardings.Get(user.TenantID, c.Param("systemID"))
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, errorResponse("system onboarding not found", "SYSTEM_ONBOARDING_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to load system onboarding", "SYSTEM_ONBOARDING_LOAD_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(onboarding))
}

func (r *Router) listSystemOnboardings(c *gin.Context) {
	user := userFromContext(c)

	onboardings, err := r.systemOnboardings.List(user.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("failed to load system onboardings", "SYSTEM_ONBOARDING_LIST_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(onboardings))
}

func (r *Router) cancelSystemOnboarding(c *gin.Context) {
	user := userFromContext(c)

	err := r.systemOnboardings.Cancel(user.TenantID, c.Param("systemID"))
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, errorResponse("system onboarding not found", "SYSTEM_ONBOARDING_NOT_FOUND", nil))
			return
		}
		if err == store.ErrConflict || err == systemonboarding.ErrOnboardingNotPending {
			c.JSON(http.StatusConflict, errorResponse("only pending system onboardings can be cancelled", "SYSTEM_ONBOARDING_NOT_PENDING", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to cancel system onboarding", "SYSTEM_ONBOARDING_CANCEL_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"status": "cancelled",
	}))
}

func (r *Router) reissueSystemOnboarding(c *gin.Context) {
	user := userFromContext(c)

	onboarding, err := r.systemOnboardings.Reissue(user.TenantID, c.Param("systemID"))
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, errorResponse("system onboarding not found", "SYSTEM_ONBOARDING_NOT_FOUND", nil))
			return
		}
		if err == store.ErrConflict || err == systemonboarding.ErrOnboardingNotPending {
			c.JSON(http.StatusConflict, errorResponse("only pending system onboardings can be reissued", "SYSTEM_ONBOARDING_NOT_PENDING", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to reissue system onboarding credentials", "SYSTEM_ONBOARDING_REISSUE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(onboarding))
}

func (r *Router) revokeViewerInvite(c *gin.Context) {
	user := userFromContext(c)

	if err := r.store.RevokeViewerInvite(user.TenantID, c.Param("inviteID")); err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, errorResponse("invite can no longer be revoked", "INVITE_REVOKE_CONFLICT", nil))
			return
		}
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, errorResponse("invite not found", "INVITE_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to revoke invite", "INVITE_REVOKE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"status": "revoked",
	}))
}

func (r *Router) disableViewer(c *gin.Context) {
	user := userFromContext(c)

	if err := r.store.DisableViewer(user.TenantID, c.Param("userID")); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, errorResponse("viewer not found", "VIEWER_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to disable viewer", "VIEWER_DISABLE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"status": "disabled",
	}))
}

func (r *Router) deleteViewer(c *gin.Context) {
	user := userFromContext(c)

	if err := r.store.DeleteViewer(user.TenantID, c.Param("userID")); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, errorResponse("viewer not found", "VIEWER_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to delete viewer", "VIEWER_DELETE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"status": "deleted",
	}))
}

func (r *Router) listServers(c *gin.Context) {
	user := userFromContext(c)
	servers := r.store.ListServers(user.TenantID)

	response := make([]gin.H, 0, len(servers))
	for _, server := range servers {
		services := r.store.ServicesByServer(user.TenantID, server.ID)
		response = append(response, gin.H{
			"server":   newServerView(server),
			"services": newServicesView(services),
		})
	}

	c.JSON(http.StatusOK, successResponse(response))
}

func (r *Router) serverDetail(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	bundle, err := r.store.ServerBundle(user.TenantID, server.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"server":           newServerView(bundle.Server),
		"services":         newServicesView(bundle.Services),
		"metrics":          bundle.Metrics,
		"containerMetrics": remapContainerMetricBundle(bundle.ContainerMetrics, bundle.Services),
	}))
}

func (r *Router) serverMetrics(c *gin.Context) {
	user := userFromContext(c)
	serverID := c.Param("serverID")
	server, err := r.store.ServerByID(user.TenantID, serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(r.store.MetricsByServer(server.ID)))
}

func (r *Router) listProjects(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"server":   newServerView(server),
		"projects": newServicesView(r.store.ProjectsByServer(user.TenantID, server.ID)),
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
		"server":  newServerView(server),
		"project": newServiceView(project),
	}))
}

func (r *Router) projectMetrics(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	projectID := c.Param("projectID")

	project, err := r.store.ProjectByID(user.TenantID, server.ID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	metrics, err := r.store.ProjectMetrics(user.TenantID, server.ID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project": newServiceView(project),
		"metrics": remapContainerMetricBundle(metrics, []domain.Service{project}),
	}))
}

func (r *Router) projectLogs(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	projectID := c.Param("projectID")

	project, err := r.store.ProjectByID(user.TenantID, server.ID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	logs := store.FilterLogs(r.store.LogsByService(project.ID), c.Query("search"), parseLimit(c, 200))
	c.JSON(http.StatusOK, successResponse(gin.H{
		"project": newServiceView(project),
		"logs":    newLogLinesView(logs),
	}))
}

func (r *Router) projectEvents(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	projectID := c.Param("projectID")

	events, project, err := r.store.ProjectEvents(user.TenantID, server.ID, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("project not found", "PROJECT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project": newServiceView(project),
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
		standalone := r.store.StandaloneContainersByServer(user.TenantID, server.ID)
		containerViews := make([]containerView, 0, len(standalone))
		for _, container := range standalone {
			containerViews = append(containerViews, newContainerView(container))
		}
		containers = containerViews
	} else {
		bundle, bundleErr := r.store.ServerBundle(user.TenantID, server.ID)
		if bundleErr != nil {
			c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
			return
		}
		containers = newStandaloneContainersView(bundle.Services)
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"server":     newServerView(server),
		"containers": containers,
	}))
}

func (r *Router) containerDetail(c *gin.Context) {
	user := userFromContext(c)
	serverRouteID := c.Param("serverID")
	containerID := c.Param("containerID")

	server, err := r.store.ServerByID(user.TenantID, serverRouteID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}

	container, project, err := r.store.ContainerByID(user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"server":    newServerView(server),
		"project":   newServiceView(project),
		"container": newContainerView(container),
	}))
}

func (r *Router) containerMetrics(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	containerID := c.Param("containerID")

	metrics, container, project, err := r.store.ContainerMetrics(user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project":   newServiceView(project),
		"container": newContainerView(container),
		"metrics":   metrics,
	}))
}

func (r *Router) containerLogs(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	containerID := c.Param("containerID")

	container, project, err := r.store.ContainerByID(user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	logs := store.FilterLogs(r.store.LogsByContainer(project.ID, container.ID), c.Query("search"), parseLimit(c, 200))
	c.JSON(http.StatusOK, successResponse(gin.H{
		"project":   newServiceView(project),
		"container": newContainerView(container),
		"logs":      newLogLinesView(logs),
	}))
}

func (r *Router) containerEvents(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	containerID := c.Param("containerID")

	events, container, project, err := r.store.ContainerEvents(user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project":   newServiceView(project),
		"container": newContainerView(container),
		"events":    limitEvents(events, parseLimit(c, 20)),
	}))
}

func (r *Router) containerEnv(c *gin.Context) {
	user := userFromContext(c)
	server, err := r.store.ServerByID(user.TenantID, c.Param("serverID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("server not found", "SERVER_NOT_FOUND", nil))
		return
	}
	containerID := c.Param("containerID")

	env, container, project, err := r.store.ContainerEnv(user.TenantID, server.ID, containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("container not found", "CONTAINER_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"project":   newServiceView(project),
		"container": newContainerView(container),
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

	c.JSON(http.StatusOK, successResponse(newServiceView(service)))
}

func (r *Router) serviceLogs(c *gin.Context) {
	user := userFromContext(c)
	service, err := r.store.ServiceByID(user.TenantID, c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("service not found", "SERVICE_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"service": newServiceView(service),
		"logs":    newLogLinesView(r.store.LogsByService(service.ID)),
	}))
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

func (r *Router) agentEnroll(c *gin.Context) {
	apiKey := c.GetHeader("X-Agent-Key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("missing agent api key", "UNAUTHORIZED", nil))
		return
	}

	agent, err := r.store.AgentByAPIKey(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("invalid agent api key", "UNAUTHORIZED", nil))
		return
	}

	var request struct {
		AgentID  string `json:"agent_id"`
		ServerID string `json:"server_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		return
	}

	request.AgentID = strings.TrimSpace(request.AgentID)
	request.ServerID = strings.TrimSpace(request.ServerID)
	if request.AgentID == "" || request.ServerID == "" {
		c.JSON(http.StatusBadRequest, errorResponse("agent_id and server_id are required", "INVALID_REQUEST", nil))
		return
	}
	if request.AgentID != agent.ID {
		c.JSON(http.StatusConflict, errorResponse("agent identity does not match the enrollment token", "AGENT_ENROLLMENT_CONFLICT", nil))
		return
	}

	enrolledAgent, err := r.store.SelfEnrollPendingAgent(agent.ID, request.ServerID)
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, errorResponse("agent can no longer self-enroll", "AGENT_ENROLLMENT_CONFLICT", nil))
			return
		}
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, errorResponse("pending system onboarding not found", "AGENT_ENROLLMENT_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("failed to enroll agent", "AGENT_ENROLLMENT_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{
		"agent_id":  enrolledAgent.ID,
		"server_id": enrolledAgent.ServerID,
		"api_key":   enrolledAgent.APIKey,
	}))
}

func (r *Router) ingestSnapshot(c *gin.Context) {
	apiKey := c.GetHeader("X-Agent-Key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("missing agent api key", "UNAUTHORIZED", nil))
		return
	}

	agent, err := r.store.AgentByAPIKey(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("invalid agent api key", "UNAUTHORIZED", nil))
		return
	}
	if agent.Version == "pending" {
		c.JSON(http.StatusConflict, errorResponse("agent must self-enroll before sending snapshots", "AGENT_ENROLLMENT_REQUIRED", nil))
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
		token := r.requestAuthToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("missing auth session", "UNAUTHORIZED", nil))
			return
		}

		user, err := r.store.UserByToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("invalid token", "UNAUTHORIZED", nil))
			return
		}

		c.Set("user", user)
		c.Set("auth_token", token)
		c.Next()
	}
}

func (r *Router) adminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := userFromContext(c)
		if !hasAdminAccess(user.Role) {
			c.AbortWithStatusJSON(http.StatusForbidden, errorResponse("admin access required", "FORBIDDEN", nil))
			return
		}
		c.Next()
	}
}

func (r *Router) requestAuthToken(c *gin.Context) string {
	if token, err := c.Cookie(sessionCookieName); err == nil && strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token)
	}

	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return ""
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || token == header {
		return ""
	}
	return token
}

func (r *Router) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, token, sessionCookieMaxAgeSeconds, "/", "", false, true)
}

func (r *Router) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
}

func userFromContext(c *gin.Context) domain.User {
	value, _ := c.Get("user")
	user, _ := value.(domain.User)
	return user
}

func authTokenFromContext(c *gin.Context) string {
	value, _ := c.Get("auth_token")
	token, _ := value.(string)
	return token
}

func hasAdminAccess(role domain.UserRole) bool {
	switch role {
	case domain.RoleAdmin, domain.RoleOwner:
		return true
	default:
		return false
	}
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
