package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RBACMiddleware(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "no role found in context; ensure AuthMiddleware runs first",
			})
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "role context value has unexpected type",
			})
			return
		}

		if _, permitted := allowed[roleStr]; !permitted {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "you do not have permission to access this resource",
			})
			return
		}

		c.Next()
	}
}
