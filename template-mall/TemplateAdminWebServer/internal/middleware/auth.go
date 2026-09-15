package middleware

import (
	"net/http"
	"strings"

	"template-mall/TemplateAdminWebServer/internal/auth"

	"github.com/gin-gonic/gin"
)

const CtxAdminID = "admin_id"
const CtxNickname = "nickname"

func AdminJWT(issuer *auth.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin auth required"})
			return
		}
		claims, err := issuer.Parse(strings.TrimSpace(strings.TrimPrefix(h, "Bearer ")))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(CtxAdminID, claims.AdminID)
		c.Set(CtxNickname, claims.Nickname)
		c.Next()
	}
}

func AdminID(c *gin.Context) string {
	v, _ := c.Get(CtxAdminID)
	s, _ := v.(string)
	return s
}
