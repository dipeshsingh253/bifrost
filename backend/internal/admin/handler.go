package admin

import (
	"net/http"
	"strings"

	authctx "github.com/dipesh/bifrost/backend/internal/auth"
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

func (h *Handler) Summary(c *gin.Context) {
	user := authctx.UserFromContext(c)

	summary, err := h.service.TenantSummary(c.Request.Context(), user.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load tenant summary", "TENANT_SUMMARY_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"tenant": summary,
		"user":   user,
	}))
}

func (h *Handler) ViewerAccess(c *gin.Context) {
	user := authctx.UserFromContext(c)

	access, err := h.service.ViewerAccess(c.Request.Context(), user.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load viewer access", "VIEWER_ACCESS_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(access))
}

func (h *Handler) CreateViewerInvite(c *gin.Context) {
	user := authctx.UserFromContext(c)

	var request struct {
		Email string `json:"email"`
	}
	if !sharedhttp.DecodeJSON(c, &request) {
		return
	}

	request.Email = strings.TrimSpace(strings.ToLower(request.Email))
	if request.Email == "" {
		c.JSON(http.StatusBadRequest, sharedhttp.Error("email is required", "INVALID_REQUEST", nil))
		return
	}

	invite, err := h.service.CreateViewerInvite(c.Request.Context(), user.TenantID, user.ID, request.Email)
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, sharedhttp.Error("viewer or invite already exists", "INVITE_CONFLICT", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to create invite", "INVITE_CREATE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusCreated, sharedhttp.Success(invite))
}

func (h *Handler) RevokeViewerInvite(c *gin.Context) {
	user := authctx.UserFromContext(c)

	if err := h.service.RevokeViewerInvite(c.Request.Context(), user.TenantID, c.Param("inviteID")); err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, sharedhttp.Error("invite can no longer be revoked", "INVITE_REVOKE_CONFLICT", nil))
			return
		}
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("invite not found", "INVITE_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to revoke invite", "INVITE_REVOKE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"status": "revoked",
	}))
}

func (h *Handler) DisableViewer(c *gin.Context) {
	user := authctx.UserFromContext(c)

	if err := h.service.DisableViewer(c.Request.Context(), user.TenantID, c.Param("userID")); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("viewer not found", "VIEWER_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to disable viewer", "VIEWER_DISABLE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"status": "disabled",
	}))
}

func (h *Handler) DeleteViewer(c *gin.Context) {
	user := authctx.UserFromContext(c)

	if err := h.service.DeleteViewer(c.Request.Context(), user.TenantID, c.Param("userID")); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("viewer not found", "VIEWER_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to delete viewer", "VIEWER_DELETE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"status": "deleted",
	}))
}
