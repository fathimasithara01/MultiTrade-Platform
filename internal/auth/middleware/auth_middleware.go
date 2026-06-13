package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/fathimasithara01/multitrade-platform/pkg/jwt"
)

const (
	ContextKeyUserID = "user_id"
	ContextKeyRole   = "user_role"
)

func AuthMiddleware(jwtService *jwt.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header must be in the form 'Bearer <token>'",
			})
			return
		}

		tokenStr := parts[1]
		claims, err := jwtService.ParseToken(tokenStr)
		if err != nil {
			status := http.StatusUnauthorized
			msg := "invalid token"
			if err == jwt.ErrTokenExpired {
				msg = "token has expired"
			}
			c.AbortWithStatusJSON(status, gin.H{"error": msg})
			return
		}

		if claims.TokenType != jwt.TokenTypeAccess {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "access token required",
			})
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRole, claims.Role)
		c.Next()
	}
}
