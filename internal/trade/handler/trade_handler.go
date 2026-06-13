package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/fathimasithara01/multitrade-platform/internal/shared/response"
	tradesvc "github.com/fathimasithara01/multitrade-platform/internal/trade/service"
)

type TradeHandler struct {
	service tradesvc.TradeService
}

func NewTradeHandler(service tradesvc.TradeService) *TradeHandler {
	return &TradeHandler{service: service}
}

func (h *TradeHandler) GetMyTrades(c *gin.Context) {
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

	trades, err := h.service.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to retrieve trade history")
		return
	}

	response.Success(c, http.StatusOK, trades)
}
