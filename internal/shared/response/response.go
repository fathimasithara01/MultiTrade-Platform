package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the canonical API envelope.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PaginationMeta contains pagination information
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalCount int64 `json:"totalCount"`
	TotalPages int   `json:"totalPages"`
}

// Success writes a 2xx JSON response using the standard envelope.
func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, Response{Success: true, Data: data})
}

// Error writes an error JSON response using the standard envelope.
func Error(c *gin.Context, statusCode int, errMsg string) {
	c.JSON(statusCode, Response{Success: false, Error: errMsg})
}

// JSON is a thin wrapper around gin.Context.JSON for backward compatibility.
func JSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// NotFound is a convenience 404.
func NotFound(c *gin.Context, msg string) {
	if msg == "" {
		msg = "resource not found"
	}
	Error(c, http.StatusNotFound, msg)
}

// BadRequest is a convenience 400.
func BadRequest(c *gin.Context, msg string) {
	Error(c, http.StatusBadRequest, msg)
}

// InternalError is a convenience 500 that hides internal details.
func InternalError(c *gin.Context) {
	Error(c, http.StatusInternalServerError, "an unexpected error occurred")
}
