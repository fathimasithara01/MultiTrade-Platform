package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Error().Interface("panic", recovered).Msg("CRITICAL: Panic recovered in HTTP handler")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error occurred",
		})
	})
}
