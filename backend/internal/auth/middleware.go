package auth

import (
	"net/http"
	"strings"

	sharedhttp "github.com/dipesh/bifrost/backend/internal/shared/http"
	"github.com/gin-gonic/gin"
)

const SessionCookieName = "bifrost_session"
const SessionCookieMaxAgeSeconds = 30 * 24 * 60 * 60

func AuthRequired(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := RequestAuthToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, sharedhttp.Error("missing auth session", "UNAUTHORIZED", nil))
			return
		}

		user, err := service.UserByToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, sharedhttp.Error("invalid token", "UNAUTHORIZED", nil))
			return
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("auth_token", token)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := UserFromContext(c)
		if !HasAdminAccess(user.Role) {
			c.AbortWithStatusJSON(http.StatusForbidden, sharedhttp.Error("admin access required", "FORBIDDEN", nil))
			return
		}
		c.Next()
	}
}

func RequestAuthToken(c *gin.Context) string {
	if token, err := c.Cookie(SessionCookieName); err == nil && strings.TrimSpace(token) != "" {
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

func SetSessionCookie(c *gin.Context, token string, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, token, SessionCookieMaxAgeSeconds, "/", "", secure, true)
}

func ClearSessionCookie(c *gin.Context, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, "", -1, "/", "", secure, true)
}

func UserFromContext(c *gin.Context) User {
	value, _ := c.Get("user")
	user, _ := value.(User)
	return user
}

func AuthTokenFromContext(c *gin.Context) string {
	value, _ := c.Get("auth_token")
	token, _ := value.(string)
	return token
}

func HasAdminAccess(role UserRole) bool {
	switch role {
	case RoleAdmin, RoleOwner:
		return true
	default:
		return false
	}
}
