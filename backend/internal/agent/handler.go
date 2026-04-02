package agent

import (
	"net/http"
	"strings"
	"time"

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

func (h *Handler) Heartbeat(c *gin.Context) {
	var request struct {
		AgentID string `json:"agent_id"`
	}

	if !sharedhttp.DecodeJSON(c, &request) {
		return
	}

	if err := h.service.UpdateAgentLastSeen(c.Request.Context(), request.AgentID); err != nil {
		c.JSON(http.StatusNotFound, sharedhttp.Error("agent not found", "AGENT_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{"status": "ok"}))
}

func (h *Handler) Enroll(c *gin.Context) {
	apiKey := c.GetHeader("X-Agent-Key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, sharedhttp.Error("missing agent api key", "UNAUTHORIZED", nil))
		return
	}

	agent, err := h.service.AgentByAPIKey(c.Request.Context(), apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, sharedhttp.Error("invalid agent api key", "UNAUTHORIZED", nil))
		return
	}

	var request struct {
		AgentID  string `json:"agent_id"`
		ServerID string `json:"server_id"`
	}
	if !sharedhttp.DecodeJSON(c, &request) {
		return
	}

	request.AgentID = strings.TrimSpace(request.AgentID)
	request.ServerID = strings.TrimSpace(request.ServerID)
	if request.AgentID == "" || request.ServerID == "" {
		c.JSON(http.StatusBadRequest, sharedhttp.Error("agent_id and server_id are required", "INVALID_REQUEST", nil))
		return
	}
	if request.AgentID != agent.ID {
		c.JSON(http.StatusConflict, sharedhttp.Error("agent identity does not match the enrollment token", "AGENT_ENROLLMENT_CONFLICT", nil))
		return
	}

	enrolledAgent, err := h.service.SelfEnrollPendingAgent(c.Request.Context(), agent.ID, request.ServerID)
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, sharedhttp.Error("agent can no longer self-enroll", "AGENT_ENROLLMENT_CONFLICT", nil))
			return
		}
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("pending system onboarding not found", "AGENT_ENROLLMENT_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to enroll agent", "AGENT_ENROLLMENT_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"agent_id":  enrolledAgent.ID,
		"server_id": enrolledAgent.ServerID,
		"api_key":   enrolledAgent.APIKey,
	}))
}

func (h *Handler) Snapshot(c *gin.Context) {
	apiKey := c.GetHeader("X-Agent-Key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, sharedhttp.Error("missing agent api key", "UNAUTHORIZED", nil))
		return
	}

	agent, err := h.service.AgentByAPIKey(c.Request.Context(), apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, sharedhttp.Error("invalid agent api key", "UNAUTHORIZED", nil))
		return
	}
	if agent.Version == "pending" {
		c.JSON(http.StatusConflict, sharedhttp.Error("agent must self-enroll before sending snapshots", "AGENT_ENROLLMENT_REQUIRED", nil))
		return
	}

	var payload IngestPayload
	if !sharedhttp.DecodeJSON(c, &payload) {
		return
	}

	if payload.Server.CollectedAt.IsZero() {
		payload.Server.CollectedAt = time.Now().UTC()
	}

	if err := h.service.Ingest(c.Request.Context(), payload); err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to persist snapshot", "INGEST_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusAccepted, sharedhttp.Success(gin.H{"status": "accepted"}))
}
