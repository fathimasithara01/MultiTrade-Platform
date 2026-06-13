package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/fathimasithara01/multitrade-platform/internal/shared/response"
	portfoliosvc "github.com/fathimasithara01/multitrade-platform/internal/portfolio/service"
)

type PortfolioHandler struct {
	service portfoliosvc.PortfolioService
}

func NewPortfolioHandler(service portfoliosvc.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{service: service}
}

func (h *PortfolioHandler) GetMyPortfolio(c *gin.Context) {
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

	holdings, err := h.service.GetMyPortfolio(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to retrieve portfolio holdings")
		return
	}

	response.Success(c, http.StatusOK, holdings)
}
