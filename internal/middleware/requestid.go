package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerRequestID = "X-Request-ID"

// RequestID injects a unique request ID into every request.
// If the caller already supplies an X-Request-ID header it is preserved;
// otherwise a new UUID v4 is generated. The ID is echoed back in the response
// header and stored in the Gin context under key "request_id".
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(headerRequestID)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		c.Header(headerRequestID, rid)
		c.Next()
	}
}
