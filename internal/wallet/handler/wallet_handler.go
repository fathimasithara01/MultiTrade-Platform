package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/middleware"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet/dto"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/wallet/service"
)

type WalletHandler struct {
	walletSvc *service.WalletService
}

func NewWalletHandler(walletSvc *service.WalletService) *WalletHandler {
	return &WalletHandler{walletSvc: walletSvc}
}

func (h *WalletHandler) GetWallet(c *gin.Context) {
	userID := mustUserID(c)

	w, err := h.walletSvc.GetWallet(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		log.Error().Err(err).Int64("user_id", userID).Msg("get wallet error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet": w})
}

func (h *WalletHandler) Deposit(c *gin.Context) {
	userID := mustUserID(c)

	var input dto.AmountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	w, tx, err := h.walletSvc.Deposit(c.Request.Context(), userID, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidAmount) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrWalletNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		log.Error().Err(err).Int64("user_id", userID).Msg("deposit error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "deposit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallet":      w,
		"transaction": tx,
	})
}

func (h *WalletHandler) Withdraw(c *gin.Context) {
	userID := mustUserID(c)

	var input dto.AmountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	w, tx, err := h.walletSvc.Withdraw(c.Request.Context(), userID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAmount):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, repository.ErrInsufficientBalance):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "insufficient balance"})
		case errors.Is(err, repository.ErrWalletNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		default:
			log.Error().Err(err).Int64("user_id", userID).Msg("withdraw error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "withdrawal failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallet":      w,
		"transaction": tx,
	})
}

func (h *WalletHandler) ListTransactions(c *gin.Context) {
	userID := mustUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.walletSvc.ListTransactions(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		log.Error().Err(err).Int64("user_id", userID).Msg("list transactions error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve transactions"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func mustUserID(c *gin.Context) int64 {
	v, _ := c.Get(middleware.ContextKeyUserID)
	id, _ := v.(int64)
	return id
}
