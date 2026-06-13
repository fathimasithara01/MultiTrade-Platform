package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/fathimasithara01/multitrade-platform/internal/shared/response"
	compliancesvc "github.com/fathimasithara01/multitrade-platform/internal/compliance/service"
)

type ComplianceHandler struct {
	service compliancesvc.ComplianceService
}

func NewComplianceHandler(service compliancesvc.ComplianceService) *ComplianceHandler {
	return &ComplianceHandler{service: service}
}

func (h *ComplianceHandler) CheckStatus(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, ok := userIDVal.(int64)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "Invalid user claim type")
		return
	}

	passed, reason, limit, err := h.service.CheckUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Compliance check failed")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"passed":    passed,
		"reason":    reason,
		"limit_usd": limit,
	})
}
