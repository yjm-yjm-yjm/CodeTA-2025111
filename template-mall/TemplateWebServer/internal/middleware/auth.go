package middleware

import (
	"net/http"
	"strings"

	"template-mall/TemplateWebServer/internal/auth"

	"github.com/gin-gonic/gin"
)

const ContextUserIDKey = "user_id"
const ContextUsernameKey = "username"

func JWTAuth(issuer *auth.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		claims, err := issuer.Parse(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUsernameKey, claims.Username)
		c.Next()
	}
}

func UserID(c *gin.Context) string {
	v, _ := c.Get(ContextUserIDKey)
	s, _ := v.(string)
	return s
}
