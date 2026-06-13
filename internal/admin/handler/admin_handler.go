package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/multitrade-platform/internal/admin/service"
	"github.com/fathimasithara01/multitrade-platform/internal/middleware"
	userrepo "github.com/fathimasithara01/multitrade-platform/internal/user/repository"
)

type AdminHandler struct {
	adminSvc *service.AdminService
}

func NewAdminHandler(adminSvc *service.AdminService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	role := c.DefaultQuery("role", "")
	status := c.DefaultQuery("status", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.adminSvc.ListUsers(c.Request.Context(), role, status, page, pageSize)
	if err != nil {
		log.Error().Err(err).Msg("admin list users error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list users"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	adminID := mustUserID(c)
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.adminSvc.UpdateUserStatus(c.Request.Context(), adminID, targetID, body.Status)
	if err != nil {
		switch {
		case errors.Is(err, userrepo.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		case errors.Is(err, service.ErrInvalidStatus):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			log.Error().Err(err).Int64("target_id", targetID).Msg("admin update user status error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update user status"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": u})
}

func (h *AdminHandler) VolumeAnalytics(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))

	points, err := h.adminSvc.VolumeAnalytics(c.Request.Context(), hours)
	if err != nil {
		log.Error().Err(err).Msg("admin volume analytics error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not compute volume analytics"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"window_hours": hours,
		"data":         points,
		"count":        len(points),
	})
}

func (h *AdminHandler) SuspiciousUsers(c *gin.Context) {
	windowHours, _ := strconv.Atoi(c.DefaultQuery("window_hours", "1"))
	threshold, _ := strconv.Atoi(c.DefaultQuery("threshold", "10"))

	suspects, err := h.adminSvc.SuspiciousUsers(c.Request.Context(), windowHours, threshold)
	if err != nil {
		log.Error().Err(err).Msg("admin suspicious users error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not compute suspicious users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"window_hours": windowHours,
		"threshold":    threshold,
		"suspects":     suspects,
		"count":        len(suspects),
	})
}

func (h *AdminHandler) AdminHealth(c *gin.Context) {
	health := h.adminSvc.Health(c.Request.Context())

	statusCode := http.StatusOK
	if health.Database == "DOWN" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  health,
	})
}

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	action := c.DefaultQuery("action", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var userIDPtr *int64
	if uidStr := c.Query("user_id"); uidStr != "" {
		uid, err := strconv.ParseInt(uidStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		userIDPtr = &uid
	}

	logs, total, err := h.adminSvc.ListAuditLogs(c.Request.Context(), userIDPtr, action, page, pageSize)
	if err != nil {
		log.Error().Err(err).Msg("admin audit logs error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve audit logs"})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        logs,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
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
