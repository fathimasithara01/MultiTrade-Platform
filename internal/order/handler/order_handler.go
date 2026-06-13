package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	assetrepo "github.com/fathimasithara01/multitrade-platform/internal/asset/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/middleware"
	"github.com/fathimasithara01/multitrade-platform/internal/order/dto"
	orderrepo "github.com/fathimasithara01/multitrade-platform/internal/order/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/order/service"
)

type OrderHandler struct {
	orderSvc *service.OrderService
}

func NewOrderHandler(orderSvc *service.OrderService) *OrderHandler {
	return &OrderHandler{orderSvc: orderSvc}
}

func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	userID := mustUserID(c)

	var input dto.PlaceOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if headerKey := c.GetHeader("Idempotency-Key"); headerKey != "" {
		input.IdempotencyKey = &headerKey
	}

	o, err := h.orderSvc.PlaceOrder(c.Request.Context(), userID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderInsufficientFunds):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrOrderInsufficientHolding):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrOrderAssetNotActive):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, assetrepo.ErrAssetNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		default:
			log.Error().Err(err).Int64("user_id", userID).Msg("place order error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not place order"})
		}
		return
	}

	statusCode := http.StatusCreated
	if input.IdempotencyKey != nil && o.CreatedAt.IsZero() {
		statusCode = http.StatusOK
	}
	c.JSON(statusCode, gin.H{"order": o})
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID := mustUserID(c)
	orderID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	o, err := h.orderSvc.CancelOrder(c.Request.Context(), userID, orderID)
	if err != nil {
		switch {
		case errors.Is(err, orderrepo.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		case errors.Is(err, service.ErrOrderNotOwned):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrOrderNotCancellable):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			log.Error().Err(err).Int64("order_id", orderID).Msg("cancel order error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not cancel order"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": o})
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	userID := mustUserID(c)
	orderID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	o, err := h.orderSvc.GetOrder(c.Request.Context(), userID, orderID)
	if err != nil {
		switch {
		case errors.Is(err, orderrepo.ErrOrderNotFound), errors.Is(err, service.ErrOrderNotOwned):
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		default:
			log.Error().Err(err).Int64("order_id", orderID).Msg("get order error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve order"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": o})
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID := mustUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.orderSvc.ListOrders(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		log.Error().Err(err).Int64("user_id", userID).Msg("list orders error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list orders"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func mustUserID(c *gin.Context) int64 {
	v, _ := c.Get(middleware.ContextKeyUserID)
	id, _ := v.(int64)
	return id
}

func parseIDParam(c *gin.Context, param string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + param})
		return 0, err
	}
	return id, nil
}
