package install

import (
	"net/http"
	"strings"

	sharedhttp "github.com/dipesh/bifrost/backend/internal/shared/http"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) InstallScript(c *gin.Context) {
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
		`AGENT_IMAGE="${BIFROST_AGENT_IMAGE:-` + h.service.AgentDockerImage() + `}"`,
		`AGENT_BINARY_URL="${BIFROST_AGENT_BINARY_URL:-` + h.defaultBinaryURL(c) + `}"`,
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

func (h *Handler) InstallBinary(c *gin.Context) {
	binaryPath, err := h.service.ResolveAgentBinaryPath()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, sharedhttp.Error("failed to prepare agent binary", "AGENT_BINARY_UNAVAILABLE", gin.H{"reason": err.Error()}))
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="bifrost-agent"`)
	c.File(binaryPath)
}

func (h *Handler) defaultBinaryURL(c *gin.Context) string {
	if !h.service.BinaryDownloadEnabled() {
		return ""
	}
	return h.service.InstallAgentBinaryURL(h.service.DefaultBackendURL(c.Request))
}
