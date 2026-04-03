package auth

import (
	"net/http"
	"strings"

	sharedhttp "github.com/dipesh/bifrost/backend/internal/shared/http"
	"github.com/dipesh/bifrost/backend/internal/store"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service   *Service
	inspector *sharedhttp.RequestInspector
}

func NewHandler(service *Service, inspector *sharedhttp.RequestInspector) *Handler {
	return &Handler{service: service, inspector: inspector}
}

func (h *Handler) BootstrapStatus(c *gin.Context) {
	needsBootstrap, err := h.service.BootstrapStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to read bootstrap status", "BOOTSTRAP_STATUS_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"needs_bootstrap": needsBootstrap,
	}))
}

func (h *Handler) InviteDetail(c *gin.Context) {
	invite, err := h.service.InviteByToken(c.Request.Context(), strings.TrimSpace(c.Param("token")))
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, sharedhttp.Error("invite is no longer available", "INVITE_UNAVAILABLE", nil))
			return
		}
		c.JSON(http.StatusNotFound, sharedhttp.Error("invite not found", "INVITE_NOT_FOUND", nil))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(invite))
}

func (h *Handler) AcceptInvite(c *gin.Context) {
	var request struct {
		Token    string `json:"token"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	if !sharedhttp.DecodeJSON(c, &request) {
		return
	}

	request.Token = strings.TrimSpace(request.Token)
	request.Name = strings.TrimSpace(request.Name)
	request.Password = strings.TrimSpace(request.Password)
	if request.Token == "" || request.Name == "" || request.Password == "" {
		c.JSON(http.StatusBadRequest, sharedhttp.Error("token, name, and password are required", "INVALID_REQUEST", nil))
		return
	}

	user, err := h.service.AcceptViewerInvite(c.Request.Context(), request.Token, request.Name, request.Password)
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, sharedhttp.Error("invite is no longer available", "INVITE_UNAVAILABLE", nil))
			return
		}
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("invite not found", "INVITE_NOT_FOUND", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to accept invite", "INVITE_ACCEPT_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	SetSessionCookie(c, user.AuthToken, h.inspector.IsSecure(c.Request))
	c.JSON(http.StatusCreated, sharedhttp.Success(gin.H{
		"user": user,
	}))
}

func (h *Handler) BootstrapAdmin(c *gin.Context) {
	var request struct {
		TenantName string `json:"tenant_name"`
		Name       string `json:"name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
	}

	if !sharedhttp.DecodeJSON(c, &request) {
		return
	}

	request.TenantName = strings.TrimSpace(request.TenantName)
	request.Name = strings.TrimSpace(request.Name)
	request.Email = strings.TrimSpace(strings.ToLower(request.Email))
	request.Password = strings.TrimSpace(request.Password)

	if request.Name == "" || request.Email == "" || request.Password == "" {
		c.JSON(http.StatusBadRequest, sharedhttp.Error("name, email, and password are required", "INVALID_REQUEST", nil))
		return
	}

	user, err := h.service.BootstrapAdmin(c.Request.Context(), request.TenantName, request.Name, request.Email, request.Password)
	if err != nil {
		if err == store.ErrConflict {
			c.JSON(http.StatusConflict, sharedhttp.Error("bootstrap has already been completed", "BOOTSTRAP_ALREADY_COMPLETED", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to bootstrap admin", "BOOTSTRAP_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	SetSessionCookie(c, user.AuthToken, h.inspector.IsSecure(c.Request))
	c.JSON(http.StatusCreated, sharedhttp.Success(gin.H{
		"user": user,
	}))
}

func (h *Handler) Login(c *gin.Context) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if !sharedhttp.DecodeJSON(c, &request) {
		return
	}

	needsBootstrap, err := h.service.BootstrapStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to read bootstrap status", "BOOTSTRAP_STATUS_FAILED", gin.H{"reason": err.Error()}))
		return
	}
	if needsBootstrap {
		c.JSON(http.StatusConflict, sharedhttp.Error("bootstrap is required before login", "BOOTSTRAP_REQUIRED", nil))
		return
	}

	user, err := h.service.Authenticate(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, sharedhttp.Error("invalid email or password", "INVALID_CREDENTIALS", nil))
		return
	}

	SetSessionCookie(c, user.AuthToken, h.inspector.IsSecure(c.Request))
	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"user": user,
	}))
}

func (h *Handler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, sharedhttp.Success(UserFromContext(c)))
}

func (h *Handler) Logout(c *gin.Context) {
	token := AuthTokenFromContext(c)
	if token != "" {
		if err := h.service.RevokeSession(c.Request.Context(), token); err != nil && err != store.ErrNotFound {
			c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to revoke session", "LOGOUT_FAILED", gin.H{"reason": err.Error()}))
			return
		}
	}

	ClearSessionCookie(c, h.inspector.IsSecure(c.Request))
	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"status": "logged_out",
	}))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	user := UserFromContext(c)

	var request struct {
		Name string `json:"name"`
	}
	if !sharedhttp.DecodeJSON(c, &request) {
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		c.JSON(http.StatusBadRequest, sharedhttp.Error("name is required", "INVALID_REQUEST", nil))
		return
	}

	updatedUser, err := h.service.UpdateUserName(c.Request.Context(), user.ID, request.Name)
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("user not found", "USER_NOT_FOUND", nil))
			return
		}
		if err == store.ErrConflict {
			c.JSON(http.StatusBadRequest, sharedhttp.Error("name is required", "INVALID_REQUEST", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to update profile", "PROFILE_UPDATE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(updatedUser))
}

func (h *Handler) ChangePassword(c *gin.Context) {
	user := UserFromContext(c)

	var request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !sharedhttp.DecodeJSON(c, &request) {
		return
	}

	request.CurrentPassword = strings.TrimSpace(request.CurrentPassword)
	request.NewPassword = strings.TrimSpace(request.NewPassword)
	if request.CurrentPassword == "" || request.NewPassword == "" {
		c.JSON(http.StatusBadRequest, sharedhttp.Error("current password and new password are required", "INVALID_REQUEST", nil))
		return
	}
	if len(request.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, sharedhttp.Error("new password must be at least 8 characters", "INVALID_REQUEST", nil))
		return
	}

	if err := h.service.ChangeUserPassword(c.Request.Context(), user.ID, request.CurrentPassword, request.NewPassword); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, sharedhttp.Error("user not found", "USER_NOT_FOUND", nil))
			return
		}
		if err == store.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, sharedhttp.Error("current password is incorrect", "INVALID_CREDENTIALS", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, sharedhttp.Error("failed to update password", "PASSWORD_UPDATE_FAILED", gin.H{"reason": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"status": "password_updated",
	}))
}
