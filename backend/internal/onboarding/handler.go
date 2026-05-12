package onboarding

import (
	"net/http"

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

func (h *Handler) Create(c *gin.Context) {
	user := authctx.UserFromContext(c)

	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !sharedhttp.DecodeJSON(c, &request) {
		return
	}

	onboarding, err := h.service.Create(c.Request.Context(), CreateInput{
		TenantID:        user.TenantID,
		CreatedByUserID: user.ID,
		Name:            request.Name,
		Description:     request.Description,
	})
	if err == ErrInvalidName {
		c.JSON(http.StatusBadRequest, sharedhttp.Error("name is required", "INVALID_REQUEST", nil))
		return
	}
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, sharedhttp.Error("system could not be created", "SYSTEM_ONBOARDING_CONFLICT", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to create system onboarding", "SYSTEM_ONBOARDING_CREATE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusCreated, sharedhttp.Success(onboarding))
}

func (h *Handler) Detail(c *gin.Context) {
	user := authctx.UserFromContext(c)

	onboarding, err := h.service.Get(c.Request.Context(), user.TenantID, c.Param("systemID"))
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("system onboarding not found", "SYSTEM_ONBOARDING_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load system onboarding", "SYSTEM_ONBOARDING_LOAD_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(onboarding))
}

func (h *Handler) List(c *gin.Context) {
	user := authctx.UserFromContext(c)

	onboardings, err := h.service.List(c.Request.Context(), user.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to load system onboardings", "SYSTEM_ONBOARDING_LIST_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(onboardings))
}

func (h *Handler) Cancel(c *gin.Context) {
	user := authctx.UserFromContext(c)

	err := h.service.Cancel(c.Request.Context(), user.TenantID, c.Param("systemID"))
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("system onboarding not found", "SYSTEM_ONBOARDING_NOT_FOUND", nil))
			return
		}
		if err == store.ErrConflict || err == ErrOnboardingNotPending {
			c.JSON(http.StatusConflict, sharedhttp.Error("only pending system onboardings can be cancelled", "SYSTEM_ONBOARDING_NOT_PENDING", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to cancel system onboarding", "SYSTEM_ONBOARDING_CANCEL_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"status": "cancelled",
	}))
}

func (h *Handler) Reissue(c *gin.Context) {
	user := authctx.UserFromContext(c)

	onboarding, err := h.service.Reissue(c.Request.Context(), user.TenantID, c.Param("systemID"))
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("system onboarding not found", "SYSTEM_ONBOARDING_NOT_FOUND", nil))
			return
		}
		if err == store.ErrConflict || err == ErrOnboardingNotPending {
			c.JSON(http.StatusConflict, sharedhttp.Error("only pending system onboardings can be reissued", "SYSTEM_ONBOARDING_NOT_PENDING", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to reissue system onboarding credentials", "SYSTEM_ONBOARDING_REISSUE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(onboarding))
}
