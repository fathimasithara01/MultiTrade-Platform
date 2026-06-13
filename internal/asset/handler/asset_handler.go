package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/asset/dto"
	"github.com/fathimasithara01/multitrade-platform/internal/asset/repository"
	"github.com/fathimasithara01/multitrade-platform/internal/asset/service"
	"github.com/fathimasithara01/multitrade-platform/internal/middleware"
)

type AssetHandler struct {
	assetSvc *service.AssetService
}

func NewAssetHandler(assetSvc *service.AssetService) *AssetHandler {
	return &AssetHandler{assetSvc: assetSvc}
}

func (h *AssetHandler) CreateAsset(c *gin.Context) {
	brokerID := mustUserID(c)

	var input dto.CreateAssetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a, err := h.assetSvc.CreateAsset(c.Request.Context(), brokerID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDuplicateSymbol):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidAssetData):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			log.Error().Err(err).Int64("broker_id", brokerID).Msg("create asset error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create asset"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"asset": a})
}

func (h *AssetHandler) UpdateAsset(c *gin.Context) {
	brokerID := mustUserID(c)
	assetID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	var input dto.UpdateAssetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a, err := h.assetSvc.UpdateAsset(c.Request.Context(), brokerID, assetID, input)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrAssetNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		case errors.Is(err, service.ErrNotAssetOwner):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidAssetData):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			log.Error().Err(err).Int64("asset_id", assetID).Msg("update asset error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update asset"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"asset": a})
}

func (h *AssetHandler) DisableAsset(c *gin.Context) {
	brokerID := mustUserID(c)
	assetID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	a, err := h.assetSvc.DisableAsset(c.Request.Context(), brokerID, assetID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrAssetNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		case errors.Is(err, service.ErrNotAssetOwner):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			log.Error().Err(err).Int64("asset_id", assetID).Msg("disable asset error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not disable asset"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"asset": a})
}

func (h *AssetHandler) GetAsset(c *gin.Context) {
	assetID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	a, err := h.assetSvc.GetAsset(c.Request.Context(), assetID)
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}
		log.Error().Err(err).Int64("asset_id", assetID).Msg("get asset error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve asset"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"asset": a})
}

func (h *AssetHandler) ListAssets(c *gin.Context) {
	assets, err := h.assetSvc.ListActiveAssets(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("list assets error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list assets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assets": assets,
		"count":  len(assets),
	})
}

func parseIDParam(c *gin.Context, param string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + param})
		return 0, err
	}
	return id, nil
}

func mustUserID(c *gin.Context) int64 {
	v, _ := c.Get(middleware.ContextKeyUserID)
	id, _ := v.(int64)
	return id
}
