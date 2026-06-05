package alerts

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rmm-platform/backend/internal/auth"
	"github.com/rmm-platform/backend/internal/realtime"
	"github.com/rmm-platform/backend/internal/shared/config"
)

type Handler struct {
	repo *Repository
	hub  *realtime.Hub
}

func NewHandler(db *sqlx.DB, hub *realtime.Hub) *Handler {
	return &Handler{repo: NewRepository(db), hub: hub}
}

// List godoc
// @Summary List alerts
// @Description Get paginated list of alerts with optional filters
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 50, max 100)"
// @Param status query string false "Filter by status (open/acknowledged/resolved)"
// @Param severity query string false "Filter by severity (critical/warning/info)"
// @Param device_id query string false "Filter by device ID"
// @Success 200 {object} AlertListResponse
// @Failure 500 {object} map[string]string
// @Router /alerts [get]
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	status := c.Query("status")
	severity := c.Query("severity")
	deviceID := c.Query("device_id")

	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 50 }

	alerts, total, err := h.repo.List(page, limit, status, severity, deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list alerts: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, AlertListResponse{Alerts: alerts, Total: total})
}

// Get godoc
// @Summary Get alert by ID
// @Description Get detailed alert information
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Alert ID"
// @Success 200 {object} Alert
// @Failure 404 {object} map[string]string
// @Router /alerts/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	alert, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}
	c.JSON(http.StatusOK, alert)
}

// Acknowledge godoc
// @Summary Acknowledge alert
// @Description Mark alert as acknowledged
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Alert ID"
// @Success 200 {object} Alert
// @Failure 500 {object} map[string]string
// @Router /alerts/{id}/acknowledge [post]
func (h *Handler) Acknowledge(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	if err := h.repo.Acknowledge(id, userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to acknowledge alert"})
		return
	}

	alert, _ := h.repo.GetByID(id)
	h.hub.Broadcast(realtime.Message{Type: "alert.updated", Payload: alert})
	c.JSON(http.StatusOK, alert)
}

// Resolve godoc
// @Summary Resolve alert
// @Description Mark alert as resolved with optional note
// @Tags alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Alert ID"
// @Param body body ResolveRequest false "Resolution note"
// @Success 200 {object} Alert
// @Failure 500 {object} map[string]string
// @Router /alerts/{id}/resolve [post]
func (h *Handler) Resolve(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	var req ResolveRequest
	c.ShouldBindJSON(&req)

	if err := h.repo.Resolve(id, userID.(string), req.ResolutionNote); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve alert"})
		return
	}

	alert, _ := h.repo.GetByID(id)
	h.hub.Broadcast(realtime.Message{Type: "alert.updated", Payload: alert})
	c.JSON(http.StatusOK, alert)
}

func RegisterRoutes(rg *gin.RouterGroup, db *sqlx.DB, cfg *config.Config, hub *realtime.Hub) {
	h := NewHandler(db, hub)
	mid := auth.NewMiddleware(&cfg.JWT)

	alerts := rg.Group("/alerts")
	alerts.Use(mid.RequireAuth())
	{
		alerts.GET("", h.List)
		alerts.GET("/:id", h.Get)
		alerts.POST("/:id/acknowledge", h.Acknowledge)
		alerts.POST("/:id/resolve", h.Resolve)
	}
}
